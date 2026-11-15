package check

import "testing"

// I0028 S7, S13 (review round 3): the project root, and a folder that contains
// a human-only entry, are human-only as the target of cp, mv or rm. Each
// reviewer example is pinned.
func TestHumanOnlyShellWritesRootAndFolderAboveEntry(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"cp /tmp/x/gabo.yaml .",
		"mv /tmp/payload/AGENTS.md ./",
		"cp -t . /tmp/gabo.yaml",
		"rm -rf .",
		"rm -rf .github/..",
		"rm -rf ./",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	// Only cp, mv and rm judge a folder above an entry: a redirection or tee
	// into a folder does not, and a folder with no entry below it is free.
	allowed := []string{"rm -rf build", "cp x apps/y", "rm -rf apps"}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %q; want allowed", c, got)
		}
	}
}

// I0028 S7: a folder that contains a human-only entry of the list is
// human-only as the target of rm or mv, and its own entries stay as they are.
func TestHumanOnlyShellWritesFolderAboveListedEntry(t *testing.T) {
	t.Parallel()
	u := humanUseCase(Settings{Root: "/proj"})
	list := []string{"docs/product/"}
	blocked := []string{"rm -rf docs", "rm -rf docs/", "rm -rf ./docs", "mv x docs"}
	for _, c := range blocked {
		if got := u.humanOnlyShellWrites(c, true, list); len(got) == 0 {
			t.Errorf("%q is allowed; want blocked", c)
		}
	}
	allowed := []string{"rm -rf docs/other", "rm -rf docs/policies"}
	for _, c := range allowed {
		if got := u.humanOnlyShellWrites(c, true, list); len(got) != 0 {
			t.Errorf("%v is blocked by %v; want allowed", c, got)
		}
	}
}

// I0028 S7 (review round 3): a cd earlier on the line sets the folder later
// relative paths are read from; a cd the guard cannot read blocks the writes
// after it on that line.
func TestHumanOnlyShellWritesFollowsCd(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"cd .github && tee ci.yml",
		"cd .claude; sed -i s/a/b/ settings.json",
		"cd apps && cp x ../gabo.yaml",
		"pushd .github && tee ci.yml",
		"cd $DIR && tee x",
		"cd && echo x > a.txt",
		"cd - && tee x",
		"cd {a,b} && tee x",
		"cd .github && bash -c 'tee ci.yml'",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	allowed := []string{
		"cd apps && tee out.txt",
		"cd apps && cp x y",
		"cd .. && echo x > .github/x",
		"cd apps && echo x > apps/out.txt",
	}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %q; want allowed", c, got)
		}
	}
}

// I0028 S6 (review round 3): timeout, stdbuf, doas, ionice and chrt are command
// wrappers like env and sudo, and timeout and chrt take a number first.
func TestHumanOnlyShellWritesMoreWrappers(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"timeout 5 tee .github/a",
		"stdbuf -o0 tee AGENTS.md",
		"doas tee .github/a",
		"ionice -c 3 tee .github/a",
		"chrt 10 tee .github/a",
		"chrt -f 10 tee .github/a",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
}

// I0028 S7 (review round 3): sed's script is not a path, and -f reads its file.
func TestHumanOnlyShellWritesSedScript(t *testing.T) {
	t.Parallel()
	allowed := []string{
		"sed -i -f AGENTS.md out.txt",
		"sed -i 's/a/AGENTS.md/' out.txt",
		"sed -i -e 's/a/b/' out.txt",
	}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %q; want allowed", c, got)
		}
	}
	blocked := []string{"sed -i -e 's/a/b/' AGENTS.md", "sed -i -es/a/b/ CLAUDE.md"}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
}

// I0028 S7 (review round 4): a target that resolves outside the root but
// contains it (.., /, a cd up and back to the project) is the root's ancestor,
// so cp, mv and rm of it are human-only; a path beside the project is not.
func TestHumanOnlyShellWritesOutsideAncestorOfRoot(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"rm -rf ..",
		"rm -rf /",
		"mv .. /tmp/x",
		"cd .. && rm -rf proj",
		"cp x ..",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	allowed := []string{"rm -rf /tmp/x", "cd .. && rm -rf other", "rm -rf ../x-worktrees/y/z"}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %v; want allowed", c, got)
		}
	}
}

// I0028 S6, S7 (review round 4): each wrapper has its own value options. sudo -n
// is a flag; env -C, env --chdir and sudo -D, sudo --chdir change the folder the
// command runs in, as cd does; timeout -s and -k take a value.
func TestHumanOnlyShellWritesWrapperOptionsPerWrapper(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"sudo -n tee AGENTS.md",
		"env -C .github tee ci.yml",
		"env --chdir=.github tee ci.yml",
		"env --chdir .github tee ci.yml",
		"env -C.github tee ci.yml",
		"sudo -D .github tee ci.yml",
		"sudo --chdir=.github tee ci.yml",
		"timeout -s KILL 5 tee .github/a",
		"timeout --signal=KILL 5 tee .github/a",
		"timeout -k 1 5 tee .github/a",
		"timeout --kill-after 1 5 tee .github/a",
		"env -C $DIR tee x",
	}
	for _, c := range blocked {
		if got := humanWrites(c, true); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	allowed := []string{
		"sudo -n echo x > apps/out.txt",
		"timeout -s KILL 5 echo x > apps/x",
		"env -C apps tee out.txt",
		"sudo -D apps tee out.txt",
	}
	for _, c := range allowed {
		if got := humanWrites(c, true); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %v; want allowed", c, got)
		}
	}
}
