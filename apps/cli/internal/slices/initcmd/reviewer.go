package initcmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ReviewerStamp is the user copy's own version file, inside the skill dir.
const ReviewerStamp = "VERSION"

// ReviewerSettings locate the user-level gabo-reviewer copy (TRD 9.4, D20).
type ReviewerSettings struct {
	// Dir is ${XDG_DATA_HOME:-~/.local/share}/gbai/skills/gabo-reviewer.
	Dir string
	// ProjectDir is refused as a parent of Dir: the scripts never enter the
	// repository they review.
	ProjectDir string
	Version    string
}

// Reviewer installs the embedded gabo-reviewer skill at user level.
type Reviewer struct {
	store Store
	files []File
	s     ReviewerSettings
}

// NewReviewer wires the installer; files are relative to the skill dir
// (SKILL.md, scripts/reviewer.sh, ...).
func NewReviewer(store Store, files []File, s ReviewerSettings) *Reviewer {
	return &Reviewer{store: store, files: files, s: s}
}

// Install writes the user copy when it is missing or its VERSION differs from
// the binary version, and leaves a current copy untouched.
func (r *Reviewer) Install(context.Context) (Output, error) {
	bin, ok := parseVersion(strings.TrimPrefix(r.s.Version, "v"))
	if !ok {
		return Output{}, fmt.Errorf("binary version %q is not a release version x.y.z; build gbai with -ldflags \"-X main.buildVersion=<x.y.z>\"", r.s.Version)
	}
	dir := filepath.Clean(r.s.Dir)
	if err := r.checkDir(dir); err != nil {
		return Output{}, err
	}
	mode, err := r.store.Lstat(dir)
	exists := err == nil
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return Output{}, fmt.Errorf("stat %s: %w", dir, err)
	case mode&fs.ModeSymlink != 0:
		return Output{}, fmt.Errorf("%s is a symlink; gbai init installs the reviewer scripts only into a real directory, so remove the link and rerun gbai init", dir)
	case !mode.IsDir():
		return Output{}, fmt.Errorf("%s is not a directory; move it away and rerun gbai init", dir)
	}
	old := "unknown"
	if exists {
		b, err := r.store.ReadFile(filepath.Join(dir, ReviewerStamp))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return Output{}, fmt.Errorf("read %s: %w", filepath.Join(dir, ReviewerStamp), err)
		}
		if cur, ok := parseVersion(strings.TrimSpace(string(b))); ok {
			if cur.compare(bin) == 0 {
				return Output{Lines: []string{"reviewer skill is current (" + bin.String() + ") in " + dir}}, nil
			}
			old = cur.String()
		}
	}
	if err := r.write(dir, bin); err != nil {
		return Output{}, err
	}
	if !exists {
		return Output{Lines: []string{"reviewer skill installed to " + dir + " (" + bin.String() + ")"}}, nil
	}
	return Output{Lines: []string{"reviewer skill replaced " + old + " → " + bin.String() + " in " + dir}}, nil
}

// checkDir refuses a relative dir and one inside the project.
func (r *Reviewer) checkDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("reviewer skill dir %q is not an absolute path; set XDG_DATA_HOME to an absolute path or unset it", dir)
	}
	rel, err := filepath.Rel(filepath.Clean(r.s.ProjectDir), dir)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("reviewer skill dir %s is inside the project; the reviewer scripts never enter the repository (D20), so point XDG_DATA_HOME outside it", dir)
	}
	return nil
}

// write replaces the dir and writes VERSION last, so an interrupted install
// is repeated on the next run.
func (r *Reviewer) write(dir string, v version) error {
	if err := r.store.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	for _, f := range r.files {
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := r.store.WriteFile(target, f.Data); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	stamp := filepath.Join(dir, ReviewerStamp)
	if err := r.store.WriteFile(stamp, []byte(v.String()+"\n")); err != nil {
		return fmt.Errorf("write %s: %w", stamp, err)
	}
	return nil
}

// Init is one `init` run: the user-level reviewer install, then the catalog.
type Init struct {
	Reviewer *Reviewer
	Catalog  *UseCase
}

// Run installs the reviewer on every run, whatever the repository stamp says,
// then copies or refreshes the catalog; output keeps that order.
func (i *Init) Run(ctx context.Context) (Output, error) {
	rev, err := i.Reviewer.Install(ctx)
	if err != nil {
		return rev, err
	}
	cat, err := i.Catalog.Run(ctx)
	return Output{Lines: append(rev.Lines, cat.Lines...)}, err
}
