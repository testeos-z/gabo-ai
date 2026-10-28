package check

import (
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

const (
	userFile    = "/home/o/.config/gabo/config.yaml"
	projectFile = "/src/p/gabo.yaml"
)

// validConfig sets every required key; both runtimes are claude.
const validConfig = `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
`

// spec lists two agents; the project's catalog copy is the only source.
const spec = `agents:
  - id: alpha
    role: first
    tools: [read]
    launched_by: main-session
    template: agents/alpha.md
  - id: beta
    role: second
    tools: [read]
    launched_by: alpha
    template: agents/beta.md
`

const claudeSkill = "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\nbody\n"

// configFiles is an in-memory config.Reader.
type configFiles map[string]string

func (c configFiles) ReadFile(name string) ([]byte, error) {
	s, ok := c[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return []byte(s), nil
}

type failingReader struct{}

func (failingReader) ReadFile(string) ([]byte, error) { return nil, errors.New("permission denied") }

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

// project is a passing project at catalog version 1.2.3: stamp, spec,
// Claude Code runtime skill and both agents built with 1.2.3.
func project() fstest.MapFS {
	return fstest.MapFS{
		".gabo/VERSION": file("1.2.3\n"),
		".gabo/skills/gabo-agent-templates/agents.yaml":     file(spec),
		".gabo/skills/gabo-agent-templates/agents/alpha.md": file("gabo-version: {{gabo_version}}\n"),
		".gabo/skills/gabo-agent-templates/agents/beta.md":  file("gabo-version: {{gabo_version}}\n"),
		".gabo/skills/gabo-runtime-claude/SKILL.md":         file(claudeSkill),
		".claude/agents/alpha.md":                           file("---\nname: alpha\ngabo-version: 1.2.3\n---\nbody\n"),
		".claude/agents/beta.md":                            file("---\nname: beta\ngabo-version: 1.2.3\n---\nbody\n"),
	}
}

func layers() []config.Layer {
	return []config.Layer{{Name: config.LayerUser, Path: userFile}, {Name: config.LayerProject, Path: projectFile}}
}

func run(t *testing.T, proj fs.FS, cfg config.Reader, s Settings) Report {
	t.Helper()
	if s.Version == "" {
		s.Version = "1.2.3"
	}
	s.Layers = layers()
	rep, err := NewUseCase(proj, cfg, s).Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return rep
}

func with(base fstest.MapFS, edits map[string]string) fstest.MapFS {
	out := maps.Clone(base)
	for k, v := range edits {
		if v == "" {
			delete(out, k)
			continue
		}
		out[k] = file(v)
	}
	return out
}

func joined(r Report) string { return strings.Join(r.Lines, "\n") }

func wantFailed(t *testing.T, r Report, lines ...string) {
	t.Helper()
	if !r.Failed {
		t.Fatalf("Failed = false, want true; output:\n%s", joined(r))
	}
	wantLines(t, r, lines...)
}

func wantPassed(t *testing.T, r Report, lines ...string) {
	t.Helper()
	if r.Failed {
		t.Fatalf("Failed = true, want false; output:\n%s", joined(r))
	}
	wantLines(t, r, lines...)
}

func wantLines(t *testing.T, r Report, lines ...string) {
	t.Helper()
	for _, l := range lines {
		if !slices.Contains(r.Lines, l) {
			t.Fatalf("output lacks line %q:\n%s", l, joined(r))
		}
	}
}

func TestPassingProject(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

func TestMissingProjectConfig(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{userFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ no gabo.yaml found", "next: run gabo init, then ask your agent to configure gabo-ai")
}

// A config still under its name from before the gabo-ai rename (TRD D26)
// fails with the rename to make; it is never read.
func TestFormerConfigNamesFailWithTheRename(t *testing.T) {
	t.Parallel()
	former := []config.FormerName{
		{Probe: "/home/o/.config/gbai/config.yaml", From: "/home/o/.config/gbai", To: "/home/o/.config/gabo"},
		{Probe: "/src/p/gbai.yaml", From: "/src/p/gbai.yaml", To: "/src/p/gabo.yaml"},
	}
	for _, upgrade := range []bool{false, true} {
		r := run(t, project(), configFiles{
			projectFile:                        validConfig,
			"/src/p/gbai.yaml":                 "parallelism: 0\n",
			"/home/o/.config/gbai/config.yaml": "interaction_level: loud\n",
		}, Settings{Former: former, UpgradePR: upgrade})
		wantFailed(t, r,
			"✗ /home/o/.config/gbai/config.yaml uses a name from before the gabo-ai rename and is not read",
			"next: rename /home/o/.config/gbai to /home/o/.config/gabo, then rerun gabo check",
			"✗ /src/p/gbai.yaml uses a name from before the gabo-ai rename and is not read",
			"next: rename /src/p/gbai.yaml to /src/p/gabo.yaml, then rerun gabo check")
		if strings.Contains(joined(r), "parallelism") || strings.Contains(joined(r), "interaction_level") {
			t.Fatalf("a former file was read:\n%s", joined(r))
		}
	}
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{Former: former})
	wantPassed(t, r, PassedLine)
	_, err := NewUseCase(project(), failingReader{}, Settings{Version: "1.2.3", Layers: layers(), Former: former}).Run()
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want the read error", err)
	}
}

func TestConfigFailureNamesKeyFileAndNext(t *testing.T) {
	t.Parallel()
	bad := strings.Replace(validConfig, "parallelism: 3", "parallelism: 0", 1)
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	if !r.Failed {
		t.Fatalf("Failed = false:\n%s", joined(r))
	}
	if !strings.Contains(joined(r), "✗ "+projectFile+": parallelism: ") {
		t.Fatalf("no key path and file in:\n%s", joined(r))
	}
	wantLines(t, r, NextFixConfig)
}

func TestFormerCloudKeyFailsNamingAgentHost(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig + "server:\n  host: x\n"}, Settings{})
	if !r.Failed || !strings.Contains(joined(r), "✗ "+projectFile+": server.host: ") || !strings.Contains(joined(r), "slots.agent_host") {
		t.Fatalf("output:\n%s", joined(r))
	}
}

