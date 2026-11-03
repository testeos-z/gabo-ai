package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ErrPreWriteUsage marks a usage error of the pre-write hook mode.
var ErrPreWriteUsage = errors.New("usage: gabo check --pre-write <path>...")

// PreWrite is the hook mode (intent 0001 S11): it judges pending writes to
// paths with the intent stage gate before they happen, and writes nothing.
// Each path is relative to the working directory or absolute inside the
// project root; a path outside the root (its root prefix compared without
// case only on a case-insensitive filesystem, Settings.FoldCase), or one
// that is not docs/product/intents/<dir>/plan.md (compared without case),
// passes. A path is judged as written and at its symlink-resolved target,
// each ".." applied after the symlinks before it (a dangling link at the
// target it would create), and blocked when
// either is a plan.md whose folder's spec.md is missing or not accepted. A
// spec.md that cannot be read, or a path whose symlinks cannot be resolved,
// counts as not accepted (fail closed). The report holds one
// "✗ <path>: <reason>" line per blocked path, in argument order.
func (u *UseCase) PreWrite(paths []string) (Report, error) {
	var r Report
	if err := checkPreWritePaths(paths); err != nil {
		return r, err
	}
	for _, p := range paths {
		if filepath.IsAbs(p) && u.s.Root == "" {
			return Report{}, fmt.Errorf("%w: absolute path %s with no project root", ErrPreWriteUsage, p)
		}
		plans, err := u.plansOf(p)
		if err != nil {
			shown := p
			if rel, ok := relInside(u.s.Root, u.absOf(p), u.s.FoldCase); ok {
				shown = rel
			}
			r.fail("✗ " + shown + ": its symlinks cannot be resolved (" + err.Error() + "); it counts as not accepted")
			continue
		}
		for _, plan := range plans {
			reason, err := u.specGate(path.Dir(plan))
			if err != nil {
				reason = "spec.md cannot be read (" + err.Error() + "); it counts as not accepted"
			}
			if reason != "" {
				r.fail("✗ " + plan + ": " + reason)
				break
			}
		}
	}
	return r, nil
}

// checkPreWritePaths rejects an empty path list or an empty path.
func checkPreWritePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("%w: no path given", ErrPreWriteUsage)
	}
	for _, p := range paths {
		if p == "" {
			return fmt.Errorf("%w: empty path", ErrPreWriteUsage)
		}
	}
	return nil
}

// maxLinks bounds how many dangling symlinks one path may follow, like the
// kernel's limit on a symlink chain.
const maxLinks = 40

// absOf returns p made absolute against the working directory (Settings
// WorkDir, else Root) and cleaned lexically; with neither it returns p
// unchanged. It is the path as written: plansOf resolves the raw path.
func (u *UseCase) absOf(p string) string {
	return filepath.Clean(u.rawAbsOf(p))
}

// rawAbsOf returns p made absolute like absOf but not cleaned, so its ".."
// components can be applied after the symlinks before them.
func (u *UseCase) rawAbsOf(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	base := u.s.WorkDir
	if base == "" {
		base = u.s.Root
	}
	if base == "" {
		return p
	}
	return base + string(filepath.Separator) + p
}

// plansOf returns the project-relative plan.md files a write to p would
// create or change: p as written and p with its symlinks resolved. An error
// means the symlinks could not be resolved.
func (u *UseCase) plansOf(p string) ([]string, error) {
	var plans []string
	if rel, ok := relInside(u.s.Root, u.absOf(p), u.s.FoldCase); ok && isPlan(rel) {
		plans = append(plans, rel)
	}
	if u.s.Resolve == nil || u.s.Root == "" {
		return plans, nil
	}
	root, err := u.s.Resolve(u.s.Root)
	if err != nil {
		return nil, err
	}
	target, ok, err := u.resolveRaw(u.rawAbsOf(p))
	if err != nil {
		return nil, err
	}
	if !ok {
		return plans, nil
	}
	if rel, ok := relInside(root, target, u.s.FoldCase); ok && isPlan(rel) && !slices.Contains(plans, rel) {
		plans = append(plans, rel)
	}
	return plans, nil
}

// resolveRaw resolves the absolute, uncleaned path abs component by
// component as the kernel does: each existing component has its symlinks
// resolved before the next ".." applies, so lnk/../x climbs from lnk's
// target. A dangling symlink is followed to its target, since a write
// through it creates that target; a missing component is kept as written
// (a plan.md being created does not exist yet). It reports false when no
// component exists, and any error but a missing entry, including too many
// links.
func (u *UseCase) resolveRaw(abs string) (string, bool, error) {
	vol := filepath.VolumeName(abs)
	cur := vol + string(filepath.Separator)
	comps := splitPath(abs[len(vol):])
	found := false
	for links := 0; len(comps) > 0; {
		c := comps[0]
		comps = comps[1:]
		switch c {
		case ".":
			continue
		case "..":
			// cur holds no symlink, so its parent is lexical.
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, c)
		real, err := u.s.Resolve(next)
		if err == nil {
			cur, found = real, true
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false, err
		}
		target, isLink, err := u.readlink(next)
		if err != nil {
			return "", false, err
		}
		if !isLink {
			cur = next
			continue
		}
		if links++; links > maxLinks {
			return "", false, fmt.Errorf("%s: too many links", abs)
		}
		if tv := filepath.VolumeName(target); filepath.IsAbs(target) {
			cur = tv + string(filepath.Separator)
			target = target[len(tv):]
		}
		comps = append(splitPath(target), comps...)
	}
	return cur, found, nil
}

// splitPath returns the non-empty components of p, keeping "." and "..".
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == filepath.Separator })
}

// readlink reports whether p is a symlink and its target; a missing p is no
// link. Without Settings.Readlink nothing is a link.
func (u *UseCase) readlink(p string) (string, bool, error) {
	if u.s.Readlink == nil {
		return "", false, nil
	}
	target, isLink, err := u.s.Readlink(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	return target, isLink, err
}

// relInside returns p as a clean slash-separated path relative to root,
// and whether it lies inside root. With fold (a case-insensitive
// filesystem) the root prefix is compared without case. A relative p is
// taken as relative to root already; callers pass absolute paths
// (UseCase.absOf).
func relInside(root, p string, fold bool) (string, bool) {
	rel := filepath.Clean(p)
	if filepath.IsAbs(rel) {
		root = filepath.Clean(root)
		abs := rel
		var err error
		if rel, err = filepath.Rel(root, abs); err != nil {
			return "", false
		}
		prefix := strings.TrimSuffix(root, string(filepath.Separator)) + string(filepath.Separator)
		if fold && len(abs) > len(prefix) && strings.EqualFold(abs[:len(prefix)], prefix) {
			rel = abs[len(prefix):]
		}
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// isPlan reports whether rel is docs/product/intents/<dir>/plan.md,
// without case.
func isPlan(rel string) bool {
	return strings.EqualFold(path.Base(rel), "plan.md") && strings.EqualFold(path.Dir(path.Dir(rel)), IntentsDir)
}
