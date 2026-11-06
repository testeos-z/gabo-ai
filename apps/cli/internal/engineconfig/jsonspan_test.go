package engineconfig

import (
	"slices"
	"strings"
	"testing"
)

// The span scanner reads JSONC: it skips // and /* */ comments and trailing
// commas, and finds every top-level member with its exact key and value.
func TestScanObjectSkipsCommentsAndTrailingCommas(t *testing.T) {
	cases := map[string]string{
		"plain":            `{"a": 1, "b": {"c": [1, 2]}, "d": "x,}"}`,
		"line comments":    "{\n  // first\n  \"a\": 1, // after a\n  \"b\": {\"c\": [1, 2]},\n  \"d\": \"x,}\" // last\n}\n",
		"block comments":   "{/* x */\"a\"/* y */: /* z */1 /* , */, \"b\": {\"c\": [1, 2]}, \"d\": \"x,}\"/**/}",
		"trailing commas":  "{\n  \"a\": 1,\n  \"b\": {\"c\": [1, 2,],},\n  \"d\": \"x,}\",\n}\n",
		"comment slashes":  "{\"a\": 1, \"b\": {\"c\": [1, 2]}, \"d\": \"x,}\" /* // */}",
		"value then slash": "{\"a\": 1// c\n, \"b\": {\"c\": [1, 2]}, \"d\": \"x,}\"}",
		"crlf":             "{\r\n  \"a\": 1, // a\r\n  \"b\": {\"c\": [1, 2]},\r\n  \"d\": \"x,}\"\r\n}\r\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			b := []byte(src)
			if err := checkJSONC(b); err != nil {
				t.Fatalf("checkJSONC: %v", err)
			}
			o := scanObject(b, skipWS(b, 0))
			var keys, vals []string
			for _, m := range o.members {
				keys = append(keys, m.key)
				vals = append(vals, string(b[m.valStart:m.valEnd]))
			}
			if !slices.Equal(keys, []string{"a", "b", "d"}) {
				t.Errorf("keys = %q", keys)
			}
			if vals[0] != "1" || !strings.HasPrefix(vals[1], `{"c": [1, 2`) || !strings.HasSuffix(vals[1], "}") || vals[2] != `"x,}"` {
				t.Errorf("values = %q", vals)
			}
			if b[o.close] != '}' || strings.TrimSpace(string(b[o.close+1:])) != "" {
				t.Errorf("close at %d of %q", o.close, src)
			}
		})
	}
}

// checkJSONC accepts what OpenCode accepts and refuses the rest.
func TestCheckJSONC(t *testing.T) {
	for src, ok := range map[string]bool{
		"{}":                      true,
		"{\"a\": 1, // c\n}":      true,
		"[1, /* c */ 2,]":         true,
		"{\"a\": \"// not a c\"}": true,
		"\xef\xbb\xbf{\"a\": 1}":  true,
		"{\"a\": }":               false,
		"{\"a\": 1 /* open":       false,
		"{\"a\": \"open":          false,
		"{\"a\": 1,,}":            false,
		"{,}":                     false,
		"[,]":                     false,
		"{\"a\": [1,],}":          true,
		"{\"a\": 1} // c\n{}":     false,
	} {
		if err := checkJSONC([]byte(src)); (err == nil) != ok {
			t.Errorf("checkJSONC(%q) = %v, want ok %v", src, err, ok)
		}
	}
}

// removeMember deletes one member and its separating comma; every other
// byte, comments included, is kept.
func TestRemoveMember(t *testing.T) {
	cases := []struct {
		name, in, key, out string
	}{
		{"first of three", "{\n  \"x\": \"g\",\n  \"a\": 1,\n  \"b\": 2\n}\n", "x", "{\n  \"a\": 1,\n  \"b\": 2\n}\n"},
		{"middle", "{\n  \"a\": 1,\n  \"x\": \"g\",\n  \"b\": 2\n}\n", "x", "{\n  \"a\": 1,\n  \"b\": 2\n}\n"},
		{"last", "{\n  \"a\": 1,\n  \"x\": \"g\"\n}\n", "x", "{\n  \"a\": 1\n}\n"},
		{"only", "{\n  \"x\": \"g\"\n}\n", "x", "{}\n"},
		{"inline middle", `{"a": 1, "x": "g", "b": 2}`, "x", `{"a": 1, "b": 2}`},
		{"inline last", `{"a": 1, "x": "g"}`, "x", `{"a": 1}`},
		{"inline only", `{"x": "g"}`, "x", `{}`},
		{"comment before the next key stays", "{\n  \"x\": \"g\",\n  // about a\n  \"a\": 1\n}\n", "x", "{\n  // about a\n  \"a\": 1\n}\n"},
		{"comment of the previous member stays", "{\n  \"a\": 1, // about a\n  \"x\": \"g\"\n}\n", "x", "{\n  \"a\": 1 // about a\n}\n"},
		{"trailing comma last", "{\n  \"a\": 1,\n  \"x\": \"g\",\n}\n", "x", "{\n  \"a\": 1,\n}\n"},
		{"only with a comment", "{\n  // keep\n  \"x\": \"g\"\n}\n", "x", "{\n  // keep\n}\n"},
		{"crlf", "{\r\n  \"a\": 1,\r\n  \"x\": \"g\",\r\n  \"b\": 2\r\n}\r\n", "x", "{\r\n  \"a\": 1,\r\n  \"b\": 2\r\n}\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := []byte(c.in)
			o := scanObject(b, skipWS(b, 0))
			i := slices.IndexFunc(o.members, func(m spanMember) bool { return m.key == c.key })
			if i < 0 {
				t.Fatalf("no member %q", c.key)
			}
			got := string(removeMember(b, o, i))
			if got != c.out {
				t.Fatalf("got\n%q\nwant\n%q", got, c.out)
			}
			if err := checkJSONC([]byte(got)); err != nil {
				t.Fatalf("result does not parse: %v", err)
			}
		})
	}
}

