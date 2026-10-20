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
		Orca:       Account{Name: "orca", Home: "/data/home", Path: []string{"/data/home/.local/bin"}},
		Reviewer:   Account{Name: "reviewer", Home: "/data/reviewer", UID: 2000},
		ReviewerCI: Account{Name: "reviewer-ci", Home: "/data/reviewer-ci", UID: 2001, Path: []string{"/data/reviewer-ci/.local/bin"}},
		Config:     []byte("repo:\n  name: o/r\n"),
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
	want := []string{"packages", "users", "config", "binary", "wrappers", "sudoers"}
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
	for _, want := range []string{"command -v sudo", "command -v git", "apt-get install", `if [ -n "$missing" ]`} {
		if !strings.Contains(body, want) {
			t.Errorf("packages script missing %q:\n%s", want, body)
		}
	}
}

func TestUsersAreIdempotentAndNeverModifyOrca(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "users").Script.Body
	for _, want := range []string{
		"id -u 'reviewer' >/dev/null 2>&1 || useradd --uid '2000' --gid 'reviewer' --home-dir '/data/reviewer'",
		"id -u 'reviewer-ci' >/dev/null 2>&1 || useradd --uid '2001' --gid 'reviewer-ci' --home-dir '/data/reviewer-ci'",
		"getent group 'reviewer' >/dev/null || groupadd --gid '2000' 'reviewer'",
		"chmod 700 '/data/reviewer'",
		"chmod 700 '/data/reviewer-ci'",
		"install -d -o 'reviewer' -g 'reviewer' -m 700 '/data/reviewer/work'",
		"id -u 'orca' >/dev/null 2>&1 ||",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("users script missing %q:\n%s", want, body)
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "'orca'") && !strings.HasPrefix(line, "id -u 'orca'") {
			t.Errorf("orca touched beyond reading: %q", line)
		}
	}
	if strings.Contains(body, "/data/reviewer/secrets") {
		t.Errorf("users script must not create secret dirs:\n%s", body)
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
		if c.Script.Body != script(t, shell, "binary").Script.Body && c.Stdin != "" {
			t.Errorf("only the binary step may read stdin")
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

func TestWrappersValidateArgumentsAndScrubEnvironment(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "wrappers").Script.Body
	for _, want := range []string{
		"/usr/local/libexec/gbai/review-run",
		"/usr/local/libexec/gbai/review-ci",
		`[ "$#" -eq 3 ] && [ "$1" = head ]`,
		`*[!0-9a-f]*`,
		`exec env -i HOME='/data/reviewer' USER='reviewer'`,
		`/usr/local/bin/gbai --config /etc/gbai/gbai.yaml review run head "$2" "$3"`,
		`exec env -i HOME='/data/reviewer-ci' USER='reviewer-ci'`,
		"PATH='/data/reviewer-ci/.local/bin:/usr/local/sbin:",
		`/usr/local/bin/gbai --config /etc/gbai/gbai.yaml review ci "$@"`,
		"chown root:root",
		"chmod 755",
		`sh -n "$tmp"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("wrappers script missing %q:\n%s", want, body)
		}
	}
}

func TestSudoersIsValidatedBeforeMove(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	run(t, shell, &fakeBinary{data: linuxELF(t)}, Input{})
	body := script(t, shell, "sudoers").Script.Body
	for _, want := range []string{
		"\norca ALL=(reviewer) NOPASSWD: /usr/local/libexec/gbai/review-run\n",
		"\nreviewer ALL=(reviewer-ci) NOPASSWD: /usr/local/libexec/gbai/review-ci\n",
		"chmod 440",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sudoers script missing %q:\n%s", want, body)
		}
	}
	check, move := strings.Index(body, `visudo -cf "$tmp"`), strings.Index(body, `mv -f "$tmp" '/etc/sudoers.d/gbai-reviewer'`)
	if check < 0 || move < 0 || move < check {
		t.Fatalf("visudo -cf must run before the move:\n%s", body)
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "/etc/sudoers") && !strings.Contains(line, "/etc/sudoers.d/") &&
			!strings.HasPrefix(strings.TrimSpace(line), "grep -Eq") {
			t.Errorf("line touches other sudoers files: %q", line)
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
	for _, name := range []string{"packages", "users", "config", "binary", "wrappers", "sudoers"} {
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
	shell := &kerneltest.RemoteShell{Results: []kernel.Result{{}, {ExitCode: 1, Stderr: []byte("useradd: boom")}}}
	_, err := NewUseCase(shell, &fakeBinary{data: linuxELF(t)}, settings()).Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "users") || !strings.Contains(err.Error(), "useradd: boom") {
		t.Fatalf("error = %v", err)
	}
	if len(shell.Calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(shell.Calls))
	}
}

func TestQuotesValuesFromConfig(t *testing.T) {
	t.Parallel()
	s := settings()
	s.Reviewer.Home = "/data/it's"
	shell := &kerneltest.RemoteShell{}
	if _, err := NewUseCase(shell, &fakeBinary{data: linuxELF(t)}, s).Run(context.Background(), Input{}); err != nil {
		t.Fatal(err)
	}
	if body := script(t, shell, "users").Script.Body; !strings.Contains(body, `'/data/it'\''s'`) {
		t.Fatalf("home not shell-quoted:\n%s", body)
	}
}
