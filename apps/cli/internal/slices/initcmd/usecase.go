// Package initcmd copies the embedded gabo skill catalog into the project
// (.gabo/skills/) and keeps its version stamp (.gabo/VERSION), TRD 9.4.
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

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// Project-relative locations (TRD 9.4).
const (
	CatalogDir  = ".gabo/skills"
	StampFile   = ".gabo/VERSION"
	ConfigFile  = "gabo.yaml"
	installPath = "github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo"
)

// File is one catalog file; Path is relative to the catalog directory.
type File struct {
	Path string
	Data []byte
}

// Settings locate the project and carry the binary version and the setup
// prompt printed on a fresh project (D25).
type Settings struct {
	ProjectDir  string
	Version     string
	SetupPrompt string
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
// catalog to an older version. Every successful path then runs the engine
// step (intent 0006 S10).
func (u *UseCase) Run(context.Context) (Output, error) {
	bin, ok := parseVersion(strings.TrimPrefix(u.s.Version, "v"))
	if !ok {
		return Output{}, fmt.Errorf("binary version %q is not a release version x.y.z; build gabo with -ldflags \"-X main.buildVersion=<x.y.z>\"", u.s.Version)
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
		lines := []string{
			"catalog copied to " + CatalogDir + " (" + bin.String() + ")",
			noConfigLine,
			"next: open your agent in this project and paste this prompt",
		}
		if u.s.SetupPrompt != "" {
			// Printed verbatim and written nowhere; the command adds the final newline.
			lines = append(lines, strings.TrimSuffix(u.s.SetupPrompt, "\n"))
		}
		return Output{Lines: lines}, nil
	case hasStamp && cur.compare(bin) == 0:
		return u.withEngines([]string{"catalog is current (" + bin.String() + "); nothing changed"}, hasConfig, bin)
	case hasStamp && cur.compare(bin) > 0:
		return Output{}, fmt.Errorf("catalog version %s is newer than gabo %s; upgrade with: go install %s@v%s",
			cur, bin, installPath, cur)
	}
	old := "none"
	if hasStamp {
		old = cur.String()
	}
	if err := u.copyCatalog(bin); err != nil {
		return Output{}, err
	}
	return u.withEngines([]string{
		"catalog updated " + old + " → " + bin.String(),
		"next: ask your agent to rebuild the agents, then run gabo check",
	}, hasConfig, bin)
}

// withEngines ends a successful catalog step with the engine step (intent
// 0006 S10): with a gabo.yaml the engine set from it is integrated and its
// report closes the output; without one (only the fresh project prints the
// no-gabo.yaml line) nothing is added. gabo.yaml is only read. A ✗ line
// returns ErrEngineFailures after the report.
func (u *UseCase) withEngines(lines []string, hasConfig bool, bin version) (Output, error) {
	if !hasConfig {
		return Output{Lines: lines}, nil
	}
	b, err := u.store.ReadFile(u.path(ConfigFile))
	if err != nil {
		return Output{Lines: lines}, fmt.Errorf("read %s: %w", ConfigFile, err)
	}
	set, err := engineSet(b)
	if err != nil {
		line := engineconfig.FailurePrefix + " " + ConfigFile + ": " + err.Error() + "; fix it and run gabo init again"
		return Output{Lines: append(lines, line)}, ErrEngineFailures
	}
	report := u.engineStep(set, bin.String())
	lines = append(lines, strings.Split(strings.TrimSuffix(report.Render(), "\n"), "\n")...)
	if report.Failed() {
		return Output{Lines: lines}, ErrEngineFailures
	}
	return Output{Lines: lines}, nil
}

// newerStamp reports the stamp and binary versions when a readable stamp is
// newer than a release binary; any other state is left to Run.
func (u *UseCase) newerStamp() (want, have string, ok bool) {
	bin, ok := parseVersion(strings.TrimPrefix(u.s.Version, "v"))
	if !ok {
		return "", "", false
	}
	cur, hasStamp, err := u.readStamp()
	if err != nil || !hasStamp || cur.compare(bin) <= 0 {
		return "", "", false
	}
	return cur.String(), bin.String(), true
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

// checkOwnPaths refuses a .gabo, .gabo/skills or stamp that is a symlink or of
// the wrong kind, so removing and writing never leave the project.
func (u *UseCase) checkOwnPaths() error {
	for _, p := range []struct {
		rel, kind string
		dir       bool
	}{{".gabo", "directory", true}, {CatalogDir, "directory", true}, {StampFile, "regular file", false}} {
		mode, err := u.store.Lstat(u.path(p.rel))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return fmt.Errorf("stat %s: %w", p.rel, err)
		case mode&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink; init writes only inside the project, so replace it with a real directory or file and rerun gabo init", p.rel)
		case mode.IsDir() != p.dir || (!p.dir && !mode.IsRegular()):
			return fmt.Errorf("%s is not a %s; move it away and rerun gabo init", p.rel, p.kind)
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
		return version{}, false, fmt.Errorf("%s holds %q, not a version x.y.z; restore it from git or delete it, then rerun gabo init", StampFile, raw)
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
