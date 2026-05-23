package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const dogfoodImportStatus = "dogfood_imported"

type dogfoodLatestReview struct {
	card       reviewCard
	evidence   evidenceBundle
	reportPath string
	exportPath string
}

func dogfood(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing dogfood command")
		return 2
	}

	switch args[0] {
	case "import":
		if len(args) == 1 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: missing transcript path")
			return 2
		}
		if len(args) != 2 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: dogfood import accepts exactly one transcript path")
			return 2
		}
		return dogfoodImport(args[1], stdout, stderr)
	case "latest":
		if len(args) != 1 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: dogfood latest does not accept arguments")
			return 2
		}
		return dogfoodLatest(stdout, stderr)
	case "list":
		if len(args) != 1 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: dogfood list does not accept arguments")
			return 2
		}
		return dogfoodList(stdout, stderr)
	case "show":
		if len(args) == 1 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: missing session_id")
			return 2
		}
		if len(args) != 2 {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: dogfood show accepts exactly one session_id")
			return 2
		}
		return dogfoodShow(args[1], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unsupported dogfood command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func dogfoodImport(inputPath string, stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	transcriptPath, err := resolveTranscriptPath(projectRoot, inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid transcript path: %v\n", err)
		return 1
	}
	if err := requireTranscriptFile(transcriptPath, projectRoot, inputPath); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	session, err := normalizeClaudeTranscriptFile(transcriptPath, normalizeSessionContext{
		PipelineStatus: dogfoodImportStatus,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: dogfood import failed: normalize transcript %s: %v\n", displayPath(projectRoot, transcriptPath), err)
		return 1
	}

	stem, err := artifactStemForSession(session.SessionID)
	if err != nil {
		fmt.Fprintf(stderr, "error: dogfood import failed: %v\n", err)
		return 1
	}
	paths := artifactPathsForStem(stem)
	paths.Event = ""
	session.PipelineStatus = dogfoodImportStatus
	session.TranscriptPath = ""
	session.ArtifactPaths = &paths

	if err := writeDogfoodArtifacts(projectRoot, paths, session); err != nil {
		fmt.Fprintf(stderr, "error: dogfood import failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum dogfood import")
	fmt.Fprintf(stdout, "session_id: %s\n", session.SessionID)
	fmt.Fprintf(stdout, "review_path: %s\n", paths.Review)
	fmt.Fprintf(stdout, "html_report_path: %s\n", paths.Report)
	fmt.Fprintf(stdout, "adp_export_path: %s\n", paths.Export)
	return 0
}

func writeDogfoodArtifacts(projectRoot string, paths daemonArtifactPaths, session qratumSession) error {
	redactedSession, err := redactQratumSession(session)
	if err != nil {
		return fmt.Errorf("redact session %s: %w", session.SessionID, err)
	}

	safeSession := redactedSession
	safeSession.PipelineStatus = dogfoodImportStatus
	safeSession.Redaction = nil
	safeSession.ArtifactPaths = &paths
	redactedSession.ArtifactPaths = &paths

	evidenceBundle, err := buildEvidenceBundle(redactedSession, paths)
	if err != nil {
		return fmt.Errorf("build evidence for session %s: %w", session.SessionID, err)
	}
	reviewCard, err := buildReviewCard(evidenceBundle)
	if err != nil {
		return fmt.Errorf("build review for session %s: %w", session.SessionID, err)
	}
	adpExport, err := buildADPStrictJSONL(redactedSession)
	if err != nil {
		return fmt.Errorf("build ADP strict export for session %s: %w", session.SessionID, err)
	}

	for _, file := range artifactFilesForPaths(projectRoot, paths) {
		if strings.TrimSpace(file.rel) == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.abs), 0o755); err != nil {
			return fmt.Errorf("create artifact directory for %s: %w", file.rel, err)
		}
	}

	writes := []struct {
		path string
		data []byte
	}{
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Session)), mustJSON(safeSession)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Redacted)), mustJSON(redactedSession)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Evidence)), mustJSON(evidenceBundle)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Review)), mustJSON(reviewCard)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Export)), adpExport},
	}

	for _, write := range writes {
		if err := writeFileAtomic(write.path, write.data, 0o644); err != nil {
			return fmt.Errorf("write artifact %s: %w", displayPath(projectRoot, write.path), err)
		}
	}

	reportData, err := buildReportDocument(projectRoot, reportContext{
		session:     safeSession,
		redacted:    redactedSession,
		evidence:    evidenceBundle,
		review:      reviewCard,
		paths:       paths,
		sessionPath: paths.Session,
		artifactsAt: uiArtifactCreatedAt(safeSession, evidenceBundle),
	})
	if err != nil {
		return fmt.Errorf("build report for session %s: %w", session.SessionID, err)
	}
	reportPath := filepath.Join(projectRoot, filepath.FromSlash(paths.Report))
	if err := writeFileAtomic(reportPath, reportData, 0o644); err != nil {
		return fmt.Errorf("write artifact %s: %w", displayPath(projectRoot, reportPath), err)
	}
	return nil
}

