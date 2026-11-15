package skills

import (
	"os"
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
func TestSyncConfigAsksLocalOrRemoteAndAnnouncesTheDefault(t *testing.T) {
	t.Parallel()
	first := syncStep(t, 1)
	// The question names both answers, local is pre-selected, and one sentence
	// announces the default and says how to switch to remote.
	requireAll(t, first, []string{
		"Ask one question at a time: \"local or remote?\", with local pre-selected and announced in one sentence:",
		"\"I will use the local profile; say remote to use another repository; Enter or ok keeps local.\"",
		"Enter or ok means local.",
		"Remote means another repository",
	})
	requireAll(t, syncConfig(t), []string{
		"which repository",
	})
}

// Intent 0018 S1 and D2: the default is not silent and not an exception to
// the owner-answer rule; nothing in the skill presents local as an answer the
// owner did not give.
func TestSyncConfigDefaultIsAnnouncedNotSilent(t *testing.T) {
	t.Parallel()
	rules := hardRules(t)
	requireAll(t, rules, []string{
		"Apply no value without the owner's answer.",
		"A default is allowed only when it is announced, never silent",
		"pre-selects local and says how to switch",
	})
	for _, bad := range []string{"exception", "does not answer"} {
		if strings.Contains(rules, bad) || strings.Contains(syncStep(t, 1), bad) {
			t.Errorf("%s: the default is still described as %q", setupSkill, bad)
		}
	}
}

// Intent 0018 D3: profile.yaml is its own file, not a config layer, and the
// save step of the mode writes it; the example config names the same rule.
func TestSyncConfigDefinesProfileYamlAsItsOwnFile(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 2), []string{
		"`profile.yaml` is its own file in the same folder as `config.yaml`, not a config layer",
		"the save step of this mode (T3 #501) writes it",
	})
	b, err := os.ReadFile("../../gabo.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "(profile.yaml beside it is no layer: setup's sync mode reads it, not gabo)") {
		t.Errorf("apps/cli/gabo.example.yaml: missing the profile.yaml comment next to the user-file rule")
	}
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

// lineTokens returns the backtick-quoted tokens of the first line of sec that
// holds marker. A missing marker is reported and yields nil.
func lineTokens(t *testing.T, sec, marker string) []string {
	t.Helper()
	for _, line := range strings.Split(sec, "\n") {
		if !strings.Contains(line, marker) {
			continue
		}
		var out []string
		for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(line, -1) {
			out = append(out, m[1])
		}
		return out
	}
	t.Errorf("%s ## Sync config: no line with %q", setupSkill, marker)
	return nil
}

// sameList reports whether two token lists hold the same tokens in order.
func sameList(a, b []string) bool {
	return strings.Join(a, "\n") == strings.Join(b, "\n")
}

// Intent 0018 S4: the proposal holds exactly the work-style keys, in the
// order the spec lists them, and no other key of the source.
func TestSyncProposalKeysAreExactlyTheS4Keys(t *testing.T) {
	t.Parallel()
	got := lineTokens(t, syncStep(t, 5), "exactly these (S4):")
	want := []string{
		"mode", "runtimes.coder", "runtimes.reviewer", "runtimes.assistant",
		"slots.<name>.adapter", "parallelism", "wave.after", "plan_change.scope",
		"plan_change.global_without_owner", "review.auto_merge", "reviewer.placement",
	}
	if !sameList(got, want) {
		t.Errorf("%s step 5: the S4 keys are %q, want exactly %q", setupSkill, got, want)
	}
	requireAll(t, syncStep(t, 5), []string{"No other key of the source is proposed."})
}

// Intent 0018 D4: the generic review.human_only_paths entries are exactly the
// seven the plan fixes; any other entry of the source is asked, not proposed.
func TestSyncProposalHumanOnlyPathsAreExactlyTheGenericEntries(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	got := lineTokens(t, step, "D4) are exactly these:")
	want := []string{"gabo.yaml", ".github/", "AGENTS.md", "CLAUDE.md", ".claude/", ".codex/", ".agents/"}
	if !sameList(got, want) {
		t.Errorf("%s step 5: the generic human-only entries are %q, want exactly %q", setupSkill, got, want)
	}
	requireAll(t, step, []string{
		"Any other entry of the source (for example `plugins/`) is asked about, one question at a time, and is not proposed.",
	})
}

// Intent 0018 D5 and S4: the proposal is one diff against the target
// gabo.yaml, with a status per key, and it marks the autonomy keys.
func TestSyncProposalIsOneDiffWithStatusesAndAutonomyMarks(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	requireAll(t, step, []string{
		"compared with the target `gabo.yaml`, and show it as one diff.",
		"one table with the columns `key`, `target`, `source` and `status`",
		"`same` when the values are equal",
		"`differs` when they are not",
		"`added` when the target lacks the key",
		"The status column says autonomy for the autonomy keys.",
	})
	got := lineTokens(t, step, "Autonomy keys (marked in the diff):")
	want := []string{"mode", "review.auto_merge", "plan_change.global_without_owner"}
	if !sameList(got, want) {
		t.Errorf("%s step 5: the autonomy keys are %q, want exactly %q", setupSkill, got, want)
	}
}

// Intent 0018 S4 and D5: the target value is kept unless the owner picks the
// source value, and the owner can drop any key from the diff.
func TestSyncKeepsTheTargetValueUnlessTheOwnerPicksTheSource(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 5), []string{
		"The target value stays by default.",
		"The source value replaces it only when the owner picks the source value for that key in step 6;",
	})
	requireAll(t, syncStep(t, 6), []string{
		"pick the source value for a `differs` key",
		"drop any key",
	})
}

