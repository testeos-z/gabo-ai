package engineconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// The span scanner locates members in a JSON or JSONC document (// and
// /* */ comments, trailing commas, as OpenCode configs allow) so a rule can
// insert or delete one member and keep every other byte (intent 0006 D5,
// intent 0010 S2). It only tracks positions: the caller checks the
// document first (json.Unmarshal for plain JSON, checkJSONC for JSONC).

// sameJSON reports whether raw decodes to the same value as want.
func sameJSON(raw []byte, want any) bool {
	var got, exp any
	if json.Unmarshal(raw, &got) != nil {
		return false
	}
	_ = json.Unmarshal([]byte(renderJSON(want, "", "", false)), &exp)
	return reflect.DeepEqual(got, exp)
}

// jsonObj is a JSON object with its members in order; jsonKV is one member.
// Values are jsonObj, string or bool.
type jsonObj []jsonKV

type jsonKV struct {
	key   string
	value any
}

// renderJSON renders v. Multiline puts each member on its own line at
// indent+unit and the closing brace at indent; inline uses ": " and ", ".
func renderJSON(v any, indent, unit string, multiline bool) string {
	obj, ok := v.(jsonObj)
	if !ok {
		b, _ := json.Marshal(v)
		return string(b)
	}
	parts := make([]string, len(obj))
	for i, kv := range obj {
		parts[i] = quoteJSON(kv.key) + ": " + renderJSON(kv.value, indent+unit, unit, multiline)
	}
	if !multiline {
		return "{" + strings.Join(parts, ", ") + "}"
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{\n" + indent + unit + strings.Join(parts, ",\n"+indent+unit) + "\n" + indent + "}"
}

func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// utf8BOM may open a JSONC file; OpenCode ignores it.
var utf8BOM = []byte("\xef\xbb\xbf")

// checkJSONC returns why src is not a JSONC document: valid JSON once its
// comments and trailing commas are blanked.
func checkJSONC(src []byte) error {
	plain, err := blankJSONC(bytes.TrimPrefix(src, utf8BOM))
	if err != nil {
		return err
	}
	var v any
	return json.Unmarshal(plain, &v)
}

// blankJSONC returns src with every comment and every comma whose next
// significant character closes an object or array replaced by spaces
// (newlines kept), so offsets in the result match src.
func blankJSONC(src []byte) ([]byte, error) {
	out := bytes.Clone(src)
	comma := -1    // offset of a comma not yet followed by a value
	value := false // the last significant byte ends a value
	for i := 0; i < len(out); i++ {
		switch c := out[i]; {
		case c == '"':
			j := i + 1
			for ; j < len(out) && out[j] != '"'; j++ {
				if out[j] == '\\' {
					j++
				}
			}
			if j >= len(out) {
				return nil, errors.New("unterminated string")
			}
			comma, value, i = -1, true, j
		case c == '/' && i+1 < len(out) && (out[i+1] == '/' || out[i+1] == '*'):
			end := commentEnd(out, i)
			if end < 0 {
				return nil, errors.New("unterminated comment at byte " + strconv.Itoa(i))
			}
			for k := i; k < end; k++ {
				if out[k] != '\n' && out[k] != '\r' {
					out[k] = ' '
				}
			}
			i = end - 1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		case c == '}' || c == ']':
			if comma >= 0 {
				out[comma] = ' '
			}
			comma, value = -1, true
		case c == ',':
			// Only a comma after a value may be a trailing comma.
			if comma = -1; value {
				comma = i
			}
			value = false
		case c == '{' || c == '[' || c == ':':
			comma, value = -1, false
		default:
			comma, value = -1, true
		}
	}
	return out, nil
}

// commentEnd returns the offset just past the comment starting at src[i]
// ("//" up to, not including, the newline; "/*" through "*/"), or -1 for
// an unterminated block comment.
func commentEnd(src []byte, i int) int {
	if src[i+1] == '/' {
		if n := bytes.IndexByte(src[i:], '\n'); n >= 0 {
			return i + n
		}
		return len(src)
	}
	if n := bytes.Index(src[i+2:], []byte("*/")); n >= 0 {
		return i + 2 + n + 2
	}
	return -1
}

func isComment(src []byte, i int) bool {
	return i+1 < len(src) && src[i] == '/' && (src[i+1] == '/' || src[i+1] == '*')
}

// spanObject is the byte layout of one JSON object in a valid document.
type spanObject struct {
	open, close int // offsets of '{' and '}'
	members     []spanMember
}

// spanMember is one member: where its key starts and its value's span.
type spanMember struct {
	key              string
	keyStart         int
	valStart, valEnd int
}

func (o spanObject) member(src []byte, key string) (spanMember, bool) {
	for _, m := range o.members {
		if m.key == key {
			return m, true
		}
	}
	return spanMember{}, false
}

// scanObject reads the object starting at src[i] == '{'. src must be a
// valid JSON or JSONC document (checked by the caller).
func scanObject(src []byte, i int) spanObject {
	o := spanObject{open: i}
	i = skipWS(src, i+1)
	for src[i] != '}' {
		if src[i] == ',' {
			i = skipWS(src, i+1)
			continue
		}
		keyEnd := skipValue(src, i)
		var key string
		_ = json.Unmarshal(src[i:keyEnd], &key)
		valStart := skipWS(src, skipWS(src, keyEnd)+1) // past ':'
		valEnd := skipValue(src, valStart)
		o.members = append(o.members, spanMember{key: key, keyStart: i, valStart: valStart, valEnd: valEnd})
		i = skipWS(src, valEnd)
	}
	o.close = i
	return o
}

// skipWS returns the offset of the next significant byte at or after i:
// whitespace and comments are skipped.
func skipWS(src []byte, i int) int {
	for i < len(src) {
		switch {
		case strings.IndexByte(" \t\r\n", src[i]) >= 0:
			i++
		case isComment(src, i):
			if i = commentEnd(src, i); i < 0 {
				return len(src)
			}
		default:
			return i
		}
	}
	return i
}

// skipBlank skips spaces and tabs only, never a newline or a comment.
func skipBlank(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i
}

// skipValue returns the offset just past the value starting at src[i].
func skipValue(src []byte, i int) int {
	switch src[i] {
	case '"':
		for i++; src[i] != '"'; i++ {
			if src[i] == '\\' {
				i++
			}
		}
		return i + 1
	case '{', '[':
		depth := 0
		for ; ; i++ {
			switch {
			case src[i] == '"':
				i = skipValue(src, i) - 1
			case isComment(src, i):
				i = commentEnd(src, i) - 1
			case src[i] == '{' || src[i] == '[':
				depth++
			case src[i] == '}' || src[i] == ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	default:
		for i < len(src) && strings.IndexByte(",}] \t\r\n/", src[i]) < 0 {
			i++
		}
		return i
	}
}

// lineIndent returns the leading whitespace of the line holding src[i].
func lineIndent(src []byte, i int) string {
	start := bytes.LastIndexByte(src[:i], '\n') + 1
	end := start
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return string(src[start:end])
}

// newline is the file's line ending: CRLF when it holds one, else LF.
func newline(src []byte) string {
	if bytes.Contains(src, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

// multiline reports whether the object puts its members on their own lines;
// an empty object is multiline unless the whole document is one line.
func multiline(src []byte, o spanObject) bool {
	if len(o.members) == 0 {
		return bytes.Contains(bytes.TrimSpace(src), []byte("\n")) || len(bytes.TrimSpace(src)) == 2
	}
	return bytes.Contains(src[o.open:o.members[0].keyStart], []byte("\n"))
}

// indentUnit is the file's indent step, taken from the top-level object's
// first member; two spaces when the file shows none.
func indentUnit(src []byte, top spanObject) string {
	if len(top.members) > 0 && multiline(src, top) {
		if unit := strings.TrimPrefix(lineIndent(src, top.members[0].keyStart), lineIndent(src, top.open)); unit != "" {
			return unit
		}
	}
	return "  "
}

// insertMember adds kv as the last member of o, in the object's layout:
// after the last member on a new line at its indent, or inline after ", ".
// On its own line it goes after the last member's comma and same-line
// comments, so a comment stays with the member it follows, and it gets a
// trailing comma when the last member has one.
func insertMember(src []byte, o spanObject, kv jsonKV, unit string) []byte {
	ml := multiline(src, o)
	nl := newline(src)
	member := func(indent string) string {
		return quoteJSON(kv.key) + ": " + renderJSON(kv.value, indent, unit, true)
	}
	if len(o.members) == 0 {
		indent := lineIndent(src, o.open)
		switch {
		case !ml:
			return splice(src, o.open+1, o.close, quoteJSON(kv.key)+": "+renderJSON(kv.value, "", "", false))
		case len(bytes.TrimSpace(src[o.open+1:o.close])) == 0:
			return splice(src, o.open+1, o.close, nl+indent+unit+member(indent+unit)+nl+indent)
		default: // only comments inside: add the member after them
			end := len(bytes.TrimRight(src[:o.close], " \t\r\n"))
			return splice(src, end, end, nl+indent+unit+member(indent+unit))
		}
	}
	last := o.members[len(o.members)-1]
	if !ml {
		return splice(src, last.valEnd, last.valEnd, ", "+quoteJSON(kv.key)+": "+renderJSON(kv.value, "", "", false))
	}
	indent := lineIndent(src, last.keyStart)
	// The end of the last member's line: past its comma and comments.
	at, comma := last.valEnd, false
	for {
		j := skipBlank(src, at)
		switch {
		case j < len(src) && src[j] == ',' && !comma:
			at, comma = j+1, true
			continue
		case isComment(src, j) && (src[j+1] == '/' || !bytes.Contains(src[j:commentEnd(src, j)], []byte("\n"))):
			at = commentEnd(src, j)
			continue
		}
		break
	}
	if at == last.valEnd {
		return splice(src, at, at, ","+nl+indent+member(indent))
	}
	if comma {
		return splice(src, at, at, nl+indent+member(indent)+",")
	}
	out := splice(src, at, at, nl+indent+member(indent))
	return splice(out, last.valEnd, last.valEnd, ",")
}

// removeMember deletes member i of o and its separating comma; every other
// byte, comments included, is kept. A member on lines of its own goes with
// those lines; an inline member goes with the space after its comma.
func removeMember(src []byte, o spanObject, i int) []byte {
	m := o.members[i]
	after := commaAfter(src, m.valEnd)
	if len(o.members) == 1 && !hasComment(src[o.open+1:o.close]) {
		return splice(src, o.open+1, o.close, "")
	}
	// The member's own lines: blank from the line start to the key, and from
	// the value (and its comma) to the end of the line.
	lineStart := bytes.LastIndexByte(src[:m.keyStart], '\n') + 1
	tail := m.valEnd
	if after >= 0 && strings.TrimSpace(string(src[m.valEnd:after])) == "" {
		tail = after + 1
	}
	tail = skipBlank(src, tail)
	if tail < len(src) && src[tail] == '\r' {
		tail++
	}
	ownLines := strings.TrimSpace(string(src[lineStart:m.keyStart])) == "" && tail < len(src) && src[tail] == '\n'
	switch {
	case after >= 0 && ownLines && tail > after: // with the comma after it
		return splice(src, lineStart, tail+1, "")
	case after >= 0 && i+1 < len(o.members) && strings.TrimSpace(string(src[m.valEnd:after])) == "":
		return splice(src, m.keyStart, skipBlank(src, after+1), "")
	case after >= 0:
		out := splice(src, after, after+1, "")
		return splice(out, m.keyStart, m.valEnd, "")
	}
	// The last member without a trailing comma: the comma before it goes.
	before := -1
	if i > 0 {
		before = commaAfter(src, o.members[i-1].valEnd)
	}
	switch {
	case before >= 0 && !hasComment(src[before:m.keyStart]) && strings.TrimSpace(string(src[o.members[i-1].valEnd:before])) == "":
		return splice(src, o.members[i-1].valEnd, m.valEnd, "")
	case ownLines:
		out := splice(src, lineStart, tail+1, "")
		if before >= 0 {
			out = splice(out, before, before+1, "")
		}
		return out
	default:
		out := splice(src, m.keyStart, m.valEnd, "")
		if before >= 0 {
			out = splice(out, before, before+1, "")
		}
		return out
	}
}

// commaAfter returns the offset of the comma that follows the value ending
// at end, past whitespace and comments, or -1 when none does.
func commaAfter(src []byte, end int) int {
	if j := skipWS(src, end); j < len(src) && src[j] == ',' {
		return j
	}
	return -1
}

// hasComment reports whether b holds a comment outside strings.
func hasComment(b []byte) bool {
	for i := 0; i < len(b); i++ {
		switch {
		case b[i] == '"':
			i = skipValue(b, i) - 1
		case isComment(b, i):
			return true
		}
	}
	return false
}

// splice returns src with src[from:to] replaced by text.
func splice(src []byte, from, to int, text string) []byte {
	out := make([]byte, 0, len(src)-(to-from)+len(text))
	out = append(out, src[:from]...)
	out = append(out, text...)
	return append(out, src[to:]...)
}
