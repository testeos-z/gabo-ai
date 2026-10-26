package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

type fakeLocal map[string]bool

func (f fakeLocal) LookPath(name string) (string, error) {
	if f[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

var settings = Settings{
	LocalOrca:  "orca-ide",
	Remote:     true,
	Users:      []User{{Name: "orca"}},
	OrcaUser:   "orca",
	SecretsDir: "/data/home/secrets",
}

var allTools = fakeLocal{"railway": true, "gh": true, "orca-ide": true, "go": true}

func byName(r Report) map[string]Check {
	m := map[string]Check{}
	for _, c := range r.Checks {
		m[c.Name] = c
	}
	return m
}

func TestDoctorAllPass(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{}
	rep := NewUseCase(allTools, &kerneltest.Runner{}, sh, settings).Run(context.Background())
	if rep.Failed() {
		t.Fatalf("report = %+v", rep)
	}
	if len(rep.Checks) != 5+4 {
		t.Fatalf("checks = %d: %+v", len(rep.Checks), rep.Checks)
	}
	for _, s := range sh.Calls {
		if s.Script.User != kernel.RootUser {
			t.Fatalf("remote check not run as root: %+v", s.Script)
		}
		for _, mutating := range []string{"rm ", "chmod", "chown", "useradd", "> /", "tee "} {
			if strings.Contains(s.Script.Body, mutating) {
				t.Fatalf("doctor script mutates (%q): %s", mutating, s.Script.Body)
			}
		}
	}
}

func TestDoctorReportsEachFailureIndependently(t *testing.T) {
	t.Parallel()
	results := make([]kernel.Result, 4)
	results[0] = kernel.Result{ExitCode: 1, Stdout: []byte("user orca does not exist\n")}
	results[2] = kernel.Result{ExitCode: 1, Stdout: []byte("visible: GH_TOKEN CLAUDE_CODE_OAUTH_TOKEN\n")}
	sh := &kerneltest.RemoteShell{Results: results}
	runner := &kerneltest.Runner{Result: kernel.Result{ExitCode: 1}}
	rep := NewUseCase(fakeLocal{"gh": true}, runner, sh, settings).Run(context.Background())
	m := byName(rep)
	expect := map[string]Status{
		"local: railway": StatusFail, "local: gh": StatusPass, "local: orca-ide": StatusWarn, "local: go": StatusWarn,
		"local: gh auth status": StatusFail, "remote: user orca": StatusFail, "remote: legacy reviewer sudo removed": StatusPass,
		"remote: service env tokens": StatusFail, "remote: secrets dir mode": StatusPass,
	}
	for name, want := range expect {
		if got := m[name]; got.Status != want {
			t.Errorf("%s = %+v, want %s", name, got, want)
		}
		if m[name].Status != StatusPass && m[name].Fix == "" {
			t.Errorf("%s has no fix hint", name)
		}
	}
	if d := m["remote: user orca"].Detail; d != "user orca does not exist" {
		t.Errorf("detail = %q", d)
	}
	if !rep.Failed() {
		t.Fatal("Failed() = false")
	}
}

func TestDoctorRemoteUnreachableFailsRemoteChecksOnly(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{Err: errors.New("railway: not logged in")}
	rep := NewUseCase(allTools, &kerneltest.Runner{}, sh, settings).Run(context.Background())
	for _, c := range rep.Checks {
		remote := strings.HasPrefix(c.Name, "remote:")
		if remote && (c.Status != StatusFail || !strings.Contains(c.Detail, "not logged in")) {
			t.Errorf("%+v", c)
		}
		if !remote && c.Status != StatusPass {
			t.Errorf("%+v", c)
		}
	}
}

func TestDoctorScripts(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{}
	NewUseCase(allTools, &kerneltest.Runner{}, sh, settings).Run(context.Background())
	bodies := map[int]string{}
	for i, c := range sh.Calls {
		bodies[i] = c.Script.Body
	}
	if !strings.Contains(bodies[0], "id -u 'orca'") || strings.Contains(bodies[0], "expected") {
		t.Errorf("user script = %s", bodies[0])
	}
	for _, p := range []string{"/etc/sudoers.d/gbai-reviewer", "/usr/local/libexec/gbai/review-run", "/usr/local/libexec/gbai/review-ci"} {
		if !strings.Contains(bodies[1], p) {
			t.Errorf("legacy script lacks %s: %s", p, bodies[1])
		}
	}
	tok := bodies[2]
	if !strings.Contains(tok, "pgrep -u 'orca'") || strings.Index(tok, "cut -d= -f1") > strings.Index(tok, "grep -x") {
		t.Errorf("token script must strip values before matching names: %s", tok)
	}
	if !strings.Contains(bodies[3], "stat -c %a '/data/home/secrets'") || !strings.Contains(bodies[3], "|| exit 0") {
		t.Errorf("secrets dir script = %s", bodies[3])
	}
	for _, b := range bodies {
		if strings.Contains(b, "reviewer-ci") || strings.Contains(b, "visudo") {
			t.Errorf("single-user doctor still checks the three-user model: %s", b)
		}
	}
}

func TestDoctorLocalOnly(t *testing.T) {
	t.Parallel()
	s := settings
	s.Remote = false
	sh := &kerneltest.RemoteShell{}
	rep := NewUseCase(allTools, &kerneltest.Runner{}, sh, s).Run(context.Background())
	if len(sh.Calls) != 0 || len(rep.Checks) != 5 {
		t.Fatalf("calls = %d checks = %d", len(sh.Calls), len(rep.Checks))
	}
}
