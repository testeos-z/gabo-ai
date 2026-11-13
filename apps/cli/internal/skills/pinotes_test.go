package skills

import (
	"strings"
	"testing"
)

// Intent 0023 S4 (plan T5, finding #563; four paths by intent 0026 S5, plan T6): the Pi notes name the four grants an
// unattended Pi wave needs, say the grant is the owner's and that gabo writes no
// permission to the user's engine config. A missing section makes every
// assertion fail on its own, so each one is seen red separately.
func TestRuntimePiNamesTheUnattendedWaveGrants(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	_, section, ok := strings.Cut(doc, "\n## Grants for an unattended Pi wave\n")
	if !ok {
		t.Errorf("%s: no %q section", runtimePi, "## Grants for an unattended Pi wave")
	}
	section, _, _ = strings.Cut(section, "\n## ")

	for _, s := range []string{
		// the four grants, each named by its path as the owner grants it.
		"four grants an unattended Pi wave needs",
		"`worktree.dir`",
		"`${XDG_CONFIG_HOME:-~/.config}/gabo/config.yaml`",
		"`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/waves`",
		// the grant belongs to the owner; gabo grants none of it and writes no permission.
		"The grant is the owner's",
		"gabo grants none of them",
		"writes no permission to the user's engine config",
	} {
		if !strings.Contains(section, s) {
			t.Errorf("%s: Pi grants section missing %q", runtimePi, s)
		}
	}
}
