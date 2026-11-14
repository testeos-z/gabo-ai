package check

import (
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// The release-gate rules and their routes (intent 0028 S5, S11, D4). The
// command list is the one the repository ships: the tag, plugin tag and
// release steps of plugins/claude-code/README.md, the publish of
// .github/workflows/publish-pi.yml (bun publish) and the npm publish of the
// same checklist. The rules judge every agent session.
const (
	ruleTag         = "git tag that creates a tag is blocked in an agent session"
	ruleTagsPush    = "git push --tags is blocked in an agent session"
	rulePublish     = "a package publish (npm publish, bun publish) is blocked in an agent session"
	rulePluginTag   = "claude plugin tag is blocked in an agent session"
	ruleRelease     = "gh release create is blocked in an agent session"
	ruleMergeMain   = "a merge of a pull request into main is blocked in an agent session"
	ruleMergeCoder  = "gh pr merge is blocked in a coder session"
	ruleMergeUnread = "a merge whose pull request base cannot be read is blocked"
	routeOwner      = "the owner runs this outside agent sessions"
	routeCoderMerge = "a coder never merges; the reviewer merges a clean pull request into dev"
	routeUnread     = "retry when the host answers; the owner merges into main outside agent sessions"
)

// mainBranch is the branch no agent session merges into (D4).
const mainBranch = "main"

// mergeCall is one gh pr merge of a line: the -R repository ("" for the
// current one) and the pull request as the command names it ("" for the
// current branch's pull request).
type mergeCall struct {
	Repo, Ref string
}

// releaseReading is the release gate's reading of one command line: the
// violations the text decides, and the merges whose pull request base the
// host must answer (a coder merge is decided by the text, never read).
type releaseReading struct {
	Violations []ShellViolation
	Merges     []mergeCall
}

// releaseLine reads one command line for the release gate, every command of
// the line and one level of shell -c, as ShellViolations does.
func releaseLine(command string, coder bool) releaseReading {
	var r releaseReading
	r.Violations = uniqueRules(judgeLine(command, func(argv []string) []ShellViolation {
		if m, ok := mergeOf(argv); ok {
			if coder {
				return []ShellViolation{{Rule: ruleMergeCoder, Route: routeCoderMerge}}
			}
			r.Merges = append(r.Merges, m)
			return nil
		}
		return judgeRelease(argv)
	}))
	return r
}

// judgeRelease judges one simple command by the release rules that need no
// host answer: a tag that creates, git push --tags, a publish, a plugin tag
// and a release creation.
func judgeRelease(argv []string) []ShellViolation {
	var rule string
	switch path.Base(argv[0]) {
	case "git":
		rule = releaseGit(argv[1:])
	case "npm", "bun":
		if slices.Contains(argv[1:], "publish") {
			rule = rulePublish
		}
	case "claude":
		if adjacent(argv[1:], "plugin", "tag") {
			rule = rulePluginTag
		}
	case "gh":
		if _, words := ghArgs(argv[1:]); len(words) >= 2 && words[0] == "release" && words[1] == "create" {
			rule = ruleRelease
		}
	}
	if rule == "" {
		return nil
	}
	return []ShellViolation{{Rule: rule, Route: routeOwner}}
}

// releaseGit returns the release rule a git command breaks, or "".
func releaseGit(args []string) string {
	sub, rest, ok := gitSubcommand(args)
	if !ok {
		return ""
	}
	switch sub {
	case "tag":
		if tagCreates(rest) {
			return ruleTag
		}
	case "push":
		if hasFlag(rest, "--tags") {
			return ruleTagsPush
		}
	}
	return ""
}

// tagListLong are the long git tag options that list or filter tags, so no
// tag is created; tagValueLong take a value as the next word, unless the value
// follows an '='. The short options that list, delete or verify are l, n, d, v.
var (
	tagListLong  = map[string]bool{"--list": true, "--delete": true, "--verify": true, "--contains": true, "--no-contains": true, "--points-at": true, "--merged": true, "--no-merged": true}
	tagValueLong = map[string]bool{"--message": true, "--file": true, "--local-user": true, "--sort": true, "--format": true, "--column": true}
)

// tagCreates reports whether git tag names a tag to create: a positional name
// and no list, filter, delete or verify option. The value of -m, -F and -u
// (and of their long forms) is not a name, so `git tag -m msg v1` creates v1.
func tagCreates(args []string) bool {
	name, list, endOfOpts := false, false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case !endOfOpts && a == "--":
			endOfOpts = true
		case !endOfOpts && strings.HasPrefix(a, "--"):
			long, _, hasValue := strings.Cut(a, "=")
			if tagListLong[long] {
				list = true
			}
			if !hasValue && tagValueLong[long] {
				i++
			}
		case !endOfOpts && strings.HasPrefix(a, "-") && a != "-":
			letters := a[1:]
			if strings.ContainsAny(letters, "lndv") {
				list = true
			}
			if k := strings.IndexAny(letters, "mFu"); k >= 0 && k == len(letters)-1 {
				i++
			}
		default:
			name = true
		}
	}
	return name && !list
}

