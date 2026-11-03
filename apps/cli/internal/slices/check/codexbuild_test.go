package check

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// codexConfig runs every role on Codex and points two slots at catalog
// adapters, so gabo check also wants their fixed-name links.
const codexConfig = `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: codex
  reviewer: codex
  assistant: codex
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
slots:
  reviewer_placement:
    adapter: .gabo/skills/gabo-reviewer-placement-local/SKILL.md
  agent_host:
    adapter: .gabo/skills/gabo-agent-host-local/SKILL.md
`

// codexUserLink is metadata.reviewer_link of gabo-runtime-codex for HOME.
const codexUserLink = home + "/.agents/skills/gabo-reviewer"

// tomlEscape writes s for a TOML basic string the way build step 6 of
// gabo-runtime-codex says: each \ as \\, each " (one-line) or """ (multi-line)
// escaped.
func tomlEscape(s string, multi bool) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	if multi {
		return strings.ReplaceAll(s, `"""`, `""\"`)
	}
	return strings.ReplaceAll(s, `"`, `\"`)
}

// codexBuilt is a project with the real embedded catalog and every spec agent
// built in the shape gabo-runtime-codex writes (.codex/agents/<id>.toml), each
// spec skill linked as .agents/skills/<name> -> ../../.gabo/skills/<name> and
// the fixed-name links of codexConfig's skill adapters.
func codexBuilt(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := fstest.MapFS{".gabo/VERSION": file(version + "\n")}
	files, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		proj[path.Join(CatalogDir, f.Path)] = &fstest.MapFile{Data: f.Data}
	}
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	fill := strings.NewReplacer(
		"{{gabo_version}}", version, "{{methodology}}", "gabo-methodology",
		"{{adapter:tasks}}", "gabo-tasks", "{{adapter:repository_host}}", "gabo-repository-host",
		"{{adapter:memory}}", "gabo-memory",
		"{{reviewer_script}}", "$HOME/.agents/skills/gabo-reviewer/scripts/reviewer.sh",
	)
	for _, a := range spec.Agents {
		tmpl, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
		if err != nil {
			t.Fatal(err)
		}
		_, rest, _ := strings.Cut(string(tmpl), "\n")
		body := strings.NewReplacer("{{id}}", a.ID, "{{role}}", a.Role).Replace(fill.Replace(strings.TrimPrefix(rest, "\n")))
		list := slices.Clone(a.Skills)
		for _, l := range [][2]string{{"{{methodology}}", "gabo-methodology"}, {"{{adapter:tasks}}", "gabo-tasks"}, {"{{adapter:repository_host}}", "gabo-repository-host"}, {"{{adapter:memory}}", "gabo-memory"}} {
			if strings.Contains(string(tmpl), l[0]) {
				list = append(list, l[1])
			}
		}
		body = strings.TrimRight(body, "\n") + "\n"
		if len(list) > 0 {
			body += "\nLoad these skills by name before work: " + strings.Join(list, ", ") + ".\n"
		}
		proj[".codex/agents/"+a.ID+".toml"] = file("# gabo-version: " + version + "\n" +
			`name = "` + a.ID + "\"\n" +
			`description = "` + tomlEscape(a.Role, false) + "\"\n" +
			`developer_instructions = """` + "\n" + tomlEscape(body, true) + `"""` + "\n")
		for _, name := range a.Skills {
			proj[".agents/skills/"+name] = symlink("../../.gabo/skills/" + name)
		}
	}
	proj[".agents/skills/gabo-reviewer-placement"] = symlink("../../.gabo/skills/gabo-reviewer-placement-local")
	proj[".agents/skills/gabo-agent-host"] = symlink("../../.gabo/skills/gabo-agent-host-local")
	return proj
}

// codexUser has the user-level reviewer link of gabo-runtime-codex.
func codexUser() userFS {
	u := goodUser().clone()
	delete(u.links, userLink)
	u.links[codexUserLink] = reviewerDir
	return u
}

func runCodex(t *testing.T, proj fs.FS, user UserFS, version string) Report {
	t.Helper()
	rep, err := NewUseCase(proj, configFiles{projectFile: codexConfig}, user,
		Settings{Version: version, Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// parseTOML parses the TOML subset gabo-runtime-codex writes: comments, blank
// lines, and bare keys set to a basic or multi-line basic string. Anything
// else is an error, so a file that passes is valid TOML.
func parseTOML(doc string) (map[string]string, error) {
	out := map[string]string{}
	for doc != "" {
		line, rest, _ := strings.Cut(doc, "\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			doc = rest
			continue
		}
		key, val, ok := strings.Cut(line, " = ")
		if !ok || strings.Trim(key, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-") != "" || key == "" {
			return nil, fmt.Errorf("not a key line: %q", line)
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("duplicate key %s", key)
		}
		var s string
		var err error
		if strings.HasPrefix(val, `"""`) {
			s, doc, err = basicString(strings.TrimPrefix(strings.TrimPrefix(doc[len(key)+3+3:], "\r"), "\n"), true)
		} else if strings.HasPrefix(val, `"`) {
			s, doc, err = basicString(doc[len(key)+3+1:], false)
		} else {
			return nil, fmt.Errorf("key %s: not a string", key)
		}
		if err != nil {
			return nil, fmt.Errorf("key %s: %w", key, err)
		}
		out[key] = s
		line, doc, _ = strings.Cut(doc, "\n")
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
			return nil, fmt.Errorf("key %s: text after the string: %q", key, line)
		}
	}
	return out, nil
}

// basicString reads a (multi-line) basic string body up to its closing
// quotes and returns the value and what follows them.
func basicString(s string, multi bool) (string, string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\':
			if i+1 >= len(s) {
				return "", "", fmt.Errorf("dangling backslash")
			}
			i++
			switch s[i] {
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", "", fmt.Errorf("unsupported escape \\%c", s[i])
			}
		case multi && strings.HasPrefix(s[i:], `"""`):
			return b.String(), s[i+3:], nil
		case !multi && c == '"':
			return b.String(), s[i+1:], nil
		case !multi && c == '\n':
			return "", "", fmt.Errorf("newline in a basic string")
		default:
			b.WriteByte(c)
		}
	}
	return "", "", fmt.Errorf("unterminated string")
}

