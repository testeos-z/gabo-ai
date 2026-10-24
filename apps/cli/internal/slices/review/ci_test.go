package review

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

type fakeWorkspace struct {
	created map[string]string
	removed []string
	homes   []string
}

// fakeProcesses is an in-memory process table.
type fakeProcesses struct {
	t          *trace
	self       Process
	table      []Process
	listErr    error
	termErr    error
	terminated [][]int
}

func (f *fakeProcesses) Self() Process { return f.self }

func (f *fakeProcesses) List() ([]Process, error) { return f.table, f.listErr }

func (f *fakeProcesses) Terminate(_ context.Context, pids []int) error {
	f.t.add("procs.terminate %v", pids)
	f.terminated = append(f.terminated, pids)
	return f.termErr
}

// tracingRunner records runner calls in the shared trace.
type tracingRunner struct {
	*kerneltest.Runner
	t *trace
}

func (r tracingRunner) Run(ctx context.Context, c kernel.Command) (kernel.Result, error) {
	r.t.add("run %s", c.Name)
	return r.Runner.Run(ctx, c)
}

// procTable is a single-user table: the reviewer, its checks and the owner's
// own sessions share one uid, so only cwd inside the untrusted root marks a
// leftover.
func procTable() *fakeProcesses {
	return &fakeProcesses{
		t:    &trace{},
		self: Process{PID: 100, PPID: 90, UID: 1000},
		table: []Process{
			{PID: 1, PPID: 0, UID: 0},
			{PID: 90, PPID: 1, UID: 1000, Cwd: "/data/home"},                // the review terminal, our parent
			{PID: 100, PPID: 90, UID: 1000, Cwd: "/data/home"},              // ourselves
			{PID: 101, PPID: 100, UID: 1000, Cwd: "/ci/work/" + wsName},     // our own child
			{PID: 200, PPID: 1, UID: 1000, Cwd: "/ci/work/" + wsName},       // leftover from a previous check
			{PID: 201, PPID: 200, UID: 1000, Cwd: "/ci/home"},               // its child
			{PID: 210, PPID: 1, UID: 1000, Cwd: "/data/home/projects/repo"}, // the owner's coder session
			{PID: 211, PPID: 1, UID: 1000, Cwd: "/cix"},                     // a sibling of the root, not inside it
			{PID: 300, PPID: 1, UID: 2000, Cwd: "/ci/work/" + wsName},       // another user
		},
	}
}

func (f *fakeWorkspace) Create(name string, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if f.created == nil {
		f.created = map[string]string{}
	}
	f.created[name] = string(b)
	return "/ci/work/" + name, nil
}

func (f *fakeWorkspace) Dir(name string) (string, error) {
	if _, ok := f.created[name]; !ok {
		return "", errors.New("missing workspace")
	}
	return "/ci/work/" + name, nil
}

func (f *fakeWorkspace) Home(name string) (string, error) {
	f.homes = append(f.homes, name)
	return "/ci/work/" + name + ".home", nil
}

func (f *fakeWorkspace) Remove(name string) error {
	f.removed = append(f.removed, name)
	return nil
}

func newCI(r *kerneltest.Runner) (*CIUseCase, *fakeWorkspace) {
	uc, ws, _ := newCIWith(r, &fakeProcesses{t: &trace{}, self: Process{PID: 100, PPID: 90, UID: 1000}})
	return uc, ws
}

func newCIWith(r *kerneltest.Runner, p *fakeProcesses) (*CIUseCase, *fakeWorkspace, *trace) {
	ws := &fakeWorkspace{}
	cfg := CIConfig{
		CredentialEnv: map[string]string{RuntimeClaude: "CLAUDE_CODE_OAUTH_TOKEN", RuntimePi: "ANTHROPIC_API_KEY"},
		Path:          "/usr/bin:/bin",
		Home:          "/ci/home",
		User:          "orca",
		UntrustedRoot: "/ci",
	}
	return NewCIUseCase(ws, tracingRunner{Runner: r, t: p.t}, p, cfg), ws, p.t
}

