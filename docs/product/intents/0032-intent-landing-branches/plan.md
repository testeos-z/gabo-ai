# Plan 0032 Intent landing branches

Spec accepted by the owner 2026-11-26 (`spec.md`, status Accepted).
This plan lands before the implementation, per the reviewer's compliance
finding on PR #913 (planning only, no code before an accepted plan).

## Goal

One landing branch per intent (`<NNNN-slug>`, homonymous with the intent
folder), task PRs targeting it, finished intents pouring into `dev` through
the same reviewer gate. No new `gabo.yaml` key.

## Phases

- P1 — Land this plan on `dev` through the intent branch
  (`0032-intent-landing-branches`), as a single-doc pour. No code.
- P2 — Land the T2 local-workflow implementation (PR #913: contract and
  adapter skill edits, cycle pour step, reviewer base guard). It merges only
  after P1 is on `dev`.
- P3 — Open the 0031 and 0032 waves (one wave = one intent branch, cut from
  `dev`, protected like `dev`); task PRs target the intent branch.
- P4 — Release 0.6.0 after both intents pour into `dev`.

## Pour rule refinement (implements S5, D5)

The intent PR into `dev` auto-merges on a clean verdict and no conflicts,
with two reviewer-side restrictions (reviewer finding, PR #913):

- Every commit in the pour diff must be provably approved: reachable from
  `dev` (arrived via a sync), or associated with a merged PR whose base is
  the intent branch (a task PR or an `<NNNN-slug>-sync` PR). Anything else
  keeps the human-only hold.
- Built-in human-only paths (`.gabo/VERSION`, the `.gabo/skills/`
  subtree, the reviewer's own source) always hold, pour or not. Only
  configured `review.human_only_paths` are bypassed on a proven pour,
  because they were already approved on the task PRs.

Sync (S9) is unchanged: cut `<NNNN-slug>-sync` from the intent branch,
merge `dev` into it, resolve there, PR it into the intent branch through
the normal review. Never a rebase, never a force-push.

## Tests

- Reviewer shell suite: pour-proof fixtures (proven pour merges despite
  configured human-only paths; unproven pour and built-in paths hold).
- `go -C apps/cli test ./...` green before every push.
- Reviewer verdict `clean` on the P2 PR and on each intent PR.

## Acceptance

- 0031 and 0032 each landed on their intent branch and poured into `dev`.
- 0.6.0 tagged after both pours.
