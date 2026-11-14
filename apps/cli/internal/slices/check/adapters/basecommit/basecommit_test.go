package basecommit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/execrunner"
)

// gitIn runs git in dir with a fixed identity and fails the test on error.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.test",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// fixture makes a repository whose main branch tracks foo_test.go at its base
// commit, and whose feat branch adds bar_test.go after that base.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("foo_test.go")
	gitIn(t, dir, "add", "foo_test.go")
	gitIn(t, dir, "commit", "-q", "-m", "base")
	gitIn(t, dir, "switch", "-q", "-c", "feat")
	write("bar_test.go")
	gitIn(t, dir, "add", "bar_test.go")
	gitIn(t, dir, "commit", "-q", "-m", "feat")
	return dir
}

// I0028 D6: a path is tracked when the merge base of HEAD and the ref holds it;
// a file the branch adds after that base is not.
func TestTrackedAtMergeBase(t *testing.T) {
	t.Parallel()
	g := Git{Runner: execrunner.New(), Dir: fixture(t)}
	for _, tt := range []struct {
		rel  string
		want bool
	}{
		{"foo_test.go", true},
		{"bar_test.go", false},
		{"missing_test.go", false},
	} {
		got, err := g.Tracked("main", tt.rel)
		if err != nil || got != tt.want {
			t.Fatalf("Tracked(main, %s) = %v, %v; want %v, nil", tt.rel, got, err, tt.want)
		}
	}
}

// I0028 D6: a ref that does not resolve is an error, never a tracked file.
func TestTrackedUnknownRefIsError(t *testing.T) {
	t.Parallel()
	g := Git{Runner: execrunner.New(), Dir: fixture(t)}
	if got, err := g.Tracked("origin/nope", "foo_test.go"); err == nil || got {
		t.Fatalf("Tracked(origin/nope) = %v, %v; want false and an error", got, err)
	}
}

// A ref that gh or git would read as an option is refused before any command runs.
func TestTrackedRefStartingWithDashIsError(t *testing.T) {
	t.Parallel()
	g := Git{Runner: execrunner.New(), Dir: fixture(t)}
	if _, err := g.Tracked("--output=/tmp/x", "foo_test.go"); err == nil {
		t.Fatal("Tracked with a ref starting with - returned no error")
	}
}
