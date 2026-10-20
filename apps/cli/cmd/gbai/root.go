package main

import (
	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/version"
)

// deps holds everything the composition root wires into the slices.
type deps struct {
	version string
}

func newRootCommand(d deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "gbai",
		Short:         "Provision and operate autonomous cloud coding agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	return root
}
