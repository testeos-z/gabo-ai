package engineconfig

import "strings"

// FailurePrefix starts every report line that is a failure (S5).
const FailurePrefix = "✗"

// Section is one engine's part of the report: its lines (files written,
// unchanged, kept keys, failures) and its steps outside the project.
type Section struct {
	Engine string
	Lines  []string
	Steps  []string
}

// Report is the `engine integration:` block, one section per engine in set
// order (S9).
type Report struct {
	Sections []Section
}

// Failed reports whether any line is a failure; the run then exits 1 after
// the report (D10).
func (r Report) Failed() bool {
	for _, s := range r.Sections {
		for _, l := range s.Lines {
			if strings.HasPrefix(l, FailurePrefix) {
				return true
			}
		}
	}
	return false
}

// Render returns the block: `engine integration:`, each section with its
// lines, then `you still need to:` with every step, each named by its
// engine. The steps list is omitted when no engine has a step.
func (r Report) Render() string {
	var b strings.Builder
	b.WriteString("engine integration:\n")
	var steps []string
	for _, s := range r.Sections {
		b.WriteString("  " + s.Engine + ":\n")
		for _, l := range s.Lines {
			b.WriteString("    " + l + "\n")
		}
		for _, st := range s.Steps {
			steps = append(steps, s.Engine+": "+st)
		}
	}
	if len(steps) > 0 {
		b.WriteString("you still need to:\n")
		for _, st := range steps {
			b.WriteString("  - " + st + "\n")
		}
	}
	return b.String()
}

// removedMainSessionLine is the report line of a removed gabo main-session
// value: it names runtimes.assistant as listed, one engine as in intent 0010
// S4, several joined with ", " (intent 0013 S3).
func removedMainSessionLine(file, key, value string, in Input) string {
	return "removed " + file + " " + key + ": " + quoteJSON(value) + " (runtimes.assistant is " + strings.Join(in.assistants(), ", ") + ")"
}
