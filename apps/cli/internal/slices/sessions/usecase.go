// Package sessions starts and stops the named gbai agent sessions as Orca
// terminals, on the server (cloud mode) or on this machine (local mode).
package sessions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Role is an agent role with one long-lived session.
type Role string

// Roles.
const (
	RoleCoder     Role = "coder"
	RoleReviewer  Role = "reviewer"
	RoleAssistant Role = "assistant"
)

var allRoles = []Role{RoleCoder, RoleReviewer, RoleAssistant}

// Mode tells the session where it runs; it is exported as GBAI_MODE.
type Mode string

// Modes.
const (
	ModeCloud Mode = "cloud"
	ModeLocal Mode = "local"
)

// ParseRoles accepts one role name or "all".
func ParseRoles(s string) ([]Role, error) {
	if s == "all" {
		return append([]Role(nil), allRoles...), nil
	}
	for _, r := range allRoles {
		if string(r) == s {
			return []Role{r}, nil
		}
	}
	return nil, fmt.Errorf("role must be coder, reviewer, assistant or all, got %q", s)
}

// ParseMode accepts cloud or local.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeCloud, ModeLocal:
		return Mode(s), nil
	}
	return "", fmt.Errorf("mode must be cloud or local, got %q", s)
}

// Title is the exact Orca terminal title of a role's session.
func Title(r Role) string { return "gbai-" + string(r) }

// RoleSettings is the runtime (claude or pi) and optional startup prompt file of a role.
type RoleSettings struct {
	Runtime    string
	PromptFile string
}

// Settings maps each role to its settings.
type Settings map[Role]RoleSettings

// Output is the human-readable report, one line per role.
type Output struct {
	Lines []string
}

// UseCase manages the sessions through Orca.
type UseCase struct {
	terms kernel.Terminals
	s     Settings
}

// NewUseCase wires the use case to the Orca host selected by the mode.
func NewUseCase(terms kernel.Terminals, s Settings) *UseCase { return &UseCase{terms: terms, s: s} }

// Start opens a terminal per role unless one with its title already exists.
func (u *UseCase) Start(ctx context.Context, roles []Role, mode Mode) (Output, error) {
	live, err := u.terms.List(ctx)
	if err != nil {
		return Output{}, fmt.Errorf("list orca terminals: %w", err)
	}
	var out Output
	var errs []error
	for _, r := range roles {
		title := Title(r)
		if h, ok := findTitle(live, title); ok {
			out.Lines = append(out.Lines, fmt.Sprintf("%s: already running (%s)", title, h))
			continue
		}
		command, err := u.command(r, mode)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", title, err))
			continue
		}
		res, err := u.terms.Create(ctx, title, command)
		if err != nil {
			errs = append(errs, fmt.Errorf("start %s: %w", title, err))
			continue
		}
		out.Lines = append(out.Lines, fmt.Sprintf("%s: started %s", title, res))
	}
	return out, errors.Join(errs...)
}

// Stop closes the terminals whose title is exactly one of the roles' titles.
func (u *UseCase) Stop(ctx context.Context, roles []Role) (Output, error) {
	live, err := u.terms.List(ctx)
	if err != nil {
		return Output{}, fmt.Errorf("list orca terminals: %w", err)
	}
	var out Output
	var errs []error
	for _, r := range roles {
		title := Title(r)
		found := false
		for _, t := range live {
			if t.Title != title {
				continue
			}
			found = true
			if err := u.terms.Close(ctx, t.Handle); err != nil {
				errs = append(errs, fmt.Errorf("stop %s (%s): %w", title, t.Handle, err))
				continue
			}
			out.Lines = append(out.Lines, fmt.Sprintf("%s: stopped (%s)", title, t.Handle))
		}
		if !found {
			out.Lines = append(out.Lines, title+": not running")
		}
	}
	return out, errors.Join(errs...)
}

func (u *UseCase) command(r Role, mode Mode) (string, error) {
	rs := u.s[r]
	if rs.Runtime != "claude" && rs.Runtime != "pi" {
		return "", fmt.Errorf("unsupported runtime %q", rs.Runtime)
	}
	c := "env GBAI_MODE=" + shellQuote(string(mode)) + " " + rs.Runtime
	if rs.PromptFile != "" {
		c += ` "$(cat ` + shellQuote(rs.PromptFile) + `)"`
	}
	return c, nil
}

func findTitle(live []kernel.Terminal, title string) (string, bool) {
	for _, t := range live {
		if t.Title == title {
			return t.Handle, true
		}
	}
	return "", false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
