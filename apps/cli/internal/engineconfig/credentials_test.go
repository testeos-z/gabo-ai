package engineconfig

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// The base deny list of intent 0028 S8, in each engine's native form. The
// forms are pinned from the official docs (see the PR body): Claude Code
// permissions.deny Read(...) and Edit(...) rules, OpenCode V2 permissions
// rules of read and edit with effect deny, and Codex permission profile
// filesystem entries set to deny.
var wantClaudeDeny = []string{
	"Read(**/.env)", "Edit(**/.env)",
	"Read(**/.env.*)", "Edit(**/.env.*)",
	"Read(**/*.pem)", "Edit(**/*.pem)",
	"Read(**/*.key)", "Edit(**/*.key)",
	"Read(**/secrets/**)", "Edit(**/secrets/**)",
	"Read(~/.ssh/**)", "Edit(~/.ssh/**)",
	"Read(~/.aws/credentials)", "Edit(~/.aws/credentials)",
	"Read(~/.config/gh/hosts.yml)", "Edit(~/.config/gh/hosts.yml)",
	"Read(~/Library/Keychains/**)", "Edit(~/Library/Keychains/**)",
	"Read(~/.credentials/**)", "Edit(~/.credentials/**)",
}

var wantOpencodeResources = []string{
	"*.env", "*.env.*", "*.pem", "*.key", "*secrets/*",
	"~/.ssh/*", "~/.aws/credentials", "~/.config/gh/hosts.yml",
	"~/Library/Keychains/*", "~/.credentials/*",
}

var wantCodexWorkspaceDeny = []string{"**/.env", "**/.env.*", "**/*.pem", "**/*.key", "**/secrets/**"}

var wantCodexHomeDeny = []string{"~/.ssh", "~/.aws/credentials", "~/.config/gh/hosts.yml", "~/Library/Keychains", "~/.credentials"}

func decodeJSONMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, b)
	}
	return doc
}

func claudeDenyOf(t *testing.T, b []byte) []string {
	t.Helper()
	perm, ok := decodeJSONMap(t, b)["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions is not an object in\n%s", b)
	}
	list, _ := perm["deny"].([]any)
	var out []string
	for _, v := range list {
		s, _ := v.(string)
		out = append(out, s)
	}
	return out
}

// Claude Code: an empty project gets one Read and one Edit deny per base path.
func TestCredentialDenyClaudeWritesEveryBasePath(t *testing.T) {
	res := claudeRule(Input{Files: map[string][]byte{}, Version: "0.2.6"})
	got := claudeDenyOf(t, res.Writes[ClaudeSettings])
	if !slices.Equal(got, wantClaudeDeny) {
		t.Errorf("permissions.deny =\n%q\nwant\n%q", got, wantClaudeDeny)
	}
}

// A project's own deny list is kept as it is: no base entry is added to it, so
// an entry the project removed stays removed (intent 0028 S8).
func TestCredentialDenyClaudeKeepsTheProjectsOwnEntries(t *testing.T) {
	src := []byte("{\n  \"permissions\": {\n    \"allow\": [\"Bash(go test *)\"],\n    \"deny\": [\"Bash(curl *)\", \"Read(./custom.txt)\"]\n  }\n}\n")
	res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: src}, Version: "0.2.6"})
	got := res.Writes[ClaudeSettings]
	if deny := claudeDenyOf(t, got); !slices.Equal(deny, []string{"Bash(curl *)", "Read(./custom.txt)"}) {
		t.Errorf("permissions.deny = %q, want the project's own list only", deny)
	}
	if allow := anyStrings(decodeJSONMap(t, got)["permissions"].(map[string]any)["allow"]); !slices.Equal(allow, []string{"Bash(go test *)"}) {
		t.Errorf("permissions.allow changed: %q", allow)
	}
	if !slices.Contains(res.Section.Lines, "kept .claude/settings.json permissions.deny (your setting; gabo leaves it)") {
		t.Errorf("lines = %q, want the kept line", res.Section.Lines)
	}
}

// An entry the project removed from the base list stays removed on the next
// run (intent 0028 S8: a project adds or removes entries).
func TestCredentialDenyClaudeKeepsARemovedEntryRemoved(t *testing.T) {
	var list []string
	for _, e := range wantClaudeDeny {
		if e != "Read(**/.env)" && e != "Edit(**/.env)" {
			list = append(list, e)
		}
	}
	src, err := json.Marshal(map[string]any{"permissions": map[string]any{"deny": list}})
	if err != nil {
		t.Fatal(err)
	}
	res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: src}, Version: "0.2.6"})
	if deny := claudeDenyOf(t, res.Writes[ClaudeSettings]); slices.Contains(deny, "Read(**/.env)") || !slices.Equal(deny, list) {
		t.Errorf("permissions.deny = %q, want the reduced list without the removed entries", deny)
	}
}

// A second run on the output writes nothing, so a re-run keeps the list as is.
func TestCredentialDenyClaudeSecondRunWritesNothing(t *testing.T) {
	first := claudeRule(Input{Files: map[string][]byte{}, Version: "0.2.6"}).Writes[ClaudeSettings]
	again := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: first}, Version: "0.2.6"})
	if len(again.Writes) != 0 {
		t.Errorf("second run writes %q, want none", again.Writes)
	}
}

// A permissions.deny that is not an array is one failure line and no write.
func TestCredentialDenyClaudeRefusesADenyThatIsNotAnArray(t *testing.T) {
	src := []byte(`{"permissions": {"deny": "Read(./.env)"}}`)
	res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: src}, Version: "0.2.6"})
	if len(res.Writes) != 0 {
		t.Errorf("writes %q, want none", res.Writes)
	}
	if !slices.ContainsFunc(res.Section.Lines, func(l string) bool {
		return strings.HasPrefix(l, FailurePrefix) && strings.Contains(l, "permissions.deny is not an array")
	}) {
		t.Errorf("lines = %q, want a failure line for permissions.deny", res.Section.Lines)
	}
}

// tomlTable returns the table at key in m, failing the test when it is not one.
func tomlTable(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	tab, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not a table: %v", key, m[key])
	}
	return tab
}

func anyStrings(v any) []string {
	list, _ := v.([]any)
	var out []string
	for _, e := range list {
		s, _ := e.(string)
		out = append(out, s)
	}
	return out
}

type opencodePerm struct {
	Action, Resource, Effect string
}

func opencodeRulesOf(t *testing.T, b []byte) []opencodePerm {
	t.Helper()
	plain, err := blankJSONC(b)
	if err != nil {
		t.Fatalf("not JSONC: %v", err)
	}
	list, _ := decodeJSONMap(t, plain)["permissions"].([]any)
	var out []opencodePerm
	for _, e := range list {
		m, _ := e.(map[string]any)
		r := opencodePerm{Action: str(m["action"]), Resource: str(m["resource"]), Effect: str(m["effect"])}
		out = append(out, r)
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// OpenCode: an empty project gets a read and an edit deny rule per base path,
// appended after the base policy, so the last match is gabo's deny.
func TestCredentialDenyOpencodeWritesReadAndEditDenyRules(t *testing.T) {
	res := opencodeRule(Input{Files: map[string][]byte{}, Version: "0.2.6", Assistants: []string{Opencode}})
	got := opencodeRulesOf(t, res.Writes[OpencodeNewConfig])
	var want []opencodePerm
	for _, r := range wantOpencodeResources {
		want = append(want, opencodePerm{"read", r, "deny"}, opencodePerm{"edit", r, "deny"})
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rules, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rule %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// OpenCode: a permissions array the project has is its own and is kept; no
// base rule is added to it, and its comments stay.
func TestCredentialDenyOpencodeKeepsTheProjectsRules(t *testing.T) {
	src := "{\n  // project\n  \"permissions\": [\n    { \"action\": \"read\", \"resource\": \"*.env.example\", \"effect\": \"allow\" }\n  ]\n}\n"
	res := opencodeRule(Input{Files: map[string][]byte{"opencode.json": []byte(src)}, Version: "0.2.6"})
	if len(res.Writes) != 0 {
		t.Errorf("writes = %q, want none", res.Writes)
	}
	if !slices.Contains(res.Section.Lines, "kept opencode.json permissions (your setting; gabo leaves it)") {
		t.Errorf("lines = %q, want the kept line", res.Section.Lines)
	}
}

// Codex: an empty project gets the active profile and its deny entries; the
// profile extends :workspace, and the workspace-root globs are bounded.
func TestCredentialDenyCodexWritesTheActiveProfile(t *testing.T) {
	res := codexRule(Input{Files: map[string][]byte{}, Version: "0.2.6"})
	var doc map[string]any
	if err := toml.Unmarshal(res.Writes[CodexConfigPath], &doc); err != nil {
		t.Fatalf("not TOML: %v\n%s", err, res.Writes[CodexConfigPath])
	}
	if doc["default_permissions"] != "gabo-credentials" {
		t.Errorf("default_permissions = %v", doc["default_permissions"])
	}
	profile := tomlTable(t, tomlTable(t, doc, "permissions"), "gabo-credentials")
	if profile["extends"] != ":workspace" {
		t.Errorf("extends = %v, want :workspace", profile["extends"])
	}
	fs := tomlTable(t, profile, "filesystem")
	if fs["glob_scan_max_depth"] != int64(5) {
		t.Errorf("glob_scan_max_depth = %v, want 5", fs["glob_scan_max_depth"])
	}
	for _, p := range wantCodexHomeDeny {
		if fs[p] != "deny" {
			t.Errorf("filesystem[%q] = %v, want deny", p, fs[p])
		}
	}
	ws := tomlTable(t, fs, ":workspace_roots")
	for _, p := range wantCodexWorkspaceDeny {
		if ws[p] != "deny" {
			t.Errorf(":workspace_roots[%q] = %v, want deny", p, ws[p])
		}
	}
}

// Codex: a second run writes nothing, and the project's own deny entry stays.
func TestCredentialDenyCodexKeepsTheProjectsOwnEntries(t *testing.T) {
	src := []byte("default_permissions = \"gabo-credentials\"\n\n[permissions.gabo-credentials.filesystem.\":workspace_roots\"]\n\"secrets.local\" = \"deny\"\n")
	res := codexRule(Input{Files: map[string][]byte{CodexConfigPath: src}, Version: "0.2.6"})
	out := res.Writes[CodexConfigPath]
	var doc map[string]any
	if err := toml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("not TOML: %v", err)
	}
	ws := tomlTable(t, tomlTable(t, tomlTable(t, tomlTable(t, doc, "permissions"), "gabo-credentials"), "filesystem"), ":workspace_roots")
	if ws["secrets.local"] != "deny" {
		t.Errorf("the project's own entry was lost: %v", ws)
	}
	if _, added := ws["**/.env"]; added {
		t.Errorf("base entries were added to the project's own profile: %v", ws)
	}
	again := codexRule(Input{Files: map[string][]byte{CodexConfigPath: out}, Version: "0.2.6"})
	if len(again.Writes) != 0 {
		t.Errorf("second run writes %q, want none", again.Writes)
	}
}

// Codex: a sandbox_mode or another default profile makes the profile inert,
// so gabo writes nothing and says why in a line that is not a failure.
func TestCredentialDenyCodexDoesNotOverrideAnotherSandboxChoice(t *testing.T) {
	cases := map[string]string{
		"sandbox_mode":  "sandbox_mode = \"workspace-write\"\n",
		"other default": "default_permissions = \":read-only\"\n",
		"sandbox table": "[sandbox_workspace_write]\nnetwork_access = false\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			res := codexRule(Input{Files: map[string][]byte{CodexConfigPath: []byte(src)}, Version: "0.2.6"})
			if out, ok := res.Writes[CodexConfigPath]; ok {
				if strings.Contains(string(out), "gabo-credentials") {
					t.Errorf("wrote the gabo profile:\n%s", out)
				}
			}
			if !slices.ContainsFunc(res.Section.Lines, func(l string) bool {
				return strings.HasPrefix(l, "not applied: credential deny list")
			}) {
				t.Errorf("lines = %q, want a not applied credential line", res.Section.Lines)
			}
			for _, l := range res.Section.Lines {
				if strings.HasPrefix(l, FailurePrefix) {
					t.Errorf("a failure line for a valid config: %q", l)
				}
			}
		})
	}
}

// Pi has no native deny format: the section says so and nothing about
// credentials is written (intent 0028 S14).
func TestPiCredentialNotRun(t *testing.T) {
	res := piRule(Input{Files: map[string][]byte{}, Version: "0.2.6", PiPackage: true})
	if !slices.ContainsFunc(res.Section.Lines, func(l string) bool {
		return strings.HasPrefix(l, "not run: ") && strings.Contains(l, "no native deny format")
	}) {
		t.Errorf("lines = %q, want a not run credential line", res.Section.Lines)
	}
	for path := range res.Writes {
		if path != PiSettings {
			t.Errorf("pi wrote %s, want only %s", path, PiSettings)
		}
	}
	if strings.Contains(string(res.Writes[PiSettings]), "deny") {
		t.Errorf("pi settings hold a deny list:\n%s", res.Writes[PiSettings])
	}
}

// OpenCode as a role engine that is not a listed assistant: gabo creates no
// config for it (intent 0015 S8), so the list is written only into an
// existing config, and with none the section says so.
func TestCredentialDenyOpencodeNeedsAConfigUnlessAssistant(t *testing.T) {
	res := opencodeRule(Input{Files: map[string][]byte{}, Version: "0.2.6"})
	if len(res.Writes) != 0 {
		t.Errorf("writes = %q, want none", res.Writes)
	}
	if !slices.Contains(res.Section.Lines, opencodeNoConfigNotRun) {
		t.Errorf("lines = %q, want the not run line", res.Section.Lines)
	}
	existing := opencodeRule(Input{Files: map[string][]byte{"opencode.json": []byte("{}\n")}, Version: "0.2.6"})
	if len(opencodeRulesOf(t, existing.Writes["opencode.json"])) == 0 {
		t.Errorf("existing config got no deny rules: %q", existing.Writes)
	}
}
