package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// fakeTools writes executable fakes into a temporary directory and makes it
// the only PATH entry, so `pi` and `gentle-shell` resolve to the fakes and
// never to the real ones (intent 0016 S1, S2). Every call of a fake pi with
// arguments other than `list` is logged to called in that directory.
func fakeTools(t *testing.T, tools map[string]string) (dir, called string) {
	t.Helper()
	dir = t.TempDir()
	called = filepath.Join(dir, "called")
	for name, body := range tools {
		body = strings.ReplaceAll(body, "CALLED", called)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	return dir, called
}

// piListing is a fake pi that prints out for `pi list` and nothing else.
func piListing(out string) string {
	return `if [ "$1" = list ] && [ $# -eq 1 ]; then printf '%s\n' '` + out + `'; exit 0; fi
echo "$*" >> CALLED
exit 1`
}

// S1, S2 through `gabo init`: a fake pi on PATH decides the Pi refusal; the
// fake logs any call other than `pi list` and init writes no file of its own
// for the check.
func TestInitPiRefusalFollowsThePackageCheck(t *testing.T) {
	cfg := "runtimes:\n  coder: pi\n  reviewer: pi\n"
	cases := []struct {
		name    string
		tools   map[string]string
		refusal bool
	}{
		{"npm package", map[string]string{"pi": piListing("npm:gentle-pi")}, false},
		{"npm package with version", map[string]string{"pi": piListing("npm:gentle-pi@1.2.3")}, false},
		{"git source", map[string]string{"pi": piListing("git:github.com/someone/gentle-pi")}, false},
		{"gentle-shell only", map[string]string{"pi": piListing(""), "gentle-shell": "exit 0"}, false},
		{"other package", map[string]string{"pi": piListing("npm:@gabo-ai/pi")}, true},
		{"pi not on PATH", map[string]string{}, true},
		{"pi list exits 1", map[string]string{"pi": `if [ "$1" = list ]; then exit 1; fi
exit 1`}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, "gabo.yaml"), []byte(cfg), 0o644); err != nil {
				t.Fatal(err)
			}
			_, called := fakeTools(t, c.tools)
			out, err := runInit(t, project, "1.2.3")
			if err != nil {
				t.Fatalf("init: %v\n%s", err, out)
			}
			const line = "gabo refuses to build on Pi without a subagent package:"
			if got := strings.Contains(out, line) || strings.Contains(out, engineconfig.SubagentRefusal) || strings.Contains(out, "install a subagent package (see the refusal above)"); got != c.refusal {
				t.Errorf("refusal printed = %v, want %v:\n%s", got, c.refusal, out)
			}
			if !strings.Contains(out, "trust the project in Pi") {
				t.Errorf("trust step missing:\n%s", out)
			}
			if _, err := os.Stat(called); err == nil {
				b, _ := os.ReadFile(called)
				t.Errorf("pi was called with something other than list: %s", b)
			}
		})
	}
}

// PD3: the adapter gives up on a `pi list` that outlives its context and
// reports it as not listed.
func TestPiProbeGivesUpAtTheDeadline(t *testing.T) {
	// The real sleep is resolved before PATH is replaced: a fake that cannot
	// start would exit at once and pass without waiting for the deadline.
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary to hold pi list open")
	}
	fakeTools(t, map[string]string{"pi": "exec '" + sleep + "' 30"})
	const deadline = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	start := time.Now()
	out, ok := (piProbe{}).PiList(ctx)
	elapsed := time.Since(start)
	if ok || len(out) != 0 {
		t.Fatalf("PiList = %q, %v; want no output and not listed", out, ok)
	}
	if elapsed < deadline/2 || elapsed > 5*time.Second {
		t.Fatalf("PiList returned after %v; want it to end at the %v deadline", elapsed, deadline)
	}
}
