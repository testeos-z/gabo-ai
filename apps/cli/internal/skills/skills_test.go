package skills

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var want = []string{
	"gabo-agent-host-railway-orca", "gabo-agent-templates", "gabo-assistant", "gabo-coder", "gabo-contract-agent-host", "gabo-contract-direct-sessions",
	"gabo-contract-memory", "gabo-contract-methodology", "gabo-contract-notifications",
	"gabo-contract-repository-host", "gabo-contract-reviewer-placement", "gabo-contract-tasks", "gabo-cycle", "gabo-db-master", "gabo-dev-lead",
	"gabo-direct-sessions-orca", "gabo-doc-guide", "gabo-frontend", "gabo-pm", "gabo-reviewer-notes", "gabo-reviewer-placement-local", "gabo-setup",
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
			// T016: the methodology comes from the slot; the TRD 9.2 outcome and the round limit stay.
			"`gabo-methodology`", "`Closes #<n>`", "summary", "resources created", "test databases",
			"fix rounds used", "Max 4 fix rounds",
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
		// T055: the doc guide maps every product doc, the setup prompt and the catalog.
		"gabo-doc-guide/SKILL.md": {
			"docs/product/PRD.md", "docs/product/stories/", "docs/product/TRD.md", "docs/product/IMPLEMENTATION_PLAN.md",
			"docs/product/DB_SCHEMA.md", "docs/product/UX_UI.md", "docs/product/APP_FLOW.md", "DESIGN.md",
			"setup prompt", ".gbai/skills/", "adds no rule",
		},
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

// The coder skill hardcodes no methodology (T016): the former ODD and
// gentle-ai text is gone and lives only in the `methodology` slot adapter.
func TestCoderSkillHasNoHardcodedMethodology(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-coder/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"ODD", "Test-first", "test-first", "gentle-ai", "receipt-driven", "RDD",
		"Work-unit commits", "Conventional Commits", "no AI attribution",
	} {
		if strings.Contains(string(b), s) {
			t.Errorf("gabo-coder/SKILL.md: must not contain %q", s)
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

// The shipped direct_sessions adapter (T060) covers every contract operation
// with the Orca CLI, and the retired gbai-orca-inbox skill is named nowhere in
// the catalog.
func TestDirectSessionsOrcaAdapter(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var adapter string
	for _, f := range files {
		if f.Path == "gabo-direct-sessions-orca/SKILL.md" {
			adapter = string(f.Data)
		}
		if strings.Contains(string(f.Data), "orca-inbox") {
			t.Errorf("%s: references the removed orca-inbox skill", f.Path)
		}
	}
	if got, want := operations(adapter), contracts["gabo-contract-direct-sessions"].ops; !slices.Equal(got, want) {
		t.Errorf("gabo-direct-sessions-orca/SKILL.md: operations %q, want %q", got, want)
	}
	for _, s := range []string{
		"orca terminal create", "--worktree path:<worktree>", "--title", "orca terminal send",
		"--retry-request", "orca terminal show", "orca terminal read", "orca terminal close",
		"orca terminal list", "never merge", "never force-push", "ledger",
	} {
		if !strings.Contains(adapter, s) {
			t.Errorf("gabo-direct-sessions-orca/SKILL.md: missing %q", s)
		}
	}
}

// The shipped reviewer_placement adapter (T057) covers every contract
// operation for the local single-user placement with the installed user-level
// reviewer scripts, and names isolated mode as another adapter.
func TestReviewerPlacementLocalAdapter(t *testing.T) {
	t.Parallel()
	data, err := content.ReadFile("gabo-reviewer-placement-local/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	adapter := string(data)
	if got, want := operations(adapter), contracts["gabo-contract-reviewer-placement"].ops; !slices.Equal(got, want) {
		t.Errorf("gabo-reviewer-placement-local/SKILL.md: operations %q, want %q", got, want)
	}
	for _, s := range []string{
		`bash "${XDG_DATA_HOME:-$HOME/.local/share}/gbai/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha>`,
		"PR #<n> head <40-hex sha>", "placement=local mode=single-user", "PREFLIGHT-FAILED",
		"VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>",
		"gbai/reviewer", "isolated", "another adapter", "merges itself", "never posts the reviewer marker",
	} {
		if !strings.Contains(adapter, s) {
			t.Errorf("gabo-reviewer-placement-local/SKILL.md: missing %q", s)
		}
	}
}

// The shipped agent_host adapter (T059) covers every contract operation for
// agents on a Railway server driven through Orca, holds the former cloud
// fields, and names secrets only by env var.
func TestAgentHostRailwayOrcaAdapter(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var adapter string
	for _, f := range files {
		if f.Path == "gabo-agent-host-railway-orca/SKILL.md" {
			adapter = string(f.Data)
		}
	}
	if got, want := operations(adapter), contracts["gabo-contract-agent-host"].ops; !slices.Equal(got, want) {
		t.Errorf("gabo-agent-host-railway-orca/SKILL.md: operations %q, want %q", got, want)
	}
	for _, s := range []string{
		"server.railway", "users.orca", "bot.login", "prompt_file", "sessions.local_orca", "/etc/gbai/gbai.yaml",
		"RAILWAY_TOKEN", "railway status", "Railway MCP", "orca terminal create", "--environment",
		"orca terminal close", "orca terminal list", "single-user", "isolated",
	} {
		if !strings.Contains(adapter, s) {
			t.Errorf("gabo-agent-host-railway-orca/SKILL.md: missing %q", s)
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

// reviewerFiles are the files of the embedded gabo-reviewer skill (T040): the
// scripts live in its scripts/ folder, next to its SKILL.md, with their tests.
var reviewerFiles = []string{
	"gabo-reviewer/README.md",
	"gabo-reviewer/SKILL.md",
	"gabo-reviewer/scripts/common.sh",
	"gabo-reviewer/scripts/config.env",
	"gabo-reviewer/scripts/merge-head.sh",
	"gabo-reviewer/scripts/review-head.sh",
	"gabo-reviewer/scripts/reviewer.sh",
	"gabo-reviewer/scripts/tests/lib.sh",
	"gabo-reviewer/scripts/tests/merge-head.test.sh",
	"gabo-reviewer/scripts/tests/preflight.test.sh",
	"gabo-reviewer/scripts/tests/review.test.sh",
	"gabo-reviewer/scripts/tests/stubs.sh",
}

func TestReviewerSkillShipsItsScripts(t *testing.T) {
	t.Parallel()
	files, err := ReviewerFiles()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Path)
		// The repository comes from gbai.yaml; only the TRD 9.4 module path
		// of the upgrade command may name the old repository.
		rest := strings.ReplaceAll(string(f.Data), "github.com/testeos-z/gbai-team/apps/cli/cmd/gbai", "")
		if strings.Contains(rest, "testeos-z/gbai-team") {
			t.Errorf("%s: hardcodes the repository testeos-z/gbai-team", f.Path)
		}
		if f.Path == "gabo-reviewer/SKILL.md" {
			fm, ok := frontmatter(string(f.Data))
			d := fm["description"]
			if !ok || fm["name"] != "gabo-reviewer" || !strings.HasPrefix(d, `"Trigger: `) || len(d) > 252 {
				t.Errorf("%s: bad frontmatter %v", f.Path, fm)
			}
		}
	}
	if !slices.Equal(got, reviewerFiles) {
		t.Fatalf("gabo-reviewer files = %v\nwant %v", got, reviewerFiles)
	}
}

// The catalog that init installs never carries the reviewer skill: it is
// installed only at user level (T041), so PR code cannot change it.
func TestCatalogLeavesOutTheReviewerSkill(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Path, "gabo-reviewer/") {
			t.Errorf("catalog holds %s", f.Path)
		}
	}
	if slices.Contains(Names(), "gabo-reviewer") {
		t.Error("Names() lists gabo-reviewer")
	}
}

// The shell tests of the reviewer scripts run against fake gh, claude, go and
// gbai and a local bare repository, so they run with the Go suite (and in
// review.checks) wherever their tools exist.
func TestReviewerScriptsShellTests(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	for _, tool := range []string{"bash", "git", "jq", "flock", "setsid"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found: %v", tool, err)
		}
	}
	tests, err := filepath.Glob("gabo-reviewer/scripts/tests/*.test.sh")
	if err != nil || len(tests) != 3 {
		t.Fatalf("shell tests = %v (%v), want 3", tests, err)
	}
	for _, p := range tests {
		t.Run(filepath.Base(p), func(t *testing.T) {
			t.Parallel()
			out, err := exec.Command("bash", p).CombinedOutput()
			if err != nil {
				t.Fatalf("bash %s: %v\n%s", p, err, out)
			}
		})
	}
}

// The neutral agent spec ships with the catalog, so the project copy under
// .gbai/skills/ holds it too (TRD 8.1, D24).
func TestCatalogShipsTheAgentSpec(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	if !slices.Contains(paths, "gabo-agent-templates/agents.yaml") {
		t.Fatalf("Files() misses gabo-agent-templates/agents.yaml (has %v)", paths)
	}
}
