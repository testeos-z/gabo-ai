package skills

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

var want = []string{
	"gabo-assistant", "gabo-coder", "gabo-contract-agent-host", "gabo-contract-direct-sessions",
	"gabo-contract-memory", "gabo-contract-methodology", "gabo-contract-notifications",
	"gabo-contract-repository-host", "gabo-contract-reviewer-placement", "gabo-contract-tasks", "gabo-cycle", "gabo-db-master", "gabo-dev-lead",
	"gabo-frontend", "gabo-pm", "gabo-reviewer-notes", "gabo-setup", "gbai-orca-inbox",
}

// contracts maps each contract skill to its TRD 9.1 operations, in order, and
// to the example adapters that must cover every one of them.
var contracts = map[string]struct {
	ops      []string
	examples []string
}{
	"gabo-contract-tasks": {
		ops: []string{
			"create", "read unblocked", "read one", "claim", "log progress",
			"change status", "link PR", "verify by re-read",
		},
		examples: []string{"tasks-gh.md"},
	},
	"gabo-contract-repository-host": {
		ops: []string{
			"open PR", "comment on PR", "read PR status and checks",
			"merge into integration branch", "delete branch",
		},
		examples: []string{"host-github.md"},
	},
	"gabo-contract-reviewer-placement": {
		ops:      []string{"trigger review", "report placement", "return verdict"},
		examples: []string{"reviewer-local.md"},
	},
	"gabo-contract-memory": {
		ops: []string{
			"save decision", "list decisions", "search", "read one",
			"save ledger", "read ledger", "save wave trace", "read wave trace",
			"save preference", "read preferences",
		},
		examples: []string{"memory-engram.md"},
	},
	"gabo-contract-notifications": {
		ops:      []string{"notify owner", "notify developer by alias"},
		examples: []string{"notify-task-comment.md"},
	},
	"gabo-contract-direct-sessions": {
		ops:      []string{"open session", "send message", "read state", "close session"},
		examples: []string{"sessions-orca.md"},
	},
	"gabo-contract-methodology": {
		ops:      []string{"coder instructions", "coder-side review", "report per task"},
		examples: []string{"methodology-tdd.md"},
	},
	"gabo-contract-agent-host": {
		ops:      []string{"start agent session", "stop agent session", "report where agents run"},
		examples: []string{"agent-host-railway.md"},
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
		"gabo-coder/SKILL.md": {
			"leftovers:", "containers:", "mktemp -d", "--ignored=matching", "immutable",
			"## Task Close", "PR #<n> merged", "repository_host", "never `--force`", "--git-common-dir", "resumable",
			"`worktree.keep`", "before any removal", "unverified:",
			"D23", "only writer of DONE", "lock error", "retry once",
		},
		"gabo-cycle/SKILL.md": {
			"ledger", "worktree.keep", "Backlog", "Ready", "parallelism", "REFUSED",
			"handle", "fallback sweep", "in-process", "D23", "never writes DONE",
		},
		// J9 and R16/R18: spokesperson, identity resolution and terminal rules; D23 task close.
		"gabo-assistant/SKILL.md": {
			"default point of contact", "Never block", "only the owner", "user.email",
			"`developers`", "global `alias` wins", "Ask once", "No `gbai` command",
			"`tone`", "`language`", "terminal",
			"resume the owning coder", "PR #<n> merged", "D23", "close report",
		},
		"gabo-contract-repository-host/SKILL.md": {"without force", "merged head", "leftover"},
		"gabo-contract-repository-host/examples/host-github.md": {
			"baseRefName,headRefName,headRefOid", "git branch -d <branch>", "ls-remote", "leftover",
			"local `origin/<branch>` tracking ref", "--prune",
		},
		"gabo-reviewer-notes/SKILL.md": {
			"never approves", "marker", "## Review layers", "Coder-side review (optional)",
			"methodology", "Reviewer-side gate (mandatory)", "outside every repository",
			"own clone", "own `HOME`", "cache", "clean environment", "Single-user mode", "Isolated mode",
			"sha=<head>", "One verdict per pushed head", "`VERDICT`", "Later",
		},
		// TRD 9.2 auto decision record fields and the list filters (R6, R30).
		"gabo-contract-memory/SKILL.md": {
			"`what`", "`question`", "`choice`", "`reason`", "`mode`", "`wave`", "`task`", "`resolver`",
			"filter by `mode`, `wave` and `task`", "user/preferences",
		},
		"gabo-contract-methodology/SKILL.md": {"coder-side review", "RDD", "gabo-methodology"},
		// The former cloud keys of the core config (DB_SCHEMA 2.3).
		"gabo-contract-agent-host/SKILL.md": {
			"server.railway", "users.orca", "bot.login", "sessions.{coder, reviewer, assistant}.prompt_file",
			"sessions.local_orca", "/etc/gbai/gbai.yaml",
		},
		"gabo-contract-agent-host/examples/agent-host-railway.md": {
			"server.railway", "users.orca", "bot.login", "prompt_file", "sessions.local_orca", "/etc/gbai/gbai.yaml",
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

// The task close never force-deletes a branch: the task branch goes with
// `git branch -d` while its upstream still exists.
func TestTaskCloseNeverForceDeletesBranches(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	guarded := []string{
		"gabo-coder/SKILL.md", "gabo-cycle/SKILL.md",
		"gabo-contract-repository-host/SKILL.md", "gabo-contract-repository-host/examples/host-github.md",
	}
	seen := 0
	for _, f := range files {
		if !slices.Contains(guarded, f.Path) {
			continue
		}
		seen++
		if strings.Contains(string(f.Data), "branch -D") {
			t.Errorf("%s: must not contain %q", f.Path, "branch -D")
		}
	}
	if seen != len(guarded) {
		t.Fatalf("checked %d files, want %d", seen, len(guarded))
	}
	// The Claude Code adapter and agent are not embedded; they live in the
	// repository checkout, three levels above this package's module dir.
	for _, p := range []string{
		"../../../../.claude/skills/gabo-repository-host-github/SKILL.md",
		"../../../../.claude/agents/gabo-coder.md",
	} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if strings.Contains(string(b), "branch -D") {
			t.Errorf("%s: must not contain %q", p, "branch -D")
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
		if _, ok := contracts[n]; strings.HasPrefix(n, "gabo-contract-") && !ok {
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
