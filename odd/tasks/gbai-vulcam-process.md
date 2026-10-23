# gbai: vulcam process improvements

Locator: `odd/tasks/gbai-vulcam-process.md` · Engram mirror: `odd/gbai-vulcam-process/tasks` (project gabo-ai)
Branch: `feat/gbai-single-user-reviewer` (from `feat/gbai-cli-mvp` b1ab6fa)

## Objective

Port into gbai the process improvements the vulcam project adopted on 2026-10-21..23 (reviewer, worktrees, containers, board, clean worktrees), adapted to the owner decisions below.

## Problem / why

vulcam ran into a suspended bot account, a full disk (worktrees, e2e copies, podman) and worktrees refused on close because of leftover files. gbai encodes the same flow as a product, so the fixes belong in its reviewer, CLI and embedded skills.

## Owner decisions (2026-10-23)

- Reviewer model: **vulcam single-user mode replaces the separate bot and the reviewer/reviewer-ci OS users.** The reviewer posts with the owner's gh account. It never approves (GitHub forbids self-approval), only comments with a hidden marker plus a visible footer, and sets the `reviewer/tests` and `reviewer/review` statuses. The merge gate requires the newest marker comment to be newer than the head and to start with `No blocking issues found.`.
- Branches: the cycle **keeps deleting merged branches** (gbai rule wins over vulcam's never-delete).
- Stack-specific rules (Postgres image, migrations path) stay in the target repo's AGENTS.md. Embedded skills carry only the generic form: reuse one test container per issue and report it, committed migrations are immutable.

## Scope

- Review slice: marker + footer, no approve, merge gate by marker, nonce-delimited diff, unforgeable verdict lines, one conflict comment per head, keep-worktree flag.
- Repo ruleset and audit: no required approval; audit checks the marker comment.
- Provision and docs: single OS user for the reviewer, its own state dir, refuse to run on a dirty checkout.
- Worktree close: `gbai worktree close <path>` removes only approved worktrees (merged into the integration branch, clean, no ignored secrets, no open PR, not main), never `--force`, deletes the merged branch.
- Embedded skills: gbai-coder, gbai-cycle, gbai-reviewer-notes; config key `parallelism`.
- Out of scope: any mutation of the real server or GitHub; guard hooks in the target repo.

## Constraints

- Test-first with `cd apps/cli && go test ./... && go vet ./...`; `bun run lint` (go vet + gofmt).
- Skill frontmatter description starts with `"Trigger: ` and is at most 252 chars (`internal/skills/skills_test.go`).
- Hexagonal vertical slices, English artifacts, Conventional Commits, no AI attribution.
- ~400 changed lines per task is advisory only.

## Delivery

- Forecast: ~1,500 authored changed lines across 6 tasks, over the 400-line delivery budget. Strategy: `ask-on-risk`; owner answer 2026-10-23: "like vulcam", meaning one PR per task to the integration branch, each reviewed on its own head and merged in order, never to `main`. gabo-ai has no `dev` yet, so the integration branch is `feat/gbai-cli-mvp` until one exists. Push and PR creation stay the owner's call.

## Tasks

- [x] T1 Review verdict channel: hidden marker + visible footer in the comment, drop approve, merge gate by newest marker comment (newer than head, starts with `No blocking issues found.`). [delegated: 2+ non-trivial files]
- [ ] T2 Review hardening: nonce-delimited diff in the prompt, unforgeable verdict line prefix, one conflict comment per head, keep-worktree flag. [delegated]
- [ ] T3 Repo ruleset without required approval; audit checks the marker comment on the head. [delegated]
- [ ] T4 Single-user reviewer: provision drops the reviewer/reviewer-ci users, reviewer runs as the owner with its own state dir, refuses a dirty checkout; docs/cloud-agent-flow.md updated. [delegated]
- [ ] T5 `gbai worktree close <path>` slice with the approval checks; deletes the merged branch, never `--force`. [delegated]
- [ ] T6 Skills + config: gbai-coder (clean worktree, scratch in temp dir, leftovers report, container reuse, heavy-suite limits, immutable migrations), gbai-cycle (board Backlog->Ready, never skip silently, worktree close via T5, one owner block per wave close), gbai-reviewer-notes, `parallelism` config key. [delegated]

## Progress

- 2026-10-23: feature document created after read-only mapping of the 8 improvements.
- 2026-10-23: T1 done (route: delegated, trigger: 2+ non-trivial files in the review slice). Domain owns `CommentClean`/`CommentBlocking`/`CommentFooter`, `Marker(sha)`, `MarkerSHA(body)` and `CleanVerdictFor(comments, head)`; `DecideMerge` takes the PR comments (oldest first) and needs the newest marker comment to sign the head and open with `No blocking issues found.`. `GitHub.Approve` removed; `GitHub.Comments` added (`gh api --paginate .../issues/<n>/comments --jq '[.[].body]'`). A failed comment listing never opens the gate. RED: `go test ./internal/slices/review/` failed to build (undefined `Marker`/`MarkerSHA`, missing `Comments`, `DecideMerge` arity). Evidence: `cd apps/cli && go test ./... && go vet ./...`: 27 packages ok, vet clean; `gofmt -l .`: no output. Commit: `feat(review): sign verdict comments and gate merge on the marker instead of approving`.

## Next step

T2 (review hardening).
