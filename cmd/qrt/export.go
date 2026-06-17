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
)

const (
	adpStrictProfile       = "adp-strict"
	adpStrictSchemaVersion = "1.1.0"
)

type adpStrictTrajectory struct {
	SchemaVersion string           `json:"schema_version"`
	ID            string           `json:"id"`
	Content       []any            `json:"content"`
	Details       adpStrictDetails `json:"details"`
}

type adpStrictDetails struct {
	Source string `json:"source"`
}

type adpTextObservation struct {
	Class   string `json:"class_"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

type adpAPIAction struct {
	Class    string         `json:"class_"`
	Function string         `json:"function"`
	Kwargs   map[string]any `json:"kwargs"`
}

type adpCodeAction struct {
	Class    string `json:"class_"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

type adpContentRecord struct {
	at       time.Time
	hasTime  bool
	sequence int
	content  any
}

type commandMatchKey struct {
	timestamp string
	command   string
}

func exportCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing session path")
		return 2
	}
	if len(args) != 3 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: export accepts exactly one session path and --profile adp-strict")
		return 2
	}
	if args[1] != "--profile" {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: export requires --profile adp-strict")
		return 2
	}
	if args[2] != adpStrictProfile {
		fmt.Fprintf(stderr, "error: unsupported export profile %q\n", args[2])
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

	paths, err := artifactPathsForSession(projectRoot, session, sessionPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve export artifact paths: %v\n", err)
		return 1
	}
	outputPath, err := resolveProjectOutputPath(projectRoot, paths.Export, "export")
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve export output: %v\n", err)
		return 1
	}

	data, err := buildADPStrictJSONL(session)
	if err != nil {
		fmt.Fprintf(stderr, "error: build ADP strict export for %s: %v\n", displayPath(projectRoot, sessionPath), err)
		return 1
	}
	if err := writeFileAtomic(outputPath, data, 0o600); err != nil {
		fmt.Fprintf(stderr, "error: write export %s: %v\n", displayPath(projectRoot, outputPath), err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", displayPath(projectRoot, outputPath))
	return 0
}

func buildADPStrictJSONL(session qratumSession) ([]byte, error) {
	redacted, err := redactQratumSession(session)
	if err != nil {
		return nil, err
	}
	trajectory, err := buildADPStrictTrajectory(redacted)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(trajectory)
	if err != nil {
		return nil, fmt.Errorf("encode ADP strict JSONL: %w", err)
	}
	return append(data, '\n'), nil
}

func buildADPStrictTrajectory(session qratumSession) (adpStrictTrajectory, error) {
	if err := validateQratumSession(session, session.SessionID); err != nil {
		return adpStrictTrajectory{}, err
	}

	records, err := buildADPContentRecords(session)
	if err != nil {
		return adpStrictTrajectory{}, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		left, right := records[i], records[j]
		if left.hasTime && right.hasTime {
			if !left.at.Equal(right.at) {
				return left.at.Before(right.at)
			}
		} else if left.hasTime != right.hasTime {
			return left.hasTime
		}
		return left.sequence < right.sequence
	})

	content := make([]any, 0, len(records))
	for _, record := range records {
		content = append(content, record.content)
	}

	return adpStrictTrajectory{
		SchemaVersion: adpStrictSchemaVersion,
		ID:            session.SessionID,
		Content:       content,
		Details: adpStrictDetails{
			Source: session.Source,
		},
	}, nil
}

