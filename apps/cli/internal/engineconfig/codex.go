package engineconfig

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// CodexConfigPath is the project Codex config gabo writes (S3). gabo never
// writes the user-level Codex config or anything under CODEX_HOME.
const CodexConfigPath = ".codex/config.toml"

// codexFallback prints the Codex install commands until the S12 smoke
// confirms that the two project tables alone install and load the plugin
// in a trusted project (D4). A follow-up release clears it.
var codexFallback = true

// codexProfile is the permission profile that holds the base credential deny
// list (intent 0028 S8, D5). Codex applies a profile only when
// default_permissions names it.
const codexProfile = "gabo-credentials"

// codexGlobScanDepth bounds the expansion of the unbounded ** deny globs on
// Linux and WSL (Codex permissions docs recommend glob_scan_max_depth with
// them). Five directory levels cover a .env in a monorepo package.
const codexGlobScanDepth = 5

// codexTable is one table gabo needs in .codex/config.toml, with its keys
// in the order they are written. A zero header means the top level of the
// file, where a key must come before the first table header.
type codexTable struct {
	path   []string
	header string
	match  *regexp.Regexp
	keys   []codexKey
	// group, when set, is the one report name of the table's missing keys,
	// so a missing credential profile is one line and not one per path.
	group string
}

// codexKey is one key of a table: its decoded name, its gabo value and the
// TOML text gabo writes for the value.
type codexKey struct {
	key     string
	value   any
	literal string
}

var codexTables = []codexTable{
	{
		path:   []string{"marketplaces", "gabo-ai"},
		header: "marketplaces.gabo-ai",
		match:  codexHeader(`marketplaces`, `gabo-ai`),
		keys: []codexKey{
			{"source_type", "git", `"git"`},
			{"source", "https://github.com/testeos-z/gabo-ai.git", `"https://github.com/testeos-z/gabo-ai.git"`},
		},
	},
	{
		path:   []string{"plugins", "gabo@gabo-ai"},
		header: `plugins."gabo@gabo-ai"`,
		match:  codexHeader(`plugins`, `gabo@gabo-ai`),
		keys:   []codexKey{{"enabled", true, "true"}},
	},
}

// codexDefaultTable is the top-level key that makes the profile the default.
var codexDefaultTable = codexTable{
	keys:  []codexKey{{"default_permissions", codexProfile, strconv.Quote(codexProfile)}},
	group: "default_permissions",
}

// codexCredentialTables are the profile's tables: it extends :workspace,
// denies the home paths and the workspace-root globs, and bounds the globs.
var codexCredentialTables = []codexTable{
	{
		path:   []string{"permissions", codexProfile},
		header: "permissions." + codexProfile,
		match:  codexHeader(`permissions`, codexProfile),
		keys:   []codexKey{{"extends", ":workspace", `":workspace"`}},
		group:  "permissions." + codexProfile,
	},
	{
		path:   []string{"permissions", codexProfile, "filesystem"},
		header: "permissions." + codexProfile + ".filesystem",
		match:  codexHeader(`permissions`, codexProfile, `filesystem`),
		keys: append(codexDenyKeys(codexHomeDeny),
			codexKey{"glob_scan_max_depth", int64(codexGlobScanDepth), strconv.Itoa(codexGlobScanDepth)}),
		group: "permissions." + codexProfile,
	},
	{
		path:   []string{"permissions", codexProfile, "filesystem", ":workspace_roots"},
		header: "permissions." + codexProfile + `.filesystem.":workspace_roots"`,
		match:  codexHeader(`permissions`, codexProfile, `filesystem`, `:workspace_roots`),
		keys:   codexDenyKeys(codexWorkspaceDeny),
		group:  "permissions." + codexProfile,
	},
}

// codexDenyKeys returns one deny key per path.
func codexDenyKeys(paths []string) []codexKey {
	out := make([]codexKey, len(paths))
	for i, p := range paths {
		out[i] = codexKey{p, "deny", `"deny"`}
	}
	return out
}

var (
	codexBareKey   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	codexAnyHeader = regexp.MustCompile(`(?m)^[ \t]*\[`)
)

// tomlKey writes a key as TOML: bare when it can be, quoted otherwise.
func tomlKey(s string) string {
	if codexBareKey.MatchString(s) {
		return s
	}
	return strconv.Quote(s)
}

