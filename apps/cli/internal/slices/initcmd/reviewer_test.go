package initcmd

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"
)

const (
	userDir      = "/home/u/.local/share/gbai/skills/gabo-reviewer"
	userStamp    = userDir + "/VERSION"
	userSkill    = userDir + "/SKILL.md"
	userScript   = userDir + "/scripts/reviewer.sh"
	userLeftover = userDir + "/scripts/old.sh"
)

var reviewerFiles = []File{{Path: "SKILL.md", Data: []byte("skill")}, {Path: "scripts/reviewer.sh", Data: []byte("script")}}

func newInit(s *memStore, version string) *Init {
	return &Init{
		Reviewer: NewReviewer(s, reviewerFiles, ReviewerSettings{Dir: userDir, ProjectDir: root, Version: version}),
		Catalog:  NewUseCase(s, catalog, Settings{ProjectDir: root, Version: version}),
	}
}

func runInit(t *testing.T, s *memStore, version string) (Output, error) {
	t.Helper()
	return newInit(s, version).Run(context.Background())
}

// assertReviewer checks the user copy holds exactly the embedded skill and
// its own VERSION file.
func assertReviewer(t *testing.T, s *memStore, version string) {
	t.Helper()
	if s.Files[userSkill] != "skill" || s.Files[userScript] != "script" {
		t.Fatalf("user copy = %q, %q", s.Files[userSkill], s.Files[userScript])
	}
	if _, ok := s.Files[userLeftover]; ok {
		t.Fatal("stale user copy file kept")
	}
	if got := s.Files[userStamp]; got != version+"\n" {
		t.Fatalf("user copy VERSION = %q, want %q", got, version+"\n")
	}
	// VERSION is written last, so an interrupted install is retried.
	stampAt := slices.Index(s.Writes, userStamp)
	for i, w := range s.Writes {
		if strings.HasPrefix(w, userDir+"/") && i > stampAt {
			t.Fatalf("%s written after VERSION: %v", w, s.Writes)
		}
	}
}

// Writes to the user dir must never land inside the project (D20).
func assertNoReviewerInProject(t *testing.T, s *memStore) {
	t.Helper()
	for f := range s.Files {
		if strings.HasPrefix(f, root+"/") && strings.Contains(f, "gabo-reviewer") {
			t.Fatalf("reviewer file inside the project: %s", f)
		}
	}
}

func TestFreshProjectInstallsReviewerAtUserLevel(t *testing.T) {
	t.Parallel()
	s := newStore(nil)
	out, err := runInit(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertReviewer(t, s, "1.2.3")
	assertCatalog(t, s, "1.2.3")
	assertNoReviewerInProject(t, s)
	if out.Lines[0] != "reviewer skill installed to "+userDir+" (1.2.3)" {
		t.Fatalf("output:\n%s", joined(out))
	}
	if last := out.Lines[len(out.Lines)-1]; last != "next: open your agent in this project and paste this prompt" {
		t.Fatalf("last line = %q", last)
	}
}

// A teammate's clone: the repository stamp matches, but this machine has no
// user copy, so it is installed while the catalog stays untouched.
func TestMatchingStampWithoutUserCopyInstallsReviewer(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{config: cfgYAML, stamp: "1.2.3\n", setup: "edited"})
	out, err := runInit(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertReviewer(t, s, "1.2.3")
	if s.Files[setup] != "edited" || s.Files[config] != cfgYAML || s.Files[stamp] != "1.2.3\n" {
		t.Fatal("matching stamp changed the project")
	}
	for _, w := range s.Writes {
		if !strings.HasPrefix(w, userDir+"/") {
			t.Fatalf("write outside the user copy: %s", w)
		}
	}
	want := "reviewer skill installed to " + userDir + " (1.2.3)\ncatalog is current (1.2.3); nothing changed"
	if joined(out) != want {
		t.Fatalf("output:\n%s", joined(out))
	}
}

// A user copy whose VERSION differs from the binary (older, newer, garbage
// or missing) is replaced and reported.
func TestDifferingUserCopyIsReplaced(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		before string
	}{
		{"older", map[string]string{userStamp: "1.2.2\n", userLeftover: "x"}, "1.2.2"},
		{"newer", map[string]string{userStamp: "2.0.0\n", userLeftover: "x"}, "2.0.0"},
		{"garbage", map[string]string{userStamp: "latest\n", userLeftover: "x"}, "unknown"},
		{"no VERSION", map[string]string{userLeftover: "x"}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.files[stamp] = "1.2.3\n"
			s := newStore(tt.files)
			out, err := runInit(t, s, "1.2.3")
			if err != nil {
				t.Fatal(err)
			}
			assertReviewer(t, s, "1.2.3")
			if want := "reviewer skill replaced " + tt.before + " → 1.2.3 in " + userDir; out.Lines[0] != want {
				t.Fatalf("output:\n%s\nwant first line %q", joined(out), want)
			}
		})
	}
}