func TestPersonalKeyInProjectFileWarns(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig + "tone: dry\n"}, Settings{})
	wantPassed(t, r)
	if !strings.Contains(joined(r), "! "+projectFile+`: personal key "tone" is ignored`) {
		t.Fatalf("no warning:\n%s", joined(r))
	}
}

func TestUserFileIsValidated(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{userFile: "interaction_level: loud\n", projectFile: validConfig}, Settings{})
	if !r.Failed || !strings.Contains(joined(r), "✗ "+userFile+": interaction_level: ") {
		t.Fatalf("output:\n%s", joined(r))
	}
}

func TestConfigReadErrorIsAnError(t *testing.T) {
	t.Parallel()
	_, err := NewUseCase(project(), failingReader{}, Settings{Version: "1.2.3", Layers: layers()}).Run()
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want the read error", err)
	}
}

func TestMarkdownAndTOMLAgentFilesCount(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{
		".claude/agents/beta.md":   "",
		".claude/agents/beta.toml": "# gabo-version: 1.2.3\nname = \"beta\"\n",
	})
	wantPassed(t, run(t, proj, configFiles{projectFile: validConfig}, Settings{}))
}

func TestMissingAgent(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{".claude/agents/beta.md": ""})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ agent beta missing", NextBuildAgent)
}

// Only the declared directory counts: a same-named doc, test fixture or
// catalog template elsewhere, or a file in a subdirectory, is not the agent.
func TestSameNamedFileOutsideTheAgentDirDoesNotCount(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{
		".claude/agents/beta.md":        "",
		"docs/beta.md":                  "gabo-version: 1.2.3\n",
		"testdata/beta.toml":            "# gabo-version: 1.2.3\n",
		".claude/agents/nested/beta.md": "gabo-version: 1.2.3\n",
		".claude/beta.md":               "gabo-version: 1.2.3\n",
		".gabo/skills/x/agents/beta.md": "gabo-version: 1.2.3\n",
	})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ agent beta missing")
}

func TestRuntimeWithoutSkillNamesTheBuilder(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{".gabo/skills/gabo-runtime-claude/SKILL.md": ""})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ runtime claude has no runtime skill gabo-runtime-claude", NextRuntimeBuilder)
}

func TestRuntimeSkillWithoutAgentDirNamesTheBuilder(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nname: gabo-runtime-claude\n---\n"})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ runtime claude: gabo-runtime-claude declares no metadata.agent_dir", NextRuntimeBuilder)
}

