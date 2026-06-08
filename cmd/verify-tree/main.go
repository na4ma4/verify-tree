package main

import (
	"os"

	"github.com/na4ma4/go-contextual"
	"github.com/spf13/cobra"
)

func mainCommand(cmd *cobra.Command, _ []string) error {
	ctx := contextual.NewCancellable(cmd.Context(),
		contextual.WithSignalCancelOption(),
	)
	defer ctx.Cancel()

	vars, err := collectVariables()
	if err != nil {
		return err
	}

	return runVerify(ctx, os.Stdout, vars)
}
