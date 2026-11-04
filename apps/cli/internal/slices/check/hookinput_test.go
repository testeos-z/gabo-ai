package check

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

func TestPreWriteTargets(t *testing.T) {
	t.Parallel()
	patch := "*** Begin Patch\n" +
		"*** Add File: docs/product/intents/0009-x/plan.md\n+a\n" +
		"*** Update File: README.md\r\n@@\n-a\n+b\n" +
		"*** Update File: old.md\n*** Move to: docs/product/intents/0010-y/plan.md\n@@\n" +
		"*** Delete File: gone.md\n" +
		"*** End Patch\n"
	tests := []struct {
		name, in string
		want     []string
	}{
		{name: "Claude Code file_path", in: `{"tool_name":"Edit","tool_input":{"file_path":"/p/plan.md","old_string":"a"}}`, want: []string{"/p/plan.md"}},
		{name: "Codex command string", in: `{"tool_name":"apply_patch","tool_input":{"command":` + quote(patch) + `}}`, want: []string{"docs/product/intents/0009-x/plan.md", "README.md", "old.md", "docs/product/intents/0010-y/plan.md"}},
		{name: "Codex command argv", in: `{"tool_input":{"command":["apply_patch",` + quote(patch) + `]}}`, want: []string{"docs/product/intents/0009-x/plan.md", "README.md", "old.md", "docs/product/intents/0010-y/plan.md"}},
		{name: "patch without a written file", in: `{"tool_input":{"command":"*** Begin Patch\n*** Delete File: x\n*** End Patch"}}`, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := PreWriteTargets([]byte(tt.in))
			if err != nil || !slices.Equal(got, tt.want) {
				t.Fatalf("PreWriteTargets = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestPreWriteTargetsRejectsBadInput(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "  \n", "{", "null", "[]", `{}`, `{"tool_input":null}`, `{"tool_input":{}}`, `{"tool_input":{"file_path":""}}`, `{"tool_input":{"file_path":1}}`, `{"tool_input":{"command":1}}`, `{"tool_input":{"command":["a",1]}}`, `{} {}`} {
		if got, err := PreWriteTargets([]byte(in)); !errors.Is(err, ErrPreWriteUsage) {
			t.Fatalf("PreWriteTargets(%q) = %q, %v; want ErrPreWriteUsage", in, got, err)
		}
	}
}

func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
