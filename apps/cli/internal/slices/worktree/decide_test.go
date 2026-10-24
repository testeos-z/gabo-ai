package worktree

import (
	"strings"
	"testing"
)

func okPath() PathFacts {
	return PathFacts{
		Arg:          "/src/repo-worktrees/12-x",
		Real:         "/src/repo-worktrees/12-x",
		Main:         "/src/repo",
		WorktreesDir: "/src/repo-worktrees",
		Current:      "/src/repo",
		Registered:   true,
	}
}

func TestCheckPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*PathFacts)
		want   string // substring of the refusal; "" means accepted
	}{
		{"approved path", func(*PathFacts) {}, ""},
		{"relative path", func(f *PathFacts) { f.Arg = "repo-worktrees/12-x" }, "not absolute"},
		{"dot dot", func(f *PathFacts) { f.Arg = "/src/repo-worktrees/../repo-worktrees/12-x" }, ".."},
		{"symlink", func(f *PathFacts) { f.Symlink = true }, "symlink"},
		{"missing", func(f *PathFacts) { f.Real = "" }, "does not exist"},
		{"main worktree", func(f *PathFacts) { f.Arg, f.Real = "/src/repo", "/src/repo" }, "main worktree is never closed"},
		{"no worktrees dir", func(f *PathFacts) { f.WorktreesDir = "" }, "worktrees directory"},
		{"outside worktrees dir", func(f *PathFacts) { f.Arg, f.Real = "/tmp/12-x", "/tmp/12-x" }, "not directly in"},
		{"nested in worktrees dir", func(f *PathFacts) { f.Arg, f.Real = "/src/repo-worktrees/a/b", "/src/repo-worktrees/a/b" }, "not directly in"},
		{"symlinked parent escape", func(f *PathFacts) { f.Real = "/src/repo-worktrees/other" }, "not directly in"},
		{"current worktree", func(f *PathFacts) { f.Current = f.Real }, "runs in"},
		{"not registered", func(f *PathFacts) { f.Registered = false }, "not a worktree of this repository"},
		{"kept by name", func(f *PathFacts) { f.Keep = []string{"pr-399", "12-x"} }, "keep list"},
		{"kept by path", func(f *PathFacts) { f.Keep = []string{"/src/repo-worktrees/12-x"} }, "keep list"},
		{"other kept", func(f *PathFacts) { f.Keep = []string{"13-y"} }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := okPath()
			tt.mutate(&f)
			got := CheckPath(f)
			if tt.want == "" && got != "" || tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("CheckPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func okState() StateFacts {
	return StateFacts{
		Integration: "dev",
		Branch:      "feat/x",
		Head:        "h1",
		BaseTip:     "b1",
		OwnCommits:  true,
		HasUpstream: true,
		InBase:      true,
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*StateFacts)
		close  bool
		want   string // the CLOSED reason, or a substring of the refusal
	}{
		{"in base", func(*StateFacts) {}, true, ReasonInBase},
		{"merged pr", func(f *StateFacts) { f.InBase, f.MergedPR = false, 41 }, true, ReasonMergedPR},
		{"merged pr without upstream", func(f *StateFacts) { f.InBase, f.MergedPR, f.HasUpstream = false, 41, false }, true, ReasonMergedPR},
		{"in base without upstream", func(f *StateFacts) { f.HasUpstream = false }, true, ReasonInBase},
		{"detached in base", func(f *StateFacts) { f.Branch, f.OwnCommits = "", false }, true, ReasonInBase},
		{"merged pr at the base tip", func(f *StateFacts) { f.Head, f.OwnCommits, f.MergedPR = "b1", false, 41 }, true, ReasonInBase},
		{"dirty", func(f *StateFacts) { f.Changed = 2 }, false, "not clean"},
		{"ignored env", func(f *StateFacts) { f.Ignored = []string{"node_modules/", "apps/web/.env.local"} }, false, "apps/web/.env.local"},
		{"ignored key under dist", func(f *StateFacts) { f.Ignored = []string{"dist/certs/server.key"} }, false, "dist/certs/server.key"},
		{"ignored db under build", func(f *StateFacts) { f.Ignored = []string{"build/data.db"} }, false, "build/data.db"},
		{"ignored rebuildables", func(f *StateFacts) { f.Ignored = []string{"node_modules/", "dist/index.js", "x.tsbuildinfo"} }, true, ReasonInBase},
		{"unpushed", func(f *StateFacts) { f.Unpushed = 3 }, false, "3 unpushed"},
		{"open pr", func(f *StateFacts) { f.OpenPR = 7 }, false, "open PR #7"},
		{"never worked at base tip", func(f *StateFacts) { f.Head = "b1" }, false, "never worked"},
		{"never worked no own commits", func(f *StateFacts) { f.OwnCommits = false }, false, "never worked"},
		{"detached at base tip", func(f *StateFacts) { f.Branch, f.Head = "", "b1" }, false, "never worked"},
		{"not merged", func(f *StateFacts) { f.InBase = false }, false, "not merged"},
		{"detached not merged", func(f *StateFacts) { f.Branch, f.InBase, f.MergedPR = "", false, 41 }, false, "not merged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := okState()
			tt.mutate(&f)
			d := Decide(f)
			if d.Close != tt.close || !strings.Contains(d.Reason, tt.want) || tt.close && d.Reason != tt.want {
				t.Fatalf("Decide() = %+v, want close=%v reason %q", d, tt.close, tt.want)
			}
		})
	}
}

func TestIsSecret(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		".env": true, "apps/api/.env.production": true, "certs/tls.pem": true, "a/b.key": true,
		"id_ed25519": true, "home/.ssh/id_rsa.pub": true, "store.p12": true, "local.sqlite": true,
		"data/app.db": true, "app.sqlite3": true, "app.db-wal": true, "BIG.PEM": true,
		"dist/index.js": false, "node_modules/": false, "envoy.yaml": false, "keys.ts": false, "x.tsbuildinfo": false,
	} {
		if got := IsSecret(path); got != want {
			t.Errorf("IsSecret(%q) = %v, want %v", path, got, want)
		}
	}
}
