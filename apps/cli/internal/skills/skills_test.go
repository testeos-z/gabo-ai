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
	"gabo-adapter-writer", "gabo-agent-host-local", "gabo-agent-host-railway-orca", "gabo-agent-templates", "gabo-assistant", "gabo-coder", "gabo-config-writer", "gabo-contract-agent-host", "gabo-contract-direct-sessions",
	"gabo-contract-memory", "gabo-contract-methodology", "gabo-contract-notifications",
	"gabo-contract-repository-host", "gabo-contract-reviewer-placement", "gabo-contract-tasks", "gabo-cycle", "gabo-db-master", "gabo-dev-lead",
	"gabo-direct-sessions-orca", "gabo-doc-guide", "gabo-frontend", "gabo-plan-reviewer", "gabo-pm", "gabo-reviewer-notes", "gabo-reviewer-placement-local", "gabo-runtime-builder", "gabo-runtime-claude", "gabo-runtime-pi", "gabo-setup",
	"gabo-wave-report",
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
		// T056: J1 steps 5 and 6 and J7: co-write an adapter from its contract,
		// never invent a command, record it through the config writer, then check.
		"gabo-adapter-writer/SKILL.md": {
			"J1 steps 5 and 6", "J7", "`gabo-contract-<slot>`", "`## Operations`",
			"one `### <operation>` heading per contract operation", "spelled exactly",
			"fenced code block", "exact commands", "Never invent a command",
			"one question at a time", "R12", "auto mode", "save decision", "R21", "env var name",
			"`.gabo/adapters/<slot>-<tool>.md`", "never under `.gabo/skills/`",
			"`gabo-config-writer`", "`slots.<slot>.adapter`", "`gabo check`",
			"does not cover", "rerun `gabo check`", "until it passes", "unverified",
		},
		// T051: J1 step 7 and J4 step 5; replaces `gabo config set` (T015).
		"gabo-config-writer/SKILL.md": {
			"personal keys", "global user file", "`gabo.yaml`", "`alias`, `interaction_level`, `tone`, `language`",
			"Keep every comment", "`# yaml-language-server: $schema=", "R12", "auto mode", "save decision",
			"R21", "env var name", "refuse", "outside the schemas", "`gabo check`", "after each write",
			"`gabo config set`", "`additionalProperties: false`",
		},
		"gabo-cycle/SKILL.md": {
			"ledger", "worktree.keep", "Backlog", "Ready", "parallelism", "REFUSED",
			"handle", "fallback sweep", "in-process", "D23", "never writes DONE",
			// T024: J3, J8, J10 and J11 through the skill alone (no gabo command, no state dir).
			"blocked-by links", "before each coder round", "rule 1", "rule 2", "rule 3",
			"4 fix rounds", "Unverified writes", "test databases", "no `gabo` command",
			"no state dir", "D15", "D17", "skips DONE", "reattach", "disagreement",
			"never guess", "TRD 9.2", "save the wave report", "`gabo/trace/<wave>`",
			"`gabo/ledger/<wave>`",
			// T061: the report format lives in gabo-wave-report.
			"`gabo-wave-report`",
		},
		// T061: J3 wave report (TRD 9.2) formatted once and saved through memory.
		"gabo-wave-report/SKILL.md": {
			"TRD 9.2", "J3", "status", "PR", "last verdict", "fix rounds", "coder handle",
			"answer and resolver", "R5", "interaction count", "reasons", "R29",
			"Unverified writes", "R24", "leftovers", "test databases", "merged tasks still open", "R13",
			"plan-change proposals", "denied board moves", "resume disagreements", "comfort answer", "R17",
			"`memory`", "save wave trace", "last entry of `gabo/trace/<wave>`", "read it back", "unverified",
			"`Wave <n>: <done> done, <blocked> blocked`", "UX 1.5", "none",
		},
		// J9 and R16/R18: spokesperson, identity resolution and terminal rules; D23 task close.
		"gabo-assistant/SKILL.md": {
			"default point of contact", "Never block", "only the owner", "user.email",
			"`developers`", "global `alias` wins", "Ask once", "No `gabo` command",
			"`tone`", "`language`", "terminal",
			"resume the owning coder", "PR #<n> merged", "D23", "close report",
			// T026: J3 and J4 wave start and report, modes, decision records, comfort
			// question and the personalization layers (D22).
			"## Wave Start", "launch `gabo-cycle`", "## Wave Close", "UX 1.5",
			"`Wave <n>: <done> done, <blocked> blocked`", "answer and resolver",
			"one question at a time", "`interaction_level`", "`low`", "`balanced`", "`high`",
			"never prompt", "save decision", "`what`, `question`, `choice`, `reason`, `mode`, `wave`, `task`",
			"list decisions", "filter", "critical question", "directiva",
			"comfort question", "fewer, about right, or more", "`gabo check`",
			"## Personalization", "stable layer", "learning layer", "`user/preferences`",
			"global scope", "project scope", "about 3 times", "read preferences", "save preference",
			"secrets", "sensitive", "setup start", "time since setup start", "first wave report",
			// The setup start is saved only once a memory slot exists; the wave start survives a restart.
			"after `gabo-setup` verifies the `memory` slot", "save wave trace",
			// T061: the summary and the comfort answer follow gabo-wave-report.
			"`gabo-wave-report`",
			// T033: J5 intervene in a task: mediate by default, offer a direct session
			// once after three mediated exchanges (APP_FLOW A5), open it through the
			// `direct_sessions` slot, resume from the traces; R14, R18, D9, D18.
			"## Intervene in a Task (J5)", "mediate by default", "three mediated exchanges",
			"offer a direct session once", "APP_FLOW A5", "`direct_sessions` slot",
			"`gabo-direct-sessions-orca`", "open session", "`gabo-<task id>`", "context brief",
			"no `gabo` command", "Orca is unavailable", "keep mediating", "say why",
			"commits, task and memory", "without redoing work", "close session",
			"only your own terminal", "only on the owner's request", "R14", "R18", "D9",
		},
		// T032: the one plan-change reviewer skill (D8, D24) and the directiva flow (D7, J6).
		"gabo-plan-reviewer/SKILL.md": {
			"## Approval Scale", "## Plan-Change Flow", "## Review",
			"`plan_change.scope`", "`plan_change.global_without_owner`",
			"One issue and one PR per proposal", "only `docs/product` and tasks",
			"only when the owner is absent", "never for a proposal debated with the owner",
			"fresh", "clean context", "index the repository", "never edit",
			"`AGREE`", "`COUNTER-PROPOSAL`", "BLOCKED", "R19", "D7", "D8",
			"never applies automatically", "save decision",
		},
		"gabo-pm/SKILL.md":                       {"`gabo-plan-reviewer`", "## Plan-Change Flow"},
		"gabo-dev-lead/SKILL.md":                 {"`gabo-plan-reviewer`", "## Plan-Change Flow"},
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
			"setup prompt", ".gabo/skills/", "adds no rule", "`gabo-plan-reviewer`",
		},
		// The former cloud keys of the core config (DB_SCHEMA 2.3).
		"gabo-contract-agent-host/SKILL.md": {
			"server.railway", "users.orca", "bot.login", "sessions.{coder, reviewer, assistant}.prompt_file",
			"sessions.local_orca", "/etc/gabo/gabo.yaml",
		},
		"gabo-contract-agent-host/examples/agent-host-railway.md": {
			"server.railway", "users.orca", "bot.login", "prompt_file", "sessions.local_orca", "/etc/gabo/gabo.yaml",
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
// with the Orca CLI, and the retired gabo-orca-inbox skill is named nowhere in
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
		`bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha>`,
		"PR #<n> head <40-hex sha>", "placement=local mode=single-user", "PREFLIGHT-FAILED",
		"VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>",
		"gabo/reviewer", "isolated", "another adapter", "merges itself", "never posts the reviewer marker",
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
		"server.railway", "users.orca", "bot.login", "prompt_file", "sessions.local_orca", "/etc/gabo/gabo.yaml",
		"RAILWAY_TOKEN", "railway status", "Railway MCP", "orca terminal create", "--environment",
		"orca terminal close", "orca terminal list", "single-user", "isolated",
	} {
		if !strings.Contains(adapter, s) {
			t.Errorf("gabo-agent-host-railway-orca/SKILL.md: missing %q", s)
		}
	}
}

