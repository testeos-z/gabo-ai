package check

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// probeRecordPath is the engine probe record of intent 0028 (S1), relative to this package.
const probeRecordPath = "../../../../../docs/product/intents/0028-deterministic-hooks-as-locks/probe.md"

var (
	probeEngines = []string{"claude", "codex", "pi", "opencode"}
	probeItems   = []string{"a", "b", "c", "d"}
	probeCell    = regexp.MustCompile(`^(yes|no|not testable: \S.*)$`)
)

// probeRows returns the cells of each engine row of the probe table, keyed by engine name,
// and the engines that have a second row. A cell may not contain `|`: the split reads it as
// a new cell, and the row then fails the cell count.
func probeRows(doc string) (rows map[string][]string, dups []string) {
	rows = map[string][]string{}
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		name := strings.Trim(cells[0], "`* ")
		if !slices.Contains(probeEngines, name) {
			continue
		}
		if _, seen := rows[name]; seen {
			dups = append(dups, name)
		}
		rows[name] = cells
	}
	return rows, dups
}

// probeRecordProblems lists every way the S1 record falls short: a missing or duplicate
// engine row, a row with the wrong cell count, an empty version, or a cell that is neither
// yes, no nor not testable with a reason. An empty result means the record holds.
func probeRecordProblems(doc string) []string {
	rows, dups := probeRows(doc)
	var problems []string
	for _, engine := range dups {
		problems = append(problems, "probe record has more than one row for engine "+engine)
	}
	for _, engine := range probeEngines {
		cells, ok := rows[engine]
		if !ok {
			problems = append(problems, "probe record lacks a row for engine "+engine)
			continue
		}
		// engine, version, then items (a) to (d)
		if len(cells) != 2+len(probeItems) {
			problems = append(problems, engine+" row has "+strconv.Itoa(len(cells))+" cells, want engine, version and items a to d (a | inside a cell splits it)")
			continue
		}
		if cells[1] == "" {
			problems = append(problems, engine+" row lacks the engine version")
		}
		for i, item := range probeItems {
			if got := cells[2+i]; !probeCell.MatchString(got) {
				problems = append(problems, engine+" item ("+item+") = "+got+", want yes, no or not testable: <reason>")
			}
		}
	}
	return problems
}

// probeS14Missing returns the engines, in probe order, whose item (a) is no but whose record
// lacks a line of its own that reads "S14 applies to <engine>".
func probeS14Missing(doc string) []string {
	rows, _ := probeRows(doc)
	var missing []string
	for _, engine := range probeEngines {
		cells, ok := rows[engine]
		if !ok || len(cells) < 3 || cells[2] != "no" {
			continue
		}
		// A dedicated line for this engine, not a sentence that qualifies or mentions it.
		statement := regexp.MustCompile(`(?m)^S14 applies to ` + regexp.QuoteMeta(engine) + `\.?\s*$`)
		if !statement.MatchString(doc) {
			missing = append(missing, engine)
		}
	}
	return missing
}

// probeFixture is a valid record with one row per engine.
const probeFixture = "| engine | version | (a) | (b) | (c) | (d) |\n" +
	"|---|---|---|---|---|---|\n" +
	"| claude | 2.1 | yes | yes | yes | no |\n" +
	"| codex | 0.1 | yes | yes | yes | no |\n" +
	"| pi | 1 | not testable: emulated only | not testable: emulated only | not testable: typings only | no |\n" +
	"| opencode | 2 | yes | yes | yes | no |\n"

// TestEngineProbeRecordHoldsSixteenCells reads the S1 record: one row per engine,
// the engine version, and items (a) to (d) each as yes, no or not testable: <reason>.
func TestEngineProbeRecordHoldsSixteenCells(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	for _, problem := range probeRecordProblems(string(raw)) {
		t.Error(problem)
	}
}

// TestEngineProbeRecordRejectsBadRecords shows the record check fails on each kind of
// bad record, and passes the valid fixture.
func TestEngineProbeRecordRejectsBadRecords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		doc   string
		valid bool
	}{
		{"valid fixture", probeFixture, true},
		{"cell that is not a result", strings.Replace(probeFixture, "| 2.1 | yes", "| 2.1 | maybe", 1), false},
		{"not testable without a reason", strings.Replace(probeFixture, "| 2.1 | yes", "| 2.1 | not testable:", 1), false},
		{"row with a missing item", strings.Replace(probeFixture, "| 0.1 | yes | yes | yes | no |", "| 0.1 | yes | yes | yes |", 1), false},
		{"empty version", strings.Replace(probeFixture, "| 0.1 | yes", "|  | yes", 1), false},
		{"missing engine row", strings.Replace(probeFixture, "| opencode | 2 | yes | yes | yes | no |\n", "", 1), false},
	}
	for _, tc := range cases {
		problems := probeRecordProblems(tc.doc)
		if tc.valid && len(problems) != 0 {
			t.Errorf("%s: problems = %v, want none", tc.name, problems)
		}
		if !tc.valid && len(problems) == 0 {
			t.Errorf("%s: no problem reported, want one", tc.name)
		}
	}
}

// TestEngineProbeRowsReportsDuplicateRow: a second row for an engine is reported, not replaced silently.
func TestEngineProbeRowsReportsDuplicateRow(t *testing.T) {
	t.Parallel()
	doc := "| engine | version | (a) | (b) | (c) | (d) |\n" +
		"|---|---|---|---|---|---|\n" +
		"| claude | 2.1 | yes | yes | yes | no |\n" +
		"| claude | 2.2 | no | no | no | no |\n"
	_, dups := probeRows(doc)
	if !slices.Equal(dups, []string{"claude"}) {
		t.Errorf("duplicate rows = %v, want [claude]", dups)
	}
}

// TestEngineProbeS14StatementNeedsItemANo checks the S14 rule on fixtures: a no on (a)
// without its own S14 line is reported, a qualified sentence does not count, and the line clears it.
func TestEngineProbeS14StatementNeedsItemANo(t *testing.T) {
	t.Parallel()
	table := "| engine | version | (a) | (b) | (c) | (d) |\n" +
		"|---|---|---|---|---|---|\n" +
		"| claude | 2.1 | no | yes | yes | no |\n" +
		"| codex | 0.1 | yes | yes | yes | no |\n"
	if got := probeS14Missing(table); !slices.Equal(got, []string{"claude"}) {
		t.Errorf("missing S14 statements = %v, want [claude]", got)
	}
	if got := probeS14Missing(table + "\nS14 applies to claude only if the probe says so.\n"); !slices.Equal(got, []string{"claude"}) {
		t.Errorf("qualified sentence counted: missing = %v, want [claude]", got)
	}
	if got := probeS14Missing(table + "\nS14 applies to claude.\n"); len(got) != 0 {
		t.Errorf("missing S14 statements = %v, want none once the line is present", got)
	}
}

// TestEngineProbeNoOnItemANamesS14 requires every engine whose item (a) is no to
// state "S14 applies to <engine>" on its own line in the record, so the plan knows the fallback applies to it.
func TestEngineProbeNoOnItemANamesS14(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	for _, engine := range probeS14Missing(string(raw)) {
		t.Errorf("%s has item (a) no but the record lacks the line %q", engine, "S14 applies to "+engine+".")
	}
}
