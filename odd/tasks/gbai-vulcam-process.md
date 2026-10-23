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
- [x] T2 Review hardening: marker author check, nonce-delimited diff in the prompt, unforgeable verdict line prefix, one conflict comment per head, keep-worktree flag. [delegated: 2+ non-trivial files]
- [x] T3 Repo ruleset without required approval; audit checks the marker comment on the head. [delegated: 2+ non-trivial files]
- [ ] T4 Single-user reviewer: provision drops the reviewer/reviewer-ci users, reviewer runs as the owner with its own state dir, refuses a dirty checkout; docs/cloud-agent-flow.md updated. [delegated]
- [ ] T5 `gbai worktree close <path>` slice with the approval checks; deletes the merged branch, never `--force`. [delegated]
- [ ] T6 Skills + config: gbai-coder (clean worktree, scratch in temp dir, leftovers report, container reuse, heavy-suite limits, immutable migrations), gbai-cycle (board Backlog->Ready, never skip silently, worktree close via T5, one owner block per wave close), gbai-reviewer-notes, `parallelism` config key. [delegated]

## Progress

- 2026-10-23: feature document created after read-only mapping of the 8 improvements.
- 2026-10-23: T1 done (route: delegated, trigger: 2+ non-trivial files in the review slice). Domain owns `CommentClean`/`CommentBlocking`/`CommentFooter`, `Marker(sha)`, `MarkerSHA(body)` and `CleanVerdictFor(comments, head)`; `DecideMerge` takes the PR comments (oldest first) and needs the newest marker comment to sign the head and open with `No blocking issues found.`. `GitHub.Approve` removed; `GitHub.Comments` added (`gh api --paginate .../issues/<n>/comments --jq '[.[].body]'`). A failed comment listing never opens the gate. RED: `go test ./internal/slices/review/` failed to build (undefined `Marker`/`MarkerSHA`, missing `Comments`, `DecideMerge` arity). Evidence: `cd apps/cli && go test ./... && go vet ./...`: 27 packages ok, vet clean; `gofmt -l .`: no output. Commit: `feat(review): sign verdict comments and gate merge on the marker instead of approving`.
- 2026-10-23: T1 RDD: commit 3ec059d, assessment `medium`, `review_due=false` (`under_budget`, pending in the slice).
- 2026-10-23: T2 done (route: delegated, trigger: 2+ non-trivial files in the review slice). (1) `GitHub.Comments` returns `Comment{Author, Body}` (`--jq '[.[] | {author: .user.login, body: .body}]'`); new port `GitHub.Login` (`gh api user --jq .login`) instead of a config value, because the login must be the identity gh actually posts with and a config copy could drift from it (T4 moves the reviewer to the owner's account). `DecideMerge`/`CleanVerdictFor` take the reviewer login and only count its marker comments; an empty login or a failed lookup keeps the gate closed. (2) `run.go` draws a 128-bit `crypto/rand` hex nonce per prompt; the template delimits the diff with `<diff-NONCE>...</diff-NONCE>` and says only that pair delimits it; byte cap and truncation note unchanged. (3) Unforgeable summary: `review run` prints every note line (which may carry check or engine output) through `Quote`, prefixing `| `, so the summary on stdout is the only unquoted line that can parse; `review notify` now relays only when exactly one line matches PR/SHA, otherwise `verdict=error`, so a forged line can at most downgrade a result, never upgrade it. (4) On a conflict the run reads the thread first and skips the comment when the newest reviewer marker comment signs the same head with the fixed conflict line (`Head ... does not merge cleanly onto ...`, written only by `CommentBody`); statuses are always set. (5) `gbai review run head <n> <sha> --keep-worktree` -> `Factories.Run(ctx, keepWorktree)` -> `RunConfig.KeepWorktree` skips both deferred cleanups and notes the kept name. RED: `go test ./internal/slices/review/...` failed to build (undefined `Comment`, `GitHub.Login`, `Factories.Run` arity). Evidence: `cd apps/cli && go test ./... && go vet ./...`: all packages ok, vet clean; `gofmt -l .`: no output. Commit: `feat(review): harden verdict channel against forged markers, diffs and lines`.
- 2026-10-23: T2 RDD: commit 46688e8. RDD slice review of T1+T2 (base b1ab6fa, 15 files, 827 lines, `medium`, granted) approved and acknowledged, lineage `review-b95abae00be0f972`; two non-blocking findings (keep-worktree rerun collision, vacuous forged-relay assertion) fixed in T3. The reviewed boundary is now 46688e8.
- 2026-10-23: T3 done (route: delegated, trigger: 2+ non-trivial files across repo, audit, review and a new kernel package). (1) Ruleset `pull_request` rule now has `required_approving_review_count: 0`; required statuses `reviewer/tests` and `reviewer/review`, empty bypass list and the other rules unchanged; `repo setup --help` text updated. (2) The verdict comment format moved to the shared kernel package `internal/kernel/verdict` (`Clean`/`Blocking`/`Footer`, `Marker`, `MarkerSHA`, `Comment`, `NewestSigned`, `Lines`, `OpensClean`, `CleanFor`), because no slice imports another (no arch test exists, but the layout rule puts shared types in `internal/kernel`); `review` re-exports it (`CommentClean`, `type Comment = verdict.Comment`, `Marker`, `MarkerSHA`, `CleanVerdictFor`) so its callers are unchanged. Audit port `GitHub.Comments(ctx, number)` (adapter: `gh api --paginate repos/<repo>/issues/<n>/comments --jq '[.[] | {author: .user.login, body: .body}]'`); audit flags a merged PR with no bot marker comment (marker comments by other authors ignored), a newest marker for another sha, a newest marker that does not open with `No blocking issues found.`, or unreadable comments; bot login match is case-insensitive and ignores `[bot]`. (3) Review fixes: `adapters.Git.AddWorktree` removes a stale same-name worktree first (name validated and resolved under the reviewer work dir, never the clone), proven with a real-git test that adds the same worktree twice; the forged-relay test now asserts the stdout summary, exactly one parseable summary line in the merged stream and the exact relayed summary, and adds a clean run whose cleanup note carries a forged downgrade (a mutation dropping `Quote` makes it fail). RED: `go test ./internal/...` failed to build `kernel/verdict` and `audit` (undefined `Marker`/`Comment`/`Comments`), `TestRulesetBody` failed (`required_approving_review_count:1`), `TestGitAddWorktreeReplacesAKeptWorktree` failed (`fatal: '.../pr-12-0123456' already exists`). Evidence: `cd apps/cli && go test ./... && go vet ./...`: 28 packages ok, vet clean; `gofmt -l .`: no output.

## Next step

T4 (single-user reviewer: provision, state dir, dirty-checkout refusal, docs). Note: the root-owned `review-run` wrapper passes only `head <n> <sha>`, so `--keep-worktree` is for manual runs as the reviewer.
