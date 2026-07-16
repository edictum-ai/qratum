package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	qschema "github.com/acartag7/qratum/internal/schema"
	"github.com/acartag7/qratum/internal/workspace"
)

const (
	evidenceStatusComplete = "complete"

	findingFinalEditAfterLastTest   = "verification.final_edit_after_last_test"
	findingMissingFinalVerification = "verification.missing_final_verification"
	findingFinalVerificationFailed  = "verification.final_verification_failed"
	findingOnlyFailedVerification   = "verification.only_failed_verification"
	findingRepeatedFailingCommand   = "reliability.repeated_failing_command"
	findingDestructiveCommand       = "tool_risk.destructive_command"
	findingNetworkCallWithoutNeed   = "tool_risk.network_call_without_need"
	findingSourceChangedWithoutTest = "scope.source_changed_without_test_change"
	findingBroadFileChange          = "scope.broad_file_change"

	broadFileChangeThreshold = 8
)

var supportedEvidenceFindingTypes = map[string]struct{}{
	findingFinalEditAfterLastTest:   {},
	findingMissingFinalVerification: {},
	findingFinalVerificationFailed:  {},
	findingOnlyFailedVerification:   {},
	findingRepeatedFailingCommand:   {},
	findingDestructiveCommand:       {},
	findingNetworkCallWithoutNeed:   {},
	findingSourceChangedWithoutTest: {},
	findingBroadFileChange:          {},
}

type evidenceBundle struct {
	SchemaVersion   string                `json:"schema_version"`
	DataClass       string                `json:"data_class"`
	SessionID       string                `json:"session_id"`
	Summary         evidenceBundleSummary `json:"summary"`
	Findings        []evidenceFinding     `json:"findings"`
	MissingEvidence []string              `json:"missing_evidence"`
	SourceEventID   string                `json:"source_event_id,omitempty"`
	ArtifactPaths   daemonArtifactPaths   `json:"artifact_paths"`
}

type evidenceBundleSummary struct {
	Status                 string `json:"status"`
	Source                 string `json:"source"`
	StartedAt              string `json:"started_at,omitempty"`
	EndedAt                string `json:"ended_at,omitempty"`
	SourceEventID          string `json:"source_event_id,omitempty"`
	SourceEventType        string `json:"source_event_type,omitempty"`
	SourceEventTimestamp   string `json:"source_event_timestamp,omitempty"`
	FilesChanged           int    `json:"files_changed"`
	CommandsRun            int    `json:"commands_run"`
	TestsRun               int    `json:"tests_run"`
	LastFileChangeAt       string `json:"last_file_change_at,omitempty"`
	LastTestCommandAt      string `json:"last_test_command_at,omitempty"`
	LastSuccessfulVerifyAt string `json:"last_successful_verification_at,omitempty"`
}

type evidenceFinding struct {
	FindingID       string         `json:"finding_id"`
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Summary         string         `json:"summary"`
	Evidence        []evidenceFact `json:"evidence"`
	MissingEvidence []string       `json:"missing_evidence"`
}

type evidenceFact struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	Timestamp     string `json:"timestamp,omitempty"`
	Path          string `json:"path,omitempty"`
	Operation     string `json:"operation,omitempty"`
	Command       string `json:"command,omitempty"`
	Success       *bool  `json:"success,omitempty"`
	OutputExcerpt string `json:"output_excerpt,omitempty"`
}

type indexedFileChange struct {
	Index  int
	Change qratumFileChange
	At     time.Time
}

type indexedCommand struct {
	Index   int
	Command qratumCommand
	At      time.Time
	HasTime bool
}

type reviewCard struct {
	SchemaVersion      string              `json:"schema_version"`
	DataClass          string              `json:"data_class"`
	SessionID          string              `json:"session_id"`
	Verdict            string              `json:"verdict"`
	MainFinding        string              `json:"main_finding"`
	Evidence           []string            `json:"evidence"`
	SuggestedNextHabit string              `json:"suggested_next_habit"`
	SuggestedSkill     string              `json:"suggested_skill,omitempty"`
	Warnings           []string            `json:"warnings"`
	SourceEventID      string              `json:"source_event_id,omitempty"`
	ArtifactPaths      daemonArtifactPaths `json:"artifact_paths"`
}

