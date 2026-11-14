package check

import (
	"testing"
)

// judged reports whether a shell command breaks a rule, and checks that every
// violation names a rule and a route.
func judged(t *testing.T, command string, coder bool) []ShellViolation {
	t.Helper()
	vs := ShellViolations(command, coder)
	for _, v := range vs {
		if v.Rule == "" || v.Route == "" {
			t.Fatalf("%q: violation %+v lacks a rule or a route", command, v)
		}
	}
	return vs
}

// I0028 S4: each form a coder session blocks, and one allowed neighbour each.
func TestShellGuardCoderSessionForms(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"git push origin main",
		"git push origin master",
		"git push origin dev",
		"git push --force origin feat/x",
		"git push -f origin feat/x",
		"git push --force-with-lease origin feat/x",
		"git push --mirror",
		"git push origin +feat/x",
		"git branch -D x",
		"git branch --delete --force x",
		"git worktree remove --force p",
		"git worktree remove -f p",
		"rm -rf build/*",
		"rm -rf a?",
		"rm -rf a[0-9]",
		"rm -rf ../x-worktrees/y",
	}
	allowed := []string{
		"git push origin feat/x",
		"git branch -d feat/x",
		"git worktree remove p",
		"rm -rf /tmp/tmp.Ab12/out",
		"rm -f one.txt",
		"git status",
	}
	for _, c := range blocked {
		if len(judged(t, c, true)) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	for _, c := range allowed {
		if vs := judged(t, c, true); len(vs) != 0 {
			t.Errorf("coder: %q is blocked by %+v; want allowed", c, vs)
		}
	}
}

// I0028 S4: a session that is not a coder blocks only the push to main; the
// reviewer's merge into dev is exempt (decision 13), so dev is allowed too.
func TestShellGuardMainSessionOnlyPushToMain(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"git push origin main",
		"git push -d origin main",
		"git push origin HEAD:main",
		"git push origin feat/x:refs/heads/main",
		"git push --repo=origin main",
		"git push --repo origin main",
	}
	allowed := []string{
		"git push --force origin feat/x",
		"git push origin dev",
		"git push origin feat/x",
		"git branch -D x",
		"rm -rf build/*",
		"git push origin main:feat/x",
	}
	for _, c := range blocked {
		if len(judged(t, c, false)) == 0 {
			t.Errorf("main session: %q is allowed; want blocked", c)
		}
	}
	for _, c := range allowed {
		if vs := judged(t, c, false); len(vs) != 0 {
			t.Errorf("main session: %q is blocked by %+v; want allowed", c, vs)
		}
	}
}

// I0028 S6: one level of unwrap for sh -c, bash -c and git -C. Aliases, eval
// and xargs are not unwrapped, so their text is allowed.
func TestShellGuardOneLevelUnwrap(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"bash -c 'git push origin main'",
		"sh -c 'git push -f origin x'",
		"bash -lc 'git push origin main'",
		"bash -o pipefail -c 'git push origin main'",
		"bash -O extglob -c 'git push origin main'",
		"bash -eo pipefail -c 'git push -f origin x'",
		"git -C ../wt push -f",
		"/usr/bin/git push --force origin x",
		"bash -c 'git -C ../wt push -f'",
	}
	for _, c := range blocked {
		if len(judged(t, c, true)) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	if len(judged(t, "bash -c 'git push origin main'", false)) == 0 {
		t.Errorf("main session: bash -c push to main is allowed; want blocked")
	}
	allowed := []string{
		`eval "git push -f"`,
		"xargs git push -f",
		`bash -c "sh -c 'git push origin main'"`,
	}
	for _, c := range allowed {
		if vs := judged(t, c, true); len(vs) != 0 {
			t.Errorf("coder: %q is blocked by %+v; want allowed (one level only)", c, vs)
		}
	}
}

// Command lines are split and read as the shell reads them: each command of a
// list, pipe, substitution or newline is judged; quoted text, comments and
// arguments of other commands are not.
func TestShellGuardReadsCommandsNotText(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"git status && git push origin main",
		"git status; git push --force origin feat/x",
		"echo ok | git push -f origin x",
		"FOO=1 git push origin main",
		"if git push origin main; then echo ok; fi",
		"echo $(git push origin main)",
		"git status\ngit push origin main",
		"(git push origin main)",
		`echo "$(git push origin main)"`,
		"echo \"`git push -f origin x`\"",
	}
	allowed := []string{
		`echo "git push origin main"`,
		"echo 'git push -f'",
		"# git push origin main",
		"git push origin main:feat/x",
		"grep -r 'rm -rf build/*' .",
		"git log --grep 'push -f'",
		"git status >/dev/null 2>&1",
		"echo x > apps/cli/out.txt",
		"",
	}
	for _, c := range blocked {
		if len(judged(t, c, true)) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	for _, c := range allowed {
		if vs := judged(t, c, true); len(vs) != 0 {
			t.Errorf("coder: %q is blocked by %+v; want allowed", c, vs)
		}
	}
}

// Every violated rule is reported, in command order.
func TestShellGuardReportsEachViolation(t *testing.T) {
	t.Parallel()
	vs := judged(t, "git push origin main && git push -f origin x", true)
	if len(vs) != 2 {
		t.Fatalf("violations = %+v, want two", vs)
	}
	if vs[0].Rule == vs[1].Rule {
		t.Fatalf("both violations name rule %q", vs[0].Rule)
	}
}

// The rm rules judge the recursive flag and the path, and only for a coder.
func TestShellGuardRmNeedsRecursiveAndGlobOrWorktree(t *testing.T) {
	t.Parallel()
	blocked := []string{"rm -r build/*", "rm -fR build/*", "rm --recursive build/*", "rm -rf ../x-worktrees"}
	allowed := []string{"rm -f build/*", "rm build/x.txt"}
	for _, c := range blocked {
		if len(judged(t, c, true)) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	for _, c := range allowed {
		if vs := judged(t, c, true); len(vs) != 0 {
			t.Errorf("coder: %q is blocked by %+v; want allowed", c, vs)
		}
	}
}