// Intent 0018 S4: no write happens before the owner confirms the diff; the
// proposal writes nothing, and in step 6 the confirmation comes before the
// first call of gabo-config-writer.
func TestSyncWritesNothingBeforeTheOwnerConfirms(t *testing.T) {
	t.Parallel()
	propose := syncStep(t, 5)
	if strings.Contains(propose, "gabo-config-writer") {
		t.Errorf("%s step 5: the proposal must not call gabo-config-writer", setupSkill)
	}
	requireAll(t, propose, []string{"Writes nothing in this step."})

	apply := syncStep(t, 6)
	answer := strings.Index(apply, "Write nothing before that answer;")
	writer := strings.Index(apply, "gabo-config-writer")
	if answer < 0 || writer < 0 || answer > writer {
		t.Errorf("%s step 6: the owner's answer must gate the first gabo-config-writer call (answer at %d, writer at %d)", setupSkill, answer, writer)
	}
	requireAll(t, apply, []string{
		"a dropped or declined key is not written",
		"through `gabo-config-writer`",
		"its minimal edit, then `gabo check` before the next key",
		"the writer's recorded-decision path applies to each value",
		"reported as unverified",
	})
}

// Intent 0018 S5: the repository-specific and personal keys are never copied.
func TestSyncNeverCopiesRepositorySpecificOrPersonalKeys(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	got := lineTokens(t, step, "Never copied (S5):")
	want := []string{
		"repo.name", "repo.integration_branch", "review.checks", "review.check_env",
		"developers", "worktree.keep", "alias", "language", "tone", "interaction_level",
	}
	if !sameList(got, want) {
		t.Errorf("%s step 5: the never-copied keys are %q, want exactly %q", setupSkill, got, want)
	}
	requireAll(t, step, []string{
		"a slot adapter path",
		"`repo.name` is detected from the target's git remote",
		"Each key of this list is skipped, not refused: its key path is reported under the table, and the sync continues with the other keys.",
	})
}

// Owner decision 2026-11-11 (intent 0018 S5): a repository-specific key is
// skipped and reported by key path; the sync never stops on it and never asks.
func TestSyncSkipsRepositorySpecificKeysInsteadOfStopping(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	requireAll(t, step, []string{
		"Each key of this list is skipped, not refused",
		"the sync continues with the other keys",
	})
	if strings.Contains(step, "asked one at a time") {
		t.Errorf("%s step 5: repository-specific keys are skipped, not asked", setupSkill)
	}
}

// Owner decision 2026-11-11 (intent 0018 S5, S6): a key ending in _env is
// skipped and reported by key path, and the sync continues with the rest. The
// name rule runs first, so a credential-named _env key still stops the sync.
func TestSyncSkipsEnvKeysAndContinues(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	requireAll(t, step, []string{
		"a key whose last segment ends in `_env` is skipped, not refused",
		"its key path is reported under the table, and the sync continues with the other keys",
		"The name rule above runs first",
		"`review.pi_credential_env`",
		"still stops the sync",
		"Every key and value of the source, including a skipped one, is checked for these before the proposal is built.",
	})
}

// Owner decision 2026-11-11: the skipped keys (the _env and S5 skips) are
// listed after the table by key path, one line each, and are not table rows.
func TestSyncReportsSkippedKeysByKeyPath(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 5), []string{
		"Skipped keys:",
		"one line per key path",
		"is not a row of the table",
	})
}

