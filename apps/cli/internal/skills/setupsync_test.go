package skills

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// syncConfig returns the ## Sync config section of the setup skill. A missing
// section is reported and yields an empty string, so every assertion that
// reads it fails on its own.
func syncConfig(t *testing.T) string {
	t.Helper()
	_, sec, ok := strings.Cut(setupDoc(t), "\n## Sync config")
	if !ok {
		t.Errorf("%s: no ## Sync config section", setupSkill)
		return ""
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	return sec
}

// syncStep returns step n of the ## Sync config section without its number.
// A missing step is reported and yields an empty string.
func syncStep(t *testing.T, n int) string {
	t.Helper()
	_, after, ok := strings.Cut(syncConfig(t), "\n"+strconv.Itoa(n)+". ")
	if !ok {
		t.Errorf("%s ## Sync config: no step %d", setupSkill, n)
		return ""
	}
	if before, _, ok := strings.Cut(after, "\n"+strconv.Itoa(n+1)+". "); ok {
		return before
	}
	return after
}

// nameClass returns the one-segment character class the mode states in
// backticks for owner, repo and ref. It is read from the skill, not repeated
// here, so the behavior checks run against the rule as written.
func nameClass(t *testing.T) string {
	t.Helper()
	m := regexp.MustCompile("`(\\[[^`]+\\]\\+)`").FindStringSubmatch(syncConfig(t))
	if m == nil {
		t.Fatalf("%s ## Sync config: no backtick name pattern", setupSkill)
	}
	return m[1]
}

// nameOK reports whether one owner, repo or ref value is accepted: it matches
// the skill's pattern as a whole and is not a dot name.
func nameOK(class string, v string) bool {
	ok, _ := regexp.MatchString("^"+class+"$", v)
	return ok && v != "." && v != ".."
}

// urlForms returns the GitHub URL forms the mode lists in backticks, each as
// an anchored pattern whose <owner>, <repo> and <ref> placeholders capture
// their values by name.
func urlForms(t *testing.T) ([]string, []*regexp.Regexp) {
	t.Helper()
	class := nameClass(t)
	placeholder := regexp.MustCompile("<(owner|repo|ref)>")
	var forms []string
	var res []*regexp.Regexp
	for _, m := range regexp.MustCompile("`(https://[^`]+)`").FindAllStringSubmatch(syncConfig(t), -1) {
		form := m[1]
		forms = append(forms, form)
		var b strings.Builder
		b.WriteString("^")
		last := 0
		for _, loc := range placeholder.FindAllStringSubmatchIndex(form, -1) {
			b.WriteString(regexp.QuoteMeta(form[last:loc[0]]))
			b.WriteString("(?P<" + form[loc[2]:loc[3]] + ">" + class + ")")
			last = loc[1]
		}
		b.WriteString(regexp.QuoteMeta(form[last:]) + "$")
		res = append(res, regexp.MustCompile(b.String()))
	}
	return forms, res
}

// urlAccepted reports whether one of the listed forms matches the URL and
// every captured value passes nameOK.
func urlAccepted(class string, res []*regexp.Regexp, url string) bool {
	for _, re := range res {
		m := re.FindStringSubmatch(url)
		if m == nil {
			continue
		}
		ok := true
		for i, name := range re.SubexpNames() {
			if i > 0 && name != "" && !nameOK(class, m[i]) {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// Intent 0018 S1: the mode is named by the owner's words "sync config" in the
// front matter and the Activation Contract, and it is a section of the setup
// skill itself, not a new skill.
func TestSyncConfigIsNamedInTheActivationContract(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	desc := ""
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "description:") {
			desc = line
			break
		}
	}
	if !strings.Contains(desc, "sync config") {
		t.Errorf("%s front matter description: missing %q", setupSkill, "sync config")
	}
	if !strings.Contains(desc, "poll-config") {
		t.Errorf("%s front matter description: missing %q", setupSkill, "poll-config")
	}
	_, activation, ok := strings.Cut(doc, "## Activation Contract")
	if !ok {
		t.Fatalf("%s: no ## Activation Contract section", setupSkill)
	}
	activation, _, _ = strings.Cut(activation, "\n## ")
	if !strings.Contains(activation, "sync config") {
		t.Errorf("%s ## Activation Contract: missing %q", setupSkill, "sync config")
	}
	if !strings.Contains(activation, "poll-config") {
		t.Errorf("%s ## Activation Contract: missing %q", setupSkill, "poll-config")
	}
}

// Intent 0018 S1 and D2: the mode asks "local or remote" first, takes local
// when the owner does not answer, and the remote is another repository.
func TestSyncConfigAsksLocalOrRemoteAndDefaultsToLocal(t *testing.T) {
	t.Parallel()
	first := syncStep(t, 1)
	if !strings.Contains(first, "\"local or remote? (default: local)\"") {
		t.Errorf("%s ## Sync config step 1: the first question is not the local-or-remote question with local as default", setupSkill)
	}
	requireAll(t, first, []string{
		"local is the source when the owner does not answer", "Remote means another repository",
	})
	requireAll(t, syncConfig(t), []string{
		"which repository",
	})
}

// Intent 0018 S3 and D6: a remote source reads only one gabo.yaml, by local
// path or GitHub URL, read-only, without cloning and storing nothing.
func TestSyncConfigReadsOnlyOneRemoteFile(t *testing.T) {
	t.Parallel()
	sec := syncConfig(t)
	requireAll(t, sec, []string{
		"by local path or by GitHub URL", "`<path>/gabo.yaml`", "read-only", "without cloning",
		"stores nothing", "`gh api -H \"Accept: application/vnd.github.raw+json\" repos/<owner>/<repo>/contents/gabo.yaml`",
		"`?ref=<ref>`", "only when the URL names a branch", "existing `gh` login",
	})
	for _, write := range []string{"git clone", "git push", "gh api -X", "--method"} {
		if strings.Contains(sec, write) {
			t.Errorf("%s ## Sync config: the remote read must not use %q", setupSkill, write)
		}
	}
}

// Intent 0018 S3: an unreadable or missing source reports the reason and
// changes nothing.
func TestSyncConfigReportsAnUnreadableSource(t *testing.T) {
	t.Parallel()
	requireAll(t, syncConfig(t), []string{
		"unreadable", "missing", "reports the reason", "changes nothing",
	})
}

// Intent 0018 D6: a remote on another host than GitHub is not read through
// gh; the mode asks the owner for a local path instead.
func TestSyncConfigAsksForALocalPathOnAnotherHost(t *testing.T) {
	t.Parallel()
	requireAll(t, syncConfig(t), []string{
		"another host", "ask for a local path",
	})
}

// Intent 0018 S3: the owner's repository names and refs are checked before
// they reach gh. The skill's one-segment pattern accepts a plain name and
// refuses a slash, a space and the dot names, whatever the values are.
func TestSyncConfigNamesAreOneSegmentAndNeverDotNames(t *testing.T) {
	t.Parallel()
	class := nameClass(t)
	requireAll(t, syncConfig(t), []string{
		"is refused when it is `.` or `..`", "refusing `.` and `..`", "each validated value passed as its own argument",
	})
	for _, v := range []string{"main", "v1.2.3", "feat-x", "a_b.c", "gabo-ai"} {
		if !nameOK(class, v) {
			t.Errorf("%s ## Sync config: a plain name %q must be accepted", setupSkill, v)
		}
	}
	for _, v := range []string{"feat/x", ".", "..", "", "a b", "a?b", "a%2Fb"} {
		if nameOK(class, v) {
			t.Errorf("%s ## Sync config: the name %q must be refused", setupSkill, v)
		}
	}
}

// Intent 0018 S3: the GitHub URL forms are exactly the three the mode lists,
// any other URL is refused, and the ref is the one segment between /blob/ or
// /tree/ and the final /gabo.yaml.
func TestSyncConfigAcceptsExactlyTheThreeURLForms(t *testing.T) {
	t.Parallel()
	class := nameClass(t)
	forms, res := urlForms(t)
	want := []string{
		"https://github.com/<owner>/<repo>",
		"https://github.com/<owner>/<repo>/blob/<ref>/gabo.yaml",
		"https://github.com/<owner>/<repo>/tree/<ref>/gabo.yaml",
	}
	if strings.Join(forms, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s ## Sync config: the accepted URL forms are %q, want exactly %q", setupSkill, forms, want)
	}
	accept := []string{
		"https://github.com/gobern/gabo-ai",
		"https://github.com/o.w-n/r.e_p",
		"https://github.com/gobern/gabo-ai/blob/main/gabo.yaml",
		"https://github.com/gobern/gabo-ai/tree/feat-x/gabo.yaml",
		"https://github.com/gobern/gabo-ai/blob/v1.2.3/gabo.yaml",
	}
	refuse := []string{
		"http://github.com/gobern/gabo-ai",
		"https://gitlab.com/gobern/gabo-ai",
		"https://github.com/gobern/gabo-ai/",
		"https://github.com/gobern/gabo-ai/tree/main",
		"https://github.com/gobern/gabo-ai/raw/main/gabo.yaml",
		"https://github.com/gobern/gabo-ai/blob/main/other.yaml",
		"https://github.com/gobern/gabo-ai/blob/main/sub/gabo.yaml",
		"https://github.com/gobern/gabo-ai/blob/feat/x/gabo.yaml",
		"https://github.com/gobern/gabo-ai/blob/../gabo.yaml",
		"https://github.com/gobern/gabo-ai/tree/./gabo.yaml",
		"https://github.com/gobern/gabo-ai/blob/main/gabo.yaml/",
		"https://github.com/gobern/gabo-ai/blob/main/gabo.yaml?ref=x",
		"https://github.com/./gabo-ai",
		"https://github.com/gobern/..",
		"https://github.com//gabo-ai",
	}
	for _, u := range accept {
		if !urlAccepted(class, res, u) {
			t.Errorf("%s ## Sync config: the URL %q must be accepted", setupSkill, u)
		}
	}
	for _, u := range refuse {
		if urlAccepted(class, res, u) {
			t.Errorf("%s ## Sync config: the URL %q must be refused", setupSkill, u)
		}
	}
}

// hardRules returns the ## Hard Rules section of the setup skill.
func hardRules(t *testing.T) string {
	t.Helper()
	_, sec, ok := strings.Cut(setupDoc(t), "## Hard Rules")
	if !ok {
		t.Errorf("%s: no ## Hard Rules section", setupSkill)
		return ""
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	return sec
}

// Intent 0018 S1 and D2: the owner-answer rule of Hard Rules has exactly one
// explicit exception, the sync source default; the sync question shows local
// as its default, and nothing applies until the owner confirms the diff.
func TestSyncConfigDefaultIsTheOneExceptionToTheOwnerAnswerRule(t *testing.T) {
	t.Parallel()
	requireAll(t, hardRules(t), []string{
		"Apply no value without the owner's answer.",
		"The one explicit exception is the sync source default (the Sync config section, step 1)",
		"local is taken when the owner does not answer",
		"because it is read-only and nothing applies until the owner confirms the diff",
	})
}

// Intent 0018 S2, S4 and D5: the default source is read-only, so the local
// default (step 1) reads and reports, step 2 writes nothing, and step 4 says
// that a write of any key waits for the owner's confirmation of the diff.
func TestSyncConfigLocalDefaultIsReadOnlyUntilTheOwnerConfirms(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 2), []string{"write nothing"})
	requireAll(t, syncStep(t, 4), []string{
		"this part reads the source and writes no key",
		"is a later step, so it shows the diff first",
		"applies it only after the owner confirms the diff",
	})
}

// Intent 0018 D3: the profile path is the absolute $XDG_CONFIG_HOME/gabo/profile.yaml
// when that variable is an absolute path, else ~/.config/gabo/profile.yaml.
// The rule is stated once, in step 2, and no other profile path is written.
func TestSyncConfigPinsTheProfilePathRule(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 2)
	requireAll(t, step, []string{
		"the user config folder: `$XDG_CONFIG_HOME/gabo/profile.yaml` when `XDG_CONFIG_HOME` is an absolute path, else `~/.config/gabo/profile.yaml`",
		"no profile at <path>",
	})
	if n := strings.Count(syncConfig(t), "gabo/profile.yaml"); n != 2 {
		t.Errorf("%s ## Sync config: the profile path is spelled %d times, want exactly the two forms of the rule", setupSkill, n)
	}
}

// Intent 0018 S2: the local source reads the profile and no other file.
func TestSyncConfigReadsTheLocalProfile(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 2), []string{
		"Read the profile `profile.yaml`", "normal setup", "write nothing",
	})
}
