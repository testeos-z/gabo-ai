package skills

import (
	"slices"
	"strings"
	"testing"
)

const (
	agentHostLocal   = "gabo-agent-host-local/SKILL.md"
	agentHostRailway = "gabo-agent-host-railway-orca/SKILL.md"
	directSessions   = "gabo-direct-sessions-orca/SKILL.md"
)

// sessionResolution is how every host resolves `session` (intent 0013 S10):
// the engine of the calling session, read from the caller's built agent
// (`{{engine}}`, intent 0013 S6), and unresolved when that line is missing.
var sessionResolution = []string{
	"intent 0013 S10",
	"`session` is the engine of the calling session",
	"the `gabo-engine:` value of the caller's built agent (`{{engine}}`, intent 0013 S6)",
	"`refused: session engine unknown (rebuild the agents)`",
	"a fixed `runtimes.coder` is that engine",
}

// assistantRule starts an assistant only on an engine listed in
// `runtimes.assistant` and refuses any other naming that key (intent 0013
// S10: an `assistant` start on `codex` with `assistant: [claude, pi]` is
// refused).
var assistantRule = []string{
	"an `assistant` only on an engine listed in `runtimes.assistant` (one engine or a list, intent 0013 S4)",
	"`refused: <engine> is not listed in runtimes.assistant (<value>)`",
}

// operationHeadings returns the `###` headings under `## Operations`.
func operationHeadings(doc string) []string {
	i := strings.Index(doc, "\n## Operations\n")
	if i < 0 {
		return nil
	}
	rest := doc[i+len("\n## Operations\n"):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	var hs []string
	for _, l := range strings.Split(rest, "\n") {
		if strings.HasPrefix(l, "### ") {
			hs = append(hs, l)
		}
	}
	return hs
}

// Intent 0013 S10 (plan T11): the local host resolves `session` to the
// calling session's engine, starts a coder on the resolved coder engine
// (`coder: session` from a Pi session ⇒ `pi`) and an assistant only on an
// engine listed in `runtimes.assistant`.
func TestLocalAgentHostResolvesSessionAndTheAssistantList(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, agentHostLocal)
	requireIn(t, agentHostLocal, doc, sessionResolution)
	requireIn(t, agentHostLocal, doc, assistantRule)
	requireIn(t, agentHostLocal, doc, []string{
		"a `coder` on the resolved coder engine",
		"`coder: session` from a Pi session ⇒ `pi`",
		"`assistant: [claude, pi]`: an `assistant` on `codex` ⇒ `refused: codex is not listed in runtimes.assistant ([claude, pi])`",
		// the local start commands exist for claude and pi only.
		"`refused: <engine> has no local start command in gabo-agent-host-local`",
	})
	if strings.Contains(doc, "(`runtimes.coder` or `runtimes.assistant`: `claude` or `pi`)") {
		t.Errorf("%s: still reads one engine per role from `runtimes.*`", agentHostLocal)
	}
}

// Intent 0013 S10 (plan T11): the Orca direct sessions resolve the coder
// engine the same way.
func TestDirectSessionsResolveSession(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	requireIn(t, directSessions, doc, sessionResolution)
	requireIn(t, directSessions, doc, []string{
		"`<agent cli>` is the resolved coder engine",
	})
}

// Intent 0013 S10 (plan T11): the Railway and Orca host resolves `session`
// for coder and reviewer and starts an assistant only on a listed engine.
func TestRailwayAgentHostResolvesSessionAndTheAssistantList(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, agentHostRailway)
	requireIn(t, agentHostRailway, doc, sessionResolution)
	requireIn(t, agentHostRailway, doc, assistantRule)
	requireIn(t, agentHostRailway, doc, []string{
		"a fixed `runtimes.reviewer` is that engine",
		"| coder | the resolved coder engine |",
		"| reviewer | the resolved reviewer engine |",
		"| assistant | the requested engine, listed in `runtimes.assistant` |",
	})
}

// Intent 0013 S10 (plan T11): resolving engines adds no operation, so the
// slot coverage `gabo check` reads is unchanged.
func TestAgentHostOperationHeadingsAreUnchanged(t *testing.T) {
	t.Parallel()
	host := []string{"### start agent session", "### stop agent session", "### report where agents run"}
	for path, want := range map[string][]string{
		agentHostLocal:   host,
		agentHostRailway: host,
		directSessions:   {"### open session", "### send message", "### read state", "### close session"},
	} {
		if got := operationHeadings(readCatalog(t, path)); !slices.Equal(got, want) {
			t.Errorf("%s: operation headings %q, want %q", path, got, want)
		}
	}
}

// Intent 0015 S1 (plan T4): the host skills name the main session, not a
// `gabo-cycle` agent, as the caller.
func TestAgentHostsNameTheMainSessionAsCaller(t *testing.T) {
	t.Parallel()
	for _, name := range []string{agentHostLocal, agentHostRailway} {
		doc := readCatalog(t, name)
		requireIn(t, name, doc, []string{"the main session (the assistant running the `gabo-cycle` skill)"})
		for _, s := range []string{"`gabo-cycle` or `gabo-assistant`", "`gabo-cycle` launching"} {
			if strings.Contains(doc, s) {
				t.Errorf("%s: must not say %q", name, s)
			}
		}
	}
}
