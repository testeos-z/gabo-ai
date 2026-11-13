//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// version is the release version the test binary is built with; gabo init
// stamps it on the sample and installs the reviewer skill at it.
const version = "0.9.0"

// templateSlotRE finds the slots of an agent template that name a fixed-name link.
var templateSlotRE = regexp.MustCompile(`\{\{(methodology|adapter:([A-Za-z0-9_-]+))\}\}`)

// claudeMode is what the fake review engine answers (stubs.sh: pass, fail, ...).
var claudeMode = "pass"

// world is the scratch environment of one mini wave: the gabo binary, a fake
// HOME holding the user-level reviewer skill, the sample repository's main
// checkout on dev, a bare repository standing in for GitHub, the fixture dir
// of the fake gh and the fake tracker and memory.
type world struct {
	t       *testing.T
	root    string
	bin     string
	home    string
	config  string // XDG_CONFIG_HOME for gabo check
	main    string
	origin  string
	stubs   string
	fixture string
	wts     string
	gitLog  []string // every git command the wave ran, to audit the ledger
	tasks   *tracker
	mem     *memory
}

func itoa(n int) string { return strconv.Itoa(n) }

func newWorld(t *testing.T) *world {
	t.Helper()
	for _, tool := range []string{"bash", "git", "jq", "flock", "setsid"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("the mini wave needs %s on PATH", tool)
		}
	}
	root := t.TempDir()
	w := &world{
		t: t, root: root,
		bin:     filepath.Join(root, "bin", "gabo"),
		home:    filepath.Join(root, "home"),
		config:  filepath.Join(root, "config"),
		main:    filepath.Join(root, "sample"),
		origin:  filepath.Join(root, "origin.git"),
		stubs:   filepath.Join(root, "stubs"),
		fixture: filepath.Join(root, "github"),
		wts:     filepath.Join(root, "worktrees"),
	}
	for _, d := range []string{w.home, w.config, w.main, w.fixture, w.wts} {
		w.mkdir(d)
	}
	w.tasks = &tracker{path: filepath.Join(root, "tracker.json"), drop: map[string]bool{}}
	w.mem = &memory{path: filepath.Join(root, "memory.json")}
	w.run("", nil, "go", "build", "-ldflags", "-X main.buildVersion="+version, "-o", w.bin, "../cmd/gabo")
	w.run("", nil, "bash", "-c", `source "$1/lib.sh" && source "$1/stubs.sh" && make_stubs "$2"`,
		"stubs", "../internal/skills/gabo-reviewer/scripts/tests", w.stubs)
	w.must(os.WriteFile(filepath.Join(w.fixture, "user_login"), []byte("owner\n"), 0o600))
	w.sample()
	return w
}

func (w *world) mkdir(d string) { w.must(os.MkdirAll(d, 0o755)) }

func (w *world) must(err error) {
	w.t.Helper()
	if err != nil {
		w.t.Fatal(err)
	}
}