func TestCurrentUserCopyIsUntouched(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stamp: "1.2.3\n", userStamp: "1.2.3\n", userSkill: "edited by hand"})
	before := maps.Clone(s.Files)
	out, err := runInit(t, s, "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Writes) != 0 || len(s.Removed) != 0 || !maps.Equal(s.Files, before) {
		t.Fatalf("writes %v removed %v", s.Writes, s.Removed)
	}
	want := "reviewer skill is current (1.2.3) in " + userDir + "\ncatalog is current (1.2.3); nothing changed"
	if joined(out) != want {
		t.Fatalf("output:\n%s", joined(out))
	}
}

// The user copy is refreshed whatever the repository stamp says, even when
// the catalog refuses a newer stamp.
func TestNewerStampStillInstallsReviewer(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stamp: "2.0.0\n"})
	out, err := runInit(t, s, "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "is newer than gbai 1.2.3") {
		t.Fatalf("err = %v", err)
	}
	assertReviewer(t, s, "1.2.3")
	if s.Files[stamp] != "2.0.0\n" {
		t.Fatal("newer stamp changed")
	}
	if len(out.Lines) != 1 || !strings.HasPrefix(out.Lines[0], "reviewer skill installed to ") {
		t.Fatalf("output:\n%s", joined(out))
	}
}

func TestReviewerRefusesUnsafeLocations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, dir, version string
		modes              map[string]fs.FileMode
		want               string
	}{
		{"dev binary", userDir, "dev", nil, `binary version "dev"`},
		{"relative dir", "share/gbai/skills/gabo-reviewer", "1.2.3", nil, "not an absolute path"},
		{"inside the project", root + "/.local/gabo-reviewer", "1.2.3", nil, "inside the project"},
		{"project itself", root, "1.2.3", nil, "inside the project"},
		{"symlink", userDir, "1.2.3", map[string]fs.FileMode{userDir: fs.ModeSymlink}, "symlink"},
		{"file", userDir, "1.2.3", map[string]fs.FileMode{userDir: 0}, "not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(nil)
			s.Modes = tt.modes
			in := &Init{
				Reviewer: NewReviewer(s, reviewerFiles, ReviewerSettings{Dir: tt.dir, ProjectDir: root, Version: tt.version}),
				Catalog:  NewUseCase(s, catalog, Settings{ProjectDir: root, Version: tt.version}),
			}
			_, err := in.Run(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
			if len(s.Writes) != 0 || len(s.Removed) != 0 {
				t.Fatalf("writes %v removed %v", s.Writes, s.Removed)
			}
		})
	}
}

func TestReviewerWriteErrorIsReported(t *testing.T) {
	t.Parallel()
	s := newStore(nil)
	s.WriteErr = errors.New("disk full")
	_, err := runInit(t, s, "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err = %v", err)
	}
	if _, ok := s.Files[stamp]; ok {
		t.Fatal("catalog written after the reviewer install failed")
	}
}
