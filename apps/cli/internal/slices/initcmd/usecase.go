// Package initcmd installs the embedded gbai skills into the local agent
// skill directories.
package initcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

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

// Settings locate the user's home and the project.
type Settings struct {
	Home       string
	ProjectDir string
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

// UseCase installs skills.
type UseCase struct {
	fs    kernel.FileSystem
	files []File
	s     Settings
}

// NewUseCase wires the use case.
func NewUseCase(fsys kernel.FileSystem, files []File, s Settings) *UseCase {
	return &UseCase{fs: fsys, files: files, s: s}
}

// Run installs the skills for each runtime.
func (u *UseCase) Run(_ context.Context, in Input) (Output, error) {
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