// env is a fresh environment: nothing of the caller's tokens or REVIEWER_*
// settings reaches gabo, git or the reviewer scripts.
func (w *world) env(extra ...string) []string {
	return append([]string{
		"PATH=" + w.stubs + ":" + os.Getenv("PATH"),
		"HOME=" + w.home,
		"XDG_CONFIG_HOME=" + w.config,
		"TMPDIR=" + w.root,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(w.root, "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=coder", "GIT_AUTHOR_EMAIL=coder@example.com",
		"GIT_COMMITTER_NAME=coder", "GIT_COMMITTER_EMAIL=coder@example.com",
		"GOFLAGS=" + os.Getenv("GOFLAGS"),
		"GOCACHE=" + goEnv("GOCACHE"), "GOMODCACHE=" + goEnv("GOMODCACHE"),
	}, extra...)
}

func goEnv(key string) string {
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// run runs a command in dir (the package dir when empty) and fails the test
// on a non-zero exit; env nil means w.env().
func (w *world) run(dir string, env []string, name string, args ...string) string {
	w.t.Helper()
	out, err := w.try(dir, env, name, args...)
	if err != nil {
		w.t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return out
}

func (w *world) try(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if env == nil && name != "go" {
		env = w.env()
	}
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdin = nil // a prompt would read EOF and fail, never wait
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

// git runs git in dir and logs the command for the ledger audit.
func (w *world) git(dir string, args ...string) string {
	w.t.Helper()
	w.gitLog = append(w.gitLog, strings.Join(args, " "))
	return strings.TrimSpace(w.run(dir, nil, "git", append([]string{"-c", "commit.gpgsign=false"}, args...)...))
}

// link makes the symlink name -> target, keeping an existing one.
func (w *world) link(target, name string) {
	w.mkdir(filepath.Dir(name))
	if _, err := os.Lstat(name); err == nil {
		return
	}
	w.must(os.Symlink(target, name))
}

func (w *world) write(rel, data string) {
	p := filepath.Join(w.main, filepath.FromSlash(rel))
	w.mkdir(filepath.Dir(p))
	w.must(os.WriteFile(p, []byte(data), 0o644))
}

// sample builds the sample repository: gabo init, a gabo.yaml whose tasks and
// memory slots point at the fake adapters, a runtime skill and every agent of
// the spec; then the bare origin and two dependent tasks in the fake tracker.
func (w *world) sample() {
	w.git(w.main, "init", "-q", "-b", "dev")
	// gabo.yaml first: init writes the engine files of runtimes.assistant.
	w.write("gabo.yaml", sampleConfig)
	w.run(w.main, nil, w.bin, "init")
	w.write(".gabo/runtimes/gabo-runtime-claude/SKILL.md",
		"---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\n")
	spec, err := agentspec.Load(skills.Catalog())
	w.must(err)
	agents := []agentspec.Agent{{ID: spec.MainSession.ID, Skills: spec.MainSession.Skills, Template: spec.MainSession.Template}}
	agents = append(agents, spec.Agents...)
	for _, a := range agents {
		// The agent file lists its spec skills and the fixed-name links its
		// template names (gabo check compares both).
		listed := slices.Clone(a.Skills)
		tmpl, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
		w.must(err)
		for _, m := range templateSlotRE.FindAllStringSubmatch(string(tmpl), -1) {
			link := "gabo-methodology"
			if m[2] != "" {
				link = "gabo-" + strings.ReplaceAll(m[2], "_", "-")
			}
			if !slices.Contains(listed, link) {
				listed = append(listed, link)
			}
		}
		w.write(".claude/agents/"+a.ID+".md", "---\nname: "+a.ID+"\ngabo-version: "+version+"\nskills:\n  - "+strings.Join(listed, "\n  - ")+"\n---\n")
		for _, name := range a.Skills {
			dst := filepath.Join(w.main, ".claude", "skills", name)
			if _, err := os.Lstat(dst); err != nil {
				w.link("../../.gabo/skills/"+name, dst)
			}
		}
	}
	// The user-level reviewer link gabo-runtime-claude makes (build step 4).
	w.link(filepath.Join(w.home, ".local", "share", "gabo", "skills", "gabo-reviewer"),
		filepath.Join(w.home, ".claude", "skills", "gabo-reviewer"))
	w.write(".gabo/adapters/tasks-fake.md", fakeAdapter("tasks", w.contractOps("gabo-contract-tasks")))
	w.write(".gabo/adapters/memory-fake.md", fakeAdapter("memory", w.contractOps("gabo-contract-memory")))
	w.write("README.md", "# sample\n")
	w.git(w.main, "add", "-A")
	w.git(w.main, "commit", "-qm", "chore: sample repository")
	w.git(w.root, "clone", "-q", "--bare", w.main, w.origin)
	w.git(w.main, "remote", "add", "origin", w.origin)
	w.git(w.main, "fetch", "-q", "origin")
	w.git(w.main, "branch", "-q", "--set-upstream-to=origin/dev")
	b, err := json.Marshal([]issue{
		{Number: 1, Title: "T001: first feature", State: "open"},
		{Number: 2, Title: "T002: second feature", State: "open", BlockedBy: []int{1}},
	})
	w.must(err)
	w.must(os.WriteFile(w.tasks.path, b, 0o600))
}

// contractOps lists the `### <op>` headings under `## Operations` of a
// contract skill in the sample's catalog copy.
func (w *world) contractOps(contract string) []string {
	f, err := os.Open(filepath.Join(w.main, ".gabo", "skills", contract, "SKILL.md"))
	w.must(err)
	defer f.Close()
	var ops []string
	in := false
	for s := bufio.NewScanner(f); s.Scan(); {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "## "):
			in = line == "## Operations"
		case in && strings.HasPrefix(line, "### "):
			ops = append(ops, strings.TrimPrefix(line, "### "))
		}
	}
	return ops
}

func fakeAdapter(slot string, ops []string) string {
	var b strings.Builder
	b.WriteString("# " + slot + "-fake: test adapter of the e2e mini wave\n\nState lives in a JSON file of the test's scratch dir.\n\n## Operations\n")
	for _, op := range ops {
		b.WriteString("\n### " + op + "\n\nServed by the e2e fake.\n")
	}
	return b.String()
}

const sampleConfig = `version: 1
mode: auto
repo:
  name: example-org/sample
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
slots:
  tasks:
    adapter: .gabo/adapters/tasks-fake.md
  repository_host:
    adapter: .gabo/skills/gabo-contract-repository-host/examples/host-github.md
  memory:
    adapter: .gabo/adapters/memory-fake.md
  methodology:
    adapter: .gabo/skills/gabo-contract-methodology/examples/methodology-tdd.md
  reviewer_placement:
    adapter: .gabo/skills/gabo-contract-reviewer-placement/examples/reviewer-local.md
reviewer:
  placement: local
  signing:
    marker: gbai-reviewer-bot
review:
  checks:
    - gabo check
  auto_merge: true
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md, gabo.yaml, .claude/]
parallelism: 1
worktree:
  keep: []
`

// --- fake GitHub ---------------------------------------------------------------

func (w *world) fx(name string) string { return filepath.Join(w.fixture, name) }

func (w *world) writeJSON(name string, v any) {
	b, err := json.Marshal(v)
	w.must(err)
	w.must(os.WriteFile(w.fx(name), b, 0o600))
}

type pullRequest struct {
	Number     int    `json:"number"`
	State      string `json:"state"`
	HeadRefOid string `json:"headRefOid"`
	HeadRef    string `json:"headRefName"`
	BaseRef    string `json:"baseRefName"`
	IsDraft    bool   `json:"isDraft"`
	IsFork     bool   `json:"isCrossRepository"`
	Mergeable  string `json:"mergeable"`
}

// openPR is GitHub opening (or updating) PR n for branch at head: the PR
// refs, its merge result on the dev tip and the files fake gh answers with.
func (w *world) openPR(n int, branch, head string) {
	dev := w.git(w.origin, "rev-parse", "refs/heads/dev")
	tree := w.git(w.origin, "rev-parse", head+"^{tree}")
	merge := w.git(w.origin, "commit-tree", tree, "-p", dev, "-p", head, "-m", "Merge PR #"+itoa(n))
	w.git(w.origin, "update-ref", "refs/pull/"+itoa(n)+"/head", head)
	w.git(w.origin, "update-ref", "refs/pull/"+itoa(n)+"/merge", merge)
	files := []map[string]string{}
	for _, f := range strings.Fields(w.git(w.origin, "diff", "--name-only", dev, head)) {
		files = append(files, map[string]string{"filename": f})
	}
	w.writeJSON("files_"+itoa(n), files)
	w.writeJSON("pr_"+itoa(n)+".json", pullRequest{Number: n, State: "OPEN", HeadRefOid: head, HeadRef: branch, BaseRef: "dev", Mergeable: "MERGEABLE"})
	if _, err := os.Stat(w.fx("comments_" + itoa(n) + ".json")); err != nil {
		w.must(os.WriteFile(w.fx("comments_"+itoa(n)+".json"), []byte("[]"), 0o600))
	}
	w.writeJSON("commit_"+head+".json", map[string]any{"commit": map[string]any{"committer": map[string]string{"date": "2026-10-20T11:00:00Z"}}})
}

// mergePR is GitHub carrying out the `gh pr merge` the reviewer asked for:
// dev moves to the PR's merge result and the PR shows MERGED.
func (w *world) mergePR(n int) {
	merge := w.git(w.origin, "rev-parse", "refs/pull/"+itoa(n)+"/merge")
	w.git(w.origin, "update-ref", "refs/heads/dev", merge)
	pr := w.readPR(n)
	pr.State = "MERGED"
	w.writeJSON("pr_"+itoa(n)+".json", pr)
}

func (w *world) readPR(n int) pullRequest {
	var pr pullRequest
	b, err := os.ReadFile(w.fx("pr_" + itoa(n) + ".json"))
	if err != nil {
		return pullRequest{}
	}
	w.must(json.Unmarshal(b, &pr))
	return pr
}

// review runs the installed user-level reviewer scripts for one head from the
// sample's main checkout, as the gabo-reviewer agent does, and returns the
// VERDICT line.
func (w *world) review(n int, head string) string {
	w.t.Helper()
	scripts := filepath.Join(w.home, ".local", "share", "gabo", "skills", "gabo-reviewer", "scripts")
	env := w.env(
		"STUB_LOG="+w.fx("calls.log"), "FIXTURE="+w.fixture, "CLAUDE_MODE="+claudeMode,
		"REVIEWER_GH="+filepath.Join(w.stubs, "gh"), "REVIEWER_CLAUDE="+filepath.Join(w.stubs, "claude"),
		"REVIEWER_GABO_BIN="+w.bin, "REVIEWER_REMOTE_URL="+w.origin,
		"GABO_REVIEWER_STATE_DIR="+filepath.Join(w.root, "reviewer-state"),
		"REVIEWER_ENV_FILE=/nonexistent/gabo-e2e-reviewer-env",
		"REVIEWER_NOW=2026-10-20T12:00:00Z", "REVIEWER_MERGE_DELAY=0", "REVIEWER_MERGE_TRIES=2",
	)
	ctx := "STUB_LOG=" + w.fx("calls.log") + "\nFIXTURE=" + w.fixture + "\nCLAUDE_MODE=" + claudeMode + "\n"
	w.must(os.WriteFile(filepath.Join(w.stubs, "stub-context"), []byte(ctx), 0o600))
	out, _ := w.try(w.main, env, "bash", filepath.Join(scripts, "reviewer.sh"), "head", itoa(n), head)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, "VERDICT ") {
		w.t.Fatalf("reviewer gave no VERDICT line:\n%s", out)
	}
	return last
}

// markers counts the reviewer's verdict comments on PR n that sign head.
func (w *world) markers(n int, head string) int {
	b, err := os.ReadFile(w.fx("comments_" + itoa(n) + ".json"))
	w.must(err)
	return strings.Count(string(b), "gbai-reviewer-bot sha="+head)
}

// worktrees lists the paths of the sample's linked worktrees.
func (w *world) worktrees() []string {
	var paths []string
	for _, l := range strings.Split(w.git(w.main, "worktree", "list", "--porcelain"), "\n") {
		if p, ok := strings.CutPrefix(l, "worktree "); ok && p != w.main {
			paths = append(paths, p)
		}
	}
	return paths
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
