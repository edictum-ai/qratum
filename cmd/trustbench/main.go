// Command trustbench runs Qratum's local trust scorecard for CI.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edictum-ai/qratum/internal/trust"
)

func main() {
	var qrtPath string
	var jsonOut string
	flag.StringVar(&qrtPath, "qrt", "", "path to qrt binary")
	flag.StringVar(&jsonOut, "json-out", "", "path to write scorecard JSON")
	flag.Parse()

	scorecard, ci, err := trust.Evaluate(trust.Options{QRTPath: qrtPath})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	data, err := trust.Marshal(scorecard)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := trust.NoLeakCheck(data); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut != "" {
		if err := os.MkdirAll(filepath.Dir(jsonOut), 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(jsonOut, data, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	printHuman(scorecard, ci)
	if !ci.Pass {
		os.Exit(1)
	}
}

func printHuman(scorecard trust.Scorecard, ci trust.CIStatus) {
	fmt.Printf("qratum trust\n")
	fmt.Printf("headline: %s\n", scorecard.Headline)
	fmt.Printf("gap_count: %d\n", scorecard.GapCount)
	fmt.Printf("ci_pass: %t\n", ci.Pass)
	for _, reason := range ci.Reasons {
		fmt.Printf("- %s\n", reason)
	}
	fmt.Println("dimensions:")
	for _, dimension := range scorecard.Dimensions {
		fmt.Printf("- %s: %s - %s\n", dimension.ID, dimension.State, dimension.Summary)
	}
	fmt.Println("honest_residual:")
	for _, residual := range scorecard.HonestResidual {
		fmt.Printf("- %s\n", residual)
	}
}
