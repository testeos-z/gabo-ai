package check

import (
	"path"
	"strings"
)

// ShellViolation is one shell-guard rule a shell command breaks (intent 0028
// S4): the rule in words and the route a person or the coder takes instead.
type ShellViolation struct {
	Rule  string
	Route string
}

// The shell-guard rules and their routes (intent 0028 S4, S5, S11).
const (
	ruleMainPush      = "git push to main is blocked"
	routeMainCoder    = "push your task branch and open a pull request into dev; the reviewer merges it"
	routeMainOwner    = "the owner pushes to main outside agent sessions"
	ruleDevPush       = "git push to master or dev is blocked in a coder session"
	routeDevPush      = "push your task branch instead; changes reach dev through a pull request"
	ruleForcePush     = "git push with --force, -f, --force-with-lease, --mirror or a + refspec is blocked in a coder session"
	routeForcePush    = "push without force; a rewritten shared branch is the owner's decision"
	ruleBranchForce   = "git branch -D (delete with force) is blocked in a coder session"
	routeBranchForce  = "use git branch -d, which refuses unmerged work; the owner deletes a branch that needs force"
	ruleWorktreeForce = "git worktree remove with --force or -f is blocked in a coder session"
	routeWorktree     = "remove the worktree without --force; the Task Close removes it"
	ruleRmGlob        = "rm with a recursive flag on a glob (*, ? or [) is blocked in a coder session"
	routeRmGlob       = "name each path to remove, without a glob"
	ruleRmWorktrees   = "rm with a recursive flag under a *-worktrees/ folder is blocked in a coder session"
	routeRmWorktrees  = "a worktree is removed by its Task Close, never by rm"
)

// ShellViolations returns the shell-guard violations of one command line, in
// command order, each rule once (intent 0028 S4, S6). coder selects the coder
// session rules; a main session gets the push to main rule alone. The line is
// read the way the shell reads it: each command of a list, pipe, substitution,
// group or newline is judged, quoted text and comments are not commands, and
// one level of sh -c or bash -c is unwrapped (never eval, xargs or an alias).
// git's own -C and -c options are read where git runs.
func ShellViolations(command string, coder bool) []ShellViolation {
	return uniqueRules(judgeLine(command, func(argv []string) []ShellViolation {
		return judgeCommand(argv, coder)
	}))
}

// uniqueRules keeps the first violation of each rule, in order.
func uniqueRules(vs []ShellViolation) []ShellViolation {
	var out []ShellViolation
	seen := map[string]bool{}
	for _, v := range vs {
		if !seen[v.Rule] {
			seen[v.Rule] = true
			out = append(out, v)
		}
	}
	return out
}

// shellSegments splits a command line into the words of each simple command.
// Operators (;, &, |, &&, ||, newline, parentheses, backquotes and $( ) end
// a command; quotes and backslashes make a word literal; a # at the start of
// a word comments out the rest of the line. Redirection operators stay
// inside their words, which are harmless to the rules. This is the rule view:
// every & ends a command, as the shell guard has always read it.
func shellSegments(s string) [][]string {
	cmds := shellCommands(s, false)
	segs := make([][]string, 0, len(cmds))
	for _, c := range cmds {
		segs = append(segs, c.words)
	}
	return segs
}

// simpleCommand is one simple command of a command line: its words, and the
// targets of its unquoted write redirections (intent 0028 S7, PD6).
type simpleCommand struct {
	words   []string
	targets []string
}

