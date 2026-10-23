package review

import (
	"errors"
	"strings"
	"testing"
)

const (
	sha = "0123456789abcdef0123456789abcdef01234567"
	old = "fedcba9876543210fedcba9876543210fedcba98"
)

func TestDecideVerdict(t *testing.T) {
	t.Parallel()
	pass := CheckResult{Command: "bun run test", Passed: true}
	fail := CheckResult{Command: "bun run test", Passed: false}
	minor := Finding{Severity: SeverityMinor, Title: "nit"}
	blocking := Finding{Severity: SeverityBlocking, Title: "bug"}
	unknown := Finding{Severity: "critical", Title: "?"}
	tests := []struct {
		name string
		ev   Evidence
		want Verdict
	}{
		{"clean with minor findings", Evidence{Checks: []CheckResult{pass}, Findings: []Finding{minor}}, VerdictClean},
		{"failed check", Evidence{Checks: []CheckResult{pass, fail}}, VerdictBlocking},
		{"blocking finding", Evidence{Checks: []CheckResult{pass}, Findings: []Finding{minor, blocking}}, VerdictBlocking},
		{"unknown severity counts as blocking", Evidence{Checks: []CheckResult{pass}, Findings: []Finding{unknown}}, VerdictBlocking},
		{"conflict", Evidence{Conflict: true}, VerdictBlocking},
		{"infra wins over everything", Evidence{Infra: errors.New("boom"), Conflict: true, Checks: []CheckResult{fail}}, VerdictError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DecideVerdict(tt.ev); got != tt.want {
				t.Fatalf("DecideVerdict() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecideMerge(t *testing.T) {
	t.Parallel()
	policy := MergePolicy{AutoMerge: true, Integration: "dev", HumanOnlyPaths: DefaultHumanOnlyPaths()}
	cloud := PullRequest{HeadSHA: sha, BaseBranch: "dev", Labels: []string{"bug", CloudLabel}}
	local := PullRequest{HeadSHA: sha, BaseBranch: "dev", Labels: []string{"mode:local"}}
	files := []string{"apps/web/src/a.ts", "README.md"}
	clean := []string{CommentBody(sha, "dev", Evidence{}, VerdictClean)}
	tests := []struct {
		name     string
		verdict  Verdict
		pr       PullRequest
		files    []string
		policy   MergePolicy
		comments []string
		want     Merge
	}{
		{"clean cloud auto merge", VerdictClean, cloud, files, policy, clean, MergeMerged},
		{"clean without cloud label", VerdictClean, local, files, policy, clean, MergeNeedsHuman},
		{"clean with auto merge off", VerdictClean, cloud, files, MergePolicy{Integration: "dev", HumanOnlyPaths: DefaultHumanOnlyPaths()}, clean, MergeNeedsHuman},
		{"base is not integration", VerdictClean, PullRequest{BaseBranch: "main", Labels: []string{CloudLabel}}, files, policy, clean, MergeNeedsHuman},
		{"workflow change", VerdictClean, cloud, []string{"a.ts", ".github/workflows/ci.yml"}, policy, clean, MergeNeedsHuman},
		{"claude settings", VerdictClean, cloud, []string{".claude/settings.json"}, policy, clean, MergeNeedsHuman},
		{"opencode", VerdictClean, cloud, []string{".opencode/x"}, policy, clean, MergeNeedsHuman},
		{"root AGENTS.md", VerdictClean, cloud, []string{"AGENTS.md"}, policy, clean, MergeNeedsHuman},
		{"nested CLAUDE.md", VerdictClean, cloud, []string{"apps/web/CLAUDE.md"}, policy, clean, MergeNeedsHuman},
		{"policies", VerdictClean, cloud, []string{"docs/policies/merge.md"}, policy, clean, MergeNeedsHuman},
		{"hook tests", VerdictClean, cloud, []string{"tools/hook-tests/a.sh"}, policy, clean, MergeNeedsHuman},
		{"agents dir", VerdictClean, cloud, []string{"agents/pr-reviewer/run.sh"}, policy, clean, MergeNeedsHuman},
		{"similar names are not human-only", VerdictClean, cloud, []string{"docs/policies-old.md", "src/agents/x.ts", "AGENTS.md.bak"}, policy, clean, MergeMerged},
		{"no changed files is suspicious", VerdictClean, cloud, nil, policy, clean, MergeNeedsHuman},
		{"blocking", VerdictBlocking, cloud, files, policy, clean, MergeNone},
		{"error", VerdictError, cloud, files, policy, clean, MergeNone},
		{"skipped", VerdictSkipped, cloud, files, policy, clean, MergeSkipped},
		{"no marker comment", VerdictClean, cloud, files, policy, []string{"No blocking issues found.\n"}, MergeNeedsHuman},
		{"newest marker is for an older head", VerdictClean, cloud, files, policy, []string{clean[0], CommentBody(old, "dev", Evidence{}, VerdictClean)}, MergeNeedsHuman},
		{"newest marker is blocking", VerdictClean, cloud, files, policy, []string{clean[0], CommentBody(sha, "dev", Evidence{}, VerdictBlocking)}, MergeNeedsHuman},
		{"older blocking marker, newest clean for head", VerdictClean, cloud, files, policy, []string{CommentBody(sha, "dev", Evidence{}, VerdictBlocking), clean[0], "thanks!"}, MergeMerged},
		{"no comments", VerdictClean, cloud, files, policy, nil, MergeNeedsHuman},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DecideMerge(tt.verdict, tt.pr, tt.files, tt.comments, tt.policy); got != tt.want {
				t.Fatalf("DecideMerge() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSummaryRoundTrip(t *testing.T) {
	t.Parallel()
	s := Summary{PR: 12, SHA: sha, Verdict: VerdictClean, Merge: MergeMerged}
	line := s.String()
	if line != "PR #12 head "+sha+" verdict=clean merge=merged" {
		t.Fatalf("String() = %q", line)
	}
	got, ok := ParseSummary(line)
	if !ok || got != s {
		t.Fatalf("ParseSummary() = %+v, %v", got, ok)
	}
	for _, bad := range []string{
		"", "PR #12 head " + sha + " verdict=ok merge=merged", "x PR #12 head " + sha + " verdict=clean merge=merged",
		"PR #12 head " + strings.ToUpper(sha) + " verdict=clean merge=merged", "PR #12 head " + sha + " verdict=clean merge=merged extra",
	} {
		if _, ok := ParseSummary(bad); ok {
			t.Errorf("ParseSummary(%q) ok = true", bad)
		}
	}
}

func TestValidators(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		if ValidatePR(n) == nil {
			t.Errorf("ValidatePR(%d) = nil", n)
		}
	}
	if ValidatePR(1) != nil || ValidateSHA(sha) != nil {
		t.Error("valid input rejected")
	}
	for _, s := range []string{"", sha[:39], strings.ToUpper(sha), sha + "0", "g" + sha[1:], "$(rm -rf)" + sha[9:]} {
		if ValidateSHA(s) == nil {
			t.Errorf("ValidateSHA(%q) = nil", s)
		}
	}
	for _, m := range []string{"run:abc-1", "dispatch:x.Y_9"} {
		if ValidateMailbox(m) != nil {
			t.Errorf("ValidateMailbox(%q) rejected", m)
		}
	}
	for _, m := range []string{"", "run:", "user:x", "run:a b", "run:a;rm", "dispatch:$(x)"} {
		if ValidateMailbox(m) == nil {
			t.Errorf("ValidateMailbox(%q) = nil", m)
		}
	}
}

func TestTailKeepsLastLines(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for i := range 100 {
		b.WriteString(strings.Repeat("x", i%3) + "\n")
	}
	b.WriteString("last")
	got := Tail([]byte(b.String()), 60)
	if lines := strings.Split(got, "\n"); len(lines) != 60 || lines[59] != "last" {
		t.Fatalf("Tail() has %d lines, last %q", len(lines), lines[len(lines)-1])
	}
}

func TestCommentBodyIsSignedWithFooterAndMarker(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		verdict Verdict
		first   string
	}{{VerdictClean, "No blocking issues found."}, {VerdictBlocking, "Blocking issues found."}, {VerdictError, "Blocking issues found."}} {
		body := CommentBody(sha, "dev", Evidence{Findings: []Finding{{Severity: SeverityMinor, Title: "nit"}}}, tt.verdict)
		lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
		if lines[0] != tt.first {
			t.Errorf("%s: first line = %q, want %q", tt.verdict, lines[0], tt.first)
		}
		n := len(lines)
		if lines[n-2] != "🤖 gbai reviewer" || lines[n-1] != "<!-- gbai-reviewer-bot sha="+sha+" -->" {
			t.Errorf("%s: body does not end with footer and marker:\n%s", tt.verdict, body)
		}
		if got, ok := MarkerSHA(body); !ok || got != sha {
			t.Errorf("%s: MarkerSHA() = %q, %v", tt.verdict, got, ok)
		}
	}
}

func TestMarkerSHA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{"marker last", "x\n" + Marker(sha), sha, true},
		{"marker with trailing newline and CRLF", "x\r\n" + Marker(sha) + "\r\n", sha, true},
		{"no marker", "No blocking issues found.\n", "", false},
		{"marker not on the last line", Marker(sha) + "\nquoted", "", false},
		{"short sha", "<!-- gbai-reviewer-bot sha=0123456 -->", "", false},
		{"uppercase sha", Marker(strings.ToUpper(sha)), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := MarkerSHA(tt.body)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("MarkerSHA() = %q, %v; want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
