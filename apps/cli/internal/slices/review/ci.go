package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Review runtimes.
const (
	RuntimeClaude = "claude"
	RuntimePi     = "pi"
)

const maxCredentialBytes = 64 << 10

var workspacePattern = regexp.MustCompile(`^pr-[1-9][0-9]*-[0-9a-f]{7}$`)

// EngineArgv returns the read-only engine invocation for runtime.
func EngineArgv(runtime string) ([]string, error) {
	switch runtime {
	case RuntimeClaude:
		return []string{"claude", "-p", "--setting-sources", "user", "--strict-mcp-config", "--output-format", "json", "--allowedTools", "Read,Grep,Glob"}, nil
	case RuntimePi:
		return []string{"pi", "-p", "--mode", "json", "--no-mcp", "--no-extensions", "--no-context-files", "--no-skills", "--tools", "read,grep,find,ls"}, nil
	default:
		return nil, fmt.Errorf("unknown review runtime %q", runtime)
	}
}

// CIConfig configures the reviewer-ci side.
type CIConfig struct {
	CredentialEnv map[string]string // runtime -> env var receiving the model credential
}

// CIUseCase is the reviewer-ci side: it holds copies of merged trees and runs
// untrusted commands and the engine in them. It never sees GitHub credentials.
type CIUseCase struct {
	ws     Workspace
	runner kernel.Runner
	cfg    CIConfig
}

// NewCIUseCase wires the reviewer-ci use case.
func NewCIUseCase(ws Workspace, runner kernel.Runner, cfg CIConfig) *CIUseCase {
	return &CIUseCase{ws: ws, runner: runner, cfg: cfg}
}

// ValidateWorkspace checks a workspace name produced by WorkspaceName.
func ValidateWorkspace(name string) error {
	if !workspacePattern.MatchString(name) {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	return nil
}

// Prepare replaces the workspace with the tar archive read from tree.
func (u *CIUseCase) Prepare(name string, tree io.Reader) error {
	if err := ValidateWorkspace(name); err != nil {
		return err
	}
	if _, err := u.ws.Create(name, tree); err != nil {
		return fmt.Errorf("create workspace %s: %w", name, err)
	}
	return nil
}

// Run executes argv as-is (no shell) inside the workspace.
func (u *CIUseCase) Run(ctx context.Context, name string, argv []string) (kernel.Result, error) {
	if err := ValidateWorkspace(name); err != nil {
		return kernel.Result{}, err
	}
	if len(argv) == 0 || argv[0] == "" {
		return kernel.Result{}, errors.New("missing command")
	}
	dir, err := u.ws.Dir(name)
	if err != nil {
		return kernel.Result{}, err
	}
	return u.runner.Run(ctx, kernel.Command{Name: argv[0], Args: argv[1:], Dir: dir})
}

// Engine runs the review engine in the workspace. stdin carries the model
// credential on its first line followed by the prompt.
func (u *CIUseCase) Engine(ctx context.Context, name, runtime string, stdin io.Reader) (kernel.Result, error) {
	if err := ValidateWorkspace(name); err != nil {
		return kernel.Result{}, err
	}
	argv, err := EngineArgv(runtime)
	if err != nil {
		return kernel.Result{}, err
	}
	env := u.cfg.CredentialEnv[runtime]
	if env == "" {
		return kernel.Result{}, fmt.Errorf("no credential env configured for %s", runtime)
	}
	r := bufio.NewReaderSize(stdin, 4096)
	line, err := readLine(r, maxCredentialBytes)
	if err != nil {
		return kernel.Result{}, fmt.Errorf("read credential: %w", err)
	}
	cred := strings.TrimSpace(line)
	if cred == "" {
		return kernel.Result{}, errors.New("empty model credential")
	}
	dir, err := u.ws.Dir(name)
	if err != nil {
		return kernel.Result{}, err
	}
	return u.runner.Run(ctx, kernel.Command{Name: argv[0], Args: argv[1:], Dir: dir, Env: []string{env + "=" + cred}, Stdin: r})
}

// Cleanup removes the workspace.
func (u *CIUseCase) Cleanup(name string) error {
	if err := ValidateWorkspace(name); err != nil {
		return err
	}
	return u.ws.Remove(name)
}

func readLine(r *bufio.Reader, limit int) (string, error) {
	var b strings.Builder
	for {
		c, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if c == '\n' {
			return b.String(), nil
		}
		if b.Len() >= limit {
			return "", errors.New("line too long")
		}
		b.WriteByte(c)
	}
}