// Intent 0018 D4 and PD4: an adapter is copied by its skill name and resolved
// in the target, asking the owner when the target has no such adapter.
func TestSyncCopiesAdapterNamesAndResolvesThemInTheTarget(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 5), []string{
		"the source names an adapter by its skill name.",
		"In the target it resolves to `.gabo/skills/<name>/SKILL.md` when that file exists;",
		"otherwise the skill asks the owner which adapter the target uses for that slot, one question.",
	})
}

// Intent 0018 S6 and PD5: a secret-like key or value stops the sync; the
// refusal names the key path only, and the rule is the one PD5 defines.
func TestSyncRefusesSecretLikeKeysAndValuesByKeyPath(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 5)
	gotKeys := lineTokens(t, step, "Secret-like key (PD5)")
	wantKeys := []string{"token", "secret", "password", "passwd", "credential", "api_key", "_key", "_login", "login"}
	if !sameList(gotKeys, wantKeys) {
		t.Errorf("%s step 5: the secret-like key rule is %q, want exactly %q", setupSkill, gotKeys, wantKeys)
	}
	gotValues := lineTokens(t, step, "Secret-like value (PD5):")
	wantValues := []string{"ghp_", "gho_", "github_pat_", "sk-", "xox", "AKIA", "eyJ"}
	if !sameList(gotValues, wantValues) {
		t.Errorf("%s step 5: the secret-like value prefixes are %q, want exactly %q", setupSkill, gotValues, wantValues)
	}
	requireAll(t, step, []string{
		"Every key and value of the source, including a skipped one, is checked for these before the proposal is built.",
		"name the key path only, never the value, and write nothing",
	})
}

// Intent 0018 S7 and D3: step 7 finishes setup after a sync that wrote a key;
// it runs gabo init first when the catalog is missing or runtimes changed,
// then setup's steps 4, 5 and 6 in that order, skipping none.
func TestSyncFinishRunsInitAndTheRestOfSetupInOrder(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 7)
	requireAll(t, step, []string{
		"after the apply step writes at least one key",
		"`gabo init` first when `.gabo/skills/` is missing or `runtimes.*` changed",
		"the agent build for each engine of the engine set",
		"none is skipped",
		"reported as `unverified`",
	})
	marks := []string{"`gabo init` first", "setup's step 4", "setup's step 5", "setup's step 6"}
	last := -1
	for _, m := range marks {
		at := strings.Index(step, m)
		if at < 0 || at < last {
			t.Errorf("%s step 7: %q missing or out of order", setupSkill, m)
			return
		}
		last = at
	}
}

// Intent 0018 S7 and D3: step 8 saves the style of a configured repository, or
// of the sync that just ran, as the local profile. It writes the profile at the
// path of step 2 and spells no second copy of the path rule.
func TestSyncSaveWritesTheProfileAtTheStepTwoPath(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	requireAll(t, step, []string{
		"save this style as the local profile",
		"the profile path of step 2",
	})
	if strings.Contains(step, "gabo/profile.yaml") {
		t.Errorf("%s step 8: the profile path is spelled again; it must reference step 2", setupSkill)
	}
}

// Intent 0018 S7, S4 and D4: the saved keys are exactly the S4 keys of step 5,
// in the same order, plus review.human_only_paths, whose generic entries the
// save step keeps under the same rule as the proposal. No other key is saved.
func TestSyncSaveKeysAreExactlyTheS4Keys(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	got := lineTokens(t, step, "Saved keys (S4):")
	want := []string{
		"mode", "runtimes.coder", "runtimes.reviewer", "runtimes.assistant",
		"slots.<name>.adapter", "parallelism", "wave.after", "plan_change.scope",
		"plan_change.global_without_owner", "review.auto_merge", "reviewer.placement",
		"review.human_only_paths",
	}
	if !sameList(got, want) {
		t.Errorf("%s step 8: the saved keys are %q, want exactly %q", setupSkill, got, want)
	}
	s5 := lineTokens(t, syncStep(t, 5), "exactly these (S4):")
	if !sameList(got, append(append([]string{}, s5...), "review.human_only_paths")) {
		t.Errorf("%s step 8: the saved keys %q differ from the proposal keys of step 5 %q", setupSkill, got, s5)
	}
	requireAll(t, step, []string{"No other key of the target is saved."})
}

// Intent 0018 S4 and D4: the save step saves the generic review.human_only_paths
// entries and no other entry, the same seven entries the proposal of step 5
// holds.
func TestSyncSaveHumanOnlyPathsAreTheGenericEntries(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	got := lineTokens(t, step, "D4) are exactly these:")
	want := lineTokens(t, syncStep(t, 5), "D4) are exactly these:")
	if !sameList(got, want) || len(got) != 7 {
		t.Errorf("%s step 8: the saved human-only entries are %q, want exactly the step 5 generic entries %q", setupSkill, got, want)
	}
	requireAll(t, step, []string{
		"Any other entry of the configured repository is not saved.",
	})
}

