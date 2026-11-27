# Plan 0031 Automerge resumes after a pending review check turns to success

| field | value |
|---|---|
| intent | [0031-automerge-resumes-after-pending-review](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-26 |

Spec accepted by the owner 2026-11-26 (`spec.md` section 7 Acceptance record).
No tracker access was performed for this plan; tracker traceability is not verified.
No commit was made.

## Steps

1. Add the explicit continuation entry point `reviewer.sh resume <pr> <sha>` that re-reads the PR and reaches the unchanged `merge-head.sh` gate for the pinned head, with no new model review and no new verdict comment (S1, S2, D1, D2, D3).
2. Lock the refusal and repeat-invocation behavior: moved heads, pending or failed checks, disabled automerge, human-only paths and eval holds still end at the gate's existing outcomes, and an already-merged PR changes nothing (S3, S4).
3. Ship the deterministic pinned-mock regression suite replaying the #908 pending-to-success sequence on a fixed head, including the installed-mode (`0644`, `bash <script>`) invocation, plus the entry-point docs (S5, D4).
4. No `gabo check` update: the change touches only the `gabo-reviewer` reviewer scripts and their tests plus reviewer docs; no config shape, slot contract, or catalog rule checked by `gabo check` changes (TRD 8.5).

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Continuation `resume` entry point reusing the merge gate | S1, S2, D1, D2, D3 | — | T2, T3 | Test first: stub the model-review path to fail if entered and mock the gate; `reviewer.sh resume <pr> <sha>` validates its two arguments like `head`, re-reads open/branch/draft/pinned-head state, calls `merge-head.sh <pr> <sha>` unchanged, and never invokes the model review nor posts a verdict comment. | S |
| T2 | Refusal matrix and already-merged silence | S3, S4 | T1 | T3 | Test first: matrix over moved head, pending check, failed check, `REVIEWER_AUTO_MERGE` not true, human-only path, eval-notice hold ends unmerged with the gate's existing outcome (`MERGE-SKIPPED` or `NEEDS-HUMAN-MERGE`); invoking the continuation after the mocked merge issues no second merge, comment, or status, with exactly one merge call in the log. | S |
| T3 | Pinned-mock regression suite, installed mode, and docs | S5, S1, S2, S3, S4, D4 | T1, T2 | — | Test first: new suite under `apps/cli/internal/skills/gabo-reviewer/scripts/tests/` with mocked GitHub status reads switching pending to success on a fixed head asserts first-run `verdict=clean merge=skipped`, one continuation merge of only the pinned head, the S3 refusals, and post-merge silence; the suite invokes scripts via `bash <script>` as the `0644`-installed copies run; reviewer docs list `resume`; script suite and `go -C apps/cli test ./...` pass. | M |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| Continuation drifts from the gate (new checks bypassing `merge-head.sh`) | T1; breaks the #908 scope preservation | T1 calls the existing gate unchanged; T2 asserts every S3 refusal fires at the gate's outcome |
| Flaky live-status timing leaks into the regression test | T3; suite depends on GitHub timing | S5 mocks fix the pending-to-success sequence on a pinned head; no live status reads in the suite |
| Installed-permission behavior diverges from the repo checkout | T3; `0644` copies invoked differently | D4: the suite covers the `bash <script>` invocation the reporter used for installed scripts |

## Open questions

1. Should the continuation also cover `NEEDS-HUMAN-MERGE` and `MERGE-FAILED` exits, or only pending-skip heads? Proposed (auto): only pending-skip heads in this intent (spec Q4, owner answers).
2. Does the owner want the scheduled reviewer run to invoke the continuation on its own once this intent lands, or keep it manual until the landing-branch intent (0032) settles scheduling? Proposed (auto): keep it manual and decide automatic pickup together with 0032 (spec Q5, owner answers).