// S4: each of the eight spec agents is a .codex/agents/<id>.toml that parses
// as TOML with name equal to its id, description, developer_instructions
// with the skill line, and only those keys.
func TestCodexBuiltAgentsParseAsTOML(t *testing.T) {
	t.Parallel()
	proj := codexBuilt(t, "1.2.3")
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, a := range spec.Agents {
		ids = append(ids, a.ID)
		doc := string(proj[".codex/agents/"+a.ID+".toml"].Data)
		kv, err := parseTOML(doc)
		if err != nil {
			t.Fatalf("%s: %v", a.ID, err)
		}
		if kv["name"] != a.ID || kv["description"] != a.Role || kv["developer_instructions"] == "" || len(kv) != 3 {
			t.Errorf("%s: keys %v", a.ID, kv)
		}
		if versionLine([]byte(doc)) != "1.2.3" {
			t.Errorf("%s: gabo-version %s", a.ID, versionLine([]byte(doc)))
		}
		if len(a.Skills) > 0 && !strings.Contains(kv["developer_instructions"], "\nLoad these skills by name before work: "+a.Skills[0]) {
			t.Errorf("%s: no skill line in developer_instructions", a.ID)
		}
		if strings.Contains(kv["developer_instructions"], "Load these skills by name before work:") &&
			strings.Contains(kv["developer_instructions"], "gabo-reviewer,") {
			t.Errorf("%s: loads gabo-reviewer", a.ID)
		}
	}
	slices.Sort(ids)
	want := []string{"gabo-coder", "gabo-cycle", "gabo-db-master", "gabo-dev-lead", "gabo-frontend", "gabo-plan-reviewer", "gabo-pm", "gabo-reviewer"}
	if !slices.Equal(ids, want) {
		t.Fatalf("spec ids %v, want %v", ids, want)
	}
}

// The TOML subset reader rejects what is not TOML, so the test above means
// something.
func TestParseTOMLRejectsInvalidFiles(t *testing.T) {
	t.Parallel()
	for _, doc := range []string{
		"name = gabo\n", "name = \"a\nb\"\n", "x = \"\"\"open\n", "name = \"a\" b\n",
		"name = \"a\"\nname = \"b\"\n", "bad key = \"a\"\n", "x = \"a\\q\"\n",
	} {
		if _, err := parseTOML(doc); err == nil {
			t.Errorf("parseTOML(%q) passed", doc)
		}
	}
	kv, err := parseTOML("# c\nname = \"a \\\"b\\\"\"\nd = \"\"\"\nx \"\" \\\\ \"\"\\\"\n\"\"\"\n")
	if err != nil || kv["name"] != `a "b"` || kv["d"] != "x \"\" \\ \"\"\"\n" {
		t.Fatalf("parseTOML = %q, %v", kv, err)
	}
}

// S5: the Codex build with its .agents/skills links and the user-level
// reviewer link passes gabo check, with no skill or reviewer link failure,
// and the reviewer is never in the project.
func TestCodexBuiltSetPassesCheck(t *testing.T) {
	t.Parallel()
	proj := codexBuilt(t, "1.2.3")
	rep := runCodex(t, proj, codexUser(), "1.2.3")
	wantPassed(t, rep, PassedLine)
	for _, l := range rep.Lines {
		if strings.HasPrefix(l, "✗ skill link") || strings.HasPrefix(l, "✗ reviewer link") {
			t.Errorf("unexpected %q", l)
		}
	}
	if _, err := fs.Lstat(proj, ".agents/skills/gabo-reviewer"); err == nil {
		t.Fatal(".agents/skills/gabo-reviewer exists in the project")
	}
}

// A TOML agent built with another version fails as for other runtimes.
func TestCodexBuiltSetFromAnotherVersionFails(t *testing.T) {
	t.Parallel()
	proj := with(codexBuilt(t, "1.2.3"), map[string]string{".gabo/VERSION": "1.2.4\n"})
	rep := runCodex(t, proj, codexUser(), "1.2.4")
	wantFailed(t, rep, "✗ agent gabo-coder built with 1.2.3, catalog is 1.2.4: ask your agent to rebuild")
}

// The links are checked in .agents/skills, and the reviewer link at the
// Codex user-level skill dir.
func TestCodexBuiltSetWithMissingLinksFails(t *testing.T) {
	t.Parallel()
	proj := codexBuilt(t, "1.2.3")
	delete(proj, ".agents/skills/gabo-wave-report")
	delete(proj, ".agents/skills/gabo-agent-host")
	u := codexUser()
	delete(u.links, codexUserLink)
	rep := runCodex(t, proj, u, "1.2.3")
	wantFailed(t, rep,
		"✗ skill link .agents/skills/gabo-wave-report missing",
		"✗ skill link .agents/skills/gabo-agent-host missing",
		"✗ reviewer link "+codexUserLink+" missing")
}
