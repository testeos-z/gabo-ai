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

// session in runtimes.coder or runtimes.reviewer runs that role in the main
// session's engine, one of the listed assistants, so it adds no engine
// (intent 0013 S2).
const session = "session"

// Engines returns the engine set of a project (intent 0013 S2, D2): the
// coder and reviewer engines, then every listed assistant engine in its
// listed order, each once, in that order of first appearance. session and
// empty names add no engine. gabo check walks this role order; the rules
// that need the fixed order wrap it in Set. The reviewer scripts'
// declared_agent_dirs mirrors this rule.
func Engines(coder, reviewer string, assistants []string) []string {
	var out []string
	for _, name := range append([]string{coder, reviewer}, assistants...) {
		name = strings.TrimSpace(name)
		if name != "" && name != session && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

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
// binary version and the assistant engines (runtimes.assistant, one engine
// or a list), which pick the engines whose main-session key gabo writes
// (intent 0010 S1, D1; intent 0013 S3). No assistant engine leaves every
// main-session key alone.
type Input struct {
	Files      map[string][]byte
	Version    string
	Assistants []string
	// Mode is the gabo.yaml mode (intent 0023 S5, D4); only ModeAuto makes
	// the guardrails rule write.
	Mode string
	// PiPackage reports that the Pi subagent-package check found a package
	// (intent 0016 S1, D1); piRule then omits the refusal. Unset means no
	// package, which prints the refusal (S2).
	PiPackage bool
}

// assistants returns the listed assistant engines, trimmed, without empty
// names.
func (in Input) assistants() []string {
	var out []string
	for _, name := range in.Assistants {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// lists reports whether engine is a listed assistant engine, so gabo writes
// its main-session key (intent 0013 S3).
func (in Input) lists(engine string) bool { return slices.Contains(in.assistants(), engine) }

// unlists reports whether engine's main-session value is removed; see
// Unlisted.
func (in Input) unlists(engine string) bool { return Unlisted(in.Assistants, engine) }

// Unlisted reports whether runtimes.assistant lists some engine but not
// engine, so gabo removes engine's gabo main-session value (intent 0010 S4,
// D2; intent 0013 D7): a listed engine's value is never removed, and no
// assistant engine removes nothing.
func Unlisted(assistants []string, engine string) bool {
	in := Input{Assistants: assistants}
	return len(in.assistants()) > 0 && !in.lists(engine)
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
