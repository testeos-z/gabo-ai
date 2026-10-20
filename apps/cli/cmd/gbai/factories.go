package main

import (
	"context"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
	auditadapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit/adapters"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor"
	doctoradapters "github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor/adapters"
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
}

func newFactories(d deps, loadConfig func() (config.Config, error)) factories {
	return factories{
		secrets: func(context.Context) (*secrets.UseCase, error) {
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
		},
		provision: func(context.Context) (*provision.UseCase, error) {
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
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return sessions.NewUseCase(newTerminals(d.runner, cfg, string(mode)), sessions.Settings{
				sessions.RoleCoder:     {Runtime: string(cfg.Runtimes.Coder), PromptFile: cfg.Sessions.Coder.PromptFile},
				sessions.RoleReviewer:  {Runtime: string(cfg.Runtimes.Reviewer), PromptFile: cfg.Sessions.Reviewer.PromptFile},
				sessions.RoleAssistant: {Runtime: string(cfg.Runtimes.Assistant), PromptFile: cfg.Sessions.Assistant.PromptFile},
			}), nil
		},
		status: func(_ context.Context, mode string) (*status.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return status.NewUseCase(newRemoteShell(d.runner, cfg), newTerminals(d.runner, cfg, mode),
				statusadapters.NewGitHub(d.runner, cfg.Repo.Name),
				status.Settings{Integration: cfg.Repo.IntegrationBranch, Bot: cfg.Bot.Login}), nil
		},
		audit: func(context.Context) (*audit.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			return audit.NewUseCase(auditadapters.NewGitHub(d.runner, cfg.Repo.Name), audit.Settings{
				Integration: cfg.Repo.IntegrationBranch,
				Bot:         cfg.Bot.Login,
				Coder:       cfg.Coder.Login,
			}), nil
		},
		doctor: func(context.Context) (*doctor.UseCase, error) {
			cfg, err := loadConfig()
			if err != nil {
				return nil, err
			}
			users := make([]doctor.User, 0, 3)
			for _, u := range cfg.Users.All() {
				users = append(users, doctor.User{Name: u.Name, UID: u.UID})
			}
			return doctor.NewUseCase(doctoradapters.Path{}, d.runner, newRemoteShell(d.runner, cfg), doctor.Settings{
				LocalOrca:    cfg.Sessions.LocalOrca,
				Remote:       cfg.Server.Railway.ServiceID != "",
				Users:        users,
				OrcaUser:     cfg.Users.Orca.Name,
				ReviewerHome: cfg.Users.Reviewer.Home,
			}), nil
		},
	}
}
