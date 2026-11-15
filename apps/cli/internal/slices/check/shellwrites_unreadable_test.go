package check

import "testing"

// I0028 S7, S13 (review round 5): a write target the guard cannot read is
// blocked for a coder. A target with a variable ($), a tilde (~) or a command
// substitution ($( ) or a backquote) names a path only the shell can work
// out, so the guard cannot tell whether it is human-only. Each reviewer
// example is pinned, and the substitution that used to split the target off
// the command is pinned too.
func TestHumanOnlyShellWritesUnreadableTargets(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"echo x > $F",
		"echo x > ${F}",
		`rm -rf "$PWD"`,
		"rm -rf ~",
		"echo x > ~/proj/gabo.yaml",
		`tee "$PWD/.github/ci.yml"`,
		"cp x $DEST",
		"mv x ${DIR}/a",
		"sed -i 's/a/b/' $F",
		`rm -rf "$(pwd)"`,
		"rm -rf $(pwd)",
		"rm -rf `pwd`",
		"echo x > $(echo .github/a)",
		"echo x >$(echo apps/out.txt)",
		"echo x > `echo .github/a`",
		"tee `echo apps/a`",
		"echo $(date) > AGENTS.md",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
}

// The raw word names an unreadable target as the command spells it, so the
// rule message shows the substitution rather than a marker.
func TestHumanOnlyShellWritesUnreadableTargetNamedAsWritten(t *testing.T) {
	t.Parallel()
	got := humanWrites("echo x > $(echo .github/a)", true)
	if len(got) != 1 || got[0] != "$(echo .github/a)" {
		t.Fatalf("paths = %q, want [$(echo .github/a)]", got)
	}
}

// I0028 S13: a variable, a tilde or a substitution that is not the target of a
// write is no path the guard must judge, and a literal target next to it is
// judged as before.
func TestHumanOnlyShellWritesUnreadableNeighbours(t *testing.T) {
	t.Parallel()
	allowed := []string{
		`echo "$HOME" > apps/out.txt`,
		"echo $(date) > apps/out.txt",
		"echo ~ > apps/out.txt",
		`sed -i "s/$X/y/" apps/a.txt`,
		"echo $(date)",
		"rm -f one.txt",
	}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %q; want allowed", c, got)
		}
	}
}
