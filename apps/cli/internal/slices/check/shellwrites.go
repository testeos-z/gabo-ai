package check

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// humanHit is one path a shell command writes that is human-only (intent 0028
// S7, S13). raw is the path as the command spells it; unresolved marks a write
// after a cd the guard cannot read, which is blocked whatever the path is.
type humanHit struct {
	raw        string
	unresolved bool
}

// writeWord is one path a command writes: raw as spelled, path (relative to the
// working directory, or absolute) after the cd commands before it on the line,
// ancestor when the command takes a whole folder (cp, mv, rm), and unresolved
// after a cd the guard cannot read.
type writeWord struct {
	raw        string
	path       string
	ancestor   bool
	unresolved bool
}

// cdState is the folder the later commands of a line run in: dir relative to
// the working directory ("" is the working directory), or unresolved.
type cdState struct {
	dir        string
	unresolved bool
}

// word returns the write of raw made in this folder.
func (st cdState) word(raw string, ancestor bool) writeWord {
	w := writeWord{raw: raw, path: raw, ancestor: ancestor, unresolved: st.unresolved}
	switch {
	case st.unresolved:
		w.path = ""
	case st.dir != "" && !filepath.IsAbs(raw):
		w.path = path.Join(st.dir, raw)
	}
	return w
}

// enter returns the state after a cd or pushd with args. An operand the guard
// cannot read (none, -, a variable, a glob, a brace or a tilde) makes the folder
// unresolved for the rest of the line.
func (st cdState) enter(args []string) cdState {
	if st.unresolved {
		return st
	}
	operand, found := "", false
	for _, a := range args {
		if a != "-" && strings.HasPrefix(a, "-") {
			continue
		}
		operand, found = a, true
		break
	}
	if !found || operand == "-" || strings.ContainsAny(operand, "$`~*?[{") {
		return cdState{unresolved: true}
	}
	if filepath.IsAbs(operand) {
		return cdState{dir: path.Clean(operand)}
	}
	return cdState{dir: path.Join(st.dir, operand)}
}

// humanOnlyShellWrites returns the paths one shell command line writes that are
// human-only (intent 0028 S7, S13, PD6), each once, in command order. Only a
// coder session is judged. The paths are the targets of the command's
// redirections and the operands of tee, rm, cp, mv and sed -i, read as bash
// reads the line (shellCommands), with command wrappers taken off, cd
// commands followed, and brace expansion applied. A cp, mv or rm also writes a
// folder that contains an entry, so the root and such a folder are human-only
// as its target.
func (u *UseCase) humanOnlyShellWrites(command string, coder bool, list []string) []humanHit {
	if !coder {
		return nil
	}
	ws, _ := writeWords(shellCommands(command, true), true, cdState{})
	hs := humanEntries(list, u.s.FoldCase)
	var out []humanHit
	seen := map[humanHit]bool{}
	add := func(h humanHit) {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	for _, w := range ws {
		if w.unresolved {
			add(humanHit{raw: w.raw, unresolved: true})
			continue
		}
		for _, alt := range expandBraces(w.path) {
			if u.humanOnlyTarget(alt, hs) || (w.ancestor && u.humanAncestorTarget(alt, hs)) {
				add(humanHit{raw: w.raw})
				break
			}
		}
	}
	return out
}

// writeWords returns the paths the simple commands write, in order, and the
// folder the line ends in. Redirection targets and the operands of the writing
// commands are made in the folder the line has entered by then. One level of
// sh -c or bash -c is unwrapped (S6); its command line is read the same way,
// from the folder the line is in, and a cd in it does not leave it.
func writeWords(cmds []simpleCommand, unwrap bool, st cdState) ([]writeWord, cdState) {
	var out []writeWord
	for _, c := range cmds {
		for _, t := range c.targets {
			out = append(out, st.word(t, false))
		}
		argv := stripWrappers(skipPrefix(c.words))
		if len(argv) == 0 {
			continue
		}
		name, args := path.Base(argv[0]), argv[1:]
		switch {
		case unwrap && shellNames[name]:
			if inner, ok := shellCommandString(args); ok {
				sub, _ := writeWords(shellCommands(inner, true), false, st)
				out = append(out, sub...)
			}
		case name == "cd" || name == "pushd":
			st = st.enter(args)
		case name == "popd":
			st = cdState{unresolved: true}
		case name == "tee":
			for _, o := range operands(args, false) {
				out = append(out, st.word(o, false))
			}
		case name == "rm":
			for _, o := range operands(args, false) {
				out = append(out, st.word(o, true))
			}
		case name == "cp" || name == "mv":
			for _, o := range operands(args, true) {
				out = append(out, st.word(o, true))
			}
		case name == "sed" && inPlace(args):
			for _, o := range sedOperands(args) {
				out = append(out, st.word(o, false))
			}
		}
	}
	return out, st
}

// commandWrappers run the command that follows them, so a write by that
// command is the write (env tee, sudo cp, xargs rm, ...).
var commandWrappers = map[string]bool{
	"env": true, "command": true, "exec": true, "builtin": true, "nohup": true,
	"nice": true, "sudo": true, "xargs": true, "timeout": true, "stdbuf": true,
	"doas": true, "ionice": true, "chrt": true,
}

// wrapperValueOpts are the options of a wrapper that take the next word as their value.
var wrapperValueOpts = map[string]bool{"-u": true, "-g": true, "-n": true, "-C": true, "-I": true, "-c": true}

// wrapperNumeric are the wrappers whose first operand is a number: a duration
// (timeout) or a priority (chrt).
var wrapperNumeric = map[string]bool{"timeout": true, "chrt": true}

// number matches a duration or a priority: digits with an optional unit.
var number = regexp.MustCompile(`^[0-9.]+[smhd]?$`)

// stripWrappers drops the command wrappers in front of a command, with their
// options, the VAR=value assignments of env and the number of timeout or chrt,
// and returns the command.
func stripWrappers(argv []string) []string {
	for len(argv) > 0 && commandWrappers[path.Base(argv[0])] {
		name := path.Base(argv[0])
		argv = argv[1:]
		for len(argv) > 0 && (strings.HasPrefix(argv[0], "-") || isAssignment(argv[0])) {
			opt := argv[0]
			argv = argv[1:]
			if wrapperValueOpts[opt] && len(argv) > 0 {
				argv = argv[1:]
			}
		}
		if wrapperNumeric[name] && len(argv) > 0 && number.MatchString(argv[0]) {
			argv = argv[1:]
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

// sedOperands returns the files sed edits: the operands that are not options,
// without the values of -e and -f (a script and a read file) and, when neither
// -e nor -f is given, without the first operand, which is the script.
func sedOperands(args []string) []string {
	var files []string
	script, endOpts := false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case !endOpts && a == "--":
			endOpts = true
		case endOpts || a == "-" || !strings.HasPrefix(a, "-"):
			files = append(files, a)
		case a == "--expression" || a == "--file":
			script = true
			i++
		case strings.HasPrefix(a, "--expression=") || strings.HasPrefix(a, "--file="):
			script = true
		case strings.HasPrefix(a, "--"):
		default:
			// A short cluster: -i starts a suffix, so the rest is no option; -e and
			// -f take the rest of the word, or the next word when they end it.
			for k, r := range a[1:] {
				if r == 'i' {
					break
				}
				if r == 'e' || r == 'f' {
					script = true
					if k == len(a)-2 {
						i++
					}
					break
				}
			}
		}
	}
	if !script && len(files) > 0 {
		files = files[1:]
	}
	return files
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
