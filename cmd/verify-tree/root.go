package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/na4ma4/verify-tree/internal/config"
	"github.com/na4ma4/verify-tree/internal/output"
	"github.com/na4ma4/verify-tree/internal/verifier"

	"github.com/spf13/cobra"
)

var (
	specFile  string
	formatOpt string
	varOpts   []string
	varFile   string
	noColour  bool
)

var rootCmd = &cobra.Command{
	Use:   "verify-tree [flags]",
	Short: "Verify file tree against a specification",
	Long: `verify-tree reads a YAML specification of expected file paths,
types, ownership, and permissions, then checks the filesystem
against that specification.

Templates in the spec ({{ variable }}) are resolved via CLI flags
(--var key=value), environment variables (VT_VAR_KEY=value),
or a variables file (KEY=value per line).`,
	RunE: mainCommand,
}

func init() {
	rootCmd.Flags().StringVarP(&specFile, "spec", "s", "spec.yaml", "Path to specification YAML file")
	rootCmd.Flags().StringVarP(&formatOpt, "format", "f", "text", "Output format: text, json, or yaml")
	rootCmd.Flags().StringArrayVarP(&varOpts, "var", "v", nil, "Set template variable (key=value, repeatable)")
	rootCmd.Flags().StringVar(&varFile, "var-file", "", "Path to variables file (key=value per line)")
	rootCmd.Flags().BoolVar(&noColour, "no-colour", false, "Disable colored output")
	rootCmd.Flags().BoolVar(&noColour, "no-color", false, "Disable colored output")
}

func Help() error {
	return rootCmd.Help()
}

func main() {
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

//nolint:mnd // k=v is 2
func collectVariables() (config.Variables, error) {
	vars := make(config.Variables)

	vars.LoadFromEnv()

	if varFile != "" {
		if err := vars.LoadFromFile(varFile); err != nil {
			return nil, fmt.Errorf("loading variables file: %w", err)
		}
	}

	for _, opt := range varOpts {
		parts := splitVar(opt)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --var format: %q (expected key=value)", opt)
		}
		vars.Add(parts[0], parts[1])
	}

	return vars, nil
}

func splitVar(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func runVerify(_ context.Context, w io.Writer, vars config.Variables) error {
	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	useColor := !noColour && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

	format, err := output.ParseFormat(formatOpt, useColor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	output.Write(w, result, format)

	os.Exit(result.ExitCode())
	return nil
}
