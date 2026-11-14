package check

import (
	"reflect"
	"testing"
)

// humanList is the review.human_only_paths of the tests in this file.
var humanList = []string{".github/", ".claude/", "AGENTS.md", "CLAUDE.md"}

// humanWrites returns the human-only paths a shell command writes, judged in
// the coder rules when coder is true, with the project at /proj.
func humanWrites(command string, coder bool) []string {
	var out []string
	for _, h := range humanUseCase(Settings{Root: "/proj"}).humanOnlyShellWrites(command, coder, humanList) {
		out = append(out, h.raw)
	}
	return out
}

// I0028 S7, S13, PD6: a coder shell call that writes a human-only path or
// gabo.yaml is judged: the targets of its redirections, and the operands of
// tee, sed -i, cp, mv and rm. One level of sh -c or bash -c is unwrapped, and
// a command wrapper (env, sudo, xargs, ...) or a brace group is read as bash
// reads it.
func TestHumanOnlyShellWritesBlockedForms(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"echo x > .github/ci.yml",
		"echo x >.github/ci.yml",
		"echo x >> AGENTS.md",
		"echo x 2> CLAUDE.md",
		"echo x >| .claude/settings.json",
		"echo x &> .github/a",
		"echo x &>> .github/a",
		"echo x >& .github/a",
		"echo x <> .github/a",
		"echo x > gabo.yaml",
		"echo x > ./gabo.yaml",
		"echo x > .github",
		"echo ok && echo x > AGENTS.md",
		"tee .github/a",
		"tee -a AGENTS.md",
		"sed -i 's/a/b/' AGENTS.md",
		"sed -i.bak 's/a/b/' CLAUDE.md",
		"sed -ni 's/a/b/' .claude/x",
		"sed --in-place 's/a/b/' .claude/x",
		"cp x .github/y",
		"mv x .github/y",
		"cp -t .github/ x",
		"cp -rt.github/ x",
		"cp --target-directory=.github x",
		"rm .github/x",
		"rm -rf .github/",
		"rm -f gabo.yaml",
		"mv gabo.yaml /tmp/x",
		"rm -f .gi*/x",
		"bash -c 'echo x > .github/a'",
		"FOO=1 echo x > AGENTS.md",
		"echo x >f>.github/a",
		"echo x >.github/a>f",
		"echo x 1>.github/a",
		"echo x 2>>AGENTS.md",
		"echo x >&.github/a",
		`echo x > ".github/a"`,
		`echo x > .gi"thub"/a`,
		"echo x > '/proj/.github/a'",
		"echo x >f&& echo y > CLAUDE.md",
		"env tee .github/a",
		"env FOO=1 tee AGENTS.md",
		"command rm -f gabo.yaml",
		"nohup cp x .github/y",
		"exec tee AGENTS.md",
		"sudo tee .github/a",
		"sudo -u root rm gabo.yaml",
		"nice -n 5 cp x .github/y",
		"time cp x CLAUDE.md",
		"xargs rm .github/x",
		"xargs -I{} cp {} .github/",
		"sudo env rm gabo.yaml",
		"tee .git{hub,}/a",
		"rm -f gab{o,x}.yaml",
		"echo x > {AGENTS,README}.md",
		"cp x {apps,.github}/y",
		"cp AGENTS.md /tmp/notes",
		"bash -c 'env tee .github/a'",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
}

// I0028 S13: a write outside the human-only paths is allowed, and so is a read
// by a command that does not write (cat, a redirected input), a quoted or
// evaluated redirection, and an option that is not a path.
func TestHumanOnlyShellWritesAllowedNeighbours(t *testing.T) {
	t.Parallel()
	allowed := []string{
		"echo x > apps/cli/out.txt",
		"echo x > .githubx/a",
		"echo x > /elsewhere/.github/x",
		"echo x 2>/dev/null",
		"echo x >&2",
		"echo x 2>&1",
		"echo x &>/dev/null",
		"echo x >&-",
		"cat < AGENTS.md",
		"cat AGENTS.md > notes.txt",
		"sed -n 's/a/b/' AGENTS.md",
		"tee out.txt",
		"rm -f one.txt",
		"cp x apps/y",
		"mv apps/a apps/b",
		`echo "> .github/a"`,
		"echo '>.github/x'",
		`eval "echo x > .github/a"`,
		`bash -c "sh -c 'echo x > AGENTS.md'"`,
		"git status",
		"echo x >f>g",
		"echo x 2>&1 >out.txt",
		"echo x >&2 AGENTS.md",
		"env echo x > apps/cli/out.txt",
		"command -v rm",
		"xargs echo .github",
		"echo x > {apps,docs}/a.txt",
		"rm -f a{b}c",
	}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %q; want allowed", c, got)
		}
	}
}

// I0028 S4 and S13: a main session is not judged by the human-only rule.
func TestHumanOnlyShellWritesMainSessionNotJudged(t *testing.T) {
	t.Parallel()
	for _, c := range []string{"echo x > .github/ci.yml", "rm -f AGENTS.md", "cp x gabo.yaml", "env tee .github/a"} {
		if got := humanWrites(c, false); len(got) != 0 {
			t.Errorf("main session: %q is blocked by %q; want allowed", c, got)
		}
	}
}

// The raw words name each written path as the command spells it, in command
// order, each once.
func TestHumanOnlyShellWritesNamesPathsAsWritten(t *testing.T) {
	t.Parallel()
	got := humanWrites("cp x ./.github/y > AGENTS.md && rm -f .github/y && rm -f .github/y", true)
	want := []string{"AGENTS.md", "./.github/y", ".github/y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
}

// I0028 S7: a relative path in a shell call is read from the working
// directory, as the write hook reads it.
func TestHumanOnlyShellWritesReadsTheWorkingDirectory(t *testing.T) {
	t.Parallel()
	u := humanUseCase(Settings{Root: "/proj", WorkDir: "/proj/apps/cli"})
	if got := u.humanOnlyShellWrites("echo x > AGENTS.md", true, humanList); len(got) != 0 {
		t.Fatalf("apps/cli/AGENTS.md is blocked by %v; want allowed", got)
	}
	if got := u.humanOnlyShellWrites("echo x > ../../AGENTS.md", true, humanList); len(got) != 1 {
		t.Fatalf("root AGENTS.md from apps/cli: paths = %v; want one", got)
	}
}

// I0028 S7: brace groups expand as bash expands them, the leftmost group first;
// a brace with no comma stays a literal.
func TestExpandBraces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		word string
		want []string
	}{
		{"a{b,c}d", []string{"abd", "acd"}},
		{"{a,{b,c}}", []string{"a", "b", "c"}},
		{"{a,b}{c,d}", []string{"ac", "ad", "bc", "bd"}},
		{"x{}y", []string{"x{}y"}},
		{"a{b}c", []string{"a{b}c"}},
		{"{a{b,c}}", []string{"{ab}", "{ac}"}},
		{"plain", []string{"plain"}},
	}
	for _, tt := range tests {
		if got := expandBraces(tt.word); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("expandBraces(%q) = %q, want %q", tt.word, got, tt.want)
		}
	}
}

// I0028 S4 (the rule view is unchanged): a & ends a command for the shell guard,
// so the words after &> are read as a command of their own, as before.
func TestShellSegmentsKeepsTheRuleView(t *testing.T) {
	t.Parallel()
	got := shellSegments("echo x &>f git push -f origin y")
	want := [][]string{{"echo", "x"}, {">f", "git", "push", "-f", "origin", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("segments = %q, want %q", got, want)
	}
}
