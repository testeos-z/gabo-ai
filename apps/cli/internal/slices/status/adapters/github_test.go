package adapters

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/status"
)

func respond(m map[string]string) func(kernel.Command) (kernel.Result, error) {
	return func(c kernel.Command) (kernel.Result, error) {
		key := strings.Join(c.Args, " ")
		for prefix, body := range m {
			if strings.HasPrefix(key, prefix) {
				return kernel.Result{Stdout: []byte(body)}, nil
			}
		}
		return kernel.Result{ExitCode: 1, Stderr: []byte("unexpected " + key)}, nil
	}
}

func TestOpenPullRequests(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: respond(map[string]string{"pr list": `[{"number":7,"title":"x","headRefOid":"aaa","labels":[{"name":"mode:cloud"}]}]`})}
	prs, err := NewGitHub(r, "o/r").OpenPullRequests(context.Background(), "dev")
	if err != nil || len(prs) != 1 || prs[0].HeadSHA != "aaa" || !slices.Equal(prs[0].Labels, []string{"mode:cloud"}) {
		t.Fatalf("prs = %+v, %v", prs, err)
	}
	want := []string{"pr", "list", "--repo", "o/r", "--base", "dev", "--state", "open", "--json", "number,title,headRefOid,labels", "--limit", "100"}
	if got := r.Calls[0].Command.Args; !slices.Equal(got, want) {
		t.Fatalf("argv = %v", got)
	}
}

func TestLatestStatusPicksNewestOfContext(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: respond(map[string]string{"api repos/o/r/commits/aaa/statuses": `[{"context":"reviewer/tests","state":"failure"},{"context":"reviewer/review","state":"success"},{"context":"reviewer/review","state":"failure"}]`})}
	got, err := NewGitHub(r, "o/r").LatestStatus(context.Background(), "aaa", status.ReviewContext)
	if err != nil || got != "success" {
		t.Fatalf("LatestStatus() = %q, %v", got, err)
	}
}

func TestLastCommentByAuthorIgnoresBotSuffix(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: respond(map[string]string{"pr view 7": `{"comments":[{"author":{"login":"vulcambot"},"body":"old"},{"author":{"login":"human"},"body":"hi"},{"author":{"login":"vulcambot[bot]"},"body":"new"}]}`})}
	got, err := NewGitHub(r, "o/r").LastComment(context.Background(), 7, "vulcambot")
	if err != nil || got != "new" {
		t.Fatalf("LastComment() = %q, %v", got, err)
	}
}

func TestNeedsUserIssues(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: respond(map[string]string{"issue list": `[{"number":3,"title":"decide"}]`})}
	issues, err := NewGitHub(r, "o/r").NeedsUserIssues(context.Background())
	if err != nil || len(issues) != 1 || issues[0].Number != 3 {
		t.Fatalf("issues = %+v, %v", issues, err)
	}
	if !slices.Contains(r.Calls[0].Command.Args, "needs-user") {
		t.Fatalf("argv = %v", r.Calls[0].Command.Args)
	}
}
