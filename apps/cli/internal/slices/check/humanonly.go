package check

import (
	"path"
	"slices"
	"strings"
)

// RouteHumanEdits is the route of a path a coder may not write (intent 0028 S7, S11).
const RouteHumanEdits = "a human edits this path"

// humanEntry is one human-only entry: an entry of review.human_only_paths, or
// gabo.yaml, which is protected whatever the list says (intent 0028 S13).
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

// literal reports whether the clean, project-relative path p is the entry or,
// for a folder entry, lies under it. A file entry names one file at the root;
// the same name in a folder is the reviewer's merge check to stop, not this rule.
func (h humanEntry) literal(p string) bool {
	if h.dir {
		return p == h.raw || strings.HasPrefix(p, h.raw+"/")
	}
	return p == h.raw
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

// humanAncestorTarget reports whether a cp, mv or rm of the path p targets a
// folder that contains a human-only entry (the root, or such a folder), read as
// humanOnlyTarget reads it. A path that cannot be resolved counts as human-only.
func (u *UseCase) humanAncestorTarget(p string, hs []humanEntry) bool {
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

// globMatch matches one path component; a malformed pattern matches (fail closed).
func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return ok || err != nil
}

// humanOnlyRel reports whether the clean, project-relative path rel is under an
// entry of hs, as a literal path or as a glob that can match one.
func humanOnlyRel(rel string, hs []humanEntry) bool {
	glob := strings.ContainsAny(rel, "*?[")
	for _, h := range hs {
		if (glob && h.glob(rel)) || (!glob && h.literal(rel)) {
			return true
		}
	}
	return false
}

// HumanOnlyPath reports whether a write to the path p is a write to a
// human-only path: under an entry of list, or gabo.yaml (intent 0028 S7, S13).
// The path is read as PreWrite reads it: relative to the working directory,
// with its symlinks resolved, so both the path as written and its target are
// judged. A path that cannot be resolved counts as human-only (fail closed).
func (u *UseCase) HumanOnlyPath(p string, list []string) bool {
	return u.humanOnlyTarget(p, humanEntries(list, u.s.FoldCase))
}

// humanOnlyTarget is HumanOnlyPath with the entries already built.
func (u *UseCase) humanOnlyTarget(p string, hs []humanEntry) bool {
	targets, err := u.targetsOf(p)
	if err != nil {
		return true
	}
	for _, rel := range targets {
		if u.s.FoldCase {
			rel = strings.ToLower(rel)
		}
		if humanOnlyRel(rel, hs) {
			return true
		}
	}
	return false
}
