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
	// Conceptual PRD (S2): what the product offers and what the user gets; no
	// stories and no acceptance criteria, which live in each intent's spec.
	"gabo-pm/templates/PRD.md": {
		columns: []string{
			"| field | value |",
			"| id | type | what the product offers | what the user gets | priority |",
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
			"## 7. What the product offers",
			"## 8. Quality requirements",
			"## 9. Constraints",
			"## 13. Glossary",
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
			"| component | level | states | stories | used by |",
			"| case | what happened | why | what to do next |",
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
	// Conventions-only TRD (S3): the stack the user chose (S6), architecture,
	// code and test conventions and the MADR decision table; per-change design
	// and requirement coverage live in each intent's spec and plan.
	"gabo-dev-lead/templates/TRD.md": {
		columns: []string{
			"| field | value |",
			"| area | choice | version | chosen by | notes |",
			"| constraint | kind | source |",
			"| id | decision | options considered | choice | consequences | status | requirement ids |",
			"| id | risk or debt | decision ids | mitigation |",
			"| term | meaning |",
		},
		headings: []string{
			"# Technical Requirements (TRD)",
			"## 1. Stack and tooling",
			"## 2. Constraints",
			"## 3. Architecture conventions",
			"## 4. Code conventions",
			"## 5. Test conventions",
			"## 6. Deployment and environments",
			"## 7. Architecture decisions",
			"## 8. Risks and technical debt",
			"## 9. Glossary",
			"## 10. Open questions",
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
	// Design norms (intent 0012 S1, S3): one rules table per area, exceptions
	// and a changelog; UX and app flow and the component inventory live in each
	// spec section 3 (D2).
	"gabo-frontend/templates/DESIGN.md": {
		columns: []string{
			"| field | value |",
			"| id | rule | level | rationale | verification | examples | source |",
			"| rule id | exception | reason | owner | date |",
			"| date | rule ids | change | decision |",
		},
		headings: []string{
			"# Design norms",
			"## 1. Principles and sources",
			"## 2. Tokens",
			"## 3. Color and themes",
			"## 4. Typography",
			"## 5. Layout and responsive behavior",
			"## 6. Shape and density",
			"## 7. Motion and state",
			"## 8. Icons",
			"## 9. Accessibility",
			"## 10. Content and writing",
			"## 11. Forms",
			"## 12. Components",
			"## 13. Exceptions",
			"## 14. Changelog",
			"## 15. Open questions",
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

// S10 (intent 0008, TRD 8.5): every plan asks whether the change needs a
// `gabo check` update, answered in Steps, and the table keeps its columns.
func TestPlanTemplateAsksForGaboCheckImpact(t *testing.T) {
	t.Parallel()
	doc := embeddedByPath(t)["gabo-dev-lead/templates/plan.md"]
	qs, err := fixedQuestions(doc)
	if err != nil {
		t.Fatalf("gabo-dev-lead/templates/plan.md: %v", err)
	}
	want := [3]string{"Q9", "Does the change need a `gabo check` update, and which task covers it?", "Steps"}
	if !slices.Contains(qs, want) {
		t.Errorf("gabo-dev-lead/templates/plan.md: fixed questions %v, want row %v", qs, want)
	}
}

// Intent 0012 S1, S2, S3: the design norms template states how rules are
// written and governed, fits the per-change model and keeps no concrete
// messages table (those live in each spec section 3).
func TestDesignNormsTemplateRulesAndPerChangeFit(t *testing.T) {
	t.Parallel()
	const p = "gabo-frontend/templates/DESIGN.md"
	doc := embeddedByPath(t)[p]
	qs, err := fixedQuestions(doc)
	if err != nil {
		t.Fatalf("%s: %v", p, err)
	}
	if len(qs) != 14 {
		t.Errorf("%s: %d fixed questions, want 14", p, len(qs))
	}
	for _, s := range []string{
		"A rule that cannot state its verification is a SHOULD at most.",
		"stable and never reused",
		"An exception is a row in section 13",
		"every rule change is a row in section 14",
		"UX and app flow of each change live in section 3 of that intent's `spec.md`, which cites rule ids instead of repeating rules.",
		"A product without a GUI marks the visual sections",
		`"not applicable" with the reason`,
		"conversation or CLI conventions under sections 1 and 10",
		"(docs/product/PRD.md)", "(docs/product/TRD.md)", "(docs/product/intents/)",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", p, s)
		}
	}
	for _, s := range []string{"UX_UI", "APP_FLOW", "| case | what happened |", "| component | level |", "shared component inventory"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s: has %q", p, s)
		}
	}
	// Intent 0012 S3, D2: section 12 keeps only the K component rules, and
	// Q12 asks for those rules instead of the list of components.
	const k12, k13 = "## 12. Components", "## 13. Exceptions"
	i, j := strings.Index(doc, k12), strings.Index(doc, k13)
	if i < 0 || j < i {
		t.Fatalf("%s: section 12 not found between %q and %q", p, k12, k13)
	}
	if !strings.Contains(doc[i:j], "| K1 |") {
		t.Errorf("%s: section 12 missing the K1 rules row", p)
	}
	want := [3]string{"Q12", "Which component rules apply: atomic levels, required states, and how stories and examples are kept?", "12. Components"}
	if !slices.Contains(qs, want) {
		t.Errorf("%s: fixed questions %v, want row %v", p, qs, want)
	}
}

// Intent 0012 S3: spec section 3 holds the change's concrete messages and
// links the design norms instead of repeating tokens, accessibility or
// wording rules; product-specific keyboard paths stay in the journey.
func TestSpecTemplateSection3HoldsMessagesAndLinksNorms(t *testing.T) {
	t.Parallel()
	const p = "gabo-pm/templates/spec.md"
	doc := embeddedByPath(t)[p]
	const start, end = "## 3. UX and app flow (optional)", "## 4. DB migration steps (optional)"
	i, j := strings.Index(doc, start), strings.Index(doc, end)
	if i < 0 || j < i {
		t.Fatalf("%s: section 3 not found between %q and %q", p, start, end)
	}
	section := doc[i:j]
	for _, s := range []string{
		"| case | what happened | why | what to do next |",
		"[`DESIGN.md`](../../DESIGN.md) sections 2-3 (tokens, color)",
		"section 9 (accessibility)",
		"section 10 (wording)",
		"Product-specific keyboard paths stay in the journey.",
		"| component | level | states | stories | used by |",
		"follow the `K` rules of `DESIGN.md` section 12",
		"\"used by\" names the change's surfaces",
	} {
		if !strings.Contains(section, s) {
			t.Errorf("%s: section 3 missing %q", p, s)
		}
	}
	if !strings.Contains(doc, "Design norms stay in the repository `DESIGN.md`") {
		t.Errorf("%s: missing the design norms guidance line", p)
	}
	for _, s := range []string{"| token |", "Global styles"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s: has %q", p, s)
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

// Intent 0012 S4, D2: in gabo-frontend's steps the component inventory is
// written in the spec section 3 step (3), never in the DESIGN.md step (2).
func TestFrontendSkillPutsTheInventoryInTheSpecStep(t *testing.T) {
	t.Parallel()
	const p = "gabo-frontend/SKILL.md"
	steps := map[string]string{}
	for _, l := range strings.Split(embeddedByPath(t)[p], "\n") {
		for _, n := range []string{"2. ", "3. "} {
			if strings.HasPrefix(l, n) {
				steps[n] = l
			}
		}
	}
	if !strings.Contains(steps["3. "], "| component | level | states | stories | used by |") {
		t.Errorf("%s: step 3 does not name the component inventory table: %q", p, steps["3. "])
	}
	if steps["2. "] == "" || strings.Contains(strings.ToLower(steps["2. "]), "inventory") {
		t.Errorf("%s: step 2 missing or names an inventory: %q", p, steps["2. "])
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
	for _, p := range []string{"gabo-pm/templates/PRD.md", "gabo-dev-lead/templates/TRD.md"} {
		doc := strings.ToLower(byPath[p])
		for _, s := range []string{"stories", "story", "us-0", "requirement coverage"} {
			if strings.Contains(doc, s) {
				t.Errorf("%s: has %q", p, s)
			}
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
		"| id | type | what the product offers | what the user gets | priority |",
		"highest `NNNN` plus one", "never an agent",
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
			"templates/TRD.md", "templates/plan.md", "## Fixed questions", "MADR",
			"| id | decision | options considered | choice | consequences | status | requirement ids |",
			"| area | choice | version | chosen by | notes |",
			"| id | title | spec refs | depends_on | unblocks | acceptance | size |",
			"only after a human accepted the spec", "`parent`",
		},
		"gabo-db-master/SKILL.md": {
			"templates/MIGRATIONS.md", "## Fixed questions", "config-only", "Persistence inventory",
			"`## Conventions`",
		},
		// Intent 0012 S4, S6: the design norms, their rule shape, the spec
		// section 3 messages and links, and the conversion rule.
		"gabo-frontend/SKILL.md": {
			"templates/DESIGN.md", "## 3. UX and app flow (optional)", "## Fixed questions",
			"the design norms (from `templates/DESIGN.md`",
			"one rules table per area", "an RFC 2119 level (MUST, SHOULD or MAY), a rationale, a verification, examples and a source",
			"exceptions (section 13)", "a changelog (section 14)",
			"messages table `| case | what happened | why | what to do next |`",
			"links `DESIGN.md` sections 2-3 (tokens, color), 9 (accessibility) and 10 (wording)",
			"component inventory table `| component | level | states | stories | used by |`",
			"never rewrite an existing `DESIGN.md` in another shape silently",
			"the next `DESIGN.md` update proposes the conversion", "to the owner in manual mode",
			"as **Proposed (auto)** in auto mode", "keeps every existing rule with a new id",
			"records the conversion as one changelog row",
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

// traceabilityGaps is the gap list of the traceability check (D19, per-change
// model S7-S9): every directiva skill carries it word for word, so the four
// skills block a planning step on the same gaps.
var traceabilityGaps = []string{
	"an intent without spec;",
	"an accepted spec without plan;",
	"a plan task without a tracker task, or the reverse;",
	"a task whose parent is not exactly one intent;",
	"a requirement in the spec not covered by any plan task;",
	"a dependency cycle;",
	"a dangling id, including a cross-intent `<NNNN>/T<n>` reference;",
	"a `depends_on`/`unblocks` mismatch.",
	"Rows marked removed",
	"blocks the next planning step",
}

// The per-change model retires stories, the global plan, milestones, global
// waves and the global DB, UX and app-flow docs from every planning skill (S7, S9).
var retiredPlanningTerms = []string{
	"user stor", "docs/product/stories", "US-0", "story without", "without story",
	"IMPLEMENTATION_PLAN", "milestone", "DB_SCHEMA", "UX_UI", "APP_FLOW",
}

func TestPlanningSkillsDropRetiredGlobalPlanning(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for _, skill := range []string{
		"gabo-pm/SKILL.md", "gabo-dev-lead/SKILL.md", "gabo-db-master/SKILL.md", "gabo-frontend/SKILL.md",
		"gabo-doc-guide/SKILL.md",
	} {
		body := byPath[skill]
		if i := strings.Index(body, "\n---\n"); i >= 0 {
			body = body[i:]
		}
		for _, s := range retiredPlanningTerms {
			if strings.Contains(body, s) {
				t.Errorf("%s: still has %q", skill, s)
			}
		}
	}
	if strings.Contains(strings.ToLower(byPath["gabo-dev-lead/SKILL.md"]), "wave") {
		t.Errorf("gabo-dev-lead/SKILL.md: still plans waves")
	}
}

// Ownership in the per-change model (S7, S8, S14).
func TestPlanningSkillsOwnTheirIntentArtifacts(t *testing.T) {
	t.Parallel()
	byPath := embeddedByPath(t)
	for skill, want := range map[string][]string{
		"gabo-pm/SKILL.md":        {"intent.md", "spec.md", "Acceptance"},
		"gabo-dev-lead/SKILL.md":  {"plan.md", "`parent`", "product card"},
		"gabo-db-master/SKILL.md": {"spec.md", "MIGRATIONS.md"},
		"gabo-frontend/SKILL.md":  {"spec.md", "DESIGN.md"},
		"gabo-doc-guide/SKILL.md": {
			"docs/product/PRD.md", "docs/product/TRD.md", "docs/product/DESIGN.md", "docs/product/MIGRATIONS.md",
			"docs/product/intents/<NNNN-slug>/", "intent.md", "spec.md", "plan.md", "review findings",
			"Which design norms apply", "How are the design norms",
		},
	} {
		for _, s := range want {
			if !strings.Contains(byPath[skill], s) {
				t.Errorf("%s: missing %q", skill, s)
			}
		}
	}
	// Intent 0012 S4: `DESIGN.md` is the design norms, no longer global styles.
	for _, skill := range []string{"gabo-frontend/SKILL.md", "gabo-doc-guide/SKILL.md"} {
		if strings.Contains(strings.ToLower(byPath[skill]), "global styles") {
			t.Errorf("%s: still describes DESIGN.md as global styles", skill)
		}
	}
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
