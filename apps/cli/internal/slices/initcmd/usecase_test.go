package initcmd

import (
	"context"
	"io/fs"
	"maps"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/kerneltest"
)

const (
	root    = "/work/p"
	config  = root + "/gabo.yaml"
	stamp   = root + "/.gabo/VERSION"
	setup   = root + "/.gabo/skills/gabo-setup/SKILL.md"
	coder   = root + "/.gabo/skills/gabo-coder/SKILL.md"
	stale   = root + "/.gabo/skills/old-skill/SKILL.md"
	userCfg = "/home/u/.config/gabo/config.yaml"
	cfgYAML = "repo:\n  name: o/r\n"
)

var catalog = []File{{Path: "gabo-coder/SKILL.md", Data: []byte("coder")}, {Path: "gabo-setup/SKILL.md", Data: []byte("setup")}}

// memStore adds RemoveAll and Lstat to the shared in-memory fake; Modes
// overrides the mode Lstat reports for a path (for example a symlink).
type memStore struct {
	kerneltest.FS
	Removed []string
	Modes   map[string]fs.FileMode
}

func (m *memStore) Lstat(name string) (fs.FileMode, error) {
	if mode, ok := m.Modes[name]; ok {
		return mode, nil
	}
	if _, ok := m.Files[name]; ok {
		return 0, nil
	}
	for f := range m.Files {
		if strings.HasPrefix(f, name+"/") {
			return fs.ModeDir, nil
		}
	}
	return 0, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
}

func (m *memStore) RemoveAll(dir string) error {
	m.Removed = append(m.Removed, dir)
	for name := range m.Files {
		if strings.HasPrefix(name, dir+"/") {
			delete(m.Files, name)
		}
	}
	return nil
}

func newStore(files map[string]string) *memStore {
	return &memStore{FS: kerneltest.FS{Files: files}}
}

func run(t *testing.T, s *memStore, version string) (Output, error) {
	t.Helper()
	return NewUseCase(s, catalog, Settings{ProjectDir: root, Version: version}).Run(context.Background())
}

func joined(out Output) string { return strings.Join(out.Lines, "\n") }

// assertCatalog checks the copy holds exactly the embedded catalog and stamp.
func assertCatalog(t *testing.T, s *memStore, version string) {
	t.Helper()
	if s.Files[setup] != "setup" || s.Files[coder] != "coder" {
		t.Fatalf("catalog copy = %q, %q", s.Files[setup], s.Files[coder])
	}
	if _, ok := s.Files[stale]; ok {
		t.Fatal("stale catalog file kept")
	}
	if got := s.Files[stamp]; got != version+"\n" {
		t.Fatalf("stamp = %q, want %q", got, version+"\n")
	}
}

func TestFreshProjectCopiesCatalogAndStamp(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{userCfg: "user"})
	out, err := run(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertCatalog(t, s, "1.2.3")
	if _, ok := s.Files[config]; ok {
		t.Fatal("init created gabo.yaml")
	}
	if s.Files[userCfg] != "user" {
		t.Fatal("init changed the global user file")
	}
	for _, w := range s.Writes {
		if !strings.HasPrefix(w, root+"/.gabo/") {
			t.Fatalf("write outside .gabo/: %s", w)
		}
	}
	if last := out.Lines[len(out.Lines)-1]; last != "next: open your agent in this project and paste this prompt" {
		t.Fatalf("last line = %q\n%s", last, joined(out))
	}
}

func TestSameVersionProducesSameFiles(t *testing.T) {
	t.Parallel()
	a, b := newStore(nil), newStore(nil)
	if _, err := run(t, a, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, b, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(a.Files, b.Files) {
		t.Fatalf("files differ:\n%v\n%v", a.Files, b.Files)
	}
}

// Older or missing stamp refreshes the catalog, with or without a config,
// and never touches the config.
func TestOlderOrMissingStampRefreshesCatalog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		before string
	}{
		{"older stamp with config", map[string]string{config: cfgYAML, stamp: "1.2.2\n", stale: "old"}, "1.2.2"},
		{"older minor", map[string]string{config: cfgYAML, stamp: "0.9.10\n", stale: "old"}, "0.9.10"},
		{"missing stamp with config", map[string]string{config: cfgYAML, stale: "old"}, "none"},
		{"older stamp without config", map[string]string{stamp: "1.0.0\n", stale: "old"}, "1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(tt.files)
			_, hadConfig := tt.files[config]
			out, err := run(t, s, "1.2.3")
			if err != nil {
				t.Fatal(err)
			}
			assertCatalog(t, s, "1.2.3")
			if got, ok := s.Files[config]; ok != hadConfig || (ok && got != cfgYAML) {
				t.Fatalf("config = %q (present %v)", got, ok)
			}
			want := []string{
				"catalog updated " + tt.before + " → 1.2.3",
				"next: ask your agent to rebuild the agents, then run gabo check",
			}
			if joined(out) != strings.Join(want, "\n") {
				t.Fatalf("output:\n%s", joined(out))
			}
		})
	}
}

