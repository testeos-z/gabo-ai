package orcacli

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestLocalListParsesTerminals(t *testing.T) {
	t.Parallel()
	for name, doc := range map[string]string{
		"object": `{"terminals":[{"handle":"term_1","title":"gbai-coder"},{"handle":"term_2","title":"other"}]}`,
		"array":  `[{"handle":"term_1","title":"gbai-coder"},{"handle":"term_2","title":"other"}]`,
		"nested": `{"result":{"terminals":[{"id":"term_1","title":"gbai-coder"},{"id":"term_2","name":"other"}]}}`,
	} {
		r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(doc)}}
		got, err := NewLocal(r, "orca-ide").List(context.Background())
		if err != nil {
			t.Fatalf("%s: List() error = %v", name, err)
		}
		want := []kernel.Terminal{{Handle: "term_1", Title: "gbai-coder"}, {Handle: "term_2", Title: "other"}}
		if !slices.Equal(got, want) {
			t.Fatalf("%s: terminals = %+v", name, got)
		}
		if argv := append([]string{r.Calls[0].Command.Name}, r.Calls[0].Command.Args...); !slices.Equal(argv, []string{"orca-ide", "terminal", "list", "--json"}) {
			t.Fatalf("%s: argv = %v", name, argv)
		}
	}
}

func TestLocalCreateAndClose(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"handle":"term_9"}` + "\n")}}
	c := NewLocal(r, "orca")
	out, err := c.Create(context.Background(), "gbai-coder", "env GBAI_MODE=local claude")
	if err != nil || out != `{"handle":"term_9"}` {
		t.Fatalf("Create() = %q, %v", out, err)
	}
	if err := c.Close(context.Background(), "term_9"); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	want := [][]string{
		{"terminal", "create", "--title", "gbai-coder", "--command", "env GBAI_MODE=local claude", "--json"},
		{"terminal", "close", "--terminal", "term_9", "--tab", "--json"},
	}
	for i, w := range want {
		if got := r.Calls[i].Command.Args; !slices.Equal(got, w) {
			t.Fatalf("call %d argv = %v", i, got)
		}
	}
}

func TestRemoteRunsQuotedArgvAsUser(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{Results: []kernel.Result{{Stdout: []byte(`[]`)}, {}}}
	c := NewRemote(sh, "orca")
	if _, err := c.List(context.Background()); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := c.Create(context.Background(), "gbai-coder", "env GBAI_MODE='cloud' claude \"$(cat 'p')\""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if s := sh.Calls[0].Script; s.User != "orca" || s.Body != "exec 'orca' 'terminal' 'list' '--json'\n" {
		t.Fatalf("script = %+v", s)
	}
	if body := sh.Calls[1].Script.Body; !strings.Contains(body, `'env GBAI_MODE='\''cloud'\'' claude "$(cat '\''p'\'')"'`) {
		t.Fatalf("create body not quoted as one argument: %s", body)
	}
}

func TestNonZeroExitIsError(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{Results: []kernel.Result{{ExitCode: 3, Stderr: []byte("orca not running")}}}
	if _, err := NewRemote(sh, "orca").List(context.Background()); err == nil || !strings.Contains(err.Error(), "orca not running") {
		t.Fatalf("List() error = %v", err)
	}
}
