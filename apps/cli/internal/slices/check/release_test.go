package check

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// The route every release block names (intent 0028 S5, S11).
const releaseRoute = "the owner runs this outside agent sessions"

// fakePRs is the hand-written pull-request port: bases maps a ref to its base
// branch; err makes every read fail; calls records each read as repo#ref.
type fakePRs struct {
	bases map[string]string
	err   error
	calls []string
}

func (f *fakePRs) PRBase(repo, ref string) (string, error) {
	f.calls = append(f.calls, repo+"#"+ref)
	if f.err != nil {
		return "", f.err
	}
	base, ok := f.bases[ref]
	if !ok {
		return "", errors.New("no such pull request")
	}
	return base, nil
}

// judgeReleaseCall runs the shell judgement of one call with cfg as gabo.yaml,
// the engine and agent id of the call, prs as the host and out as the sink.
func judgeReleaseCall(t *testing.T, cfg config.Reader, engine, agentID, command string, prs PRBaseReader, out *fakeOutcomes) (ShellVerdict, error) {
	t.Helper()
	s := Settings{Layers: layers(), Engine: engine, Task: "I0028/T5", OutcomePath: "/scratch/outcomes.jsonl", Outcomes: out, PullRequests: prs}
	return NewUseCase(project(), cfg, goodUser(), s).JudgeShell(ShellCall{Command: command, AgentID: agentID})
}

// The two session kinds of S5: a main session (no agent id, claude) and a
// coder subagent (an agent id).
var (
	mainSession  = struct{ engine, agent string }{"claude", ""}
	coderSession = struct{ engine, agent string }{"claude", "a1"}
)

// I0028 S5, S11: each release command is blocked in a main and a coder
// session, with one release_gate line naming the rule and the route.
func TestReleaseGateBlocksReleaseCommandsInEverySession(t *testing.T) {
	t.Parallel()
	tests := []struct{ command, rule string }{
		{"git tag v1.2.0", "git tag that creates a tag is blocked in an agent session"},
		{"git push --tags", "git push --tags is blocked in an agent session"},
		{"npm publish", "a package publish (npm publish, bun publish) is blocked in an agent session"},
		{"bun publish --access public", "a package publish (npm publish, bun publish) is blocked in an agent session"},
		{"claude plugin tag plugins/claude-code", "claude plugin tag is blocked in an agent session"},
		{"gh release create v1", "gh release create is blocked in an agent session"},
		{"git tag -a v1.2.0 -m\"Release 1.2\"", "git tag that creates a tag is blocked in an agent session"},
		{"npx npm publish", "a package publish (npm publish, bun publish) is blocked in an agent session"},
	}
	for _, tt := range tests {
		for _, s := range []struct{ name, engine, agent string }{
			{"main", mainSession.engine, mainSession.agent},
			{"coder", coderSession.engine, coderSession.agent},
		} {
			t.Run(s.name+" "+tt.command, func(t *testing.T) {
				t.Parallel()
				out := &fakeOutcomes{}
				v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, s.engine, s.agent, tt.command, &fakePRs{}, out)
				want := "✗ release_gate: " + tt.rule + "\nnext: " + releaseRoute + "\n"
				if err != nil || v.Exit != ExitFailed || v.Out != want {
					t.Fatalf("verdict = %+v, err = %v; want exit %d and %q", v, err, ExitFailed, want)
				}
				if len(out.lines) != 1 || !strings.Contains(out.lines[0], `"lock":"release_gate"`) || !strings.Contains(out.lines[0], `"outcome":"blocked"`) {
					t.Fatalf("records = %q; want one blocked release_gate record", out.lines)
				}
			})
		}
	}
}

