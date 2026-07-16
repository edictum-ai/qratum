package main

import (
	"fmt"
	"io"

	"github.com/acartag7/qratum/internal/capture"
)

const (
	captureEventSchemaVersion        = capture.EventSchemaVersion
	claudeCodeSource                 = capture.ClaudeCodeSource
	deprecatedUnixZeroHookTimestamp  = capture.DeprecatedUnixZeroHookTimestamp
	hookTimestampSourceHookPayload   = capture.HookTimestampSourceHookPayload
	hookTimestampSourceCaptureTime   = capture.HookTimestampSourceCaptureTime
	hookTimestampSourceTranscriptEnd = capture.HookTimestampSourceTranscriptEnd
	maxHookPayloadBytes              = capture.MaxHookPayloadBytes
)

type captureEvent = capture.Event
type captureSessionRef = capture.SessionRef
type captureWorkspaceRef = capture.WorkspaceRef

func hook(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	_ = stdout

	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing hook adapter")
		return 2
	}

	switch args[0] {
	case claudeCodeSource:
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: hook claude-code does not accept arguments")
			printUsage(stderr)
			return 2
		}
		event, err := capture.SpoolClaudeCodeHook(stdin, currentTimestamp())
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if warning := hookWarning(event); warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", warning)
		}
		return 0
	case "install":
		return hookInstall(args[1:], stdin, stdout, stderr)
	case "status":
		return hookStatus(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unsupported hook adapter %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func hookWarning(event captureEvent) string {
	return capture.HookWarning(event)
}
