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
	"gbai-assistant", "gbai-coder", "gbai-contract-repository-host", "gbai-contract-reviewer-placement",
	"gbai-contract-tasks", "gbai-cycle", "gbai-db-master", "gbai-dev-lead",
	"gbai-frontend", "gbai-orca-inbox", "gbai-pm", "gbai-reviewer-notes", "gbai-setup",
}

// contracts maps each contract skill to its TRD 9.1 operations, in order, and
// to the example adapters that must cover every one of them.
var contracts = map[string]struct {
	ops      []string
	examples []string
}{
	"gbai-contract-tasks": {
		ops: []string{
			"create", "read unblocked", "read one", "claim", "log progress",
			"change status", "link PR", "verify by re-read",
		},
		examples: []string{"tasks-gh.md"},
	},
	"gbai-contract-repository-host": {
		ops: []string{
			"open PR", "comment on PR", "read PR status and checks",
			"merge into integration branch", "delete branch",
		},
		examples: []string{"host-github.md"},
	},
	"gbai-contract-reviewer-placement": {
		ops:      []string{"trigger review", "report placement", "return verdict"},
		examples: []string{"reviewer-local.md"},
	},
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

// The process rules other tools and reports rely on stay in the skill text.
func TestSkillsCarryTheProcessRules(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"gbai-coder/SKILL.md": {
			"leftovers:", "containers:", "mktemp -d", "--ignored=matching", "immutable",
			"## Task Close", "PR #<n> merged", "repository_host", "never `--force`", "--git-common-dir", "resumable",
		},
		"gbai-cycle/SKILL.md": {
			"ledger", "worktree.keep", "Backlog", "Ready", "parallelism", "REFUSED",
			"handle", "resume the owning coder", "fallback sweep", "in-process",
		},
		"gbai-assistant/SKILL.md": {"resume the owning coder", "PR #<n> merged"},
		"gbai-reviewer-notes/SKILL.md": {
			"never approves", "marker", "## Review layers", "Coder-side review (optional)",
			"methodology", "Reviewer-side gate (mandatory)", "outside every repository",
			"own clone", "own `HOME`", "cache", "clean environment", "Single-user mode", "Isolated mode",
			"sha=<head>", "One verdict per pushed head", "`VERDICT`", "Later",
		},
	}
	for _, f := range files {
		for _, s := range want[f.Path] {
			if !strings.Contains(string(f.Data), s) {
				t.Errorf("%s: missing %q", f.Path, s)
			}
		}
	}
}

// Every contract lists its operations as stable "### <op>" headings under
// "## Operations", and every example adapter repeats exactly those headings.
func TestContractsListOperationsAndExamplesCoverThem(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range Names() {
		if _, ok := contracts[n]; strings.HasPrefix(n, "gbai-contract-") && !ok {
			t.Errorf("%s: contract skill not listed in contracts", n)
		}
	}
	byPath := map[string]string{}
	for _, f := range files {
		byPath[f.Path] = string(f.Data)
	}
	for name, c := range contracts {
		skill, ok := byPath[name+"/SKILL.md"]
		if !ok {
			t.Errorf("%s: contract skill missing", name)
			continue
		}
		if got := operations(skill); !slices.Equal(got, c.ops) {
			t.Errorf("%s/SKILL.md: operations %q, want %q", name, got, c.ops)
		}
		for _, ex := range c.examples {
			p := name + "/examples/" + ex
			doc, ok := byPath[p]
			if !ok {
				t.Errorf("%s: example adapter missing", p)
				continue
			}
			if got := operations(doc); !slices.Equal(got, c.ops) {
				t.Errorf("%s: operations %q, want %q", p, got, c.ops)
			}
		}
	}
}

// operations returns the "### " headings inside the "## Operations" section.
func operations(doc string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(doc, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			in = strings.TrimSpace(line) == "## Operations"
		case in && strings.HasPrefix(line, "### "):
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "### ")))
		}
	}
	return out
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
