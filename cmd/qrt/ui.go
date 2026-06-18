package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	qschema "github.com/edictum-ai/qratum/internal/schema"
	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	qratumUISessionListItemSchemaVersion = "qratum.ui.session_list_item.v1"
	qratumUISessionDetailSchemaVersion   = "qratum.ui.session_detail.v1"
	qratumUIReviewCardSchemaVersion      = "qratum.ui.review_card.v1"
	qratumUIEvidenceFindingSchemaVersion = "qratum.ui.evidence_finding.v1"
	qratumUIArtifactLinkSchemaVersion    = "qratum.ui.artifact_link.v1"
)

type uiSessionListItem struct {
	SchemaVersion string           `json:"schema_version"`
	DataClass     string           `json:"data_class"`
	SessionID     string           `json:"session_id"`
	Source        string           `json:"source"`
	AgentModel    string           `json:"agent_model,omitempty"`
	Repo          uiRepoSummary    `json:"repo"`
	Review        uiReviewSummary  `json:"review"`
	Metrics       uiSessionMetrics `json:"metrics"`
	Artifacts     []uiArtifactLink `json:"artifacts"`
}

type uiSessionDetail struct {
	SchemaVersion string              `json:"schema_version"`
	DataClass     string              `json:"data_class"`
	SessionID     string              `json:"session_id"`
	Source        string              `json:"source"`
	AgentModel    string              `json:"agent_model,omitempty"`
	Repo          uiRepoSummary       `json:"repo"`
	Time          uiSessionTime       `json:"time"`
	Summary       uiSessionSummary    `json:"summary"`
	Review        uiReviewCardDTO     `json:"review"`
	Findings      []uiEvidenceFinding `json:"findings"`
	Artifacts     []uiArtifactLink    `json:"artifacts"`
}

type uiRepoSummary struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	RepoID      string `json:"repo_id,omitempty"`
	CWD         string `json:"cwd,omitempty"`
	GitRemote   string `json:"git_remote,omitempty"`
	GitBranch   string `json:"git_branch,omitempty"`
	GitHeadSHA  string `json:"git_head_sha,omitempty"`
}

type uiReviewSummary struct {
	Verdict       string   `json:"verdict"`
	MainFinding   string   `json:"main_finding"`
	Warnings      []string `json:"warnings"`
	FindingsCount int      `json:"findings_count"`
	EvidenceCount int      `json:"evidence_count"`
}

type uiSessionMetrics struct {
	DurationSeconds int `json:"duration_seconds"`
	ToolCalls       int `json:"tool_calls"`
	FilesChanged    int `json:"files_changed"`
	CommandsRun     int `json:"commands_run"`
	TestsRun        int `json:"tests_run"`
	Findings        int `json:"findings"`
	Warnings        int `json:"warnings"`
}

type uiSessionTime struct {
	StartedAt            string `json:"started_at,omitempty"`
	EndedAt              string `json:"ended_at,omitempty"`
	DurationSeconds      int    `json:"duration_seconds"`
	SourceEventTimestamp string `json:"source_event_timestamp,omitempty"`
}

type uiSessionSummary struct {
	Status                    string   `json:"status"`
	SourceEventID             string   `json:"source_event_id,omitempty"`
	SourceEventType           string   `json:"source_event_type,omitempty"`
	SourceEventTimestamp      string   `json:"source_event_timestamp,omitempty"`
	SourceTranscriptSessionID string   `json:"source_transcript_session_id,omitempty"`
	Turns                     int      `json:"turns"`
	ToolCalls                 int      `json:"tool_calls"`
	FilesChanged              int      `json:"files_changed"`
	CommandsRun               int      `json:"commands_run"`
	TestsRun                  int      `json:"tests_run"`
	MissingEvidence           []string `json:"missing_evidence"`
}

