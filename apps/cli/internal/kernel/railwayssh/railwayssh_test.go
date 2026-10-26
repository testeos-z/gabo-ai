package railwayssh

import (
	"context"
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

var b64 = regexp.MustCompile(`printf %s ([A-Za-z0-9+/=]+) \| base64 -d`)

// decodeAll expands every base64 layer found in s, outermost first.
func decodeAll(t *testing.T, s string) []string {
	t.Helper()
	layers := []string{s}
	for {
		m := b64.FindStringSubmatch(layers[len(layers)-1])
		if m == nil {
			return layers
		}
		raw, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			t.Fatalf("decode base64: %v", err)
		}
		layers = append(layers, string(raw))
	}
}

func newShell(r kernel.Runner) *Shell {
	return New(r,
		Target{ProjectID: "proj", EnvironmentID: "env", ServiceID: "svc"},
		map[string]User{
			"orca":     {Home: "/data/home", ExtraPath: []string{"/data/home/.local/bin"}},
			"reviewer": {Home: "/data/reviewer"},
		},
	)
}

func TestExecBuildsRailwaySSHInvocationForUser(t *testing.T) {
	t.Parallel()
	fake := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("ok"), ExitCode: 0}}
	sh := newShell(fake)

	res, err := sh.Exec(context.Background(), kernel.Script{User: "orca", Body: "echo hello"})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if string(res.Stdout) != "ok" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(fake.Calls))
	}
	cmd := fake.Calls[0].Command
	if cmd.Name != "railway" {
		t.Fatalf("name = %q, want railway", cmd.Name)
	}
	wantPrefix := []string{"ssh", "--project", "proj", "--environment", "env", "--service", "svc", "--"}
	if len(cmd.Args) != len(wantPrefix)+1 {
		t.Fatalf("args = %q", cmd.Args)
	}
	for i, w := range wantPrefix {
		if cmd.Args[i] != w {
			t.Fatalf("args[%d] = %q, want %q (args %q)", i, cmd.Args[i], w, cmd.Args)
		}
	}
	layers := decodeAll(t, cmd.Args[len(cmd.Args)-1])
	if len(layers) != 3 {
		t.Fatalf("got %d script layers, want 3: %q", len(layers), layers)
	}
	root, user := layers[1], layers[2]
	for _, want := range []string{
		"unset GH_TOKEN",
		"setpriv --reuid=orca --regid=orca --init-groups",
		"env -u GH_TOKEN",
		"HOME=/data/home",
		"PATH=/data/home/.local/bin:",
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root wrapper missing %q:\n%s", want, root)
		}
	}
	if user != "echo hello" {
		t.Errorf("user script = %q, want %q", user, "echo hello")
	}
}

func TestExecAsRootSkipsSetpriv(t *testing.T) {
	t.Parallel()
	fake := &kerneltest.Runner{}
	if _, err := newShell(fake).Exec(context.Background(), kernel.Script{User: kernel.RootUser, Body: "id"}); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	layers := decodeAll(t, fake.Calls[0].Command.Args[len(fake.Calls[0].Command.Args)-1])
	root := layers[1]
	if strings.Contains(root, "setpriv") {
		t.Errorf("root script must not drop privileges:\n%s", root)
	}
	if !strings.Contains(root, "unset GH_TOKEN") || !strings.Contains(root, "HOME=/root") {
		t.Errorf("root script missing env hygiene:\n%s", root)
	}
	if layers[len(layers)-1] != "id" {
		t.Errorf("body = %q", layers[len(layers)-1])
	}
}

func TestExecStreamsStdinAndNeverPutsItInArgv(t *testing.T) {
	t.Parallel()
	const secret = "ghp_SuperSecretValue123"
	fake := &kerneltest.Runner{}
	_, err := newShell(fake).Exec(context.Background(), kernel.Script{
		User:  "reviewer",
		Body:  "cat > /dev/null",
		Stdin: strings.NewReader(secret),
	})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	call := fake.Calls[0]
	if call.Stdin != secret {
		t.Fatalf("stdin = %q, want secret streamed through", call.Stdin)
	}
	for _, a := range call.Command.Args {
		for _, layer := range decodeAll(t, a) {
			if strings.Contains(layer, secret) {
				t.Fatalf("secret leaked into argv layer %q", layer)
			}
		}
	}
	for _, e := range call.Command.Env {
		if strings.Contains(e, secret) {
			t.Fatalf("secret leaked into env %q", e)
		}
	}
}

func TestExecUnknownUserFails(t *testing.T) {
	t.Parallel()
	fake := &kerneltest.Runner{}
	if _, err := newShell(fake).Exec(context.Background(), kernel.Script{User: "mallory", Body: "id"}); err == nil {
		t.Fatal("Exec() error = nil, want unknown user error")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("runner called %d times, want 0", len(fake.Calls))
	}
}

func TestExecRejectsUnsafeUserEnvironment(t *testing.T) {
	t.Parallel()
	fake := &kerneltest.Runner{}
	sh := New(fake, Target{}, map[string]User{"evil": {Home: "/data/x; rm -rf /"}})
	if _, err := sh.Exec(context.Background(), kernel.Script{User: "evil", Body: "id"}); err == nil {
		t.Fatal("Exec() error = nil, want unsafe home error")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("runner called %d times, want 0", len(fake.Calls))
	}
}

// An empty target id would let `railway ssh` fall back to whatever project is
// linked in the working directory, so Exec refuses before running anything.
func TestExecRejectsAnEmptyTargetID(t *testing.T) {
	t.Parallel()
	for name, target := range map[string]Target{
		"project":     {EnvironmentID: "env", ServiceID: "svc"},
		"environment": {ProjectID: "proj", ServiceID: "svc"},
		"service":     {ProjectID: "proj", EnvironmentID: "env"},
	} {
		r := &kerneltest.Runner{}
		_, err := New(r, target, nil).Exec(context.Background(), kernel.Script{User: kernel.RootUser, Body: "true"})
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: Exec() error = %v, want it to name the empty %s id", name, err, name)
		}
		if len(r.Calls) != 0 {
			t.Errorf("%s: railway ran with an empty id: %+v", name, r.Calls)
		}
	}
}
