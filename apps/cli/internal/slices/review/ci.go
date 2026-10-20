package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
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
	Path          string            // PATH given to the engine, its only inherited variable
	// UntrustedHome is writable by PR code; PATH entries inside it are dropped
	// so checks cannot plant the engine binary.
	UntrustedHome string
}

func (c CIConfig) enginePath() string {
	var keep []string
	for _, dir := range strings.Split(c.Path, ":") {
		clean := path.Clean(dir)
		if !path.IsAbs(dir) || (c.UntrustedHome != "" && (clean == c.UntrustedHome || strings.HasPrefix(clean, c.UntrustedHome+"/"))) {
			continue
		}
		keep = append(keep, dir)
	}
	return strings.Join(keep, ":")
}

// CIUseCase is the reviewer-ci side: it holds copies of merged trees and runs
// untrusted commands and the engine in them. It never sees GitHub credentials.
type CIUseCase struct {
	ws     Workspace
	runner kernel.Runner
	procs  Processes
	cfg    CIConfig
}

// NewCIUseCase wires the reviewer-ci use case.
func NewCIUseCase(ws Workspace, runner kernel.Runner, procs Processes, cfg CIConfig) *CIUseCase {
	return &CIUseCase{ws: ws, runner: runner, procs: procs, cfg: cfg}
}

// killLeftovers stops every process of the current user outside the current
// process tree, so nothing a previous check started survives into the next
// check or the engine.
func (u *CIUseCase) killLeftovers(ctx context.Context) error {
	self := u.procs.Self()
	if self.UID == 0 {
		return errors.New("review ci must not run as root")
	}
	table, err := u.procs.List()
	if err != nil {
		return fmt.Errorf("list processes: %w", err)
	}
	keep := processTree(table, self)
	var stale []int
	for _, p := range table {
		if p.UID == self.UID && !keep[p.PID] {
			stale = append(stale, p.PID)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	if err := u.procs.Terminate(ctx, stale); err != nil {
		return fmt.Errorf("kill leftover processes %v: %w", stale, err)
	}
	return nil
}

// processTree returns self, its ancestors and its descendants.
func processTree(table []Process, self Process) map[int]bool {
	parent := map[int]int{self.PID: self.PPID}
	children := map[int][]int{}
	for _, p := range table {
		parent[p.PID] = p.PPID
		children[p.PPID] = append(children[p.PPID], p.PID)
	}
	keep := map[int]bool{}
	for pid := self.PID; pid > 0 && !keep[pid]; pid = parent[pid] {
		keep[pid] = true
	}
	for queue := children[self.PID]; len(queue) > 0; queue = queue[1:] {
		if pid := queue[0]; !keep[pid] {
			keep[pid] = true
			queue = append(queue, children[pid]...)
		}
	}
	return keep
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
	if err := u.killLeftovers(ctx); err != nil {
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
	if err := u.killLeftovers(ctx); err != nil {
		return kernel.Result{}, err
	}
	home, err := u.ws.Home(name)
	if err != nil {
		return kernel.Result{}, fmt.Errorf("fresh engine home: %w", err)
	}
	return u.runner.Run(ctx, kernel.Command{
		Name: argv[0], Args: argv[1:], Dir: dir, Stdin: r, CleanEnv: true,
		Env: []string{"PATH=" + u.cfg.enginePath(), "HOME=" + home, env + "=" + cred},
	})
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