// A runtime skill written by the runtime builder lives in .gabo/runtimes/.
func TestBuilderWrittenRuntimeSkill(t *testing.T) {
	t.Parallel()
	pi := strings.ReplaceAll(validConfig, "reviewer: claude", "reviewer: pi") + "  pi_credential_env: ANTHROPIC_API_KEY\n"
	proj := with(project(), map[string]string{
		".gabo/runtimes/gabo-runtime-pi/SKILL.md": "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n---\n",
		".pi/agents/alpha.md":                     "---\ngabo-version: 1.2.3\n---\n",
	})
	r := run(t, proj, configFiles{projectFile: pi}, Settings{})
	wantFailed(t, r, "✗ agent beta missing", "why: no file named beta in .pi/agents (runtime pi)")
	if strings.Contains(joined(r), "✗ agent alpha") {
		t.Fatalf("alpha is built for pi:\n%s", joined(r))
	}
}

func TestAgentDirMustStayInsideTheRepository(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"/etc/agents", "../agents", ".claude/../../agents"} {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			proj := with(project(), map[string]string{
				".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nmetadata:\n  agent_dir: " + dir + "\n---\n",
			})
			r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
			wantFailed(t, r, "✗ runtime claude: agent_dir must stay inside the repository")
		})
	}
}

func TestAgentVersionLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, content, want string
	}{
		{"matching", "---\ngabo-version: 1.2.3\n---\n", ""},
		{"differing", "---\ngabo-version: 1.2.0\n---\n", "✗ agent beta built with 1.2.0, catalog is 1.2.3: ask your agent to rebuild"},
		{"missing", "---\nname: beta\n---\n", "✗ agent beta built with none, catalog is 1.2.3: ask your agent to rebuild"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj := with(project(), map[string]string{".claude/agents/beta.md": tt.content})
			r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
			if tt.want == "" {
				wantPassed(t, r)
				return
			}
			wantFailed(t, r, tt.want, NextRebuild)
		})
	}
}

func TestCatalogVersionStamp(t *testing.T) {
	t.Parallel()
	older := with(project(), map[string]string{".gabo/VERSION": "1.0.0\n"})
	for _, p := range []string{".claude/agents/alpha.md", ".claude/agents/beta.md"} {
		older[p] = file("gabo-version: 1.0.0\n")
	}
	tests := []struct {
		name   string
		proj   fstest.MapFS
		failed bool
		lines  []string
	}{
		{"older warns", older, false, []string{"! this project uses gabo 1.0.0; run gabo init to upgrade it"}},
		{"missing fails", with(project(), map[string]string{".gabo/VERSION": ""}), true, []string{"✗ no stamp: run gabo init"}},
		{"newer fails", with(project(), map[string]string{".gabo/VERSION": "2.0.0\n"}), true, []string{
			"✗ catalog version 2.0.0 is newer than gabo 1.2.3",
			"next: run go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v2.0.0",
		}},
		{"not a version fails", with(project(), map[string]string{".gabo/VERSION": "latest\n"}), true, []string{
			`✗ .gabo/VERSION holds "latest", not a version x.y.z`,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := run(t, tt.proj, configFiles{projectFile: validConfig}, Settings{})
			if r.Failed != tt.failed {
				t.Fatalf("Failed = %v, want %v:\n%s", r.Failed, tt.failed, joined(r))
			}
			wantLines(t, r, tt.lines...)
		})
	}
}

// A CLI built from source has no release version: the stamp is not compared.
func TestNonReleaseCLIVersionSkipsTheComparison(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{Version: "(devel)"})
	wantPassed(t, r, "! gabo (devel) is not a release version x.y.z; catalog version 1.2.3 not compared")
}

func TestVPrefixedCLIVersion(t *testing.T) {
	t.Parallel()
	wantPassed(t, run(t, project(), configFiles{projectFile: validConfig}, Settings{Version: "v1.2.3"}))
}

// A newer CLI checks an older project against the project's own spec: an
// agent only a newer spec has is not required.
func TestNewerCLIUsesTheProjectSpec(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{Version: "1.3.0"})
	wantPassed(t, r, "! this project uses gabo 1.2.3; run gabo init to upgrade it")
}

func TestMissingSpecFails(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{".gabo/skills/gabo-agent-templates/agents.yaml": ""})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	if !r.Failed || !strings.Contains(joined(r), "✗ agent spec: ") {
		t.Fatalf("output:\n%s", joined(r))
	}
}

