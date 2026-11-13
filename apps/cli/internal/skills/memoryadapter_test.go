package skills

import (
	"fmt"
	"strings"
	"testing"
)

const (
	liveMemoryAdapter    = ".claude/skills/gabo-memory-engram/SKILL.md"
	catalogMemoryExample = "gabo-contract-memory/examples/memory-engram.md"
)

// memoryRecordRow is one row of a Records table: the record name and the
// topic key, the only two cells the comparison reads (intent 0024 PD7).
type memoryRecordRow struct{ name, key string }

// memoryRecordRows returns the rows of the first table of doc whose header
// starts with `| record | topic key |`. The record name is lowered and the
// topic key loses its double quotes and backticks; later columns are ignored.
func memoryRecordRows(doc string) []memoryRecordRow {
	var rows []memoryRecordRow
	in := false
	for _, l := range strings.Split(doc, "\n") {
		l = strings.TrimSpace(l)
		if !in {
			in = strings.HasPrefix(l, "| record | topic key |")
			continue
		}
		if !strings.HasPrefix(l, "|") {
			break
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		if len(cells) < 2 || strings.HasPrefix(strings.TrimSpace(cells[0]), "---") {
			continue
		}
		key := strings.ReplaceAll(strings.TrimSpace(cells[1]), "`", "")
		rows = append(rows, memoryRecordRow{
			name: strings.ToLower(strings.TrimSpace(cells[0])),
			key:  strings.ReplaceAll(key, `"`, ""),
		})
	}
	return rows
}

// compareMemoryRecords returns an error naming the first row that differs
// between the live adapter and the catalog example, and the file it came from.
func compareMemoryRecords(live, catalog string) error {
	l, c := memoryRecordRows(live), memoryRecordRows(catalog)
	if len(c) == 0 {
		return fmt.Errorf("%s: no Records table", catalogMemoryExample)
	}
	if len(l) == 0 {
		return fmt.Errorf("%s: no Records table", liveMemoryAdapter)
	}
	for i := 0; i < len(l) || i < len(c); i++ {
		switch {
		case i >= len(l):
			return fmt.Errorf("row %d (%s) of %s is missing from %s", i+1, c[i].name, catalogMemoryExample, liveMemoryAdapter)
		case i >= len(c):
			return fmt.Errorf("row %d (%s) of %s is missing from %s", i+1, l[i].name, liveMemoryAdapter, catalogMemoryExample)
		case l[i] != c[i]:
			return fmt.Errorf("row %d differs: %s has %q %q, %s has %q %q",
				i+1, liveMemoryAdapter, l[i].name, l[i].key, catalogMemoryExample, c[i].name, c[i].key)
		}
	}
	return nil
}

// Intent 0024 S8 (plan T8, PD7, PD9): the live memory adapter and the catalog
// example agree on record names and topic keys. The test only reads both files.
func TestLiveMemoryAdapterMatchesCatalogExample(t *testing.T) {
	t.Parallel()
	live := readRepoDoc(t, liveMemoryAdapter)
	catalog := readCatalog(t, catalogMemoryExample)
	if err := compareMemoryRecords(live, catalog); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryRecordsComparisonNamesTheDifferingRow(t *testing.T) {
	t.Parallel()
	live := readRepoDoc(t, liveMemoryAdapter)
	catalog := readCatalog(t, catalogMemoryExample)
	changed := strings.Replace(live, "`gabo/ledger/<run id>`", "`gabo/ledgers/<run id>`", 1)
	if changed == live {
		t.Fatal("live adapter no longer holds the ledger topic key")
	}
	err := compareMemoryRecords(changed, catalog)
	if err == nil {
		t.Fatal("changed topic key not detected")
	}
	for _, w := range []string{"row 2", "resource ledger", "gabo/ledgers/<run id>", liveMemoryAdapter, catalogMemoryExample} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q lacks %q", err, w)
		}
	}
	changedCatalog := strings.Replace(catalog, "`user/preferences`", "`user/prefs`", 1)
	if err := compareMemoryRecords(live, changedCatalog); err == nil || !strings.Contains(err.Error(), "row 5") {
		t.Errorf("changed catalog key: got %v", err)
	}
}