// I0028 S5: listing tags and every other command of the same tools is allowed,
// with no output and no record.
func TestReleaseGateAllowsListingAndNeighbours(t *testing.T) {
	t.Parallel()
	allowed := []string{
		"git tag",
		"git tag -l 'v*'",
		"git tag --list",
		"git tag -n",
		"git tag -d v0.0.1",
		"git tag --points-at HEAD",
		"git push origin feat/x",
		"npm pack",
		"npm view @gabo-ai/pi version",
		"claude plugin validate .",
		"gh release list",
		"gh release view v1",
		"gh pr view 12",
		"git status",
	}
	for _, c := range allowed {
		for _, s := range []string{mainSession.agent, coderSession.agent} {
			out := &fakeOutcomes{}
			v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", s, c, &fakePRs{}, out)
			if err != nil || v.Exit != 0 || v.Out != "" || len(out.lines) != 0 {
				t.Fatalf("%q (agent %q): verdict = %+v, err = %v, records = %q; want allowed", c, s, v, err, out.lines)
			}
		}
	}
}

// I0028 S6: one level of bash -c is read, eval is not; every command of a line
// is judged.
func TestReleaseGateUnwrapsOneLevelAndJudgesEveryCommand(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"bash -c 'git tag v1'",
		"sh -c \"npm publish\"",
		"git status && git tag v1",
		"echo ok; gh release create v1",
	}
	for _, c := range blocked {
		v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", "", c, &fakePRs{}, &fakeOutcomes{})
		if err != nil || v.Exit != ExitFailed {
			t.Fatalf("%q: verdict = %+v, err = %v; want blocked", c, v, err)
		}
	}
	v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", "", "eval \"git tag v1\"", &fakePRs{}, &fakeOutcomes{})
	if err != nil || v.Exit != 0 {
		t.Fatalf("eval \"git tag v1\": verdict = %+v, err = %v; want allowed", v, err)
	}
}

// I0028 D4: a merge in a main session reads the pull request base once; main
// is blocked, dev is allowed, and a failed read is blocked.
func TestReleaseGateMergeReadsTheBase(t *testing.T) {
	t.Parallel()
	const mainRule = "a merge of a pull request into main is blocked in an agent session"
	const unreadRule = "a merge whose pull request base cannot be read is blocked"
	tests := []struct {
		name    string
		command string
		prs     *fakePRs
		exit    int
		rule    string
		calls   []string
	}{
		{name: "base main blocks", command: "gh pr merge 12", prs: &fakePRs{bases: map[string]string{"12": "main"}}, exit: ExitFailed, rule: mainRule, calls: []string{"#12"}},
		{name: "base dev allowed", command: "gh pr merge 12", prs: &fakePRs{bases: map[string]string{"12": "dev"}}, exit: 0, calls: []string{"#12"}},
		{name: "subject value is not the reference", command: "gh pr merge -t main 12", prs: &fakePRs{bases: map[string]string{"12": "dev", "main": "main"}}, exit: 0, calls: []string{"#12"}},
		{name: "body value does not hide a main merge", command: "gh pr merge --body feat/x 12", prs: &fakePRs{bases: map[string]string{"12": "main", "feat/x": "dev"}}, exit: ExitFailed, rule: mainRule, calls: []string{"#12"}},
		{name: "current branch pull request", command: "gh pr merge", prs: &fakePRs{bases: map[string]string{"": "main"}}, exit: ExitFailed, rule: mainRule, calls: []string{"#"}},
		{name: "repo flag is passed", command: "gh -R o/r pr merge 12", prs: &fakePRs{bases: map[string]string{"12": "dev"}}, exit: 0, calls: []string{"o/r#12"}},
		{name: "failed read blocks", command: "gh pr merge 12", prs: &fakePRs{err: errors.New("host down")}, exit: ExitFailed, rule: unreadRule, calls: []string{"#12"}},
		{name: "unknown pull request blocks", command: "gh pr merge 99", prs: &fakePRs{bases: map[string]string{}}, exit: ExitFailed, rule: unreadRule, calls: []string{"#99"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", "", tt.command, tt.prs, &fakeOutcomes{})
			if err != nil || v.Exit != tt.exit {
				t.Fatalf("verdict = %+v, err = %v; want exit %d", v, err, tt.exit)
			}
			if tt.rule != "" && !strings.HasPrefix(v.Out, "✗ release_gate: "+tt.rule+"\nnext: ") {
				t.Fatalf("out = %q; want the rule %q", v.Out, tt.rule)
			}
			if !reflect.DeepEqual(tt.prs.calls, tt.calls) {
				t.Fatalf("reads = %q; want %q", tt.prs.calls, tt.calls)
			}
		})
	}
}

