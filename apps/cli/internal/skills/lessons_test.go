package skills

import (
	"os"
	"strings"
	"testing"
)

// Spec 0030 S14 and S17, decisions D7 and D8: the lessons file is the one
// versioned place for incidents and for mistakes seen once. These tests read
// the real document, which holds the templates and no invented incident.
const lessonsDoc = "../../../../docs/product/LESSONS.md"

// lessonsEntryFields are the lines every incident entry carries (S14).
var lessonsEntryFields = []string{
	"- date:",
	"- task and PR:",
	"- what was reported:",
	"- what was found wrong:",
	"- cause:",
	"- draft intent:",
	"- eval task:",
}

// lessonsOccurrenceFields let a second occurrence of a mistake seen once be
// recognized (S17, D8).
var lessonsOccurrenceFields = []string{
	"- first occurrence (task and PR):",
	"- second occurrence (task and PR):",
}

func readLessons(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(lessonsDoc)
	if err != nil {
		t.Fatalf("read %s: %v", lessonsDoc, err)
	}
	return string(b)
}

// lessonsSection returns the lines between "## <name>" and the next "## "
// heading.
func lessonsSection(t *testing.T, doc, name string) string {
	t.Helper()
	var out []string
	found := false
	in := false
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			in = line == "## "+name
			found = found || in
			continue
		}
		if in {
			out = append(out, line)
		}
	}
	if !found {
		t.Fatalf("section %q missing from %s", name, lessonsDoc)
	}
	return strings.Join(out, "\n")
}

// lessonsEntries splits a section into its "### " entries.
func lessonsEntries(section string) []string {
	var entries []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "### ") {
			entries = append(entries, "")
		}
		if len(entries) > 0 {
			entries[len(entries)-1] += line + "\n"
		}
	}
	return entries
}

// Spec 0030 S14: the entry template names every field of an incident.
func TestLessonsEntryTemplateHasEveryField(t *testing.T) {
	t.Parallel()
	tpl := lessonsSection(t, readLessons(t), "Entry template")
	for _, f := range lessonsEntryFields {
		if !strings.Contains(tpl, f) {
			t.Errorf("entry template lacks %q", f)
		}
	}
	for _, want := range []string{"cause: unknown", "eval task: pending fix"} {
		if !strings.Contains(tpl, want) {
			t.Errorf("entry template lacks %q", want)
		}
	}
}

// Spec 0030 S17, D8: mistakes seen once have their own section, with the
// occurrence fields and the two-times rule.
func TestLessonsSeenOnceSectionKeepsOccurrenceFields(t *testing.T) {
	t.Parallel()
	seen := lessonsSection(t, readLessons(t), "Mistakes seen once")
	for _, f := range lessonsOccurrenceFields {
		if !strings.Contains(seen, f) {
			t.Errorf("seen-once section lacks %q", f)
		}
	}
	if !strings.Contains(seen, "two-times rule") {
		t.Error("seen-once section does not state the two-times rule")
	}
}

// Spec 0030 S14 and S17: every recorded entry carries its fields; an empty
// section holds no invented entry.
func TestLessonsEntriesCarryTheirFields(t *testing.T) {
	t.Parallel()
	doc := readLessons(t)
	cases := []struct {
		section string
		fields  []string
	}{
		{"Incidents", lessonsEntryFields},
		{"Mistakes seen once", lessonsOccurrenceFields},
	}
	for _, c := range cases {
		for i, e := range lessonsEntries(lessonsSection(t, doc, c.section)) {
			for _, f := range c.fields {
				if !strings.Contains(e, f) {
					t.Errorf("%s entry %d lacks %q", c.section, i+1, f)
				}
			}
		}
	}
}
