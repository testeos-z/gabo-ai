package check

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ErrPreWriteUsage marks a usage error of the pre-write hook mode.
var ErrPreWriteUsage = errors.New("usage: gabo check --pre-write <path>...")

// PreWrite is the hook mode (intent 0001 S11): it judges pending writes to
// paths with the intent stage gate before they happen, and writes nothing.
// Each path is relative to the project root or absolute inside it; a path
// outside the root, or one that is not docs/product/intents/<dir>/plan.md,
// passes. A plan.md is blocked while its folder's spec.md is missing or not
// accepted; a spec.md that cannot be read counts as not accepted (fail
// closed). The report holds one "✗ <path>: <reason>" line per blocked path,
// in argument order, and nothing else.
func (u *UseCase) PreWrite(paths []string) (Report, error) {
	var r Report
	if len(paths) == 0 {
		return r, fmt.Errorf("%w: no path given", ErrPreWriteUsage)
	}
	for _, p := range paths {
		rel, inside, err := u.projectPath(p)
		if err != nil {
			return Report{}, err
		}
		if !inside || path.Base(rel) != "plan.md" || path.Dir(path.Dir(rel)) != IntentsDir {
			continue
		}
		reason, err := u.specGate(path.Dir(rel))
		if err != nil {
			reason = "spec.md cannot be read (" + err.Error() + "); it counts as not accepted"
		}
		if reason != "" {
			r.fail("✗ " + rel + ": " + reason)
		}
	}
	return r, nil
}

// projectPath returns p as a clean slash-separated path relative to the
// project root, and whether it lies inside the root.
func (u *UseCase) projectPath(p string) (string, bool, error) {
	if p == "" {
		return "", false, fmt.Errorf("%w: empty path", ErrPreWriteUsage)
	}
	rel := filepath.Clean(p)
	if filepath.IsAbs(rel) {
		if u.s.Root == "" {
			return "", false, fmt.Errorf("%w: absolute path %s with no project root", ErrPreWriteUsage, p)
		}
		var err error
		if rel, err = filepath.Rel(filepath.Clean(u.s.Root), rel); err != nil {
			return "", false, nil
		}
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false, nil
	}
	return rel, true, nil
}
