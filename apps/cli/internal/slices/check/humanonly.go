package check

import (
	"path"
	"path/filepath"
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
	// dir is true when the entry ends with a slash: a folder and what lies under it.
	dir bool
	// comps are the components of raw.
	comps []string
}

// humanEntries returns the entries of the list, with gabo.yaml added.
func humanEntries(list []string) []humanEntry {
	out := make([]humanEntry, 0, len(list)+1)
	for _, e := range append(slices.Clone(list), "gabo.yaml") {
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

// globMatch matches one path component; a malformed pattern matches (fail closed).
func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return ok || err != nil
}

// humanOnlyMatch reports whether a write to the path p is a write to a human-only
// path. An absolute p is read against root (inside it, with case folded when
// fold is true); a relative p is read from the root.
func humanOnlyMatch(p, root string, fold bool, hs []humanEntry) bool {
	rel, ok := projectRel(p, root, fold)
	if !ok {
		return false
	}
	glob := strings.ContainsAny(rel, "*?[")
	for _, h := range hs {
		if (glob && h.glob(rel)) || (!glob && h.literal(rel)) {
			return true
		}
	}
	return false
}

// projectRel returns p as a clean, slash-separated path relative to the project
// root, and whether it lies inside the root. An empty root is unknown: no
// absolute path is inside it.
func projectRel(p, root string, fold bool) (string, bool) {
	if filepath.IsAbs(p) {
		if root == "" {
			return "", false
		}
		return relInside(root, p, fold)
	}
	rel := path.Clean(filepath.ToSlash(p))
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// HumanOnlyPath reports whether a write to the path p is a write to a
// human-only path: under an entry of list, or gabo.yaml (intent 0028 S7, S13).
// The path is read as HumanOnlyShellWrites reads the paths of a shell command.
func HumanOnlyPath(p, root string, fold bool, list []string) bool {
	return humanOnlyMatch(p, root, fold, humanEntries(list))
}
