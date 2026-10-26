package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gbai-team/apps/cli/internal/skills"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
	auditadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor"
	doctoradapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	provisionadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/repo"
	repoadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/repo/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets"
	secretsadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/secrets/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/sessions"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/status"
	statusadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/status/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/worktree"
	worktreeadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/worktree/adapters"
)

// factories build each slice's use case lazily from the config, so help and
// argument errors never need configuration. Cobra commands and MCP tools share them.
type factories struct {
	secrets   secrets.Factory
	provision provision.Factory
	review    review.Factories
	repo      repo.Factory
	sessions  sessions.Factory
	status    status.Factory
	audit     audit.Factory
	doctor    doctor.Factory
	init      initcmd.Factory
	worktree  worktree.Factory
}

func newFactories(d deps, loadConfig func() (config.Config, error)) factories {
	return factories{
		secrets: func(context.Context) (*secrets.UseCase, error) {
			cfg, err := loadCloudConfig(loadConfig)
			if err != nil {
				return nil, err
			}
			// Single-user mode: the reviewer runs as the orca user, so its
			// credentials belong to that user.
			owner := cfg.Users.Orca
			return secrets.NewUseCase(
				newRemoteShell(d.runner, cfg),
				secretsadapters.NewStdinReader(d.stdin, d.stderr),
				secrets.Account{Name: owner.Name, Home: owner.Home},
			), nil
		},
		provision: func(context.Context) (*provision.UseCase, error) {
			cfg, err := loadCloudConfig(loadConfig)
			if err != nil {
				return nil, err
			}
			doc, err := cfg.Marshal()
			if err != nil {
				return nil, err
			}
			skillsTar, err := skills.Tar()
			if err != nil {
				return nil, err
			}
			return provision.NewUseCase(newRemoteShell(d.runner, cfg), provisionadapters.FileBinary{}, provision.Settings{
				Orca:      provisionAccount(cfg.Users.Orca),
				Config:    doc,
				SkillsTar: skillsTar,
			}), nil
		},
		review: reviewFactories(d.runner, loadConfig),
		repo: func(context.Context) (*repo.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return repo.NewUseCase(repoadapters.NewAPI(d.runner), repo.Settings{
				Repo:        cfg.Repo.Name,
				Integration: cfg.Repo.IntegrationBranch,
			}), nil
		},
		sessions: func(_ context.Context, mode sessions.Mode) (*sessions.UseCase, error) {
			load := loadConfig
			if mode == sessions.ModeCloud {
				load = func() (config.Config, error) { return loadCloudConfig(loadConfig) }
			}
			cfg, err := load()
			if err != nil {
				return nil, err
			}
			return sessions.NewUseCase(newTerminals(d.runner, cfg, string(mode)), sessions.Settings{
				sessions.RoleCoder:     {Runtime: string(cfg.Runtimes.Coder), PromptFile: cfg.Sessions.Coder.PromptFile},
				sessions.RoleReviewer:  {Runtime: string(cfg.Runtimes.Reviewer), PromptFile: cfg.Sessions.Reviewer.PromptFile},
				sessions.RoleAssistant: {Runtime: string(cfg.Runtimes.Assistant), PromptFile: cfg.Sessions.Assistant.PromptFile},
			}), nil
		},
		status: func(ctx context.Context, mode string) (*status.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			// status is read-only: a failed owner login lookup degrades the
			// verdict lookups to unknown instead of failing the snapshot.
			var gh status.GitHub = statusadapters.NewGitHub(d.runner, cfg.Repo.Name)
			login, err := ownerLogin(ctx, d.runner, cfg.Bot.Login)
			if err != nil {
				gh = loginlessGitHub{GitHub: gh, err: err}
			}
			return status.NewUseCase(newRemoteShell(d.runner, cfg), newTerminals(d.runner, cfg, mode), gh,
				status.Settings{Integration: cfg.Repo.IntegrationBranch, Bot: login}), nil
		},
		audit: func(ctx context.Context) (*audit.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			login, err := ownerLogin(ctx, d.runner, cfg.Bot.Login)
			if err != nil {
				return nil, err
			}
			if strings.EqualFold(cfg.Coder.Login, login) {
				return nil, fmt.Errorf("coder.login %s is the owner login the reviewer merges with; unset it in single-user mode", cfg.Coder.Login)
			}
			return audit.NewUseCase(auditadapters.NewGitHub(d.runner, cfg.Repo.Name), audit.Settings{
				Integration: cfg.Repo.IntegrationBranch,
				Bot:         login,
				Coder:       cfg.Coder.Login,
			}), nil
		},
		worktree: func(context.Context) (*worktree.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return worktree.NewUseCase(worktreeadapters.NewGit(d.runner), worktreeadapters.NewGitHub(d.runner, cfg.Repo.Name),
				worktreeadapters.Paths{}, worktree.Settings{
					Repo:        cfg.Repo.Name,
					Integration: cfg.Repo.IntegrationBranch,
					Keep:        cfg.Worktree.Keep,
				}), nil
		},
		init: func(context.Context) (*initcmd.UseCase, error) {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			bin, err := os.Executable()
			if err != nil {
				return nil, fmt.Errorf("resolve running executable: %w", err)
			}
			embedded, err := skills.Files()
			if err != nil {
				return nil, err
			}
			files := make([]initcmd.File, 0, len(embedded))
			for _, f := range embedded {
				files = append(files, initcmd.File{Path: f.Path, Data: f.Data})
			}
			return initcmd.NewUseCase(osfs.FS{}, d.runner, files, initcmd.Settings{Home: home, ProjectDir: wd, Binary: bin}), nil
		},
		doctor: func(context.Context) (*doctor.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			users := make([]doctor.User, 0, 1)
			for _, u := range cfg.Users.All() {
				users = append(users, doctor.User{Name: u.Name, UID: u.UID})
			}
			return doctor.NewUseCase(doctoradapters.Path{}, d.runner, newRemoteShell(d.runner, cfg), doctor.Settings{
				LocalOrca:  cfg.Sessions.LocalOrca,
				Remote:     cfg.HasCloud(),
				Users:      users,
				OrcaUser:   cfg.Users.Orca.Name,
				SecretsDir: path.Join(cfg.Users.Orca.Home, "secrets"),
			}), nil
		},
	}
}

// loadCloudConfig loads the config for a command that reaches the cloud
// server; a local-only config fails naming each missing server key.
func loadCloudConfig(loadConfig func() (config.Config, error)) (config.Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return config.Config{}, err
	}
	if err := cfg.RequireCloud(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// loginlessGitHub answers every verdict lookup with the owner login error, so
// status reports the verdicts as unknown and keeps its other probes.
type loginlessGitHub struct {
	status.GitHub
	err error
}

func (g loginlessGitHub) LastVerdict(context.Context, int, string) (string, error) {
	return "", g.err
}
