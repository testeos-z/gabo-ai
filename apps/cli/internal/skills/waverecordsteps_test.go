package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const waveRecord = "`sh .gabo/skills/gabo-wave-report/scripts/wave-record.sh --wave <run id>"

// failureRule is the sentence every skill carries for a missing or failing
// wave-record.sh (intent 0024 S1).
const failureRule = "report `unverified: wave-record <subcommand> failed (<reason>)`, continue the wave and never write the line the script would print"

// section returns doc from the first occurrence of from up to the first later
// occurrence of to (the end of doc when to is empty).
func section(t *testing.T, doc, from, to string) string {
	t.Helper()
	i := strings.Index(doc, from)
	if i < 0 {
		t.Fatalf("missing %q", from)
	}
	rest := doc[i:]
	if to == "" {
		return rest
	}
	j := strings.Index(rest[len(from):], to)
	if j < 0 {
		t.Fatalf("missing %q after %q", to, from)
	}
	return rest[:len(from)+j]
}

// Intent 0024 plan T3 (S1, S2, S4): the cycle runs the stamps and the saves
// ledger of wave-record.sh and pastes the output unchanged.
func TestCycleRunsWaveRecordStamps(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	steps := section(t, doc, "\n2. Intent claim", "\n3. Resume")
	requireIn(t, "gabo-cycle step 2", steps, []string{waveRecord + " stamp wave-started`", "paste each output unchanged", "On a wave start only (never when the holder resumes)",
		waveRecord + " stamp session-started gabo-assistant main`", "the stamps are kept per run id"})
	launch := section(t, doc, "\n5. Per task:", "\n6. On each outcome")
	requireIn(t, "gabo-cycle step 5", launch, []string{
		waveRecord + " stamp session-started <agent> <task id>` before the launch call",
		"a coder resumed by its handle runs none",
	})
	outcome := section(t, doc, "\n6. On each outcome", "\n7. The wave closes")
	requireIn(t, "gabo-cycle step 6", outcome, []string{
		waveRecord + " stamp pr-opened <pr>`", waveRecord + " stamp pr-merged <pr>`", "paste its output unchanged",
	})
	report := section(t, doc, "\n8. Report:", "\n## Output Contract")
	requireIn(t, "gabo-cycle step 8", report, []string{waveRecord + " stamp wave-reported`"})
	step3 := section(t, doc, "\n3. Resume (D17)", "\n4. Read this intent")
	requireIn(t, "gabo-cycle step 3", step3, []string{"stamped as in the Trace rule"})
	trace := section(t, doc, "- Trace (D17)", "\n\n")
	requireIn(t, "gabo-cycle Trace", trace, []string{
		waveRecord + " stamp session-started <agent> <task id>` (step 5, a replacement coder of step 3 or of the turn-cap resume included, none for a resume by a coder's handle)",
		"${XDG_STATE_HOME:-$HOME/.local/state}/gabo/waves/<run id>/stamps",
		"as one more entry of `gabo/trace/<run id>`, never replacing the entries above",
		"adds the content of the stamps file (",
		waveRecord + " saves add '<topic key>#<entry>' <observation id>` after every memory write of the wave",
		waveRecord + " saves read '<topic key>#<entry>'` after each read-back whose observation shows that entry's text",
	})
	requireIn(t, "gabo-cycle", doc, []string{failureRule})
}

func TestWaveReportMeasuresAndSavesComeFromTheScript(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	measures := section(t, doc, "9. **Measures**", "\n10. ")
	requireIn(t, "gabo-wave-report Measures", measures, []string{
		"the wall time line, the agent sessions lines (one per agent, in the script's format",
		"are the lines of " + waveRecord + " measures` pasted unchanged",
		"each at its own place in the order above (the block may be split there)",
		"the other lines (fix rounds per task, plan-change proposals, interactions, tokens, each task's outcome) are still built as below",
		"never a figure with `~`",
	})
	unverified := section(t, doc, "5. **Unverified writes**", "\n6. ")
	requireIn(t, "gabo-wave-report Unverified writes", unverified, []string{
		"the output of " + waveRecord + " saves unverified`",
		"the final answer is not written before it ran",
	})
	requireIn(t, "gabo-wave-report", doc, []string{failureRule})
}

func TestAssistantWaveCloseRunsSavesUnverified(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	step1 := section(t, doc, "\n1. Before anything else, read", "\n2. **In the first wave report**")
	requireIn(t, "gabo-assistant Wave Close step 1", step1, []string{
		"run " + waveRecord + " saves unverified`",
		"before the final answer",
		failureRule,
	})
}

// Every command the skills document must pass the script's own argument checks
// (wave id, the event table, saves): the review of PR 684 found a missing
// --wave that wording tests cannot see.
func TestDocumentedWaveRecordCommandsRun(t *testing.T) {
	t.Parallel()
	script := filepath.Join(t.TempDir(), "wave-record.sh")
	if err := os.WriteFile(script, []byte(readCatalog(t, "gabo-wave-report/scripts/wave-record.sh")), 0o755); err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("`sh \\.gabo/skills/gabo-wave-report/scripts/wave-record\\.sh ([^`]*)`")
	repl := strings.NewReplacer("<run id>", "w1", "<agent>", "gabo-coder", "<task id>", "T1", "<pr>", "7", "'<topic key>#<entry>'", "k#start", "'<topic key>#start'", "k#start", "<topic key>", "k", "<observation id>", "1", "<subcommand>", "measures")
	seen := 0
	for _, name := range []string{"gabo-cycle", "gabo-wave-report", "gabo-assistant"} {
		for _, m := range re.FindAllStringSubmatch(readCatalog(t, name+"/SKILL.md"), -1) {
			args := strings.Fields(repl.Replace(m[1]))
			if len(args) > 0 && args[0] == "caps" {
				continue // caps takes no --wave and is covered by waverecordcaps_test.go
			}
			if len(args) < 3 || args[0] != "--wave" {
				t.Errorf("%s: %q does not start with --wave <run id>", name, m[1])
				continue
			}
			seen++
			state := t.TempDir()
			env := append(os.Environ(), "XDG_STATE_HOME="+state)
			for _, key := range []string{"k", "k#start"} {
				seed := exec.Command("sh", script, "--wave", "w1", "saves", "add", key, "1")
				seed.Env = env
				if out, err := seed.CombinedOutput(); err != nil {
					t.Fatalf("seed: %v: %s", err, out)
				}
			}
			cmd := exec.Command("sh", append([]string{script}, args...)...)
			cmd.Env = env
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s: %q fails: %v: %s", name, m[1], err, out)
			}
		}
	}
	if seen < 9 {
		t.Errorf("found %d documented commands, want at least 9", seen)
	}
}