type uiReviewCardDTO struct {
	SchemaVersion      string              `json:"schema_version"`
	DataClass          string              `json:"data_class"`
	SessionID          string              `json:"session_id"`
	Verdict            string              `json:"verdict"`
	MainFinding        string              `json:"main_finding"`
	Evidence           []string            `json:"evidence"`
	SuggestedNextHabit string              `json:"suggested_next_habit"`
	SuggestedSkill     string              `json:"suggested_skill,omitempty"`
	Warnings           []string            `json:"warnings"`
	Findings           []uiEvidenceFinding `json:"findings"`
	Artifacts          []uiArtifactLink    `json:"artifacts"`
}

type uiEvidenceFinding struct {
	SchemaVersion   string           `json:"schema_version"`
	DataClass       string           `json:"data_class"`
	FindingID       string           `json:"finding_id"`
	Type            string           `json:"type"`
	Severity        string           `json:"severity"`
	Confidence      string           `json:"confidence"`
	Title           string           `json:"title"`
	Summary         string           `json:"summary"`
	Evidence        []uiEvidenceFact `json:"evidence"`
	MissingEvidence []string         `json:"missing_evidence"`
}

type uiEvidenceFact struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	Display       string `json:"display"`
	Timestamp     string `json:"timestamp,omitempty"`
	Path          string `json:"path,omitempty"`
	Operation     string `json:"operation,omitempty"`
	Command       string `json:"command,omitempty"`
	Success       *bool  `json:"success,omitempty"`
	OutputExcerpt string `json:"output_excerpt,omitempty"`
}

type uiArtifactLink struct {
	SchemaVersion string `json:"schema_version"`
	DataClass     string `json:"data_class"`
	ArtifactID    string `json:"artifact_id"`
	SessionID     string `json:"session_id"`
	Type          string `json:"type"`
	Label         string `json:"label"`
	MediaType     string `json:"media_type"`
	Href          string `json:"href"`
	Digest        string `json:"digest"`
	CreatedAt     string `json:"created_at"`
}

type uiSessionContext struct {
	session   qratumSession
	redacted  qratumSession
	evidence  evidenceBundle
	review    reviewCard
	artifacts []uiArtifactLink
}

type uiFindingPresentation struct {
	severity   string
	confidence string
}

var uiSupportedFindingTypes = map[string]uiFindingPresentation{
	findingFinalEditAfterLastTest: {
		severity:   "medium",
		confidence: "high",
	},
	findingMissingFinalVerification: {
		severity:   "high",
		confidence: "high",
	},
	findingFinalVerificationFailed: {
		severity:   "high",
		confidence: "high",
	},
	findingOnlyFailedVerification: {
		severity:   "high",
		confidence: "high",
	},
	findingRepeatedFailingCommand: {
		severity:   "medium",
		confidence: "high",
	},
	findingDestructiveCommand: {
		severity:   "high",
		confidence: "high",
	},
	findingNetworkCallWithoutNeed: {
		severity:   "medium",
		confidence: "medium",
	},
	findingSourceChangedWithoutTest: {
		severity:   "medium",
		confidence: "medium",
	},
	findingBroadFileChange: {
		severity:   "medium",
		confidence: "medium",
	},
}

func ui(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing ui command")
		return 2
	}

	switch args[0] {
	case "sessions":
		if len(args) != 2 || args[1] != "--json" {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: ui sessions requires --json")
			return 2
		}
		return uiSessionsJSON(stdout, stderr)
	case "session":
		if len(args) != 3 || args[2] != "--json" {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: ui session accepts <session_id> --json")
			return 2
		}
		return uiSessionJSON(args[1], stdout, stderr)
	case "review":
		if len(args) != 3 || args[2] != "--json" {
			printUsage(stderr)
			fmt.Fprintln(stderr, "error: ui review accepts <session_id> --json")
			return 2
		}
		return uiReviewJSON(args[1], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unsupported ui command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func uiSessionsJSON(stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		writeAPIError(stderr, "ui.sessions_failed", err.Error())
		return 1
	}

	contexts, err := loadUISessionContexts(projectRoot)
	if err != nil {
		writeAPIError(stderr, "ui.sessions_failed", err.Error())
		return 1
	}

	items := make([]uiSessionListItem, 0, len(contexts))
	for _, context := range contexts {
		items = append(items, buildUISessionListItem(context))
	}
	if err := writeJSON(stdout, items); err != nil {
		writeAPIError(stderr, "ui.sessions_failed", err.Error())
		return 1
	}
	return 0
}

func uiSessionJSON(sessionID string, stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		writeAPIError(stderr, "ui.session_failed", err.Error())
		return 1
	}

	context, err := loadUISessionContextByID(projectRoot, sessionID)
	if err != nil {
		writeAPIError(stderr, "ui.session_failed", err.Error())
		return 1
	}
	if err := writeJSON(stdout, buildUISessionDetail(context)); err != nil {
		writeAPIError(stderr, "ui.session_failed", err.Error())
		return 1
	}
	return 0
}

