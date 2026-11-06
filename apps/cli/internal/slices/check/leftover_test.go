package check

import (
	"strings"
	"testing"
	"testing/fstest"
)

// formerSkill is linkedSkill with metadata.former_skill_dir: the directory
// the runtime made its links in before the move to .agents/skills. Go names
// no engine directory; the fixture picks any.
var formerSkill = strings.Replace(linkedSkill, "  skill_dir: .claude/skills\n",
	"  skill_dir: .claude/skills\n  former_skill_dir: .legacy/skills\n", 1)

const leftoverLine = "✗ .legacy/skills/gabo-pm: gabo link left after the move to .agents/skills"

// formerProject passes: the runtime skill declares a former directory that
// holds no gabo link.
func formerProject() fstest.MapFS {
	return with(linkedProject(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": formerSkill,
		".gabo/skills/gabo-pm/SKILL.md":             "# PM\n",
	})
}

func runFormer(t *testing.T, p fstest.MapFS, s Settings) Report {
	t.Helper()
	return run(t, p, configFiles{projectFile: linkedConfig}, s)
}

// S4: a catalog link left in the former directory fails with the rebuild step.
func TestLeftoverCatalogLinkFails(t *testing.T) {
	t.Parallel()
	p := formerProject()
	p[".legacy/skills/gabo-pm"] = symlink("../../.gabo/skills/gabo-pm")
	r := runFormer(t, p, Settings{})
	wantFailed(t, r, leftoverLine, NextRebuild)
	if !strings.HasPrefix(NextRebuild, "next: ask your agent to rebuild the agents") {
		t.Fatalf("NextRebuild = %q", NextRebuild)
	}
}

// S4: a fixed-name link to a slot adapter folder of gabo.yaml also fails.
func TestLeftoverSlotAdapterLinkFails(t *testing.T) {
	t.Parallel()
	p := formerProject()
	p[".legacy/skills/gabo-tasks"] = symlink("../../.claude/skills/gabo-tasks-github")
	r := runFormer(t, p, Settings{})
	wantFailed(t, r, "✗ .legacy/skills/gabo-tasks: gabo link left after the move to .agents/skills", NextRebuild)
}

// S4: a user's own entry, a real gabo-* folder, a gabo-* link that points
// elsewhere and a missing former directory print nothing.
func TestLeftoverRuleKeepsOtherEntries(t *testing.T) {
	t.Parallel()
	p := formerProject()
	r := runFormer(t, p, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "gabo link left")

	p[".legacy/skills/my-skill"] = symlink("../../.gabo/skills/gabo-pm")
	p[".legacy/skills/mine/SKILL.md"] = file("# mine\n")
	p[".legacy/skills/gabo-x/SKILL.md"] = file("# x\n")
	p[".legacy/skills/gabo-elsewhere"] = symlink("../../vendor/gabo-elsewhere")
	p[".legacy/skills/gabo-abs"] = symlink("/opt/skills/gabo-abs")
	r = runFormer(t, p, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "gabo link left")
}

// S4, S2: an engine not declared in runtimes.* and a runtime skill without
// the key are not read.
func TestLeftoverRuleReadsOnlyDeclaredRuntimesWithTheKey(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".gabo/skills/gabo-pm/SKILL.md": "# PM\n",
		".gabo/skills/gabo-runtime-zed/SKILL.md": "---\nname: gabo-runtime-zed\nmetadata:\n" +
			"  agent_dir: .zed/agents\n  former_skill_dir: .legacy/skills\n---\n",
	})
	p[".legacy/skills/gabo-pm"] = symlink("../../.gabo/skills/gabo-pm")
	r := runFormer(t, p, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "gabo link left")
}

// S4: a former directory that a declared runtime still uses as its
// skill_dir holds current links, not leftovers.
func TestLeftoverRuleSkipsACurrentSkillDir(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": strings.Replace(linkedSkill, "  skill_dir: .claude/skills\n",
			"  skill_dir: .claude/skills\n  former_skill_dir: .claude/skills\n", 1),
	})
	r := runFormer(t, p, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "gabo link left")
}

// S4: a former directory outside the repository fails like skill_dir.
func TestLeftoverRuleFormerDirMustStayInside(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": strings.Replace(formerSkill, ".legacy/skills", "../skills", 1),
	})
	r := runFormer(t, p, Settings{})
	wantFailed(t, r, "✗ runtime claude: former_skill_dir must stay inside the repository")
}

// S9: under GABO_UPGRADE_PR=1 the leftover line never prints.
func TestLeftoverRuleSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	p := formerProject()
	p[".legacy/skills/gabo-pm"] = symlink("../../.gabo/skills/gabo-pm")
	r := runFormer(t, p, Settings{UpgradePR: true})
	noLineWith(t, r, "gabo link left")
	wantLines(t, r, UpgradePRLine)
}
