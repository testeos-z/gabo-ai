package adapters

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
)

func TestMergedPullRequests(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`[{"number":5,"mergedBy":{"login":"vulcambot"},"headRefOid":"abc","mergedAt":"2026-10-02T10:00:00Z"},{"number":6,"mergedBy":null,"headRefOid":"def","mergedAt":"2026-10-03T10:00:00Z"}]`)}}
	prs, err := NewGitHub(r, "o/r").MergedPullRequests(context.Background(), "dev", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MergedPullRequests() error = %v", err)
	}
	want := []audit.MergedPR{
		{Number: 5, MergedBy: "vulcambot", HeadSHA: "abc", MergedAt: time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)},
		{Number: 6, HeadSHA: "def", MergedAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)},
	}
	if len(prs) != 2 || prs[0] != want[0] || prs[1] != want[1] {
		t.Fatalf("prs = %+v", prs)
	}
	wantArgs := []string{"pr", "list", "--repo", "o/r", "--base", "dev", "--state", "merged", "--search", "merged:>=2026-10-01",
		"--json", "number,mergedBy,headRefOid,mergedAt", "--limit", "1000"}
	if got := r.Calls[0].Command.Args; !slices.Equal(got, wantArgs) {
		t.Fatalf("argv = %v", got)
	}
}

func TestStatuses(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`[{"context":"reviewer/review","state":"success","creator":{"login":"vulcambot[bot]"}}]`)}}
	got, err := NewGitHub(r, "o/r").Statuses(context.Background(), "abc")
	if err != nil || len(got) != 1 || got[0] != (audit.CommitStatus{Context: "reviewer/review", State: "success", Creator: "vulcambot[bot]"}) {
		t.Fatalf("Statuses() = %+v, %v", got, err)
	}
	if got := r.Calls[0].Command.Args; !slices.Equal(got, []string{"api", "repos/o/r/commits/abc/statuses?per_page=100"}) {
		t.Fatalf("argv = %v", got)
	}
}
