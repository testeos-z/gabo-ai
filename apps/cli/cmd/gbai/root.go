package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/orcacli"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/railwayssh"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/configcmd"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/mcp"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/plan"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/repo"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	reviewadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/review/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/sessions"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/status"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/version"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/worktree"
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
	root.PersistentFlags().StringVar(&configPath, "config", "",
		"load exactly this config file; when not set the layers "+kernel.ServerConfigPath+
			", the user config ($XDG_CONFIG_HOME/gbai/config.yaml) and the project "+config.ProjectFile+
			" are merged (see `gbai config`)")

	resolveConfig := newWarnOnceResolver(d.stderr, func(ctx context.Context) (config.Resolution, error) {
		if root.PersistentFlags().Changed("config") {
			return config.LoadFile(osfs.FS{}, configPath)
		}
		return config.LoadLayers(osfs.FS{}, configLayers(ctx, d.runner, os.Getenv, os.Getwd))
	})
	loadConfig := func() (config.Config, error) {
		res, err := resolveConfig(context.Background())
		return res.Config, err
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
	root.AddCommand(worktree.NewCommand(f.worktree))
	root.AddCommand(initcmd.NewCommand(f.init))
	root.AddCommand(configcmd.NewCommand(resolveConfig))
	root.AddCommand(plan.NewCommand(plan.NewUseCase(osfs.FS{})))
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

// configLayers returns the system, user and project layers; the project root
// is the git top-level of the working directory, or the directory itself.
func configLayers(ctx context.Context, runner kernel.Runner, getenv func(string) string, getwd func() (string, error)) []config.Layer {
	root := ""
	if cwd, err := getwd(); err == nil {
		root = config.ProjectRoot(ctx, runner, cwd)
	}
	return config.LayerPaths(getenv, root)
}

func reviewFactories(runner kernel.Runner, loadConfig func() (config.Config, error)) review.Factories {
	return review.Factories{
		Request: func(context.Context) (*review.RequestUseCase, error) {
			if _, err := loadConfig(); err != nil {
				return nil, err
			}
			return review.NewRequestUseCase(reviewadapters.NewOrca(runner)), nil
		},
		Notify: func(context.Context) (*review.NotifyUseCase, error) {
			return review.NewNotifyUseCase(reviewadapters.NewOrca(runner)), nil
		},
		Run: func(_ context.Context, keepWorktree bool) (*review.RunUseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			state, err := reviewerState(cfg)
			if err != nil {
				return nil, err
			}
			ci, err := newCIUseCase(runner, cfg, state)
			if err != nil {
				return nil, err
			}
			local := reviewadapters.NewLocalCI(ci, string(cfg.Runtimes.Reviewer),
				reviewadapters.CredentialReader(path.Join(cfg.Users.Orca.Home, "secrets")))
			return review.NewRunUseCase(
				reviewadapters.NewGitHub(runner, cfg.Repo.Name),
				reviewadapters.NewGit(runner, cfg.Repo.Name, state, nil),
				local, local,
				review.RunConfig{
					Checks: cfg.Review.CheckArgv(),
					Policy: review.MergePolicy{
						AutoMerge:      cfg.Review.AutoMerge,
						Integration:    cfg.Repo.IntegrationBranch,
						HumanOnlyPaths: cfg.Review.HumanOnlyPaths,
					},
					KeepWorktree: keepWorktree,
					Login:        cfg.Bot.Login,
				},
			).WithLock(reviewadapters.NewFileLock(path.Join(state, "review.lock"))), nil
		},
		CI: func(context.Context) (*review.CIUseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			state, err := reviewerState(cfg)
			if err != nil {
				return nil, err
			}
			return newCIUseCase(runner, cfg, state)
		},
	}
}

// reviewerState resolves the reviewer's own state dir (clone/, worktrees/,
// ci/) and creates it private to the current user.
func reviewerState(cfg config.Config) (string, error) {
	state, err := cfg.Review.StateDirFrom(os.Getenv)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(state, 0o700); err != nil {
		return "", fmt.Errorf("create reviewer state dir: %w", err)
	}
	return state, nil
}

// newCIUseCase builds the checks side under <state>/ci: workspaces in work/,
// a persistent HOME for the checks' caches in home/. In single-user mode it
// runs as the owner; its environment is built from scratch so no GitHub
// credential reaches the checks or the engine.
func newCIUseCase(runner kernel.Runner, cfg config.Config, state string) (*review.CIUseCase, error) {
	root := path.Join(state, "ci")
	home := path.Join(root, "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("create checks home: %w", err)
	}
	return review.NewCIUseCase(
		reviewadapters.NewDirWorkspace(path.Join(root, "work")),
		runner,
		reviewadapters.NewProcFS("/proc", 2*time.Second),
		review.CIConfig{
			CredentialEnv: map[string]string{
				review.RuntimeClaude: "CLAUDE_CODE_OAUTH_TOKEN",
				review.RuntimePi:     cfg.Review.PiCredentialEnv,
			},
			Path:          os.Getenv("PATH"),
			Home:          home,
			User:          os.Getenv("USER"),
			UntrustedRoot: root,
		},
	), nil
}

// ownerLogin returns the owner's GitHub login: the configured bot.login, or
// the login gh is authenticated as.
func ownerLogin(ctx context.Context, runner kernel.Runner, configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	res, err := kernel.RunChecked(ctx, runner, kernel.Command{Name: "gh", Args: []string{"api", "user", "--jq", ".login"}})
	if err != nil {
		return "", fmt.Errorf("owner login from gh (or set bot.login): %w", err)
	}
	login := strings.TrimSpace(string(res.Stdout))
	if login == "" || strings.ContainsAny(login, " \t\n") {
		return "", fmt.Errorf("owner login from gh: unexpected output %q", login)
	}
	return login, nil
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

// printWarnings writes each non-fatal config warning to w; nil w drops them.
// newWarnOnceResolver wraps a config resolver so each distinct warning reaches
// w once per process: config is re-read on every load (the MCP server loads it
// per tool call), but the same warning is not repeated.
func newWarnOnceResolver(w io.Writer, resolve func(context.Context) (config.Resolution, error)) func(context.Context) (config.Resolution, error) {
	var mu sync.Mutex
	seen := map[string]bool{}
	return func(ctx context.Context) (config.Resolution, error) {
		res, err := resolve(ctx)
		mu.Lock()
		defer mu.Unlock()
		var fresh []string
		for _, m := range res.Warnings {
			if !seen[m] {
				seen[m] = true
				fresh = append(fresh, m)
			}
		}
		printWarnings(w, fresh)
		return res, err
	}
}

func printWarnings(w io.Writer, warnings []string) {
	if w == nil {
		return
	}
	for _, m := range warnings {
		fmt.Fprintln(w, "gbai: warning:", m)
	}
}
