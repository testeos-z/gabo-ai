package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0023 S1, D2 (plan T2, finding #562): the main session on Pi that is
// asked to start a wave under `--print` refuses only when a mechanism can read
// the mode; today none can (T1 #585), so every Pi main session prints the
// warning once, before the preflight and the claim, and continues. The exact
// strings are the spec's; they live in the Hard Rules item below. Wave Start
// names no engine (assistant_test.go pins that), so it only points at the item.

const (
	piPrintWarning = "warning: could not tell whether this Pi session runs with --print, which ends after one turn and kills the coder subagents; use an interactive session or pi --mode rpc"
	piPrintRefusal = "wave not started: Pi runs with --print, which ends after one turn and kills the coder subagents; start the wave in an interactive Pi session or with pi --mode rpc"
	piPrintHead    = "- **Print-mode check"
)

// piPrintItem returns the Hard Rules item that holds the print-mode check, or
// "" when the item is absent, so every assertion on it fails.
func piPrintItem(doc string) string {
	_, hard, _ := strings.Cut(doc, "## Hard Rules")
	hard, _, _ = strings.Cut(hard, "\n## ")
	for _, line := range strings.Split(hard, "\n") {
		if strings.HasPrefix(line, piPrintHead) {
			return line
		}
	}
	return ""
}

// The warning is printed once, before the preflight and the intent claim, and
// the session continues.
func TestPiMainSessionPrintsTheWarningOnceBeforeTheClaim(t *testing.T) {
	t.Parallel()
	item := piPrintItem(readCatalog(t, "gabo-assistant/SKILL.md"))
	requireIn(t, "gabo-assistant Hard Rules print-mode check", item, []string{
		piPrintWarning,
		"once, before the preflight and the intent claim",
		"and continues",
	})
}

// The refusal is kept only as conditional on a mechanism that reads the mode,
// and the skill says that none exists today. It claims and launches nothing.
func TestPiMainSessionRefusesOnlyWhenTheModeIsReadable(t *testing.T) {
	t.Parallel()
	item := piPrintItem(readCatalog(t, "gabo-assistant/SKILL.md"))
	requireIn(t, "gabo-assistant Hard Rules print-mode check", item, []string{
		piPrintRefusal,
		"claim nothing and launch nothing",
		"No mechanism reads the session mode today (T1 #585)",
	})
	cond := strings.Index(item, "Only if a mechanism reads the mode")
	refusal := strings.Index(item, piPrintRefusal)
	if cond < 0 || refusal < 0 || cond > refusal {
		t.Errorf("gabo-assistant Hard Rules print-mode check: the refusal must follow the readable-mode condition (condition at %d, refusal at %d)", cond, refusal)
	}
}

// Wave Start runs the check before its first step, which is the one that
// reads the intent and loads the cycle that claims it.
func TestPiPrintCheckRunsBeforeWaveStart(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	_, start, ok := strings.Cut(doc, "## Wave Start (J3)")
	if !ok {
		t.Fatal("gabo-assistant/SKILL.md: no Wave Start section")
	}
	start, _, _ = strings.Cut(start, "\n## ")
	check := strings.Index(start, "print-mode check")
	step1 := strings.Index(start, "1. Pick the wave")
	if check < 0 || step1 < 0 || check > step1 {
		t.Errorf("gabo-assistant Wave Start: the print-mode check must come before step 1 (check at %d, step 1 at %d)", check, step1)
	}
}

// The check is scoped to Pi: the engine line names it, and the other engines
// print nothing, so their behaviour is unchanged.
func TestPiPrintCheckIsScopedToPi(t *testing.T) {
	t.Parallel()
	item := piPrintItem(readCatalog(t, "gabo-assistant/SKILL.md"))
	requireIn(t, "gabo-assistant Hard Rules print-mode check", item, []string{
		"your session engine is Pi (the fixed session-engine line of your block)",
		"Claude Code, Codex and OpenCode print nothing",
		"Once a mechanism reads the mode, a session in `rpc` or interactive mode prints no line",
	})
}

// The Codex mirror (PD10) stays byte-equal to the catalog source.
func TestPiPrintCheckMirrorsTheCodexPlugin(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-assistant/SKILL.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-assistant/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != catalog {
		t.Error("plugins/codex/skills/gabo-assistant/SKILL.md differs from the catalog source (PD10)")
	}
}
