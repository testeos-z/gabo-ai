package check

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// TestEngineProbeRecordHoldsSixteenCells reads the S1 record: one row per engine,
// the engine version, and items (a) to (d) each as yes, no or not testable: <reason>.
func TestEngineProbeRecordHoldsSixteenCells(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	rows, dups := probeRows(string(raw))
	for _, engine := range dups {
		t.Errorf("probe record has more than one row for engine %s", engine)
	}
	for _, engine := range probeEngines {
		cells, ok := rows[engine]
		if !ok {
			t.Errorf("probe record lacks a row for engine %s", engine)
			continue
		}
		// engine, version, then items (a) to (d)
		if len(cells) != 2+len(probeItems) {
			t.Errorf("%s row has %d cells, want engine, version and items a to d (a | inside a cell splits it)", engine, len(cells))
			continue
		}
		if cells[1] == "" {
			t.Errorf("%s row lacks the engine version", engine)
		}
		for i, item := range probeItems {
			if got := cells[2+i]; !probeCell.MatchString(got) {
				t.Errorf("%s item (%s) = %q, want yes, no or not testable: <reason>", engine, item, got)
			}
		}
	}
	if len(rows) != len(probeEngines) {
		t.Errorf("probe record has %d engine rows, want %d", len(rows), len(probeEngines))
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

// TestEngineProbeNoOnItemANamesS14 requires every engine whose item (a) is no to
// state "S14 applies to <engine>" in the record, so the plan knows the fallback applies to it.
func TestEngineProbeNoOnItemANamesS14(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	doc := string(raw)
	rows, _ := probeRows(doc)
	for engine, cells := range rows {
		if len(cells) < 3 || cells[2] != "no" {
			continue
		}
		// A positive statement for this engine, not any line that mentions it.
		statement := regexp.MustCompile(`S14 applies to ` + regexp.QuoteMeta(engine) + `\b`)
		if !statement.MatchString(doc) {
			t.Errorf("%s has item (a) no but the record lacks the statement %q", engine, "S14 applies to "+engine)
		}
	}
}