func TestCIRunExecsPlainArgvInWorkspace(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 2}}
	ci, ws := newCI(r)
	if err := ci.Prepare(wsName, strings.NewReader("TAR")); err != nil {
		t.Fatal(err)
	}
	res, err := ci.Run(context.Background(), wsName, []string{"bun", "run", "test; rm -rf /"})
	if err != nil || res.ExitCode != 2 {
		t.Fatalf("Run() = %+v, %v", res, err)
	}
	c := r.Calls[0].Command
	if c.Name != "bun" || !slices.Equal(c.Args, []string{"run", "test; rm -rf /"}) || c.Dir != "/ci/work/"+wsName {
		t.Fatalf("command = %+v", c)
	}
	if ws.created[wsName] != "TAR" {
		t.Fatalf("workspace = %v", ws.created)
	}
	want := []string{"PATH=/usr/bin:/bin", "HOME=/ci/home", "USER=orca", "LOGNAME=orca", "GH_CONFIG_DIR=/ci/work/" + wsName + ".home"}
	if !c.CleanEnv || !slices.Equal(c.Env, want) {
		t.Fatalf("check env clean=%v %q, want only %q", c.CleanEnv, c.Env, want)
	}
}

// Without a second OS user, checks and the engine run as the owner; their
// environment is built from scratch so no GitHub credential reaches them and
// gh reads an empty config dir instead of the owner's login.
func TestCIChecksAndEngineGetNoGitHubCredentials(t *testing.T) {
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("{}")}}
	t.Setenv("GH_TOKEN", "gho_secret")
	t.Setenv("GITHUB_TOKEN", "ghs_secret")
	ci, _ := newCI(r)
	if err := ci.Prepare(wsName, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	if _, err := ci.Run(context.Background(), wsName, []string{"bun", "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ci.Engine(context.Background(), wsName, RuntimePi, strings.NewReader("tok\np")); err != nil {
		t.Fatal(err)
	}
	if len(r.Calls) != 2 {
		t.Fatalf("calls = %d", len(r.Calls))
	}
	for _, call := range r.Calls {
		c := call.Command
		if !c.CleanEnv {
			t.Errorf("%s inherits the environment", c.Name)
		}
		ghDir := false
		for _, kv := range c.Env {
			name, value, _ := strings.Cut(kv, "=")
			switch name {
			case "GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN":
				t.Errorf("%s gets %s", c.Name, name)
			case "GH_CONFIG_DIR":
				ghDir = value == "/ci/work/"+wsName+".home"
			}
			if strings.Contains(value, "secret") {
				t.Errorf("%s env leaks %q", c.Name, kv)
			}
		}
		if !ghDir {
			t.Errorf("%s: GH_CONFIG_DIR not pointed at the fresh empty dir: %q", c.Name, c.Env)
		}
	}
}

func TestCIRejectsBadWorkspaceNames(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ci, ws := newCI(r)
	for _, name := range []string{"", "../x", "pr-1-ABCDEFG", "pr-0-0123456", "/tmp/x", "pr-1-0123456/.."} {
		if err := ci.Prepare(name, strings.NewReader("")); err == nil {
			t.Errorf("Prepare(%q) error = nil", name)
		}
		if _, err := ci.Run(context.Background(), name, []string{"true"}); err == nil {
			t.Errorf("Run(%q) error = nil", name)
		}
		if err := ci.Cleanup(name); err == nil {
			t.Errorf("Cleanup(%q) error = nil", name)
		}
	}
	if len(r.Calls) != 0 || len(ws.created) != 0 || len(ws.removed) != 0 {
		t.Fatal("bad names reached ports")
	}
}

func TestCIEngineUsesReadOnlyArgvAndCredentialFromStdin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		runtime string
		argv    []string
		env     string
	}{
		{RuntimeClaude, []string{"claude", "-p", "--setting-sources", "user", "--strict-mcp-config", "--output-format", "json", "--allowedTools", "Read,Grep,Glob"}, "CLAUDE_CODE_OAUTH_TOKEN=tok"},
		{RuntimePi, []string{"pi", "-p", "--mode", "json", "--no-mcp", "--no-extensions", "--no-context-files", "--no-skills", "--tools", "read,grep,find,ls"}, "ANTHROPIC_API_KEY=tok"},
	}
	for _, tt := range tests {
		r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("{}")}}
		ci, _ := newCI(r)
		if err := ci.Prepare(wsName, strings.NewReader("")); err != nil {
			t.Fatal(err)
		}
		res, err := ci.Engine(context.Background(), wsName, tt.runtime, strings.NewReader("tok\nthe prompt\nline 2"))
		if err != nil || string(res.Stdout) != "{}" {
			t.Fatalf("%s: Engine() = %+v, %v", tt.runtime, res, err)
		}
		c := r.Calls[0]
		if got := append([]string{c.Command.Name}, c.Command.Args...); !slices.Equal(got, tt.argv) {
			t.Errorf("%s: argv = %q", tt.runtime, got)
		}
		home := "/ci/work/" + wsName + ".home"
		wantEnv := []string{"PATH=/usr/bin:/bin", "HOME=" + home, "GH_CONFIG_DIR=" + home, tt.env}
		if !c.Command.CleanEnv || !slices.Equal(c.Command.Env, wantEnv) || c.Stdin != "the prompt\nline 2" || c.Command.Dir != "/ci/work/"+wsName {
			t.Errorf("%s: env %q stdin %q dir %q", tt.runtime, c.Command.Env, c.Stdin, c.Command.Dir)
		}
		for _, a := range c.Command.Args {
			if strings.Contains(a, "tok") {
				t.Errorf("%s: credential in argv", tt.runtime)
			}
		}
	}
}

