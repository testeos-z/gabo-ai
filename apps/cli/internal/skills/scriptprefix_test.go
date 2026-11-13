package skills

import (
	"regexp"
	"strings"
	"testing"
)

// gabo installs catalog files with mode 0644 (osfs), so a skill that tells an
// agent to run a script by its bare path fails with "Permission denied". Every
// invocation of a `.gabo/skills/<name>/scripts/*.sh` file must be prefixed with
// `sh ` or `bash `.
func TestSkillScriptInvocationsHavePrefix(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`\.gabo/skills/[\w-]+/scripts/[\w-]+\.sh`)
	seen := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Path, "SKILL.md") {
			continue
		}
		text := string(f.Data)
		for _, m := range re.FindAllStringIndex(text, -1) {
			seen++
			before := text[:m[0]]
			before = strings.TrimSuffix(before, `"`)
			if !strings.HasSuffix(before, "sh ") && !strings.HasSuffix(before, "bash ") {
				line := strings.Count(text[:m[0]], "\n") + 1
				t.Errorf("%s:%d: %s must be invoked as `sh <path>` or `bash <path>`: catalog files are installed without the execute bit", f.Path, line, text[m[0]:m[1]])
			}
		}
	}
	if seen == 0 {
		t.Fatal("no script invocation found in any SKILL.md; the test would pass vacuously")
	}
}
