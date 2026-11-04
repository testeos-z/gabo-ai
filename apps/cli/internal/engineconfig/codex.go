package engineconfig

import (
	"bytes"
	"fmt"
	"regexp"
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

// codexTable is one table gabo needs in .codex/config.toml, with its keys
// in the order they are written. The marketplace is not pinned (D7).
type codexTable struct {
	path   []string
	header string
	match  *regexp.Regexp
	keys   []codexKey
}

type codexKey struct {
	name    string
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

// codexHeader matches a `[table.name]` header line in any of its key
// spellings (bare, basic or literal quoted), with an optional comment.
func codexHeader(table, name string) *regexp.Regexp {
	alt := func(s string) string {
		q := regexp.QuoteMeta(s)
		if regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(s) {
			return `(?:` + q + `|"` + q + `"|'` + q + `')`
		}
		return `(?:"` + q + `"|'` + q + `')`
	}
	return regexp.MustCompile(`(?m)^[ \t]*\[[ \t]*` + alt(table) + `[ \t]*\.[ \t]*` + alt(name) + `[ \t]*\][ \t]*(?:#.*)?\r?$`)
}

// codexRule makes .codex/config.toml hold the gabo marketplace and plugin
// tables (S3). It keeps every existing byte: missing tables are appended,
// missing keys of a table that exists are inserted below its header, and a
// key with another value is kept (S4). Invalid TOML, or a key it cannot add
// without rewriting the user's text, is one ✗ line and no write (S5).
func codexRule(in Input) Result {
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

	eol := "\n"
	if bytes.Contains(old, []byte("\r\n")) {
		eol = "\r\n"
	}
	text := string(old)
	var added []string
	var blocks []string
	for _, t := range codexTables {
		table, tableFound := codexLookup(doc, t.path)
		var missing []codexKey
		for _, k := range t.keys {
			v, ok := table[k.name]
			switch {
			case !ok:
				missing = append(missing, k)
				added = append(added, t.header+"."+k.name)
			case v != k.value:
				res.Section.Lines = append(res.Section.Lines, fmt.Sprintf("kept %s.%s = %s in %s (your choice)", t.header, k.name, codexLiteral(v), CodexConfigPath))
			}
		}
		if len(missing) == 0 {
			continue
		}
		var lines strings.Builder
		for _, k := range missing {
			lines.WriteString(k.name + " = " + k.literal + eol)
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
					lines.WriteString(eol + k.name + " = " + k.literal)
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
	if !codexHolds([]byte(text), doc) {
		res.Section.Lines = append(res.Section.Lines, FailurePrefix+" "+CodexConfigPath+": cannot add "+strings.Join(added, ", ")+" without rewriting your text; add them by hand and run gabo init again")
		return res
	}
	res.Writes = map[string][]byte{CodexConfigPath: []byte(text)}
	res.Section.Lines = append(res.Section.Lines, "wrote "+CodexConfigPath+": added "+strings.Join(added, ", "))
	return res
}

// codexHolds re-reads the new text: it must parse, hold every gabo key the
// old document lacked with gabo's value, and keep every key the old one had.
func codexHolds(text []byte, old map[string]any) bool {
	doc := map[string]any{}
	if err := toml.Unmarshal(text, &doc); err != nil {
		return false
	}
	for _, t := range codexTables {
		oldTable, _ := codexLookup(old, t.path)
		table, _ := codexLookup(doc, t.path)
		for _, k := range t.keys {
			want, had := oldTable[k.name]
			if !had {
				want = k.value
			}
			if got, ok := table[k.name]; !ok || got != want {
				return false
			}
		}
	}
	return true
}

// codexLookup walks the decoded document to the table at path.
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

func codexLiteral(v any) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
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
	return append(steps, "while testeos-z/gabo-ai is private, set up git credentials that can read it")
}
