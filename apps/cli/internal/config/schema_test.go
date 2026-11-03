package config

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// repoRoot is the repository checkout, three levels above this package's
// module dir (apps/cli).
const repoRoot = "../../../../"

// The embedded schemas are byte-identical copies of schemas/ at the
// repository root, the one source editors and the docs point at.
func TestEmbeddedSchemasMatchRepositorySchemas(t *testing.T) {
	t.Parallel()
	for _, name := range SchemaFileNames {
		want, err := os.ReadFile(repoRoot + "schemas/" + name)
		if err != nil {
			t.Fatalf("read repository schema: %v", err)
		}
		got, err := schemaFS.ReadFile("schemas/" + name)
		if err != nil {
			t.Fatalf("read embedded schema: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: embedded copy differs from schemas/%s; run go generate ./internal/config", name, name)
		}
	}
}

// Both files are draft 2020-12, and every object schema that lists
// properties closes itself with additionalProperties: false or
// unevaluatedProperties: false (strict keys, like the Go decoder); the
// fragments it skips are closed by the schema that references them.
func TestSchemasAreDraft202012AndClosed(t *testing.T) {
	t.Parallel()
	for _, name := range SchemaFileNames {
		doc := readSchemaJSON(t, name)
		if doc["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			t.Errorf("%s: $schema = %v, want draft 2020-12", name, doc["$schema"])
		}
		walkSchema(doc, "", func(path string, s map[string]any) {
			if _, ok := s["properties"]; !ok || isFragment(path) {
				return
			}
			if s["additionalProperties"] != false && s["unevaluatedProperties"] != false {
				t.Errorf("%s %s: object schema is not closed", name, path)
			}
		})
	}
}

// isFragment reports the $defs that only add constraints to an object that
// another schema closes (key groups and the required-key layer), and the
// property constraints nested inside them.
func isFragment(path string) bool {
	if path == "/$defs/projectKeys" || path == "/$defs/personalKeys" {
		return true
	}
	return strings.HasPrefix(path, "/$defs/complete")
}

// R21: no key may hold a secret value; a credential is configured only as the
// name of an environment variable (a key ending in _env with the env name
// pattern).
func TestSchemasHaveNoSecretValueFields(t *testing.T) {
	t.Parallel()
	secret := regexp.MustCompile(`(?i)token|secret|password|passwd|credential|api_?key|private|auth`)
	for _, name := range SchemaFileNames {
		walkSchema(readSchemaJSON(t, name), "", func(path string, s map[string]any) {
			// The required-key layer only restates types of keys defined above.
			if strings.HasPrefix(path, "/$defs/complete") {
				return
			}
			props, _ := s["properties"].(map[string]any)
			for key, v := range props {
				if !secret.MatchString(key) {
					continue
				}
				p, _ := v.(map[string]any)
				if !strings.HasSuffix(key, "_env") || p["pattern"] != "^([A-Z_][A-Z0-9_]*)?$" {
					t.Errorf("%s %s/properties/%s: secret-like key must be an env var name ending in _env", name, path, key)
				}
			}
		})
	}
}

