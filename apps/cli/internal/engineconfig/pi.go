package engineconfig

import (
	"encoding/json"
	"strings"
)

// SubagentRefusal is the one source of the refusal text (intent 0003 D7,
// intent 0006 S7): the quoted block of gabo-runtime-pi Prerequisite step 4,
// kept equal to the catalog skill by a test. The Pi hook mode and the pi
// section of the init report both print it.
const SubagentRefusal = "Pi has no subagents, so gabo cannot work on Pi without a subagent package. For subagents I recommend gentle-pi, installed with `pi install npm:gentle-pi` after adding `\"npmCommand\": [\"bun\"]` to Pi's global agent settings, so Pi's own installer runs through bun and needs no npm (bun must be on PATH); another subagent package is acceptable. As the recommended setup I recommend the full Gentle AI framework (https://github.com/Gentleman-Programming/gentle-ai): its methodology with RDD and the Engram memory stack, set up for Pi with `gentle-ai install --agent pi`. Then ask me to rebuild."

// PiPackage is the npm package of the gabo Pi integration, published with
// each release (D9). gabo writes it from this constant, never from
// packages/pi/package.json.
const PiPackage = "@gabo-ai/pi"

// PiSettings is the project file that declares Pi packages (D11).
const PiSettings = ".pi/settings.json"

// piSource is the npm source of PiPackage pinned to version.
func piSource(version string) string { return "npm:" + PiPackage + "@" + version }

// piRule makes .pi/settings.json hold one packages entry for PiPackage
// pinned to the binary version (S7). Existing bytes are kept; only the
// missing entry, or the missing packages key, is inserted (D5). An entry
// for PiPackage with any other version or form is kept (S4). A source
// build writes nothing. The refusal of a missing subagent package is printed
// only when the package check found none (intent 0016 S1, S2, D1).
func piRule(in Input) Result {
	res := pi(in)
	res.Section.Steps = append(res.Section.Steps, "trust the project in Pi so it installs and loads "+PiPackage+" from "+PiSettings)
	if in.PiPackage {
		return res
	}
	res.Section.Lines = append(res.Section.Lines, "gabo refuses to build on Pi without a subagent package:", SubagentRefusal)
	res.Section.Steps = append(res.Section.Steps, "install a subagent package (see the refusal above)")
	return res
}

func pi(in Input) Result {
	// `go install …@v<x.y.z>` stamps the binary v<x.y.z>; like the other
	// slices, the rule pins the plain x.y.z.
	version := strings.TrimPrefix(in.Version, "v")
	if !isRelease(version) {
		return Result{Section: Section{
			Lines: []string{"pi: development build " + in.Version + "; no published " + PiPackage + " to pin; nothing written"},
			Steps: []string{`add "` + piSource("<version>") + `" to ` + PiSettings + ` packages by hand, with <version> a released gabo version`},
		}}
	}
	want := piSource(version)
	fail := func(reason string) Result {
		return Result{Section: Section{Lines: []string{FailurePrefix + " " + PiSettings + ": " + reason + "; fix it and run gabo init again"}}}
	}
	wrote := func(b []byte) Result {
		return Result{
			Section: Section{Lines: []string{"wrote " + PiSettings + ": added packages entry " + want}},
			Writes:  map[string][]byte{PiSettings: b},
		}
	}

	data, ok := in.Files[PiSettings]
	if !ok {
		return wrote([]byte(`{"packages": ["` + want + `"]}` + "\n"))
	}
	if err := checkJSON(data); err != nil {
		return fail("invalid JSON (" + err.Error() + ")")
	}
	obj, ok := scanTopObject(data)
	if !ok {
		return fail("not a JSON object")
	}
	m, found := obj.member("packages")
	if !found {
		return wrote(obj.insert(data, `"packages": ["`+want+`"]`))
	}
	arr, ok := scanArray(data, m)
	if !ok {
		return fail("packages is not an array")
	}
	for _, e := range arr.items {
		src, ok := entrySource(data[e.start:e.end])
		if !ok || !namesPiPackage(src) {
			continue
		}
		if src == want {
			return Result{Section: Section{Lines: []string{"unchanged " + PiSettings + ": packages entry " + want + " present"}}}
		}
		return Result{Section: Section{Lines: []string{"kept " + PiSettings + " packages entry " + src + " by the user's choice (this gabo is " + version + ")"}}}
	}
	return wrote(arr.insert(data, `"`+want+`"`))
}

