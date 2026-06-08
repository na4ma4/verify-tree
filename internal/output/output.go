package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/na4ma4/verify-tree/internal/verifier"
)

type Format int

const (
	FormatText Format = iota
	FormatColourText
	FormatJSON
	FormatYAML
)

func ParseFormat(s string, colour bool) (Format, error) {
	switch s {
	case "text":
		if colour {
			return FormatColourText, nil
		}

		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return FormatText, fmt.Errorf("unknown format %q, expected text, json, or yaml", s)
	}
}

func Text(w io.Writer, result *verifier.Result) {
	totalEntries := len(result.Entries)
	totalViolations := 0

	for _, entry := range result.Entries {
		if entry.Skipped {
			fmt.Fprintf(w, "  ⚠ %s (skipped)\n", entry.Path)
			continue
		}

		if len(entry.Violations) == 0 {
			fmt.Fprintf(w, "  ✓ %s\n", entry.Path)
		} else {
			fmt.Fprintf(w, "  ✗ %s\n", entry.Path)
			for _, v := range entry.Violations {
				totalViolations++
				fmt.Fprintf(w, "      - %s: expected %s, got %s\n", v.Check, v.Expected, v.Actual)
			}
		}
	}

	fmt.Fprintln(w)

	if result.Passed {
		fmt.Fprintf(w, "PASS: %d entries verified, %d passed\n", totalEntries, totalEntries-totalViolations)
	} else {
		fmt.Fprintf(w, "FAIL: %d entries, %d violations\n", totalEntries, totalViolations)
	}
}

func JSON(w io.Writer, result *verifier.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func YAML(w io.Writer, result *verifier.Result) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(yamlIndent)
	return enc.Encode(result)
}

const (
	yamlIndent = 2

	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
)

func ColoredText(w io.Writer, result *verifier.Result) {
	totalEntries := len(result.Entries)
	totalViolations := 0

	for _, entry := range result.Entries {
		if entry.Skipped {
			fmt.Fprintf(
				w,
				"  %s⚠%s %s%s%s (%sskipped%s)\n",
				ansiYellow,
				ansiReset,
				ansiCyan,
				entry.Path,
				ansiReset,
				ansiYellow,
				ansiReset,
			)
			continue
		}

		if len(entry.Violations) == 0 {
			fmt.Fprintf(w, "  %s✓%s %s%s%s\n", ansiGreen, ansiReset, ansiCyan, entry.Path, ansiReset)
		} else {
			fmt.Fprintf(w, "  %s✗%s %s%s%s\n", ansiRed, ansiReset, ansiBold, entry.Path, ansiReset)
			for _, v := range entry.Violations {
				totalViolations++
				fmt.Fprintf(w, "    %s- %s%s: expected %s%s%s, got %s%s%s\n",
					ansiRed, v.Check, ansiReset,
					ansiGreen, v.Expected, ansiReset,
					ansiRed, v.Actual, ansiReset,
				)
			}
		}
	}

	fmt.Fprintln(w)

	if result.Passed {
		fmt.Fprintf(w, "%sPASS:%s %d entries verified\n", ansiGreen, ansiReset, totalEntries)
	} else {
		fmt.Fprintf(w, "%sFAIL:%s %d entries, %d violations\n", ansiRed, ansiReset, totalEntries, totalViolations)
	}
}

func Write(w io.Writer, result *verifier.Result, format Format) {
	switch format {
	default:
		fallthrough
	case FormatText:
		Text(w, result)
	case FormatJSON:
		if err := JSON(w, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON output: %v\n", err)
			os.Exit(1)
		}
	case FormatYAML:
		if err := YAML(w, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing YAML output: %v\n", err)
			os.Exit(1)
		}
	case FormatColourText:
		ColoredText(w, result)
	}
}

func WriteSummary(w io.Writer, result *verifier.Result) {
	var passed, failed, skipped int
	for _, e := range result.Entries {
		switch {
		case e.Skipped:
			skipped++
		case e.Passed:
			passed++
		default:
			failed++
		}
	}

	parts := []string{fmt.Sprintf("%d passed", passed)}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}

	fmt.Fprintf(w, "Summary: %s\n", strings.Join(parts, ", "))
}
