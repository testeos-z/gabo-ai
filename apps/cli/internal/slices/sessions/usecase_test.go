package sessions

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

var settings = Settings{
	RoleCoder:     {Runtime: "claude", PromptFile: "/data/home/prompts/coder.md"},
	RoleReviewer:  {Runtime: "pi"},
	RoleAssistant: {Runtime: "claude"},
}

func TestParseRoles(t *testing.T) {
	t.Parallel()
	all, err := ParseRoles("all")
	if err != nil || !slices.Equal(all, []Role{RoleCoder, RoleReviewer, RoleAssistant}) {
		t.Fatalf("all = %v, %v", all, err)
	}
	one, err := ParseRoles("reviewer")
	if err != nil || !slices.Equal(one, []Role{RoleReviewer}) {
		t.Fatalf("reviewer = %v, %v", one, err)
	}
	if _, err := ParseRoles("boss"); err == nil {
		t.Fatal("ParseRoles(boss) error = nil")
	}
	if _, err := ParseMode("hybrid"); err == nil {
		t.Fatal("ParseMode(hybrid) error = nil")
	}
}

func TestStartCreatesTerminalsWithRuntimePromptAndMode(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{}
	out, err := NewUseCase(terms, settings).Start(context.Background(), []Role{RoleCoder, RoleReviewer}, ModeCloud)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	want := []kerneltest.TerminalCreate{
		{Title: "gbai-coder", Command: `env GBAI_MODE='cloud' claude "$(cat '/data/home/prompts/coder.md')"`},
		{Title: "gbai-reviewer", Command: `env GBAI_MODE='cloud' pi`},
	}
	if !slices.Equal(terms.Created, want) {
		t.Fatalf("created = %+v", terms.Created)
	}
	if len(out.Lines) != 2 || !strings.Contains(out.Lines[0], "started") {
		t.Fatalf("lines = %v", out.Lines)
	}
}

func TestStartIsIdempotent(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{Live: []kernel.Terminal{{Handle: "term_1", Title: "gbai-coder"}}}
	out, err := NewUseCase(terms, settings).Start(context.Background(), []Role{RoleCoder}, ModeLocal)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(terms.Created) != 0 || !strings.Contains(out.Lines[0], "already running (term_1)") {
		t.Fatalf("created = %v lines = %v", terms.Created, out.Lines)
	}
}

func TestStartReportsEveryRoleOnCreateFailure(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{CreateErr: errors.New("orca down")}
	_, err := NewUseCase(terms, settings).Start(context.Background(), []Role{RoleCoder, RoleAssistant}, ModeLocal)
	if err == nil || !strings.Contains(err.Error(), "gbai-coder") || !strings.Contains(err.Error(), "gbai-assistant") {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestStopClosesOnlyOurExactTitles(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{Live: []kernel.Terminal{
		{Handle: "t1", Title: "gbai-coder"},
		{Handle: "t2", Title: "gbai-coder-old"},
		{Handle: "t3", Title: "cloud-pilot"},
		{Handle: "t4", Title: "gbai-reviewer"},
		{Handle: "t5", Title: "gbai-assistant"},
	}}
	out, err := NewUseCase(terms, settings).Stop(context.Background(), []Role{RoleCoder, RoleReviewer})
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !slices.Equal(terms.Closed, []string{"t1", "t4"}) {
		t.Fatalf("closed = %v", terms.Closed)
	}
	if len(out.Lines) != 2 {
		t.Fatalf("lines = %v", out.Lines)
	}
}

func TestStopReportsNotRunning(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{}
	out, err := NewUseCase(terms, settings).Stop(context.Background(), []Role{RoleAssistant})
	if err != nil || len(terms.Closed) != 0 || !strings.Contains(out.Lines[0], "not running") {
		t.Fatalf("Stop() = %v, %v closed=%v", out.Lines, err, terms.Closed)
	}
}
