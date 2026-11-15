package config

import (
	"strings"
	"testing"
)

// S3: the required approving reviews are read per branch, zero when absent.
func TestRepoProtectionApprovalsDefaultToZero(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.Repo.Protection.RequiredApprovals; got.Integration != 0 || got.Main != 0 {
		t.Errorf("required_approvals = %+v, want zero on both branches", got)
	}
}

func TestRepoProtectionApprovalsReadPerBranch(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(with("repo: {protection: {required_approvals: {main: 1}}}")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.Repo.Protection.RequiredApprovals; got.Main != 1 || got.Integration != 0 {
		t.Errorf("required_approvals = %+v, want main 1 and integration 0", got)
	}
}

// A negative count is refused with the field path, in the existing error shape.
func TestRepoProtectionApprovalsRejectNegative(t *testing.T) {
	t.Parallel()
	for _, branch := range []string{"integration", "main"} {
		doc := with("repo: {protection: {required_approvals: {" + branch + ": -1}}}")
		_, err := Parse([]byte(doc))
		want := "repo.protection.required_approvals." + branch
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse() error = %v, want %s named", err, want)
		}
	}
}