// Parity: the same accept and reject fixtures give the same result through
// the Go validation (Parse) and the file schema.
func TestSchemaAndGoValidationAgree(t *testing.T) {
	t.Parallel()
	accept := map[string]string{
		"minimal":                 minimal,
		"every key":               dbSchemaProjectExample + "alias: Leo\ninteraction_level: high\ntone: brief\nlanguage: es-AR\n",
		"version 0":               with("version: 0"),
		"null optional values":    with("mode:\ncoder:\nworktree:\nslots:\nreviewer:\ndevelopers:\nplan_change:\nversion:\nalias:\nlanguage:\n"),
		"empty optional strings":  with(`{mode: "", coder: {login: ""}, reviewer: {placement: "", signing: {marker: ""}}, plan_change: {scope: ""}, interaction_level: "", language: ""}`),
		"pi coder and assistant":  with("runtimes: {coder: pi, assistant: pi}"),
		"pi reviewer":             with("runtimes: {reviewer: pi}"),
		"pi reviewer with env":    with("runtimes: {reviewer: pi}\nreview: {pi_credential_env: ANTHROPIC_API_KEY}"),
		"claude reviewer env":     with("review: {pi_credential_env: ANTHROPIC_API_KEY}"),
		"bot coder login":         with("coder: {login: 'gabo-coder[bot]'}"),
		"state dir":               with("review: {state_dir: /data/gabo/reviewer}"),
		"keep name and abs path":  with("worktree: {keep: [main, /srv/wt/x]}"),
		"methodology skill name":  with("slots: {methodology: {adapter: gabo-methodology}}"),
		"human only dir and file": with("review: {human_only_paths: [.github/, AGENTS.md, ..x/]}"),
		"auto merge and scope":    with("review: {auto_merge: true}\nplan_change: {scope: global, global_without_owner: true}"),
		"empty developers":        with("developers: {}"),
		"empty keep":              with("worktree: {keep: []}"),
		"spaces in check":         with("review: {checks: ['go -C apps/cli test ./...']}"),
		"check env names":         with("review: {check_env: [AISHARE_TEST_DATABASE_URL, _X1, GHOST, CIRCLE]}"),
		"empty check env":         with("review: {check_env: []}"),
		"wave after next":         with("wave: {after: next}"),
		"wave after wait":         with("wave: {after: wait}"),
		"null wave":               with("wave:\n"),
		"empty wave after":        with(`wave: {after: ""}`),
	}
	reject := map[string]string{
		"missing runtimes.coder":      string(without(t, "runtimes.coder")),
		"missing repo":                string(without(t, "repo")),
		"missing parallelism":         string(without(t, "parallelism")),
		"missing review.checks":       string(without(t, "review.checks")),
		"missing human_only_paths":    string(without(t, "review.human_only_paths")),
		"unknown reviewer runtime":    with("runtimes: {reviewer: codex}"),
		"null parallelism":            with("parallelism: null"),
		"null runtimes.coder":         with("runtimes: {coder: null}"),
		"empty runtimes.coder":        with(`runtimes: {coder: ""}`),
		"unknown runtime":             with("runtimes: {assistant: codex}"),
		"unknown top-level key":       with("colour: blue"),
		"unknown nested key":          with("repo: {url: x}"),
		"former cloud key":            with("server: {railway: {project_id: x}}"),
		"generation key":              with("generation: {targets: [claude]}"),
		"unknown slot":                with("slots: {chat: {adapter: a.md}}"),
		"unknown slot field":          with("slots: {tasks: {adapter: a.md, mode: x}}"),
		"null slot":                   with("slots: {tasks: null}"),
		"absolute adapter":            with("slots: {tasks: {adapter: /etc/a.md}}"),
		"escaping adapter":            with("slots: {tasks: {adapter: a/../../b.md}}"),
		"blank adapter":               with(`slots: {tasks: {adapter: "  "}}`),
		"bad repo name":               with("repo: {name: vulcam}"),
		"bad branch":                  with(`repo: {integration_branch: "dev branch"}`),
		"zero parallelism":            with("parallelism: 0"),
		"string parallelism":          with(`parallelism: "3"`),
		"version 2":                   with("version: 2"),
		"negative version":            with("version: -1"),
		"bad mode":                    with("mode: yolo"),
		"empty checks":                with("review: {checks: []}"),
		"blank check":                 with(`review: {checks: ["  "]}`),
		"empty human_only_paths":      with("review: {human_only_paths: []}"),
		"absolute human_only path":    with("review: {human_only_paths: [/etc]}"),
		"escaping human_only path":    with("review: {human_only_paths: [../x/]}"),
		"bad pi env":                  with("review: {pi_credential_env: anthropic-key}"),
		"check env value":             with("review: {check_env: [A=B]}"),
		"check env lowercase":         with("review: {check_env: [db_url]}"),
		"check env blank":             with(`review: {check_env: [""]}`),
		"check env PATH":              with("review: {check_env: [PATH]}"),
		"check env HOME":              with("review: {check_env: [HOME]}"),
		"check env CI":                with("review: {check_env: [CI]}"),
		"check env GOCACHE":           with("review: {check_env: [GOCACHE]}"),
		"check env GOMODCACHE":        with("review: {check_env: [GOMODCACHE]}"),
		"check env GOPATH":            with("review: {check_env: [GOPATH]}"),
		"check env GABO_UPGRADE_PR":   with("review: {check_env: [GABO_UPGRADE_PR]}"),
		"check env GH_":               with("review: {check_env: [GH_TOKEN]}"),
		"check env GITHUB_":           with("review: {check_env: [GITHUB_TOKEN]}"),
		"check env CLAUDE_":           with("review: {check_env: [CLAUDE_CODE_OAUTH_TOKEN]}"),
		"check env REVIEWER_":         with("review: {check_env: [REVIEWER_GH_TOKEN]}"),
		"relative state dir":          with("review: {state_dir: data/reviewer}"),
		"bad coder login":             with("coder: {login: 'gabo coder'}"),
		"keep with slash":             with("worktree: {keep: [a/b]}"),
		"keep dot":                    with("worktree: {keep: [..]}"),
		"blank keep":                  with(`worktree: {keep: [""]}`),
		"later placement":             with("reviewer: {placement: github-actions}"),
		"unknown placement":           with("reviewer: {placement: moon}"),
		"bad marker":                  with("reviewer: {signing: {marker: 'a b'}}"),
		"developer key not an email":  with("developers: {leo: Leo}"),
		"blank developer alias":       with(`developers: {"a@b.c": " "}`),
		"bad plan scope":              with("plan_change: {scope: world}"),
		"string global_without_owner": with("plan_change: {global_without_owner: maybe}"),
		"bad wave after":              with("wave: {after: later}"),
		"unknown wave field":          with("wave: {chain: true}"),
		"bad interaction level":       with("interaction_level: max"),
		"bad language":                with("language: 'not a tag'"),
		"not a mapping":               "- a\n- b\n",
	}
	run := func(name, doc string, want bool) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, goErr := Parse([]byte(doc))
			schemaErr := ValidateSchema(SchemaFile, []byte(doc))
			if (goErr == nil) != want || (schemaErr == nil) != want {
				t.Fatalf("want valid=%v; Go error = %v; schema error = %v\n%s", want, goErr, schemaErr, doc)
			}
		})
	}
	for name, doc := range accept {
		run("accept/"+name, doc, true)
	}
	for name, doc := range reject {
		run("reject/"+name, doc, false)
	}
}

