package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// OpenCode's project config lookup order (spec 0004 S8); the first existing
// file is the one gabo-runtime-opencode writes and the one checked.
var opencodeConfigs = []string{"opencode.json", "opencode.jsonc", ".opencode/opencode.json", ".opencode/opencode.jsonc"}

const (
	// subagentDepthKey is the OpenCode key gabo needs at minSubagentDepth or
	// more: assistant -> cycle -> coder -> test runner (spec 0004 S8, D7).
	subagentDepthKey = "experimental.subagent_depth"
	minSubagentDepth = 3
	// opencodeNewConfig is where the runtime skill writes the key when the
	// project has no OpenCode config.
	opencodeNewConfig = ".opencode/opencode.json"
)

// checkSubagentDepth requires, when a role runs on OpenCode, the project's
// experimental.subagent_depth at 3 or more in the first existing file of the
// S8 lookup order (spec 0004 S11). It reads only the project, never the
// user's ~/.config/opencode. A missing key or a whole number below 3 is
// what gabo init raises, so that line names gabo init (intent 0010 S6); a
// value gabo init refuses to merge names setting it by hand, and a file gabo
// init refuses names its reason, so no hint loops (D7).
func (u *UseCase) checkSubagentDepth(r *Report, cfg config.Config) error {
	if !slices.Contains(runtimesOf(cfg), string(config.RuntimeOpencode)) {
		return nil
	}
	file, why, byInit := opencodeNewConfig, "no project config", true
	for _, name := range opencodeConfigs {
		b, err := fs.ReadFile(u.project, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		file = name
		why, byInit = depthProblem(b)
		break
	}
	if why == "" {
		return nil
	}
	next := fmt.Sprintf("next: set %s to %d in %s, then rerun gabo check", subagentDepthKey, minSubagentDepth, file)
	if byInit {
		// Ask gabo init, in memory: a file it refuses names its reason, so the
		// hint never loops (intent 0010 D7).
		set := engineconfig.Set(runtimesOf(cfg)...)
		run, err := u.initDryRun(set, string(config.RuntimeOpencode), assistantsOf(cfg), opencodeConfigs)
		if err != nil {
			return err
		}
		if name, reason, ok := run.refusal(file, opencodeConfigs); ok {
			next, byInit = initRefusalHint(name, reason), false
		} else if p, _ := depthProblem(run.after(file)); p != "" {
			byInit = false
		}
	}
	if byInit {
		next = fmt.Sprintf("next: run gabo init (it sets %s to %d in %s), then rerun gabo check", subagentDepthKey, minSubagentDepth, file)
	}
	r.fail(fmt.Sprintf("✗ %s must be %d or more in %s: %s", subagentDepthKey, minSubagentDepth, file, why), next)
	return nil
}

// depthProblem returns why the config b does not set
// experimental.subagent_depth to a whole number of 3 or more, or "", and
// whether gabo init fixes it: a missing key or a whole number below 3.
func depthProblem(b []byte) (string, bool) {
	doc, err := parseJSONC(b)
	if err != nil {
		return "the file does not parse: " + err.Error(), false
	}
	raw, ok := doc["experimental"]
	if !ok {
		return "not set", true
	}
	exp, ok := raw.(map[string]any)
	if !ok {
		return "experimental is not an object", false
	}
	v, ok := exp["subagent_depth"]
	if !ok {
		return "not set", true
	}
	n, isNum := v.(float64)
	whole := isNum && n == float64(int64(n))
	if !whole || n < minSubagentDepth {
		shown, _ := json.Marshal(v)
		return "it is " + string(shown), whole
	}
	return "", false
}

// parseJSONC parses a JSON object that may hold // and /* */ comments and
// trailing commas, as OpenCode's .jsonc (and .json) configs may.
func parseJSONC(b []byte) (map[string]any, error) {
	plain, err := stripJSONC(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")))
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(plain, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("not a JSON object")
	}
	return doc, nil
}

// stripJSONC blanks comments and drops each comma whose next significant
// character closes an object or array; strings are copied as they are.
func stripJSONC(b []byte) ([]byte, error) {
	out := make([]byte, 0, len(b))
	comma := -1 // index in out of a comma not yet followed by a value
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == '"':
			j := i + 1
			for ; j < len(b) && b[j] != '"'; j++ {
				if b[j] == '\\' {
					j++
				}
			}
			if j >= len(b) {
				return nil, errors.New("unterminated string")
			}
			out = append(out, b[i:j+1]...)
			comma = -1
			i = j
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return nil, errors.New("unterminated comment at byte " + strconv.Itoa(i))
			}
			out = append(out, ' ')
			i += 2 + end + 1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			out = append(out, c)
		case (c == '}' || c == ']') && comma >= 0:
			out[comma] = ' '
			comma = -1
			out = append(out, c)
		case c == ',':
			comma = len(out)
			out = append(out, c)
		default:
			comma = -1
			out = append(out, c)
		}
	}
	return out, nil
}
