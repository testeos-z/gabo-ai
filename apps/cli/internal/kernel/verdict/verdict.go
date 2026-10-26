// Package verdict holds the signed reviewer verdict comment format shared by
// the review slice, which writes and gates merges on it, and the audit slice,
// which checks it after the fact.
//
// The reviewer posts with the owner's account, which GitHub forbids from
// approving, so the verdict travels in a PR comment: its first line is Clean
// or Blocking, and it ends with the visible Footer followed by the hidden
// Marker of the reviewed head.
package verdict

import (
	"fmt"
	"regexp"
	"strings"
)

// Verdict comment lines.
const (
	Clean    = "No blocking issues found."
	Blocking = "Blocking issues found."
	Footer   = "🤖 gbai reviewer"
)

var markerPattern = regexp.MustCompile(`^<!-- gbai-reviewer-bot sha=([0-9a-f]{40}) -->$`)

// Marker returns the hidden line that signs a verdict comment for sha.
func Marker(sha string) string {
	return fmt.Sprintf("<!-- gbai-reviewer-bot sha=%s -->", sha)
}

// MarkerSHA returns the head sha signed by the marker on the last non-empty
// line of body, if that line is a marker.
func MarkerSHA(body string) (string, bool) {
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(body, "\r\n", "\n"), "\n"), "\n")
	m := markerPattern.FindStringSubmatch(strings.TrimSpace(lines[len(lines)-1]))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// Comment is one PR comment and the login that posted it.
type Comment struct {
	Author string
	Body   string
}

// NewestSigned returns the newest comment in comments (oldest first) that
// reviewer posted and that carries a marker, plus the head it signs. Marker
// comments from anyone else are ignored, and an empty reviewer matches none.
func NewestSigned(comments []Comment, reviewer string) (Comment, string, bool) {
	if reviewer == "" {
		return Comment{}, "", false
	}
	for i := len(comments) - 1; i >= 0; i-- {
		if comments[i].Author != reviewer {
			continue
		}
		if signed, ok := MarkerSHA(comments[i].Body); ok {
			return comments[i], signed, true
		}
	}
	return Comment{}, "", false
}

// Lines splits a comment body into lines without carriage returns.
func Lines(body string) []string {
	return strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
}

// OpensClean reports whether body's first line is Clean.
func OpensClean(body string) bool {
	return Lines(body)[0] == Clean
}

// CleanFor reports whether the newest marker comment reviewer posted in
// comments (oldest first) signs head and opens with Clean.
func CleanFor(comments []Comment, reviewer, head string) bool {
	c, signed, ok := NewestSigned(comments, reviewer)
	return ok && signed == head && OpensClean(c.Body)
}
