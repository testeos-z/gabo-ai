package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

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