func evidence(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing redacted session path")
		return 2
	}
	if len(args) != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: evidence accepts exactly one redacted session path")
		return 2
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project: %v\n", err)
		return 1
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project absolute path: %v\n", err)
		return 1
	}

	sessionPath, err := resolveProjectFilePath(projectRoot, args[0], "redacted session")
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid redacted session path: %v\n", err)
		return 1
	}
	session, err := readQratumSessionFile(sessionPath, projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := requireRedactedSession(session, displayPath(projectRoot, sessionPath)); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	paths, err := artifactPathsForSession(projectRoot, session, sessionPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve evidence artifact paths: %v\n", err)
		return 1
	}
	bundle, err := buildEvidenceBundle(session, paths)
	if err != nil {
		fmt.Fprintf(stderr, "error: build evidence for %s: %v\n", displayPath(projectRoot, sessionPath), err)
		return 1
	}
	outputPath, err := resolveProjectOutputPath(projectRoot, bundle.ArtifactPaths.Evidence, "evidence")
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve evidence output: %v\n", err)
		return 1
	}
	if err := writeFileAtomic(outputPath, mustJSON(bundle), 0o600); err != nil {
		fmt.Fprintf(stderr, "error: write evidence %s: %v\n", displayPath(projectRoot, outputPath), err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", displayPath(projectRoot, outputPath))
	return 0
}

func review(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing evidence path")
		return 2
	}
	if len(args) != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: review accepts exactly one evidence path")
		return 2
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project: %v\n", err)
		return 1
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project absolute path: %v\n", err)
		return 1
	}

	evidencePath, err := resolveProjectFilePath(projectRoot, args[0], "evidence")
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid evidence path: %v\n", err)
		return 1
	}
	bundle, err := readEvidenceBundleFile(evidencePath, projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if strings.TrimSpace(bundle.ArtifactPaths.Evidence) == "" {
		bundle.ArtifactPaths.Evidence = displayPath(projectRoot, evidencePath)
	}
	if strings.TrimSpace(bundle.ArtifactPaths.Review) == "" {
		stem, stemErr := artifactStemForSession(bundle.SessionID)
		if stemErr != nil {
			fmt.Fprintf(stderr, "error: resolve review artifact path: %v\n", stemErr)
			return 1
		}
		bundle.ArtifactPaths.Review = artifactPathsForStem(stem).Review
	}
	if err := validateArtifactPathsScoped(projectRoot, bundle.ArtifactPaths); err != nil {
		fmt.Fprintf(stderr, "error: invalid evidence artifact paths: %v\n", err)
		return 1
	}

	card, err := buildReviewCard(bundle)
	if err != nil {
		fmt.Fprintf(stderr, "error: build review for %s: %v\n", displayPath(projectRoot, evidencePath), err)
		return 1
	}
	outputPath, err := resolveProjectOutputPath(projectRoot, card.ArtifactPaths.Review, "review")
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve review output: %v\n", err)
		return 1
	}
	if err := writeFileAtomic(outputPath, mustJSON(card), 0o600); err != nil {
		fmt.Fprintf(stderr, "error: write review %s: %v\n", displayPath(projectRoot, outputPath), err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", displayPath(projectRoot, outputPath))
	return 0
}

func artifactPathsForSession(projectRoot string, session qratumSession, sessionPath string) (daemonArtifactPaths, error) {
	stem, err := artifactStemForSession(session.SessionID)
	if err != nil {
		return daemonArtifactPaths{}, err
	}
	paths := artifactPathsForStem(stem)
	if err := validateArtifactPathsScoped(projectRoot, paths); err != nil {
		return daemonArtifactPaths{}, err
	}
	_ = sessionPath
	return paths, nil
}

func artifactStemForSession(sessionID string) (string, error) {
	stem := strings.TrimSpace(sessionID)
	if !validArtifactStem(stem) {
		return "", fmt.Errorf("session_id %q is not a safe artifact id", sessionID)
	}
	return stem, nil
}

func artifactPathsForStem(stem string) daemonArtifactPaths {
	sessionDir := filepath.Join("sessions", stem)
	return daemonArtifactPaths{
		Event:    slashPath(filepath.Join("events", stem+".json")),
		Session:  slashPath(filepath.Join(sessionDir, "normalized.json")),
		Redacted: slashPath(filepath.Join(sessionDir, "redacted.json")),
		Evidence: slashPath(filepath.Join(sessionDir, "evidence.json")),
		Review:   slashPath(filepath.Join(sessionDir, "review.json")),
		Report:   slashPath(filepath.Join(sessionDir, "report.html")),
		Export:   slashPath(filepath.Join(sessionDir, "session.adp.jsonl")),
	}
}

func resolveProjectOutputPath(projectRoot string, inputPath string, label string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", fmt.Errorf("missing %s output path", label)
	}

	qratumHome, err := workspace.Resolve()
	if err != nil {
		return "", err
	}
	var resolved string
	if filepath.IsAbs(inputPath) {
		resolved = filepath.Clean(inputPath)
	} else {
		resolved = filepath.Clean(filepath.Join(qratumHome.Root, filepath.FromSlash(inputPath)))
	}
	rel, err := filepath.Rel(qratumHome.Root, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve %s output path: %w", label, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%s output path %q escapes qratum home", label, inputPath)
	}
	_ = projectRoot
	return resolved, nil
}

func validateArtifactPathsScoped(projectRoot string, paths daemonArtifactPaths) error {
	for _, item := range []struct {
		label string
		path  string
	}{
		{label: "event", path: paths.Event},
		{label: "session", path: paths.Session},
		{label: "redacted", path: paths.Redacted},
		{label: "evidence", path: paths.Evidence},
		{label: "review", path: paths.Review},
		{label: "report", path: paths.Report},
		{label: "export", path: paths.Export},
	} {
		if strings.TrimSpace(item.path) == "" {
			continue
		}
		if item.label == "event" {
			if err := validateEventArtifactPath(projectRoot, item.path); err != nil {
				return err
			}
			continue
		}
		if _, err := resolveProjectOutputPath(projectRoot, item.path, item.label); err != nil {
			return err
		}
	}
	return nil
}

