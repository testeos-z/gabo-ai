package adapters

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// ParseFindings extracts the findings document from engine output. claude
// prints one result object; pi prints JSON lines of events. Any JSON string
// in the output may carry the document, possibly wrapped in prose or fences;
// the last valid one wins.
func ParseFindings(runtime string, out []byte) ([]review.Finding, error) {
	if runtime == review.RuntimeClaude {
		var res struct {
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
		}
		if json.Unmarshal(bytes.TrimSpace(out), &res) == nil && res.IsError {
			return nil, errors.New("claude reported an error result")
		}
	}
	var candidates []string
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var v any
		if json.Unmarshal(line, &v) != nil {
			candidates = append(candidates, string(line))
			continue
		}
		candidates = append(candidates, string(line))
		collectStrings(v, &candidates)
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		if fs, ok := findingsIn(candidates[i]); ok {
			return fs, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", runtime, review.ErrUnparseable)
}

func collectStrings(v any, out *[]string) {
	switch t := v.(type) {
	case string:
		if strings.Contains(t, "findings") {
			*out = append(*out, t)
		}
	case []any:
		for _, e := range t {
			collectStrings(e, out)
		}
	case map[string]any:
		for _, e := range t {
			collectStrings(e, out)
		}
	}
}

// findingsIn returns the last JSON object in text that has a findings list.
func findingsIn(text string) ([]review.Finding, bool) {
	var found []review.Finding
	ok := false
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		dec := json.NewDecoder(strings.NewReader(text[i:]))
		var doc struct {
			Findings *[]review.Finding `json:"findings"`
		}
		if dec.Decode(&doc) != nil || doc.Findings == nil {
			continue
		}
		found, ok = normalize(*doc.Findings), true
		i += int(dec.InputOffset()) - 1
	}
	return found, ok
}

func normalize(fs []review.Finding) []review.Finding {
	out := make([]review.Finding, 0, len(fs))
	for _, f := range fs {
		if review.Severity(strings.ToLower(string(f.Severity))) == review.SeverityMinor {
			f.Severity = review.SeverityMinor
		} else {
			f.Severity = review.SeverityBlocking
		}
		if strings.TrimSpace(f.Title) == "" {
			f.Title = "(untitled finding)"
		}
		out = append(out, f)
	}
	return out
}
