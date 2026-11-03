package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// ErrPreWriteUsage marks a usage error of the pre-write hook mode.
var ErrPreWriteUsage = errors.New("usage: gabo check --pre-write <path>...")

// PreWrite is the hook mode (intent 0001 S11): it judges pending writes to
// paths with the intent stage gate before they happen, and writes nothing.
// Each path is relative to the project root or absolute inside it; a path
// outside the root, or one that is not docs/product/intents/<dir>/plan.md
// (compared without case, for case-insensitive filesystems), passes. A path
// is judged as written and with its symlinks resolved, so a symlink can
// neither hide a plan nor move it outside the root. A plan.md is blocked
// while its folder's spec.md is missing or not accepted; a spec.md that
// cannot be read counts as not accepted (fail closed). The report holds one
// "✗ <path>: <reason>" line per blocked path, in argument order.
func (u *UseCase) PreWrite(paths []string) (Report, error) {
	var r Report
	if err := checkPreWritePaths(paths); err != nil {
		return r, err
	}
	for _, p := range paths {
		plan, ok, err := u.gatedPlan(p)
		if err != nil {
			return Report{}, err
		}
		if !ok {
			continue
		}
		reason, err := u.specGate(path.Dir(plan))
		if err != nil {
			reason = "spec.md cannot be read (" + err.Error() + "); it counts as not accepted"
		}
		if reason != "" {
			r.fail("✗ " + plan + ": " + reason)
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

// gatedPlan returns the project-relative plan.md that a write to p would
// create or change, first as written, then with its symlinks resolved.
func (u *UseCase) gatedPlan(p string) (string, bool, error) {
	if filepath.IsAbs(p) && u.s.Root == "" {
		return "", false, fmt.Errorf("%w: absolute path %s with no project root", ErrPreWriteUsage, p)
	}
	if rel, ok := relInside(u.s.Root, p); ok && isPlan(rel) {
		return rel, true, nil
	}
	if u.s.Resolve == nil || u.s.Root == "" {
		return "", false, nil
	}
	root, err := u.s.Resolve(u.s.Root)
	if err != nil {
		return "", false, nil
	}
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(u.s.Root, p)
	}
	target, ok := u.resolveExisting(abs)
	if !ok {
		return "", false, nil
	}
	if rel, ok := relInside(root, target); ok && isPlan(rel) {
		return rel, true, nil
	}
	return "", false, nil
}

// resolveExisting resolves the symlinks of the longest existing prefix of
// abs and appends the rest; a plan.md being created does not exist yet.
func (u *UseCase) resolveExisting(abs string) (string, bool) {
	dir, rest := filepath.Clean(abs), ""
	for {
		real, err := u.s.Resolve(dir)
		if err == nil {
			return filepath.Join(real, rest), true
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		rest = filepath.Join(filepath.Base(dir), rest)
		dir = parent
	}
}

// relInside returns p as a clean slash-separated path relative to root,
// and whether it lies inside root. A relative p is taken as relative to
// root already.
func relInside(root, p string) (string, bool) {
	rel := filepath.Clean(p)
	if filepath.IsAbs(rel) {
		var err error
		if rel, err = filepath.Rel(filepath.Clean(root), rel); err != nil {
			return "", false
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