func validateEventArtifactPath(projectRoot string, inputPath string) error {
	_, err := resolveProjectOutputPath(projectRoot, inputPath, "event")
	return err
}

func requireRedactedSession(session qratumSession, label string) error {
	if strings.TrimSpace(session.PipelineStatus) != redactionStatus {
		return fmt.Errorf("redacted session %s has pipeline_status %q, want %q", label, session.PipelineStatus, redactionStatus)
	}
	if session.Redaction == nil || strings.TrimSpace(session.Redaction.Status) != redactionStatus {
		return fmt.Errorf("redacted session %s is missing redaction summary", label)
	}
	return nil
}

func buildEvidenceBundle(session qratumSession, paths daemonArtifactPaths) (evidenceBundle, error) {
	if err := validateQratumSession(session, session.SessionID); err != nil {
		return evidenceBundle{}, err
	}

	lastFileChange, hasLastFileChange, _, missing, err := collectFileChanges(session.FileChanges)
	if err != nil {
		return evidenceBundle{}, err
	}
	commands, commandMissing, err := collectCommands(session.Commands)
	if err != nil {
		return evidenceBundle{}, err
	}
	missing = append(missing, commandMissing...)

	lastTestCommand, hasLastTestCommand := lastCommandMatching(commands, func(command qratumCommand) bool {
		return isTestCommand(command.Command)
	})
	lastSuccessfulVerification, hasLastSuccessfulVerification := lastCommandMatching(commands, func(command qratumCommand) bool {
		return isVerificationCommand(command.Command) && command.Success != nil && *command.Success
	})

	verificationCommands := filterCommands(commands, func(c qratumCommand) bool {
		return isVerificationCommand(c.Command)
	})
	onlyFailedVerification := hasOnlyFailedVerifications(verificationCommands)

	lastVerificationAfterEdit, hasLastVerificationAfterEdit := lastVerificationAfterFileChange(commands, lastFileChange, hasLastFileChange)
	finalVerificationFailed := hasLastVerificationAfterEdit &&
		lastVerificationAfterEdit.Command.Success != nil &&
		!*lastVerificationAfterEdit.Command.Success

	missingFinalVerification := hasLastFileChange &&
		(!hasLastSuccessfulVerification || !lastSuccessfulVerification.At.After(lastFileChange.At))

	findings := []evidenceFinding{}

	// Priority order: strongest deterministic signals first.
	if onlyFailedVerification {
		findings = append(findings, newOnlyFailedVerificationFinding(len(findings)+1, verificationCommands))
	}
	if finalVerificationFailed {
		findings = append(findings, newFinalVerificationFailedFinding(len(findings)+1, lastFileChange, lastVerificationAfterEdit))
	}
	for _, cmd := range filterCommands(commands, func(c qratumCommand) bool { return isDestructiveCommand(c.Command) }) {
		findings = append(findings, newDestructiveCommandFinding(len(findings)+1, cmd))
	}
	if missingFinalVerification {
		findings = append(findings, newMissingFinalVerificationFinding(len(findings)+1, lastFileChange, lastSuccessfulVerification, hasLastSuccessfulVerification))
	}
	if hasLastFileChange && hasLastTestCommand && lastFileChange.At.After(lastTestCommand.At) {
		findings = append(findings, newFinalEditAfterLastTestFinding(len(findings)+1, lastFileChange, lastTestCommand))
	}
	if hasLastFileChange && len(session.FileChanges) >= broadFileChangeThreshold &&
		(!hasLastSuccessfulVerification || finalVerificationFailed || onlyFailedVerification) {
		findings = append(findings, newBroadFileChangeFinding(len(findings)+1, len(session.FileChanges), lastFileChange, lastSuccessfulVerification, hasLastSuccessfulVerification))
	}
	for _, group := range repeatedFailingCommandGroups(commands) {
		findings = append(findings, newRepeatedFailingCommandFinding(len(findings)+1, group))
	}
	if hasNetworkCallWithoutPackageContext(commands, session) {
		for _, cmd := range filterCommands(commands, func(c qratumCommand) bool { return isNetworkCommand(c.Command) }) {
			findings = append(findings, newNetworkCallWithoutNeedFinding(len(findings)+1, cmd))
		}
	}
	if sourceChangedWithoutTestChange(session.FileChanges) {
		findings = append(findings, newSourceChangedWithoutTestFinding(len(findings)+1, session.FileChanges))
	}

	for _, finding := range findings {
		missing = append(missing, finding.MissingEvidence...)
	}
	missing = uniqueStrings(missing)

	summary := evidenceBundleSummary{
		Status:               evidenceStatusComplete,
		Source:               session.Source,
		StartedAt:            session.StartedAt,
		EndedAt:              session.EndedAt,
		SourceEventID:        session.SourceEventID,
		SourceEventType:      session.SourceEventType,
		SourceEventTimestamp: session.SourceEventTimestamp,
		FilesChanged:         len(session.FileChanges),
		CommandsRun:          len(commands),
		TestsRun:             countTestCommands(session.Commands),
	}
	if hasLastFileChange {
		summary.LastFileChangeAt = lastFileChange.Change.Timestamp
	}
	if hasLastTestCommand {
		summary.LastTestCommandAt = lastTestCommand.Command.Timestamp
	}
	if hasLastSuccessfulVerification {
		summary.LastSuccessfulVerifyAt = lastSuccessfulVerification.Command.Timestamp
	}

	return evidenceBundle{
		SchemaVersion:   qratumEvidenceSchemaVersion,
		DataClass:       qschema.DataClassReview,
		SessionID:       session.SessionID,
		Summary:         summary,
		Findings:        findings,
		MissingEvidence: missing,
		SourceEventID:   session.SourceEventID,
		ArtifactPaths:   paths,
	}, nil
}

