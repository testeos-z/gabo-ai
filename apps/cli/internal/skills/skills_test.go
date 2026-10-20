package skills

import (
	"archive/tar"
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
)

var want = []string{
	"gbai-assistant", "gbai-coder", "gbai-cycle", "gbai-db-master", "gbai-dev-lead",
	"gbai-frontend", "gbai-orca-inbox", "gbai-pm", "gbai-reviewer-notes", "gbai-setup",
}

func TestEmbeddedSkillsAreListed(t *testing.T) {
	t.Parallel()
	if got := Names(); !slices.Equal(got, want) {
		t.Fatalf("Names() = %v\nwant     %v", got, want)
	}
}

func TestEverySkillHasValidFrontmatter(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range files {
		dir, base, _ := strings.Cut(f.Path, "/")
		if base != "SKILL.md" {
			continue
		}
		fm, ok := frontmatter(string(f.Data))
		if !ok {
			t.Errorf("%s: no frontmatter", f.Path)
			continue
		}
		if fm["name"] != dir {
			t.Errorf("%s: name %q, want %q", f.Path, fm["name"], dir)
		}
		if seen[fm["name"]] {
			t.Errorf("%s: duplicate name", f.Path)
		}
		seen[fm["name"]] = true
		d := fm["description"]
		if !strings.HasPrefix(d, `"Trigger: `) || !strings.HasSuffix(d, `"`) || len(d) > 252 {
			t.Errorf("%s: description %q must be one quoted line starting with Trigger: and <= 250 chars", f.Path, d)
		}
		for _, k := range []string{"license", "metadata"} {
			if _, ok := fm[k]; !ok {
				t.Errorf("%s: missing %s", f.Path, k)
			}
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("validated %d skills, want %d", len(seen), len(want))
	}
}

func frontmatter(s string) (map[string]string, bool) {
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return nil, false
	}
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return nil, false
	}
	out := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		if strings.HasPrefix(line, " ") {
			continue
		}
		k, v, _ := strings.Cut(line, ":")
		out[k] = strings.TrimSpace(v)
	}
	return out, true
}

func TestTarHoldsEverySkillFile(t *testing.T) {
	t.Parallel()
	b, err := Tar()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			names = append(names, h.Name)
			if h.Mode != 0o644 || h.Uid != 0 {
				t.Errorf("%s: mode %o uid %d", h.Name, h.Mode, h.Uid)
			}
		}
	}
	for _, n := range want {
		if !slices.Contains(names, n+"/SKILL.md") {
			t.Errorf("tar misses %s/SKILL.md (has %v)", n, names)
		}
	}
}
