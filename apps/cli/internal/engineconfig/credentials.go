package engineconfig

import "encoding/json"

// The base credential deny list (intent 0028 S8, D5). gabo init writes it at
// project level in each engine's own format, once per base path, for read
// and for edit. The user's global engine config is never read or written
// (PRD R32). Pi has no native deny format, so it reports not run (S14).

// claudeDenyKey is the report name of the Claude Code deny list.
const claudeDenyKey = "permissions.deny"

// claudeDenyPatterns are the base paths as Claude Code path patterns. Read
// and Edit rules take the same pattern (Claude Code permissions docs: a
// Read deny also blocks Edit, and Edit rules cover every file-editing tool).
var claudeDenyPatterns = []string{
	"**/.env", "**/.env.*", "**/*.pem", "**/*.key", "**/secrets/**",
	"~/.ssh/**", "~/.aws/credentials", "~/.config/gh/hosts.yml",
	"~/Library/Keychains/**", "~/.credentials/**",
}

// claudeDenyEntries returns the Read and Edit deny rule of each pattern, in
// that order per pattern.
func claudeDenyEntries() []any {
	var out []any
	for _, p := range claudeDenyPatterns {
		out = append(out, "Read("+p+")", "Edit("+p+")")
	}
	return out
}

// claudeDenyMerge writes the base Claude Code deny list into the settings
// bytes src when the project has no deny list yet (intent 0028 S8: a project
// adds or removes entries, and only a human changes them). A permissions.deny
// array that exists is the project's own and is kept as it is, so a removed
// entry stays removed; kept reports it only when the array lacks a base entry,
// so a second run on gabo's own output is silent. It reports whether it added
// the list, or the reason it cannot merge (a value of another shape), in which
// case src is returned unchanged.
func claudeDenyMerge(src []byte) (out []byte, added, kept bool, reason string) {
	top := scanObject(src, skipWS(src, 0))
	unit := indentUnit(src, top)
	list := claudeDenyEntries()
	perm, ok := top.member(src, "permissions")
	if !ok {
		return insertMember(src, top, jsonKV{"permissions", jsonObj{{"deny", list}}}, unit), true, false, ""
	}
	if src[perm.valStart] != '{' {
		return src, false, false, "permissions is not a JSON object"
	}
	obj := scanObject(src, perm.valStart)
	deny, ok := obj.member(src, "deny")
	if !ok {
		return insertMember(src, obj, jsonKV{"deny", list}, unit), true, false, ""
	}
	if src[deny.valStart] != '[' {
		return src, false, false, "permissions.deny is not an array"
	}
	have := map[string]bool{}
	for _, e := range arrayElems(src, deny.valStart) {
		var s string
		if json.Unmarshal(src[e.start:e.end], &s) == nil {
			have[s] = true
		}
	}
	for _, e := range list {
		if !have[e.(string)] {
			return src, false, true, ""
		}
	}
	return src, false, false, ""
}

// opencodeNoConfigNotRun is the report line of the OpenCode credential list
// when the project has no OpenCode config and OpenCode is not a listed
// assistant (intent 0015 S8).
const opencodeNoConfigNotRun = "not run: credential deny list: no OpenCode config yet (gabo creates one only for a listed assistant, intent 0015 S8)"

// opencodeDenyResources are the base paths as OpenCode V2 resource patterns:
// whole-value wildcards, where * also matches a path separator, and a leading
// ~/ is expanded when the config loads (OpenCode permissions docs). The
// *secrets/* pattern matches a secrets directory at any depth, and also a
// name that ends in secrets; the deny is kept broad on purpose.
var opencodeDenyResources = []string{
	"*.env", "*.env.*", "*.pem", "*.key", "*secrets/*",
	"~/.ssh/*", "~/.aws/credentials", "~/.config/gh/hosts.yml",
	"~/Library/Keychains/*", "~/.credentials/*",
}

// opencodeDenyRules returns the read and edit deny rule of each resource, in
// that order per resource. A later rule wins, so these rules follow the
// project's own rules when they are appended.
func opencodeDenyRules() []any {
	var out []any
	for _, r := range opencodeDenyResources {
		for _, action := range []string{"read", "edit"} {
			out = append(out, jsonObj{{"action", action}, {"resource", r}, {"effect", "deny"}})
		}
	}
	return out
}

// opencodeDenyMerge writes the base OpenCode deny rules into the config bytes
// src when the project has no permissions array yet, keeping every existing
// byte (D5, comments included). A permissions array that exists is the
// project's own and is kept (intent 0028 S8); kept reports it only when the
// array lacks a base rule. When the file does not exist, it returns a new
// config holding only the rules. It reports whether it added the rules, or the
// reason it cannot merge.
func opencodeDenyMerge(src []byte, exists bool) (out []byte, added, kept bool, reason string) {
	rules := opencodeDenyRules()
	if !exists {
		return []byte(renderJSON(jsonObj{{"permissions", rules}}, "", "  ", true) + "\n"), true, false, ""
	}
	top, reason := opencodeTop(src)
	if reason != "" {
		return src, false, false, reason
	}
	unit := indentUnit(src, top)
	perm, ok := top.member(src, "permissions")
	if !ok {
		return insertMember(src, top, jsonKV{"permissions", rules}, unit), true, false, ""
	}
	if src[perm.valStart] != '[' {
		return src, false, false, "permissions is not an array"
	}
	arr := arrayElems(src, perm.valStart)
	for _, r := range rules {
		found := false
		for _, e := range arr {
			if sameJSON(src[e.start:e.end], r) {
				found = true
				break
			}
		}
		if !found {
			return src, false, true, ""
		}
	}
	return src, false, false, ""
}

// codexHomeDeny are the base paths under the user's home that the profile
// denies for reads and writes; a deny covers both (Codex permissions docs).
var codexHomeDeny = []string{"~/.ssh", "~/.aws/credentials", "~/.config/gh/hosts.yml", "~/Library/Keychains", "~/.credentials"}

// codexWorkspaceDeny are the glob subpaths, relative to each workspace root,
// that the profile denies (Codex permissions docs, deny reads with globs).
var codexWorkspaceDeny = []string{"**/.env", "**/.env.*", "**/*.pem", "**/*.key", "**/secrets/**"}

// piCredentialNotRun is the report line of the credential list for Pi: it has
// no native deny format, so no file is written (intent 0028 S8, S14).
const piCredentialNotRun = "not run: credential deny list: Pi has no native deny format (" + PiSettings + " holds no permission or deny key)"

// spanElem is the byte range of one array element.
type spanElem struct {
	start, end int
}

// arrayElems returns the byte ranges of the elements of the array whose
// opening bracket is at src[open]. src must be valid JSON or JSONC (checked by
// the caller).
func arrayElems(src []byte, open int) []spanElem {
	var out []spanElem
	i := skipWS(src, open+1)
	for src[i] != ']' {
		if src[i] == ',' {
			i = skipWS(src, i+1)
			continue
		}
		end := skipValue(src, i)
		out = append(out, spanElem{start: i, end: end})
		i = skipWS(src, end)
	}
	return out
}
