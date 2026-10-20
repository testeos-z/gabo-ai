package secrets

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

const secretValue = "s3cr3t-Value-0123456789"

type fakeReader struct {
	value []byte
	err   error
	calls int
}

func (f *fakeReader) ReadSecret(string) ([]byte, error) {
	f.calls++
	return append([]byte(nil), f.value...), f.err
}

var reviewer = Account{Name: "reviewer", Home: "/data/reviewer"}

func newUC(shell kernel.RemoteShell, r SecretReader) *UseCase {
	return NewUseCase(shell, r, reviewer)
}

func assertNoLeak(t *testing.T, shell *kerneltest.RemoteShell, out Output) {
	t.Helper()
	for _, c := range shell.Calls {
		if strings.Contains(c.Script.Body, secretValue) {
			t.Fatalf("secret leaked into remote script:\n%s", c.Script.Body)
		}
	}
	all := out.Message + strings.Join(out.Scripts, "")
	if strings.Contains(all, secretValue) || strings.Contains(out.Message, strconv.Itoa(len(secretValue))) {
		t.Fatalf("output leaks the secret or its length: %+v", out)
	}
}

func TestSetFileSecretWritesAtomicallyThroughStdin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path string
	}{
		{"reviewer-claude", "/data/reviewer/secrets/claude-oauth-token"},
		{"reviewer-pi", "/data/reviewer/secrets/pi-api-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			shell := &kerneltest.RemoteShell{}
			out, err := newUC(shell, &fakeReader{value: []byte(secretValue)}).Set(context.Background(), Input{Name: tt.name})
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if len(shell.Calls) != 1 {
				t.Fatalf("remote calls = %d, want 1", len(shell.Calls))
			}
			call := shell.Calls[0]
			if call.Script.User != "reviewer" {
				t.Errorf("user = %q, want reviewer", call.Script.User)
			}
			if call.Stdin != secretValue {
				t.Errorf("stdin = %q, want the secret", call.Stdin)
			}
			body := call.Script.Body
			for _, want := range []string{"umask 077", "chmod 700", "mktemp", `cat > "$tmp"`, `chmod 600 "$tmp"`, "mv -f", tt.path, "/data/reviewer/secrets"} {
				if !strings.Contains(body, want) {
					t.Errorf("script missing %q:\n%s", want, body)
				}
			}
			if i, j := strings.Index(body, `cat > "$tmp"`), strings.Index(body, "mv -f"); i < 0 || j < i {
				t.Errorf("mv must come after writing the temp file:\n%s", body)
			}
			if want := tt.name + " stored for reviewer"; out.Message != want {
				t.Errorf("message = %q, want %q", out.Message, want)
			}
			assertNoLeak(t, shell, out)
		})
	}
}

func TestSetGitHubLogsInThenChecksStatus(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	out, err := newUC(shell, &fakeReader{value: []byte(secretValue)}).Set(context.Background(), Input{Name: "reviewer-github"})
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(shell.Calls) != 2 {
		t.Fatalf("remote calls = %d, want 2", len(shell.Calls))
	}
	login, status := shell.Calls[0], shell.Calls[1]
	if !strings.Contains(login.Script.Body, "gh auth login") || !strings.Contains(login.Script.Body, "--with-token") {
		t.Errorf("login script = %q", login.Script.Body)
	}
	if login.Stdin != secretValue || login.Script.User != "reviewer" {
		t.Errorf("login call = %+v", login)
	}
	if !strings.Contains(status.Script.Body, "gh auth status") || status.Stdin != "" || status.Script.User != "reviewer" {
		t.Errorf("status call = %+v", status)
	}
	if out.Message != "reviewer-github stored for reviewer" {
		t.Errorf("message = %q", out.Message)
	}
	assertNoLeak(t, shell, out)
}

func TestSetGitHubFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		results   []kernel.Result
		wantCalls int
	}{
		{"login fails", []kernel.Result{{ExitCode: 1, Stderr: []byte("bad token")}}, 1},
		{"status fails", []kernel.Result{{}, {ExitCode: 1}}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			shell := &kerneltest.RemoteShell{Results: tt.results}
			_, err := newUC(shell, &fakeReader{value: []byte(secretValue)}).Set(context.Background(), Input{Name: "reviewer-github"})
			if err == nil {
				t.Fatal("Set() error = nil, want failure")
			}
			if strings.Contains(err.Error(), secretValue) {
				t.Fatalf("error leaks secret: %v", err)
			}
			if len(shell.Calls) != tt.wantCalls {
				t.Fatalf("remote calls = %d, want %d", len(shell.Calls), tt.wantCalls)
			}
		})
	}
}

func TestSetRedactsSecretFromRemoteErrors(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{Results: []kernel.Result{{ExitCode: 2, Stderr: []byte("echoed " + secretValue)}}}
	_, err := newUC(shell, &fakeReader{value: []byte(secretValue)}).Set(context.Background(), Input{Name: "reviewer-pi"})
	if err == nil || strings.Contains(err.Error(), secretValue) {
		t.Fatalf("Set() error = %v, want redacted failure", err)
	}
}

func TestSetRejectsEmptyValue(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"", "   ", "\n"} {
		shell := &kerneltest.RemoteShell{}
		_, err := newUC(shell, &fakeReader{value: []byte(v)}).Set(context.Background(), Input{Name: "reviewer-pi"})
		if !errors.Is(err, ErrEmptyValue) {
			t.Fatalf("Set(%q) error = %v, want ErrEmptyValue", v, err)
		}
		if len(shell.Calls) != 0 {
			t.Fatalf("remote calls = %d, want 0", len(shell.Calls))
		}
	}
}

func TestSetRejectsUnknownName(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	r := &fakeReader{value: []byte(secretValue)}
	_, err := newUC(shell, r).Set(context.Background(), Input{Name: "github"})
	if !errors.Is(err, ErrUnknownSecret) {
		t.Fatalf("Set() error = %v, want ErrUnknownSecret", err)
	}
	if r.calls != 0 || len(shell.Calls) != 0 {
		t.Fatalf("reader calls = %d, remote calls = %d, want 0", r.calls, len(shell.Calls))
	}
}

func TestSetDryRunPrintsScriptsWithoutReadingOrExecuting(t *testing.T) {
	t.Parallel()
	for _, name := range Names() {
		shell := &kerneltest.RemoteShell{}
		r := &fakeReader{value: []byte(secretValue)}
		out, err := newUC(shell, r).Set(context.Background(), Input{Name: name, DryRun: true})
		if err != nil {
			t.Fatalf("Set(%s) error = %v", name, err)
		}
		if r.calls != 0 || len(shell.Calls) != 0 {
			t.Fatalf("%s: reader calls = %d, remote calls = %d, want 0", name, r.calls, len(shell.Calls))
		}
		if len(out.Scripts) == 0 || out.Message != "" {
			t.Fatalf("%s: output = %+v, want scripts only", name, out)
		}
	}
}

func TestSetReaderErrorStopsBeforeRemote(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	_, err := newUC(shell, &fakeReader{err: errors.New("no tty")}).Set(context.Background(), Input{Name: "reviewer-pi"})
	if err == nil || len(shell.Calls) != 0 {
		t.Fatalf("Set() error = %v, remote calls = %d", err, len(shell.Calls))
	}
}
