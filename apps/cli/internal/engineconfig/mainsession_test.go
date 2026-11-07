package engineconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// mainSessionKeys are the main-session keys gabo init writes, per engine
// with a main-session rule (intent 0010 S1, S2, D1).
var mainSessionKeys = map[string]string{
	Claude:   claudeAgentKey,
	Opencode: opencodeAgentKey,
}

// mainSessionMeta is metadata.main_session of a runtime skill (spec 0009
// D5), parsed minimally: the check slice's reader is unexported.
type mainSessionMeta struct {
	Config []string `yaml:"config"`
	Key    string   `yaml:"key"`
}

// readMainSession reads metadata.main_session from a SKILL.md frontmatter.
func readMainSession(skillMD []byte) (mainSessionMeta, error) {
	rest, ok := bytes.CutPrefix(bytes.ReplaceAll(skillMD, []byte("\r\n"), []byte("\n")), []byte("---\n"))
	if !ok {
		return mainSessionMeta{}, errors.New("missing --- frontmatter")
	}
	front, _, ok := bytes.Cut(rest, []byte("\n---"))
	if !ok {
		return mainSessionMeta{}, errors.New("unterminated frontmatter")
	}
	var fm struct {
		Metadata struct {
			MainSession *mainSessionMeta `yaml:"main_session"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(front, &fm); err != nil {
		return mainSessionMeta{}, err
	}
	if fm.Metadata.MainSession == nil {
		return mainSessionMeta{}, errors.New("no metadata.main_session")
	}
	return *fm.Metadata.MainSession, nil
}

// matchMainSession compares the runtime skill's metadata.main_session with
// the key and config files the engine's gabo init rule uses.
func matchMainSession(engine string, skillMD []byte) error {
	meta, err := readMainSession(skillMD)
	if err != nil {
		return err
	}
	var diffs []string
	if want := mainSessionKeys[engine]; strings.TrimSpace(meta.Key) != want {
		diffs = append(diffs, fmt.Sprintf("key: metadata %q, engineconfig %q", meta.Key, want))
	}
	if want := RemovalFiles(engine); !slices.Equal(meta.Config, want) {
		diffs = append(diffs, fmt.Sprintf("config: metadata %q, engineconfig %q", meta.Config, want))
	}
	if len(diffs) > 0 {
		return errors.New(strings.Join(diffs, "; "))
	}
	return nil
}

// The keys gabo init writes match what the catalog runtime skills declare
// (intent 0010 S7): a changed key or config list on either side fails.
func TestMainSessionKeysMatchRuntimeSkillMetadata(t *testing.T) {
	for _, engine := range []string{Claude, Opencode} {
		t.Run(engine, func(t *testing.T) {
			skill := "gabo-runtime-" + engine + "/SKILL.md"
			b, err := fs.ReadFile(skills.Catalog(), skill)
			if err != nil {
				t.Fatalf("read %s: %v", skill, err)
			}
			if err := matchMainSession(engine, b); err != nil {
				t.Errorf("%s metadata.main_session differs from engineconfig: %v", skill, err)
			}
		})
	}
}

// Every engine with a main-session removal has a key in the comparison, so
// a new main-session engine cannot skip S7.
func TestMainSessionKeysCoverEveryRemovableEngine(t *testing.T) {
	for _, engine := range order {
		if _, ok := mainSessionKeys[engine]; ok != Removable(engine) {
			t.Errorf("%s: in mainSessionKeys %v, Removable %v", engine, ok, Removable(engine))
		}
	}
}

// A test copy of a catalog skill with a changed key or config list is
// reported as a difference (S7 acceptance: given a changed key in one
// runtime skill's metadata then the test fails).
func TestMatchMainSessionReportsAChangedCopy(t *testing.T) {
	b, err := fs.ReadFile(skills.Catalog(), "gabo-runtime-claude/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, old, new, want string
	}{
		{"changed key", "    key: agent\n", "    key: default_agent\n", `key: metadata "default_agent", engineconfig "agent"`},
		{"changed config", "    config: [.claude/settings.json]\n", "    config: [.claude/settings.local.json]\n", `config: metadata ["`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !bytes.Contains(b, []byte(c.old)) {
				t.Fatalf("catalog gabo-runtime-claude has no %q to change", c.old)
			}
			changed := bytes.Replace(b, []byte(c.old), []byte(c.new), 1)
			err := matchMainSession(Claude, changed)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("matchMainSession(changed copy) = %v, want an error containing %q", err, c.want)
			}
		})
	}
}

// Intent 0013 S3: every listed assistant engine gets its own main-session
// key and none removes another; a second run on the output writes nothing.
func TestEveryListedAssistantGetsItsMainSessionKey(t *testing.T) {
	assistants := []string{Claude, Opencode}
	in := Input{Files: map[string][]byte{}, Version: "0.2.6", Assistants: assistants}
	set := Set(Engines("", "", assistants)...)
	report, writes := Integrate(set, in)
	removals, removed := Remove(set, in)
	if report.Failed() || len(removed) != 0 || len(removals.Sections) != 0 {
		t.Fatalf("report:\n%s\nremovals %+v writes %q", report.Render(), removals.Sections, removed)
	}
	for engine, file := range map[string]string{Claude: ClaudeSettings, Opencode: OpencodeNewConfig} {
		if !bytes.Contains(writes[file], []byte(`"`+mainSessionKeys[engine]+`": "gabo-assistant"`)) {
			t.Errorf("%s =\n%s\nwant the %s key", file, writes[file], mainSessionKeys[engine])
		}
	}
	maps.Copy(in.Files, writes)
	if _, again := Integrate(set, in); len(again) != 0 {
		t.Errorf("second run writes %q, want none", again)
	}
}