func dogfoodLatest(stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	latest, err := findLatestDogfoodReview(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "session_id: %s\n", latest.card.SessionID)
	fmt.Fprintf(stdout, "verdict: %s\n", latest.card.Verdict)
	fmt.Fprintf(stdout, "main_finding: %s\n", latest.card.MainFinding)
	writeDogfoodFindingsList(stdout, "top_findings", latest.evidence.Findings, 5)
	writeDogfoodEvidenceList(stdout, "evidence", latest.card.Evidence, 6)
	fmt.Fprintf(stdout, "suggested_next_habit: %s\n", latest.card.SuggestedNextHabit)
	fmt.Fprintf(stdout, "html_report_path: %s\n", latest.reportPath)
	fmt.Fprintf(stdout, "adp_export_path: %s\n", latest.exportPath)
	return 0
}

func writeDogfoodFindingsList(w io.Writer, label string, findings []evidenceFinding, max int) {
	fmt.Fprintf(w, "%s:\n", label)
	if len(findings) == 0 {
		fmt.Fprintln(w, "- (none)")
		return
	}
	for i, finding := range findings {
		if i >= max {
			break
		}
		fmt.Fprintf(w, "- %s: %s\n", finding.Type, finding.Summary)
	}
}

func writeDogfoodEvidenceList(w io.Writer, label string, evidence []string, max int) {
	fmt.Fprintf(w, "%s:\n", label)
	if len(evidence) == 0 {
		fmt.Fprintln(w, "- (none)")
		return
	}
	for i, item := range evidence {
		if i >= max {
			break
		}
		fmt.Fprintf(w, "- %s\n", item)
	}
}

func findLatestDogfoodReview(projectRoot string) (dogfoodLatestReview, error) {
	entries, err := loadDogfoodSessions(projectRoot)
	if err != nil {
		return dogfoodLatestReview{}, err
	}
	if len(entries) == 0 {
		return dogfoodLatestReview{}, fmt.Errorf("no processed dogfood sessions found")
	}
	latest := entries[0]
	return dogfoodLatestReview{
		card:       latest.review,
		evidence:   latest.evidence,
		reportPath: latest.paths.Report,
		exportPath: latest.paths.Export,
	}, nil
}

type dogfoodSessionEntry struct {
	session  qratumSession
	evidence evidenceBundle
	review   reviewCard
	paths    daemonArtifactPaths
}

func dogfoodList(stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	entries, err := loadDogfoodSessions(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "error: no processed dogfood sessions found")
		return 1
	}

	for i, entry := range entries {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "session_id: %s\n", entry.session.SessionID)
		fmt.Fprintf(stdout, "source: %s\n", entry.session.Source)
		fmt.Fprintf(stdout, "started_at: %s\n", dashIfEmpty(entry.session.StartedAt))
		fmt.Fprintf(stdout, "ended_at: %s\n", dashIfEmpty(entry.session.EndedAt))
		fmt.Fprintf(stdout, "verdict: %s\n", entry.review.Verdict)
		fmt.Fprintf(stdout, "main_finding: %s\n", entry.review.MainFinding)
		fmt.Fprintf(stdout, "files_changed: %d\n", entry.evidence.Summary.FilesChanged)
		fmt.Fprintf(stdout, "commands_run: %d\n", entry.evidence.Summary.CommandsRun)
		fmt.Fprintf(stdout, "tests_run: %d\n", entry.evidence.Summary.TestsRun)
		fmt.Fprintf(stdout, "report_path: %s\n", entry.paths.Report)
	}
	return 0
}