func TestUpgradePRValidatesOnlyTheConfig(t *testing.T) {
	t.Parallel()
	broken := fstest.MapFS{".gabo/VERSION": file("9.9.9\n")}
	r := run(t, broken, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	wantPassed(t, r, UpgradePRLine)
	if len(r.Lines) != 1 {
		t.Fatalf("upgrade PR ran more than the config check:\n%s", joined(r))
	}
	bad := strings.Replace(validConfig, "parallelism: 3", "parallelism: 0", 1)
	r = run(t, broken, configFiles{projectFile: bad}, Settings{UpgradePR: true})
	wantFailed(t, r, UpgradePRLine)
}

// slotConfig wires the tasks and methodology slots to adapters in the repo.
const slotConfig = validConfig + `slots:
  tasks:
    adapter: docs/adapters/tasks.md
  methodology:
    adapter: docs/adapters/methodology.md
`

const methodologyContract = "---\nname: gabo-contract-methodology\n---\n## Operations\n\n### coder instructions\n\n### report per task\n"

// slotProject adds both contracts to the catalog copy and an adapter
// covering each.
func slotProject() fstest.MapFS {
	return with(project(), map[string]string{
		".gabo/skills/gabo-contract-tasks/SKILL.md":       tasksContract,
		".gabo/skills/gabo-contract-methodology/SKILL.md": methodologyContract,
		"docs/adapters/tasks.md":                          "## Operations\n\n### create\n\n### read one\n\n### claim\n",
		"docs/adapters/methodology.md":                    "## Operations\n\n### coder instructions\n\n### report per task\n",
	})
}

func TestSlotWithCompleteAdapterPasses(t *testing.T) {
	t.Parallel()
	r := run(t, slotProject(), configFiles{projectFile: slotConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

func TestSlotWithGappedAdapterListsMissingOperations(t *testing.T) {
	t.Parallel()
	proj := with(slotProject(), map[string]string{"docs/adapters/tasks.md": "## Operations\n\n### create\n"})
	r := run(t, proj, configFiles{projectFile: slotConfig}, Settings{})
	wantFailed(t, r,
		"✗ adapter docs/adapters/tasks.md does not cover: read one, claim",
		"why: slots.tasks.adapter, contract "+CatalogDir+"/gabo-contract-tasks/SKILL.md",
		NextCoverAdapter)
	if strings.Contains(joined(r), "methodology.md") {
		t.Fatalf("a complete adapter was reported:\n%s", joined(r))
	}
}

func TestSlotWithMissingAdapterNamesTheSlot(t *testing.T) {
	t.Parallel()
	proj := with(slotProject(), map[string]string{"docs/adapters/methodology.md": ""})
	r := run(t, proj, configFiles{projectFile: slotConfig}, Settings{})
	wantFailed(t, r,
		"✗ slot methodology: adapter docs/adapters/methodology.md not found",
		"next: write it with gabo-adapter-writer or fix slots.methodology.adapter, then rerun gabo check")
}

func TestSlotWithoutContractInTheCatalog(t *testing.T) {
	t.Parallel()
	proj := with(slotProject(), map[string]string{".gabo/skills/gabo-contract-tasks/SKILL.md": ""})
	r := run(t, proj, configFiles{projectFile: slotConfig}, Settings{})
	wantFailed(t, r,
		"✗ slot tasks: contract gabo-contract-tasks missing from "+CatalogDir,
		"next: run gabo init to restore the catalog, then rerun gabo check")
}

// The contract comes from the project's catalog copy: an operation the
// project's contract lacks is not required.
func TestSlotUsesTheProjectContract(t *testing.T) {
	t.Parallel()
	proj := with(slotProject(), map[string]string{
		".gabo/skills/gabo-contract-tasks/SKILL.md": "## Operations\n\n### create\n",
		"docs/adapters/tasks.md":                    "## Operations\n\n### create\n",
	})
	r := run(t, proj, configFiles{projectFile: slotConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

func TestUpgradePRSkipsSlotCoverage(t *testing.T) {
	t.Parallel()
	proj := with(slotProject(), map[string]string{"docs/adapters/tasks.md": ""})
	r := run(t, proj, configFiles{projectFile: slotConfig}, Settings{UpgradePR: true})
	wantPassed(t, r, UpgradePRLine)
}

// A methodology adapter may be a bare skill name in the catalog copy.
func TestMethodologyAdapterAsSkillName(t *testing.T) {
	t.Parallel()
	cfg := strings.Replace(slotConfig, "adapter: docs/adapters/methodology.md", "adapter: gabo-methodology-tdd", 1)
	proj := with(slotProject(), map[string]string{
		".gabo/skills/gabo-methodology-tdd/SKILL.md": "## Operations\n\n### coder instructions\n",
	})
	r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
	wantFailed(t, r, "✗ adapter "+CatalogDir+"/gabo-methodology-tdd/SKILL.md does not cover: report per task")
}