func collectFileChanges(changes []qratumFileChange) (indexedFileChange, bool, []indexedFileChange, []string, error) {
	indexed := make([]indexedFileChange, 0, len(changes))
	missing := []string{}
	var last indexedFileChange
	for i, change := range changes {
		at, ok, err := parseEvidenceTimestamp(change.Timestamp, fmt.Sprintf("file_changes[%d].timestamp", i))
		if err != nil {
			return indexedFileChange{}, false, nil, nil, err
		}
		if !ok {
			missing = append(missing, fmt.Sprintf("file_changes[%d].timestamp", i))
			continue
		}
		item := indexedFileChange{Index: i, Change: change, At: at}
		indexed = append(indexed, item)
		if len(indexed) == 1 || item.At.After(last.At) || item.At.Equal(last.At) && item.Index > last.Index {
			last = item
		}
	}
	return last, len(indexed) > 0, indexed, missing, nil
}

func collectCommands(commands []qratumCommand) ([]indexedCommand, []string, error) {
	indexed := make([]indexedCommand, 0, len(commands))
	missing := []string{}
	for i, command := range commands {
		at, ok, err := parseEvidenceTimestamp(command.Timestamp, fmt.Sprintf("commands[%d].timestamp", i))
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			missing = append(missing, fmt.Sprintf("commands[%d].timestamp", i))
		}
		indexed = append(indexed, indexedCommand{Index: i, Command: command, At: at, HasTime: ok})
		if command.Success == nil {
			missing = append(missing, fmt.Sprintf("commands[%d].success", i))
		}
	}
	return indexed, missing, nil
}

func parseEvidenceTimestamp(value string, field string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%s must be RFC3339: %w", field, err)
	}
	return parsed, true, nil
}

func lastCommandMatching(commands []indexedCommand, matches func(qratumCommand) bool) (indexedCommand, bool) {
	var last indexedCommand
	hasLast := false
	for _, command := range commands {
		if !command.HasTime || !matches(command.Command) {
			continue
		}
		if !hasLast || command.At.After(last.At) || command.At.Equal(last.At) && command.Index > last.Index {
			last = command
			hasLast = true
		}
	}
	return last, hasLast
}

func filterCommands(commands []indexedCommand, matches func(qratumCommand) bool) []indexedCommand {
	out := make([]indexedCommand, 0, len(commands))
	for _, c := range commands {
		if matches(c.Command) {
			out = append(out, c)
		}
	}
	return out
}

func hasOnlyFailedVerifications(verificationCommands []indexedCommand) bool {
	if len(verificationCommands) == 0 {
		return false
	}
	for _, c := range verificationCommands {
		if c.Command.Success == nil || *c.Command.Success {
			return false
		}
	}
	return true
}

func lastVerificationAfterFileChange(commands []indexedCommand, lastFileChange indexedFileChange, hasLastFileChange bool) (indexedCommand, bool) {
	if !hasLastFileChange {
		return indexedCommand{}, false
	}
	var last indexedCommand
	has := false
	for _, c := range commands {
		if !c.HasTime || !isVerificationCommand(c.Command.Command) {
			continue
		}
		if !c.At.After(lastFileChange.At) {
			continue
		}
		if !has || c.At.After(last.At) || c.At.Equal(last.At) && c.Index > last.Index {
			last = c
			has = true
		}
	}
	return last, has
}

func repeatedFailingCommandGroups(commands []indexedCommand) [][]indexedCommand {
	groups := map[string][]indexedCommand{}
	firstIndex := map[string]int{}
	for _, command := range commands {
		if command.Command.Success == nil || *command.Command.Success {
			continue
		}
		normalized := normalizeCommandText(command.Command.Command)
		if normalized == "" {
			continue
		}
		if _, ok := groups[normalized]; !ok {
			firstIndex[normalized] = command.Index
		}
		groups[normalized] = append(groups[normalized], command)
	}

	keys := make([]string, 0, len(groups))
	for command, group := range groups {
		if len(group) >= 2 {
			keys = append(keys, command)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return firstIndex[keys[i]] < firstIndex[keys[j]]
	})

	out := make([][]indexedCommand, 0, len(keys))
	for _, key := range keys {
		out = append(out, groups[key])
	}
	return out
}

