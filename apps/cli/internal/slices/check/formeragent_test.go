package check

import (
	"strings"
	"testing"
	"testing/fstest"
)

// Intent 0015 S7, D3, D4: the agent files an older build left for the two
// agents the spec no longer lists.
func formerLine(file, id string) string {
	return "✗ " + file + ": agent " + id + " is no longer in the agent spec"
}

type formerEngine struct {
	name  string
	build func(t *testing.T) fstest.MapFS
	run   func(t *testing.T, p fstest.MapFS) Report
	dir   string
	ext   string
	body  string
}

func formerEngines() []formerEngine {
	return []formerEngine{
		{"claude", func(t *testing.T) fstest.MapFS { return claudeBuilt(t, "1.2.3") },
			func(t *testing.T, p fstest.MapFS) Report {
				return run(t, p, configFiles{projectFile: validConfig}, Settings{})
			}, ".claude/agents", ".md", "---\nname: X\ngabo-version: 1.2.3\n---\nbody\n"},
		{"codex", func(t *testing.T) fstest.MapFS { return codexBuilt(t, "1.2.3") },
			func(t *testing.T, p fstest.MapFS) Report { return runCodex(t, p, codexUser(), "1.2.3") },
			".codex/agents", ".toml", "# gabo-version: 1.2.3\nname = \"X\"\n"},
		{"opencode", func(t *testing.T) fstest.MapFS { return opencodeBuilt(t, "1.2.3") },
			func(t *testing.T, p fstest.MapFS) Report { return runOpencode(t, p, "1.2.3") },
			".opencode/agents", ".md", "---\ngabo-version: 1.2.3\n---\nbody\n"},
		{"pi", func(t *testing.T) fstest.MapFS { return piBuilt(t, "1.2.3") },
			func(t *testing.T, p fstest.MapFS) Report { return runPi(t, p, Settings{}) },
			".pi/agents", ".md", "---\nname: X\ngabo-version: 1.2.3\n---\nbody\n"},
	}
}

// (a) a project built per S6 passes with no former-agent line.
func TestSixAgentProjectPassesOnEveryEngine(t *testing.T) {
	t.Parallel()
	for _, e := range formerEngines() {
		t.Run(e.name, func(t *testing.T) {
			t.Parallel()
			r := e.run(t, e.build(t))
			wantPassed(t, r, PassedLine)
			noLineWith(t, r, "no longer in the agent spec")
		})
	}
}

// (b) a leftover gabo-cycle or gabo-reviewer file with a gabo-version line
// fails in every declared agent directory, one line per file.
func TestFormerAgentFileFails(t *testing.T) {
	t.Parallel()
	for _, e := range formerEngines() {
		for _, id := range []string{"gabo-cycle", "gabo-reviewer"} {
			t.Run(e.name+"/"+id, func(t *testing.T) {
				t.Parallel()
				p := e.build(t)
				name := e.dir + "/" + id + e.ext
				p[name] = file(strings.ReplaceAll(e.body, "X", id))
				r := e.run(t, p)
				wantFailed(t, r, formerLine(name, id), NextRebuild)
				var n int
				for _, l := range r.Lines {
					if strings.Contains(l, "no longer in the agent spec") {
						n++
					}
				}
				if n != 1 {
					t.Fatalf("%d former-agent lines, want 1:\n%s", n, joined(r))
				}
			})
		}
	}
}

// (b) both ids at once give one line each.
func TestFormerAgentFilesOneLineEach(t *testing.T) {
	t.Parallel()
	p := claudeBuilt(t, "1.2.3")
	p[".claude/agents/gabo-cycle.md"] = file("---\ngabo-version: 1.2.3\n---\n")
	p[".claude/agents/gabo-reviewer.md"] = file("---\ngabo-version: 1.2.3\n---\n")
	r := run(t, p, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, formerLine(".claude/agents/gabo-cycle.md", "gabo-cycle"), formerLine(".claude/agents/gabo-reviewer.md", "gabo-reviewer"), NextRebuild)
}

// (b) a file without a gabo-version line is the user's own: no line.
func TestFormerAgentFileWithoutVersionLinePasses(t *testing.T) {
	t.Parallel()
	p := claudeBuilt(t, "1.2.3")
	p[".claude/agents/gabo-cycle.md"] = file("---\nname: gabo-cycle\n---\nmine\n")
	p[".claude/agents/gabo-reviewer.md"] = file("mine\n")
	r := run(t, p, configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "no longer in the agent spec")
}

// (b, PD5) a catalog copy that still lists the id as a spec agent keeps the
// agents-exist rule and gives no former-agent line.
func TestFormerAgentLineWaitsForTheCatalogUpgrade(t *testing.T) {
	t.Parallel()
	p := claudeBuilt(t, "1.2.3")
	p[".gabo/skills/gabo-agent-templates/agents.yaml"] = file(strings.Replace(string(p[".gabo/skills/gabo-agent-templates/agents.yaml"].Data),
		"agents:\n", "agents:\n  - id: gabo-cycle\n    role: old\n    tools: [read]\n    launched_by: main-session\n    template: agents/gabo-coder.md\n", 1))
	p[".claude/agents/gabo-cycle.md"] = file("---\nname: gabo-cycle\ngabo-version: 1.2.3\n---\nbody\n")
	r := run(t, p, configFiles{projectFile: validConfig}, Settings{})
	noLineWith(t, r, "no longer in the agent spec")
}

// (b) GABO_UPGRADE_PR=1 validates only the config: no agent line.
func TestFormerAgentLineSkippedInUpgradePR(t *testing.T) {
	t.Parallel()
	p := claudeBuilt(t, "1.2.3")
	p[".claude/agents/gabo-cycle.md"] = file("---\ngabo-version: 1.2.3\n---\n")
	r := run(t, p, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	noLineWith(t, r, "no longer in the agent spec")
}