// Intent 0018 S5 and S6: the save never writes a repository-specific, personal
// or secret-like key; the PD5 rules of step 5 run on every saved key and value
// before the profile is built, and a match writes nothing.
func TestSyncSaveNeverWritesRepositorySpecificPersonalOrSecretKeys(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	requireAll(t, step, []string{
		"Never saved (S5): the keys of the **Never copied (S5)** list of step 5",
		"the secret-like key and value rules (PD5) of step 5 run on every saved key and value before the profile is built",
		"a match writes nothing and names the key path only, never the value",
	})
}

// Intent 0018 PD4: a saved adapter is its skill name, never a path; an adapter
// outside the catalog has no portable name, so the owner is asked for it.
func TestSyncSaveAdapterIsASkillNameNeverAPath(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 8), []string{
		"`slots.<name>.adapter` is saved as the adapter's skill name, the folder name of `.gabo/skills/<name>/SKILL.md`, never a path",
		"asks the owner for the skill name, one question, or leaves the key out",
	})
}

// Intent 0018 D3: the saved keys are checked against gabo.project.schema.json
// before the profile is written.
func TestSyncSaveValidatesAgainstTheProjectSchemaBeforeWriting(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	schema := strings.Index(step, "`gabo.project.schema.json`")
	write := strings.Index(step, "Write the profile")
	if schema < 0 || write < 0 || write < schema {
		t.Errorf("%s step 8: the keys must be checked against `gabo.project.schema.json` before the profile is written (schema at %d, write at %d)", setupSkill, schema, write)
	}
}

// Intent 0018 S7: an existing profile is never overwritten without the owner's
// yes; the diff is shown first and nothing is written before that answer.
func TestSyncSaveExistingProfileGatesTheWrite(t *testing.T) {
	t.Parallel()
	step := syncStep(t, 8)
	requireAll(t, step, []string{
		"If the profile already exists, show the difference first, as a table with the columns `key`, `profile`, `new` and `status`",
		"Write nothing without the owner's yes.",
	})
	gate := strings.Index(step, "Write nothing without the owner's yes.")
	write := strings.Index(step, "Write the profile")
	if gate < 0 || write < 0 || write < gate {
		t.Errorf("%s step 8: the owner's yes must gate the write (gate at %d, write at %d)", setupSkill, gate, write)
	}
}

// Intent 0018 D3 and S6: the profile is written with the writer's rules; the
// writer itself keeps writing only the two config files, and gabo check does
// not read the profile. The written profile is read back.
func TestSyncSaveUsesTheWriterRulesAndReadsItBack(t *testing.T) {
	t.Parallel()
	requireAll(t, syncStep(t, 8), []string{
		"the rules of `gabo-config-writer`: a minimal YAML edit, owner values only, no secret, keys outside the schemas refused",
		"that skill writes only the two config files",
		"`gabo check` does not read the profile, so the save runs no `gabo check`",
		"Read the profile back",
		"reported as unverified",
	})
}

// Intent 0018 T3: docs/setup-prompt.md describes the mode in a section outside
// the embedded prompt block, which TestPublishedPromptMatchesEmbedded keeps
// byte-identical.
func TestSyncModeIsDocumentedOutsideThePromptBlock(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../../../docs/setup-prompt.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	const begin = "<!-- gabo-setup-prompt:begin -->"
	heading := strings.Index(doc, "\n## Sync config\n")
	block := strings.Index(doc, begin)
	if heading < 0 || block < 0 || heading > block {
		t.Fatalf("docs/setup-prompt.md: want a ## Sync config section before the prompt block (heading at %d, block at %d)", heading, block)
	}
	sec := doc[heading:block]
	requireAll(t, sec, []string{
		"local or remote",
		"Enter or ok keeps local",
		"`profile.yaml`",
		"Save",
		"never copies",
	})
}

// The plugins/codex copy of the setup skill is identical to the embedded one.
func TestSyncSetupSkillMirrorIsIdentical(t *testing.T) {
	t.Parallel()
	a, err := os.ReadFile("gabo-setup/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../../../plugins/codex/skills/gabo-setup/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Errorf("plugins/codex/skills/gabo-setup/SKILL.md differs from apps/cli/internal/skills/gabo-setup/SKILL.md")
	}
}