// name is the report name of key k of this table, as the report names it.
func (t codexTable) name(k codexKey) string {
	if t.header == "" {
		return tomlKey(k.key)
	}
	return t.header + "." + tomlKey(k.key)
}

// codexHeader matches a `[table.name]` header line of the given path
// segments in any of their key spellings (bare, basic or literal quoted),
// with an optional comment.
func codexHeader(segs ...string) *regexp.Regexp {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = codexSegment(s)
	}
	return regexp.MustCompile(`(?m)^[ \t]*\[[ \t]*` + strings.Join(parts, `[ \t]*\.[ \t]*`) + `[ \t]*\][ \t]*(?:#.*)?\r?$`)
}

// codexSegment matches one header segment in its key spellings.
func codexSegment(s string) string {
	q := regexp.QuoteMeta(s)
	if codexBareKey.MatchString(s) {
		return `(?:` + q + `|"` + q + `"|'` + q + `')`
	}
	return `(?:"` + q + `"|'` + q + `')`
}

// codexTablesFor returns the tables gabo keeps in the file, and the note of
// why the credential profile is not applied when another choice holds: a
// sandbox_mode or sandbox table makes Codex ignore profiles, and another
// default_permissions is the user's choice (intent 0028 S8, Codex permissions
// docs). The note is not a failure and writes nothing for the profile.
func codexTablesFor(doc map[string]any, profile bool) ([]codexTable, string) {
	if !profile {
		return codexTables, ""
	}
	for _, key := range []string{"sandbox_mode", "sandbox_workspace_write"} {
		if _, ok := doc[key]; ok {
			return codexTables, "not applied: credential deny list: " + key + " is set, and Codex then ignores permission profiles; remove " + key + " to apply it"
		}
	}
	if dp, ok := doc["default_permissions"]; ok && dp != any(codexProfile) {
		return codexTables, fmt.Sprintf("not applied: credential deny list: default_permissions is %s (your choice); set it to %q to apply the list", codexLiteral(dp), codexProfile)
	}
	tables := append(slices.Clone(codexTables), codexDefaultTable)
	return append(tables, codexCredentialTables...), ""
}

// codexRule makes .codex/config.toml hold the gabo marketplace and plugin
// tables (S3) and the credential profile (S8). It keeps every existing byte:
// missing tables are appended, missing keys of a table that exists are
// inserted below its header, and a key with another value is kept (S4).
// Invalid TOML, or a key it cannot add without rewriting the user's text, is
// one ✗ line and no write (S5).
func codexRule(in Input) Result { return codexSettingsRule(in, true) }

