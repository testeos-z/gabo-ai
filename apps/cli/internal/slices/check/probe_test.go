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

// probeRows returns the cells of each engine row of the probe table, keyed by engine name.
func probeRows(doc string) map[string][]string {
	rows := map[string][]string{}
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		name := strings.Trim(cells[0], "`* ")
		if slices.Contains(probeEngines, name) {
			rows[name] = cells
		}
	}
	return rows
}

// TestEngineProbeRecordHoldsSixteenCells reads the S1 record: one row per engine,
// the engine version, and items (a) to (d) each as yes, no or not testable: <reason>.
func TestEngineProbeRecordHoldsSixteenCells(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	rows := probeRows(string(raw))
	for _, engine := range probeEngines {
		cells, ok := rows[engine]
		if !ok {
			t.Errorf("probe record lacks a row for engine %s", engine)
			continue
		}
		// engine, version, then items (a) to (d)
		if len(cells) != 2+len(probeItems) {
			t.Errorf("%s row has %d cells, want engine, version and items a to d", engine, len(cells))
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

// TestEngineProbeNoOnItemANamesS14 requires every engine whose item (a) is no to
// name S14 in the record, so the plan knows the fallback applies to that engine.
func TestEngineProbeNoOnItemANamesS14(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.FromSlash(probeRecordPath))
	if err != nil {
		t.Fatalf("read probe record: %v", err)
	}
	doc := string(raw)
	for engine, cells := range probeRows(doc) {
		if len(cells) < 3 || cells[2] != "no" {
			continue
		}
		named := false
		for _, line := range strings.Split(doc, "\n") {
			if strings.Contains(line, engine) && strings.Contains(line, "S14") {
				named = true
				break
			}
		}
		if !named {
			t.Errorf("%s has item (a) no but the record does not name S14 for it", engine)
		}
	}
}
