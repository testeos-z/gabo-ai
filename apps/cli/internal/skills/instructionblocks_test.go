package skills

import (
	"strings"
	"testing"
)

// Spec 0030 S19: the catalog carries a template of the three instruction-file
// blocks, and the gabo-setup skill offers it for a project's instruction file.
// Setup never overwrites text the project already has.
//
// Setup is an agent procedure written in gabo-setup/SKILL.md, not Go code, so
// these are text checks: they read the catalog template and the setup skill
// text. They do not run setup against a fixture project.

const (
	instructionTemplatePath = "gabo-setup/instruction-blocks.md"
	instructionCommands = "## Commands"
	instructionMistakes = "## Repeated mistakes"
	instructionFindings = "## Review findings"
)

func instructionTemplateText(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(instructionTemplatePath)
	if err != nil {
		t.Fatalf("read template %s: %v", instructionTemplatePath, err)
	}
	return string(b)
}

// templateSection returns the lines of one level-2 section of the template,
// from its heading up to the next level-2 heading.
func templateSection(t *testing.T, doc, heading string) string {
	t.Helper()
	lines := strings.Split(doc, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " ") == heading {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("template has no %q section", heading)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func TestInstructionTemplateTextHasTheThreeBlocks(t *testing.T) {
	t.Parallel()
	tpl := instructionTemplateText(t)
	var at int
	for _, heading := range []string{instructionCommands, instructionMistakes, instructionFindings} {
		i := strings.Index(tpl[at:], heading+"\n")
		if i < 0 {
			t.Fatalf("template has no %q section, in that order", heading)
		}
		at += i + len(heading)
	}
	if !strings.Contains(tpl, "{{commands}}") {
		t.Error("commands block has no {{commands}} placeholder for the project's check commands")
	}
	mistakes := templateSection(t, tpl, instructionMistakes)
	for _, want := range []string{"two-times rule", "second time", "No entries yet."} {
		if !strings.Contains(strings.ToLower(mistakes), strings.ToLower(want)) {
			t.Errorf("repeated-mistakes section does not contain %q", want)
		}
	}
	findings := strings.ToLower(templateSection(t, tpl, instructionFindings))
	for _, want := range []string{"second time", "proposed", "no agent merges"} {
		if !strings.Contains(findings, want) {
			t.Errorf("review-findings rule does not contain %q", want)
		}
	}
}

func TestSetupSkillTextOffersTheInstructionBlocksWithoutOverwriting(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		// The offer names the template file and its three blocks.
		"Offer the instruction-file blocks", "`instruction-blocks.md`",
		"the commands block, the repeated-mistakes section with the two-times rule, and the review-findings rule",
		// The commands block is filled from the project's own checks, and the
		// text says what to do when there are none.
		"project's own check commands", "its `review.checks`",
		"when `gabo.yaml` lists no `review.checks`, ask the owner for the commands",
		// A section counts as present by its heading line only, outside fenced
		// code blocks, ignoring trailing spaces and CRLF line endings.
		"heading line", "outside fenced code blocks", "CRLF line endings",
		// Setup never overwrites: a missing section is appended, a complete file
		// stays unchanged, and the owner accepts the offer before any write.
		"never overwrites text the project already has",
		"a missing section is appended", "stays unchanged",
		"The owner accepts the offer before any write",
	})
}
