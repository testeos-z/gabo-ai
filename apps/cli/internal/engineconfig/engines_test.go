package engineconfig

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSetTakesEachEngineOnceInFixedOrder(t *testing.T) {
	cases := []struct {
		name     string
		runtimes []string
		want     []string
	}{
		{"spec example", []string{"codex", "claude", "claude"}, []string{"claude", "codex"}},
		{"all known reversed", []string{"pi", "opencode", "codex"}, []string{"codex", "opencode", "pi"}},
		{"unknown names last in appearance order", []string{"zeta", "pi", "acme"}, []string{"pi", "zeta", "acme"}},
		{"unknown once", []string{"acme", "acme", "claude"}, []string{"claude", "acme"}},
		{"empty role skipped", []string{"", "claude", " "}, []string{"claude"}},
		{"nothing", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Set(c.runtimes...); !slices.Equal(got, c.want) {
				t.Errorf("Set(%q) = %q, want %q", c.runtimes, got, c.want)
			}
		})
	}
}

// A name outside the full tier is in the flexible tier (S8, amended
// 2026-11-04): one line, no step, no write, no failure.
func TestIntegrateFlexibleTierEngineWritesNothingAndDoesNotFail(t *testing.T) {
	report, writes := Integrate(Set("acme"), Input{Version: "0.2.6"})
	if len(writes) != 0 {
		t.Errorf("writes = %v, want none", writes)
	}
	if report.Failed() {
		t.Error("Failed() = true, want false for a flexible-tier engine")
	}
	want := []Section{{Engine: "acme", Lines: []string{"acme: flexible tier: skills and configuration only; nothing written"}}}
	if len(report.Sections) != 1 || report.Sections[0].Engine != want[0].Engine || !slices.Equal(report.Sections[0].Lines, want[0].Lines) || len(report.Sections[0].Steps) != 0 {
		t.Errorf("Sections = %+v, want %+v", report.Sections, want)
	}
}

func TestIntegrateUsesTheRuleTableInSetOrder(t *testing.T) {
	saved := rules
	t.Cleanup(func() { rules = saved })
	var seen []string
	rules = map[string]Rule{
		"claude": func(in Input) Result {
			seen = append(seen, "claude")
			return Result{Section: Section{Lines: []string{"wrote .claude/settings.json"}}, Writes: map[string][]byte{".claude/settings.json": []byte("{}\n")}}
		},
		"codex": func(in Input) Result {
			seen = append(seen, "codex")
			return Result{Section: Section{Lines: []string{"✗ .codex/config.toml: bad"}}}
		},
	}
	report, writes := Integrate(Set("codex", "acme", "claude"), Input{})
	if !slices.Equal(seen, []string{"claude", "codex"}) {
		t.Errorf("rules ran %q, want claude then codex", seen)
	}
	var engines []string
	for _, s := range report.Sections {
		engines = append(engines, s.Engine)
	}
	if !slices.Equal(engines, []string{"claude", "codex", "acme"}) {
		t.Errorf("sections = %q, want claude, codex, acme", engines)
	}
	if len(writes) != 1 || string(writes[".claude/settings.json"]) != "{}\n" {
		t.Errorf("writes = %q, want only .claude/settings.json", writes)
	}
	if !report.Failed() {
		t.Error("Failed() = false, want true with a ✗ line")
	}
}

// The engine domain never looks for engines on the machine (S1, D1).
func TestPackageSourceDoesNoMachineLookup(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"exec.LookPath", `"os/exec"`, "UserHomeDir", `"HOME"`, "~/", "$HOME"}
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		for _, bad := range forbidden {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s contains %s", f, bad)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no package source checked")
	}
}
