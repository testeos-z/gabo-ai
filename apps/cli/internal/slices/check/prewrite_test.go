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
	rep, err := NewUseCase(intentWith(unfilledMD), nil, nil, Settings{Root: preRoot, FoldCase: true}).PreWrite([]string{"/PROJ/" + prePlan})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rep.Lines, []string{notAccepted}) {
		t.Fatalf("lines = %q, want %q", rep.Lines, []string{notAccepted})
	}
}

// fakeReadlink reports the links of a fake tree like os.Lstat plus
// os.Readlink: a link gives its target as stored, an existing entry that is
// no link gives false, anything else a missing entry.
func fakeReadlink(links map[string]string, exists ...string) func(string) (string, bool, error) {
	return func(p string) (string, bool, error) {
		if target, ok := links[p]; ok {
			return target, true, nil
		}
		if slices.Contains(exists, p) {
			return "", false, nil
		}
		return "", false, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
	}
}

// A dangling symlink (its target does not exist yet) is judged at its
// target too: a write through it creates that target.
func TestPreWriteJudgesADanglingLinkAtItsTarget(t *testing.T) {
	t.Parallel()
	other := "docs/product/intents/0010-y/plan.md"
	blocked := []string{"✗ " + other + ": no spec.md; a plan starts only after a human accepts the spec"}
	exists := []string{"/proj", "/proj/docs", "/proj/docs/product", "/proj/docs/product/intents", "/proj/docs/product/intents/0009-x", "/proj/" + preSpec}
	tests := []struct {
		name, path string
		links      map[string]string
		want       []string
	}{
		{name: "absolute target", path: "notes.md", links: map[string]string{"/proj/notes.md": "/proj/" + other}, want: blocked},
		{name: "relative target", path: "notes.md", links: map[string]string{"/proj/notes.md": other}, want: blocked},
		{name: "accepted plan linked to an unaccepted one", path: prePlan, links: map[string]string{"/proj/" + prePlan: "../0010-y/plan.md"}, want: blocked},
		{name: "dangling directory link", path: "drafts/plan.md", links: map[string]string{"/proj/drafts": "docs/product/intents/0010-y"}, want: blocked},
		{name: "chain of dangling links", path: "notes.md", links: map[string]string{"/proj/notes.md": "next.md", "/proj/next.md": other}, want: blocked},
		{name: "dangling link outside the gate passes", path: "notes.md", links: map[string]string{"/proj/notes.md": "drafts/todo.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Settings{Root: preRoot, Resolve: fakeResolve(tt.links, exists...), Readlink: fakeReadlink(tt.links, exists...)}
			rep, err := NewUseCase(intentWith(acceptedMD), nil, nil, s).PreWrite([]string{tt.path})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(rep.Lines, tt.want) || rep.Failed != (tt.want != nil) {
				t.Fatalf("lines = %q, want %q", rep.Lines, tt.want)
			}
		})
	}
}

// A dangling link whose target cannot be read, or a loop of links, fails
// closed.
func TestPreWriteUnreadableDanglingLinkFailsClosed(t *testing.T) {
	t.Parallel()
	exists := []string{"/proj"}
	denied := func(p string) (string, bool, error) {
		if p == "/proj/notes.md" {
			return "", false, &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrPermission}
		}
		return fakeReadlink(nil, exists...)(p)
	}
	loop := map[string]string{"/proj/notes.md": "next.md", "/proj/next.md": "notes.md"}
	for name, readlink := range map[string]func(string) (string, bool, error){"unreadable": denied, "loop": fakeReadlink(loop, exists...)} {
		s := Settings{Root: preRoot, Resolve: fakeResolve(nil, exists...), Readlink: readlink}
		rep, err := NewUseCase(intentWith(acceptedMD), nil, nil, s).PreWrite([]string{"notes.md"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Lines) != 1 || !rep.Failed || !strings.HasPrefix(rep.Lines[0], "✗ notes.md: its symlinks cannot be resolved (") {
			t.Fatalf("%s: lines = %q", name, rep.Lines)
		}
	}
}

// A relative path is taken from the working directory (S11), which may be
// below the project root.
func TestPreWriteRelativeToTheWorkingDirectory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, workDir, path string
		want                []string
	}{
		{name: "from a subdirectory", workDir: "/proj/docs/product", path: "intents/0009-x/plan.md", want: []string{notAccepted}},
		{name: "root-relative spelling from a subdirectory passes", workDir: "/proj/docs", path: prePlan},
		{name: "from the root", workDir: preRoot, path: prePlan, want: []string{notAccepted}},
		{name: "no working directory falls back to the root", path: prePlan, want: []string{notAccepted}},
		{name: "up and out of the root passes", workDir: "/proj/docs", path: "../../elsewhere/" + prePlan},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Settings{Root: preRoot, WorkDir: tt.workDir}
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

// A ".." after a symlink climbs from the link's target, as the kernel
// does, not from the link's own folder: lnk/../0009-x/plan.md with lnk
// pointing at an intent folder writes that intent's plan.md.
func TestPreWriteDotDotAfterASymlink(t *testing.T) {
	t.Parallel()
	links := map[string]string{"/proj/lnk": "/proj/docs/product/intents/0010-y"}
	exists := []string{"/proj", "/proj/docs", "/proj/docs/product", "/proj/docs/product/intents", "/proj/docs/product/intents/0009-x", "/proj/docs/product/intents/0010-y", "/proj/" + preSpec}
	for _, p := range []string{"lnk/../0009-x/plan.md", "/proj/lnk/../0009-x/plan.md", "./lnk/./../0009-x//plan.md"} {
		s := Settings{Root: preRoot, WorkDir: preRoot, Resolve: fakeResolve(links, exists...), Readlink: fakeReadlink(links, exists...)}
		rep, err := NewUseCase(intentWith(unfilledMD), nil, nil, s).PreWrite([]string{p})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(rep.Lines, []string{notAccepted}) || !rep.Failed {
			t.Fatalf("%s: lines = %q, want %q", p, rep.Lines, []string{notAccepted})
		}
	}
}

// On a case-sensitive filesystem /PROJ is another directory than /proj:
// a path there is outside the root and passes.
func TestPreWriteRootComparedWithCaseOnACaseSensitiveFilesystem(t *testing.T) {
	t.Parallel()
	rep, err := NewUseCase(intentWith(unfilledMD), nil, nil, Settings{Root: preRoot}).PreWrite([]string{"/PROJ/" + prePlan})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Lines) != 0 || rep.Failed {
		t.Fatalf("lines = %q, want none", rep.Lines)
	}
}
