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
// engine files by project-relative path (a missing file has no entry), the
// binary version and the assistant engine (runtimes.assistant), which picks
// the engine whose main-session key gabo writes (intent 0010 S1, D1). An
// empty Assistant leaves every main-session key alone.
type Input struct {
	Files     map[string][]byte
	Version   string
	Assistant string
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
	Claude:   claudeRule,
	Codex:    codexRule,
	Opencode: opencodeRule,
	Pi:       piRule,
}

// Integrate runs the rule of each engine in set, in set order, and returns
// the report and the new bytes of every changed file. An engine without a
// rule is in the flexible tier: it writes nothing and gets the flexible-tier
// line (S8), with no step and no failure.
func Integrate(set []string, in Input) (Report, map[string][]byte) {
	var report Report
	writes := map[string][]byte{}
	for _, engine := range set {
		rule, ok := rules[engine]
		if !ok {
			report.Sections = append(report.Sections, Section{Engine: engine, Lines: []string{engine + ": flexible tier: skills and configuration only; nothing written"}})
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

// removals hold, per engine, the removal of the gabo main-session value
// from that engine's file when the engine is not in the set (intent 0010
// S4, D2). The engine's own rule removes it when the engine is in the set.
var removals = map[string]Rule{
	Claude:   claudeRemoval,
	Opencode: opencodeRemoval,
}

// removalFiles are, per engine with a removal, the files that may hold its
// main-session key.
var removalFiles = map[string][]string{
	Claude:   {ClaudeSettings},
	Opencode: OpencodeConfigs,
}

// Removable reports whether engine has a main-session removal, so the
// caller knows whose file to read even when the engine is not in the set.
func Removable(engine string) bool {
	_, ok := removals[engine]
	return ok
}

// RemovalFiles returns the project files the main-session removal of engine
// reads and may change, in lookup order; nil for an engine without one.
func RemovalFiles(engine string) []string {
	return slices.Clone(removalFiles[engine])
}

// Remove runs the main-session removal of each engine that is not in set,
// in the fixed order, and returns a section only for an engine whose files
// changed. A missing or unreadable file changes nothing and reports
// nothing: gabo init does not own the files of an engine it no longer
// integrates, it only clears its own value from them.
func Remove(set []string, in Input) (Report, map[string][]byte) {
	var report Report
	writes := map[string][]byte{}
	for _, engine := range order {
		remove, ok := removals[engine]
		if !ok || slices.Contains(set, engine) {
			continue
		}
		res := remove(in)
		if len(res.Writes) == 0 {
			continue
		}
		res.Section.Engine = engine
		report.Sections = append(report.Sections, res.Section)
		for path, b := range res.Writes {
			writes[path] = b
		}
	}
	return report, writes
}