// shellCommands splits a command line into its simple commands, as
// shellSegments describes. The words are the same in both views, except where
// bash is true: a & that joins a redirection (&> or >&) does not end the
// command, and | or & right after > is part of the operator, so the word after
// it is that redirection's target, as bash reads it. The rules read bash false
// and the write rule reads bash true. Each target is recorded once, from the
// unquoted operator that precedes it.
func shellCommands(s string, bash bool) []simpleCommand {
	const (
		plain = iota
		single
		double
	)
	var cmds []simpleCommand
	var cur, targets []string
	var word []byte
	inWord := false
	state := plain
	// pending: the next word is the target of the operator just read.
	// targetAt: where in word an attached target starts, -1 when none.
	// redirOp: the operator of that target.
	pending, targetAt, redirOp := false, -1, ""
	recordTarget := func(t string) {
		if t == "" || (redirOp == ">&" && isFdOrClose(t)) {
			return
		}
		targets = append(targets, t)
	}
	flushWord := func() {
		if !inWord {
			return
		}
		cur = append(cur, string(word))
		switch {
		case targetAt >= 0:
			recordTarget(string(word[targetAt:]))
		case pending:
			recordTarget(string(word))
		}
		word = word[:0]
		inWord = false
		pending, targetAt, redirOp = false, -1, ""
	}
	flushSeg := func() {
		flushWord()
		if len(cur) > 0 {
			cmds = append(cmds, simpleCommand{words: cur, targets: targets})
		}
		cur, targets = nil, nil
		pending, targetAt, redirOp = false, -1, ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		// A command substitution is judged as its own command line, in
		// plain or double-quoted text alike; the outer word is kept.
		if state != single && (c == '`' || (c == '$' && i+1 < len(s) && s[i+1] == '(')) {
			body, end := substitution(s, i)
			flushSeg()
			cmds = append(cmds, shellCommands(body, bash)...)
			i = end
			continue
		}
		switch state {
		case single:
			if c == '\'' {
				state = plain
			} else {
				word = append(word, c)
			}
			continue
		case double:
			switch {
			case c == '"':
				state = plain
			case c == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`\n", s[i+1]) >= 0:
				i++
				if s[i] != '\n' {
					word = append(word, s[i])
				}
			default:
				word = append(word, c)
			}
			continue
		}
		switch c {
		case ' ', '\t', '\r':
			flushWord()
		case '\n', ';', '|', '(', ')':
			flushSeg()
		case '&':
			if bash && i+1 < len(s) && s[i+1] == '>' {
				// &> redirects both streams: the & joins the > that follows.
				word = append(word, c)
				inWord = true
			} else {
				flushSeg()
			}
		case '>', '<':
			// An input redirection and a here-document are plain words here.
			if c == '<' && (i+1 >= len(s) || s[i+1] != '>') {
				word = append(word, c)
				inWord = true
				break
			}
			start, j := i, i+1
			if c == '<' || (j < len(s) && s[j] == '>') {
				j++ // <> and >>
			}
			if bash && c == '>' && j < len(s) && (s[j] == '|' || s[j] == '&') {
				j++ // >| and >&
			}
			op := string(s[start:j])
			if targetAt >= 0 {
				recordTarget(string(word[targetAt:]))
				targetAt = -1
			}
			word = append(word, op...)
			inWord = true
			i = j - 1
			if j >= len(s) || strings.IndexByte(" \t\r\n;|&()<>", s[j]) >= 0 {
				// No target follows the operator in this word: the next word is it.
				flushWord()
				redirOp, pending = op, true
			} else {
				redirOp, targetAt = op, len(word)
			}
		case '$':
			word = append(word, c)
			inWord = true
		case '\'':
			state = single
			inWord = true
		case '"':
			state = double
			inWord = true
		case '\\':
			if i+1 < len(s) {
				i++
				if s[i] != '\n' {
					word = append(word, s[i])
					inWord = true
				}
			}
		case '#':
			if !inWord {
				for i+1 < len(s) && s[i+1] != '\n' {
					i++
				}
				continue
			}
			word = append(word, c)
			inWord = true
		default:
			word = append(word, c)
			inWord = true
		}
	}
	flushSeg()
	return cmds
}

// isFdOrClose reports whether the word after >& is a file descriptor (digits)
// or the close form (-), so that it is no file target.
func isFdOrClose(t string) bool {
	if t == "-" {
		return true
	}
	for _, r := range t {
		if r < '0' || r > '9' {
			return false
		}
	}
	return t != ""
}

// substitution returns the body of the command substitution that starts at
// s[i] ("$(" or a backquote) and the index of its closing byte. An unclosed
// substitution runs to the end of the line.
func substitution(s string, i int) (string, int) {
	if s[i] == '`' {
		for j := i + 1; j < len(s); j++ {
			if s[j] == '\\' {
				j++
				continue
			}
			if s[j] == '`' {
				return s[i+1 : j], j
			}
		}
		return s[i+1:], len(s) - 1
	}
	depth := 0
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[i+2 : j], j
			}
		}
	}
	return s[i+2:], len(s) - 1
}