// hasFlag reports whether flag stands among args before a "--".
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == flag {
			return true
		}
	}
	return false
}

// adjacent reports whether a is directly followed by b in args.
func adjacent(args []string, a, b string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}

// ghArgs returns the -R or --repo value of a gh command and its words that are
// not options; an option's value is not a word.
func ghArgs(args []string) (string, []string) {
	var repo string
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-R" || a == "--repo":
			if i+1 < len(args) {
				repo = args[i+1]
			}
			i++
		case strings.HasPrefix(a, "--repo="):
			repo = strings.TrimPrefix(a, "--repo=")
		case strings.HasPrefix(a, "-R") && len(a) > 2:
			repo = a[2:]
		case strings.HasPrefix(a, "-") && a != "-":
		default:
			words = append(words, a)
		}
	}
	return repo, words
}

// mergeOf reports whether argv is a gh pr merge and the pull request it names.
func mergeOf(argv []string) (mergeCall, bool) {
	if path.Base(argv[0]) != "gh" {
		return mergeCall{}, false
	}
	repo, words := ghArgs(argv[1:])
	if len(words) < 2 || words[0] != "pr" || words[1] != "merge" {
		return mergeCall{}, false
	}
	m := mergeCall{Repo: repo}
	if len(words) > 2 {
		m.Ref = words[2]
	}
	return m, true
}

// errNoPullRequests is the base read of a settings without a host port: such
// a merge counts as unreadable (D4).
var errNoPullRequests = errors.New("no pull request port is wired")

// readBase asks the host port for the base branch of a merge's pull request.
func (u *UseCase) readBase(m mergeCall) (string, error) {
	if u.s.PullRequests == nil {
		return "", errNoPullRequests
	}
	return u.s.PullRequests.PRBase(m.Repo, m.Ref)
}

// judgeReleaseGate judges the release_gate lock of one command line (intent
// 0028 S5, D4): the text rules of releaseLine, then each merge's pull request
// base, read once per distinct merge. A base of main, or a base that cannot
// be read, is a violation. With the lock off nothing is read.
func (u *UseCase) judgeReleaseGate(cfg config.Config, command string, coder bool) ShellVerdict {
	if cfg.Lock(config.LockReleaseGate) == config.LockOff {
		return ShellVerdict{}
	}
	r := releaseLine(command, coder)
	vs := r.Violations
	read := map[mergeCall]bool{}
	for _, m := range r.Merges {
		if read[m] {
			continue
		}
		read[m] = true
		base, err := u.readBase(m)
		switch {
		case err != nil || base == "":
			vs = append(vs, ShellViolation{Rule: ruleMergeUnread, Route: routeUnread})
		case base == mainBranch:
			vs = append(vs, ShellViolation{Rule: ruleMergeMain, Route: routeOwner})
		}
	}
	return u.judgeLock(cfg, config.LockReleaseGate, uniqueRules(vs))
}
