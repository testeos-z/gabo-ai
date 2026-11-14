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

// claudeDenyMerge adds the missing Claude Code deny entries to the settings
// bytes src, keeping every existing entry and byte (D5). It reports whether
// it added any, or the reason it cannot merge (a permissions or deny value of
// another shape), in which case src is returned unchanged.
func claudeDenyMerge(src []byte) ([]byte, bool, string) {
	top := scanObject(src, skipWS(src, 0))
	unit := indentUnit(src, top)
	list := claudeDenyEntries()
	perm, ok := top.member(src, "permissions")
	if !ok {
		return insertMember(src, top, jsonKV{"permissions", jsonObj{{"deny", list}}}, unit), true, ""
	}
	if src[perm.valStart] != '{' {
		return src, false, "permissions is not a JSON object"
	}
	obj := scanObject(src, perm.valStart)
	deny, ok := obj.member(src, "deny")
	if !ok {
		return insertMember(src, obj, jsonKV{"deny", list}, unit), true, ""
	}
	if src[deny.valStart] != '[' {
		return src, false, "permissions.deny is not an array"
	}
	arr := scanSpanArray(src, deny.valStart)
	have := map[string]bool{}
	for _, e := range arr.elems {
		var s string
		if json.Unmarshal(src[e.start:e.end], &s) == nil {
			have[s] = true
		}
	}
	var missing []any
	for _, e := range list {
		if !have[e.(string)] {
			missing = append(missing, e)
		}
	}
	if len(missing) == 0 {
		return src, false, ""
	}
	return appendElems(src, arr, missing, unit), true, ""
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

// opencodeDenyMerge adds the missing OpenCode deny rules to the config bytes
// src, keeping every existing rule and byte (D5, comments included). When the
// file does not exist, it returns a new config holding only the rules. It
// reports whether it added any, or the reason it cannot merge.
func opencodeDenyMerge(src []byte, exists bool) ([]byte, bool, string) {
	rules := opencodeDenyRules()
	if !exists {
		return []byte(renderJSON(jsonObj{{"permissions", rules}}, "", "  ", true) + "\n"), true, ""
	}
	top, reason := opencodeTop(src)
	if reason != "" {
		return src, false, reason
	}
	unit := indentUnit(src, top)
	perm, ok := top.member(src, "permissions")
	if !ok {
		return insertMember(src, top, jsonKV{"permissions", rules}, unit), true, ""
	}
	if src[perm.valStart] != '[' {
		return src, false, "permissions is not an array"
	}
	arr := scanSpanArray(src, perm.valStart)
	var missing []any
	for _, r := range rules {
		found := false
		for _, e := range arr.elems {
			if sameJSON(src[e.start:e.end], r) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		return src, false, ""
	}
	return appendElems(src, arr, missing, unit), true, ""
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
