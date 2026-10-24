// Package worktree closes finished coder worktrees: only approved ones
// (merged into the integration branch, clean, pushed, without ignored
// secrets or an open PR, never the main one), never with --force, and it
// deletes the merged local branch with `git branch -d`.
package worktree

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// CLOSED reasons.
const (
	ReasonInBase   = "in-base"
	ReasonMergedPR = "merged-pr"
)

// PathFacts locate the target. Every path except Arg is absolute and
// symlink-free; "" means it does not exist.
type PathFacts struct {
	Arg          string // the path as given
	Real         string
	Symlink      bool   // the last component of Arg is a symlink
	Main         string // the main worktree
	WorktreesDir string // <main>-worktrees, the sibling of the main checkout
	Current      string // the worktree the command runs in
	Registered   bool   // a linked (non-main) worktree of this repository
	Keep         []string
}

// CheckPath returns why the target may not be closed, or "".
func CheckPath(f PathFacts) string {
	switch {
	case !filepath.IsAbs(f.Arg):
		return "the path is not absolute"
	case slices.Contains(strings.Split(f.Arg, "/"), ".."):
		return "paths with .. are not accepted"
	case f.Symlink:
		return "the path is a symlink"
	case f.Real == "":
		return "the path does not exist"
	case f.Real == f.Main:
		return "the main worktree is never closed"
	case f.WorktreesDir == "":
		return "the worktrees directory " + f.Main + "-worktrees does not exist"
	case filepath.Dir(f.Real) != f.WorktreesDir || filepath.Base(f.Real) != filepath.Base(filepath.Clean(f.Arg)):
		return "not directly in the worktrees directory " + f.WorktreesDir + " (or a symlink escape)"
	case f.Real == f.Current:
		return "this is the worktree the command runs in; run it from another checkout"
	case !f.Registered:
		return "not a worktree of this repository"
	}
	for _, k := range f.Keep {
		if k == filepath.Base(f.Real) || k == f.Real || k == filepath.Clean(f.Arg) {
			return "on the keep list (worktree.keep)"
		}
	}
	return ""
}

// StateFacts describe the target worktree and its branch after a fresh
// fetch of origin/<Integration>.
type StateFacts struct {
	Integration string
	Branch      string // short name; "" when detached
	Head        string
	BaseTip     string // origin/<Integration>
	OwnCommits  bool   // the branch's reflog holds more than its creation
	Changed     int    // tracked changes and untracked files
	Ignored     []string
	HasUpstream bool
	Unpushed    int // commits ahead of the upstream
	OpenPR      int // an open PR of the branch; 0 when none
	InBase      bool
	MergedPR    int // a PR merged into Integration whose head holds HEAD; 0 when none
}

// Decision is the outcome of Decide. Reason is ReasonInBase or
// ReasonMergedPR when Close, else why the worktree is kept.
type Decision struct {
	Close  bool
	Reason string
}

func refuse(format string, a ...any) Decision { return Decision{Reason: fmt.Sprintf(format, a...)} }

// Decide approves closing a worktree only when every check passes.
func Decide(f StateFacts) Decision {
	if f.Changed > 0 {
		return refuse("not clean: %d changed or untracked path(s)", f.Changed)
	}
	for _, p := range f.Ignored {
		if IsSecret(p) {
			return refuse("ignored secret or local data would be lost: %s", p)
		}
	}
	if f.HasUpstream && f.Unpushed > 0 {
		return refuse("has %d unpushed commit(s)", f.Unpushed)
	}
	if f.Branch != "" && f.OpenPR > 0 {
		return refuse("has an open PR #%d", f.OpenPR)
	}
	merged := f.Branch != "" && f.MergedPR > 0
	if !merged {
		if f.Head == f.BaseTip {
			return refuse("never worked: HEAD is the origin/%s tip (a worker may be starting in it)", f.Integration)
		}
		if f.Branch != "" && !f.OwnCommits {
			return refuse("never worked: %s has no commits of its own (a worker may be starting in it)", f.Branch)
		}
	}
	switch {
	case f.InBase:
		return Decision{Close: true, Reason: ReasonInBase}
	case merged:
		return Decision{Close: true, Reason: ReasonMergedPR}
	}
	name := f.Branch
	if name == "" {
		name = "detached HEAD"
	}
	return refuse("not merged: %s is not in origin/%s and no merged PR into %s holds its HEAD", name, f.Integration, f.Integration)
}

var secretPattern = regexp.MustCompile(`^(\.env.*|id_.*|.*\.(pem|key|p12|pfx|sqlite|sqlite3|sqlite-journal|db|db-wal|db-shm|db-journal))$`)

// IsSecret reports whether an ignored path looks like a secret or local
// data (.env*, keys, certificates, SQLite databases) by its base name.
func IsSecret(path string) bool {
	return secretPattern.MatchString(strings.ToLower(filepath.Base(path)))
}

// OriginMatches reports whether url is the GitHub remote of repo (owner/name).
func OriginMatches(url, repo string) bool {
	re := `(?i)^(git@github\.com:|https://github\.com/|ssh://git@github\.com/)` + regexp.QuoteMeta(repo) + `(\.git)?/?$`
	return regexp.MustCompile(re).MatchString(url)
}