func TestMatchingStampChangesNothing(t *testing.T) {
	t.Parallel()
	for _, files := range []map[string]string{
		{config: cfgYAML, stamp: "1.2.3\n", setup: "edited"},
		{stamp: "1.2.3\n"},
	} {
		s := newStore(files)
		before := maps.Clone(s.Files)
		out, err := run(t, s, "v1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		if len(s.Writes) != 0 || len(s.Removed) != 0 || !maps.Equal(s.Files, before) {
			t.Fatalf("writes %v removed %v", s.Writes, s.Removed)
		}
		if joined(out) != "catalog is current (1.2.3); nothing changed" {
			t.Fatalf("output:\n%s", joined(out))
		}
	}
}

func TestNewerStampChangesNothingAndFails(t *testing.T) {
	t.Parallel()
	for _, files := range []map[string]string{
		{config: cfgYAML, stamp: "1.10.0\n", setup: "newer"},
		{stamp: "2.0.0\n"},
	} {
		s := newStore(files)
		before := maps.Clone(s.Files)
		_, err := run(t, s, "1.2.3")
		if err == nil {
			t.Fatal("newer stamp accepted")
		}
		if len(s.Writes) != 0 || len(s.Removed) != 0 || !maps.Equal(s.Files, before) {
			t.Fatalf("writes %v removed %v", s.Writes, s.Removed)
		}
		w := strings.TrimSpace(files[stamp])
		for _, want := range []string{
			"catalog version " + w + " is newer than gabo 1.2.3",
			"go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v" + w,
		} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q lacks %q", err, want)
			}
		}
	}
}

func TestUnusableVersionsChangeNothing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, version string
		files         map[string]string
		want          string
	}{
		{"dev binary", "dev", map[string]string{}, `binary version "dev"`},
		{"vcs pseudo-version", "v1.4.1-0.20261027120000-abc123def456+dirty", map[string]string{}, `binary version "v1.4.1-0.20261027120000-abc123def456+dirty"`},
		{"garbage stamp", "1.2.3", map[string]string{stamp: "latest\n"}, `.gabo/VERSION holds "latest"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(tt.files)
			_, err := run(t, s, tt.version)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
			if len(s.Writes) != 0 || len(s.Removed) != 0 {
				t.Fatalf("writes %v removed %v", s.Writes, s.Removed)
			}
		})
	}
}

// gabo no longer ships an MCP server (T042) nor places skills in runtime
// folders (T039): every write stays under .gabo/.
func TestWritesNoRuntimeFoldersNorMCP(t *testing.T) {
	t.Parallel()
	s := newStore(nil)
	out, err := run(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range s.Writes {
		if strings.Contains(w, "/.claude/") || strings.Contains(w, "/.pi/") {
			t.Fatalf("runtime folder write: %s", w)
		}
	}
	if strings.Contains(strings.ToLower(joined(out)), "mcp") {
		t.Fatalf("output mentions MCP:\n%s", joined(out))
	}
}

// A symlinked .gabo, .gabo/skills or stamp would make init delete or write
// outside the project, so init refuses before any change.
func TestRefusesSymlinkedCatalogPaths(t *testing.T) {
	t.Parallel()
	for _, p := range []string{root + "/.gabo", root + "/.gabo/skills", stamp} {
		for _, files := range []map[string]string{{}, {config: cfgYAML}} {
			s := newStore(files)
			s.Modes = map[string]fs.FileMode{p: fs.ModeSymlink}
			_, err := run(t, s, "1.2.3")
			if err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("%s: err = %v", p, err)
			}
			if len(s.Writes) != 0 || len(s.Removed) != 0 {
				t.Fatalf("%s: writes %v removed %v", p, s.Writes, s.Removed)
			}
		}
	}
	s := newStore(nil)
	s.Modes = map[string]fs.FileMode{root + "/.gabo": 0}
	if _, err := run(t, s, "1.2.3"); err == nil || len(s.Writes) != 0 {
		t.Fatalf(".gabo as a file: err %v writes %v", err, s.Writes)
	}
}

// TestSetupPromptPrintedOnlyOnFreshProject: a fresh project gets the setup
// prompt verbatim right after the next step; any other run never prints it,
// and init writes it nowhere (TRD 9.4, D25).
func TestSetupPromptPrintedOnlyOnFreshProject(t *testing.T) {
	t.Parallel()
	const prompt = "Set up gabo-ai.\n\n1. Run gabo init.\n"
	runWith := func(s *memStore) (Output, error) {
		return NewUseCase(s, catalog, Settings{ProjectDir: root, Version: "1.2.3", SetupPrompt: prompt}).Run(context.Background())
	}
	s := newStore(map[string]string{})
	out, err := runWith(s)
	if err != nil {
		t.Fatal(err)
	}
	want := "catalog copied to .gabo/skills (1.2.3)\n" +
		"next: open your agent in this project and paste this prompt\n" + prompt
	if got := joined(out) + "\n"; got != want {
		t.Fatalf("output:\n%q\nwant:\n%q", got, want)
	}
	for name, data := range s.Files {
		if strings.Contains(data, "Set up gabo-ai") {
			t.Fatalf("init wrote the prompt to %s", name)
		}
	}
	for name, files := range map[string]map[string]string{
		"config without stamp":   {config: cfgYAML},
		"config, older stamp":    {config: cfgYAML, stamp: "1.2.2\n"},
		"config, matching stamp": {config: cfgYAML, stamp: "1.2.3\n"},
		"stamp without config":   {stamp: "1.2.3\n"},
	} {
		out, err := runWith(newStore(files))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(joined(out), "Set up gabo-ai") {
			t.Fatalf("%s: prompt printed:\n%s", name, joined(out))
		}
	}
}
