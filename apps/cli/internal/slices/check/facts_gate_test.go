package check

import (
	"testing"
	"testing/fstest"
)

// withStamp adds .gabo/VERSION holding stamp to proj.
func withStamp(proj fstest.MapFS, stamp string) fstest.MapFS {
	proj[StampFile] = file(stamp)
	return proj
}

// The version rule (intent 0003 S7): on session_start gabo compares the
// package version fact with .gabo/VERSION and returns one info notification
// naming both and the next step when they differ; equal versions, no
// .gabo/VERSION or no package version give none. tool_call never runs it.
func TestJudgeFactsVersionRule(t *testing.T) {
	t.Parallel()
	gentle := []string{"read", "write", "subagent_run"}
	tests := []struct {
		name  string
		proj  fstest.MapFS
		event string
		pkg   string
		want  []Notification
	}{
		{name: "package newer than the stamp", proj: withStamp(intentWith(""), "0.4.0\n"), event: EventSessionStart, pkg: "0.5.0",
			want: []Notification{{Level: LevelInfo, Text: "The gabo Pi package is 0.5.0 but this project's .gabo/VERSION is 0.4.0; next: run gabo init with the gabo CLI 0.5.0 to update the project's catalog."}}},
		{name: "stamp newer than the package", proj: withStamp(intentWith(""), "0.6.0\n"), event: EventSessionStart, pkg: "0.5.0",
			want: []Notification{{Level: LevelInfo, Text: "The gabo Pi package is 0.5.0 but this project's .gabo/VERSION is 0.6.0; next: update the gabo Pi package to 0.6.0."}}},
		{name: "unparsable stamp", proj: withStamp(intentWith(""), "dev\n"), event: EventSessionStart, pkg: "0.5.0",
			want: []Notification{{Level: LevelInfo, Text: "The gabo Pi package is 0.5.0 but this project's .gabo/VERSION is dev; next: run gabo init or update the gabo Pi package so both match."}}},
		{name: "equal versions", proj: withStamp(intentWith(""), "0.5.0\n"), event: EventSessionStart, pkg: "0.5.0"},
		{name: "equal versions, v prefix", proj: withStamp(intentWith(""), "0.5.0\n"), event: EventSessionStart, pkg: "v0.5.0"},
		{name: "no .gabo/VERSION", proj: intentWith(""), event: EventSessionStart, pkg: "0.5.0"},
		{name: "no package version", proj: withStamp(intentWith(""), "0.4.0\n"), event: EventSessionStart},
		{name: "tool_call", proj: withStamp(intentWith(""), "0.4.0\n"), event: EventToolCall, pkg: "0.5.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := Facts{Event: tt.event, Engine: "pi", Tools: gentle, PackageVersion: tt.pkg}
			if tt.event == EventToolCall {
				f.TargetPaths = []string{"README.md"}
			}
			got := NewUseCase(tt.proj, failingReader{}, goodUser(), Settings{Root: preRoot}).JudgeFacts(f)
			want := FactsResult{Notify: tt.want}
			if want.Notify == nil {
				want.Notify = []Notification{}
			}
			if g, w := mustJSON(t, got), mustJSON(t, want); g != w {
				t.Fatalf("JudgeFacts = %s, want %s", g, w)
			}
		})
	}
}

// The version notification follows the subagent refusal.
func TestJudgeFactsVersionAfterRefusal(t *testing.T) {
	t.Parallel()
	u := NewUseCase(withStamp(intentWith(""), "0.4.0\n"), failingReader{}, goodUser(), Settings{Root: preRoot})
	got := u.JudgeFacts(Facts{Event: EventSessionStart, Engine: "pi", Tools: []string{"read"}, PackageVersion: "0.5.0"})
	if len(got.Notify) != 2 || got.Notify[0].Text != SubagentRefusal || got.Notify[1].Level != LevelInfo {
		t.Fatalf("JudgeFacts = %s, want the refusal warning, then the version info", mustJSON(t, got))
	}
}

// The stage gate (intent 0003 S6): a tool_call is judged with the
// path-argument mode of intent 0001 S11, and block.reason holds its "✗"
// lines; with the subagent refusal too, the refusal comes first.
func TestJudgeFactsStageGate(t *testing.T) {
	t.Parallel()
	gentle := []string{"write", "subagent_run"}
	pi := []string{"write"}
	unreadable := unreadableFS{MapFS: intentWith(acceptedMD), name: preSpec}
	cases := []struct {
		name    string
		u       *UseCase
		tools   []string
		targets []string
		want    *Block
	}{
		{name: "unaccepted spec blocks", u: NewUseCase(intentWith(unfilledMD), failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: gentle, targets: []string{prePlan}, want: &Block{Reason: notAccepted}},
		{name: "missing spec blocks", u: NewUseCase(intentWith(""), failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: gentle, targets: []string{prePlan}, want: &Block{Reason: noSpec}},
		{name: "accepted spec passes", u: NewUseCase(intentWith(acceptedMD), failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: gentle, targets: []string{prePlan}},
		{name: "unreadable spec fails closed", u: NewUseCase(unreadable, failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: gentle, targets: []string{prePlan},
			want: &Block{Reason: "✗ " + prePlan + ": spec.md cannot be read (read " + preSpec + ": open " + preSpec + ": permission denied); it counts as not accepted"}},
		{name: "other paths pass", u: NewUseCase(intentWith(unfilledMD), failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: gentle, targets: []string{"README.md", preSpec}},
		{name: "blocked by both rules, refusal first", u: NewUseCase(intentWith(unfilledMD), failingReader{}, goodUser(), Settings{Root: preRoot}),
			tools: pi, targets: []string{piAgent, prePlan}, want: &Block{Reason: SubagentRefusal + "\n" + notAccepted}},
		{name: "absolute path with no project root fails closed", u: NewUseCase(intentWith(acceptedMD), failingReader{}, goodUser(), Settings{}),
			tools: gentle, targets: []string{"/elsewhere/" + prePlan},
			want: &Block{Reason: "usage: gabo check --pre-write <path>... (or no path and the hook's tool-call JSON on stdin): absolute path /elsewhere/" + prePlan + " with no project root"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.u.JudgeFacts(Facts{Event: EventToolCall, Engine: "pi", Tools: tt.tools, TargetPaths: tt.targets})
			want := FactsResult{Notify: []Notification{}, Block: tt.want}
			if g, w := mustJSON(t, got), mustJSON(t, want); g != w {
				t.Fatalf("JudgeFacts = %s, want %s", g, w)
			}
		})
	}
}