func newFinalEditAfterLastTestFinding(sequence int, change indexedFileChange, testCommand indexedCommand) evidenceFinding {
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingFinalEditAfterLastTest, sequence),
		Type:      findingFinalEditAfterLastTest,
		Title:     "Final edit happened after the last test",
		Summary: fmt.Sprintf("%s changed via %s at %s after the last test command at %s.",
			change.Change.Path, change.Change.Operation, change.Change.Timestamp, testCommand.Command.Timestamp),
		Evidence: []evidenceFact{
			fileChangeFact("final_file_change", change),
			commandFact("last_test_command", testCommand),
		},
		MissingEvidence: []string{},
	}
}

func newMissingFinalVerificationFinding(sequence int, change indexedFileChange, lastSuccess indexedCommand, hasLastSuccess bool) evidenceFinding {
	missing := []string{fmt.Sprintf("successful verification command after %s", change.Change.Timestamp)}
	evidence := []evidenceFact{fileChangeFact("final_file_change", change)}
	if hasLastSuccess {
		evidence = append(evidence, commandFact("last_successful_verification", lastSuccess))
	}
	return evidenceFinding{
		FindingID:       fmt.Sprintf("%s.%04d", findingMissingFinalVerification, sequence),
		Type:            findingMissingFinalVerification,
		Title:           "Final verification is missing",
		Summary:         fmt.Sprintf("No successful verification command ran after the final file change at %s.", change.Change.Timestamp),
		Evidence:        evidence,
		MissingEvidence: missing,
	}
}

func newFinalVerificationFailedFinding(sequence int, change indexedFileChange, failed indexedCommand) evidenceFinding {
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingFinalVerificationFailed, sequence),
		Type:      findingFinalVerificationFailed,
		Title:     "Final verification command failed",
		Summary: fmt.Sprintf("The last verification command %q after the final file change at %s failed at %s.",
			normalizeCommandText(failed.Command.Command), change.Change.Timestamp, failed.Command.Timestamp),
		Evidence: []evidenceFact{
			fileChangeFact("final_file_change", change),
			commandFact("last_verification_after_edit", failed),
		},
		MissingEvidence: []string{
			fmt.Sprintf("successful verification command after %s", failed.Command.Timestamp),
		},
	}
}

func newOnlyFailedVerificationFinding(sequence int, verifications []indexedCommand) evidenceFinding {
	evidence := make([]evidenceFact, 0, len(verifications))
	for i, c := range verifications {
		label := "failed_verification_command"
		if i == 0 {
			label = "first_failed_verification_command"
		}
		evidence = append(evidence, commandFact(label, c))
	}
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingOnlyFailedVerification, sequence),
		Type:      findingOnlyFailedVerification,
		Title:     "Verification never succeeded in this session",
		Summary: fmt.Sprintf("%d verification command(s) ran in this session and none succeeded.",
			len(verifications)),
		Evidence:        evidence,
		MissingEvidence: []string{"successful verification command at any point in this session"},
	}
}

func newDestructiveCommandFinding(sequence int, cmd indexedCommand) evidenceFinding {
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingDestructiveCommand, sequence),
		Type:      findingDestructiveCommand,
		Title:     "Destructive shell command ran",
		Summary: fmt.Sprintf("Destructive command %q ran at %s.",
			normalizeCommandText(cmd.Command.Command), cmd.Command.Timestamp),
		Evidence: []evidenceFact{
			commandFact("destructive_command", cmd),
		},
		MissingEvidence: []string{},
	}
}

func newNetworkCallWithoutNeedFinding(sequence int, cmd indexedCommand) evidenceFinding {
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingNetworkCallWithoutNeed, sequence),
		Type:      findingNetworkCallWithoutNeed,
		Title:     "Network or install command ran without a package task",
		Summary: fmt.Sprintf("Network or install command %q ran at %s with no package or dependency file change in this session.",
			normalizeCommandText(cmd.Command.Command), cmd.Command.Timestamp),
		Evidence: []evidenceFact{
			commandFact("network_command", cmd),
		},
		MissingEvidence: []string{},
	}
}

func newSourceChangedWithoutTestFinding(sequence int, changes []qratumFileChange) evidenceFinding {
	sources := classifiableSourcePaths(changes)
	evidence := make([]evidenceFact, 0, len(sources))
	for i, change := range sources {
		label := fmt.Sprintf("source_file_change_%d", i+1)
		evidence = append(evidence, evidenceFact{
			Label:     label,
			Kind:      "file_change",
			Timestamp: change.Timestamp,
			Path:      change.Path,
			Operation: change.Operation,
		})
	}
	summary := "Source files changed in this session but no test files changed."
	if len(sources) > 0 {
		summary = fmt.Sprintf("%d source file change(s) recorded in this session, but no test files changed.", len(sources))
	}
	return evidenceFinding{
		FindingID:       fmt.Sprintf("%s.%04d", findingSourceChangedWithoutTest, sequence),
		Type:            findingSourceChangedWithoutTest,
		Title:           "Source changed without test change",
		Summary:         summary,
		Evidence:        evidence,
		MissingEvidence: []string{"test file change in the same session"},
	}
}

