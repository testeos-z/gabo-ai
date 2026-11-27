# Evidence 0031 — T1–T3 (S1–S4)

Date: 2026-11-27. Worktree task: implement T1–T3 covering spec S1–S4.
`merge-head.sh` and `review-head.sh` are byte-identical to their pinned
digests (the S3/S13 pins in `merge-head.test.sh` still pass): the gate is
reused without changes.

## What was built

- **T1** — `reviewer.sh resume <pr> <40-hex sha>` (`apps/cli/internal/skills/gabo-reviewer/scripts/reviewer.sh`):
  validates its two arguments like `head` (same usage/nargs path, optional
  trailing `engine <engine>` under `session`), runs the same preflight and
  prune best-effort steps, then runs only `merge-head.sh` for the pinned head.
  Prints the gate's outcome line plus the same one-line `VERDICT` shape `head`
  prints for that gate outcome (`verdict=clean` is the head's standing
  verdict, re-verified by the gate for this exact head).
- **T2** — refusal parity (the "pending-commit" pinning): `resume` adds no
  merge path of its own. A moved head, a pending/failed required check,
  `REVIEWER_AUTO_MERGE` not true, human-only paths and eval-notice holds end
  exactly where the gate ends them (`MERGE-SKIPPED` / `NEEDS-HUMAN-MERGE`,
  `VERDICT ... merge=skipped|needs-human-merge`); an already-merged PR ends
  `MERGE-SKIPPED` (not open) with no second merge, no comment, no status.
- **T3** — docs + this file: `SKILL.md` (Hard Rules accept the
  `PR #<n> head <40-hex sha> resume` request form and run exactly
  `reviewer.sh resume <n> <sha>`; Facts documents the continuation) and
  `README.md` (modes list) name `resume`.

## S-coverage

| S | covered by |
|---|---|
| S1 | `review.test.sh` “S1”: seeded final `reviewer/review` + signed clean verdict → `head` still `SKIP … verdict=skipped merge=none` with no merge; `resume` → `VERDICT … verdict=clean merge=merged`, exactly one `--match-head-commit` merge call |
| S2 | “S2”: `resume` with every engine stubbed to fail still merges; zero `claude\|pi\|codex\|opencode` calls; no new comment file |
| S3 | “S3” matrix: moved head, pending `reviewer/tests`, failed `reviewer/review`, automerge off, human-only path (`AGENTS.md`), eval notice — each unmerged with the gate's own outcome |
| S4 | “S4”: `resume` → merge once; PR mock flipped to `MERGED` → second `resume` → `merge=skipped`, still one merge call, no new comment, no `STATUS` write |
| S5 | **not covered** (out of scope for this task): needs a new file under `scripts/tests/`, which requires `skills_test.go` (`reviewerFiles`, count `6`) — outside the allowed edit surfaces of this task |

## Test evidence (test-first: RED then GREEN)

- RED (before the `reviewer.sh` + docs change):
  `bash apps/cli/internal/skills/gabo-reviewer/scripts/tests/review.test.sh`
  → `458 passed, 20 failed` (all 20 failures are the new resume expectations;
  every pre-existing check passed).
- GREEN (after):
  same command → `478 passed, 0 failed`, `all tests passed`.
- Focused Go gate (skill-text side + shell suite):
  `go -C apps/cli test ./internal/skills/ -run 'TestReviewerScriptsShellTests|TestReviewerSkillShipsItsScripts|TestReviewerDocsNameThePolicy'`
  → `PASS`, `ok … 116.334s` (all six shell files pass, including the extended
  `review.test.sh` at 116.31s).
- `go -C apps/cli build ./...` and `go -C apps/cli vet ./internal/skills/` → clean.