// I0028 D4: in a coder session every merge command is blocked without a read,
// and with release_gate off a merge is allowed without a read.
func TestReleaseGateMergeInCoderSessionAndOff(t *testing.T) {
	t.Parallel()
	prs := &fakePRs{bases: map[string]string{"12": "dev"}}
	v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", "a1", "gh pr merge 12", prs, &fakeOutcomes{})
	want := "✗ release_gate: gh pr merge is blocked in a coder session\nnext: a coder never merges; the reviewer merges a clean pull request into dev\n"
	if err != nil || v.Exit != ExitFailed || v.Out != want {
		t.Fatalf("coder merge: verdict = %+v, err = %v; want %q", v, err, want)
	}
	if len(prs.calls) != 0 {
		t.Fatalf("coder merge read %q; want no read", prs.calls)
	}
	prs = &fakePRs{bases: map[string]string{"12": "main"}}
	v, err = judgeReleaseCall(t, configFiles{projectFile: validConfig + "locks:\n  release_gate: off\n"}, "claude", "", "gh pr merge 12", prs, &fakeOutcomes{})
	if err != nil || v.Exit != 0 || len(prs.calls) != 0 {
		t.Fatalf("release_gate off: verdict = %+v, err = %v, reads = %q; want allowed with no read", v, err, prs.calls)
	}
}

// I0028 S10, S12, S13: the release_gate states give the same exits, messages
// and records as shell_guard does; the all-off switch turns it off.
func TestReleaseGateStates(t *testing.T) {
	t.Parallel()
	block := "✗ release_gate: git tag that creates a tag is blocked in an agent session\nnext: " + releaseRoute + "\n"
	tests := []struct {
		name    string
		cfg     string
		exit    int
		out     string
		outcome string
	}{
		{name: "no key blocks by default", cfg: validConfig, exit: ExitFailed, out: block, outcome: "blocked"},
		{name: "off runs with nothing recorded", cfg: validConfig + "locks:\n  release_gate: off\n", exit: 0},
		{name: "warn runs and records warned", cfg: validConfig + "locks:\n  release_gate: warn\n", exit: 0, outcome: "warned"},
		{name: "ask exits 3 and records asked", cfg: validConfig + "locks:\n  release_gate: ask\n", exit: ExitAsk, out: block, outcome: "asked"},
		{name: "ask in auto blocks with the note", cfg: validConfig + "mode: auto\nlocks:\n  release_gate: ask\n", exit: ExitFailed,
			out: "✗ release_gate: git tag that creates a tag is blocked in an agent session (the state was ask; mode auto blocks instead of prompting)\nnext: " + releaseRoute + "\n", outcome: "blocked"},
		{name: "all switch off", cfg: validConfig + "locks:\n  enabled: false\n", exit: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			v, err := judgeReleaseCall(t, configFiles{projectFile: tt.cfg}, "claude", "a1", "git tag v1", &fakePRs{}, out)
			if err != nil || v.Exit != tt.exit || v.Out != tt.out {
				t.Fatalf("verdict = %+v, err = %v; want exit %d and out %q", v, err, tt.exit, tt.out)
			}
			if tt.outcome == "" {
				if len(out.lines) != 0 {
					t.Fatalf("records = %q, want none", out.lines)
				}
				return
			}
			if len(out.lines) != 1 || !strings.Contains(out.lines[0], `"outcome":"`+tt.outcome+`"`) || !strings.Contains(out.lines[0], `"lock":"release_gate"`) {
				t.Fatalf("records = %q; want one %s release_gate record", out.lines, tt.outcome)
			}
		})
	}
}

