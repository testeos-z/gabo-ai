package skills

import (
	"strings"
	"testing"
)

// Intent 0020 (plan T3): the optional turn limit reaches the built coder
// through a template slot (text only, no agents.yaml field), the Claude Code
// and OpenCode engine keys, and the coder's stop rule (PD7, PD11).
const turnSlot = "{{turn_limit_line}}"

func TestAgentTemplatesListTheTurnLimitSlot(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-agent-templates/SKILL.md", readCatalog(t, "gabo-agent-templates/SKILL.md"), []string{
		"| `" + turnSlot + "` |",
		"`Turn limit: <n> turns; stop at it and report STOPPED at turn limit.`",
		"`wave.max_coder_turns` is set", "the whole line is removed when it is unset",
		"byte-identical", "the engine key is written into `gabo-coder` only",
	})
}

func TestOnlyTheCoderTemplateCarriesTheTurnLimitSlot(t *testing.T) {
	t.Parallel()
	for path, body := range embeddedByPath(t) {
		if !strings.Contains(path, "/agents/") || !strings.HasSuffix(path, ".md") {
			continue
		}
		has := strings.Contains(body, turnSlot)
		if want := strings.HasSuffix(path, "/gabo-coder.md"); has != want {
			t.Errorf("%s: carries %s = %v, want %v", path, turnSlot, has, want)
		}
		if has {
			own := false
			for _, l := range strings.Split(body, "\n") {
				if l == turnSlot {
					own = true
				}
			}
			if !own {
				t.Errorf("%s: %s is not on its own line", path, turnSlot)
			}
		}
	}
}

func TestAgentsSpecHasNoTurnLimitField(t *testing.T) {
	t.Parallel()
	spec := readCatalog(t, "gabo-agent-templates/agents.yaml")
	for _, s := range []string{"turn_limit", "max_coder_turns", "maxTurns", "steps"} {
		if strings.Contains(spec, s) {
			t.Errorf("agents.yaml: carries %q; the turn limit is a template slot, not a spec field (PD7)", s)
		}
	}
}

func TestRuntimeSkillsMapTheTurnLimit(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-runtime-claude/SKILL.md", readCatalog(t, "gabo-runtime-claude/SKILL.md"), []string{
		"`" + turnSlot + "`", "`maxTurns: <wave.max_coder_turns>`", "for `gabo-coder` only",
		"after `skills` and before `gabo-version`", "omit it when `wave.max_coder_turns` is unset",
		"keys, in this order and no others, except the one `maxTurns` key of `gabo-coder`",
		"changed `wave.max_coder_turns` needs a rebuild",
	})
	requireIn(t, "gabo-runtime-opencode/SKILL.md", readCatalog(t, "gabo-runtime-opencode/SKILL.md"), []string{
		"`" + turnSlot + "`", "`steps: <wave.max_coder_turns>`", "`gabo-coder` only",
		"omit it when `wave.max_coder_turns` is unset",
		"frontmatter holds `description`, `mode` and `permissions` and no other key, except `steps` for `gabo-coder`",
		"changed `wave.max_coder_turns` needs a rebuild",
	})
	for _, name := range []string{"gabo-runtime-codex", "gabo-runtime-pi"} {
		requireIn(t, name+"/SKILL.md", readCatalog(t, name+"/SKILL.md"), []string{
			"`" + turnSlot + "`", "the coder gets no engine key for the turn limit",
			"it travels in the launch input as `turn limit: <n>` and is advisory",
		})
	}
}

func TestCoderStopsAtItsTurnLimit(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-coder/SKILL.md", readCatalog(t, "gabo-coder/SKILL.md"), []string{
		"`turn limit: <n>`", "stops at it", "commits nothing new",
		"reports `STOPPED at turn limit` with its resources",
		"Max 4 fix rounds; then label the issue `needs-user`, report BLOCKED with the question.",
	})
}
