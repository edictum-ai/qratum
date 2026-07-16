package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/acartag7/qratum/internal/trust"
)

func trustCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("trust", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: trust usage: trust [--json]")
		return 2
	}
	if fs.NArg() != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: trust does not accept arguments")
		return 2
	}
	qrtPath := os.Getenv("QRATUM_QRT_BIN")
	if qrtPath == "" {
		if exe, err := os.Executable(); err == nil {
			qrtPath = exe
		}
	}
	scorecard, ci, err := trust.Evaluate(trust.Options{QRTPath: qrtPath})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	data, err := trust.Marshal(scorecard)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := trust.NoLeakCheck(data); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if *jsonOut {
		fmt.Fprint(stdout, string(data))
	} else {
		printTrustHuman(stdout, scorecard, ci)
	}
	if !ci.Pass {
		return 1
	}
	return 0
}

func printTrustHuman(stdout io.Writer, scorecard trust.Scorecard, ci trust.CIStatus) {
	fmt.Fprintln(stdout, "qratum trust")
	fmt.Fprintf(stdout, "headline: %s\n", scorecard.Headline)
	fmt.Fprintf(stdout, "gap_count: %d\n", scorecard.GapCount)
	fmt.Fprintf(stdout, "ci_pass: %s\n", yesNo(ci.Pass))
	for _, reason := range ci.Reasons {
		fmt.Fprintf(stdout, "- %s\n", reason)
	}
	fmt.Fprintln(stdout, "dimensions:")
	for _, dimension := range scorecard.Dimensions {
		fmt.Fprintf(stdout, "- %s: %s - %s\n", dimension.ID, dimension.State, dimension.Summary)
	}
	fmt.Fprintln(stdout, "honest_residual:")
	for _, residual := range scorecard.HonestResidual {
		fmt.Fprintf(stdout, "- %s\n", residual)
	}
}