// reservedWords start a command without being its name (if, then, ...).
var reservedWords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "do": true,
	"while": true, "until": true, "!": true, "{": true, "time": true,
}

// shellNames are the shells whose -c argument is one command line.
var shellNames = map[string]bool{"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true}

// judgeLine runs visit on each simple command of a line, in command order,
// and collects its violations. One level of sh -c or bash -c is read as its
// own command line (S6): its commands are visited, never unwrapped again.
func judgeLine(command string, visit func(argv []string) []ShellViolation) []ShellViolation {
	var out []ShellViolation
	for _, argv := range shellSegments(command) {
		out = append(out, judgeArgv(argv, visit, true)...)
	}
	return out
}

// judgeArgv visits one simple command, or with unwrap the commands of its
// one level of shell -c.
func judgeArgv(argv []string, visit func(argv []string) []ShellViolation, unwrap bool) []ShellViolation {
	argv = skipPrefix(argv)
	if len(argv) == 0 {
		return nil
	}
	if unwrap && shellNames[path.Base(argv[0])] {
		inner, ok := shellCommandString(argv[1:])
		if !ok {
			return nil
		}
		var out []ShellViolation
		for _, seg := range shellSegments(inner) {
			out = append(out, judgeArgv(seg, visit, false)...)
		}
		return out
	}
	return visit(argv)
}

// judgeCommand judges one shell-guard command by its name: git, and rm in a
// coder session.
func judgeCommand(argv []string, coder bool) []ShellViolation {
	switch path.Base(argv[0]) {
	case "git":
		return judgeGit(argv[1:], coder)
	case "rm":
		if coder {
			return judgeRm(argv[1:])
		}
	}
	return nil
}

// skipPrefix drops the reserved words and the VAR=value assignments that
// come before a command's name.
func skipPrefix(argv []string) []string {
	for len(argv) > 0 && (reservedWords[argv[0]] || isAssignment(argv[0])) {
		argv = argv[1:]
	}
	return argv
}

// isAssignment reports whether a word is a NAME=value assignment.
func isAssignment(w string) bool {
	eq := strings.IndexByte(w, '=')
	if eq <= 0 {
		return false
	}
	for i, r := range w[:eq] {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// shellCommandString returns the command line of a shell's -c option: the
// argument after the first option cluster that holds c, among the options
// before the first non-option word (a script name ends them). The options
// -o, +o and -O take the next word as their value, and --rcfile and
// --init-file do too.
func shellCommandString(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" || a == "-" || (!strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+")) {
			return "", false
		}
		if strings.HasPrefix(a, "--") {
			if a == "--rcfile" || a == "--init-file" {
				i++
			}
			continue
		}
		letters := a[1:]
		if strings.ContainsRune(letters, 'c') {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
		if letters != "" && strings.ContainsRune("oO", rune(letters[len(letters)-1])) {
			i++
		}
	}
	return "", false
}

// gitSubcommand returns git's subcommand and the words after it, skipping
// git's global options: -C <dir> and -c <name=value> take a value, the
// others stand alone.
func gitSubcommand(args []string) (string, []string, bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" || a == "--namespace":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return a, args[i+1:], true
		}
	}
	return "", nil, false
}

// judgeGit judges a git command by its subcommand.
func judgeGit(args []string, coder bool) []ShellViolation {
	sub, rest, ok := gitSubcommand(args)
	if !ok {
		return nil
	}
	switch sub {
	case "push":
		return judgePush(rest, coder)
	case "branch":
		if coder {
			return judgeBranch(rest)
		}
	case "worktree":
		if coder {
			return judgeWorktree(rest)
		}
	}
	return nil
}

