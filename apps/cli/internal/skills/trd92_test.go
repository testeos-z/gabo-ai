package skills

import (
	"os"
	"strings"
	"testing"
)

// Intent 0027 S9 (plan T6, TRD 9.2 gains the fields, D3): section 9.2 of the
// TRD names the coder outcome fields of S8 and the wave report Tasks values of
// S9. These tests read the real document, so an edit that drops a field fails.
const trdDoc = "../../../../docs/product/TRD.md"

// trd92Bullet returns the one bullet of TRD section 9.2 that starts with prefix.
func trd92Bullet(t *testing.T, prefix string) string {
	t.Helper()
	b, err := os.ReadFile(trdDoc)
	if err != nil {
		t.Fatalf("read %s: %v", trdDoc, err)
	}
	_, rest, ok := strings.Cut(string(b), "### 9.2 Inter-agent payloads")
	if !ok {
		t.Fatal("TRD: no section 9.2")
	}
	sec, _, _ := strings.Cut(rest, "### 9.3 ")
	for _, line := range strings.Split(sec, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("TRD 9.2: no bullet starting %q", prefix)
	return ""
}

// The coder outcome carries the S8 methodology block: every field, and the
// not-run rule that keeps a missing field from reading as a pass.
func TestTRD92ListsTheCoderOutcomeFields(t *testing.T) {
	t.Parallel()
	requireIn(t, "TRD 9.2 coder outcome", trd92Bullet(t, "- **Coder → cycle (outcome).**"), []string{
		"`odd:`",
		"`spec S<n>:`",
		"`contract:`",
		"`tests:`",
		"`integration:`",
		"`check:`",
		"`exit: <n>`",
		"`commits:`",
		"`verifier:`",
		"`RDD:`",
		"`risk`",
		"`review_due_reason`",
		"`not run: <reason>`",
	})
}

// The wave report Tasks line shows the RDD, ODD and gates values of S9 for
// every task, copied from the wave trace.
func TestTRD92ListsTheWaveReportTaskFields(t *testing.T) {
	t.Parallel()
	requireIn(t, "TRD 9.2 wave report", trd92Bullet(t, "- **Cycle → assistant (wave report).**"), []string{
		"`Tasks`",
		"`RDD: <outcome>`",
		"`ODD: <RED/GREEN or exception>`",
		"`gates: done <passed or refused>, report <passed or refused>`",
		"`not run: <reason>`",
	})
}