// PiListNamesPackage reports whether the output of `pi list` names a
// subagent package of gentle-pi (intent 0016 PD4, the rule of
// gabo-runtime-pi step 2): a listed source that is npm:gentle-pi, with or
// without @<version>, or a git: source of gentle-pi. Any other package, such
// as npm:@gabo-ai/pi or a git source whose name only contains gentle-pi,
// does not count.
func PiListNamesPackage(list []byte) bool {
	for _, line := range strings.Split(string(list), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && namesGentlePi(fields[0]) {
			return true
		}
	}
	return false
}

// namesGentlePi reports whether one listed source is gentle-pi: an npm
// source named gentle-pi, or a git source whose last path segment, without
// an @<ref> and a .git suffix, is gentle-pi.
func namesGentlePi(src string) bool {
	if rest, ok := strings.CutPrefix(src, "npm:gentle-pi"); ok {
		return rest == "" || strings.HasPrefix(rest, "@")
	}
	rest, ok := strings.CutPrefix(src, "git:")
	if !ok {
		return false
	}
	name := rest[strings.LastIndex(rest, "/")+1:]
	name, _, _ = strings.Cut(name, "@")
	return strings.TrimSuffix(name, ".git") == "gentle-pi"
}

// isRelease reports whether version is a plain x.y.z release; anything else
// is a source build with no published package to pin.
func isRelease(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			return false
		}
	}
	return true
}

// namesPiPackage reports whether an npm source names PiPackage, with or
// without a version.
func namesPiPackage(src string) bool {
	rest, ok := strings.CutPrefix(src, "npm:"+PiPackage)
	return ok && (rest == "" || strings.HasPrefix(rest, "@"))
}

// entrySource returns the source of a packages entry: the string itself or
// an object's "source" string.
func entrySource(raw []byte) (string, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, true
	}
	var o struct {
		Source *string `json:"source"`
	}
	if json.Unmarshal(raw, &o) == nil && o.Source != nil {
		return *o.Source, true
	}
	return "", false
}

// checkJSON returns the decoder's error for data that is not valid JSON.
func checkJSON(data []byte) error {
	var v any
	return json.Unmarshal(data, &v)
}

// The scanner below locates values in JSON already checked by checkJSON, so
// it only tracks positions and never re-validates; it uses skipWS and
// skipValue of the claude rule. Inserting at those positions keeps every
// existing byte (D5).

// item is one array element or object member: its key (objects only), the
// byte range of its value, and the whitespace before it (before the key for
// a member), which an inserted item copies to keep the file's layout.
type item struct {
	key        string
	ws         []byte
	start, end int
}

// container is a scanned object or array: its opening bracket and items.
type container struct {
	open  int
	items []item
}

// scanTopObject and scanArray require data to be valid JSON (checkJSON
// passed): the scanner does no bounds or syntax checks of its own.
func scanTopObject(data []byte) (container, bool) {
	i := skipWS(data, 0)
	if data[i] != '{' {
		return container{}, false
	}
	return scanContainer(data, i), true
}

func scanArray(data []byte, v item) (container, bool) {
	if data[v.start] != '[' {
		return container{}, false
	}
	return scanContainer(data, v.start), true
}

func scanContainer(data []byte, open int) container {
	c := container{open: open}
	isObject := data[open] == '{'
	i := open + 1
	for {
		ws := i
		i = skipWS(data, i)
		if data[i] == '}' || data[i] == ']' {
			return c
		}
		it := item{ws: data[ws:i]}
		if isObject {
			ks := i
			i = skipValue(data, i)
			_ = json.Unmarshal(data[ks:i], &it.key)
			i = skipWS(data, skipWS(data, i)+1) // past the colon
		}
		it.start = i
		i = skipValue(data, i)
		it.end = i
		c.items = append(c.items, it)
		i = skipWS(data, i)
		if data[i] == '}' || data[i] == ']' {
			return c
		}
		i++ // the comma
	}
}

// member returns the last member named key, as a JSON decoder would.
func (c container) member(key string) (item, bool) {
	for i := len(c.items) - 1; i >= 0; i-- {
		if c.items[i].key == key {
			return c.items[i], true
		}
	}
	return item{}, false
}

// insert returns data with text added as the container's last item, after
// the last existing item with that item's leading whitespace (a single space
// when there is none), or right after the opening bracket when empty.
func (c container) insert(data []byte, text string) []byte {
	at, add := c.open+1, text
	if n := len(c.items); n > 0 {
		last := c.items[n-1]
		ws := string(last.ws)
		if ws == "" {
			ws = " "
		}
		at, add = last.end, ","+ws+text
	}
	out := make([]byte, 0, len(data)+len(add))
	out = append(out, data[:at]...)
	out = append(out, add...)
	return append(out, data[at:]...)
}
