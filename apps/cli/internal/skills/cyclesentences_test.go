package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sentenceEnd splits a skill text at sentence ends: a full stop, question or
// exclamation mark followed by white space, and at every line break.
var sentenceEnd = regexp.MustCompile(`[.!?]\s+|\n`)

// sentences returns the trimmed sentences of doc, in order.
func sentences(doc string) []string {
	var out []string
	rest := doc
	for {
		loc := sentenceEnd.FindStringIndex(rest)
		if loc == nil {
			break
		}
		end := loc[0]
		if rest[loc[0]] != '\n' {
			end++ // keep the stop in the sentence
		}
		out = append(out, strings.TrimSpace(rest[:end]))
		rest = rest[loc[1]:]
	}
	return append(out, strings.TrimSpace(rest))
}

// Intent 0025 S5 (plan T4): every sentence of `gabo-cycle` of 50 or more
// characters appears once; the first repeat is named in the failure.
func TestCycleSentencesAppearOnce(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-cycle/SKILL.md")
	b, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-cycle/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	codex := string(b)
	if catalog != codex {
		t.Fatal("plugins/codex/skills/gabo-cycle/SKILL.md differs from the catalog file")
	}
	for _, doc := range []string{catalog, codex} {
		seen := map[string]bool{}
		for _, s := range sentences(doc) {
			if len(s) < 50 {
				continue
			}
			if seen[s] {
				t.Fatalf("gabo-cycle/SKILL.md: sentence occurs twice: %q", s)
			}
			seen[s] = true
		}
	}
}

// The three sentences that were repeated stay once, in the step that runs
// them (intent 0025 S5: no rule changes).
func TestCycleKeepsTheOnceSentencesInTheirStep(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"The entry saved to `gabo/trace/<run id>` at the report step holds the whole output of `cat` of the stamps file",
		"Each coder's `session started <agent> <task id> <UTC time>` line is saved to the trace in the same step as its `stamp session-started`",
		"The wave start entry save is followed by `sh .gabo/skills/gabo-wave-report/scripts/wave-record.sh --wave <run id> saves add '<topic key>#start' <observation id>`",
	})
}