func uiReviewJSON(sessionID string, stdout io.Writer, stderr io.Writer) int {
	projectRoot, err := currentProjectRoot()
	if err != nil {
		writeAPIError(stderr, "ui.review_failed", err.Error())
		return 1
	}

	context, err := loadUISessionContextByID(projectRoot, sessionID)
	if err != nil {
		writeAPIError(stderr, "ui.review_failed", err.Error())
		return 1
	}
	if err := writeJSON(stdout, buildUIReviewCard(context)); err != nil {
		writeAPIError(stderr, "ui.review_failed", err.Error())
		return 1
	}
	return 0
}

func currentProjectRoot() (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current project: %w", err)
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve current project absolute path: %w", err)
	}
	return projectRoot, nil
}

func loadUISessionContexts(projectRoot string) ([]uiSessionContext, error) {
	paths, err := listUISessionFiles(projectRoot)
	if err != nil {
		return nil, err
	}

	contexts := make([]uiSessionContext, 0, len(paths))
	seen := map[string]string{}
	for _, path := range paths {
		session, err := readQratumSessionFile(path, projectRoot)
		if err != nil {
			return nil, err
		}
		if previous, ok := seen[session.SessionID]; ok {
			return nil, fmt.Errorf("duplicate session_id %q in %s and %s", session.SessionID, previous, displayPath(projectRoot, path))
		}
		seen[session.SessionID] = displayPath(projectRoot, path)

		context, err := loadUISessionContext(projectRoot, path, session)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, context)
	}
	return contexts, nil
}

func loadUISessionContextByID(projectRoot string, sessionID string) (uiSessionContext, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return uiSessionContext{}, fmt.Errorf("missing session_id")
	}

	paths, err := listUISessionFiles(projectRoot)
	if err != nil {
		return uiSessionContext{}, err
	}

	var matchedPath string
	var matchedSession qratumSession
	for _, path := range paths {
		session, err := readQratumSessionFile(path, projectRoot)
		if err != nil {
			return uiSessionContext{}, err
		}
		if session.SessionID != sessionID {
			continue
		}
		if matchedPath != "" {
			return uiSessionContext{}, fmt.Errorf("duplicate session_id %q in %s and %s", sessionID, displayPath(projectRoot, matchedPath), displayPath(projectRoot, path))
		}
		matchedPath = path
		matchedSession = session
	}
	if matchedPath == "" {
		return uiSessionContext{}, fmt.Errorf("session %q not found in qratum sessions", sessionID)
	}

	return loadUISessionContext(projectRoot, matchedPath, matchedSession)
}

