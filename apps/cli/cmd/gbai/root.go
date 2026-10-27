package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/plan"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	reviewadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/review/adapters"
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
		Short:         "Compile the gbai agent team into a project",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&configPath, "config", "",
		"load exactly this config file; when not set the layers "+kernel.ServerConfigPath+
			", the user config ($XDG_CONFIG_HOME/gbai/config.yaml) and the project "+config.ProjectFile+
			" are merged")

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

	root.AddCommand(version.NewCommand(version.NewUseCase(d.version)))
	root.AddCommand(review.NewCommand(reviewFactories(d.runner, loadConfig)))
	root.AddCommand(initcmd.NewCommand(initFactory(d.runner, d.version)))
	root.AddCommand(plan.NewCommand(plan.NewUseCase(osfs.FS{})))
	return root
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
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("reviewer credentials: %w", err)
			}
			local := reviewadapters.NewLocalCI(ci, string(cfg.Runtimes.Reviewer),
				reviewadapters.CredentialReader(path.Join(home, "secrets")))
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

// newWarnOnceResolver wraps a config resolver so each distinct warning reaches
// w once per process: config is re-read on every load, but the same warning
// is not repeated.
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

// printWarnings writes each non-fatal config warning to w; nil w drops them.
func printWarnings(w io.Writer, warnings []string) {
	if w == nil {
		return
	}
	for _, m := range warnings {
		fmt.Fprintln(w, "gbai: warning:", m)
	}
}
