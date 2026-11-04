// Package engineconfig is the pure engine integration domain shared by the
// initcmd and check slices: the engine set, the per-engine rules and the
// report. It has no ports and no I/O: bytes in, bytes and report lines out.
// It never looks for engines on the machine (intent 0006 S1, D1).
package engineconfig

import (
	"slices"
	"strings"
)

// Engines with a gabo integration, in the fixed handling order (S1).
const (
	Claude   = "claude"
	Codex    = "codex"
	Opencode = "opencode"
	Pi       = "pi"
)

var order = []string{Claude, Codex, Opencode, Pi}

// Set returns the engine set named by the runtime roles (coder, reviewer,
// assistant), each engine once: the known engines in the fixed order, then
// any other name in the order it appears. Empty names are skipped.
func Set(runtimes ...string) []string {
	var known, other []string
	for _, name := range runtimes {
		name = strings.TrimSpace(name)
		if name == "" || slices.Contains(known, name) || slices.Contains(other, name) {
			continue
		}
		if slices.Contains(order, name) {
			known = append(known, name)
		} else {
			other = append(other, name)
		}
	}
	slices.SortFunc(known, func(a, b string) int { return slices.Index(order, a) - slices.Index(order, b) })
	return append(known, other...)
}

// Input is what an engine rule reads: the current bytes of the project's
// engine files by project-relative path (a missing file has no entry) and
// the binary version.
type Input struct {
	Files   map[string][]byte
	Version string
}

// Result is what an engine rule returns: its report section and the new
// bytes of each file it changes (no entry means no change).
type Result struct {
	Section Section
	Writes  map[string][]byte
}

// Rule integrates one engine. It is pure: it reads only its Input.
type Rule func(Input) Result

// rules holds one entry per engine with a gabo integration.
var rules = map[string]Rule{
	Pi: piRule,
}

// Integrate runs the rule of each engine in set, in set order, and returns
// the report and the new bytes of every changed file. An engine without a
// rule writes nothing and gets the unknown-engine line (S8), which is not a
// failure.
func Integrate(set []string, in Input) (Report, map[string][]byte) {
	var report Report
	writes := map[string][]byte{}
	for _, engine := range set {
		rule, ok := rules[engine]
		if !ok {
			report.Sections = append(report.Sections, Section{Engine: engine, Lines: []string{engine + ": no gabo integration known; nothing written"}})
			continue
		}
		res := rule(in)
		res.Section.Engine = engine
		report.Sections = append(report.Sections, res.Section)
		for path, b := range res.Writes {
			writes[path] = b
		}
	}
	return report, writes
}
