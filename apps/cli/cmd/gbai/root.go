package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/check"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/version"
)

// deps holds everything the composition root wires into the slices.
type deps struct {
	version string
	runner  kernel.Runner
}

func newRootCommand(d deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "gbai",
		Short:         "Compile the gbai agent team into a project",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	root.AddCommand(initcmd.NewCommand(initFactory(d.runner, d.version)))
	root.AddCommand(check.NewCommand(checkFactory(d.runner, d.version, os.Getenv, os.Getwd)))
	return root
}