// The shipped local agent_host adapter (T058) covers every contract operation
// for agents on the owner's machine and states every server field as not used.
func TestAgentHostLocalAdapter(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var adapter string
	for _, f := range files {
		if f.Path == "gabo-agent-host-local/SKILL.md" {
			adapter = string(f.Data)
		}
	}
	if got, want := operations(adapter), contracts["gabo-contract-agent-host"].ops; !slices.Equal(got, want) {
		t.Errorf("gabo-agent-host-local/SKILL.md: operations %q, want %q", got, want)
	}
	for _, s := range []string{
		"gabo-contract-agent-host", "`not used`", "single-user", "ledger",
		// claude and pi command lines, each with its session id and the prompt kept off argv.
		// Only the agent command is backgrounded, so $! is the agent, not a subshell.
		`{ nohup claude --print --agent <role agent> --session-id "$sid" --permission-mode dontAsk --permission-prompts none --allowedTools "$tools" <"$dir/prompt.md"`,
		`{ nohup pi --print --approve --tools read,bash,edit,write,grep,find,ls --session-id "$sid" --append-system-prompt <role agent file> @"$dir/prompt.md"`,
		"</dev/null",
		// a headless session cannot answer a permission prompt: the tools it may use are
		// pinned on the command line and checked before the session starts.
		// the tools line is read from the frontmatter only, and exactly once.
		`tools=$(awk '/^---$/ { if (++n == 2) exit; next } n == 1 && sub(/^tools: */, "") { print; c++ } END { exit c != 1 }' <role agent file>)`,
		// a reviewer would load settings, hooks and agents from the branch under review:
		// only coder and assistant sessions start, on either runtime, and the role check runs first.
		"case \"<runtime role>\" in coder|assistant) ;; *) false ;; esac &&\n  tools=$(awk",
		"case \"<runtime role>\" in coder|assistant) ;; *) false ;; esac &&\n  pi --help",
		"Never start a `reviewer` session",
		// pi must support every flag the start command relies on.
		`pi --help | grep -qF -- --session-id && pi --help | grep -qF -- --append-system-prompt`,
		// pi reads the role agent file as a file: its help states the flag takes text or file contents.
		"Append text or file contents to the system prompt",
		// each role has a defined directory, and a refused tools line names the format it needs.
		"An `assistant` session starts in the repository's main checkout",
		"one-line `tools: A, B, C`",
		// the scratch dir comes from the ledger, not a shell variable of the start step.
		"rm -r -- <scratch dir>",
		// the report reads the elapsed time only after the session id matched.
		`ps -o args= -p <pid> | grep -qF -- "--session-id <sid>" && ps -o etime= -p <pid>`,
		`claude --help | grep -qF -- --permission-prompts`,
		`pi --help | grep -qF -- --approve`,
		// the start check and the report never print the process arguments.
		`ps -o args= -p "$(cat "$dir/pid")" | grep -qF -- "--session-id $sid"`,
		// a reused process id is never signalled: the session id is matched before kill.
		`grep -qF -- "--session-id <sid>" && kill -TERM <pid>`,
		// every former cloud field of the contract is stated as not used.
		"| `server.railway.{project_id, environment_id, service_id}` | `not used` |",
		"| `users.orca.{name, home, uid, extra_path}` | `not used` |",
		"| `bot.login` | `not used` |",
		"| `sessions.{coder, reviewer, assistant}.prompt_file` | `not used` |",
		"| `sessions.local_orca` | `not used` |",
		"| `/etc/gabo/gabo.yaml` | `not used` |",
	} {
		if !strings.Contains(adapter, s) {
			t.Errorf("gabo-agent-host-local/SKILL.md: missing %q", s)
		}
	}
	if strings.Contains(adapter, "| grep -F --") {
		t.Errorf("gabo-agent-host-local/SKILL.md: a ps check prints the process arguments; use grep -qF")
	}
	for _, s := range []string{"railway status", "railway mcp", "--environment", "ssh ", "orca "} {
		if strings.Contains(strings.ToLower(adapter), s) {
			t.Errorf("gabo-agent-host-local/SKILL.md: reaches a server or remote host %q", s)
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
		// The repository comes from gabo.yaml; only the TRD 9.4 module path
		// of the upgrade command may name the repository.
		rest := strings.ReplaceAll(string(f.Data), "github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo", "")
		if strings.Contains(rest, "testeos-z/gabo-ai") {
			t.Errorf("%s: hardcodes the repository testeos-z/gabo-ai", f.Path)
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
// gabo and a local bare repository, so they run with the Go suite (and in
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
// .gabo/skills/ holds it too (TRD 8.1, D24).
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

// T061: the wave report format lives only in gabo-wave-report; the cycle and
// the assistant reference it instead of repeating its fields, and the cycle
// agent loads it.
func TestWaveReportFormatLivesInOneSkill(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	repeated := []string{"denied board moves", "merged tasks still open", "interaction count with reasons", "plan-change proposals"}
	for _, f := range files {
		switch f.Path {
		case "gabo-cycle/SKILL.md", "gabo-assistant/SKILL.md":
			for _, s := range repeated {
				if strings.Contains(string(f.Data), s) {
					t.Errorf("%s repeats the wave report field %q; reference `gabo-wave-report` instead", f.Path, s)
				}
			}
		case "gabo-agent-templates/agents.yaml":
			if !strings.Contains(string(f.Data), "skills: [gabo-cycle, gabo-wave-report]") {
				t.Errorf("%s: the gabo-cycle agent must load gabo-wave-report", f.Path)
			}
		}
	}
}

// The skill carries the APP_FLOW plan-change approval scale (TRD D7) row for
// row, so directiva applies the same scale the product docs define.
func TestPlanReviewerCarriesTheAppFlowApprovalScale(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-plan-reviewer/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	// APP_FLOW lives in the repository checkout, three levels above this
	// package's module dir.
	doc, err := os.ReadFile("../../../../docs/product/APP_FLOW.md")
	if err != nil {
		t.Fatal(err)
	}
	want := scaleRows(t, string(doc), "## Plan-change approval scale")
	got := scaleRows(t, string(b), "## Approval Scale")
	if !slices.Equal(got, want) {
		t.Fatalf("approval scale rows\n got %q\nwant %q", got, want)
	}
	if len(got) != 6 {
		t.Fatalf("approval scale has %d table lines, want header, rule and 4 scopes", len(got))
	}
	for i, scope := range []string{"One task", "Group of tasks", "Global plan", "Any scope beyond `plan_change.scope`"} {
		if !strings.HasPrefix(got[i+2], "| "+scope) {
			t.Errorf("row %d = %q, want scope %q", i+1, got[i+2], scope)
		}
	}
}

// scaleRows returns the first Markdown table under heading.
func scaleRows(t *testing.T, doc, heading string) []string {
	t.Helper()
	_, rest, ok := strings.Cut(doc, "\n"+heading+"\n")
	if !ok {
		t.Fatalf("heading %q not found", heading)
	}
	var rows []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "|") {
			rows = append(rows, line)
		} else if len(rows) > 0 || strings.HasPrefix(line, "## ") {
			break
		}
	}
	return rows
}
