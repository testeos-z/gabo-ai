package check

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// fakeBase is an in-memory BaseTracker: tracked holds the paths tracked at the
// base commit, err makes every read fail, and refs records the refs asked for.
type fakeBase struct {
	tracked map[string]bool
	err     error
	refs    []string
}

func (f *fakeBase) Tracked(ref, rel string) (bool, error) {
	f.refs = append(f.refs, ref)
	if f.err != nil {
		return false, f.err
	}
	return f.tracked[rel], nil
}

// runTestWriteHook runs the write hook as runWriteHook does, with base as the
// base-commit reader; the run's outcome file is named, so records are kept.
func runTestWriteHook(t *testing.T, cfg config.Reader, base BaseTracker, out *fakeOutcomes, engine, stdin string) (string, error) {
	t.Helper()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		s := Settings{Root: preRoot, Layers: layers(), Engine: engine, Task: "I0028/T8", OutcomePath: "/scratch/outcomes.jsonl", Outcomes: out, Base: base}
		return NewUseCase(project(), cfg, goodUser(), s), nil
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs([]string{"--pre-write"})
	err := cmd.ExecuteContext(context.Background())
	return buf.String(), err
}

// testFileConfig is a project whose integration branch is dev, with the
// test_files lock at its default (warn) unless lock says otherwise.
func testFileConfig(lock string) string {
	return validConfig + lock
}

// I0028 S9: a coder edit to an existing tracked test file runs, prints nothing
// and appends one warned record with the lock, the rule and the path.
func TestTestFileEditWarnsOnceForTrackedTestFile(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"foo_test.go", "a.test.ts", "b_test.rb", "test/x.go", "tests/y.py"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			base := &fakeBase{tracked: map[string]bool{path: true}}
			got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", coderWriteJSON(path))
			if err != nil || got != "" {
				t.Fatalf("err = %v, output = %q; want nil and no output", err, got)
			}
			if len(out.lines) != 1 {
				t.Fatalf("records = %q, want one", out.lines)
			}
			line := out.lines[0]
			for _, want := range []string{`"outcome":"warned"`, `"lock":"test_files"`, `"engine":"claude"`, `"task":"I0028/T8"`, `"rule":"a coder write to the existing test file ` + path + `"`} {
				if !strings.Contains(line, want) {
					t.Fatalf("record %q lacks %s", line, want)
				}
			}
		})
	}
}

// I0028 S9: the base is the merge base of HEAD and origin/<integration_branch>.
func TestTestFileAsksBaseOfIntegrationBranch(t *testing.T) {
	t.Parallel()
	base := &fakeBase{tracked: map[string]bool{"foo_test.go": true}}
	_, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, &fakeOutcomes{}, "claude", coderWriteJSON("foo_test.go"))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(base.refs) != 1 || base.refs[0] != "origin/dev" {
		t.Fatalf("base refs = %q, want [origin/dev]", base.refs)
	}
}

// I0028 S9: a new test file, one the base commit does not track, records nothing.
func TestTestFileNewFileRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	base := &fakeBase{tracked: map[string]bool{}}
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", coderWriteJSON("bar_test.go"))
	if err != nil || got != "" || len(out.lines) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q; want nil, no output, no record", err, got, out.lines)
	}
}

// I0028 S9, S10: at test_files off nothing is judged: no record and no base read.
func TestTestFileOffRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	base := &fakeBase{tracked: map[string]bool{"foo_test.go": true}}
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("locks:\n  test_files: off\n")}, base, out, "claude", coderWriteJSON("foo_test.go"))
	if err != nil || got != "" || len(out.lines) != 0 || len(base.refs) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q, base reads = %d; want nil, nothing", err, got, out.lines, len(base.refs))
	}
}

// I0028 S9: an ordinary path is not a test file: no record and no base read.
func TestTestFileOrdinaryPathRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	base := &fakeBase{tracked: map[string]bool{"apps/cli/main.go": true}}
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", coderWriteJSON("apps/cli/main.go"))
	if err != nil || got != "" || len(out.lines) != 0 || len(base.refs) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q, base reads = %d; want nil, nothing", err, got, out.lines, len(base.refs))
	}
}

// I0028 S9 (decision 13): the main session is not judged, even for a tracked
// test file.
func TestTestFileMainSessionRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	base := &fakeBase{tracked: map[string]bool{"foo_test.go": true}}
	stdin := `{"tool_input":{"file_path":"foo_test.go"}}`
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", stdin)
	if err != nil || got != "" || len(out.lines) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q; want nil, no output, no record", err, got, out.lines)
	}
}

// I0028 D6: a base that cannot be read counts the file as not tracked: the
// write runs and nothing is recorded.
func TestTestFileUnreadableBaseRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	base := &fakeBase{err: errors.New("no merge base")}
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", coderWriteJSON("foo_test.go"))
	if err != nil || got != "" || len(out.lines) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q; want nil, no output, no record", err, got, out.lines)
	}
}

// I0028 S9, S10: a warn record that cannot be appended never blocks the write;
// the failure is printed as one line and the write runs (exit 0).
func TestTestFileRecordFailureNeverBlocks(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{err: fs.ErrPermission}
	base := &fakeBase{tracked: map[string]bool{"foo_test.go": true}}
	got, err := runTestWriteHook(t, configFiles{projectFile: testFileConfig("")}, base, out, "claude", coderWriteJSON("foo_test.go"))
	if err != nil {
		t.Fatalf("err = %v, want nil (the write runs)", err)
	}
	want := "test_files: the warning could not be recorded (permission denied); the write runs\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