func dogfoodShow(sessionID string, stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		fmt.Fprintln(stderr, "error: missing session_id")
		return 2
	}

	entries, err := loadDogfoodSessions(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	var match *dogfoodSessionEntry
	for i := range entries {
		if entries[i].session.SessionID == sessionID {
			match = &entries[i]
			break
		}
	}
	if match == nil {
		fmt.Fprintf(stderr, "error: session %q not found among processed dogfood sessions\n", sessionID)
		return 1
	}

	fmt.Fprintf(stdout, "session_id: %s\n", match.session.SessionID)
	fmt.Fprintf(stdout, "source: %s\n", match.session.Source)
	fmt.Fprintf(stdout, "agent_model: %s\n", dashIfEmpty(match.session.AgentModel))
	fmt.Fprintf(stdout, "started_at: %s\n", dashIfEmpty(match.session.StartedAt))
	fmt.Fprintf(stdout, "ended_at: %s\n", dashIfEmpty(match.session.EndedAt))
	fmt.Fprintf(stdout, "verdict: %s\n", match.review.Verdict)
	fmt.Fprintf(stdout, "main_finding: %s\n", match.review.MainFinding)
	fmt.Fprintln(stdout, "findings:")
	if len(match.evidence.Findings) == 0 {
		fmt.Fprintln(stdout, "- (none)")
	} else {
		for _, finding := range match.evidence.Findings {
			fmt.Fprintf(stdout, "- %s: %s\n", finding.Type, finding.Summary)
		}
	}
	fmt.Fprintln(stdout, "evidence:")
	if len(match.review.Evidence) == 0 {
		fmt.Fprintln(stdout, "- (none)")
	} else {
		for _, item := range match.review.Evidence {
			fmt.Fprintf(stdout, "- %s\n", item)
		}
	}
	fmt.Fprintf(stdout, "suggested_next_habit: %s\n", match.review.SuggestedNextHabit)
	fmt.Fprintf(stdout, "files_changed: %d\n", match.evidence.Summary.FilesChanged)
	fmt.Fprintf(stdout, "commands_run: %d\n", match.evidence.Summary.CommandsRun)
	fmt.Fprintf(stdout, "tests_run: %d\n", match.evidence.Summary.TestsRun)
	fmt.Fprintf(stdout, "last_file_change_at: %s\n", dashIfEmpty(match.evidence.Summary.LastFileChangeAt))
	fmt.Fprintf(stdout, "last_test_command_at: %s\n", dashIfEmpty(match.evidence.Summary.LastTestCommandAt))
	fmt.Fprintf(stdout, "last_successful_verification_at: %s\n", dashIfEmpty(match.evidence.Summary.LastSuccessfulVerifyAt))
	fmt.Fprintf(stdout, "html_report_path: %s\n", match.paths.Report)
	fmt.Fprintf(stdout, "adp_export_path: %s\n", match.paths.Export)
	return 0
}

func loadDogfoodSessions(projectRoot string) ([]dogfoodSessionEntry, error) {
	sessionsDir := filepath.Join(projectRoot, ".qratum", "sessions")
	info, err := os.Stat(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sessions directory %s does not exist", displayPath(projectRoot, sessionsDir))
		}
		return nil, fmt.Errorf("inspect sessions directory %s: %w", displayPath(projectRoot, sessionsDir), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("sessions path %s is not a directory", displayPath(projectRoot, sessionsDir))
	}

	paths, err := filepath.Glob(filepath.Join(sessionsDir, "*.normalized.json"))
	if err != nil {
		return nil, fmt.Errorf("list sessions directory %s: %w", displayPath(projectRoot, sessionsDir), err)
	}
	sort.Strings(paths)

	entries := make([]dogfoodSessionEntry, 0, len(paths))
	for _, path := range paths {
		session, err := readQratumSessionFile(path, projectRoot)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(session.PipelineStatus) != dogfoodImportStatus {
			continue
		}
		artifactPaths, err := artifactPathsForSession(projectRoot, session, path)
		if err != nil {
			return nil, fmt.Errorf("resolve artifact paths for session %s: %w", session.SessionID, err)
		}
		reviewPath, err := resolveOptionalProjectFilePath(projectRoot, artifactPaths.Review)
		if err != nil {
			return nil, err
		}
		if reviewPath == "" {
			continue
		}
		evidencePath, err := resolveOptionalProjectFilePath(projectRoot, artifactPaths.Evidence)
		if err != nil {
			return nil, err
		}
		if evidencePath == "" {
			continue
		}
		bundle, err := readEvidenceBundleFile(evidencePath, projectRoot)
		if err != nil {
			return nil, err
		}
		card, err := readReviewCardFile(reviewPath, projectRoot)
		if err != nil {
			return nil, err
		}
		entries = append(entries, dogfoodSessionEntry{
			session:  session,
			evidence: bundle,
			review:   card,
			paths:    artifactPaths,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return dogfoodSortKey(entries[i]) > dogfoodSortKey(entries[j])
	})
	return entries, nil
}

func dogfoodSortKey(entry dogfoodSessionEntry) string {
	if v := strings.TrimSpace(entry.session.EndedAt); v != "" {
		return v
	}
	if v := strings.TrimSpace(entry.session.StartedAt); v != "" {
		return v
	}
	return ""
}

func resolveOptionalProjectFilePath(projectRoot string, inputPath string) (string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return "", nil
	}
	resolved, err := resolveProjectFilePath(projectRoot, inputPath, "artifact")
	if err != nil {
		if strings.Contains(err.Error(), "missing artifact") {
			return "", nil
		}
		return "", err
	}
	return resolved, nil
}

func dashIfEmpty(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
