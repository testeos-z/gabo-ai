package check

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
)

// Design norms (spec 0012 S7, TRD 9.4 Design norms): docs/product/DESIGN.md,
// when present, is checked offline against the shape of the
// gabo-frontend/templates/DESIGN.md norms template.
const (
	DesignFile     = "docs/product/DESIGN.md"
	NextFixDesign  = "next: fix " + DesignFile + " above (gabo-frontend keeps it), then rerun gabo check"
	designTitle    = "# Design norms"
	designLegacy   = DesignFile + `: not in the design norms shape (no "# Design norms" title); ask gabo-frontend to propose the conversion`
	designRuleCols = "| id | rule | level | rationale | verification | examples | source |"
	designExcCols  = "| rule id | exception | reason | owner | date |"
	designLogCols  = "| date | rule ids | change | decision |"
	// designRuleSections are the sections whose tables are rules tables.
	designRuleSections = 12
)

// designSections are the 15 section headings of the norms template.
var designSections = []string{
	"Principles and sources",
	"Tokens",
	"Color and themes",
	"Typography",
	"Layout and responsive behavior",
	"Shape and density",
	"Motion and state",
	"Icons",
	"Accessibility",
	"Content and writing",
	"Forms",
	"Components",
	"Exceptions",
	"Changelog",
	"Open questions",
}

var (
	designDateRE        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	designPlaceholderRE = regexp.MustCompile(`^<[^>]*>$`)
	designLevelRE       = map[string]*regexp.Regexp{
		"MUST":   regexp.MustCompile(`\bMUST\b`),
		"SHOULD": regexp.MustCompile(`\bSHOULD\b`),
		"MAY":    regexp.MustCompile(`\bMAY\b`),
	}
)

// designTable is one Markdown table: its header cells and data rows, the
// separator row dropped.
type designTable struct {
	header []string
	rows   [][]string
}

// designSection is the text and tables under one "## " heading, outside
// code fences.
type designSection struct {
	text   string
	tables []designTable
}

// checkDesign validates DesignFile. Without it nothing is checked and
// nothing is reported; a file whose title is not designTitle only warns.
func (u *UseCase) checkDesign(r *Report) error {
	b, err := fs.ReadFile(u.project, DesignFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", DesignFile, err)
	}
	title, sections := splitDesign(string(b))
	if title != designTitle {
		r.warn(designLegacy)
		return nil
	}
	var fails []string
	bad := func(what string) { fails = append(fails, "✗ "+DesignFile+": "+what) }
	for i, name := range designSections {
		if _, ok := sections[designHeading(i+1, name)]; !ok {
			bad(fmt.Sprintf("missing section %q", designHeading(i+1, name)))
		}
	}
	var rules [][]string
	for n := 1; n <= designRuleSections; n++ {
		s, ok := sections[designHeading(n, designSections[n-1])]
		if !ok {
			continue
		}
		found := false
		for _, t := range s.tables {
			if joinCells(t.header) != designRuleCols {
				bad(fmt.Sprintf("section %d: table columns must be %s", n, designRuleCols))
				found = true
				break
			}
		}
		for _, t := range s.tables {
			if joinCells(t.header) != designRuleCols {
				continue
			}
			found = true
			for _, row := range t.rows {
				rules = append(rules, append([]string{fmt.Sprint(n)}, padCells(row, 7)...))
			}
		}
		if !found && !strings.Contains(strings.ToLower(s.text), "not applicable") {
			bad(fmt.Sprintf("section %d has no rules table and does not say \"not applicable\"", n))
		}
	}
	known, twice := map[string]bool{}, map[string]bool{}
	rowInSection := map[string]int{}
	for _, row := range rules {
		section, id, rule, level, verification := row[0], row[1], row[2], row[3], row[5]
		rowInSection[section]++
		if id == "" {
			bad(fmt.Sprintf("section %s row %d: no rule id", section, rowInSection[section]))
			continue
		}
		if known[id] && !twice[id] {
			twice[id] = true
			bad("rule " + id + " used twice")
		}
		known[id] = true
		re, ok := designLevelRE[level]
		switch {
		case !ok:
			bad(fmt.Sprintf("rule %s: level %q is not MUST, SHOULD or MAY", id, level))
		case !re.MatchString(rule):
			bad(fmt.Sprintf("rule %s: level %s but the rule does not say %s", id, level, level))
		}
		if level == "MUST" && blankCell(verification) {
			bad("rule " + id + ": a MUST rule states no verification")
		}
	}
	if s, ok := sections[designHeading(13, designSections[12])]; ok {
		t, found := s.table(designExcCols)
		if !found {
			bad("section 13 needs the table " + designExcCols)
		}
		for _, row := range t.rows {
			c := padCells(row, 5)
			if !known[c[0]] {
				bad("exception for " + c[0] + ": no such rule")
			}
			if blankCell(c[2]) || blankCell(c[3]) || !designDateRE.MatchString(c[4]) {
				bad("exception for " + c[0] + ": reason, owner and date (YYYY-MM-DD) required")
			}
		}
	}
	if s, ok := sections[designHeading(14, designSections[13])]; ok {
		t, found := s.table(designLogCols)
		if !found {
			bad("section 14 needs the table " + designLogCols)
		}
		for k, row := range t.rows {
			c := padCells(row, 4)
			// The rule ids are not resolved: they may name retired rules.
			if !designDateRE.MatchString(c[0]) || blankCell(c[2]) || blankCell(c[3]) {
				bad(fmt.Sprintf("changelog row %d: date (YYYY-MM-DD), change and decision required", k+1))
			}
		}
	}
	if len(fails) > 0 {
		r.fail(append(fails, NextFixDesign)...)
	}
	return nil
}

