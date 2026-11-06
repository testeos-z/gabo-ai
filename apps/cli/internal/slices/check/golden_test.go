package check

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the testdata/golden files")

// TestCommandGolden pins `gabo check` without --pre-write: its output and
// exit code on every intent fixture, on the plain project and on usage
// errors equal the golden files written before the hook mode existed.
func TestCommandGolden(t *testing.T) {
	t.Parallel()
	fixtures, err := os.ReadDir(path.Join("testdata", "intents"))
	if err != nil {
		t.Fatal(err)
	}
	type tc struct {
		name string
		args []string
		tree string
		// edit changes project files; an empty value removes the file.
		edit map[string]string
	}
	cases := []tc{
		{name: "no-intents"},
		{name: "usage-extra-arg", args: []string{"extra"}},
		{name: "usage-unknown-flag", args: []string{"--bogus"}},
		{name: "versions-no-stamp", edit: map[string]string{".gabo/VERSION": ""}},
		{name: "versions-no-version-line", edit: map[string]string{".claude/agents/beta.md": "---\nname: beta\n---\nbody\n"}},
	}
	for _, f := range fixtures {
		cases = append(cases, tc{name: "intents-" + f.Name(), tree: f.Name()})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			proj := project()
			if c.edit != nil {
				proj = with(proj, c.edit)
			}
			if c.tree != "" {
				proj = withTree(t, proj, c.tree)
			}
			// A nil slice would make cobra read os.Args (the test flags).
			out, err := runCommand(t, proj, append([]string{}, c.args...)...)
			code := 0
			var ee *ExitError
			if errors.As(err, &ee) {
				code = ee.Code
			} else if err != nil {
				t.Fatalf("err = %v, want an ExitError", err)
			}
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			got := fmt.Sprintf("exit %d\nerror: %q\n--- stdout\n%s", code, msg, out)
			file := path.Join("testdata", "golden", c.name+".golden")
			if *update {
				if err := os.MkdirAll(path.Dir(file), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read golden (run with -update on the base to write it): %v", err)
			}
			if got != string(want) {
				t.Fatalf("output changed:\n got: %s\nwant: %s", got, want)
			}
		})
	}
}