func listUISessionFiles(projectRoot string) ([]string, error) {
	qratumHome, err := workspace.Resolve()
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(qratumHome.Root, "sessions")
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

	paths, err := filepath.Glob(filepath.Join(sessionsDir, "*", "normalized.json"))
	if err != nil {
		return nil, fmt.Errorf("list sessions directory %s: %w", displayPath(projectRoot, sessionsDir), err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("sessions directory %s has no normalized session JSON files", displayPath(projectRoot, sessionsDir))
	}
	return paths, nil
}

func loadUISessionContext(projectRoot string, sessionPath string, session qratumSession) (uiSessionContext, error) {
	paths, err := uiArtifactPathsForSession(projectRoot, session, sessionPath)
	if err != nil {
		return uiSessionContext{}, err
	}

	redacted, err := readRequiredUIRedactedSession(projectRoot, session.SessionID, paths.Redacted)
	if err != nil {
		return uiSessionContext{}, err
	}
	evidence, err := readRequiredUIEvidence(projectRoot, session.SessionID, paths.Evidence)
	if err != nil {
		return uiSessionContext{}, err
	}
	review, err := readRequiredUIReview(projectRoot, session.SessionID, paths.Review)
	if err != nil {
		return uiSessionContext{}, err
	}

	paths = mergeUIArtifactPaths(paths, evidence.ArtifactPaths, review.ArtifactPaths)
	artifacts, err := buildUIArtifactLinks(projectRoot, session.SessionID, uiArtifactCreatedAt(session, evidence), paths)
	if err != nil {
		return uiSessionContext{}, err
	}

	return uiSessionContext{
		session:   session,
		redacted:  redacted,
		evidence:  evidence,
		review:    review,
		artifacts: artifacts,
	}, nil
}

func uiArtifactPathsForSession(projectRoot string, session qratumSession, sessionPath string) (daemonArtifactPaths, error) {
	paths := daemonArtifactPaths{
		Session: displayPath(projectRoot, sessionPath),
	}
	if session.ArtifactPaths != nil {
		paths = mergeUIArtifactPaths(paths, *session.ArtifactPaths, daemonArtifactPaths{})
		if strings.TrimSpace(paths.Session) == "" {
			paths.Session = displayPath(projectRoot, sessionPath)
		}
	}

	if strings.TrimSpace(paths.Redacted) == "" ||
		strings.TrimSpace(paths.Evidence) == "" ||
		strings.TrimSpace(paths.Review) == "" ||
		strings.TrimSpace(paths.Report) == "" ||
		strings.TrimSpace(paths.Export) == "" {
		stem, err := artifactStemForSession(session.SessionID)
		if err != nil {
			return daemonArtifactPaths{}, err
		}
		defaults := artifactPathsForStem(stem)
		if strings.TrimSpace(paths.Redacted) == "" {
			paths.Redacted = defaults.Redacted
		}
		if strings.TrimSpace(paths.Evidence) == "" {
			paths.Evidence = defaults.Evidence
		}
		if strings.TrimSpace(paths.Review) == "" {
			paths.Review = defaults.Review
		}
		if strings.TrimSpace(paths.Report) == "" {
			paths.Report = defaults.Report
		}
		if strings.TrimSpace(paths.Export) == "" {
			paths.Export = defaults.Export
		}
	}
	if err := validateArtifactPathsScoped(projectRoot, paths); err != nil {
		return daemonArtifactPaths{}, err
	}
	return paths, nil
}

func mergeUIArtifactPaths(base daemonArtifactPaths, overlays ...daemonArtifactPaths) daemonArtifactPaths {
	paths := base
	for _, overlay := range overlays {
		if strings.TrimSpace(overlay.Event) != "" {
			paths.Event = overlay.Event
		}
		if strings.TrimSpace(overlay.Session) != "" {
			paths.Session = overlay.Session
		}
		if strings.TrimSpace(overlay.Redacted) != "" {
			paths.Redacted = overlay.Redacted
		}
		if strings.TrimSpace(overlay.Evidence) != "" {
			paths.Evidence = overlay.Evidence
		}
		if strings.TrimSpace(overlay.Review) != "" {
			paths.Review = overlay.Review
		}
		if strings.TrimSpace(overlay.Report) != "" {
			paths.Report = overlay.Report
		}
		if strings.TrimSpace(overlay.Export) != "" {
			paths.Export = overlay.Export
		}
	}
	return paths
}

func readRequiredUIRedactedSession(projectRoot string, sessionID string, path string) (qratumSession, error) {
	if strings.TrimSpace(path) == "" {
		return qratumSession{}, fmt.Errorf("session %s is missing redacted artifact path", sessionID)
	}
	redactedPath, err := resolveProjectFilePath(projectRoot, path, "redacted session")
	if err != nil {
		return qratumSession{}, err
	}
	session, err := readQratumSessionFile(redactedPath, projectRoot)
	if err != nil {
		return qratumSession{}, err
	}
	if session.SessionID != sessionID {
		return qratumSession{}, fmt.Errorf("redacted session %s has session_id %q, want %q", displayPath(projectRoot, redactedPath), session.SessionID, sessionID)
	}
	if strings.TrimSpace(session.PipelineStatus) != redactionStatus {
		return qratumSession{}, fmt.Errorf("redacted session %s has pipeline_status %q, want %q", displayPath(projectRoot, redactedPath), session.PipelineStatus, redactionStatus)
	}
	return session, nil
}

func readRequiredUIEvidence(projectRoot string, sessionID string, path string) (evidenceBundle, error) {
	if strings.TrimSpace(path) == "" {
		return evidenceBundle{}, fmt.Errorf("session %s is missing evidence artifact path", sessionID)
	}
	evidencePath, err := resolveProjectFilePath(projectRoot, path, "evidence")
	if err != nil {
		return evidenceBundle{}, err
	}
	bundle, err := readEvidenceBundleFile(evidencePath, projectRoot)
	if err != nil {
		return evidenceBundle{}, err
	}
	if bundle.SessionID != sessionID {
		return evidenceBundle{}, fmt.Errorf("evidence %s has session_id %q, want %q", displayPath(projectRoot, evidencePath), bundle.SessionID, sessionID)
	}
	return bundle, nil
}

func readRequiredUIReview(projectRoot string, sessionID string, path string) (reviewCard, error) {
	if strings.TrimSpace(path) == "" {
		return reviewCard{}, fmt.Errorf("session %s is missing review artifact path", sessionID)
	}
	reviewPath, err := resolveProjectFilePath(projectRoot, path, "review")
	if err != nil {
		return reviewCard{}, err
	}
	card, err := readReviewCardFile(reviewPath, projectRoot)
	if err != nil {
		return reviewCard{}, err
	}
	if card.SessionID != sessionID {
		return reviewCard{}, fmt.Errorf("review %s has session_id %q, want %q", displayPath(projectRoot, reviewPath), card.SessionID, sessionID)
	}
	return card, nil
}

func readReviewCardFile(path string, projectRoot string) (reviewCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reviewCard{}, fmt.Errorf("read review %s: %w", displayPath(projectRoot, path), err)
	}
	var card reviewCard
	if err := json.Unmarshal(data, &card); err != nil {
		return reviewCard{}, fmt.Errorf("invalid review JSON %s: %w", displayPath(projectRoot, path), err)
	}
	if err := validateReviewCard(card, displayPath(projectRoot, path)); err != nil {
		return reviewCard{}, err
	}
	return card, nil
}