func newBroadFileChangeFinding(sequence int, count int, lastChange indexedFileChange, lastSuccess indexedCommand, hasLastSuccess bool) evidenceFinding {
	evidence := []evidenceFact{
		fileChangeFact("final_file_change", lastChange),
	}
	if hasLastSuccess {
		evidence = append(evidence, commandFact("last_successful_verification", lastSuccess))
	}
	missing := []string{"successful verification command covering this broad change"}
	return evidenceFinding{
		FindingID: fmt.Sprintf("%s.%04d", findingBroadFileChange, sequence),
		Type:      findingBroadFileChange,
		Title:     "Broad multi-file change without successful verification",
		Summary: fmt.Sprintf("%d files changed in this session and no successful verification command covered the result.",
			count),
		Evidence:        evidence,
		MissingEvidence: missing,
	}
}

func newRepeatedFailingCommandFinding(sequence int, group []indexedCommand) evidenceFinding {
	command := normalizeCommandText(group[0].Command.Command)
	evidence := make([]evidenceFact, 0, len(group))
	for i, item := range group {
		label := "failed_command"
		switch i {
		case 0:
			label = "first_failed_command"
		case 1:
			label = "repeated_failed_command"
		}
		evidence = append(evidence, commandFact(label, item))
	}
	return evidenceFinding{
		FindingID:       fmt.Sprintf("%s.%04d", findingRepeatedFailingCommand, sequence),
		Type:            findingRepeatedFailingCommand,
		Title:           "Command failed repeatedly",
		Summary:         fmt.Sprintf("%q failed %d times in this session.", command, len(group)),
		Evidence:        evidence,
		MissingEvidence: []string{},
	}
}

func fileChangeFact(label string, change indexedFileChange) evidenceFact {
	return evidenceFact{
		Label:     label,
		Kind:      "file_change",
		Timestamp: change.Change.Timestamp,
		Path:      change.Change.Path,
		Operation: change.Change.Operation,
	}
}

func commandFact(label string, command indexedCommand) evidenceFact {
	return evidenceFact{
		Label:         label,
		Kind:          "command",
		Timestamp:     command.Command.Timestamp,
		Command:       command.Command.Command,
		Success:       command.Command.Success,
		OutputExcerpt: outputExcerpt(command.Command.Output),
	}
}

func outputExcerpt(output string) string {
	output = strings.TrimSpace(strings.ReplaceAll(output, "\r\n", "\n"))
	if output == "" {
		return ""
	}
	runes := []rune(output)
	if len(runes) <= 160 {
		return output
	}
	return string(runes[:157]) + "..."
}

func isTestCommand(command string) bool {
	return strings.Contains(strings.ToLower(command), "test")
}

func isVerificationCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if strings.Contains(lower, "test") {
		return true
	}
	for _, token := range []string{"make build", "make demo", "make verify", "make lint", "make vet", "go vet", "go build", "golangci-lint"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// isDestructiveCommand flags obvious destructive shell patterns. Conservative
// by design: pattern must appear as a contiguous fragment of the command.
func isDestructiveCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return false
	}
	patterns := []string{
		"rm -rf",
		"rm -fr",
		"rm -r -f",
		"rm -f -r",
		"git reset --hard",
		"chmod -r",
		"chmod -rf",
		"chown -r",
		"chown -rf",
		"sudo rm",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// isNetworkCommand flags curl/wget and common package install commands. Returns
// true only when the network/install verb is the *first* token (or follows sudo),
// to avoid flagging substrings like comments or pipeline tails.
func isNetworkCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return false
	}
	fields := strings.Fields(lower)
	if len(fields) == 0 {
		return false
	}
	head := fields[0]
	rest := fields[1:]
	if head == "sudo" && len(rest) > 0 {
		head = rest[0]
		rest = rest[1:]
	}
	switch head {
	case "curl", "wget":
		return true
	case "npm", "pnpm", "yarn":
		if len(rest) > 0 && (rest[0] == "install" || rest[0] == "i" || rest[0] == "add") {
			return true
		}
	case "pip", "pip3":
		if len(rest) > 0 && rest[0] == "install" {
			return true
		}
	case "brew":
		if len(rest) > 0 && rest[0] == "install" {
			return true
		}
	case "apt", "apt-get":
		if len(rest) > 0 && rest[0] == "install" {
			return true
		}
	case "go":
		if len(rest) > 0 && (rest[0] == "install" || rest[0] == "get") {
			return true
		}
	case "cargo":
		if len(rest) > 0 && rest[0] == "install" {
			return true
		}
	}
	return false
}

// hasNetworkCallWithoutPackageContext returns true if the session has any network
// command AND no dependency-manifest file change appears in the session.
func hasNetworkCallWithoutPackageContext(commands []indexedCommand, session qratumSession) bool {
	hasNetwork := false
	for _, c := range commands {
		if isNetworkCommand(c.Command.Command) {
			hasNetwork = true
			break
		}
	}
	if !hasNetwork {
		return false
	}
	return !sessionHasDependencyManifestChange(session.FileChanges)
}

func sessionHasDependencyManifestChange(changes []qratumFileChange) bool {
	for _, fc := range changes {
		base := strings.ToLower(filepath.Base(fc.Path))
		switch base {
		case "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
			"go.mod", "go.sum",
			"requirements.txt", "pyproject.toml", "poetry.lock", "pipfile", "pipfile.lock",
			"cargo.toml", "cargo.lock",
			"gemfile", "gemfile.lock",
			"composer.json", "composer.lock":
			return true
		}
	}
	return false
}

