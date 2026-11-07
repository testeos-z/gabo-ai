package check

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

const designNext = "next: fix docs/product/DESIGN.md above (gabo-frontend keeps it), then rerun gabo check"

// designProject is the passing project with testdata/design/<name>.md as
// docs/product/DESIGN.md.
func designProject(t *testing.T, name string) fstest.MapFS {
	t.Helper()
	b, err := os.ReadFile(path.Join("testdata", "design", name+".md"))
	if err != nil {
		t.Fatal(err)
	}
	return with(project(), map[string]string{designFixture: string(b)})
}

// Spec 0012 S7: like the intent folders, the design rule runs whether or
// not the config is valid.
func TestDesignRuleRunsWithAnInvalidConfig(t *testing.T) {
	t.Parallel()
	bad := strings.Replace(validConfig, "parallelism: 3", "parallelism: 0", 1)
	r := run(t, designProject(t, "design-no-table"), configFiles{projectFile: bad}, Settings{})
	wantFailed(t, r, NextFixConfig,
		`✗ docs/product/DESIGN.md: section 6 has no rules table and does not say "not applicable"`, designNext)
	if r.Lines[len(r.Lines)-1] != designNext {
		t.Fatalf("design lines are not last:\n%s", joined(r))
	}
}

// Spec 0012 S7: under GABO_UPGRADE_PR=1 only the config is validated.
func TestDesignRuleSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	r := run(t, designProject(t, "design-rule-rows"), configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	if r.Failed || !slices.Equal(r.Lines, []string{UpgradePRLine}) {
		t.Fatalf("upgrade PR: Failed = %v, lines = %q", r.Failed, r.Lines)
	}
}

// A DESIGN.md that cannot be read is a read error (exit 2), not a finding.
func TestDesignReadErrorIsAnError(t *testing.T) {
	t.Parallel()
	proj := project()
	proj[designFixture] = &fstest.MapFile{Mode: fs.ModeDir}
	s := Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}
	_, err := NewUseCase(proj, configFiles{projectFile: validConfig}, goodUser(), s).Run()
	if err == nil || !strings.Contains(err.Error(), designFixture) {
		t.Fatalf("Run() error = %v, want a read error naming %s", err, designFixture)
	}
}

// The title is the first "# " heading outside code fences; a file without
// one is in the earlier shape and only warns.
func TestDesignTitleIsTheFirstHeadingOutsideFences(t *testing.T) {
	t.Parallel()
	valid, err := os.ReadFile(path.Join("testdata", "design", "design-valid.md"))
	if err != nil {
		t.Fatal(err)
	}
	legacy := "! docs/product/DESIGN.md: not in the design norms shape (no \"# Design norms\" title); ask gabo-frontend to propose the conversion"
	tests := []struct {
		name, body string
		legacy     bool
	}{
		{name: "no title", body: "## 1. Principles and sources\n", legacy: true},
		{name: "fenced heading first", body: "```\n# Design\n```\n\n" + string(valid)},
		{name: "other title first", body: "# Design\n\n" + string(valid), legacy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := run(t, with(project(), map[string]string{designFixture: tt.body}), configFiles{projectFile: validConfig}, Settings{})
			if tt.legacy {
				wantPassed(t, r, legacy, PassedLine)
				if len(r.Lines) != 2 {
					t.Fatalf("legacy file got more than the warning:\n%s", joined(r))
				}
				return
			}
			wantPassed(t, r, PassedLine)
			if len(r.Lines) != 1 {
				t.Fatalf("lines = %q, want only the passed line", r.Lines)
			}
		})
	}
}

// Headings and tables inside a code fence are not sections or rules.
func TestDesignIgnoresFencedBlocks(t *testing.T) {
	t.Parallel()
	valid, err := os.ReadFile(path.Join("testdata", "design", "design-valid.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(string(valid), "## 2. Tokens\n",
		"```markdown\n## 7. Motion and state\n| id | rule |\n|---|---|\n| P1 | x |\n```\n\n## 2. Tokens\n", 1)
	r := run(t, with(project(), map[string]string{designFixture: body}), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	if len(r.Lines) != 1 {
		t.Fatalf("lines = %q, want only the passed line", r.Lines)
	}
}
