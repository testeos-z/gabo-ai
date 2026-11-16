package check

import (
	"testing"
)

// Issue #904: a lockfile whose manager the checks use, without an install
// step, fails with the detected command.
func TestInstallStepMissingFails(t *testing.T) {
	t.Parallel()
	cfg := `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
parallelism: 3
review:
  checks: [bun run test, bun run lint]
  human_only_paths: [.github/]
`
	proj := with(project(), map[string]string{"bun.lock": "{}"})
	r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
	wantFailed(t, r,
		"✗ review.checks has no dependency install step: checks run in a fresh worktree with nothing installed",
		"next: list bun install --frozen-lockfile as the first review.checks entry, then rerun gabo check")
}

// The suggested install command as the first check passes.
func TestInstallStepPresentPasses(t *testing.T) {
	t.Parallel()
	cfg := `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
parallelism: 3
review:
  checks: [bun install --frozen-lockfile, bun run test]
  human_only_paths: [.github/]
`
	proj := with(project(), map[string]string{"bun.lock": "{}"})
	wantPassed(t, run(t, proj, configFiles{projectFile: cfg}, Settings{}), PassedLine)
}

// A lockfile whose manager the checks never use needs no install step:
// Go-only checks pass with a bun.lock present.
func TestUnusedLockfileNeedsNoInstallStep(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{"bun.lock": "{}"})
	wantPassed(t, run(t, proj, configFiles{projectFile: validConfig}, Settings{}), PassedLine)
}

// A Go module needs no install step: go downloads its own dependencies.
func TestGoModuleNeedsNoInstallStep(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{"go.mod": "module example.com/x\n"})
	wantPassed(t, run(t, proj, configFiles{projectFile: validConfig}, Settings{}), PassedLine)
}