// isRedactedPath is true when the path has been deterministically redacted and
// can no longer be classified by suffix or directory.
func isRedactedPath(path string) bool {
	return strings.HasPrefix(strings.TrimSpace(path), "[REDACTED_PATH_")
}

func isTestFilePath(path string) bool {
	if path == "" || isRedactedPath(path) {
		return false
	}
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	if strings.HasSuffix(base, "_test.py") {
		return true
	}
	if strings.HasPrefix(base, "test_") && (strings.HasSuffix(base, ".py") || strings.HasSuffix(base, ".rb")) {
		return true
	}
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	slashed := "/" + strings.ReplaceAll(lower, "\\", "/")
	for _, dir := range []string{"/tests/", "/test/", "/__tests__/", "/spec/"} {
		if strings.Contains(slashed, dir) {
			return true
		}
	}
	return false
}

// isSourceFilePath returns true for likely source code files. Excludes docs,
// configs, fully-redacted paths, and known test files.
func isSourceFilePath(path string) bool {
	if path == "" || isRedactedPath(path) || isTestFilePath(path) {
		return false
	}
	lower := strings.ToLower(filepath.Base(path))
	for _, suffix := range []string{
		".md", ".txt", ".rst", ".adoc",
		".yaml", ".yml", ".json", ".toml", ".ini", ".lock", ".env",
		".gitignore", ".gitattributes",
	} {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	// Require a recognized source extension to avoid false positives.
	for _, suffix := range []string{
		".go", ".py", ".rb", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		".java", ".kt", ".scala", ".rs", ".c", ".cc", ".cpp", ".h", ".hpp",
		".cs", ".php", ".sh", ".bash", ".zsh", ".swift", ".m", ".mm",
		".html", ".css", ".scss", ".sass", ".vue", ".svelte",
	} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func classifiableSourcePaths(changes []qratumFileChange) []qratumFileChange {
	out := make([]qratumFileChange, 0, len(changes))
	for _, fc := range changes {
		if isSourceFilePath(fc.Path) {
			out = append(out, fc)
		}
	}
	return out
}

func sourceChangedWithoutTestChange(changes []qratumFileChange) bool {
	hasSource := false
	for _, fc := range changes {
		if isTestFilePath(fc.Path) {
			return false
		}
		if isSourceFilePath(fc.Path) {
			hasSource = true
		}
	}
	return hasSource
}

func normalizeCommandText(command string) string {
	return strings.Join(strings.Fields(command), " ")
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func readEvidenceBundleFile(path string, projectRoot string) (evidenceBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return evidenceBundle{}, fmt.Errorf("read evidence %s: %w", displayPath(projectRoot, path), err)
	}
	var bundle evidenceBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return evidenceBundle{}, fmt.Errorf("invalid evidence JSON %s: %w", displayPath(projectRoot, path), err)
	}
	if err := validateEvidenceBundle(bundle, displayPath(projectRoot, path)); err != nil {
		return evidenceBundle{}, err
	}
	return bundle, nil
}

func validateEvidenceBundle(bundle evidenceBundle, label string) error {
	if strings.TrimSpace(bundle.SchemaVersion) != qratumEvidenceSchemaVersion {
		return fmt.Errorf("evidence %s has unsupported schema_version %q", label, bundle.SchemaVersion)
	}
	if strings.TrimSpace(bundle.SessionID) == "" {
		return fmt.Errorf("evidence %s is missing session_id", label)
	}
	if strings.TrimSpace(bundle.Summary.Status) != evidenceStatusComplete {
		return fmt.Errorf("evidence %s has unsupported summary.status %q", label, bundle.Summary.Status)
	}
	if strings.TrimSpace(bundle.Summary.Source) != claudeCodeSource {
		return fmt.Errorf("evidence %s has unsupported summary.source %q", label, bundle.Summary.Source)
	}
	if bundle.Findings == nil {
		return fmt.Errorf("evidence %s is missing findings", label)
	}
	if bundle.MissingEvidence == nil {
		return fmt.Errorf("evidence %s is missing missing_evidence", label)
	}
	for i, finding := range bundle.Findings {
		if strings.TrimSpace(finding.FindingID) == "" {
			return fmt.Errorf("evidence %s is missing findings[%d].finding_id", label, i)
		}
		if _, ok := supportedEvidenceFindingTypes[finding.Type]; !ok {
			return fmt.Errorf("evidence %s has unsupported findings[%d].type %q", label, i, finding.Type)
		}
		if strings.TrimSpace(finding.Title) == "" {
			return fmt.Errorf("evidence %s is missing findings[%d].title", label, i)
		}
		if strings.TrimSpace(finding.Summary) == "" {
			return fmt.Errorf("evidence %s is missing findings[%d].summary", label, i)
		}
		if finding.Evidence == nil {
			return fmt.Errorf("evidence %s is missing findings[%d].evidence", label, i)
		}
		if finding.MissingEvidence == nil {
			return fmt.Errorf("evidence %s is missing findings[%d].missing_evidence", label, i)
		}
	}
	return nil
}

func buildReviewCard(bundle evidenceBundle) (reviewCard, error) {
	if err := validateEvidenceBundle(bundle, bundle.SessionID); err != nil {
		return reviewCard{}, err
	}

	card := reviewCard{
		SchemaVersion: qratumReviewSchemaVersion,
		DataClass:     qschema.DataClassReview,
		SessionID:     bundle.SessionID,
		Warnings:      append([]string{}, bundle.MissingEvidence...),
		SourceEventID: bundle.SourceEventID,
		ArtifactPaths: bundle.ArtifactPaths,
	}
	if len(card.Warnings) == 0 {
		card.Warnings = []string{}
	}

	if len(bundle.Findings) == 0 {
		card.Verdict = "clean"
		card.MainFinding = "No evidence findings were detected in this session."
		card.Evidence = buildCleanReviewProof(bundle.Summary)
		card.SuggestedNextHabit = "Keep ending sessions with a visible verification command."
		return card, nil
	}

	main := bundle.Findings[0]
	card.Verdict = "needs_attention"
	card.MainFinding = main.Summary
	card.Evidence = reviewEvidenceStrings(bundle)
	card.SuggestedNextHabit, card.SuggestedSkill = reviewHabitAndSkill(main.Type)
	return card, nil
}

func buildCleanReviewProof(summary evidenceBundleSummary) []string {
	out := []string{
		fmt.Sprintf("files_changed: %d", summary.FilesChanged),
		fmt.Sprintf("commands_run: %d", summary.CommandsRun),
		fmt.Sprintf("tests_run: %d", summary.TestsRun),
	}
	if summary.LastFileChangeAt != "" {
		out = append(out, fmt.Sprintf("final_file_edit_at: %s", summary.LastFileChangeAt))
	}
	if summary.LastTestCommandAt != "" {
		out = append(out, fmt.Sprintf("last_test_command_at: %s", summary.LastTestCommandAt))
	}
	if summary.LastSuccessfulVerifyAt != "" {
		out = append(out, fmt.Sprintf("last_successful_verification_at: %s", summary.LastSuccessfulVerifyAt))
	}
	if summary.LastFileChangeAt != "" && summary.LastSuccessfulVerifyAt != "" {
		fileAt, fileErr := time.Parse(time.RFC3339, summary.LastFileChangeAt)
		verifyAt, verifyErr := time.Parse(time.RFC3339, summary.LastSuccessfulVerifyAt)
		if fileErr == nil && verifyErr == nil {
			out = append(out, fmt.Sprintf("verification_after_final_edit: %t", verifyAt.After(fileAt) || verifyAt.Equal(fileAt)))
		}
	}
	return out
}

func reviewEvidenceStrings(bundle evidenceBundle) []string {
	out := []string{}
	for _, finding := range bundle.Findings {
		out = append(out, finding.Summary)
		for _, fact := range finding.Evidence {
			out = append(out, formatEvidenceFact(fact))
		}
		for _, missing := range finding.MissingEvidence {
			out = append(out, "missing: "+missing)
		}
	}
	return uniqueStrings(out)
}

func formatEvidenceFact(fact evidenceFact) string {
	parts := []string{}
	if fact.Timestamp != "" {
		parts = append(parts, fact.Timestamp)
	}
	switch fact.Kind {
	case "file_change":
		text := strings.TrimSpace(strings.Join([]string{fact.Operation, fact.Path}, " "))
		if text != "" {
			parts = append(parts, text)
		}
	case "command":
		if fact.Command != "" {
			parts = append(parts, fmt.Sprintf("command %q", fact.Command))
		}
		if fact.Success != nil {
			if *fact.Success {
				parts = append(parts, "succeeded")
			} else {
				parts = append(parts, "failed")
			}
		}
		if fact.OutputExcerpt != "" {
			parts = append(parts, "output: "+fact.OutputExcerpt)
		}
	default:
		if fact.Label != "" {
			parts = append(parts, fact.Label)
		}
	}
	if len(parts) == 0 {
		return fact.Label
	}
	return strings.Join(parts, " | ")
}

func reviewHabitAndSkill(findingType string) (string, string) {
	switch findingType {
	case findingFinalEditAfterLastTest, findingMissingFinalVerification,
		findingFinalVerificationFailed, findingOnlyFailedVerification:
		return "After the last edit, run the project's verification command and keep the result in the session.", "final-verification-loop"
	case findingRepeatedFailingCommand:
		return "When a command fails twice, change the code or command before running it again.", "debug-failing-command-loop"
	case findingDestructiveCommand:
		return "Before running destructive shell commands, confirm exactly what they will delete and avoid running them inside an unrelated task.", "destructive-command-guard"
	case findingNetworkCallWithoutNeed:
		return "Avoid network or install commands unless the task explicitly adds or upgrades a dependency.", "network-call-guard"
	case findingSourceChangedWithoutTest:
		return "When you change a source file, add or update a test in the same session.", "source-with-test-loop"
	case findingBroadFileChange:
		return "Before making broad multi-file changes, run a fast verification command and capture its result.", "broad-change-verification-loop"
	default:
		return "Keep the next session evidence explicit and deterministic.", ""
	}
}
