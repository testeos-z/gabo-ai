package check

import (
	"strings"
	"testing"
	"testing/fstest"
)

const (
	reviewerNext       = "next: remove it; the reviewer runs from the user-level copy"
	reviewerInProject  = ": gabo-reviewer must not be in the project"
	claudeReviewerLine = "✗ .claude/skills/gabo-reviewer" + reviewerInProject
)

// otherSkill is a second runtime skill (pi) with its own skill directory.
// Go names no engine directory; the fixture picks any.
const otherSkill = "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .claude/agents\n  skill_dir: .other/skills\n  agent_skills: frontmatter:skills\n---\nbody\n"

func runReviewer(t *testing.T, p fstest.MapFS, cfg string, s Settings) Report {
	t.Helper()
	return run(t, p, configFiles{projectFile: cfg}, s)
}

// S8: a gabo-reviewer link in the project skill dir fails with its path and
// the removal step.
func TestReviewerLinkInProjectFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".claude/skills/gabo-reviewer"] = symlink("../../.gabo/skills/sk-a")
	r := runReviewer(t, p, linkedConfig, Settings{})
	wantFailed(t, r, claudeReviewerLine, reviewerNext)
}

// S8: a real gabo-reviewer folder fails the same way.
func TestReviewerFolderInProjectFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".claude/skills/gabo-reviewer/SKILL.md"] = file("# reviewer\n")
	r := runReviewer(t, p, linkedConfig, Settings{})
	wantFailed(t, r, claudeReviewerLine, reviewerNext)
}

// S8: the skill dir of every engine declared in runtimes.* is read.
func TestReviewerInEveryDeclaredSkillDirFails(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{".gabo/skills/gabo-runtime-pi/SKILL.md": otherSkill})
	p[".other/skills/gabo-reviewer/SKILL.md"] = file("# reviewer\n")
	cfg := strings.Replace(linkedConfig, "assistant: claude", "assistant: pi", 1)
	r := runReviewer(t, p, cfg, Settings{})
	wantFailed(t, r, "✗ .other/skills/gabo-reviewer"+reviewerInProject, reviewerNext)
	noLineWith(t, r, claudeReviewerLine)
}

// S8: an engine not declared in runtimes.* is not read.
func TestReviewerRuleSkipsUndeclaredEngines(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{".gabo/skills/gabo-runtime-pi/SKILL.md": otherSkill})
	p[".other/skills/gabo-reviewer/SKILL.md"] = file("# reviewer\n")
	r := runReviewer(t, p, linkedConfig, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, reviewerInProject)
}

// S8: an agent file whose declared skills include gabo-reviewer fails with
// that file's path.
func TestReviewerInAgentSkillsFails(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".claude/agents/alpha.md": "---\nname: alpha\nskills:\n  - sk-a\n  - sk-b\n  - gabo-reviewer\ngabo-version: 1.2.3\n---\nbody\n",
		".claude/agents/mine.md":  "---\nname: mine\nskills: [gabo-reviewer]\n---\nbody\n",
	})
	r := runReviewer(t, p, linkedConfig, Settings{})
	wantFailed(t, r,
		"✗ .claude/agents/alpha.md"+reviewerInProject,
		"✗ .claude/agents/mine.md"+reviewerInProject,
		reviewerNext)
}

// S8: a name that only starts with gabo-reviewer is not the reviewer.
func TestReviewerConfigIsNotTheReviewer(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".claude/agents/beta.md": "---\nname: beta\nskills: [sk-b, gabo-reviewer-config]\ngabo-version: 1.2.3\n---\nbody\n",
	})
	p[".claude/skills/gabo-reviewer-config/SKILL.md"] = file("# config\n")
	r := runReviewer(t, p, linkedConfig, Settings{})
	noLineWith(t, r, reviewerInProject)
}

// S8: the clean project passes with no reviewer line.
func TestReviewerRuleCleanProjectPasses(t *testing.T) {
	t.Parallel()
	r := runReviewer(t, linkedProject(), linkedConfig, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, reviewerInProject)
}

// S9: under GABO_UPGRADE_PR=1 the reviewer line never prints.
func TestReviewerRuleSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".claude/skills/gabo-reviewer"] = symlink("../../.gabo/skills/sk-a")
	r := runReviewer(t, p, linkedConfig, Settings{UpgradePR: true})
	noLineWith(t, r, reviewerInProject)
	wantLines(t, r, UpgradePRLine)
}

// Intent 0013 S5: with runtimes.reviewer session the isolation rule reads
// the skill dir of every listed assistant engine; session is no engine.
func TestReviewerInEveryListedAssistantSkillDirFails(t *testing.T) {
	t.Parallel()
	p := with(listedProject(), map[string]string{".gabo/skills/gabo-runtime-pi/SKILL.md": otherSkill})
	p[".other/skills/gabo-reviewer/SKILL.md"] = file("# reviewer\n")
	r := runReviewer(t, p, withRuntimes(t, linkedConfig, sessionRuntimes), Settings{})
	wantFailed(t, r, "✗ .other/skills/gabo-reviewer"+reviewerInProject, reviewerNext)
	noLineWith(t, r, claudeReviewerLine)
	noLineWith(t, r, "runtime session")
}
