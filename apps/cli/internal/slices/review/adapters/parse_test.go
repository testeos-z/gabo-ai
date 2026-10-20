package adapters

import (
	"errors"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

func TestParseFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		runtime string
		out     string
		want    int
		wantErr error
	}{
		{"claude plain", review.RuntimeClaude, `{"type":"result","is_error":false,"result":"{\"findings\":[{\"severity\":\"blocking\",\"file\":\"a.ts\",\"line\":3,\"title\":\"bug\",\"detail\":\"d\"}]}"}`, 1, nil},
		{"claude fenced with prose", review.RuntimeClaude, "{\"type\":\"result\",\"result\":\"Here you go:\\n```json\\n{\\\"findings\\\":[]}\\n```\"}", 0, nil},
		{"claude error", review.RuntimeClaude, `{"type":"result","is_error":true,"result":"rate limited"}`, 0, errors.New("x")},
		{"claude garbage", review.RuntimeClaude, `not json`, 0, review.ErrUnparseable},
		{"claude no findings key", review.RuntimeClaude, `{"type":"result","result":"{\"ok\":true}"}`, 0, review.ErrUnparseable},
		{"pi jsonl", review.RuntimePi, "{\"type\":\"start\"}\n{\"type\":\"message_end\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"{\\\"findings\\\":[{\\\"severity\\\":\\\"minor\\\",\\\"title\\\":\\\"nit\\\"}]}\"}]}}\n", 1, nil},
		{"pi garbage", review.RuntimePi, "{\"type\":\"start\"}\nnoise\n", 0, review.ErrUnparseable},
		{"findings not a list", review.RuntimePi, `{"findings":"none"}`, 0, review.ErrUnparseable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseFindings(tt.runtime, []byte(tt.out))
			if tt.wantErr != nil {
				if err == nil || (errors.Is(tt.wantErr, review.ErrUnparseable) && !errors.Is(err, review.ErrUnparseable)) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil || len(got) != tt.want {
				t.Fatalf("ParseFindings() = %+v, %v", got, err)
			}
		})
	}
}

func TestParseFindingsNormalizesSeverity(t *testing.T) {
	t.Parallel()
	got, err := ParseFindings(review.RuntimePi, []byte(`{"findings":[{"severity":"MINOR","title":"a"},{"severity":"high","title":""}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Severity != review.SeverityMinor || got[1].Severity != review.SeverityBlocking || got[1].Title == "" {
		t.Fatalf("findings = %+v", got)
	}
}
