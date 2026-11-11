package skills

import (
	"slices"
	"strings"
	"testing"
)

// directSessionsContract is the contract every direct_sessions adapter covers
// (intent 0019, plan T1).
const directSessionsContract = "gabo-contract-direct-sessions/SKILL.md"

// contractSection returns the text under heading (`### send message`, `## Handle`,
// ...) up to the next heading of the same or a higher level, or "" when the
// document has no such heading.
func contractSection(doc, heading string) string {
	level := len(heading) - len(strings.TrimLeft(heading, "#"))
	i := strings.Index(doc, "\n"+heading+"\n")
	if i < 0 {
		return ""
	}
	var out []string
	for _, l := range strings.Split(doc[i+len(heading)+2:], "\n") {
		if n := len(l) - len(strings.TrimLeft(l, "#")); n > 0 && n <= level && strings.HasPrefix(l[n:], " ") {
			break
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// sectionRequire fails for every want string missing from the section named
// heading of doc.
func sectionRequire(t *testing.T, heading, section string, want []string) {
	t.Helper()
	if section == "" {
		t.Errorf("%s: section missing", heading)
		return
	}
	for _, s := range want {
		if !strings.Contains(section, s) {
			t.Errorf("%s: missing %q", heading, s)
		}
	}
}

// Intent 0019 S1, S7 (plan T1): the four operation headings stay, in order, and
// every existing hard rule of the contract stays verbatim. These assertions hold
// on the contract before this change, so they pin behavior the change touches.
func TestDirectSessionsContractKeepsItsHeadingsAndHardRules(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	want := []string{"### open session", "### send message", "### read state", "### close session"}
	if got := operationHeadings(doc); !slices.Equal(got, want) {
		t.Errorf("%s: operation headings %q, want %q", directSessionsContract, got, want)
	}
	requireIn(t, directSessionsContract, doc, []string{
		"A session opens only inside an existing task worktree; it never creates, moves or removes worktrees and never touches another task's worktree.",
		"The brief names the task id, the worktree path and the branch, and repeats the coder's hard rules: never merge, never force-push, never post the reviewer marker.",
		"A session is not a second coder: while the task's coder is running, messages go to the open session only when the owner asked for it.",
		"Every session the adapter opens is closed or reported as still open; a session is a resource and goes into the wave ledger.",
	})
}

// Intent 0019 S1, S2, PD3 (plan T1): send message and read state take the
// handle of any session agent_host reports, local or remote; the handle is two
// named fields; a handle naming no known host is refused and nothing is sent.
func TestDirectSessionsContractTakesAnyAgentHostSessionHandle(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	sectionRequire(t, "## Handle", contractSection(doc, "## Handle"), []string{
		"A handle has two named fields, `host` and `handle`, never one joined string.",
		"`host` is the host name that `agent_host` reports for where the session runs; the literal `local` names the local host.",
		"`handle` is the session handle that host returned.",
		"A handle that names no known host is refused, and nothing is sent or read.",
		"The ledger records both fields of the handle.",
	})
	for _, op := range []string{"### send message", "### read state"} {
		sectionRequire(t, op, contractSection(doc, op), []string{
			"any session that `agent_host` reports (`start agent session` or `report where agents run`)",
			"on this machine or on a remote host",
			"the handle defined under `## Handle`",
		})
	}
	sectionRequire(t, "### open session", contractSection(doc, "### open session"), []string{
		"Start an agent session in one task worktree with a brief.",
		"Return a session handle as the two fields of `## Handle`: `host` and `handle`.",
	})
}

// Intent 0019 S1, D6 (plan T1): close session closes only a session this slot
// opened; any other handle is refused, naming stop agent session of the host.
func TestDirectSessionsContractClosesOnlyOpenedSessions(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	sectionRequire(t, "### close session", contractSection(doc, "### close session"), []string{
		"closes only a session this slot opened",
		"For any other handle, refuse and name `stop agent session` of the host that reported it",
	})
}

// Intent 0019 S4, D5, PD4 (plan T1): any remote use needs the owner's explicit
// authorization naming host, operation and credential or session; a missing one
// sends and reads nothing; no ambient access; host A never reaches host B.
func TestDirectSessionsContractRequiresOwnerAuthorizationForRemoteUse(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	requireIn(t, directSessionsContract, contractSection(doc, "## Hard Rules"), []string{
		"Any remote use needs the owner's explicit authorization, given in this session, that names the host, the operation (`send message` or `read state`) and the credential or session to use.",
		"A missing authorization sends and reads nothing, and the adapter asks the caller for it.",
		"An authorization for host A never reaches host B.",
		"The adapter never discovers or reuses ambient credentials, SSH agents, ControlMaster sockets, Orca pairings or authenticated sessions.",
	})
}

// Intent 0019 S5, D4 (plan T1): a direct session is live conversation only;
// what must persist goes to the tracker as an issue comment and a decision is
// also saved through memory.
func TestDirectSessionsContractSaysSessionsAreLiveOnly(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	requireIn(t, directSessionsContract, contractSection(doc, "## Hard Rules"), []string{
		"A direct session is live conversation only",
		"a question, an answer or a decision that must persist goes to the tracker as an issue comment through the `tasks` adapter",
		"a decision is also saved through the `memory` slot",
	})
}

// Intent 0019 S6 (plan T1): an adapter that cannot return the reply says so in
// read state, names where the answer is read, and never returns an empty output.
func TestDirectSessionsContractReadStateNamesUnreturnedReplies(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessionsContract)
	sectionRequire(t, "### read state", contractSection(doc, "### read state"), []string{
		"An adapter that cannot return the reply says so and names where the answer is read",
		"reply not returned by this command; read it in the session at claude.ai/code",
		"`claude -p ... --cloud`",
		"never returns an empty output as if the agent were silent",
	})
}
