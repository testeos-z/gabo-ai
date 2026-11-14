package skills

import "strings"

// instructionTemplate is the catalog template of the three instruction-file
// blocks (spec 0030 S19): the commands block, the repeated-mistakes section
// and the review-findings rule.
const instructionTemplate = "gabo-setup/instruction-blocks.md"

// commandsPlaceholder marks where setup puts the project's review.checks.
const commandsPlaceholder = "{{commands}}"

// InstructionTemplate returns the catalog template as it is embedded.
func InstructionTemplate() (string, error) {
	b, err := content.ReadFile(instructionTemplate)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// InstructionOffer returns the text setup offers for a project's instruction
// file. exists reports whether the file is present; existing is its text. When
// the file is absent, the offer is the whole template with the project's check
// commands in the commands block. When the file is present, the offer holds
// only the sections the file does not have yet. An empty offer means the file
// already has every section and stays unchanged. The offer is only ever
// appended by setup; it never replaces text the project has.
func InstructionOffer(existing string, exists bool, checks []string) (string, error) {
	tpl, err := InstructionTemplate()
	if err != nil {
		return "", err
	}
	have := map[string]bool{}
	if exists {
		for _, l := range strings.Split(existing, "\n") {
			have[strings.TrimRight(l, " \t")] = true
		}
	}
	var parts []string
	for _, section := range splitSections(tpl) {
		if have[section.heading] {
			continue
		}
		parts = append(parts, strings.ReplaceAll(section.text, commandsPlaceholder, commandLines(checks)))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, "\n\n") + "\n", nil
}

// instructionSection is one level-2 section of the template: its heading line
// and its text up to the next heading, without trailing blank lines.
type instructionSection struct {
	heading string
	text    string
}

func splitSections(doc string) []instructionSection {
	var out []instructionSection
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		text := strings.TrimRight(strings.Join(cur, "\n"), "\n")
		out = append(out, instructionSection{heading: cur[0], text: text})
	}
	for _, l := range strings.Split(doc, "\n") {
		if strings.HasPrefix(l, "## ") {
			flush()
			cur = nil
		}
		if strings.HasPrefix(l, "## ") || len(cur) > 0 {
			cur = append(cur, l)
		}
	}
	flush()
	return out
}

// commandLines lists each check command with the marker of its healthy output
// that setup replaces with the real output of the command.
func commandLines(checks []string) string {
	if len(checks) == 0 {
		return "<no review.checks in gabo.yaml: ask the owner for the commands>"
	}
	var lines []string
	for _, c := range checks {
		lines = append(lines, "$ "+c, "<healthy output: run the command and paste what it prints when it passes>", "exit: 0")
	}
	return strings.Join(lines, "\n")
}