// The one deliberate difference: yaml.v3 reads a YAML number or boolean
// into a string field, the schema does not. The loader runs both, so such a
// value fails on load, here through the --config layer.
func TestSchemaRejectsNonStringScalarGoWouldCoerce(t *testing.T) {
	t.Parallel()
	doc := with("tone: 42")
	if _, err := Parse([]byte(doc)); err != nil {
		t.Fatalf("Parse() error = %v, want yaml.v3 to coerce 42 into a string", err)
	}
	_, err := LoadFile(files{"/x/c.yaml": doc}, "/x/c.yaml")
	if err == nil || !strings.Contains(err.Error(), "config /x/c.yaml: tone: ") {
		t.Fatalf("LoadFile() error = %v, want the schema error naming tone", err)
	}
}

// dbSchemaProjectExample is the project example of DB_SCHEMA 4.2, with a
// Claude reviewer (the DB_SCHEMA wording follows in T075).
const dbSchemaProjectExample = `
version: 1
mode: manual
repo:
  name: testeos-z/gabo-ai
  integration_branch: dev
runtimes:
  coder: claude
  assistant: claude
  reviewer: claude
slots:
  tasks:
    adapter: .gabo/adapters/tasks-gh.md
  memory:
    adapter: .gabo/adapters/memory-engram.md
  notifications:
    adapter: .gabo/adapters/notify-task-comment.md
  direct_sessions:
    adapter: .gabo/adapters/sessions-orca.md
  methodology:
    adapter: .gabo/adapters/methodology-tdd.md
  repository_host:
    adapter: .gabo/adapters/host-github.md
  reviewer_placement:
    adapter: .gabo/adapters/reviewer-local.md
  agent_host:
    adapter: .gabo/adapters/agent-host-local.md
reviewer:
  placement: local
  signing:
    marker: gabo-review
review:
  checks:
    - go test ./...
    - go vet ./...
  auto_merge: false
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md]
  pi_credential_env: ANTHROPIC_API_KEY
  state_dir: /data/gabo/reviewer
coder:
  login: gabo-coder
worktree:
  keep: [main]
developers:
  "leo@example.com": Leo
plan_change:
  scope: group
  global_without_owner: false
parallelism: 3
`

// The editor-facing project schema accepts the repository's own gabo.yaml
// and the DB_SCHEMA example, and flags personal keys, which belong in the
// user file.
func TestProjectSchema(t *testing.T) {
	t.Parallel()
	own, err := os.ReadFile(repoRoot + ProjectFile)
	if err != nil {
		t.Fatal(err)
	}
	for name, doc := range map[string]string{"gabo.yaml": string(own), "example": dbSchemaProjectExample} {
		if err := ValidateSchema(SchemaProject, []byte(doc)); err != nil {
			t.Errorf("%s: ValidateSchema(project) error = %v", name, err)
		}
	}
	err = ValidateSchema(SchemaProject, []byte(dbSchemaProjectExample+"alias: Leo\n"))
	if err == nil || !strings.Contains(err.Error(), "alias") {
		t.Errorf("personal key in project schema: error = %v, want it named", err)
	}
	if err := ValidateSchema(SchemaProject, []byte("version: 1\n")); err == nil {
		t.Error("incomplete project file: want required keys to fail")
	}
}

