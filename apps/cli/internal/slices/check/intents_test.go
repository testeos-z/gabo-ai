package check

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"testing"
	"testing/fstest"
)

// withTree adds the files of testdata/intents/<name> to a copy of base, so
// a fixture tree is checked through the public Run on a passing project.
func withTree(t *testing.T, base fstest.MapFS, name string) fstest.MapFS {
	t.Helper()
	out := with(base, nil)
	root := os.DirFS(path.Join("testdata", "intents", name))
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		out[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return out
}

func runIntents(t *testing.T, name string) Report {
	t.Helper()
	return run(t, withTree(t, project(), name), configFiles{projectFile: validConfig}, Settings{})
}

const intentsP = "docs/product/intents/"

func TestIntentStages(t *testing.T) {
	t.Parallel()
	a := intentsP + "0001-alpha"
	tests := []struct {
		fixture string
		fail    []string
	}{
		{fixture: "valid"},
		{fixture: "empty"},
		{fixture: "bad-name", fail: []string{
			"✗ " + intentsP + "0002-Bad_Slug: folder name must be NNNN-slug (4 digits, then lowercase words joined by hyphens)",
			"✗ " + intentsP + "1-Login: folder name must be NNNN-slug (4 digits, then lowercase words joined by hyphens)",
		}},
		{fixture: "duplicate-number", fail: []string{
			"✗ intent number 0001 used twice: " + a + " and " + intentsP + "0001-beta",
		}},
		{fixture: "no-intent", fail: []string{
			"✗ " + a + ": no intent.md",
		}},
		{fixture: "plan-without-spec", fail: []string{
			"✗ " + a + "/plan.md: no spec.md; a plan starts only after a human accepts the spec",
		}},
		{fixture: "spec-not-accepted", fail: []string{
			"✗ " + a + "/plan.md: spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan",
		}},
		{fixture: "no-task-table", fail: []string{
			"✗ " + a + "/plan.md: no task table (| id | title | spec refs | depends_on | unblocks | acceptance | size |)",
		}},
		{fixture: "duplicate-task", fail: []string{
			"✗ " + a + "/plan.md: task T1 listed twice",
			"✗ " + a + `/plan.md: task id "X2" is not T<n>`,
		}},
		{fixture: "mirror", fail: []string{
			"✗ " + a + "/plan.md: T1 unblocks T2, but T2 does not list T1 in depends_on",
			"✗ " + a + "/plan.md: T3 depends_on T1, but T1 does not list T3 in unblocks",
			"✗ " + a + "/plan.md: T3 depends_on T9, which is not a task of this plan",
			"✗ " + a + `/plan.md: T4 depends_on "foo", not T<n> or <NNNN>/T<n>`,
		}},
		{fixture: "cross-missing", fail: []string{
			"✗ " + a + "/plan.md: T1 depends_on 0009/T1, but no intent 0009 exists",
			"✗ " + a + "/plan.md: T1 depends_on 0002/T7, but " + intentsP + "0002-beta/plan.md has no task T7",
			"✗ " + a + "/plan.md: T2 depends_on 0003/T1, but " + intentsP + "0003-gamma/plan.md has no task T1",
		}},
		{fixture: "cycle", fail: []string{
			"✗ dependency cycle: 0001/T1 -> 0001/T2 -> 0001/T1",
			"✗ dependency cycle: 0001/T3 -> 0002/T1 -> 0001/T3",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			t.Parallel()
			r := runIntents(t, tt.fixture)
			if len(tt.fail) == 0 {
				wantPassed(t, r, PassedLine)
				if !slices.Equal(r.Lines, []string{PassedLine}) {
					t.Fatalf("lines = %q, want only the passed line", r.Lines)
				}
				return
			}
			wantFailed(t, r, append(tt.fail, NextFixIntents)...)
			var got []string
			for _, l := range r.Lines {
				if l != NextFixIntents {
					got = append(got, l)
				}
			}
			if !slices.Equal(got, tt.fail) {
				t.Fatalf("lines =\n%q\nwant exactly\n%q", got, tt.fail)
			}
			if r.Lines[len(r.Lines)-1] != NextFixIntents {
				t.Fatalf("last line = %q, want %q", r.Lines[len(r.Lines)-1], NextFixIntents)
			}
		})
	}
}

// A project without docs/product/intents/ is not touched by the intent
// check: its output is exactly the passed line, as before the check.
func TestIntentsDirMissingLeavesOutputUnchanged(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{})
	if r.Failed || !slices.Equal(r.Lines, []string{PassedLine}) {
		t.Fatalf("Failed = %v, lines = %q, want only %q", r.Failed, r.Lines, PassedLine)
	}
}

// The intent check also runs when another check fails, and never in
// upgrade-PR mode, where only the config is validated.
func TestIntentCheckRunsBesideOtherFailuresButNotInUpgradePR(t *testing.T) {
	t.Parallel()
	proj := with(withTree(t, project(), "no-intent"), map[string]string{".claude/agents/beta.md": ""})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ agent beta missing", "✗ "+intentsP+"0001-alpha: no intent.md", NextFixIntents)

	r = run(t, withTree(t, project(), "no-intent"), configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	if r.Failed || !slices.Equal(r.Lines, []string{UpgradePRLine}) {
		t.Fatalf("upgrade PR: Failed = %v, lines = %q", r.Failed, r.Lines)
	}
}
