// Package initcmd copies the embedded gbai skill catalog into the project
// (.gbai/skills/) and keeps its version stamp (.gbai/VERSION), TRD 9.4.
package initcmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Project-relative locations (TRD 9.4).
const (
	CatalogDir  = ".gbai/skills"
	StampFile   = ".gbai/VERSION"
	ConfigFile  = "gbai.yaml"
	installPath = "github.com/testeos-z/gbai-team/apps/cli/cmd/gbai"
)

// File is one catalog file; Path is relative to the catalog directory.
type File struct {
	Path string
	Data []byte
}

// Settings locate the project and carry the binary version.
type Settings struct {
	ProjectDir string
	Version    string
}

// Output is the human-readable report.
type Output struct {
	Lines []string
}

// UseCase copies or refreshes the catalog.
type UseCase struct {
	store Store
	files []File
	s     Settings
}

// NewUseCase wires the use case. files must already exclude gabo-reviewer,
// which never enters the repository (D20).
func NewUseCase(store Store, files []File, s Settings) *UseCase {
	return &UseCase{store: store, files: files, s: s}
}

// Run copies the catalog on a fresh project and refreshes it when the stamp
// is older or missing; it never touches the config and never moves the
// catalog to an older version.
func (u *UseCase) Run(context.Context) (Output, error) {
	bin, ok := parseVersion(strings.TrimPrefix(u.s.Version, "v"))
	if !ok {
		return Output{}, fmt.Errorf("binary version %q is not a release version x.y.z; build gbai with -ldflags \"-X main.buildVersion=<x.y.z>\"", u.s.Version)
	}
	cur, hasStamp, err := u.readStamp()
	if err != nil {
		return Output{}, err
	}
	hasConfig, err := u.exists(ConfigFile)
	if err != nil {
		return Output{}, err
	}
	switch {
	case !hasStamp && !hasConfig:
		if err := u.copyCatalog(bin); err != nil {
			return Output{}, err
		}
		return Output{Lines: []string{
			"catalog copied to " + CatalogDir + " (" + bin.String() + ")",
			"next: open your agent in this project and paste this prompt",
		}}, nil
	case hasStamp && cur.compare(bin) == 0:
		return Output{Lines: []string{"catalog is current (" + bin.String() + "); nothing changed"}}, nil
	case hasStamp && cur.compare(bin) > 0:
		return Output{}, fmt.Errorf("catalog version %s is newer than gbai %s; upgrade with: go install %s@v%s",
			cur, bin, installPath, cur)
	}
	old := "none"
	if hasStamp {
		old = cur.String()
	}
	if err := u.copyCatalog(bin); err != nil {
		return Output{}, err
	}
	return Output{Lines: []string{
		"catalog updated " + old + " → " + bin.String(),
		"next: ask your agent to rebuild the agents, then run gabo check",
	}}, nil
}

// copyCatalog replaces the catalog directory and writes the stamp last, so an
// interrupted copy keeps the old stamp and the next run repeats it.
func (u *UseCase) copyCatalog(v version) error {
	if err := u.checkOwnPaths(); err != nil {
		return err
	}
	dir := u.path(CatalogDir)
	if err := u.store.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	for _, f := range u.files {
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := u.store.WriteFile(target, f.Data); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	stamp := u.path(StampFile)
	if err := u.store.WriteFile(stamp, []byte(v.String()+"\n")); err != nil {
		return fmt.Errorf("write %s: %w", stamp, err)
	}
	return nil
}

// checkOwnPaths refuses a .gbai, .gbai/skills or stamp that is a symlink or of
// the wrong kind, so removing and writing never leave the project.
func (u *UseCase) checkOwnPaths() error {
	for _, p := range []struct {
		rel, kind string
		dir       bool
	}{{".gbai", "directory", true}, {CatalogDir, "directory", true}, {StampFile, "regular file", false}} {
		mode, err := u.store.Lstat(u.path(p.rel))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return fmt.Errorf("stat %s: %w", p.rel, err)
		case mode&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink; init writes only inside the project, so replace it with a real directory or file and rerun gbai init", p.rel)
		case mode.IsDir() != p.dir || (!p.dir && !mode.IsRegular()):
			return fmt.Errorf("%s is not a %s; move it away and rerun gbai init", p.rel, p.kind)
		}
	}
	return nil
}

func (u *UseCase) readStamp() (version, bool, error) {
	b, err := u.store.ReadFile(u.path(StampFile))
	if errors.Is(err, fs.ErrNotExist) {
		return version{}, false, nil
	}
	if err != nil {
		return version{}, false, fmt.Errorf("read %s: %w", StampFile, err)
	}
	raw := strings.TrimSpace(string(b))
	v, ok := parseVersion(raw)
	if !ok {
		return version{}, false, fmt.Errorf("%s holds %q, not a version x.y.z; restore it from git or delete it, then rerun gbai init", StampFile, raw)
	}
	return v, true, nil
}

func (u *UseCase) exists(rel string) (bool, error) {
	_, err := u.store.ReadFile(u.path(rel))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	}
	return false, fmt.Errorf("read %s: %w", rel, err)
}

func (u *UseCase) path(rel string) string {
	return filepath.Join(u.s.ProjectDir, filepath.FromSlash(rel))
}

// version is a bare semantic version without pre-release or build parts.
type version [3]int

var versionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func parseVersion(s string) (version, bool) {
	m := versionRE.FindStringSubmatch(s)
	if m == nil {
		return version{}, false
	}
	var v version
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return version{}, false
		}
		v[i] = n
	}
	return v, true
}

func (v version) compare(o version) int {
	for i := range v {
		if v[i] != o[i] {
			if v[i] < o[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (v version) String() string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }
