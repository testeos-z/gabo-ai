# End-to-end mini wave (T034)

`go test ./...` skips this package: its tests carry the build tag `e2e`.

```bash
go -C apps/cli test -tags e2e ./e2e/   # or: bun run --cwd apps/cli test:e2e
```

Needs `bash`, `git`, `jq`, `flock` and `setsid`; runs offline in a `t.TempDir()`.

## What the test does

1. Builds `gabo` at a release version, runs `gabo init` (catalog copy and the user-level `gabo-reviewer` skill in a fake `HOME`) and `gabo check` on a sample repository whose `tasks` and `memory` slots point at fake adapters.
2. Plays wave 1 the way `gabo-cycle` and `gabo-coder` describe it, with two tasks (T002 blocked by T001), a fake tracker and a fake memory (JSON files) and a local bare repository standing in for GitHub.
3. Every PR head is reviewed by the installed reviewer scripts with the fake `gh` and `claude` of their own tests and the real `gabo check` as `review.checks`.
4. The first run stops when T002's coder is about to start; a stray worktree outside the ledger is planted; a fresh cycle resumes from tracker and memory only.

It checks: one verdict per head, merges pinned to the reviewed head, statuses written and re-read (one dropped write is caught), no DONE task reworked, every created worktree and branch in the ledger and removed, the stray worktree reported and kept, the wave report sections of `gabo-wave-report`, and no prompt in auto mode. Scope: the prompt check covers the stand-in coder and cycle, not the shipped agents (those run with no stdin here, and the demo checks them); resume is exercised for a task with a live ledger worktree, not for the open-PR reattach or the disagreement branches of D17. `TestSkillsCarryTheRulesTheWaveFollows` pins the skill sentences the played steps follow.

## Demo with real agents

`demo-mini-wave.sh <owner>/<throwaway-repo>` builds a local sample in a scratch dir, runs `gabo check` and prints the steps and checklist for a real wave in Orca. It never pushes or creates issues itself and refuses the gabo-ai repository and its former name gbai-team.
