package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const dogfoodImportStatus = "dogfood_imported"

type dogfoodLatestReview struct {
	card       reviewCard
	reviewPath string
	reportPath string
	exportPath string
	modTime    time.Time
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
	fmt.Fprintf(stdout, "suggested_next_habit: %s\n", latest.card.SuggestedNextHabit)
	fmt.Fprintf(stdout, "html_report_path: %s\n", latest.reportPath)
	fmt.Fprintf(stdout, "adp_export_path: %s\n", latest.exportPath)
	return 0
}

func findLatestDogfoodReview(projectRoot string) (dogfoodLatestReview, error) {
	reviewsDir := filepath.Join(projectRoot, ".qratum", "reviews")
	info, err := os.Stat(reviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return dogfoodLatestReview{}, fmt.Errorf("reviews directory %s does not exist", displayPath(projectRoot, reviewsDir))
		}
		return dogfoodLatestReview{}, fmt.Errorf("inspect reviews directory %s: %w", displayPath(projectRoot, reviewsDir), err)
	}
	if !info.IsDir() {
		return dogfoodLatestReview{}, fmt.Errorf("reviews path %s is not a directory", displayPath(projectRoot, reviewsDir))
	}

	paths, err := filepath.Glob(filepath.Join(reviewsDir, "*.review.json"))
	if err != nil {
		return dogfoodLatestReview{}, fmt.Errorf("list reviews directory %s: %w", displayPath(projectRoot, reviewsDir), err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return dogfoodLatestReview{}, fmt.Errorf("reviews directory %s has no review JSON files", displayPath(projectRoot, reviewsDir))
	}

	var latest dogfoodLatestReview
	for _, path := range paths {
		card, err := readReviewCardFile(path, projectRoot)
		if err != nil {
			return dogfoodLatestReview{}, err
		}
		reportPath, exportPath, err := dogfoodReviewArtifactPaths(projectRoot, card)
		if err != nil {
			return dogfoodLatestReview{}, err
		}
		stat, err := os.Stat(path)
		if err != nil {
			return dogfoodLatestReview{}, fmt.Errorf("inspect review %s: %w", displayPath(projectRoot, path), err)
		}
		candidate := dogfoodLatestReview{
			card:       card,
			reviewPath: displayPath(projectRoot, path),
			reportPath: reportPath,
			exportPath: exportPath,
			modTime:    stat.ModTime(),
		}
		if latest.card.SessionID == "" ||
			candidate.modTime.After(latest.modTime) ||
			(candidate.modTime.Equal(latest.modTime) && candidate.reviewPath > latest.reviewPath) {
			latest = candidate
		}
	}
	return latest, nil
}

func dogfoodReviewArtifactPaths(projectRoot string, card reviewCard) (string, string, error) {
	paths := card.ArtifactPaths
	if strings.TrimSpace(paths.Report) == "" || strings.TrimSpace(paths.Export) == "" {
		stem, err := artifactStemForSession(card.SessionID)
		if err != nil {
			return "", "", err
		}
		defaults := artifactPathsForStem(stem)
		if strings.TrimSpace(paths.Report) == "" {
			paths.Report = defaults.Report
		}
		if strings.TrimSpace(paths.Export) == "" {
			paths.Export = defaults.Export
		}
	}
	if _, err := resolveProjectFilePath(projectRoot, paths.Report, "HTML report"); err != nil {
		return "", "", err
	}
	if _, err := resolveProjectFilePath(projectRoot, paths.Export, "ADP export"); err != nil {
		return "", "", err
	}
	return paths.Report, paths.Export, nil
}
