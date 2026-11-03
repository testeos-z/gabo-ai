package check

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

const (
	preRoot     = "/proj"
	prePlan     = "docs/product/intents/0009-x/plan.md"
	preSpec     = "docs/product/intents/0009-x/spec.md"
	acceptedMD  = "## Acceptance\n\n| accepted by | date | notes |\n|---|---|---|\n| owner | 2026-11-03 | ok |\n"
	unfilledMD  = "## Acceptance\n\n| accepted by | date | notes |\n|---|---|---|\n|  |  |  |\n"
	notAccepted = "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan"
	noSpec      = "✗ " + prePlan + ": no spec.md; a plan starts only after a human accepts the spec"
)

// unreadableFS fails every read of one path with a permission error.
type unreadableFS struct {
	fstest.MapFS
	name string
}

func (u unreadableFS) Open(name string) (fs.File, error) {
	if name == u.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.MapFS.Open(name)
}

func (u unreadableFS) ReadFile(name string) ([]byte, error) {
	if name == u.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.MapFS.ReadFile(name)
}

func intentWith(specMD string) fstest.MapFS {
	m := fstest.MapFS{"docs/product/intents/0009-x/intent.md": file("# intent\n")}
	if specMD != "" {
		m[preSpec] = file(specMD)
	}
	return m
}

func preWrite(t *testing.T, proj fs.FS, paths ...string) Report {
	t.Helper()
	rep, err := NewUseCase(proj, nil, nil, Settings{Root: preRoot}).PreWrite(paths)
	if err != nil {
		t.Fatalf("PreWrite(%q) error = %v", paths, err)
	}
	return rep
}

func TestPreWriteStageGate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		proj  fs.FS
		paths []string
		want  []string
	}{
		{name: "unaccepted spec blocks the plan", proj: intentWith(unfilledMD), paths: []string{prePlan}, want: []string{notAccepted}},
		{name: "spec without an Acceptance table blocks", proj: intentWith("# spec\n"), paths: []string{prePlan}, want: []string{notAccepted}},
		{name: "filled Acceptance row passes", proj: intentWith(acceptedMD), paths: []string{prePlan}},
		{name: "missing spec.md blocks", proj: intentWith(""), paths: []string{prePlan}, want: []string{noSpec}},
		{name: "missing intent folder blocks", proj: fstest.MapFS{}, paths: []string{prePlan}, want: []string{noSpec}},
		{
			name:  "unreadable spec fails closed",
			proj:  unreadableFS{MapFS: intentWith(acceptedMD), name: preSpec},
			paths: []string{prePlan},
			want:  []string{"✗ " + prePlan + ": spec.md cannot be read (read " + preSpec + ": open " + preSpec + ": permission denied); it counts as not accepted"},
		},
		{name: "absolute path inside the root equals the relative one", proj: intentWith(unfilledMD), paths: []string{preRoot + "/" + prePlan}, want: []string{notAccepted}},
		{name: "unclean relative path is cleaned", proj: intentWith(unfilledMD), paths: []string{"./docs/product/../product/intents/0009-x/plan.md"}, want: []string{notAccepted}},
		{name: "case-insensitive file name is gated", proj: intentWith(unfilledMD), paths: []string{"docs/product/intents/0009-x/Plan.md"}, want: []string{"✗ docs/product/intents/0009-x/Plan.md: spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan"}},
		{name: "case-insensitive intents dir is gated", proj: intentWith(""), paths: []string{"Docs/Product/Intents/0009-x/plan.md"}, want: []string{"✗ Docs/Product/Intents/0009-x/plan.md: no spec.md; a plan starts only after a human accepts the spec"}},
		{name: "spec.md itself is not gated", proj: intentWith(""), paths: []string{preSpec}},
		{name: "other files are not gated", proj: intentWith(""), paths: []string{"README.md", "docs/product/intents/0009-x/intent.md", "docs/product/intents/0009-x/sub/plan.md", "docs/product/intents/plan.md", "plan.md"}},
		{name: "relative path outside the root passes", proj: intentWith(""), paths: []string{"../other/" + prePlan}},
		{name: "absolute path outside the root passes", proj: intentWith(""), paths: []string{"/elsewhere/" + prePlan, "/proj2/" + prePlan}},
		{
			name:  "one line per blocked path, in argument order",
			proj:  intentWith(unfilledMD),
			paths: []string{"README.md", prePlan, "docs/product/intents/0010-y/plan.md"},
			want:  []string{notAccepted, "✗ docs/product/intents/0010-y/plan.md: no spec.md; a plan starts only after a human accepts the spec"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rep := preWrite(t, tt.proj, tt.paths...)
			if !slices.Equal(rep.Lines, tt.want) {
				t.Fatalf("lines = %q, want %q", rep.Lines, tt.want)
			}
			if rep.Failed != (len(tt.want) > 0) {
				t.Fatalf("failed = %v, want %v", rep.Failed, len(tt.want) > 0)
			}
		})
	}
}

// The same paths judged with the plan already on disk give the same
// result: the gate looks only at spec.md.
func TestPreWriteIgnoresAnExistingPlan(t *testing.T) {
	t.Parallel()
	proj := intentWith(unfilledMD)
	proj[prePlan] = file("# plan\n")
	if rep := preWrite(t, proj, prePlan); !slices.Equal(rep.Lines, []string{notAccepted}) {
		t.Fatalf("lines = %q", rep.Lines)
	}
}

func TestPreWriteUsageErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		s     Settings
		paths []string
	}{
		{name: "no path", s: Settings{Root: preRoot}},
		{name: "empty path", s: Settings{Root: preRoot}, paths: []string{""}},
		{name: "absolute path without a root", paths: []string{"/proj/" + prePlan}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewUseCase(intentWith(""), nil, nil, tt.s).PreWrite(tt.paths)
			if err == nil {
				t.Fatal("err = nil, want a usage error")
			}
			if !errors.Is(err, ErrPreWriteUsage) {
				t.Fatalf("err = %v, want ErrPreWriteUsage", err)
			}
		})
	}
}

// fakeResolve resolves symlinks like filepath.EvalSymlinks on a fake tree:
// links maps a link path to its target, exists lists the real directories
// and files.
func fakeResolve(links map[string]string, exists ...string) func(string) (string, error) {
	return func(p string) (string, error) {
		for link, target := range links {
			if p == link || strings.HasPrefix(p, link+"/") {
				p = target + strings.TrimPrefix(p, link)
			}
		}
		if !slices.Contains(exists, p) {
			return "", &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
		}
		return p, nil
	}
}

// The hook sends absolute paths that may keep a symlink the project root
// (git's resolved top level) does not, or the reverse: both sides are
// resolved, so the gate cannot be skipped through a symlink.
func TestPreWriteResolvesSymlinks(t *testing.T) {
	t.Parallel()
	exists := []string{"/real/proj", "/real/proj/docs", "/real/proj/docs/product", "/real/proj/docs/product/intents", "/real/proj/docs/product/intents/0009-x", "/real/proj/notes.md"}
	tests := []struct {
		name, root, path string
		links            map[string]string
		want             []string
	}{
		{name: "path through a symlinked parent", root: "/real/proj", path: "/link/proj/" + prePlan, links: map[string]string{"/link": "/real"}, want: []string{notAccepted}},
		{name: "symlinked root, resolved path", root: "/link/proj", path: "/real/proj/" + prePlan, links: map[string]string{"/link": "/real"}, want: []string{notAccepted}},
		{name: "new intent folder through a symlink", root: "/real/proj", path: "/link/proj/docs/product/intents/0010-y/plan.md", links: map[string]string{"/link": "/real"}, want: []string{"✗ docs/product/intents/0010-y/plan.md: no spec.md; a plan starts only after a human accepts the spec"}},
		{name: "relative link to the plan", root: "/real/proj", path: "notes.md", links: map[string]string{"/real/proj/notes.md": "/real/proj/" + prePlan}, want: []string{notAccepted}},
		{name: "resolved outside the root passes", root: "/real/proj", path: "/link/other/" + prePlan, links: map[string]string{"/link": "/real"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := exists
			if tt.name == "relative link to the plan" {
				ex = append(slices.Clone(exists), "/real/proj/"+prePlan)
			}
			s := Settings{Root: tt.root, Resolve: fakeResolve(tt.links, ex...)}
			rep, err := NewUseCase(intentWith(unfilledMD), nil, nil, s).PreWrite([]string{tt.path})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(rep.Lines, tt.want) {
				t.Fatalf("lines = %q, want %q", rep.Lines, tt.want)
			}
		})
	}
}

// A plan.md that is a symlink to another intent's plan is judged as written
// and at its target: the write is blocked when either spec is not accepted.
func TestPreWriteJudgesALinkedPlanAtBothEnds(t *testing.T) {
	t.Parallel()
	proj := intentWith(acceptedMD)
	other := "docs/product/intents/0010-y/plan.md"
	exists := []string{"/proj", "/proj/docs", "/proj/docs/product", "/proj/docs/product/intents", "/proj/docs/product/intents/0009-x", "/proj/docs/product/intents/0010-y", "/proj/" + other}
	s := Settings{Root: preRoot, Resolve: fakeResolve(map[string]string{"/proj/" + prePlan: "/proj/" + other}, exists...)}
	rep, err := NewUseCase(proj, nil, nil, s).PreWrite([]string{prePlan})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"✗ " + other + ": no spec.md; a plan starts only after a human accepts the spec"}
	if !slices.Equal(rep.Lines, want) || !rep.Failed {
		t.Fatalf("lines = %q, want %q", rep.Lines, want)
	}
}

// A path whose symlinks cannot be resolved for another reason than a
// missing entry is blocked: the gate fails closed.
func TestPreWriteUnresolvablePathFailsClosed(t *testing.T) {
	t.Parallel()
	denied := func(p string) (string, error) {
		return "", &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrPermission}
	}
	for _, root := range []string{preRoot, "/broken"} {
		resolve := denied
		if root == preRoot {
			resolve = func(p string) (string, error) {
				if p == preRoot {
					return p, nil
				}
				return denied(p)
			}
		}
		s := Settings{Root: root, Resolve: resolve}
		rep, err := NewUseCase(intentWith(acceptedMD), nil, nil, s).PreWrite([]string{"notes.md"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Lines) != 1 || !rep.Failed || !strings.HasPrefix(rep.Lines[0], "✗ notes.md: its symlinks cannot be resolved (") || !strings.HasSuffix(rep.Lines[0], "); it counts as not accepted") {
			t.Fatalf("root %s: lines = %q", root, rep.Lines)
		}
	}
}

// On a case-insensitive filesystem an absolute path may spell the root in
// another case; it is still inside the root.
func TestPreWriteRootComparedWithoutCase(t *testing.T) {
	t.Parallel()
	rep := preWrite(t, intentWith(unfilledMD), "/PROJ/"+prePlan)
	if !slices.Equal(rep.Lines, []string{notAccepted}) {
		t.Fatalf("lines = %q, want %q", rep.Lines, []string{notAccepted})
	}
}
