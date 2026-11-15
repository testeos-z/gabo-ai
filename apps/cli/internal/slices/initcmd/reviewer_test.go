package initcmd

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

const (
	userDir      = "/home/u/.local/share/gabo/skills/gabo-reviewer"
	userStamp    = userDir + "/VERSION"
	userSkill    = userDir + "/SKILL.md"
	userScript   = userDir + "/scripts/reviewer.sh"
	userLeftover = userDir + "/scripts/old.sh"
	userHash     = userDir + "/REVIEWER_HASH"
)

var reviewerFiles = []File{{Path: "SKILL.md", Data: []byte("skill")}, {Path: "scripts/reviewer.sh", Data: []byte("script")}}

// reviewerHash is the content stamp a user copy of reviewerFiles holds.
func reviewerHash() string { return digest(reviewerFiles) + "\n" }

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
	if got := s.Files[userHash]; got != reviewerHash() {
		t.Fatalf("user copy REVIEWER_HASH = %q, want %q", got, reviewerHash())
	}
	// VERSION is written last, so an interrupted install is retried.
	stampAt := slices.Index(s.Writes, userStamp)
	if hashAt := slices.Index(s.Writes, userHash); hashAt < 0 || hashAt > stampAt {
		t.Fatalf("REVIEWER_HASH not written before VERSION: %v", s.Writes)
	}
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
	files := map[string]string{config: cfgYAML, stamp: "1.2.3\n", hashFile: catalogHash(), setup: "edited", claudeFile: claudeIntegrated()}
	block := strings.Join(claudeBlock(files), "\n")
	s := newStore(files)
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
	want := "reviewer skill installed to " + userDir + " (1.2.3)\ncatalog is current (1.2.3); nothing changed\n" + block
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

// A current copy whose content stamp matches is left alone, even when a file
// in it was edited by hand: only the stamped hash decides a same-version copy.
func TestCurrentUserCopyIsUntouched(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stamp: "1.2.3\n", hashFile: catalogHash(), userStamp: "1.2.3\n", userHash: reviewerHash(), userSkill: "edited by hand"})
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

// A same-version copy whose content differs from the embedded reviewer is
// replaced before the catalog lines, with the refreshed line and no next line.
func TestSameVersionChangedReviewerIsRefreshed(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{
		stamp: "1.2.3\n", hashFile: catalogHash(),
		userStamp: "1.2.3\n", userHash: "stale-hash\n", userScript: "old script",
	})
	out, err := runInit(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertReviewer(t, s, "1.2.3")
	want := "reviewer skill refreshed 1.2.3 in " + userDir + ": content differs from the embedded reviewer\ncatalog is current (1.2.3); nothing changed"
	if joined(out) != want {
		t.Fatalf("output:\n%s", joined(out))
	}
	for _, w := range s.Writes {
		if !strings.HasPrefix(w, userDir+"/") {
			t.Fatalf("write outside the user copy: %s", w)
		}
	}
	if _, ok := s.Files[config]; ok {
		t.Fatal("gabo.yaml written")
	}
}

// A copy installed before REVIEWER_HASH existed is refreshed once; the next
// run reports it current.
func TestSameVersionWithoutReviewerHashRefreshesOnce(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stamp: "1.2.3\n", hashFile: catalogHash(), userStamp: "1.2.3\n", userScript: "old script"})
	out, err := runInit(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertReviewer(t, s, "1.2.3")
	if !strings.HasPrefix(joined(out), "reviewer skill refreshed 1.2.3 in "+userDir+": content differs from the embedded reviewer\n") {
		t.Fatalf("output:\n%s", joined(out))
	}
	writes := len(s.Writes)
	out, err = runInit(t, s, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Writes) != writes || !strings.HasPrefix(joined(out), "reviewer skill is current (1.2.3) in "+userDir+"\n") {
		t.Fatalf("second run output:\n%s\nwrites %v", joined(out), s.Writes[writes:])
	}
}

// The reviewer and the catalog stamp their content with the same digest.
func TestReviewerAndCatalogShareOneDigest(t *testing.T) {
	t.Parallel()
	if got, want := NewUseCase(nil, reviewerFiles, Settings{}).contentHash(), digest(reviewerFiles); got != want {
		t.Fatalf("catalog digest %s, want the shared digest %s", got, want)
	}
	s := newStore(nil)
	if _, err := NewReviewer(s, reviewerFiles, ReviewerSettings{Dir: userDir, ProjectDir: root, Version: "1.2.3"}).Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(s.Files[userHash]); got != digest(reviewerFiles) {
		t.Fatalf("REVIEWER_HASH = %s, want %s", got, digest(reviewerFiles))
	}
}

// The embedded reviewer folder holds no file named like a stamp: the stamps
// live in the user copy and must never be mistaken for embedded content.
func TestEmbeddedReviewerHoldsNoStampFile(t *testing.T) {
	t.Parallel()
	files, err := skills.ReviewerFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if name := f.Path[strings.LastIndex(f.Path, "/")+1:]; name == ReviewerStamp || name == ReviewerHashFile {
			t.Fatalf("embedded reviewer holds stamp file %s", f.Path)
		}
	}
}

// The user copy is refreshed whatever the repository stamp says, even when
// the catalog refuses a newer stamp.
func TestNewerStampStillInstallsReviewer(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stamp: "2.0.0\n"})
	out, err := runInit(t, s, "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "is newer than gabo 1.2.3") {
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
		{"relative dir", "share/gabo/skills/gabo-reviewer", "1.2.3", nil, "not an absolute path"},
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
