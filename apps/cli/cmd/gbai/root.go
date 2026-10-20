package main

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/railwayssh"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	provisionadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets"
	secretsadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/version"
)

// deps holds everything the composition root wires into the slices.
type deps struct {
	version string
	runner  kernel.Runner
	stdin   *os.File
	stderr  io.Writer
}

func newRootCommand(d deps) *cobra.Command {
	var configPath string
	root := &cobra.Command{
		Use:           "gbai",
		Short:         "Provision and operate autonomous cloud coding agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&configPath, "config", "gbai.yaml", "path to the gbai config file")

	loadConfig := func() (config.Config, error) { return config.Load(configPath) }

	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	root.AddCommand(secrets.NewCommand(func(context.Context) (*secrets.UseCase, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		reviewer := cfg.Users.Reviewer
		return secrets.NewUseCase(
			newRemoteShell(d.runner, cfg),
			secretsadapters.NewStdinReader(d.stdin, d.stderr),
			secrets.Account{Name: reviewer.Name, Home: reviewer.Home},
		), nil
	}))
	root.AddCommand(provision.NewCommand(func(context.Context) (*provision.UseCase, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		doc, err := cfg.Marshal()
		if err != nil {
			return nil, err
		}
		return provision.NewUseCase(newRemoteShell(d.runner, cfg), provisionadapters.FileBinary{}, provision.Settings{
			Orca:       provisionAccount(cfg.Users.Orca),
			Reviewer:   provisionAccount(cfg.Users.Reviewer),
			ReviewerCI: provisionAccount(cfg.Users.ReviewerCI),
			Config:     doc,
		}), nil
	}, os.Executable))
	return root
}

func provisionAccount(u config.User) provision.Account {
	return provision.Account{Name: u.Name, Home: u.Home, UID: u.UID, Path: u.ExtraPath}
}

func newRemoteShell(runner kernel.Runner, cfg config.Config) *railwayssh.Shell {
	users := map[string]railwayssh.User{}
	for _, u := range cfg.Users.All() {
		users[u.Name] = railwayssh.User{Home: u.Home, ExtraPath: u.ExtraPath}
	}
	r := cfg.Server.Railway
	return railwayssh.New(runner, railwayssh.Target{
		ProjectID:     r.ProjectID,
		EnvironmentID: r.EnvironmentID,
		ServiceID:     r.ServiceID,
	}, users)
}