// judgePush judges git push: a target of main is blocked in every session; in
// a coder session also master and dev, any force form and a + refspec. The
// remote is the first positional word, or the value of --repo when given.
func judgePush(args []string, coder bool) []ShellViolation {
	var positional []string
	force, repo, endOfOpts := false, false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case !endOfOpts && a == "--":
			endOfOpts = true
		case !endOfOpts && a == "--repo":
			repo = true
			i++
		case !endOfOpts && strings.HasPrefix(a, "--repo="):
			repo = true
		case !endOfOpts && strings.HasPrefix(a, "--"):
			if a == "--force" || a == "--mirror" || a == "--force-with-lease" || strings.HasPrefix(a, "--force-with-lease=") {
				force = true
			}
		case !endOfOpts && strings.HasPrefix(a, "-") && a != "-":
			if strings.ContainsRune(a[1:], 'f') {
				force = true
			}
		default:
			positional = append(positional, a)
		}
	}
	specs := positional
	if !repo {
		specs = positional[min(1, len(positional)):]
	}
	var out []ShellViolation
	// Each spec is a refspec (src:dst or dst).
	for _, spec := range specs {
		plus := strings.HasPrefix(spec, "+")
		dst := strings.TrimPrefix(spec, "+")
		if i := strings.LastIndexByte(dst, ':'); i >= 0 {
			dst = dst[i+1:]
		}
		dst = strings.TrimPrefix(dst, "refs/heads/")
		if dst == "main" {
			if coder {
				out = append(out, ShellViolation{Rule: ruleMainPush, Route: routeMainCoder})
			} else {
				out = append(out, ShellViolation{Rule: ruleMainPush, Route: routeMainOwner})
			}
		}
		if coder && (dst == "master" || dst == "dev") {
			out = append(out, ShellViolation{Rule: ruleDevPush, Route: routeDevPush})
		}
		if coder && plus {
			force = true
		}
	}
	if coder && force {
		out = append(out, ShellViolation{Rule: ruleForcePush, Route: routeForcePush})
	}
	return out
}

// judgeBranch judges git branch deletion with force: -D, or --delete with
// --force, or a cluster that holds both d and f.
func judgeBranch(args []string) []ShellViolation {
	del, force := false, false
	for _, a := range args {
		switch {
		case a == "-D":
			return []ShellViolation{{Rule: ruleBranchForce, Route: routeBranchForce}}
		case a == "--delete":
			del = true
		case a == "--force":
			force = true
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-") && len(a) > 1:
			if strings.ContainsRune(a[1:], 'D') {
				return []ShellViolation{{Rule: ruleBranchForce, Route: routeBranchForce}}
			}
			if strings.ContainsRune(a[1:], 'd') {
				del = true
			}
			if strings.ContainsRune(a[1:], 'f') {
				force = true
			}
		}
	}
	if del && force {
		return []ShellViolation{{Rule: ruleBranchForce, Route: routeBranchForce}}
	}
	return nil
}

// judgeWorktree judges git worktree remove with --force or -f.
func judgeWorktree(args []string) []ShellViolation {
	sub, rest, ok := gitSubcommand(args)
	if !ok || sub != "remove" {
		return nil
	}
	for _, a := range rest {
		if a == "--force" || (strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a[1:], 'f')) {
			return []ShellViolation{{Rule: ruleWorktreeForce, Route: routeWorktree}}
		}
	}
	return nil
}

// judgeRm judges rm with a recursive flag on a glob or under a *-worktrees
// folder. Without a recursive flag rm is allowed.
func judgeRm(args []string) []ShellViolation {
	recursive := false
	var paths []string
	endOfOpts := false
	for _, a := range args {
		switch {
		case !endOfOpts && a == "--":
			endOfOpts = true
		case !endOfOpts && a == "--recursive":
			recursive = true
		case !endOfOpts && strings.HasPrefix(a, "--"):
		case !endOfOpts && strings.HasPrefix(a, "-") && a != "-":
			if strings.ContainsAny(a[1:], "rR") {
				recursive = true
			}
		default:
			paths = append(paths, a)
		}
	}
	if !recursive {
		return nil
	}
	var out []ShellViolation
	glob, worktrees := false, false
	for _, p := range paths {
		if strings.ContainsAny(p, "*?[") {
			glob = true
		}
		if underWorktrees(p) {
			worktrees = true
		}
	}
	if glob {
		out = append(out, ShellViolation{Rule: ruleRmGlob, Route: routeRmGlob})
	}
	if worktrees {
		out = append(out, ShellViolation{Rule: ruleRmWorktrees, Route: routeRmWorktrees})
	}
	return out
}

// underWorktrees reports whether a path has a component named *-worktrees.
func underWorktrees(p string) bool {
	for _, c := range strings.Split(path.Clean(p), "/") {
		if strings.HasSuffix(c, "-worktrees") {
			return true
		}
	}
	return false
}
