package provision

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

type fakeBinary struct {
	data  []byte
	err   error
	calls []string
}

func (f *fakeBinary) ReadBinary(path string) ([]byte, error) {
	f.calls = append(f.calls, path)
	return f.data, f.err
}

func settings() Settings {
	return Settings{
		Orca:      Account{Name: "orca", Home: "/data/home", Path: []string{"/data/home/.local/bin"}},
		Config:    []byte("repo:\n  name: o/r\n"),
		SkillsTar: []byte("SKILLS-TAR"),
	}
}

// linuxELF returns the running test binary, which is a linux ELF on the CI host.
func linuxELF(t *testing.T) []byte {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func script(t *testing.T, shell *kerneltest.RemoteShell, name string) kerneltest.ShellCall {
	t.Helper()
	for _, c := range shell.Calls {
		if strings.HasPrefix(c.Script.Body, "# step: "+name+"\n") {
			return c
		}
	}
	t.Fatalf("no %q step executed", name)
	return kerneltest.ShellCall{}
}

func run(t *testing.T, shell *kerneltest.RemoteShell, bin *fakeBinary, in Input) Output {
	t.Helper()
	out, err := NewUseCase(shell, bin, settings()).Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return out
}

func TestRunExecutesEveryStepAsRootInOrder(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	out := run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{BinaryPath: "/tmp/gbai"})
	want := []string{"packages", "user", "config", "binary", "legacy", "skills"}
	if len(shell.Calls) != len(want) {
		t.Fatalf("calls = %d, want %d", len(shell.Calls), len(want))
	}
	for i, name := range want {
		c := shell.Calls[i]
		if c.Script.User != kernel.RootUser || !strings.HasPrefix(c.Script.Body, "# step: "+name+"\n") {
			t.Errorf("call %d = user %q body %.40q, want root %s", i, c.Script.User, c.Script.Body, name)
		}
		if out.Lines[i] != "ok "+name {
			t.Errorf("line %d = %q", i, out.Lines[i])
		}
	}
}

func TestPackagesInstallOnlyWhenMissing(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "packages").Script.Body
	for _, want := range []string{"command -v git", "apt-get install", `if [ -n "$missing" ]`} {
		if !strings.Contains(body, want) {
			t.Errorf("packages script missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "sudo") {
		t.Errorf("single-user mode needs no sudo:\n%s", body)
	}
}

func TestUserStepOnlyChecksTheOrcaUser(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "user").Script.Body
	if !strings.Contains(body, "id -u 'orca' >/dev/null 2>&1 ||") {
		t.Errorf("user script does not check orca:\n%s", body)
	}
	for _, banned := range []string{"useradd", "groupadd", "chown", "chmod", "reviewer"} {
		if strings.Contains(body, banned) {
			t.Errorf("user script contains %q:\n%s", banned, body)
		}
	}
}

func TestBinaryIsStreamedOverStdinAndInstalledAtomically(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	bin := &fakeBinary{data: linuxELF(t)}
	run(t, shell, bin, Input{BinaryPath: "/tmp/gbai"})
	c := script(t, shell, "binary")
	if c.Stdin != string(bin.data) {
		t.Fatalf("binary stdin length = %d, want %d", len(c.Stdin), len(bin.data))
	}
	if len(bin.calls) != 1 || bin.calls[0] != "/tmp/gbai" {
		t.Fatalf("ReadBinary calls = %v", bin.calls)
	}
	for _, want := range []string{`cat > "$tmp"`, "chown root:root", "chmod 755", `mv -f "$tmp" '/usr/local/bin/gbai'`} {
		if !strings.Contains(c.Script.Body, want) {
			t.Errorf("binary script missing %q:\n%s", want, c.Script.Body)
		}
	}
	for _, c := range shell.Calls {
		streams := c.Script.Body == script(t, shell, "binary").Script.Body || c.Script.Body == script(t, shell, "skills").Script.Body
		if !streams && c.Stdin != "" {
			t.Errorf("only the binary and skills steps may read stdin")
		}
	}
}

func TestConfigIsInstalledReadOnlyForUsers(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "config").Script.Body
	enc := base64.StdEncoding.EncodeToString(settings().Config)
	for _, want := range []string{enc, "'/etc/gbai/gbai.yaml'", "chmod 644"} {
		if !strings.Contains(body, want) {
			t.Errorf("config script missing %q:\n%s", want, body)
		}
	}
}

