package check

import (
	"slices"
	"testing"
)

const tasksContract = "---\nname: gabo-contract-tasks\n---\n## Hard Rules\n\n### not an op\n\n## Operations\n\n### create\n\ntext\n\n### read one\n\n### claim\n\n```\n### inside a fence\n```\n\n## Output Contract\n\n### also not an op\n"

func TestCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		adapter          string
		covered, missing []string
	}{
		{
			name:    "complete adapter",
			adapter: "## Operations\r\n\r\n### claim\r\n\r\n### create\r\n\r\n### read one\r\n\r\n### extra\r\n",
			covered: []string{"create", "read one", "claim"},
		},
		{
			name:    "gapped adapter",
			adapter: "## Operations\n\n### create\n\n## Notes\n\n### claim\n\n### Read One\n",
			covered: []string{"create"},
			missing: []string{"read one", "claim"},
		},
		{
			name:    "no operations section",
			adapter: "| op | command |\n| create | gh issue create |\n",
			missing: []string{"create", "read one", "claim"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			covered, missing := Coverage([]byte(tasksContract), []byte(c.adapter))
			if !slices.Equal(covered, c.covered) || !slices.Equal(missing, c.missing) {
				t.Fatalf("Coverage() = %q, %q; want %q, %q", covered, missing, c.covered, c.missing)
			}
		})
	}
}
