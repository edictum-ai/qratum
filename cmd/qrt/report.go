package main

import (
	"crypto/sha256"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
)

type reportContext struct {
	session     qratumSession
	redacted    qratumSession
	evidence    evidenceBundle
	review      reviewCard
	paths       daemonArtifactPaths
	sessionPath string
	artifactsAt string
}

type reportArtifact struct {
	Type      string
	Label     string
	MediaType string
	Path      string
	Href      string
	Digest    string
	CreatedAt string
	Exists    bool
	Linked    bool
}

func report(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing session path")
		return 2
	}
	if len(args) != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: report accepts exactly one session path")
		return 2
	}

	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sessionPath, err := resolveProjectFilePath(projectRoot, args[0], "session")
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid session path: %v\n", err)
		return 1
	}
	session, err := readQratumSessionFile(sessionPath, projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	context, err := loadReportContext(projectRoot, sessionPath, session)
	if err != nil {
		fmt.Fprintf(stderr, "error: build report context: %v\n", err)
		return 1
	}
	outputPath, err := resolveProjectOutputPath(projectRoot, context.paths.Report, "report")
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve report output: %v\n", err)
		return 1
	}

	data, err := buildReportDocument(projectRoot, context)
	if err != nil {
		fmt.Fprintf(stderr, "error: render report: %v\n", err)
		return 1
	}
	if err := writeFileAtomic(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(stderr, "error: write report %s: %v\n", displayPath(projectRoot, outputPath), err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", displayPath(projectRoot, outputPath))
	return 0
}

func loadReportContext(projectRoot string, sessionPath string, session qratumSession) (reportContext, error) {
	if _, err := artifactStemForSession(session.SessionID); err != nil {
		return reportContext{}, err
	}

	paths, err := uiArtifactPathsForSession(projectRoot, session, sessionPath)
	if err != nil {
		return reportContext{}, err
	}
	redacted, err := readRequiredUIRedactedSession(projectRoot, session.SessionID, paths.Redacted)
	if err != nil {
		return reportContext{}, err
	}
	if redacted.Redaction == nil {
		return reportContext{}, fmt.Errorf("redacted session %s is missing redaction summary", paths.Redacted)
	}
	evidence, err := readRequiredUIEvidence(projectRoot, session.SessionID, paths.Evidence)
	if err != nil {
		return reportContext{}, err
	}
	review, err := readRequiredUIReview(projectRoot, session.SessionID, paths.Review)
	if err != nil {
		return reportContext{}, err
	}

	paths = mergeUIArtifactPaths(paths, evidence.ArtifactPaths, review.ArtifactPaths)
	if strings.TrimSpace(paths.Session) == "" {
		paths.Session = displayPath(projectRoot, sessionPath)
	}
	if err := validateArtifactPathsScoped(projectRoot, paths); err != nil {
		return reportContext{}, err
	}

	return reportContext{
		session:     session,
		redacted:    redacted,
		evidence:    evidence,
		review:      review,
		paths:       paths,
		sessionPath: displayPath(projectRoot, sessionPath),
		artifactsAt: uiArtifactCreatedAt(session, evidence),
	}, nil
}

func buildReportDocument(projectRoot string, context reportContext) ([]byte, error) {
	if context.redacted.Redaction == nil {
		return nil, fmt.Errorf("redacted session %s is missing redaction summary", context.redacted.SessionID)
	}

	artifacts, err := buildReportArtifacts(projectRoot, context)
	if err != nil {
		return nil, err
	}
	uiLinks := reportArtifactsToUILinks(context.session.SessionID, artifacts)
	uiContext := uiSessionContext{
		session:   context.session,
		redacted:  context.redacted,
		evidence:  context.evidence,
		review:    context.review,
		artifacts: uiLinks,
	}
	detail := buildUISessionDetail(uiContext)
	reviewDTO := buildUIReviewCard(uiContext)

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString("<html lang=\"en\">\n")
	b.WriteString("<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>Qratum report ")
	writeEscaped(&b, detail.SessionID)
	b.WriteString("</title>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")
	b.WriteString("<header>\n")
	b.WriteString("<h1>Qratum report</h1>\n")
	b.WriteString("<p>Session ")
	writeEscaped(&b, detail.SessionID)
	b.WriteString(" from ")
	writeEscaped(&b, detail.Source)
	b.WriteString("</p>\n")
	b.WriteString("</header>\n")
	b.WriteString("<main>\n")

	writeSessionSummary(&b, detail, context.sessionPath)
	writeReviewCardSection(&b, reviewDTO)
	writeEvidenceFindingsSection(&b, detail.Findings)
	writeMissingEvidenceSection(&b, detail.Summary.MissingEvidence)
	writeRedactionSection(&b, *context.redacted.Redaction)
	writeArtifactsSection(&b, artifacts)
	writeProvenanceDigestsSection(&b, artifacts)

	b.WriteString("</main>\n")
	b.WriteString("<footer>\n")
	b.WriteString("<p>Static local report. No JavaScript. No external assets. Raw transcript content is not rendered.</p>\n")
	b.WriteString("</footer>\n")
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")
	return []byte(b.String()), nil
}

func buildReportArtifacts(projectRoot string, context reportContext) ([]reportArtifact, error) {
	specs := []struct {
		path      string
		linkType  string
		label     string
		mediaType string
		linked    bool
		digest    bool
		required  bool
	}{
		{path: context.paths.Session, linkType: "normalized_session", label: "Normalized session", mediaType: "application/json", linked: false, digest: true, required: true},
		{path: context.paths.Redacted, linkType: "redacted_session", label: "Redacted session", mediaType: "application/json", linked: true, digest: true, required: true},
		{path: context.paths.Evidence, linkType: "evidence_bundle", label: "Evidence bundle", mediaType: "application/json", linked: true, digest: true, required: true},
		{path: context.paths.Review, linkType: "review_card", label: "Review card", mediaType: "application/json", linked: true, digest: true, required: true},
		{path: context.paths.Report, linkType: "html_report", label: "HTML report", mediaType: "text/html", linked: true, digest: false, required: false},
		{path: context.paths.Export, linkType: "adp_strict_export", label: "ADP strict export", mediaType: "application/jsonl", linked: true, digest: true, required: false},
	}

	artifacts := make([]reportArtifact, 0, len(specs))
	for _, spec := range specs {
		path := strings.TrimSpace(spec.path)
		if path == "" {
			if spec.required {
				return nil, fmt.Errorf("session %s is missing %s artifact path", context.session.SessionID, spec.linkType)
			}
			continue
		}
		resolved, err := resolveProjectOutputPath(projectRoot, path, spec.label)
		if err != nil {
			return nil, err
		}

		artifact := reportArtifact{
			Type:      spec.linkType,
			Label:     spec.label,
			MediaType: spec.mediaType,
			Path:      displayPath(projectRoot, resolved),
			Href:      safeLocalHref(displayPath(projectRoot, resolved)),
			CreatedAt: context.artifactsAt,
			Linked:    spec.linked,
		}
		if spec.digest {
			digest, exists, err := digestReportArtifact(projectRoot, resolved, spec.label, spec.required)
			if err != nil {
				return nil, err
			}
			artifact.Digest = digest
			artifact.Exists = exists
		} else {
			artifact.Exists = true
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func digestReportArtifact(projectRoot string, path string, label string, required bool) (string, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return "", false, nil
		}
		if os.IsNotExist(err) {
			return "", false, fmt.Errorf("missing %s %s", label, displayPath(projectRoot, path))
		}
		return "", false, fmt.Errorf("inspect %s %s: %w", label, displayPath(projectRoot, path), err)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("%s %s is a directory", label, displayPath(projectRoot, path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("read %s %s: %w", label, displayPath(projectRoot, path), err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:]), true, nil
}

func reportArtifactsToUILinks(sessionID string, artifacts []reportArtifact) []uiArtifactLink {
	links := []uiArtifactLink{}
	for _, artifact := range artifacts {
		if !artifact.Linked || artifact.Digest == "" {
			continue
		}
		switch artifact.Type {
		case "redacted_session", "evidence_bundle", "review_card", "html_report", "adp_strict_export":
		default:
			continue
		}
		links = append(links, uiArtifactLink{
			SchemaVersion: qratumUIArtifactLinkSchemaVersion,
			ArtifactID:    fmt.Sprintf("%s:%s", sessionID, artifact.Type),
			SessionID:     sessionID,
			Type:          artifact.Type,
			Label:         artifact.Label,
			MediaType:     artifact.MediaType,
			Href:          artifact.Path,
			Digest:        artifact.Digest,
			CreatedAt:     artifact.CreatedAt,
		})
	}
	return links
}

func writeSessionSummary(b *strings.Builder, detail uiSessionDetail, sessionPath string) {
	b.WriteString("<section id=\"session-summary\">\n")
	b.WriteString("<h2>Session summary</h2>\n")
	b.WriteString("<table>\n<tbody>\n")
	writeTableRow(b, "Session ID", detail.SessionID)
	writeTableRow(b, "Source", detail.Source)
	writeTableRow(b, "Agent model", detail.AgentModel)
	writeTableRow(b, "Input session artifact", sessionPath)
	writeTableRow(b, "Status", detail.Summary.Status)
	writeTableRow(b, "Started at", detail.Time.StartedAt)
	writeTableRow(b, "Ended at", detail.Time.EndedAt)
	writeTableRow(b, "Duration seconds", fmt.Sprintf("%d", detail.Time.DurationSeconds))
	writeTableRow(b, "Source event ID", detail.Summary.SourceEventID)
	writeTableRow(b, "Source event type", detail.Summary.SourceEventType)
	writeTableRow(b, "Source event timestamp", detail.Summary.SourceEventTimestamp)
	writeTableRow(b, "Source transcript session ID", detail.Summary.SourceTranscriptSessionID)
	writeTableRow(b, "Workspace ID", detail.Repo.WorkspaceID)
	writeTableRow(b, "Repo ID", detail.Repo.RepoID)
	writeTableRow(b, "CWD", detail.Repo.CWD)
	writeTableRow(b, "Git remote", detail.Repo.GitRemote)
	writeTableRow(b, "Git branch", detail.Repo.GitBranch)
	writeTableRow(b, "Git head SHA", detail.Repo.GitHeadSHA)
	writeTableRow(b, "Turns", fmt.Sprintf("%d", detail.Summary.Turns))
	writeTableRow(b, "Tool calls", fmt.Sprintf("%d", detail.Summary.ToolCalls))
	writeTableRow(b, "Files changed", fmt.Sprintf("%d", detail.Summary.FilesChanged))
	writeTableRow(b, "Commands run", fmt.Sprintf("%d", detail.Summary.CommandsRun))
	writeTableRow(b, "Tests run", fmt.Sprintf("%d", detail.Summary.TestsRun))
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("</section>\n")
}

func writeReviewCardSection(b *strings.Builder, review uiReviewCardDTO) {
	b.WriteString("<section id=\"review-card\">\n")
	b.WriteString("<h2>Review card</h2>\n")
	b.WriteString("<table>\n<tbody>\n")
	writeTableRow(b, "Verdict", review.Verdict)
	writeTableRow(b, "Main finding", review.MainFinding)
	writeTableRow(b, "Suggested next habit", review.SuggestedNextHabit)
	writeTableRow(b, "Suggested skill", review.SuggestedSkill)
	b.WriteString("</tbody>\n</table>\n")

	b.WriteString("<h3>Review evidence</h3>\n")
	writeStringList(b, review.Evidence, "No review evidence recorded.")
	b.WriteString("<h3>Warnings</h3>\n")
	writeStringList(b, review.Warnings, "No warnings recorded.")
	b.WriteString("</section>\n")
}

func writeEvidenceFindingsSection(b *strings.Builder, findings []uiEvidenceFinding) {
	b.WriteString("<section id=\"evidence-findings\">\n")
	b.WriteString("<h2>Evidence findings</h2>\n")
	if len(findings) == 0 {
		b.WriteString("<p>No evidence findings recorded.</p>\n")
		b.WriteString("</section>\n")
		return
	}
	for _, finding := range findings {
		b.WriteString("<article>\n")
		b.WriteString("<h3>")
		writeEscaped(b, finding.Title)
		b.WriteString("</h3>\n")
		b.WriteString("<table>\n<tbody>\n")
		writeTableRow(b, "Finding ID", finding.FindingID)
		writeTableRow(b, "Type", finding.Type)
		writeTableRow(b, "Severity", finding.Severity)
		writeTableRow(b, "Confidence", finding.Confidence)
		writeTableRow(b, "Summary", finding.Summary)
		b.WriteString("</tbody>\n</table>\n")
		b.WriteString("<h4>Evidence</h4>\n")
		if len(finding.Evidence) == 0 {
			b.WriteString("<p>No evidence facts recorded.</p>\n")
		} else {
			b.WriteString("<ol>\n")
			for _, fact := range finding.Evidence {
				b.WriteString("<li>\n")
				b.WriteString("<p>")
				writeEscaped(b, fact.Display)
				b.WriteString("</p>\n")
				b.WriteString("<table>\n<tbody>\n")
				writeTableRow(b, "Label", fact.Label)
				writeTableRow(b, "Kind", fact.Kind)
				writeTableRow(b, "Timestamp", fact.Timestamp)
				writeTableRow(b, "Path", fact.Path)
				writeTableRow(b, "Operation", fact.Operation)
				writeTableRow(b, "Command", fact.Command)
				if fact.Success != nil {
					writeTableRow(b, "Success", fmt.Sprintf("%t", *fact.Success))
				}
				writeTableRow(b, "Output excerpt", fact.OutputExcerpt)
				b.WriteString("</tbody>\n</table>\n")
				b.WriteString("</li>\n")
			}
			b.WriteString("</ol>\n")
		}
		b.WriteString("<h4>Missing evidence for this finding</h4>\n")
		writeStringList(b, finding.MissingEvidence, "No missing evidence recorded for this finding.")
		b.WriteString("</article>\n")
	}
	b.WriteString("</section>\n")
}

func writeMissingEvidenceSection(b *strings.Builder, missing []string) {
	b.WriteString("<section id=\"missing-evidence\">\n")
	b.WriteString("<h2>Missing evidence</h2>\n")
	writeStringList(b, missing, "No missing evidence recorded.")
	b.WriteString("</section>\n")
}

func writeRedactionSection(b *strings.Builder, summary qratumRedactionSummary) {
	b.WriteString("<section id=\"redaction-summary\">\n")
	b.WriteString("<h2>Redaction summary</h2>\n")
	b.WriteString("<table>\n<tbody>\n")
	writeTableRow(b, "Status", summary.Status)
	writeTableRow(b, "Secret placeholders", fmt.Sprintf("%d", summary.SecretPlaceholders))
	writeTableRow(b, "Path placeholders", fmt.Sprintf("%d", summary.PathPlaceholders))
	b.WriteString("</tbody>\n</table>\n")

	b.WriteString("<h3>Redaction findings</h3>\n")
	if len(summary.Findings) == 0 {
		b.WriteString("<p>No redaction findings recorded.</p>\n")
		b.WriteString("</section>\n")
		return
	}
	b.WriteString("<table>\n<thead><tr><th>ID</th><th>Type</th><th>Field</th><th>Replacements</th><th>Placeholders</th></tr></thead>\n<tbody>\n")
	for _, finding := range summary.Findings {
		b.WriteString("<tr><td>")
		writeEscaped(b, finding.ID)
		b.WriteString("</td><td>")
		writeEscaped(b, finding.Type)
		b.WriteString("</td><td>")
		writeEscaped(b, finding.Field)
		b.WriteString("</td><td>")
		writeEscaped(b, fmt.Sprintf("%d", finding.ReplacementCount))
		b.WriteString("</td><td>")
		writeEscaped(b, strings.Join(finding.Placeholders, ", "))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("</section>\n")
}

func writeArtifactsSection(b *strings.Builder, artifacts []reportArtifact) {
	b.WriteString("<section id=\"artifacts\">\n")
	b.WriteString("<h2>Artifacts</h2>\n")
	b.WriteString("<table>\n<thead><tr><th>Artifact</th><th>Media type</th><th>Path</th><th>Status</th></tr></thead>\n<tbody>\n")
	for _, artifact := range artifacts {
		if artifact.Type == "normalized_session" {
			continue
		}
		b.WriteString("<tr><td>")
		writeEscaped(b, artifact.Label)
		b.WriteString("</td><td>")
		writeEscaped(b, artifact.MediaType)
		b.WriteString("</td><td>")
		if artifact.Linked && artifact.Href != "" {
			b.WriteString("<a href=\"")
			writeEscaped(b, artifact.Href)
			b.WriteString("\">")
			writeEscaped(b, artifact.Path)
			b.WriteString("</a>")
		} else {
			writeEscaped(b, artifact.Path)
		}
		b.WriteString("</td><td>")
		if artifact.Exists {
			b.WriteString("available")
		} else {
			b.WriteString("not generated")
		}
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("</section>\n")
}

func writeProvenanceDigestsSection(b *strings.Builder, artifacts []reportArtifact) {
	b.WriteString("<section id=\"provenance-digests\">\n")
	b.WriteString("<h2>Provenance digests</h2>\n")
	b.WriteString("<table>\n<thead><tr><th>Artifact</th><th>Path</th><th>Digest</th></tr></thead>\n<tbody>\n")
	for _, artifact := range artifacts {
		if artifact.Digest == "" {
			continue
		}
		b.WriteString("<tr><td>")
		writeEscaped(b, artifact.Label)
		b.WriteString("</td><td>")
		writeEscaped(b, artifact.Path)
		b.WriteString("</td><td><code>")
		writeEscaped(b, artifact.Digest)
		b.WriteString("</code></td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("</section>\n")
}

func writeStringList(b *strings.Builder, values []string, emptyMessage string) {
	if len(values) == 0 {
		b.WriteString("<p>")
		writeEscaped(b, emptyMessage)
		b.WriteString("</p>\n")
		return
	}
	b.WriteString("<ul>\n")
	for _, value := range values {
		b.WriteString("<li>")
		writeEscaped(b, value)
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n")
}

func writeTableRow(b *strings.Builder, label string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "not recorded"
	}
	b.WriteString("<tr><th scope=\"row\">")
	writeEscaped(b, label)
	b.WriteString("</th><td>")
	writeEscaped(b, value)
	b.WriteString("</td></tr>\n")
}

func writeEscaped(b *strings.Builder, value string) {
	b.WriteString(html.EscapeString(value))
}

func safeLocalHref(path string) string {
	path = strings.TrimSpace(slashPath(path))
	if path == "" {
		return ""
	}
	path = strings.TrimPrefix(path, "./")
	return "./" + path
}
