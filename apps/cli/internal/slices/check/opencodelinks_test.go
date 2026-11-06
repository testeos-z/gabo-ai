package check

import (
	"io/fs"
	"path"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// opencodeLinkedByRules is opencodeBuilt with the real catalog
// gabo-runtime-opencode skill and the shared .agents/skills rebuilt by that
// skill's link rules (intent 0004 S5, S14; intent 0008 S3): each spec skill
// as .agents/skills/<name> -> ../../.gabo/skills/<name>, each SKILL.md slot
// adapter as .agents/skills/gabo-<slot> (_ written -), and the
// reviewer-configuration skill like a spec skill; no gabo-reviewer link.
func opencodeLinkedByRules(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := opencodeBuilt(t, version)
	doc, err := fs.ReadFile(skills.Catalog(), "gabo-runtime-opencode/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	proj[CatalogDir+"/gabo-runtime-opencode/SKILL.md"] = &fstest.MapFile{Data: doc}
	for p := range proj {
		if strings.HasPrefix(p, ".agents/skills/") {
			delete(proj, p)
		}
	}
	const dir = ".agents/skills/"
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		for _, name := range a.Skills {
			proj[dir+name] = symlink("../../.gabo/skills/" + name)
		}
	}
	var cfg struct {
		Slots map[string]struct {
			Adapter string `yaml:"adapter"`
		} `yaml:"slots"`
	}
	if err := yaml.Unmarshal([]byte(readTestdata(t, "gabo.yaml")), &cfg); err != nil {
		t.Fatal(err)
	}
	for slot, s := range cfg.Slots {
		if path.Base(s.Adapter) != "SKILL.md" {
			continue
		}
		proj[dir+"gabo-"+strings.ReplaceAll(slot, "_", "-")] = symlink("../../" + path.Dir(s.Adapter))
	}
	proj[dir+"gabo-reviewer-config"] = symlink("../../.gabo/skills/gabo-reviewer-config")
	return proj
}

// S5, S14: a tree linked by the skill's rules, the reviewer-configuration
// link included, gives no ✗ skill link line for .agents/skills, no reviewer
// isolation line for .agents/skills/gabo-reviewer-config (intent 0008 S8),
// and holds no .agents/skills/gabo-reviewer.
func TestOpencodeLinkRulesPassCheck(t *testing.T) {
	t.Parallel()
	proj := opencodeLinkedByRules(t, "1.2.3")
	if _, err := fs.Stat(proj, ".agents/skills/gabo-reviewer-config/SKILL.md"); err != nil {
		t.Fatalf("gabo-reviewer-config link does not resolve: %v", err)
	}
	if _, err := fs.Lstat(proj, ".agents/skills/gabo-reviewer"); err == nil {
		t.Fatal(".agents/skills/gabo-reviewer exists in the project")
	}
	rep := runOpencode(t, proj, "1.2.3")
	noLineWith(t, rep, "✗ skill link")
	noLineWith(t, rep, "must not be in the project")
	noLineWith(t, rep, "gabo link left")
	wantPassed(t, rep, PassedLine)
}
