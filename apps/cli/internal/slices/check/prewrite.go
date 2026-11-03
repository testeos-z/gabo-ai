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
// Each path is relative to the project root or absolute inside it; a path
// outside the root, or one that is not docs/product/intents/<dir>/plan.md
// (compared without case, for case-insensitive filesystems), passes. A path
// is judged as written and at its symlink-resolved target, and blocked when
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
			if rel, ok := relInside(u.s.Root, p); ok {
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

// plansOf returns the project-relative plan.md files a write to p would
// create or change: p as written and p with its symlinks resolved. An error
// means the symlinks could not be resolved.
func (u *UseCase) plansOf(p string) ([]string, error) {
	var plans []string
	if rel, ok := relInside(u.s.Root, p); ok && isPlan(rel) {
		plans = append(plans, rel)
	}
	if u.s.Resolve == nil || u.s.Root == "" {
		return plans, nil
	}
	root, err := u.s.Resolve(u.s.Root)
	if err != nil {
		return nil, err
	}
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(u.s.Root, p)
	}
	target, ok, err := u.resolveExisting(abs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return plans, nil
	}
	if rel, ok := relInside(root, target); ok && isPlan(rel) && !slices.Contains(plans, rel) {
		plans = append(plans, rel)
	}
	return plans, nil
}

// resolveExisting resolves the symlinks of the longest existing prefix of
// abs and appends the rest; a plan.md being created does not exist yet. It
// reports false when no prefix exists, and any error but a missing entry.
func (u *UseCase) resolveExisting(abs string) (string, bool, error) {
	dir, rest := filepath.Clean(abs), ""
	for {
		real, err := u.s.Resolve(dir)
		if err == nil {
			return filepath.Join(real, rest), true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		rest = filepath.Join(filepath.Base(dir), rest)
		dir = parent
	}
}

// relInside returns p as a clean slash-separated path relative to root,
// and whether it lies inside root (its prefix compared without case). A relative p is taken as relative to
// root already.
func relInside(root, p string) (string, bool) {
	rel := filepath.Clean(p)
	if filepath.IsAbs(rel) {
		root = filepath.Clean(root)
		abs := rel
		var err error
		if rel, err = filepath.Rel(root, abs); err != nil {
			return "", false
		}
		// A case-insensitive filesystem may spell the root in another case.
		prefix := strings.TrimSuffix(root, string(filepath.Separator)) + string(filepath.Separator)
		if len(abs) > len(prefix) && strings.EqualFold(abs[:len(prefix)], prefix) {
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