func validateReviewCard(card reviewCard, label string) error {
	if strings.TrimSpace(card.SchemaVersion) != qratumReviewSchemaVersion {
		return fmt.Errorf("review %s has unsupported schema_version %q", label, card.SchemaVersion)
	}
	if strings.TrimSpace(card.SessionID) == "" {
		return fmt.Errorf("review %s is missing session_id", label)
	}
	switch card.Verdict {
	case "clean", "needs_attention":
	default:
		return fmt.Errorf("review %s has unsupported verdict %q", label, card.Verdict)
	}
	if strings.TrimSpace(card.MainFinding) == "" {
		return fmt.Errorf("review %s is missing main_finding", label)
	}
	if card.Evidence == nil {
		return fmt.Errorf("review %s is missing evidence", label)
	}
	if strings.TrimSpace(card.SuggestedNextHabit) == "" {
		return fmt.Errorf("review %s is missing suggested_next_habit", label)
	}
	if card.Warnings == nil {
		return fmt.Errorf("review %s is missing warnings", label)
	}
	return nil
}

func uiArtifactCreatedAt(session qratumSession, evidence evidenceBundle) string {
	for _, value := range []string{
		session.SourceEventTimestamp,
		evidence.Summary.SourceEventTimestamp,
		session.EndedAt,
		evidence.Summary.EndedAt,
		session.StartedAt,
		evidence.Summary.StartedAt,
	} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return currentTimestamp()
}