func designHeading(n int, name string) string { return fmt.Sprintf("## %d. %s", n, name) }

// table returns the first table of s whose header is cols.
func (s designSection) table(cols string) (designTable, bool) {
	for _, t := range s.tables {
		if joinCells(t.header) == cols {
			return t, true
		}
	}
	return designTable{}, false
}

// splitDesign returns the first "# " heading and the sections by their
// "## " heading (the first of a repeated one wins). Lines inside code
// fences are skipped, so a quoted template is neither a heading nor a table.
func splitDesign(doc string) (string, map[string]designSection) {
	title := ""
	sections := map[string]designSection{}
	var current string
	var text strings.Builder
	var tables []designTable
	var table *designTable
	flush := func() {
		if current != "" {
			if _, seen := sections[current]; !seen {
				sections[current] = designSection{text: text.String(), tables: tables}
			}
		}
		text.Reset()
		tables, table = nil, nil
	}
	fenced := false
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
			table = nil
			continue
		}
		if fenced {
			continue
		}
		heading := strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(heading, "# ") {
			if title == "" {
				title = heading
			}
			flush()
			current = ""
			continue
		}
		if strings.HasPrefix(heading, "## ") {
			flush()
			current = heading
			continue
		}
		text.WriteString(line + "\n")
		cells := designCells(trimmed)
		if cells == nil {
			table = nil
			continue
		}
		if table == nil {
			tables = append(tables, designTable{header: cells})
			table = &tables[len(tables)-1]
			continue
		}
		if !isSeparator(cells) {
			table.rows = append(table.rows, cells)
		}
	}
	flush()
	return title, sections
}

// designCells splits a table line on unescaped "|"; an escaped "\|" stays
// in its cell as written. A line that does not start with "|" is no row.
func designCells(line string) []string {
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	var cells []string
	var cell strings.Builder
	escaped := false
	for _, c := range line[1:] {
		switch {
		case escaped:
			cell.WriteRune(c)
			escaped = false
		case c == '\\':
			cell.WriteRune(c)
			escaped = true
		case c == '|':
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteRune(c)
		}
	}
	if rest := strings.TrimSpace(cell.String()); rest != "" {
		cells = append(cells, rest)
	}
	return cells
}

func joinCells(cells []string) string { return "| " + strings.Join(cells, " | ") + " |" }

// padCells returns row with at least n cells.
func padCells(row []string, n int) []string {
	out := slices.Clone(row)
	for len(out) < n {
		out = append(out, "")
	}
	return out
}

// blankCell reports a cell that states nothing: empty, a dash or a
// <placeholder>.
func blankCell(c string) bool {
	return c == "" || c == "—" || c == "-" || designPlaceholderRE.MatchString(c)
}
