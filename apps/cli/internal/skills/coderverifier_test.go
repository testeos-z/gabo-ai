package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0027 S28 (plan T3): before the PR the coder has a fresh-context
// verifier run the changed behavior. The rule is a fixed gabo rule in the coder
// skill and its template (Q33); no agent file is added for it (PD4).
var coderVerifierRule = []string{
	"### Fresh-context verifier",
	"before you open the PR",
	"starts without your conversation",
	"the changed behavior and the two nearest neighboring flows",
	"reports the commands it ran and what it saw",
	"changes no file",
	"never approves or blocks",
	"`verifier: <commands and what it saw>`",
	"`verifier: not applicable: no behavior change`",
	"in the engine's read-only mode (no write tool)",
	"report `verifier: not run: <reason>`",
	"what it saw for the changed behavior and for each of the two flows",
	"lists each behavior that does not match a linked S# or a done-contract line",
}

func TestCoderSkillNamesTheFreshContextVerifier(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, coderVerifierRule)
	_, contract, ok := strings.Cut(doc, "## Output Contract")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Output Contract section")
	}
	if !strings.Contains(contract, "`verifier: <commands and what it saw>`") {
		t.Error("gabo-coder/SKILL.md: the Output Contract does not hold the verifier field")
	}
}

func TestCoderTemplateNamesTheVerifierField(t *testing.T) {
	t.Parallel()
	doc := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	for _, s := range []string{"`verifier:`", "fresh-context verifier", "`verifier: not applicable: no behavior change`", "Before you open the PR", "starts without your conversation", "each mismatch against a linked S# or the done contract", "changes no file and never approves or blocks", "`commits:`, `verifier:`", "`verifier: not run: <reason>`"} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-coder template: missing %q", s)
		}
	}
}

// PD4: the verifier is a rule, not a file; the built coder agent carries the
// field and no verifier agent file exists.
func TestCoderVerifierAddsNoAgentFile(t *testing.T) {
	t.Parallel()
	built := readRepoDoc(t, ".claude/agents/gabo-coder.md")
	if !strings.Contains(built, "`verifier:`") {
		t.Error(".claude/agents/gabo-coder.md: missing `verifier:` (rebuild it from the template)")
	}
	for _, p := range []string{".claude/agents/gabo-verifier.md", ".claude/agents/verifier.md"} {
		if _, err := os.Stat(filepath.Join(repoRoot, p)); err == nil {
			t.Errorf("%s must not exist (PD4)", p)
		}
	}
}

// Q34: gabo pins no model for the verifier; the engine chooses it. The whole
// verifier rule and the template carry no model or engine-model name.
func TestCoderVerifierPinsNoModel(t *testing.T) {
	t.Parallel()
	skill := readCatalog(t, "gabo-coder/SKILL.md")
	_, rule, ok := strings.Cut(skill, "### Fresh-context verifier")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no Fresh-context verifier section")
	}
	rule, _, _ = strings.Cut(rule, "## Output Contract")
	tmpl := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	_, verifierBullet, _ := strings.Cut(tmpl, "- Before you open the PR")
	verifierBullet, _, _ = strings.Cut(verifierBullet, "\n- ")
	for name, doc := range map[string]string{"SKILL.md rule": rule, "template bullet": verifierBullet} {
		lower := strings.ToLower(doc)
		for _, s := range []string{
			"small model", "cheaper model", "stronger than", "model:",
			"haiku", "sonnet", "opus", "fable", "gpt", "gemini", "claude-",
		} {
			if strings.Contains(lower, s) {
				t.Errorf("gabo-coder %s: must not pin a model (%q)", name, s)
			}
		}
	}
}

// The block is listed "in this order": commits, then verifier, then RDD.
func TestCoderSkillVerifierFieldOrder(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	_, block, ok := strings.Cut(doc, "### Methodology report block")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no Methodology report block")
	}
	c := strings.Index(block, "`commits: <sha> <message>`")
	v := strings.Index(block, "- Verifier: `verifier:")
	r := strings.Index(block, "`RDD: <granted")
	if c < 0 || v < 0 || r < 0 || !(c < v && v < r) {
		t.Errorf("the verifier line must sit between commits and RDD (commits %d, verifier %d, RDD %d)", c, v, r)
	}
}
