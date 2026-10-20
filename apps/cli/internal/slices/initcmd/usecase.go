// Package initcmd installs the embedded gbai skills into the local agent
// skill directories and registers the gbai MCP server with the agent.
package initcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Runtime is a local agent runtime.
type Runtime string

// Runtimes.
const (
	RuntimeClaude Runtime = "claude"
	RuntimePi     Runtime = "pi"
)

// ParseRuntimes accepts claude, pi or both.
func ParseRuntimes(s string) ([]Runtime, error) {
	switch s {
	case "claude":
		return []Runtime{RuntimeClaude}, nil
	case "pi":
		return []Runtime{RuntimePi}, nil
	case "both":
		return []Runtime{RuntimeClaude, RuntimePi}, nil
	}
	return nil, fmt.Errorf("runtime must be claude, pi or both, got %q", s)
}

// Scope picks the user-wide or the project skill directories.
type Scope string

// Scopes.
const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
)

// ParseScope accepts user or project.
func ParseScope(s string) (Scope, error) {
	switch Scope(s) {
	case ScopeUser, ScopeProject:
		return Scope(s), nil
	}
	return "", fmt.Errorf("scope must be user or project, got %q", s)
}

// File is one skill file; Path is relative to the skills directory.
type File struct {
	Path string
	Data []byte
}

// Settings locate the user's home, the project and the gbai binary.
type Settings struct {
	Home       string
	ProjectDir string
	Binary     string // absolute path registered as the MCP server
}

// Input selects what to install.
type Input struct {
	Runtimes []Runtime
	Scope    Scope
	DryRun   bool
	Force    bool
}

// Output is the human-readable report.
type Output struct {
	Lines []string
}

// UseCase installs skills and registers the MCP server.
type UseCase struct {
	fs     kernel.FileSystem
	runner kernel.Runner
	files  []File
	s      Settings
}

// NewUseCase wires the use case.
func NewUseCase(fsys kernel.FileSystem, runner kernel.Runner, files []File, s Settings) *UseCase {
	return &UseCase{fs: fsys, runner: runner, files: files, s: s}
}

// Run installs the skills for each runtime, then registers the MCP server.
func (u *UseCase) Run(ctx context.Context, in Input) (Output, error) {
	if !filepath.IsAbs(u.s.Binary) {
		return Output{}, fmt.Errorf("gbai binary path %q must be absolute", u.s.Binary)
	}
	var out Output
	var errs []error
	for _, rt := range in.Runtimes {
		dir, err := u.skillsDir(rt, in.Scope)
		if err != nil {
			return out, err
		}
		for _, f := range u.files {
			line, err := u.install(filepath.Join(dir, filepath.FromSlash(f.Path)), f.Data, in)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			out.Lines = append(out.Lines, line)
		}
		lines, err := u.register(ctx, rt, in.DryRun)
		out.Lines = append(out.Lines, lines...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return out, errors.Join(errs...)
}

func (u *UseCase) skillsDir(rt Runtime, scope Scope) (string, error) {
	base := u.s.Home
	if scope == ScopeProject {
		base = u.s.ProjectDir
	}
	switch {
	case rt == RuntimeClaude:
		return filepath.Join(base, ".claude", "skills"), nil
	case rt == RuntimePi && scope == ScopeProject:
		return filepath.Join(base, ".pi", "skills"), nil
	case rt == RuntimePi:
		return filepath.Join(base, ".pi", "agent", "skills"), nil
	}
	return "", fmt.Errorf("unsupported runtime %q", rt)
}

func (u *UseCase) install(target string, data []byte, in Input) (string, error) {
	cur, err := u.fs.ReadFile(target)
	verb := "installed"
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return "", fmt.Errorf("read %s: %w", target, err)
	case bytes.Equal(cur, data):
		return "unchanged " + target, nil
	case !in.Force:
		return "skipped " + target + " (differs; use --force)", nil
	default:
		verb = "overwrote"
	}
	if in.DryRun {
		if verb == "overwrote" {
			return "would overwrite " + target, nil
		}
		return "would install " + target, nil
	}
	if err := u.fs.WriteFile(target, data); err != nil {
		return "", fmt.Errorf("write %s: %w", target, err)
	}
	return verb + " " + target, nil
}

func (u *UseCase) register(ctx context.Context, rt Runtime, dryRun bool) ([]string, error) {
	if rt == RuntimePi {
		return []string{
			"pi: gbai does not edit pi config files; add this server to your pi MCP config:",
			`  {"mcpServers": {"gbai": {"command": "` + u.s.Binary + `", "args": ["mcp"]}}}`,
		}, nil
	}
	add := kernel.Command{Name: "claude", Args: []string{"mcp", "add", "--scope", "user", "gbai", "--", u.s.Binary, "mcp"}}
	if dryRun {
		return []string{"would run: claude " + strings.Join(add.Args, " ") + " (unless `claude mcp get gbai` succeeds)"}, nil
	}
	res, err := u.runner.Run(ctx, kernel.Command{Name: "claude", Args: []string{"mcp", "get", "gbai"}})
	if err != nil {
		return nil, fmt.Errorf("claude mcp get: %w", err)
	}
	if res.ExitCode == 0 {
		return []string{"claude: MCP server gbai already registered"}, nil
	}
	if _, err := kernel.RunChecked(ctx, u.runner, add); err != nil {
		return nil, fmt.Errorf("register MCP server: %w", err)
	}
	return []string{"claude: registered MCP server gbai (user scope)"}, nil
}