func TestCIEngineRejectsUnknownRuntimeAndEmptyCredential(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ci, _ := newCI(r)
	_ = ci.Prepare(wsName, strings.NewReader(""))
	if _, err := ci.Engine(context.Background(), wsName, "opencode", strings.NewReader("tok\np")); err == nil {
		t.Error("unknown runtime accepted")
	}
	if _, err := ci.Engine(context.Background(), wsName, RuntimeClaude, strings.NewReader("\nprompt")); err == nil {
		t.Error("empty credential accepted")
	}
	if len(r.Calls) != 0 {
		t.Fatalf("calls = %d", len(r.Calls))
	}
}

func TestCICleanupRemovesWorkspace(t *testing.T) {
	t.Parallel()
	ci, ws := newCI(&kerneltest.Runner{})
	if err := ci.Cleanup(wsName); err != nil || !slices.Equal(ws.removed, []string{wsName}) {
		t.Fatalf("Cleanup() = %v, removed %v", err, ws.removed)
	}
}

func TestCIKillsLeftoversOutsideOwnTreeBeforeEachCheckAndEngine(t *testing.T) {
	t.Parallel()
	procs := procTable()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("{}")}}
	ci, ws, tr := newCIWith(r, procs)
	if err := ci.Prepare(wsName, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	if _, err := ci.Run(context.Background(), wsName, []string{"bun", "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ci.Engine(context.Background(), wsName, RuntimeClaude, strings.NewReader("tok\np")); err != nil {
		t.Fatal(err)
	}
	want := []string{"procs.terminate [200 201]", "run bun", "procs.terminate [200 201]", "run claude"}
	if !slices.Equal(tr.calls, want) {
		t.Fatalf("calls = %v, want %v", tr.calls, want)
	}
	if !slices.Equal(ws.homes, []string{wsName, wsName}) {
		t.Fatalf("fresh homes = %v, want one per check and one for the engine", ws.homes)
	}
}

func TestCISkipsTerminateWithoutLeftovers(t *testing.T) {
	t.Parallel()
	procs := procTable()
	procs.table = procs.table[:4]
	ci, _, tr := newCIWith(&kerneltest.Runner{}, procs)
	_ = ci.Prepare(wsName, strings.NewReader(""))
	if _, err := ci.Run(context.Background(), wsName, []string{"true"}); err != nil || !slices.Equal(tr.calls, []string{"run true"}) {
		t.Fatalf("err %v calls %v", err, tr.calls)
	}
}

func TestCIRefusesToRunWhenCleanupFails(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]func(*fakeProcesses){
		"root":      func(p *fakeProcesses) { p.self.UID = 0 },
		"list":      func(p *fakeProcesses) { p.listErr = errors.New("no /proc") },
		"terminate": func(p *fakeProcesses) { p.termErr = errors.New("EPERM") },
		"no root":   nil,
	} {
		procs := procTable()
		r := &kerneltest.Runner{}
		ci, _, _ := newCIWith(r, procs)
		if mutate != nil {
			mutate(procs)
		} else {
			ci.cfg.UntrustedRoot = ""
		}
		_ = ci.Prepare(wsName, strings.NewReader(""))
		if _, err := ci.Run(context.Background(), wsName, []string{"true"}); err == nil {
			t.Errorf("%s: Run error = nil", name)
		}
		if _, err := ci.Engine(context.Background(), wsName, RuntimeClaude, strings.NewReader("tok\np")); err == nil {
			t.Errorf("%s: Engine error = nil", name)
		}
		if len(r.Calls) != 0 {
			t.Errorf("%s: commands ran: %v", name, r.Calls)
		}
		if name == "root" && len(procs.terminated) != 0 {
			t.Errorf("root: terminated %v", procs.terminated)
		}
	}
}