func buildADPContentRecords(session qratumSession) ([]adpContentRecord, error) {
	records := []adpContentRecord{}
	sequence := 0
	addRecord := func(timestamp string, content any) error {
		record, err := newADPContentRecord(timestamp, sequence, content)
		if err != nil {
			return err
		}
		records = append(records, record)
		sequence++
		return nil
	}

	for i, turn := range session.Turns {
		source, err := adpObservationSource(turn.Role)
		if err != nil {
			return nil, fmt.Errorf("turns[%d]: %w", i, err)
		}
		if err := addRecord(turn.Timestamp, adpTextObservation{
			Class:   "TextObservation",
			Source:  source,
			Content: turn.Content,
		}); err != nil {
			return nil, fmt.Errorf("turns[%d].timestamp: %w", i, err)
		}
	}

	matchedCommands := map[commandMatchKey]int{}
	for i, toolCall := range session.ToolCalls {
		if strings.EqualFold(toolCall.Name, "Bash") {
			command, err := stringFromAny(toolCall.Input, "command")
			if err != nil {
				return nil, fmt.Errorf("tool_calls[%d].input.command: %w", i, err)
			}
			if command == "" {
				return nil, fmt.Errorf("tool_calls[%d].input.command is required for Bash", i)
			}
			matchedCommands[commandMatchKey{timestamp: toolCall.Timestamp, command: command}]++
			if err := addRecord(toolCall.Timestamp, adpCodeAction{
				Class:    "CodeAction",
				Language: "shell",
				Code:     command,
			}); err != nil {
				return nil, fmt.Errorf("tool_calls[%d].timestamp: %w", i, err)
			}
		} else {
			kwargs, err := adpAPIKwargs(toolCall)
			if err != nil {
				return nil, fmt.Errorf("tool_calls[%d].input: %w", i, err)
			}
			if err := addRecord(toolCall.Timestamp, adpAPIAction{
				Class:    "ApiAction",
				Function: toolCall.Name,
				Kwargs:   kwargs,
			}); err != nil {
				return nil, fmt.Errorf("tool_calls[%d].timestamp: %w", i, err)
			}
		}

		if toolCall.Result != "" || toolCall.ResultTime != "" || toolCall.Success != nil {
			if err := addRecord(toolCall.ResultTime, adpTextObservation{
				Class:   "TextObservation",
				Source:  "environment",
				Content: toolCall.Result,
			}); err != nil {
				return nil, fmt.Errorf("tool_calls[%d].result_timestamp: %w", i, err)
			}
		}
	}

	for i, command := range session.Commands {
		key := commandMatchKey{timestamp: command.Timestamp, command: command.Command}
		if matchedCommands[key] > 0 {
			matchedCommands[key]--
			continue
		}
		if strings.TrimSpace(command.Command) == "" {
			return nil, fmt.Errorf("commands[%d].command is required", i)
		}
		if err := addRecord(command.Timestamp, adpCodeAction{
			Class:    "CodeAction",
			Language: "shell",
			Code:     command.Command,
		}); err != nil {
			return nil, fmt.Errorf("commands[%d].timestamp: %w", i, err)
		}
		if command.Output != "" || command.Success != nil {
			if err := addRecord(command.Timestamp, adpTextObservation{
				Class:   "TextObservation",
				Source:  "environment",
				Content: command.Output,
			}); err != nil {
				return nil, fmt.Errorf("commands[%d].timestamp: %w", i, err)
			}
		}
	}

	return records, nil
}

func newADPContentRecord(timestamp string, sequence int, content any) (adpContentRecord, error) {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return adpContentRecord{sequence: sequence, content: content}, nil
	}
	at, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return adpContentRecord{}, fmt.Errorf("must be RFC3339: %w", err)
	}
	return adpContentRecord{
		at:       at,
		hasTime:  true,
		sequence: sequence,
		content:  content,
	}, nil
}

func adpObservationSource(role string) (string, error) {
	switch role {
	case "user":
		return "user", nil
	case "assistant":
		return "agent", nil
	default:
		return "", fmt.Errorf("unsupported role %q", role)
	}
}

func adpAPIKwargs(toolCall qratumToolCall) (map[string]any, error) {
	name := strings.ToLower(strings.TrimSpace(toolCall.Name))
	switch name {
	case "read", "write", "edit", "multiedit", "notebookedit":
		return adpFileToolKwargs(toolCall.Input)
	default:
		return sanitizeADPMap(toolCall.Input), nil
	}
}

func adpFileToolKwargs(input map[string]any) (map[string]any, error) {
	kwargs := map[string]any{}
	if _, ok := input["file_path"]; ok {
		value, err := stringFromAny(input, "file_path")
		if err != nil {
			return nil, err
		}
		if value == "" {
			return nil, fmt.Errorf("file_path is required")
		}
		kwargs["file_path"] = value
		return kwargs, nil
	}
	if _, ok := input["path"]; ok {
		value, err := stringFromAny(input, "path")
		if err != nil {
			return nil, err
		}
		if value == "" {
			return nil, fmt.Errorf("path is required")
		}
		kwargs["path"] = value
		return kwargs, nil
	}
	return nil, fmt.Errorf("file_path is required")
}

func sanitizeADPMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		if !isAllowedADPKwargKey(key) {
			continue
		}
		output[key] = sanitizeADPValue(value)
	}
	return output
}

func sanitizeADPValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return sanitizeADPMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sanitizeADPValue(item)
		}
		return out
	default:
		return v
	}
}

func isAllowedADPKwargKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "command", "description", "file_path", "path", "pattern", "query", "url", "limit", "offset":
		return true
	default:
		return false
	}
}