// codexSettingsRule is the Codex rule. With profile, it also writes the base
// credential profile (intent 0028 S8); without, only the marketplace and
// plugin tables change, which is what the key tests pin.
func codexSettingsRule(in Input, profile bool) Result {
	res := Result{Section: Section{Steps: codexSteps()}}
	old, exists := in.Files[CodexConfigPath]
	doc := map[string]any{}
	if exists {
		if err := toml.Unmarshal(old, &doc); err != nil {
			reason := strings.Join(strings.Fields(err.Error()), " ")
			res.Section.Lines = []string{FailurePrefix + " " + CodexConfigPath + ": invalid TOML (" + reason + "); fix it and run gabo init again"}
			return res
		}
	}
	tables, note := codexTablesFor(doc, profile)
	if note != "" {
		res.Section.Lines = append(res.Section.Lines, note)
	}

	eol := "\n"
	if bytes.Contains(old, []byte("\r\n")) {
		eol = "\r\n"
	}
	text := string(old)
	var added []string
	var blocks []string
	for _, t := range tables {
		table, tableFound := codexLookup(doc, t.path)
		var missing []codexKey
		for _, k := range t.keys {
			v, ok := table[k.key]
			switch {
			case !ok:
				missing = append(missing, k)
				if t.group == "" {
					added = append(added, t.name(k))
				} else if !slices.Contains(added, t.group) {
					added = append(added, t.group)
				}
			case !reflect.DeepEqual(v, k.value):
				res.Section.Lines = append(res.Section.Lines, fmt.Sprintf("kept %s = %s in %s (your choice)", t.name(k), codexLiteral(v), CodexConfigPath))
			}
		}
		if len(missing) == 0 {
			continue
		}
		var lines strings.Builder
		for _, k := range missing {
			lines.WriteString(tomlKey(k.key) + " = " + k.literal + eol)
		}
		if t.header == "" {
			text = codexInsertTop(text, lines.String(), eol)
			continue
		}
		if loc := t.match.FindStringIndex(text); tableFound && loc != nil {
			end := loc[1]
			if strings.HasPrefix(text[end:], "\r\n") {
				end += 2
			} else if strings.HasPrefix(text[end:], "\n") {
				end++
			} else {
				lines.Reset()
				for _, k := range missing {
					lines.WriteString(eol + tomlKey(k.key) + " = " + k.literal)
				}
			}
			text = text[:end] + lines.String() + text[end:]
			continue
		}
		blocks = append(blocks, "["+t.header+"]"+eol+lines.String())
	}

	if len(added) == 0 {
		if len(res.Section.Lines) == 0 {
			res.Section.Lines = []string{CodexConfigPath + " unchanged"}
		}
		return res
	}
	if len(blocks) > 0 {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += eol
		}
		if text != "" {
			text += eol
		}
		text += strings.Join(blocks, eol)
	}
	if !codexHolds([]byte(text), doc, tables) {
		res.Section.Lines = append(res.Section.Lines, FailurePrefix+" "+CodexConfigPath+": cannot add "+strings.Join(added, ", ")+" without rewriting your text; add them by hand and run gabo init again")
		return res
	}
	res.Writes = map[string][]byte{CodexConfigPath: []byte(text)}
	res.Section.Lines = append(res.Section.Lines, "wrote "+CodexConfigPath+": added "+strings.Join(added, ", "))
	return res
}

// codexInsertTop returns text with lines inserted before its first table
// header, where a top-level key belongs; with no header it goes at the end.
func codexInsertTop(text, lines, eol string) string {
	loc := codexAnyHeader.FindStringIndex(text)
	if loc == nil {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += eol
		}
		return text + lines
	}
	head := text[:loc[0]]
	if head != "" && !strings.HasSuffix(head, "\n") {
		head += eol
	}
	return head + lines + eol + text[loc[0]:]
}

// codexHolds re-reads the new text: it must parse, hold every gabo key the
// old document lacked with gabo's value, and keep every key the old one had.
func codexHolds(text []byte, old map[string]any, tables []codexTable) bool {
	doc := map[string]any{}
	if err := toml.Unmarshal(text, &doc); err != nil {
		return false
	}
	for _, t := range tables {
		oldTable, _ := codexLookup(old, t.path)
		table, _ := codexLookup(doc, t.path)
		for _, k := range t.keys {
			want, had := oldTable[k.key]
			if !had {
				want = k.value
			}
			if got, ok := table[k.key]; !ok || !reflect.DeepEqual(got, want) {
				return false
			}
		}
	}
	return true
}

// codexLookup walks the decoded document to the table at path; an empty path
// is the document itself.
func codexLookup(doc map[string]any, path []string) (map[string]any, bool) {
	cur := doc
	for _, p := range path {
		next, ok := cur[p].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// codexLiteral names a kept value as TOML: a quoted string, a scalar as
// written, an array or a table inline.
func codexLiteral(v any) string {
	switch v := v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case []any, map[string]any:
		var b bytes.Buffer
		enc := toml.NewEncoder(&b).SetTablesInline(true).SetIndentTables(false)
		if err := enc.Encode(map[string]any{"v": v}); err == nil {
			return strings.TrimSpace(strings.TrimPrefix(b.String(), "v = "))
		}
	}
	return fmt.Sprintf("%v", v)
}

// codexSteps are the S9 steps outside the project for codex.
func codexSteps() []string {
	steps := []string{
		"trust the project in Codex so it loads " + CodexConfigPath,
		"trust both gabo hooks in /hooks",
	}
	if codexFallback {
		steps = append(steps,
			"if Codex does not list the gabo plugin after trust, run: codex plugin marketplace add testeos-z/gabo-ai",
			"if Codex does not list the gabo plugin after trust, run: codex plugin add gabo@gabo-ai",
		)
	}
	return steps
}
