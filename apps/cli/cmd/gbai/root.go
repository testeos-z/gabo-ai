package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/orcacli"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/railwayssh"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/mcp"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/repo"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	reviewadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/review/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/sessions"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/status"
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

	f := newFactories(d, loadConfig)
	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	root.AddCommand(secrets.NewCommand(f.secrets))
	root.AddCommand(provision.NewCommand(f.provision, os.Executable))
	root.AddCommand(review.NewCommand(f.review))
	root.AddCommand(repo.NewCommand(f.repo))
	root.AddCommand(sessions.NewCommand(f.sessions))
	root.AddCommand(status.NewCommand(f.status))
	root.AddCommand(audit.NewCommand(f.audit, time.Now))
	root.AddCommand(doctor.NewCommand(f.doctor))
	root.AddCommand(mcp.NewCommand(mcp.NewTools(mcpPorts(f), secrets.Names(), time.Now), serverVersion(d.version)))
	return root
}

// newTerminals reaches Orca on the server as the orca user (cloud) or on this machine (local).
func newTerminals(runner kernel.Runner, cfg config.Config, mode string) kernel.Terminals {
	if mode == "local" {
		return orcacli.NewLocal(runner, cfg.Sessions.LocalOrca)
	}
	return orcacli.NewRemote(newRemoteShell(runner, cfg), cfg.Users.Orca.Name)
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

// serverVersion is the version announced by the MCP server.
func serverVersion(v string) string {
	if v == "" {
		return "dev"
	}
	return v
}
