package check

import (
	"path"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// judgeTestFiles records the test_files lock's warning for each path a coder
// writes that is an existing test file (intent 0028 S9, S10, D6) and returns the
// lines to print. Only a coder session is judged, and an off lock judges nothing
// and reads no base. A path whose target is a test file (isTestFile) and is
// tracked at the base commit of origin/<integration_branch> is one warned record,
// with the lock, the rule and the path; the write runs and prints nothing. The
// lock is only off or warn (config refuses other states, PD1). A record that
// cannot be appended prints one line and the write still runs; a base that cannot
// be read counts the file as not tracked, so nothing is recorded.
func (u *UseCase) judgeTestFiles(paths []string, cfg config.Config, call WriteCall) string {
	if !writeSession(call.AgentID, call.Engine) || cfg.Lock(config.LockTestFiles) == config.LockOff {
		return ""
	}
	if u.s.Base == nil || cfg.Repo.IntegrationBranch == "" {
		return ""
	}
	ref := "origin/" + cfg.Repo.IntegrationBranch
	var b strings.Builder
	for _, p := range paths {
		target, ok := u.existingTestTarget(p, ref)
		if !ok {
			continue
		}
		rule := "a coder write to the existing test file " + target
		d := decideFor(cfg, config.LockTestFiles, call)
		if err := u.RecordDecision(config.LockTestFiles, rule, d); err != nil {
			b.WriteString("test_files: the warning could not be recorded (" + err.Error() + "); the write runs\n")
		}
	}
	return b.String()
}

// existingTestTarget returns the first project-relative target of a write to p
// that is a test file tracked at the base commit of ref, as written or with its
// symlinks resolved. An unresolvable path or an unreadable base yields none.
func (u *UseCase) existingTestTarget(p, ref string) (string, bool) {
	targets, err := u.targetsOf(p)
	if err != nil {
		return "", false
	}
	for _, t := range targets {
		if !isTestFile(t) {
			continue
		}
		if tracked, err := u.s.Base.Tracked(ref, t); err == nil && tracked {
			return t, true
		}
	}
	return "", false
}

// isTestFile reports whether rel, a project-relative path, is a test file (intent
// 0028 decision 15): a base name matching *_test.go, *.test.* or *_test.*, or a
// path with a test/ or tests/ directory element.
func isTestFile(rel string) bool {
	base := path.Base(rel)
	for _, pat := range []string{"*_test.go", "*.test.*", "*_test.*"} {
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
	}
	for _, dir := range strings.Split(path.Dir(rel), "/") {
		if dir == "test" || dir == "tests" {
			return true
		}
	}
	return false
}
