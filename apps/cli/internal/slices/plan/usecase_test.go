package plan

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

var docNames = []string{"PRD.md", "TRD.md", "IMPLEMENTATION_PLAN.md", "DB_SCHEMA.md", "UX_UI.md", "APP_FLOW.md"}

func TestNewScaffoldsTheSixDocsFromTemplates(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{}
	out, err := NewUseCase(fsys).New(context.Background(), "docs/product", false)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, n := range docNames {
		want = append(want, "docs/product/"+n)
	}
	if !slices.Equal(fsys.Writes, want) {
		t.Fatalf("writes = %v", fsys.Writes)
	}
	for _, p := range want {
		if !strings.Contains(fsys.Files[p], Placeholder) || !strings.HasPrefix(fsys.Files[p], "# ") {
			t.Errorf("%s is not a template:\n%.80s", p, fsys.Files[p])
		}
	}
	if !strings.Contains(fsys.Files["docs/product/IMPLEMENTATION_PLAN.md"], "| id | title | depends_on | unblocks | acceptance |") {
		t.Error("implementation plan has no task table")
	}
	if len(out.Lines) != 6 || !strings.HasPrefix(out.Lines[0], "created docs/product/PRD.md") {
		t.Fatalf("lines = %v", out.Lines)
	}
}

func TestNewNeverOverwritesWithoutForce(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{Files: map[string]string{"p/PRD.md": "# Mine\n"}}
	out, err := NewUseCase(fsys).New(context.Background(), "p", false)
	if err != nil {
		t.Fatal(err)
	}
	if fsys.Files["p/PRD.md"] != "# Mine\n" || slices.Contains(fsys.Writes, "p/PRD.md") {
		t.Fatal("PRD overwritten")
	}
	if !slices.Contains(out.Lines, "kept p/PRD.md (exists; use --force)") {
		t.Fatalf("lines = %v", out.Lines)
	}
	if _, err := NewUseCase(fsys).New(context.Background(), "p", true); err != nil || fsys.Files["p/PRD.md"] == "# Mine\n" {
		t.Fatalf("force: %v", err)
	}
}

func TestStatusReportsMissingTemplateAndWritten(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{}
	uc := NewUseCase(fsys)
	if _, err := uc.New(context.Background(), "p", false); err != nil {
		t.Fatal(err)
	}
	fsys.Files["p/PRD.md"] = "# PRD\nAll decided.\n"
	delete(fsys.Files, "p/TRD.md")
	st, err := uc.Status(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 6 {
		t.Fatalf("status = %+v", st)
	}
	if st[0].Name != "PRD.md" || st[0].State != StateComplete || st[0].Placeholders != 0 {
		t.Errorf("PRD = %+v", st[0])
	}
	if st[1].State != StateMissing {
		t.Errorf("TRD = %+v", st[1])
	}
	if st[2].State != StateTemplate || st[2].Placeholders == 0 {
		t.Errorf("plan = %+v", st[2])
	}
}
