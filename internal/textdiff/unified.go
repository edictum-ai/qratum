// Package textdiff renders small text diffs without external dependencies.
package textdiff

import "strings"

// Unified renders a whole-file line diff with +/- prefixes.
func Unified(beforeName string, before []byte, afterName string, after []byte) string {
	beforeLines := splitLines(string(before))
	afterLines := splitLines(string(after))
	ops := diffLines(beforeLines, afterLines)

	var b strings.Builder
	b.WriteString("--- ")
	b.WriteString(beforeName)
	b.WriteByte('\n')
	b.WriteString("+++ ")
	b.WriteString(afterName)
	b.WriteByte('\n')
	for _, op := range ops {
		b.WriteByte(op.prefix)
		b.WriteString(op.line)
	}
	return b.String()
}

type diffLine struct {
	prefix byte
	line   string
}

func diffLines(before []string, after []string) []diffLine {
	dp := make([][]int, len(before)+1)
	for i := range dp {
		dp[i] = make([]int, len(after)+1)
	}
	for i := len(before) - 1; i >= 0; i-- {
		for j := len(after) - 1; j >= 0; j-- {
			if before[i] == after[j] {
				dp[i][j] = dp[i+1][j+1] + 1
				continue
			}
			if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
				continue
			}
			dp[i][j] = dp[i][j+1]
		}
	}

	lines := make([]diffLine, 0, len(before)+len(after))
	i, j := 0, 0
	for i < len(before) && j < len(after) {
		if before[i] == after[j] {
			lines = append(lines, diffLine{prefix: ' ', line: before[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			lines = append(lines, diffLine{prefix: '-', line: before[i]})
			i++
			continue
		}
		lines = append(lines, diffLine{prefix: '+', line: after[j]})
		j++
	}
	for ; i < len(before); i++ {
		lines = append(lines, diffLine{prefix: '-', line: before[i]})
	}
	for ; j < len(after); j++ {
		lines = append(lines, diffLine{prefix: '+', line: after[j]})
	}
	return lines
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	lines := strings.SplitAfter(value, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{value}
	}
	return lines
}