func buildUIArtifactLinks(projectRoot string, sessionID string, createdAt string, paths daemonArtifactPaths) ([]uiArtifactLink, error) {
	specs := []struct {
		path      string
		linkType  string
		label     string
		mediaType string
	}{
		{path: paths.Redacted, linkType: "redacted_session", label: "Redacted session", mediaType: "application/json"},
		{path: paths.Evidence, linkType: "evidence_bundle", label: "Evidence bundle", mediaType: "application/json"},
		{path: paths.Review, linkType: "review_card", label: "Review card", mediaType: "application/json"},
		{path: paths.Report, linkType: "html_report", label: "HTML report", mediaType: "text/html"},
		{path: paths.Export, linkType: "adp_strict_export", label: "ADP strict export", mediaType: "application/jsonl"},
	}

	links := make([]uiArtifactLink, 0, len(specs))
	for _, spec := range specs {
		if strings.TrimSpace(spec.path) == "" {
			return nil, fmt.Errorf("session %s is missing %s artifact path", sessionID, spec.linkType)
		}
		absPath, err := resolveProjectFilePath(projectRoot, spec.path, spec.label)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read artifact %s: %w", displayPath(projectRoot, absPath), err)
		}
		sum := sha256.Sum256(data)
		href := slashPath(spec.path)
		if filepath.IsAbs(filepath.FromSlash(spec.path)) {
			href = displayPath(projectRoot, absPath)
		}
		links = append(links, uiArtifactLink{
			SchemaVersion: qratumUIArtifactLinkSchemaVersion,
			DataClass:     qschema.DataClassPublished,
			ArtifactID:    fmt.Sprintf("%s:%s", sessionID, spec.linkType),
			SessionID:     sessionID,
			Type:          spec.linkType,
			Label:         spec.label,
			MediaType:     spec.mediaType,
			Href:          href,
			Digest:        fmt.Sprintf("sha256:%x", sum[:]),
			CreatedAt:     createdAt,
		})
	}
	return links, nil
}

func buildUISessionListItem(context uiSessionContext) uiSessionListItem {
	return uiSessionListItem{
		SchemaVersion: qratumUISessionListItemSchemaVersion,
		DataClass:     qschema.DataClassPublished,
		SessionID:     context.session.SessionID,
		Source:        context.session.Source,
		AgentModel:    context.redacted.AgentModel,
		Repo:          buildUIRepoSummary(context.redacted),
		Review:        buildUIReviewSummary(context.review, context.evidence),
		Metrics:       buildUISessionMetrics(context.session, context.evidence, context.review),
		Artifacts:     context.artifacts,
	}
}

func buildUISessionDetail(context uiSessionContext) uiSessionDetail {
	findings := buildUIEvidenceFindings(context.evidence)
	return uiSessionDetail{
		SchemaVersion: qratumUISessionDetailSchemaVersion,
		DataClass:     qschema.DataClassPublished,
		SessionID:     context.session.SessionID,
		Source:        context.session.Source,
		AgentModel:    context.redacted.AgentModel,
		Repo:          buildUIRepoSummary(context.redacted),
		Time: uiSessionTime{
			DurationSeconds:      context.session.BusinessMetrics.DurationSeconds,
			SourceEventTimestamp: context.redacted.SourceEventTimestamp,
		},
		Summary: uiSessionSummary{
			Status:                    context.evidence.Summary.Status,
			SourceEventType:           context.redacted.SourceEventType,
			SourceEventTimestamp:      context.redacted.SourceEventTimestamp,
			SourceTranscriptSessionID: context.redacted.SourceTranscriptSessionID,
			Turns:                     len(context.session.Turns),
			ToolCalls:                 context.session.BusinessMetrics.ToolCalls,
			FilesChanged:              context.session.BusinessMetrics.FilesChanged,
			CommandsRun:               context.session.BusinessMetrics.CommandsRun,
			TestsRun:                  context.session.BusinessMetrics.TestsRun,
			MissingEvidence:           append([]string{}, context.evidence.MissingEvidence...),
		},
		Review:    buildUIReviewCardWithFindings(context, findings),
		Findings:  findings,
		Artifacts: context.artifacts,
	}
}

