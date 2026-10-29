package setupprompt

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docsPage is the published page, relative to this package directory.
var docsPage = filepath.Join("..", "..", "..", "..", "docs", "setup-prompt.md")

// TestPublishedPromptMatchesEmbedded keeps docs/setup-prompt.md identical to
// the embedded prompt between the two marker comments (TRD 9.4, D25, R36).
func TestPublishedPromptMatchesEmbedded(t *testing.T) {
	t.Parallel()
	page, err := os.ReadFile(docsPage)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Published(page)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte(Text)) {
		t.Fatalf("%s differs from the embedded setup-prompt.md; copy the embedded file between the markers\npublished:\n%s\nembedded:\n%s", docsPage, got, Text)
	}
}

func TestPublishedFindsTheBlockBetweenMarkers(t *testing.T) {
	t.Parallel()
	page := "intro\n" + BeginMarker + "\nline one\nline two\n" + EndMarker + "\noutro\n"
	got, err := Published([]byte(page))
	if err != nil || string(got) != "line one\nline two\n" {
		t.Fatalf("Published() = %q, %v", got, err)
	}
	for name, bad := range map[string]string{
		"no markers":       "just text\n",
		"no end":           BeginMarker + "\ntext\n",
		"end before begin": EndMarker + "\n" + BeginMarker + "\n",
		"two begins":       BeginMarker + "\n" + BeginMarker + "\ntext\n" + EndMarker + "\n",
		"two ends":         BeginMarker + "\ntext\n" + EndMarker + "\n" + EndMarker + "\n",
	} {
		if _, err := Published([]byte(bad)); err == nil {
			t.Errorf("%s: Published() accepted %q", name, bad)
		}
	}
}

// TestPromptTellsTheStepsInOrder checks the ordered steps of TRD 9.4.
func TestPromptTellsTheStepsInOrder(t *testing.T) {
	t.Parallel()
	steps := []string{
		"install the gabo CLI",
		"run `gabo init`",
		"`gabo-setup`",
		"`gabo-config-writer`",
		"`gabo-agent-templates`",
		"`gabo-runtime-builder`",
		"global user file",
		"`gabo.yaml`",
		"build every agent",
		"run `gabo check`",
		"smoke test",
	}
	rest := strings.ToLower(Text)
	for _, s := range steps {
		i := strings.Index(rest, strings.ToLower(s))
		if i < 0 {
			t.Fatalf("prompt lacks %q after the previous step", s)
		}
		rest = rest[i+len(s):]
	}
	for _, want := range []string{".gabo/skills/", "https://"} {
		if !strings.Contains(Text, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}

// TestPromptIsNeutralAndHoldsNoSecretOrLocalPath: the prompt names no engine,
// no secret and no machine-specific path (TRD 9.4, R21).
func TestPromptIsNeutralAndHoldsNoSecretOrLocalPath(t *testing.T) {
	t.Parallel()
	if !strings.HasSuffix(Text, "\n") || strings.HasSuffix(Text, "\n\n") || strings.TrimSpace(Text) == "" {
		t.Fatalf("prompt must be non-empty and end with exactly one newline")
	}
	forbidden := regexp.MustCompile(`(?i)\b(claude|pi|codex|opencode|cursor|gemini|copilot|aider|windsurf)\b` +
		`|/home/|/users/|/root/|/tmp/|~/|[a-z]:\\` +
		`|\b(gh[pousr]_[a-z0-9]{10,}|sk-[a-z0-9-]{10,}|akia[0-9a-z]{16})\b`)
	if m := forbidden.FindString(Text); m != "" {
		t.Fatalf("prompt contains %q", m)
	}
}

// TestPromptRecommendsASharedMemory: the prompt says in a short paragraph what
// gabo-setup explains before the memory slot (T070, R37, R38, R36).
func TestPromptRecommendsASharedMemory(t *testing.T) {
	t.Parallel()
	for _, want := range []string{
		"one memory backend and one memory project name", "every agent of this project",
		"except the reviewer", "never reads or writes", "local-only memory is fine",
		"Engram with Engram Cloud sync", "scratch topic", "deletes",
	} {
		if !strings.Contains(Text, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}