// insertMember keeps comments with the member they follow and keeps a
// trailing-comma style.
func TestInsertMemberKeepsComments(t *testing.T) {
	kv := jsonKV{"x", "g"}
	cases := []struct {
		name, in, out string
	}{
		{"plain", "{\n  \"a\": 1\n}\n", "{\n  \"a\": 1,\n  \"x\": \"g\"\n}\n"},
		{"line comment after the last value", "{\n  \"a\": 1 // about a\n}\n", "{\n  \"a\": 1, // about a\n  \"x\": \"g\"\n}\n"},
		{"trailing comma", "{\n  \"a\": 1, // about a\n}\n", "{\n  \"a\": 1, // about a\n  \"x\": \"g\",\n}\n"},
		{"empty with a comment", "{\n  // nothing yet\n}\n", "{\n  // nothing yet\n  \"x\": \"g\"\n}\n"},
		{"crlf", "{\r\n  \"a\": 1\r\n}\r\n", "{\r\n  \"a\": 1,\r\n  \"x\": \"g\"\r\n}\r\n"},
		{"inline", `{"a": 1}`, `{"a": 1, "x": "g"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := []byte(c.in)
			top := scanObject(b, skipWS(b, 0))
			got := string(insertMember(b, top, kv, indentUnit(b, top)))
			if got != c.out {
				t.Fatalf("got\n%q\nwant\n%q", got, c.out)
			}
			if err := checkJSONC([]byte(got)); err != nil {
				t.Fatalf("result does not parse: %v", err)
			}
		})
	}
}

// replaceValue swaps one member's value and keeps every other byte, the
// comments around the value included (intent 0010 S3).
func TestReplaceValueKeepsComments(t *testing.T) {
	cases := []struct {
		name, in, out string
	}{
		{"plain", "{\"a\": 1, \"b\": 2}", "{\"a\": 3, \"b\": 2}"},
		{"comments around", "{\n  \"a\": /* low */ 1, // keep\n  \"b\": 2\n}\n", "{\n  \"a\": /* low */ 3, // keep\n  \"b\": 2\n}\n"},
		{"crlf trailing comma", "{\r\n  \"a\": 1,\r\n}\r\n", "{\r\n  \"a\": 3,\r\n}\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := []byte(c.in)
			m, ok := scanObject(b, skipWS(b, 0)).member(b, "a")
			if !ok {
				t.Fatal("no member a")
			}
			if got := string(replaceValue(b, m, 3)); got != c.out {
				t.Fatalf("got\n%q\nwant\n%q", got, c.out)
			}
		})
	}
}

// count tells a key that appears more than once, which no rule can merge
// without loss (intent 0010 S3).
func TestSpanObjectCount(t *testing.T) {
	b := []byte("{\"a\": 1, // c\n \"b\": 2, \"a\": 3,}")
	o := scanObject(b, skipWS(b, 0))
	if got := [3]int{o.count("a"), o.count("b"), o.count("x")}; got != [3]int{2, 1, 0} {
		t.Errorf("counts = %v, want [2 1 0]", got)
	}
}

// insertMember into a nested object, empty or not, keeps the file's layout
// and comments (intent 0010 S3).
func TestInsertMemberIntoNestedObject(t *testing.T) {
	kv := jsonKV{"x", 3}
	cases := []struct {
		name, in, out string
	}{
		{"empty nested", "{\n  \"e\": {},\n  \"a\": 1\n}\n", "{\n  \"e\": {\n    \"x\": 3\n  },\n  \"a\": 1\n}\n"},
		{"nested with a comment", "{\n  \"e\": {\n    \"y\": true // keep\n  }\n}\n", "{\n  \"e\": {\n    \"y\": true, // keep\n    \"x\": 3\n  }\n}\n"},
		{"inline nested", `{"e": {"y": true}}`, `{"e": {"y": true, "x": 3}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := []byte(c.in)
			top := scanObject(b, skipWS(b, 0))
			e, _ := top.member(b, "e")
			got := string(insertMember(b, scanObject(b, e.valStart), kv, indentUnit(b, top)))
			if got != c.out {
				t.Fatalf("got\n%q\nwant\n%q", got, c.out)
			}
			if err := checkJSONC([]byte(got)); err != nil {
				t.Fatalf("result does not parse: %v", err)
			}
		})
	}
}