// The user schema accepts the DB_SCHEMA 4.1 example, a partial project
// fallback, and rejects unknown keys and bad personal values; it requires
// nothing.
func TestUserSchema(t *testing.T) {
	t.Parallel()
	for _, doc := range []string{
		"version: 1\nalias: Leo\ninteraction_level: balanced\ntone: direct, brief\nlanguage: es\n",
		"runtimes: {coder: pi}\nparallelism: 2\n",
		"{}",
	} {
		if err := ValidateSchema(SchemaUser, []byte(doc)); err != nil {
			t.Errorf("ValidateSchema(user, %q) error = %v", doc, err)
		}
	}
	for doc, key := range map[string]string{
		"nickname: Leo\n":            "nickname",
		"interaction_level: max\n":   "interaction_level",
		"runtimes: {coder: codex}\n": "runtimes.coder",
		"users: {orca: {uid: 1}}\n":  "users",
	} {
		err := ValidateSchema(SchemaUser, []byte(doc))
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("ValidateSchema(user, %q) error = %v, want it to name %s", doc, err, key)
		}
	}
}

// Schema errors name the key path of each failing value, one per line.
func TestValidateSchemaNamesKeyPaths(t *testing.T) {
	t.Parallel()
	err := ValidateSchema(SchemaLayer, []byte("colour: blue\nrepo: {name: x}\nslots: {tasks: {adapter: /a}}\n"))
	if err == nil {
		t.Fatal("ValidateSchema() error = nil")
	}
	for _, want := range []string{"(root): additional properties 'colour' not allowed", "repo.name: ", "slots.tasks.adapter: "} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want a line starting %q", err, want)
		}
	}
	if err := ValidateSchema(SchemaLayer, []byte("a: [")); err == nil {
		t.Error("malformed YAML: want an error")
	}
}

func readSchemaJSON(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := schemaFS.ReadFile("schemas/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return doc
}

// walkSchema calls fn for every schema object reachable through $defs,
// properties, items, additionalProperties and the combinators.
func walkSchema(s map[string]any, path string, fn func(string, map[string]any)) {
	fn(path, s)
	for _, k := range []string{"$defs", "properties"} {
		m, _ := s[k].(map[string]any)
		for name, v := range m {
			if sub, ok := v.(map[string]any); ok {
				walkSchema(sub, path+"/"+k+"/"+name, fn)
			}
		}
	}
	for _, k := range []string{"items", "additionalProperties", "not", "if", "then"} {
		if sub, ok := s[k].(map[string]any); ok {
			walkSchema(sub, path+"/"+k, fn)
		}
	}
	for _, k := range []string{"allOf", "anyOf", "oneOf"} {
		list, _ := s[k].([]any)
		for i, v := range list {
			if sub, ok := v.(map[string]any); ok {
				walkSchema(sub, path+"/"+k+"/"+string(rune('0'+i)), fn)
			}
		}
	}
}

// Every layer is validated against its schema on load, before the merge: a
// bad value in the user layer fails naming that file even when the project
// layer overrides it.
func TestLoadLayersValidatesEveryLayerAgainstItsSchema(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gabo/config.yaml": "runtimes: {coder: codex}\n",
		"/src/p/gabo.yaml":                 minimal,
	}
	_, err := LoadLayers(r, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "config /home/o/.config/gabo/config.yaml: runtimes.coder: ") {
		t.Fatalf("LoadLayers() error = %v, want the user layer's schema error", err)
	}
	r = files{"/src/p/gabo.yaml": minimal + "review_typo: 1\n"}
	if _, err := LoadLayers(r, twoLayers()); err == nil || !strings.Contains(err.Error(), "config /src/p/gabo.yaml: ") {
		t.Fatalf("LoadLayers() error = %v, want the project layer named", err)
	}
	// A personal key in the project file is still only a warning, whatever
	// its value: it is dropped before the project layer is validated.
	r = files{"/src/p/gabo.yaml": minimal + "interaction_level: 7\n"}
	res, err := LoadLayers(r, twoLayers())
	if err != nil || len(res.Warnings) != 1 {
		t.Fatalf("LoadLayers() = %v, %v; want one warning and no error", res.Warnings, err)
	}
}

// The key names that close the root and layer objects are exactly the keys
// their $defs define, so a new key cannot be defined and still be rejected.
func TestSchemaClosingNamesMatchTheirDefinitions(t *testing.T) {
	t.Parallel()
	project := readSchemaJSON(t, ProjectSchemaFile)
	user := readSchemaJSON(t, UserSchemaFile)
	keys := func(s map[string]any, path ...string) []string {
		for _, p := range path {
			s, _ = s[p].(map[string]any)
		}
		var out []string
		for k := range s {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}
	projectKeys := keys(project, "$defs", "projectKeys", "properties")
	allKeys := slices.Sorted(slices.Values(append(slices.Clone(projectKeys), keys(user, "$defs", "personalKeys", "properties")...)))
	if got := keys(project, "properties"); !slices.Equal(got, projectKeys) {
		t.Errorf("project root closes %v, want %v", got, projectKeys)
	}
	if got := keys(project, "$defs", "layer", "properties"); !slices.Equal(got, allKeys) {
		t.Errorf("layer closes %v, want %v", got, allKeys)
	}
}
