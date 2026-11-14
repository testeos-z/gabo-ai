package check

import (
	"path"
	"strings"
)

// humanOnlyShellWrites returns the raw words of the paths one shell command
// line writes that are human-only (intent 0028 S7, S13, PD6), each once, in
// command order. Only a coder session is judged. The paths are the targets of
// the command's redirections and the operands of tee, rm, cp, mv and sed -i,
// read as bash reads the line (shellCommands), with brace expansion and the
// command wrappers taken off each command. Each path is judged by HumanOnlyPath.
func (u *UseCase) humanOnlyShellWrites(command string, coder bool, list []string) []string {
	if !coder {
		return nil
	}
	hs := humanEntries(list, u.s.FoldCase)
	var out []string
	seen := map[string]bool{}
	for _, w := range writeWords(shellCommands(command, true), true) {
		if seen[w] {
			continue
		}
		for _, alt := range expandBraces(w) {
			if u.humanOnlyTarget(alt, hs) {
				seen[w] = true
				out = append(out, w)
				break
			}
		}
	}
	return out
}

// writeWords returns the raw words of the paths the simple commands write: each
// command's redirection targets, then the operands of tee, rm, cp, mv and
// sed -i. One level of sh -c or bash -c is unwrapped (S6); its command line is
// read the same way, without a further unwrap.
func writeWords(cmds []simpleCommand, unwrap bool) []string {
	var out []string
	for _, c := range cmds {
		out = append(out, c.targets...)
		argv := stripWrappers(skipPrefix(c.words))
		if len(argv) == 0 {
			continue
		}
		name, args := path.Base(argv[0]), argv[1:]
		switch {
		case unwrap && shellNames[name]:
			if inner, ok := shellCommandString(args); ok {
				out = append(out, writeWords(shellCommands(inner, true), false)...)
			}
		case name == "tee" || name == "rm":
			out = append(out, operands(args, false)...)
		case name == "cp" || name == "mv":
			out = append(out, operands(args, true)...)
		case name == "sed" && inPlace(args):
			out = append(out, operands(args, false)...)
		}
	}
	return out
}

// commandWrappers run the command that follows them, so a write by that
// command is the write (env tee, sudo cp, xargs rm, ...).
var commandWrappers = map[string]bool{
	"env": true, "command": true, "exec": true, "builtin": true, "nohup": true,
	"nice": true, "sudo": true, "xargs": true,
}

// wrapperValueOpts are the options of a wrapper that take the next word as their value.
var wrapperValueOpts = map[string]bool{"-u": true, "-g": true, "-n": true, "-C": true, "-I": true}

// stripWrappers drops the command wrappers in front of a command, with their
// options and the VAR=value assignments of env, and returns the command.
func stripWrappers(argv []string) []string {
	for len(argv) > 0 && commandWrappers[path.Base(argv[0])] {
		argv = argv[1:]
		for len(argv) > 0 && (strings.HasPrefix(argv[0], "-") || isAssignment(argv[0])) {
			opt := argv[0]
			argv = argv[1:]
			if wrapperValueOpts[opt] && len(argv) > 0 {
				argv = argv[1:]
			}
		}
	}
	return argv
}

// operands returns the operands among a command's arguments: the words that
// are not options, and every word after --. With targetOpts (cp and mv) the
// directory of -t or --target-directory is an operand too.
func operands(args []string, targetOpts bool) []string {
	var out []string
	endOpts := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case !endOpts && a == "--":
			endOpts = true
		case endOpts || a == "-" || !strings.HasPrefix(a, "-"):
			out = append(out, a)
		case strings.HasPrefix(a, "--"):
			if v, ok := strings.CutPrefix(a, "--target-directory="); ok && targetOpts {
				out = append(out, v)
			} else if a == "--target-directory" && targetOpts && i+1 < len(args) {
				out = append(out, args[i+1])
				i++
			}
		case targetOpts && strings.ContainsRune(a, 't'):
			// -t in a cluster: the rest of the word, else the next word, is the directory.
			if rest := a[strings.IndexByte(a, 't')+1:]; rest != "" {
				out = append(out, rest)
			} else if i+1 < len(args) {
				out = append(out, args[i+1])
				i++
			}
		}
	}
	return out
}

// inPlace reports whether sed's arguments ask for in-place editing: an -i in a
// short option cluster (before an e or f, which take the rest of the word), or
// --in-place.
func inPlace(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == "--in-place" || strings.HasPrefix(a, "--in-place=") {
			return true
		}
		if strings.HasPrefix(a, "--") || !strings.HasPrefix(a, "-") || a == "-" {
			continue
		}
		for _, r := range a[1:] {
			if r == 'e' || r == 'f' {
				break
			}
			if r == 'i' {
				return true
			}
		}
	}
	return false
}

// expandBraces returns the words a brace expression stands for, as bash expands
// them: each {a,b} group with a comma gives one word per alternative, the
// leftmost group first. A brace with no comma is a literal (intent 0028 S7).
func expandBraces(w string) []string {
	start := strings.IndexByte(w, '{')
	if start < 0 {
		return []string{w}
	}
	end, parts := braceGroup(w, start)
	if end < 0 {
		var out []string
		for _, r := range expandBraces(w[start+1:]) {
			out = append(out, w[:start+1]+r)
		}
		return out
	}
	var out []string
	for _, p := range parts {
		out = append(out, expandBraces(w[:start]+p+w[end+1:])...)
	}
	return out
}

// braceGroup finds the group that opens at w[start] and closes at its matching
// }, and the comma-separated parts of its top level. end is -1 when the brace
// has no closing brace or no top-level comma.
func braceGroup(w string, start int) (int, []string) {
	depth, from := 0, start+1
	var parts []string
	for i := start; i < len(w); i++ {
		switch w[i] {
		case '{':
			depth++
		case ',':
			if depth == 1 {
				parts = append(parts, w[from:i])
				from = i + 1
			}
		case '}':
			depth--
			if depth == 0 {
				if len(parts) == 0 {
					return -1, nil
				}
				return i, append(parts, w[from:i])
			}
		}
	}
	return -1, nil
}
