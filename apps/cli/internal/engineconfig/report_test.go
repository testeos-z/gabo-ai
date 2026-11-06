package engineconfig

import "testing"

func TestReportRendersSectionsThenSteps(t *testing.T) {
	r := Report{Sections: []Section{
		{Engine: "claude", Lines: []string{"wrote .claude/settings.json: added a, b"}, Steps: []string{"trust the folder in Claude Code"}},
		{Engine: "codex", Lines: []string{"✗ .codex/config.toml: invalid TOML; fix it and run gabo init again"}, Steps: []string{"trust the project in Codex", "trust both gabo hooks in /hooks"}},
		{Engine: "acme", Lines: []string{"acme: no gabo integration known; nothing written"}},
	}}
	want := `engine integration:
  claude:
    wrote .claude/settings.json: added a, b
  codex:
    ✗ .codex/config.toml: invalid TOML; fix it and run gabo init again
  acme:
    acme: no gabo integration known; nothing written
you still need to:
  - claude: trust the folder in Claude Code
  - codex: trust the project in Codex
  - codex: trust both gabo hooks in /hooks
`
	if got := r.Render(); got != want {
		t.Errorf("Render() =\n%s\nwant\n%s", got, want)
	}
	if !r.Failed() {
		t.Error("Failed() = false, want true with a ✗ line")
	}
}

func TestReportWithoutStepsOrFailures(t *testing.T) {
	r := Report{Sections: []Section{{Engine: "acme", Lines: []string{"acme: no gabo integration known; nothing written"}}}}
	want := "engine integration:\n  acme:\n    acme: no gabo integration known; nothing written\n"
	if got := r.Render(); got != want {
		t.Errorf("Render() =\n%s\nwant\n%s", got, want)
	}
	if r.Failed() {
		t.Error("Failed() = true, want false without a ✗ line")
	}
}

// Intent 0013 S3: the removal line names runtimes.assistant as written: one
// engine as in intent 0010, a list joined with ", ".
func TestRemovedMainSessionLine(t *testing.T) {
	for _, c := range []struct {
		assistants []string
		want       string
	}{
		{[]string{"codex"}, `removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is codex)`},
		{[]string{"claude", "pi"}, `removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude, pi)`},
	} {
		if got := removedMainSessionLine("opencode.json", "default_agent", "gabo-assistant", Input{Assistants: c.assistants}); got != c.want {
			t.Errorf("removedMainSessionLine(%q) = %q, want %q", c.assistants, got, c.want)
		}
	}
}
