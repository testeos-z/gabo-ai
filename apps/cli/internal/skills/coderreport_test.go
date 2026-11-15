package skills

import (
	"strings"
	"testing"
)

// Intent 0027 S1, S2, S8, S10, S12, S15, S18 (plan T2): the coder report
// contract. The fields below are pinned in the `gabo-coder` output contract
// and named by the agent template, so a coder cannot lose one silently.
var coderReportFields = []string{
	// S8: the block is complete and a field not produced never reads as a pass.
	"methodology report block",
	"A field you did not produce reads `not run: <reason>`, never as a pass",
	// S1: ODD evidence.
	"`odd: RED <test name and failure>; GREEN <test name and run>`",
	"`odd: exception: <reason>; checks: <proportional checks run>`",
	// S2: one line per linked S#.
	"`spec S<n>: covered`",
	"`spec S<n>: not covered: <reason>`",
	"one line for every S# the task links",
	// S12: done contract on the task issue before the first commit.
	"before your first commit, write the done contract on the task issue",
	"`not provable: <reason>`",
	"`contract: written on the task issue`",
	"`contract: not run: <reason>`",
	// S15: tests with their results.
	"`tests: <test name>: <result>`",
	"`integration: <test name>: <result>`",
	"`integration: not applicable: <reason>`",
	// S18 fields: each review.checks command, its pasted output and an exit line.
	"`check: <command line as listed in review.checks>`",
	"its pasted output",
	"`exit: <n>`",
	// S8: commits and the RDD outcome.
	"`commits: <sha> <message>`",
	"`RDD: <granted|declined|passive|off|unavailable>`",
	"`risk` and `review_due_reason`",
}

func TestCoderSkillNamesTheReportContract(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, coderReportFields)
	_, contract, ok := strings.Cut(doc, "## Output Contract")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Output Contract section")
	}
	for _, s := range []string{"methodology report block", "`exit: <n>`", "`spec S<n>: covered`"} {
		if !strings.Contains(contract, s) {
			t.Errorf("gabo-coder/SKILL.md: the Output Contract does not hold %q", s)
		}
	}
}

func TestCoderTemplateNamesTheReportFields(t *testing.T) {
	t.Parallel()
	doc := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	for _, s := range []string{
		"`odd:`", "`spec S<n>:`", "`contract:`", "`tests:`", "`integration:`",
		"`check:`", "`exit: <n>`", "`commits:`", "`RDD:`",
		"`not run: <reason>`",
		"write the done contract on the task issue before your first commit",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-coder template: missing %q", s)
		}
	}
}

// The catalog copies of the coder rules stay byte-identical: the codex plugin
// copy, the project copy and the built Claude Code agent (plan T2).
func TestCoderCopiesStayByteIdentical(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"apps/cli/internal/skills/gabo-coder/SKILL.md", ".gabo/skills/gabo-coder/SKILL.md"},
		{"apps/cli/internal/skills/gabo-coder/SKILL.md", "plugins/codex/skills/gabo-coder/SKILL.md"},
		{"apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md", ".gabo/skills/gabo-agent-templates/agents/gabo-coder.md"},
		{"apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md", "plugins/codex/skills/gabo-agent-templates/agents/gabo-coder.md"},
	}
	for _, p := range pairs {
		if readRepoDoc(t, p[0]) != readRepoDoc(t, p[1]) {
			t.Errorf("%s differs from %s", p[1], p[0])
		}
	}
	// the built agent carries the report block the template names.
	built := readRepoDoc(t, ".claude/agents/gabo-coder.md")
	for _, s := range []string{"`odd:`", "`exit: <n>`", "`RDD:`"} {
		if !strings.Contains(built, s) {
			t.Errorf(".claude/agents/gabo-coder.md: missing %q (rebuild it from the template)", s)
		}
	}
}