func TestLegacyStepRemovesTheSudoPathOfTheThreeUserModel(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "legacy").Script.Body
	for _, want := range []string{
		"rm -f '/etc/sudoers.d/gbai-reviewer' '/usr/local/libexec/gbai/review-run' '/usr/local/libexec/gbai/review-ci'",
		"rmdir '/usr/local/libexec/gbai' 2>/dev/null || true",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("legacy script missing %q:\n%s", want, body)
		}
	}
	for _, banned := range []string{"userdel", "rm -rf", "visudo", "cat >"} {
		if strings.Contains(body, banned) {
			t.Errorf("legacy script contains %q:\n%s", banned, body)
		}
	}
}

func TestPlanHasNoReviewerUsersSudoersOrWrappers(t *testing.T) {
	t.Parallel()
	out := run(t, &kerneltest.RemoteShell{}, &fakeBinary{}, Input{DryRun: true})
	plan := strings.Join(out.Lines, "\n")
	for _, banned := range []string{"useradd", "reviewer-ci", "NOPASSWD", "sudo -n", "install_wrapper", "/data/reviewer"} {
		if strings.Contains(plan, banned) {
			t.Errorf("plan contains %q:\n%s", banned, plan)
		}
	}
}

func TestDryRunPrintsPlanWithoutReadingOrExecuting(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	bin := &fakeBinary{data: []byte("not elf")}
	out := run(t, shell, bin, Input{DryRun: true})
	if len(shell.Calls) != 0 || len(bin.calls) != 0 {
		t.Fatalf("dry run touched remote (%d) or binary (%d)", len(shell.Calls), len(bin.calls))
	}
	text := strings.Join(out.Lines, "\n")
	for _, name := range []string{"packages", "user", "config", "binary", "legacy", "skills"} {
		if !strings.Contains(text, "# step: "+name) {
			t.Errorf("plan missing step %s", name)
		}
	}
}

func TestRejectsNonELFBinaryBeforeRemote(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"text":   []byte("#!/bin/sh\necho hi\n"),
		"macho":  {0xcf, 0xfa, 0xed, 0xfe, 7, 0, 0, 1},
		"empty":  nil,
		"broken": []byte("\x7fELF\x02\x01\x01"),
	} {
		shell := &kerneltest.RemoteShell{}
		_, err := NewUseCase(shell, &fakeBinary{data: data}, settings()).Run(context.Background(), Input{})
		if !errors.Is(err, ErrNotLinuxBinary) {
			t.Errorf("%s: error = %v, want ErrNotLinuxBinary", name, err)
		}
		if len(shell.Calls) != 0 {
			t.Errorf("%s: remote calls = %d, want 0", name, len(shell.Calls))
		}
	}
}

func TestFailingStepStopsWithStderr(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{Results: []kernel.Result{{}, {ExitCode: 1, Stderr: []byte("orca user is missing")}}}
	_, err := NewUseCase(shell, &fakeBinary{data: linuxELF(t)}, settings()).Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "step user") || !strings.Contains(err.Error(), "orca user is missing") {
		t.Fatalf("error = %v", err)
	}
	if len(shell.Calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(shell.Calls))
	}
}

func TestQuotesValuesFromConfig(t *testing.T) {
	t.Parallel()
	s := settings()
	s.Orca.Home = "/data/it's"
	shell := &kerneltest.RemoteShell{}
	if _, err := NewUseCase(shell, &fakeBinary{data: linuxELF(t)}, s).Run(context.Background(), Input{}); err != nil {
		t.Fatal(err)
	}
	if body := script(t, shell, "skills").Script.Body; !strings.Contains(body, `'/data/it'\''s/.claude/skills'`) {
		t.Fatalf("home not shell-quoted:\n%s", body)
	}
}

func TestSkillsAreStreamedAndCopiedAsOrca(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{BinaryPath: "/tmp/gbai"})
	call := script(t, shell, "skills")
	if call.Stdin != "SKILLS-TAR" {
		t.Fatalf("stdin = %q", call.Stdin)
	}
	body := call.Script.Body
	for _, want := range []string{
		"tmp=$(mktemp -d)", "tar -x --no-same-owner --no-same-permissions -C \"$tmp\" -f -",
		"setpriv --reuid='orca' --regid=\"$(id -g 'orca')\" --init-groups",
		"'/data/home/.claude/skills' '/data/home/.pi/agent/skills'",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("skills script missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "chown -R") {
		t.Errorf("root must not chown into the orca home:\n%s", body)
	}
	for _, c := range shell.Calls {
		if c.Stdin == "SKILLS-TAR" && !strings.HasPrefix(c.Script.Body, "# step: skills") {
			t.Errorf("skills tar sent to another step")
		}
	}
}
