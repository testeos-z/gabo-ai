package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestCommandTextAndJSON(t *testing.T) {
	t.Parallel()
	factory := func(context.Context, string) (*UseCase, error) {
		gh := &fakeGitHub{prs: []PullRequest{{Number: 7, HeadSHA: "aaa", Labels: []string{"mode:local"}}}, issuesErr: errors.New("gh down")}
		return NewUseCase(&kerneltest.RemoteShell{}, &kerneltest.Terminals{}, gh, settings), nil
	}
	out := &bytes.Buffer{}
	cmd := NewCommand(factory)
	cmd.SetOut(out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{"server: ok (reachable)", "#7 [local] review=none", "needs-user: unknown (gh down)"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("text missing %q:\n%s", want, out.String())
		}
	}

	out = &bytes.Buffer{}
	cmd = NewCommand(factory)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--json) error = %v", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(out.Bytes(), &snap); err != nil || snap.NeedsUser.State != StateUnknown {
		t.Fatalf("json = %s, %v", out.String(), err)
	}
}
