package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// taskCloseStep4 returns step 4 of the Task Close of gabo-coder, the delete
// of the task branch, and checks that the Codex copy matches the catalog.
func taskCloseStep4(t *testing.T) string {
	t.Helper()
	catalog := readCatalog(t, "gabo-coder/SKILL.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-coder/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != catalog {
		t.Error("plugins/codex/skills/gabo-coder/SKILL.md differs from the catalog source (S4, PD10)")
	}
	_, closing, ok := strings.Cut(catalog, "## Task Close")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Task Close section")
	}
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	return lineWith(t, closing, "4. Delete the branch")
}

// Intent 0026 S4 (plan T6, D4): in mode: auto the Task Close deletes the
// remote branch of the merged task the workflow created, after the merge.
func TestCoderTaskCloseDeletesTheMergedTaskBranchInAutoMode(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-coder Task Close step 4", taskCloseStep4(t), []string{
		"the local branch always",
		"in `mode: auto`, after the merge, the remote branch of the merged task the workflow created",
		"through `repository_host` (delete branch)",
	})
}

// Intent 0026 S4 (plan T6): the delete never reaches any other remote branch.
func TestCoderTaskCloseForbidsAnyOtherRemoteBranch(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-coder Task Close step 4", taskCloseStep4(t), []string{
		"It never deletes any other remote branch nor a branch the workflow did not create.",
	})
}

// Intent 0026 S4 (plan T6, D4): the 0023 S9 sentence that gentle-pi asks
// before deleting a remote branch even in autonomous mode is replaced by the
// observed behavior (I0023 T15, #599), in the runtime skill and in the Pi
// section of docs/compatibility.md.
const (
	observedDelete = "gentle-pi deleted a remote branch without asking under `autonomousMode: true`"
	observedSource = "https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718"
	oldDeleteLine  = "gentle-pi asks before deleting a remote branch, even in autonomous mode"
)

func TestRuntimePiStatesTheObservedRemoteBranchDelete(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-runtime-pi/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != doc {
		t.Error("plugins/codex/skills/gabo-runtime-pi/SKILL.md differs from the catalog source (S4, PD10)")
	}
	_, section, ok := strings.Cut(doc, "\n## Grants for an unattended Pi wave\n")
	if !ok {
		t.Fatalf("%s: no ## Grants for an unattended Pi wave section", runtimePi)
	}
	section, _, _ = strings.Cut(section, "\n## ")
	requireIn(t, runtimePi+" Pi notes", section, []string{observedDelete, observedSource})
	if strings.Contains(section, oldDeleteLine) {
		t.Errorf("%s: still states the 0023 S9 sentence %q (S4)", runtimePi, oldDeleteLine)
	}
}

func TestCompatibilityPiSectionStatesTheObservedRemoteBranchDelete(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs/compatibility.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, pi, ok := strings.Cut(string(b), "\n## Pi package\n")
	if !ok {
		t.Fatal("docs/compatibility.md: no ## Pi package section")
	}
	pi, _, _ = strings.Cut(pi, "\n## ")
	requireIn(t, "docs/compatibility.md Pi section", pi, []string{observedDelete, observedSource, "`mode: auto`"})
	if strings.Contains(pi, oldDeleteLine) {
		t.Errorf("docs/compatibility.md Pi section: still states the 0023 S9 sentence %q (S4)", oldDeleteLine)
	}
}

// Intent 0026 S5 (plan T6): the grant notes of gabo-runtime-pi name four
// outside paths, the fourth being the wave's stamps folder, say the grant is
// the owner's and given in the session prompt, and that gabo writes no grant
// and no permission to the user's engine config.
func TestRuntimePiGrantsNameTheFourOutsidePaths(t *testing.T) {
	t.Parallel()
	_, section, ok := strings.Cut(runtimePiDoc(t), "\n## Grants for an unattended Pi wave\n")
	if !ok {
		t.Fatalf("%s: no ## Grants for an unattended Pi wave section", runtimePi)
	}
	section, _, _ = strings.Cut(section, "\n## ")
	requireIn(t, runtimePi+" Grants section", section, []string{
		"four grants an unattended Pi wave needs",
		"`worktree.dir`",
		"`${XDG_CONFIG_HOME:-~/.config}/gabo/config.yaml`, which must exist before a wave",
		"`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/waves`, the stamps folder of the wave",
		"in the session prompt",
		"The grant is the owner's",
		"gabo writes no grant file",
		"writes no permission to the user's engine config",
	})
	if strings.Contains(section, "three grants") {
		t.Errorf("%s: Grants section still says three grants (S5)", runtimePi)
	}
}

// Intent 0026 S5 (plan T6): the Pi section of docs/compatibility.md carries
// the same four paths and the same owner and no-grant statements.
func TestCompatibilityPiSectionGrantsNameTheFourOutsidePaths(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs/compatibility.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, pi, ok := strings.Cut(string(b), "\n## Pi package\n")
	if !ok {
		t.Fatal("docs/compatibility.md: no ## Pi package section")
	}
	pi, _, _ = strings.Cut(pi, "\n## ")
	requireIn(t, "docs/compatibility.md Pi section", pi, []string{
		"Four paths lie outside the project and need a grant for the session",
		"`${XDG_CONFIG_HOME:-~/.config}/gabo/config.yaml`",
		"`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer`",
		"`${XDG_STATE_HOME:-~/.local/state}/gabo/waves`, the stamps folder of the wave",
		"the grant is yours to give, in the session prompt",
		"gabo writes no grant file and no permission to your engine config",
	})
	if strings.Contains(pi, "Three paths lie outside the project") {
		t.Error("docs/compatibility.md Pi section: still says three paths (S5)")
	}
}

// Intent 0026 S4 and S5 (plan T6): the Sources of gabo-runtime-pi are checked
// against the installed versions, Pi 1.5.2 and gentle-pi 4.7.0, and dated.
func TestRuntimePiSourcesAreCheckedAgainstTheInstalledVersions(t *testing.T) {
	t.Parallel()
	requireIn(t, runtimePi+" Sources", piSectionOf(t, runtimePiDoc(t), "## Sources"), []string{
		"Checked 2026-11-13 against Pi 1.5.2 and gentle-pi 4.7.0",
		"gentle-engram 0.7.0 `index.ts` `ENGRAM_TOOLS`",
		"Pi 1.5.2 `dist/core/resource-loader.js` (line 130)",
	})
}

// piSectionOf returns the body of a "## " section of doc, up to the next one.
func piSectionOf(t *testing.T, doc, head string) string {
	t.Helper()
	_, body, ok := strings.Cut(doc, "\n"+head+"\n")
	if !ok {
		t.Fatalf("no %q section", head)
	}
	body, _, _ = strings.Cut(body, "\n## ")
	return body
}
