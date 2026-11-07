package check

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
)

// designFixture is where a golden case's testdata/design/<name>.md is put
// in the project (spec 0012 S7).
const designFixture = "docs/product/DESIGN.md"

// TestDesignGolden pins `gabo check` on docs/product/DESIGN.md (spec 0012
// S7, TRD 9.4 Design norms): each case puts testdata/design/<name>.md at
// docs/product/DESIGN.md of the passing project and compares the output and
// exit code with testdata/golden/<name>.golden. design-none has no file.
func TestDesignGolden(t *testing.T) {
	t.Parallel()
	cases := []string{
		"design-none",
		"design-valid",
		"design-not-applicable",
		"design-legacy",
		"design-missing-section",
		"design-columns",
		"design-no-table",
		"design-rule-rows",
		"design-exceptions",
		"design-exceptions-missing",
		"design-changelog",
		"design-changelog-missing",
		"design-escaped-pipe",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			proj := project()
			if name != "design-none" {
				b, err := os.ReadFile(path.Join("testdata", "design", name+".md"))
				if err != nil {
					t.Fatal(err)
				}
				proj = with(proj, map[string]string{designFixture: string(b)})
			}
			out, err := runCommandConfig(t, proj, validConfig)
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
			file := path.Join("testdata", "golden", name+".golden")
			if *update {
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if got != string(want) {
				t.Fatalf("output changed:\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// Without docs/product/DESIGN.md the output is the one from before the
// design rule existed.
func TestDesignNoneEqualsNoIntents(t *testing.T) {
	t.Parallel()
	none, err := os.ReadFile(path.Join("testdata", "golden", "design-none.golden"))
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.ReadFile(path.Join("testdata", "golden", "no-intents.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(none) != string(base) {
		t.Fatalf("design-none.golden differs from no-intents.golden:\n%s\nvs\n%s", none, base)
	}
}
