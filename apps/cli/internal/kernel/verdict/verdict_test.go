package verdict

import (
	"strings"
	"testing"
)

const (
	sha      = "0123456789abcdef0123456789abcdef01234567"
	old      = "fedcba9876543210fedcba9876543210fedcba98"
	reviewer = "owner"
)

func body(first, signed string) string {
	return first + "\n\nnotes\n\n" + Footer + "\n" + Marker(signed)
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

func TestNewestSignedIgnoresOtherAuthorsAndUnsignedComments(t *testing.T) {
	t.Parallel()
	comments := []Comment{
		{Author: reviewer, Body: body(Clean, old)},
		{Author: reviewer, Body: "plain note"},
		{Author: "mallory", Body: body(Clean, sha)},
	}
	c, signed, ok := NewestSigned(comments, reviewer)
	if !ok || signed != old || c != comments[0] {
		t.Fatalf("NewestSigned() = %+v, %q, %v", c, signed, ok)
	}
	if _, _, ok := NewestSigned(comments, ""); ok {
		t.Fatal("an empty reviewer matched a comment")
	}
}

func TestCleanFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{"clean on head", []Comment{{reviewer, body(Clean, sha)}}, true},
		{"no comments", nil, false},
		{"older head", []Comment{{reviewer, body(Clean, old)}}, false},
		{"another author", []Comment{{"mallory", body(Clean, sha)}}, false},
		{"blocking", []Comment{{reviewer, body(Blocking, sha)}}, false},
		{"newest wins", []Comment{{reviewer, body(Clean, sha)}, {reviewer, body(Blocking, sha)}}, false},
		{"clean line not first", []Comment{{reviewer, "x\n" + body(Clean, sha)}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CleanFor(tt.comments, reviewer, sha); got != tt.want {
				t.Fatalf("CleanFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
