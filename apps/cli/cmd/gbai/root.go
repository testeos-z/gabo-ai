package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/railwayssh"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	provisionadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	reviewadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/review/adapters"
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
	root.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath,
		"gbai config file; when not set and ./"+defaultConfigPath+" is missing, "+kernel.ServerConfigPath+" is used")

	loadConfig := func() (config.Config, error) {
		return config.Load(resolveConfigPath(configPath, root.PersistentFlags().Changed("config")))
	}

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
	root.AddCommand(review.NewCommand(reviewFactories(d.runner, loadConfig)))
	return root
}

const defaultConfigPath = "gbai.yaml"

// resolveConfigPath uses the server config when no flag is given and there is
// no gbai.yaml in the working directory.
func resolveConfigPath(p string, explicit bool) string {
	if explicit || p != defaultConfigPath {
		return p
	}
	if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
		return kernel.ServerConfigPath
	}
	return p
}

func reviewFactories(runner kernel.Runner, loadConfig func() (config.Config, error)) review.Factories {
	return review.Factories{
		Request: func(context.Context) (*review.RequestUseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return review.NewRequestUseCase(reviewadapters.NewOrca(runner), cfg.Users.Reviewer.Name), nil
		},
		Notify: func(context.Context) (*review.NotifyUseCase, error) {
			return review.NewNotifyUseCase(reviewadapters.NewOrca(runner)), nil
		},
		Run: func(context.Context) (*review.RunUseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			rv := cfg.Users.Reviewer
			ci := reviewadapters.NewCIClient(runner, cfg.Users.ReviewerCI.Name, string(cfg.Runtimes.Reviewer),
				reviewadapters.CredentialReader(path.Join(rv.Home, "secrets")))
			return review.NewRunUseCase(
				reviewadapters.NewGitHub(runner, cfg.Repo.Name),
				reviewadapters.NewGit(runner, cfg.Repo.Name, path.Join(rv.Home, "work"), nil),
				ci, ci,
				review.RunConfig{
					Checks: cfg.Review.CheckArgv(),
					Policy: review.MergePolicy{
						AutoMerge:      cfg.Review.AutoMerge,
						Integration:    cfg.Repo.IntegrationBranch,
						HumanOnlyPaths: cfg.Review.HumanOnlyPaths,
					},
				},
			), nil
		},
		CI: func(context.Context) (*review.CIUseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return review.NewCIUseCase(
				reviewadapters.NewDirWorkspace(path.Join(cfg.Users.ReviewerCI.Home, "work")),
				runner,
				review.CIConfig{CredentialEnv: map[string]string{
					review.RuntimeClaude: "CLAUDE_CODE_OAUTH_TOKEN",
					review.RuntimePi:     cfg.Review.PiCredentialEnv,
				}},
			), nil
		},
	}
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