// I0028 S2, S4, S5: a line that breaks a shell_guard rule and a release_gate
// rule gets both messages, in command order, one record per lock, and the
// stronger exit.
func TestReleaseGateAndShellGuardTogether(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	v, err := judgeReleaseCall(t, configFiles{projectFile: validConfig}, "claude", "", "git push origin main && git tag v1", &fakePRs{}, out)
	want := "✗ shell_guard: git push to main is blocked\nnext: the owner pushes to main outside agent sessions\n" +
		"✗ release_gate: git tag that creates a tag is blocked in an agent session\nnext: " + releaseRoute + "\n"
	if err != nil || v.Exit != ExitFailed || v.Out != want {
		t.Fatalf("verdict = %+v, err = %v; want %q", v, err, want)
	}
	if len(out.lines) != 2 || !strings.Contains(out.lines[0], `"lock":"shell_guard"`) || !strings.Contains(out.lines[1], `"lock":"release_gate"`) {
		t.Fatalf("records = %q; want one shell_guard then one release_gate record", out.lines)
	}
}

// I0028 S5, D4: the pure reading of one line, with the session kind applied.
func TestReleaseLineReading(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		coder   bool
		rules   []string
		merges  []mergeCall
	}{
		{command: "git tag -a v1 -m msg", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git tag -m msg v1", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git -C ../wt tag v1", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git push origin feat/x --tags", rules: []string{"git push --tags is blocked in an agent session"}},
		{command: "git push origin --tags -- x", rules: []string{"git push --tags is blocked in an agent session"}},
		{command: "gh -R o/r release create v1", rules: []string{"gh release create is blocked in an agent session"}},
		{command: "gh pr merge --squash 12", rules: nil, merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge --repo=o/r 12", merges: []mergeCall{{Repo: "o/r", Ref: "12"}}},
		// Option values are not words: the reference is the last positional.
		{command: "gh pr merge -t x 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge --subject 7 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge -b feat/x 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge --body dev 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge --match-head-commit abc123 7", merges: []mergeCall{{Ref: "7"}}},
		{command: "gh pr merge -bfeat 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge --body=dev 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge -sb feat/x 12", merges: []mergeCall{{Ref: "12"}}},
		{command: "gh pr merge -d 12", merges: []mergeCall{{Ref: "12"}}},
		// A short option with its value attached is not a list of flags.
		{command: "git tag -mmsg v1", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git tag -a v1.2.0 -m\"Release 1.2\"", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git tag -Fnotes.txt v1", rules: []string{"git tag that creates a tag is blocked in an agent session"}},
		{command: "git tag -dm x v1", rules: nil},
		{command: "npx npm publish", rules: []string{"a package publish (npm publish, bun publish) is blocked in an agent session"}},
		{command: "pnpm publish", rules: []string{"a package publish (npm publish, bun publish) is blocked in an agent session"}},
		{command: "gh pr merge 12", coder: true, rules: []string{"gh pr merge is blocked in a coder session"}},
		{command: "git status", rules: nil},
	}
	for _, tt := range tests {
		r := releaseLine(tt.command, tt.coder)
		var got []string
		for _, v := range r.Violations {
			got = append(got, v.Rule)
		}
		if !reflect.DeepEqual(got, tt.rules) {
			t.Errorf("%q (coder %v): rules = %q; want %q", tt.command, tt.coder, got, tt.rules)
		}
		if !reflect.DeepEqual(r.Merges, tt.merges) {
			t.Errorf("%q: merges = %+v; want %+v", tt.command, r.Merges, tt.merges)
		}
	}
}
