package skills

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"
)

// fixedQuestionsHeader is the one parseable format every plan template uses
// for its fixed questions (TD4, D19).
const fixedQuestionsHeader = "| id | question | answered in |"

// templates lists the plan templates each directiva skill must ship, with the
// table header lines and headings agents and the traceability check rely on.
// These are stable: changing one is a plan change (D19).
var templates = map[string]struct {
	columns  []string
	headings []string
}{
	"gbai-pm/templates/PRD.md": {
		columns: []string{
			"| field | value |",
			"| id | title | user | requirement ids | status | file |",
			"| id | type | requirement | acceptance criteria | priority | stories | verification |",
			"| metric | how it is measured |",
			"| # | question | status |",
			"| term | meaning |",
		},
		headings: []string{
			"# Product Requirements (PRD)",
			"## 2. Problem",
			"### Evidence",
			"## 5. Appetite",
			"## 6. Assumptions and dependencies",
			"## 8. Requirements",
			"## 9. Quality requirements",
			"## 10. Constraints",
			"## 14. Glossary",
		},
	},
	"gbai-pm/templates/STORY.md": {
		columns: []string{"| field | value |"},
		headings: []string{
			"# US-0NN <Title>",
			"## Story",
			"## Acceptance criteria",
			"## Out of scope",
			"## Notes",
		},
	},
}

func embeddedByPath(t *testing.T) map[string]string {
	t.Helper()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range files {
		out[f.Path] = string(f.Data)
	}
	return out
}

func TestEveryTemplateHasParseableFixedQuestions(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	var found []string
	for p, doc := range byPath {
		if path.Base(path.Dir(p)) != "templates" {
			continue
		}
		found = append(found, p)
		qs, err := fixedQuestions(doc)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if len(qs) == 0 {
			t.Errorf("%s: no fixed questions", p)
		}
	}
	for p := range templates {
		if !slices.Contains(found, p) {
			t.Errorf("%s: template missing", p)
		}
	}
}

func TestTemplatesKeepStableColumnsAndSections(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for p, want := range templates {
		doc, ok := byPath[p]
		if !ok {
			t.Errorf("%s: template missing", p)
			continue
		}
		lines := strings.Split(doc, "\n")
		for _, l := range slices.Concat(want.columns, want.headings) {
			if !slices.Contains(lines, l) {
				t.Errorf("%s: missing line %q", p, l)
			}
		}
	}
}

func TestPMSkillPointsToTemplatesAndStoryNaming(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)["gbai-pm/SKILL.md"]
	for _, s := range []string{
		"templates/PRD.md", "templates/STORY.md", "US-0NN-<kebab>.md",
		"| id | type | requirement | acceptance criteria | priority | stories | verification |",
	} {
		if !strings.Contains(skill, s) {
			t.Errorf("gbai-pm/SKILL.md: missing %q", s)
		}
	}
}

func TestFixedQuestionsParser(t *testing.T) {
	t.Parallel()
	good := "## Fixed questions\n\n" + fixedQuestionsHeader + "\n|---|---|---|\n| Q1 | Who? | user |\n| Q2 | Why? | Story |\n\n## Template\n| Q9 | ignored | x |\n"
	qs, err := fixedQuestions(good)
	if err != nil || len(qs) != 2 || qs[1] != [3]string{"Q2", "Why?", "Story"} {
		t.Fatalf("fixedQuestions(good) = %v, %v", qs, err)
	}
	for name, doc := range map[string]string{
		"no section":   "# x\n",
		"wrong header": "## Fixed questions\n\n| id | question |\n|---|---|\n| Q1 | Who? |\n",
		"bad id":       "## Fixed questions\n\n" + fixedQuestionsHeader + "\n|---|---|---|\n| Q2 | Who? | user |\n",
		"empty cell":   "## Fixed questions\n\n" + fixedQuestionsHeader + "\n|---|---|---|\n| Q1 |  | user |\n",
	} {
		if _, err := fixedQuestions(doc); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

// fixedQuestions parses the table under "## Fixed questions": the header
// fixedQuestionsHeader, a separator row, then rows "| Q<n> | question |
// answered in |" numbered from Q1 without gaps, all cells non-empty.
func fixedQuestions(doc string) ([][3]string, error) {
	var rows [][3]string
	in, header, sep := false, false, false
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			in = strings.TrimSpace(line) == "## Fixed questions"
			continue
		}
		if !in || strings.TrimSpace(line) == "" {
			continue
		}
		switch {
		case !header:
			if line != fixedQuestionsHeader {
				return nil, fmt.Errorf("fixed questions header %q, want %q", line, fixedQuestionsHeader)
			}
			header = true
		case !sep:
			if line != "|---|---|---|" {
				return nil, fmt.Errorf("separator %q", line)
			}
			sep = true
		default:
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) != 3 {
				return nil, fmt.Errorf("row %q: want 3 cells", line)
			}
			var r [3]string
			for i, c := range cells {
				if r[i] = strings.TrimSpace(c); r[i] == "" {
					return nil, fmt.Errorf("row %q: empty cell", line)
				}
			}
			if want := fmt.Sprintf("Q%d", len(rows)+1); r[0] != want {
				return nil, fmt.Errorf("row %q: id %q, want %q", line, r[0], want)
			}
			rows = append(rows, r)
		}
	}
	if !header {
		return nil, fmt.Errorf("no %q table under ## Fixed questions", fixedQuestionsHeader)
	}
	return rows, nil
}
