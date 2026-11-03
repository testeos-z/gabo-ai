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
	"gabo-pm/templates/PRD.md": {
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
	// Per-change folder docs/product/intents/<NNNN-slug>/ (S7): the intent
	// holds the problem only; the spec holds the design of one change, its
	// optional UX/app flow (S5) and migration steps (S4), and a human acceptance
	// record (S14).
	"gabo-pm/templates/intent.md": {
		columns: []string{
			"| field | value |",
			"| # | question | status |",
			"| kind | reference |",
		},
		headings: []string{
			"# Intent <NNNN> <Title>",
			"## Problem",
			"## Who is affected",
			"## Desired outcome",
			"## Constraints",
			"## Open questions",
			"## Links",
		},
	},
	"gabo-pm/templates/spec.md": {
		columns: []string{
			"| field | value |",
			"| id | requirement | acceptance criteria | verification |",
			"| id | decision | options considered | choice | consequences |",
			"| surface | entry points | primary action | empty and error states |",
			"| order | file | purpose | rollback |",
			"| policy | conflict | resolution |",
			"| # | question | status |",
			"| accepted by | date | notes |",
		},
		headings: []string{
			"# Spec <NNNN> <Title>",
			"## 1. Requirements",
			"## 2. Design",
			"## 3. UX and app flow (optional)",
			"## 4. DB migration steps (optional)",
			"## 5. Policy conflicts",
			"## 6. Open questions",
			"## 7. Acceptance",
		},
	},
	// T028: arc42 sections, the MADR decision table and the requirement coverage table.
	"gabo-dev-lead/templates/TRD.md": {
		columns: []string{
			"| field | value |",
			"| quality goal | scenario | requirement ids |",
			"| constraint | kind | source |",
			"| id | decision | options considered | choice | consequences | status | requirement ids |",
			"| req | decisions | sections | stories | verification → test layer |",
			"| id | risk or debt | requirement and decision ids | mitigation |",
			"| term | meaning |",
		},
		headings: []string{
			"# Technical Requirements (TRD)",
			"## 1. Introduction and quality goals",
			"## 2. Constraints",
			"## 3. Context and scope",
			"## 4. Solution strategy",
			"## 5. Building block view",
			"## 6. Runtime view",
			"## 7. Deployment view",
			"## 8. Crosscutting concepts",
			"## 9. Architecture decisions",
			"### Requirement coverage",
			"## 10. Quality requirements",
			"### Testing strategy",
			"## 11. Risks and technical debt",
			"## 12. Glossary",
			"## 13. Open questions",
		},
	},
	// The plan of one intent (S9): ordered steps and the task table, no
	// milestones and no global waves; depends_on may cross intents (S8).
	"gabo-dev-lead/templates/plan.md": {
		columns: []string{
			"| field | value |",
			"| id | title | spec refs | depends_on | unblocks | acceptance | size |",
			"| risk | impact | mitigation |",
		},
		headings: []string{
			"# Plan <NNNN> <Title>",
			"## Steps",
			"## Tasks",
			"## Risks to the plan",
			"## Open questions",
		},
	},
	// No upfront schema (S4): one ordered row per migration or script.
	"gabo-db-master/templates/MIGRATIONS.md": {
		columns: []string{
			"| field | value |",
			"| order | intent | file | purpose | rollback |",
		},
		headings: []string{
			"# Migrations",
			"## Conventions",
			"## Log",
			"## Open questions",
		},
	},
	// Global styles only (S5); UX and app flow live in each spec.
	"gabo-frontend/templates/DESIGN.md": {
		columns: []string{
			"| field | value |",
			"| token | custom property | value | use |",
			"| component | level | states | used by |",
			"| case | what happened | why | what to do next |",
		},
		headings: []string{
			"# Design",
			"## 1. Principles",
			"## 2. Design tokens",
			"## 3. Component inventory",
			"## 4. Accessibility",
			"## 5. Messages and errors",
			"## 6. Open questions",
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

// The global planning templates are gone (S4, S5, S9) and the plan carries
// no milestones or global waves (S9, S10).
func TestGlobalPlanningTemplatesAreRemoved(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for _, p := range []string{
		"gabo-pm/templates/STORY.md",
		"gabo-dev-lead/templates/IMPLEMENTATION_PLAN.md",
		"gabo-db-master/templates/DB_SCHEMA.md",
		"gabo-frontend/templates/APP_FLOW.md",
		"gabo-frontend/templates/UX_UI.md",
	} {
		if _, ok := byPath[p]; ok {
			t.Errorf("%s: removed template still shipped", p)
		}
	}
	plan := strings.ToLower(byPath["gabo-dev-lead/templates/plan.md"])
	for _, s := range []string{"milestone", "| wave"} {
		if strings.Contains(plan, s) {
			t.Errorf("gabo-dev-lead/templates/plan.md: has %q", s)
		}
	}
}

func TestPMSkillPointsToTemplates(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)["gabo-pm/SKILL.md"]
	for _, s := range []string{
		"templates/PRD.md", "templates/intent.md", "templates/spec.md", "docs/product/intents/<NNNN-slug>/",
		"| id | type | requirement | acceptance criteria | priority | stories | verification |",
	} {
		if !strings.Contains(skill, s) {
			t.Errorf("gabo-pm/SKILL.md: missing %q", s)
		}
	}
}

// Every directiva skill names its templates, the fixed-questions rule and the
// stable columns it scaffolds from (T028, D19).
func TestDirectivaSkillsPointToTheirTemplates(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for skill, want := range map[string][]string{
		"gabo-dev-lead/SKILL.md": {
			"templates/TRD.md", "templates/plan.md", "## Fixed questions", "arc42", "MADR",
			"| id | decision | options considered | choice | consequences | status | requirement ids |",
			"| req | decisions | sections | stories | verification → test layer |",
			"| id | title | spec refs | depends_on | unblocks | acceptance | size |",
		},
		"gabo-db-master/SKILL.md": {
			"templates/MIGRATIONS.md", "## Fixed questions", "config-only", "Persistence inventory",
		},
		"gabo-frontend/SKILL.md": {
			"templates/DESIGN.md", "## 3. UX and app flow (optional)", "## Fixed questions",
		},
	} {
		doc, ok := byPath[skill]
		if !ok {
			t.Errorf("%s: missing", skill)
			continue
		}
		for _, s := range want {
			if !strings.Contains(doc, s) {
				t.Errorf("%s: missing %q", skill, s)
			}
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

// traceabilityGaps is the gap list of the traceability check (D19, T031):
// every directiva skill carries it word for word, so the four skills block a
// planning step on the same gaps.
var traceabilityGaps = []string{
	"story without requirement",
	"requirement without decision, task or verification",
	"task without story",
	"dependency cycle",
	"dangling id",
	"`depends_on` / `unblocks` mismatch",
	"Rows marked removed",
	"blocks the next planning step",
}

// T031 (J2, J6): the directiva skills ask their template's fixed questions one
// at a time, mark auto-mode answers as proposals and run the traceability check
// before each next planning step.
func TestDirectivaSkillsAskFixedQuestionsAndRunTheTraceabilityCheck(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for _, skill := range []string{
		"gabo-pm/SKILL.md", "gabo-dev-lead/SKILL.md", "gabo-db-master/SKILL.md", "gabo-frontend/SKILL.md",
	} {
		doc := byPath[skill]
		for _, s := range slices.Concat([]string{
			"one at a time", "**Proposed (auto)**", "## Traceability check", "before each next planning step",
		}, traceabilityGaps) {
			if !strings.Contains(doc, s) {
				t.Errorf("%s: missing %q", skill, s)
			}
		}
	}
}

// T031 (J2, J6): dev-lead creates tasks through the `tasks` contract with
// blocked-by / unblocks verified by re-read, adjusts tasks keeping their
// identity and answers research requests with sources.
func TestDevLeadCreatesAndAdjustsTasksAndResearches(t *testing.T) {
	t.Parallel()
	doc := embeddedByPath(t)["gabo-dev-lead/SKILL.md"]
	for _, s := range []string{
		"`tasks` contract", "blocked-by", "unblocks", "re-read", "unverified",
		"keep their identity", "never renumber", "(removed <date>)", "never recreate the plan",
		"researched answer", "sources",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-dev-lead/SKILL.md: missing %q", s)
		}
	}
}