func TestCIEngineDropsPathEntriesInsideTheUntrustedRoot(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ws := &fakeWorkspace{}
	procs := &fakeProcesses{t: &trace{}, self: Process{PID: 100, PPID: 90, UID: 1000}}
	ci := NewCIUseCase(ws, r, procs, CIConfig{
		CredentialEnv: map[string]string{RuntimeClaude: "CLAUDE_CODE_OAUTH_TOKEN"},
		Path:          "/s/ci/home/.local/bin:/usr/local/bin:bin:/s/ci:/data/home/.local/bin:/usr/bin",
		Home:          "/s/ci/home",
		UntrustedRoot: "/s/ci",
	})
	_ = ci.Prepare(wsName, strings.NewReader(""))
	if _, err := ci.Engine(context.Background(), wsName, RuntimeClaude, strings.NewReader("tok\np")); err != nil {
		t.Fatal(err)
	}
	if got := r.Calls[0].Command.Env[0]; got != "PATH=/usr/local/bin:/data/home/.local/bin:/usr/bin" {
		t.Fatalf("engine %s", got)
	}
}

// The checks' HOME is persistent and inside the untrusted root, so a PR can
// plant binaries there; relative PATH entries resolve against the PR's own
// workspace. Neither may reach the PATH of a later check, like the engine's.
func TestCIChecksDropPathEntriesInsideTheUntrustedRoot(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ws := &fakeWorkspace{}
	procs := &fakeProcesses{t: &trace{}, self: Process{PID: 100, PPID: 90, UID: 1000}}
	ci := NewCIUseCase(ws, r, procs, CIConfig{
		Path:          "/s/ci/home/.local/bin:/usr/local/bin::bin:/s/ci:/data/home/.local/bin:/usr/bin",
		Home:          "/s/ci/home",
		UntrustedRoot: "/s/ci",
	})
	_ = ci.Prepare(wsName, strings.NewReader(""))
	if _, err := ci.Run(context.Background(), wsName, []string{"bun", "test"}); err != nil {
		t.Fatal(err)
	}
	if got := r.Calls[0].Command.Env[0]; got != "PATH=/usr/local/bin:/data/home/.local/bin:/usr/bin" {
		t.Fatalf("checks %s", got)
	}
}

// A checks HOME outside the untrusted root would be the owner's real home
// (or an empty HOME that tools resolve to it): refuse instead of running.
func TestCIRunRefusesAHomeOutsideTheUntrustedRoot(t *testing.T) {
	t.Parallel()
	for name, home := range map[string]string{"empty": "", "owner home": "/data/home", "relative": "ci/home"} {
		r := &kerneltest.Runner{}
		ws := &fakeWorkspace{}
		procs := &fakeProcesses{t: &trace{}, self: Process{PID: 100, PPID: 90, UID: 1000}}
		ci := NewCIUseCase(ws, r, procs, CIConfig{Path: "/usr/bin", Home: home, UntrustedRoot: "/s/ci"})
		_ = ci.Prepare(wsName, strings.NewReader(""))
		if _, err := ci.Run(context.Background(), wsName, []string{"bun", "test"}); err == nil || len(r.Calls) != 0 {
			t.Errorf("%s: error = %v, calls %d", name, err, len(r.Calls))
		}
	}
}
