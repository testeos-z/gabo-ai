package check

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// humanEntry is one human-only entry for a glob pattern: an entry of
// review.human_only_paths, or gabo.yaml (intent 0028 S13). A literal path is
// matched by humanOnlyPath (the reviewer's rule, shared with --pre-write); a
// glob is matched here against the entries from the root.
type humanEntry struct {
	// raw is the entry without its trailing slash, cleaned.
	raw string
	// dir is true when the entry ends with a slash: a folder and what is under it.
	dir bool
	// comps are the components of raw.
	comps []string
}

// humanEntries returns the entries of the list with gabo.yaml added, lowercased
// when fold is true (a case-insensitive filesystem).
func humanEntries(list []string, fold bool) []humanEntry {
	out := make([]humanEntry, 0, len(list)+1)
	for _, e := range append(slices.Clone(list), "gabo.yaml") {
		if fold {
			e = strings.ToLower(e)
		}
		raw := path.Clean(strings.TrimSuffix(e, "/"))
		if raw == "." {
			continue
		}
		out = append(out, humanEntry{raw: raw, dir: strings.HasSuffix(e, "/"), comps: strings.Split(raw, "/")})
	}
	return out
}

// glob reports whether the glob pattern p can match a path the entry covers:
// the pattern's components are matched against the entry's from the root. A
// folder entry covers what lies under it, so the pattern may be longer; a file
// entry is one path, so the pattern must have the same number of components.
func (h humanEntry) glob(p string) bool {
	pc := strings.Split(p, "/")
	n := len(h.comps)
	if (h.dir && len(pc) < n) || (!h.dir && len(pc) != n) {
		return false
	}
	for i, c := range h.comps {
		if !globMatch(pc[i], c) {
			return false
		}
	}
	return true
}

// ancestor reports whether the clean, project-relative path p is a folder that
// contains the entry: the root contains every entry. Removing or moving such a
// folder takes the entry with it (intent 0028 S7, review round 3).
func (h humanEntry) ancestor(p string) bool {
	return p == "." || strings.HasPrefix(h.raw, p+"/")
}

// ancestorGlob reports whether the glob pattern p can match a folder that
// contains the entry: the pattern is no deeper than the entry, and its
// components match the entry's from the root.
func (h humanEntry) ancestorGlob(p string) bool {
	pc := strings.Split(p, "/")
	if len(pc) > len(h.comps) {
		return false
	}
	for i := range pc {
		if !globMatch(pc[i], h.comps[i]) {
			return false
		}
	}
	return true
}

// globMatch matches one path component; a malformed pattern matches (fail closed).
func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return ok || err != nil
}

// humanGlobRel reports whether the clean, project-relative glob pattern rel can
// match an entry of hs.
func humanGlobRel(rel string, hs []humanEntry) bool {
	for _, h := range hs {
		if h.glob(rel) {
			return true
		}
	}
	return false
}

// humanAncestorRel reports whether the clean, project-relative path rel is a
// folder that contains an entry of hs, literally or as a glob.
func humanAncestorRel(rel string, hs []humanEntry) bool {
	glob := strings.ContainsAny(rel, "*?[")
	for _, h := range hs {
		if (glob && h.ancestorGlob(rel)) || (!glob && h.ancestor(rel)) {
			return true
		}
	}
	return false
}

// shellHumanTarget reports whether a write to the path p, as a shell command
// spells it, is a write to a human-only path: a target of p (as written, or
// with its symlinks resolved) is human-only by humanOnlyPath, or is a glob that
// can match an entry. A path that cannot be resolved counts as human-only
// (fail closed).
func (u *UseCase) shellHumanTarget(p string, list []string, hs []humanEntry) bool {
	targets, err := u.targetsOf(p)
	if err != nil {
		return true
	}
	for _, rel := range targets {
		// The folder itself is a target too: rel with a slash matches a folder entry.
		if humanOnlyPath(rel, list, u.s.FoldCase) || humanOnlyPath(rel+"/", list, u.s.FoldCase) {
			return true
		}
		if strings.ContainsAny(rel, "*?[") && humanGlobRel(rel, hs) {
			return true
		}
	}
	return false
}

// shellAncestorTarget reports whether a cp, mv or rm of the path p targets a
// folder that contains a human-only entry, the project root among them. The
// root is checked on the path before targets drop the paths outside it, so
// ".." and "/" count as containing the root (intent 0028 S7, review round 4).
func (u *UseCase) shellAncestorTarget(p string, hs []humanEntry) bool {
	if abs := u.absOf(p); filepath.IsAbs(abs) && u.containsRoot(abs) {
		return true
	}
	targets, err := u.targetsOf(p)
	if err != nil {
		return true
	}
	for _, rel := range targets {
		if u.s.FoldCase {
			rel = strings.ToLower(rel)
		}
		if humanAncestorRel(rel, hs) {
			return true
		}
	}
	return false
}

// containsRoot reports whether the absolute path abs is the project root or one
// of its ancestors, compared without case on a case-insensitive filesystem.
func (u *UseCase) containsRoot(abs string) bool {
	if u.s.Root == "" {
		return false
	}
	root := path.Clean(filepath.ToSlash(u.s.Root))
	a := path.Clean(filepath.ToSlash(abs))
	if u.s.FoldCase {
		root, a = strings.ToLower(root), strings.ToLower(a)
	}
	return strings.HasPrefix(root+"/", strings.TrimSuffix(a, "/")+"/")
}
