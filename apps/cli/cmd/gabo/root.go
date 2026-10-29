package main

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/setupprompt"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/check"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/initcmd"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/prompt"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/version"
)

// deps holds everything the composition root wires into the slices.
type deps struct {
	version string
	runner  kernel.Runner
	update  updateDeps
}

// updateDeps wire the self-update of `gabo init` (TRD 9.4). Without a binary
// adapter init never self-updates and only prints the upgrade command.
type updateDeps struct {
	baseURL     string
	publicKey   string
	interactive bool
	in          io.Reader
	out         io.Writer
	args        []string
	binary      selfupdate.Binary
	releases    selfupdate.Releases
	getenv      func(string) string
}

func newRootCommand(d deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "gabo",
		Short:         "Compile the gabo agent team into a project",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	root.AddCommand(initcmd.NewCommand(initFactory(d.runner, d.version, d.update)))
	root.AddCommand(check.NewCommand(checkFactory(d.runner, d.version, os.Getenv, os.Getwd)))
	root.AddCommand(prompt.NewCommand(setupprompt.Text))
	return root
}
