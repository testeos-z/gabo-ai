package skills

import (
	"strings"
	"testing"
)

// syncConfig returns the ## Sync config section of the setup skill. A missing
// section is reported and yields an empty string, so every assertion that
// reads it fails on its own.
func syncConfig(t *testing.T) string {
	t.Helper()
	_, sec, ok := strings.Cut(setupDoc(t), "## Sync config")
	if !ok {
		t.Errorf("%s: no ## Sync config section", setupSkill)
		return ""
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	return sec
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
	sec := syncConfig(t)
	first, _, _ := strings.Cut(sec, "\n2. ")
	if !strings.Contains(first, "local or remote") {
		t.Errorf("%s ## Sync config: the first question is not %q", setupSkill, "local or remote")
	}
	requireAll(t, sec, []string{
		"local or remote", "local is the source when the owner does not answer", "another repository",
		"which repository",
	})
}

// Intent 0018 S2 and D3: a local source reads profile.yaml at the user config
// folder with the user file's path rule; a missing profile is reported,
// offers the remote source or the normal setup and writes nothing.
func TestSyncConfigReadsTheLocalProfile(t *testing.T) {
	t.Parallel()
	requireAll(t, syncConfig(t), []string{
		"`profile.yaml`", "`$XDG_CONFIG_HOME/gabo/profile.yaml`", "absolute path",
		"`~/.config/gabo/profile.yaml`", "no profile at <path>", "remote source",
		"normal setup", "writes nothing",
	})
}

// Intent 0018 S3, D6 and PD3: a remote source reads only one gabo.yaml, by
// local path or GitHub URL, read-only, without cloning and storing nothing.
func TestSyncConfigReadsOnlyOneRemoteFile(t *testing.T) {
	t.Parallel()
	sec := syncConfig(t)
	requireAll(t, sec, []string{
		"by local path or by GitHub URL", "`<path>/gabo.yaml`", "read-only", "without cloning",
		"stores nothing", "`gh api -H \"Accept: application/vnd.github.raw+json\" repos/<owner>/<repo>/contents/gabo.yaml`",
		"`?ref=<ref>`", "only when the URL names a branch", "existing `gh` login",
	})
	if strings.Contains(sec, "git clone") {
		t.Errorf("%s ## Sync config: the mode must not clone a repository", setupSkill)
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

// Intent 0018 S3: a repository name or ref from the owner is checked before it
// reaches the gh call, and each value is its own argument, never a shell string.
func TestSyncConfigValidatesRemoteNamesBeforeGh(t *testing.T) {
	t.Parallel()
	requireAll(t, syncConfig(t), []string{
		"[A-Za-z0-9._-]+",
		"`..`",
		"`.`",
		"subdirectory",
		"its own argument",
	})
}
