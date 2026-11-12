package skills

import (
	"strings"
	"testing"
)

// Intent 0023 S2 (plan T3, finding #562): the local host refuses an assistant
// on Pi, because `pi --print` ends a session after its first turn and would
// kill the coder subagents of a wave. The refusal starts no process and writes
// no ledger entry.
func TestLocalAgentHostRefusesAPiAssistant(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, agentHostLocal)
	requireIn(t, agentHostLocal, doc, []string{
		"An `assistant` on `pi` is refused before any check",
		"refused: a Pi assistant cannot run a wave under --print; start it from an interactive session or pi --mode rpc",
		"reports a handle of none",
		"no process starts",
		"no ledger entry is written",
		// the pi check below only ever sees a coder: the refusal comes first.
		"# pi: an assistant is refused above, before this check",
	})
	if strings.Contains(doc, "It starts only the `coder` and `assistant` roles") {
		t.Errorf("%s: still starts an assistant on pi", agentHostLocal)
	}
}

// Intent 0023 S2 (plan T3): a coder on Pi keeps its command line unchanged,
// and an assistant on claude starts as before.
func TestLocalAgentHostKeepsTheCoderOnPiAndTheAssistantOnClaude(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, agentHostLocal)
	requireIn(t, agentHostLocal, doc, []string{
		`{ nohup pi --print --approve --tools read,bash,edit,write,grep,find,ls --session-id "$sid" --append-system-prompt <role agent file> @"$dir/prompt.md" "Follow the attached task prompt." </dev/null`,
		`{ nohup claude --print --agent <role agent> --session-id "$sid" --permission-mode dontAsk --permission-prompts none --allowedTools "$tools" <"$dir/prompt.md"`,
	})
}
