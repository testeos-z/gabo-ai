package initcmd

import (
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// readLog records every path the engine step reads, so a test can assert the
// step read nothing outside the project (intent 0028 S8, PRD R32).
type readLog struct {
	*memStore
	reads []string
}

func (r *readLog) ReadFile(name string) ([]byte, error) {
	r.reads = append(r.reads, name)
	return r.memStore.ReadFile(name)
}

func credentialStep(t *testing.T, s *memStore, assistants []string, set ...string) (engineconfig.Report, *readLog) {
	t.Helper()
	rec := &readLog{memStore: s}
	r := NewUseCase(rec, catalog, Settings{ProjectDir: root, Version: "0.2.6"}).engineStep(set, assistants, "manual", "0.2.6")
	return r, rec
}

const (
	credCodexFile    = root + "/.codex/config.toml"
	credOpencodeFile = root + "/.opencode/opencode.json"
	credOpencodeRoot = root + "/opencode.json"
)

// Every engine that has a native deny format gets its base list at project
// level; pi has none and says so (intent 0028 S8, S14).
func TestCredentialDenyInitWritesEachEngineList(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{credOpencodeRoot: "{}\n"})
	r, rec := credentialStep(t, s, nil, "claude", "codex", "opencode", "pi")
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	if !strings.Contains(s.Files[claudeFile], `"Read(**/.env)"`) || !strings.Contains(s.Files[claudeFile], `"Edit(~/.ssh/**)"`) {
		t.Errorf("claude list missing:\n%s", s.Files[claudeFile])
	}
	if !strings.Contains(s.Files[credCodexFile], `default_permissions = "gabo-credentials"`) {
		t.Errorf("codex profile missing:\n%s", s.Files[credCodexFile])
	}
	if !strings.Contains(s.Files[credOpencodeRoot], `"resource": "*.env"`) {
		t.Errorf("opencode rules missing:\n%s", s.Files[credOpencodeRoot])
	}
	if _, ok := s.Files[credOpencodeFile]; ok {
		t.Errorf("created %s for a role engine that is not an assistant", credOpencodeFile)
	}
	if !strings.Contains(r.Render(), "not run: ") {
		t.Errorf("pi has no not run line:\n%s", r.Render())
	}
	for _, p := range rec.reads {
		if !strings.HasPrefix(p, root+"/") {
			t.Errorf("read outside the project: %s", p)
		}
	}
	for _, w := range s.Writes {
		if !strings.HasPrefix(w, root+"/") {
			t.Errorf("write outside the project: %s", w)
		}
	}
}

// A project's own deny list is kept as it is, and a second run writes nothing
// (intent 0028 S8).
func TestCredentialDenyInitKeepsAProjectsOwnList(t *testing.T) {
	t.Parallel()
	src := "{\n  \"permissions\": {\n    \"deny\": [\"Read(./mine.txt)\"]\n  }\n}\n"
	s := newStore(map[string]string{claudeFile: src})
	credentialStep(t, s, nil, "claude")
	if got := s.Files[claudeFile]; !strings.Contains(got, `"Read(./mine.txt)"`) || strings.Contains(got, `"Read(**/.env)"`) {
		t.Fatalf("first run changed the project's own list:\n%s", got)
	}
	s.Writes = nil
	before := s.Files[claudeFile]
	credentialStep(t, s, nil, "claude")
	if len(s.Writes) != 0 || s.Files[claudeFile] != before {
		t.Errorf("second run wrote %v", s.Writes)
	}
}

// With OpenCode listed as the assistant, the gabo default_agent and the deny
// rules land in the same config, and the project's comment stays.
func TestCredentialDenyInitOpencodeAssistantAndDenyTogether(t *testing.T) {
	t.Parallel()
	src := "{\n  // mine\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n"
	s := newStore(map[string]string{root + "/opencode.json": src})
	r, _ := credentialStep(t, s, []string{"opencode"}, "opencode")
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	out := s.Files[root+"/opencode.json"]
	for _, want := range []string{"// mine", `"default_agent": "gabo-assistant"`, `"resource": "~/.ssh/*"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if _, ok := s.Files[credOpencodeFile]; ok {
		t.Errorf("created %s next to the existing opencode.json", credOpencodeFile)
	}
}
