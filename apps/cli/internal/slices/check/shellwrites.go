package check

import (
	"path"
	"strings"
)

// HumanOnlyShellWrites returns the raw words of the paths one shell command
// line writes that are human-only (intent 0028 S7, S13, PD6), each once, in
// command order. Only a coder session is judged. The paths are the targets of
// the command's redirections and the operands of tee, rm, cp, mv and sed -i,
// read as bash reads the line (shellCommands); root and fold place relative
// paths as HumanOnlyPath does.
func HumanOnlyShellWrites(command string, coder bool, root string, fold bool, list []string) []string {
	if !coder {
		return nil
	}
	hs := humanEntries(list)
	var out []string
	seen := map[string]bool{}
	for _, w := range writeWords(shellCommands(command, true), true) {
		if !seen[w] && humanOnlyMatch(w, root, fold, hs) {
			seen[w] = true
			out = append(out, w)
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
		argv := skipPrefix(c.words)
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
