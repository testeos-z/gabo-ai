package initcmd

import (
	"context"
	"errors"
	"maps"
	"strings"
	"testing"
)

type fakeUpdater struct {
	lines   []string
	updated bool
	err     error
	calls   [][2]string
}

func (f *fakeUpdater) Update(_ context.Context, want, have string) ([]string, bool, error) {
	f.calls = append(f.calls, [2]string{want, have})
	return f.lines, f.updated, f.err
}

// An accepted update re-executes init before anything else is written.
func TestNewerStampUpdatedStopsBeforeWriting(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{config: cfgYAML, stamp: "1.10.0\n"})
	before := maps.Clone(s.Files)
	u := &fakeUpdater{updated: true, err: errors.New("restart failed")}
	i := newInit(s, "v1.2.3")
	i.Updater = u
	_, err := i.Run(context.Background())
	if err == nil || err.Error() != "restart failed" {
		t.Fatalf("err = %v", err)
	}
	if len(u.calls) != 1 || u.calls[0] != [2]string{"1.10.0", "1.2.3"} {
		t.Fatalf("calls = %v", u.calls)
	}
	if len(s.Writes) != 0 || !maps.Equal(s.Files, before) {
		t.Fatalf("writes %v", s.Writes)
	}
}

// A refused or failed update keeps today's behavior: reviewer install, the
// project unchanged and the upgrade command with exit 1, after the reason.
func TestNewerStampNotUpdatedPrintsReasonAndCommand(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{config: cfgYAML, stamp: "1.10.0\n", setup: "newer"})
	u := &fakeUpdater{lines: []string{"not updated"}}
	i := newInit(s, "1.2.3")
	i.Updater = u
	out, err := i.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v1.10.0") {
		t.Fatalf("err = %v", err)
	}
	if got := joined(out); !strings.HasPrefix(got, "not updated\nreviewer skill installed") {
		t.Fatalf("output = %q", got)
	}
	if s.Files[stamp] != "1.10.0\n" || s.Files[setup] != "newer" || s.Files[config] != cfgYAML {
		t.Fatalf("project changed: %v", s.Files)
	}
	assertReviewer(t, s, "1.2.3")
}

func TestUpdaterOnlyForNewerStamp(t *testing.T) {
	t.Parallel()
	for _, files := range []map[string]string{
		{},
		{config: cfgYAML},
		{config: cfgYAML, stamp: "1.2.3\n"},
		{config: cfgYAML, stamp: "1.0.0\n"},
		{config: cfgYAML, stamp: "garbage\n"},
	} {
		u := &fakeUpdater{updated: true}
		i := newInit(newStore(files), "1.2.3")
		i.Updater = u
		_, _ = i.Run(context.Background())
		if len(u.calls) != 0 {
			t.Fatalf("%v: updater called %v", files, u.calls)
		}
	}
}
