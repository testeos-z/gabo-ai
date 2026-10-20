package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/mcp"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/provision"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/sessions"
)

// lines is the structured form of a use case's line report.
type lines struct {
	Lines []string `json:"lines"`
}

func linesResult(out []string, err error) (any, error) {
	if err != nil {
		if len(out) > 0 {
			return nil, fmt.Errorf("%w\n%s", err, strings.Join(out, "\n"))
		}
		return nil, err
	}
	return lines{Lines: out}, nil
}

// mcpPorts adapts the slice factories to the MCP tool ports.
func mcpPorts(f factories) mcp.Ports {
	provisionRun := func(ctx context.Context, in provision.Input) (any, error) {
		uc, err := f.provision(ctx)
		if err != nil {
			return nil, err
		}
		out, err := uc.Run(ctx, in)
		return linesResult(out.Lines, err)
	}
	repoSetup := func(ctx context.Context, dryRun bool) (any, error) {
		uc, err := f.repo(ctx)
		if err != nil {
			return nil, err
		}
		out, err := uc.Setup(ctx, dryRun)
		return linesResult(out.Lines, err)
	}
	sessionsDo := func(ctx context.Context, role, mode string, stop bool) (any, error) {
		roles, err := sessions.ParseRoles(role)
		if err != nil {
			return nil, err
		}
		m, err := sessions.ParseMode(mode)
		if err != nil {
			return nil, err
		}
		uc, err := f.sessions(ctx, m)
		if err != nil {
			return nil, err
		}
		var out sessions.Output
		if stop {
			out, err = uc.Stop(ctx, roles)
		} else {
			out, err = uc.Start(ctx, roles, m)
		}
		return linesResult(out.Lines, err)
	}
	return mcp.Ports{
		Status: func(ctx context.Context, mode string) (any, error) {
			uc, err := f.status(ctx, mode)
			if err != nil {
				return nil, err
			}
			return uc.Snapshot(ctx), nil
		},
		Doctor: func(ctx context.Context) (any, error) {
			uc, err := f.doctor(ctx)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx), nil
		},
		Audit: func(ctx context.Context, since time.Time) (any, error) {
			uc, err := f.audit(ctx)
			if err != nil {
				return nil, err
			}
			return uc.Audit(ctx, since)
		},
		ProvisionPlan: func(ctx context.Context) (any, error) {
			return provisionRun(ctx, provision.Input{DryRun: true})
		},
		ProvisionApply: func(ctx context.Context) (any, error) {
			bin, err := os.Executable()
			if err != nil {
				return nil, fmt.Errorf("resolve running executable: %w", err)
			}
			return provisionRun(ctx, provision.Input{BinaryPath: bin})
		},
		RepoSetupPlan:  func(ctx context.Context) (any, error) { return repoSetup(ctx, true) },
		RepoSetupApply: func(ctx context.Context) (any, error) { return repoSetup(ctx, false) },
		SessionsStart: func(ctx context.Context, role, mode string) (any, error) {
			return sessionsDo(ctx, role, mode, false)
		},
		SessionsStop: func(ctx context.Context, role, mode string) (any, error) {
			return sessionsDo(ctx, role, mode, true)
		},
		ReviewRequest: func(ctx context.Context, pr int, sha, notify string) (any, error) {
			uc, err := f.review.Request(ctx)
			if err != nil {
				return nil, err
			}
			return uc.Request(ctx, review.RequestInput{PR: pr, SHA: sha, Notify: notify})
		},
	}
}
