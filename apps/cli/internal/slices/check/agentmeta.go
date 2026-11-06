package check

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// NextRestoreCatalog follows a catalog runtime skill that lacks a
// declaration gabo check needs.
const NextRestoreCatalog = "next: run gabo init to restore the catalog, then rerun gabo check"

// Forms of an agent place (intent 0008 D1).
const (
	// formFrontmatter reads a YAML list or scalar under <key> of the agent
	// file's frontmatter.
	formFrontmatter = "frontmatter"
	// formTOML reads a top-level TOML string under <key>.
	formTOML = "toml"
	// formLine reads the first line whose trimmed text starts with <prefix>,
	// also inside a TOML multi-line string.
	formLine = "line"
)

// agentPlace is where an agent file keeps a value, declared by a runtime
// skill as metadata.agent_skills or metadata.agent_name in the form
// <form>:<argument>. The zero value is "not declared".
type agentPlace struct {
	form, arg string
}

// parseAgentPlace parses the declaration of metadata.<key>; "" is not
// declared.
func parseAgentPlace(key, s string) (agentPlace, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return agentPlace{}, nil
	}
	form, arg, _ := strings.Cut(s, ":")
	switch form {
	case formFrontmatter, formTOML, formLine:
	default:
		return agentPlace{}, fmt.Errorf("metadata.%s %q: unknown form, want frontmatter:<key>, toml:<key> or line:<prefix>", key, s)
	}
	if strings.TrimSpace(arg) == "" {
		return agentPlace{}, fmt.Errorf("metadata.%s %q: empty argument", key, s)
	}
	return agentPlace{form: form, arg: strings.TrimSpace(arg)}, nil
}

func (p agentPlace) declared() bool { return p.form != "" }

// skills returns the skill list an agent file keeps at p; a file with no
// such place has none.
func (p agentPlace) skills(b []byte) []string {
	switch p.form {
	case formFrontmatter:
		switch v := frontmatterValue(b, p.arg).(type) {
		case string:
			return splitList(v)
		case []any:
			var out []string
			for _, item := range v {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
			return out
		}
	case formTOML:
		return splitList(tomlString(b, p.arg))
	case formLine:
		text, ok := lineAfter(b, p.arg)
		if !ok {
			return nil
		}
		if _, list, found := strings.Cut(text, " for each of "); found {
			text = list
		}
		return splitList(text)
	}
	return nil
}

// name returns the name an agent file declares at p, or "".
func (p agentPlace) name(b []byte) string {
	switch p.form {
	case formFrontmatter:
		s, _ := frontmatterValue(b, p.arg).(string)
		return strings.TrimSpace(s)
	case formTOML:
		return strings.TrimSpace(tomlString(b, p.arg))
	case formLine:
		text, _ := lineAfter(b, p.arg)
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "."))
	}
	return ""
}

// frontmatterValue returns the value under key of the file's YAML
// frontmatter, or nil.
func frontmatterValue(b []byte, key string) any {
	rest, ok := bytes.CutPrefix(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), []byte("---\n"))
	if !ok {
		return nil
	}
	front, _, ok := bytes.Cut(rest, []byte("\n---"))
	if !ok {
		return nil
	}
	var m map[string]any
	if yaml.Unmarshal(front, &m) != nil {
		return nil
	}
	return m[key]
}

// tomlString returns the top-level TOML string under key, or "".
func tomlString(b []byte, key string) string {
	var m map[string]any
	if toml.Unmarshal(b, &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

// lineAfter returns the text after prefix on the first line whose trimmed
// text starts with prefix.
func lineAfter(b []byte, prefix string) (string, bool) {
	for line := range strings.SplitSeq(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), prefix); ok {
			return rest, true
		}
	}
	return "", false
}

// splitList splits a comma-separated list without its final ".", trimming
// spaces and backticks around each item and dropping empty items.
func splitList(s string) []string {
	s = strings.TrimSuffix(strings.TrimSpace(s), ".")
	var out []string
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.Trim(strings.TrimSpace(item), "`"); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// fullTier reports whether engineconfig has an integration rule for the
// engine (the full tier, PRD section 6, TRD D28): Set orders the engines
// with a rule before any other name.
func fullTier(engine string) bool {
	return engineconfig.Set("\x00", engine)[0] == engine
}

// checkAgentMeta requires a full-tier runtime skill to declare
// metadata.agent_skills (intent 0008 S2); a flexible-tier one without it
// gets one warning and is never failed by the per-agent rules.
func checkAgentMeta(r *Report, rt string, meta runtimeMeta) {
	if meta.AgentSkills.declared() {
		return
	}
	skill := "gabo-runtime-" + rt
	if fullTier(rt) {
		r.fail("✗ runtime "+rt+": "+skill+" declares no metadata.agent_skills", NextRestoreCatalog)
		return
	}
	r.warn("runtime " + rt + ": " + skill + " declares no metadata.agent_skills; per-agent skill and name checks skipped for " + rt + " (flexible tier)")
}