func buildUIReviewCard(context uiSessionContext) uiReviewCardDTO {
	return buildUIReviewCardWithFindings(context, buildUIEvidenceFindings(context.evidence))
}

func buildUIReviewCardWithFindings(context uiSessionContext, findings []uiEvidenceFinding) uiReviewCardDTO {
	return uiReviewCardDTO{
		SchemaVersion:      qratumUIReviewCardSchemaVersion,
		DataClass:          qschema.DataClassPublished,
		SessionID:          context.review.SessionID,
		Verdict:            context.review.Verdict,
		MainFinding:        context.review.MainFinding,
		Evidence:           append([]string{}, context.review.Evidence...),
		SuggestedNextHabit: context.review.SuggestedNextHabit,
		SuggestedSkill:     context.review.SuggestedSkill,
		Warnings:           append([]string{}, context.review.Warnings...),
		Findings:           findings,
		Artifacts:          context.artifacts,
	}
}

func buildUIRepoSummary(session qratumSession) uiRepoSummary {
	repo := uiRepoSummary{
		WorkspaceID: session.WorkspaceID,
		RepoID:      session.RepoID,
	}
	if session.Workspace != nil {
		repo.CWD = session.Workspace.CWD
	}
	return repo
}

func buildUIReviewSummary(card reviewCard, evidence evidenceBundle) uiReviewSummary {
	return uiReviewSummary{
		Verdict:       card.Verdict,
		MainFinding:   card.MainFinding,
		Warnings:      append([]string{}, card.Warnings...),
		FindingsCount: len(evidence.Findings),
		EvidenceCount: len(card.Evidence),
	}
}

func buildUISessionMetrics(session qratumSession, evidence evidenceBundle, card reviewCard) uiSessionMetrics {
	return uiSessionMetrics{
		DurationSeconds: session.BusinessMetrics.DurationSeconds,
		ToolCalls:       session.BusinessMetrics.ToolCalls,
		FilesChanged:    session.BusinessMetrics.FilesChanged,
		CommandsRun:     session.BusinessMetrics.CommandsRun,
		TestsRun:        session.BusinessMetrics.TestsRun,
		Findings:        len(evidence.Findings),
		Warnings:        len(card.Warnings),
	}
}

func buildUIEvidenceFindings(bundle evidenceBundle) []uiEvidenceFinding {
	findings := make([]uiEvidenceFinding, 0, len(bundle.Findings))
	for _, finding := range bundle.Findings {
		presentation := uiSupportedFindingTypes[finding.Type]
		facts := make([]uiEvidenceFact, 0, len(finding.Evidence))
		for _, fact := range finding.Evidence {
			facts = append(facts, uiEvidenceFact{
				Label:         fact.Label,
				Kind:          fact.Kind,
				Display:       formatEvidenceFact(fact),
				Timestamp:     fact.Timestamp,
				Path:          fact.Path,
				Operation:     fact.Operation,
				Command:       fact.Command,
				Success:       fact.Success,
				OutputExcerpt: fact.OutputExcerpt,
			})
		}
		findings = append(findings, uiEvidenceFinding{
			SchemaVersion:   qratumUIEvidenceFindingSchemaVersion,
			DataClass:       qschema.DataClassPublished,
			FindingID:       finding.FindingID,
			Type:            finding.Type,
			Severity:        presentation.severity,
			Confidence:      presentation.confidence,
			Title:           finding.Title,
			Summary:         finding.Summary,
			Evidence:        facts,
			MissingEvidence: append([]string{}, finding.MissingEvidence...),
		})
	}
	return findings
}

func writeJSON(stdout io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	_, err = stdout.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
