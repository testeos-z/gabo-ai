# Spec 0016 gabo init: Pi subagent-package check

| field | value |
|---|---|
| intent | [0016-gabo-init-pi-subagent-package-check](intent.md) |
| status | Draft (awaits owner acceptance) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`: the decisions and answers marked **Proposed (auto)** are gabo-pm's proposals for the owner's one-pass approval. The owner's own words are quoted, not paraphrased.

Narrowing (2026-11-11). Owner, verbatim, by voice, answering "should intent 0016 keep only the Pi package check?": "si ... no me gusta tener nada duplicado". Item 1 of #463 (the catalog content comparison, former requirements S1 and S2 of this spec and the reviewer part of S1) is now intent 0015 task T14 ([#524](https://github.com/testeos-z/gabo-ai/issues/524), plan decision PD9 of `docs/product/intents/0015-one-subagent-layer-on-every-engine/plan.md`), implemented by PR [#526](https://github.com/testeos-z/gabo-ai/pull/526) (merged). This spec keeps only item 2 of #463, the Pi package check, and the TRD amendment that nothing else carries (section 5).

Evidence base (checked 2026-11-11 on `origin/dev` after PR #526):

- **Pi warning.** `apps/cli/internal/engineconfig/pi.go` `piRule` (from line 30): it appends `gabo refuses to build on Pi without a subagent package:`, `SubagentRefusal` and the step `install a subagent package (see the refusal above)` unconditionally.
- **Package rule.** `.gabo/skills/gabo-runtime-pi/SKILL.md` step 2: `pi list` lists user and project packages and writes nothing; a subagent package is present when it lists `npm:gentle-pi` or a `git:` source of gentle-pi, or when `gentle-shell` is on `PATH`.
- **What T14 did.** PR #526 changed `apps/cli/internal/slices/initcmd/` (a content hash in `.gabo/CATALOG_HASH` next to `.gabo/VERSION`), the README and `docs/setup-prompt.md`. It did not change `docs/product/TRD.md`, and it compares the catalog copy only, not the user-level reviewer.
- **The TRD text that is now stale.** `docs/product/TRD.md` 9.4 line 490: "Existing project, stamp equals the binary version. Changes nothing in the project, prints `catalog is current (<version>); nothing changed` and exits 0"; decision 14 (line 716) refreshes the copy "when the stamp is older than the binary or missing". The 0015 plan (T14, PD9, Q4, risks table) and the 0015 spec (section 5, S10) amend other TRD 9.4 lines (the "One layer" bullet, the `experimental.subagent_depth` key) but not this one.
- **The source.** Issue #463, item 2, and the P2a probe comment ([#414](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366)).

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When `gabo init` builds the Pi section (`pi` in `runtimes.*`) and a subagent package is present, it shall print no `gabo refuses to build on Pi without a subagent package:` line, no `SubagentRefusal` text and no `install a subagent package` step. A package is present when `pi list` (read-only) lists `npm:gentle-pi` or a `git:` source of gentle-pi, or `gentle-shell` is on `PATH`. The check runs before those lines are built (D1). | Given a fake `pi` on `PATH` whose `pi list` prints `npm:gentle-pi`, When `gabo init` runs with `pi` in `runtimes.*`, Then no refusal line, refusal text or install step is printed. Given `gentle-shell` on `PATH` and no `pi list` entry, Then the same. Given the fake `pi` writes anything, Then the test fails. | Go unit tests in `apps/cli/internal/engineconfig` and `initcmd` with fake `pi` and `gentle-shell` executables on a temporary `PATH` |
| S2 | When no subagent package is present, or `pi` is not on `PATH`, or `pi list` exits non-zero or times out, `gabo init` shall print the refusal line, the refusal text and the step exactly as today. | Given no package, Then today's three lines. Given `pi` not on `PATH`, Then today's lines (issue #463 acceptance). Given a `pi list` that exits 1, Then today's lines. | Go unit tests, one per case; golden output |
| S3 | Every other `gabo init` and `gabo check` output stays byte-identical, and the Pi check writes no file. `gabo check` is unchanged. | Given `go -C apps/cli build ./...`, `go -C apps/cli vet ./...` and `go -C apps/cli test ./...`, Then all pass with only the S1 and S2 tests added and no golden file of `gabo check` changed. Given a Pi project, Then the package check writes no file. | the three `go -C apps/cli` commands; review of the golden diff |
| S4 | The TRD 9.4 text for `gabo init` shall describe the behavior as it now is: with a stamp equal to the binary version, `gabo init` compares the content hash (`.gabo/CATALOG_HASH`) of the embedded catalog with the one stamped at the last copy, prints `catalog is current (<version>); nothing changed` when they are equal and refreshes the copy and reports it when they differ (0015 T14, PR #526); and for a Pi target it prints the subagent-package refusal only when the check of S1 finds no package. The same decision 14 sentence (line 716) shall say it. | Given the TRD after the change, Then line 490 no longer says that a same-version init "Changes nothing" and names `.gabo/CATALOG_HASH`, and decision 14 matches. Given the Pi rule, Then 9.4 says the refusal depends on the package check. | review of the TRD diff in the plan's task; `gabo check` passes |

## 2. Design

Affected components: `apps/cli/internal/engineconfig` (`piRule` in `pi.go`) and `apps/cli/internal/slices/initcmd` with their tests and golden files, and `docs/product/TRD.md` 9.4 (S4). No new config key, no new file written by the Pi check and no change to `gabo check`.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | When and how the Pi refusal is decided | (a) run the package check (read-only `pi list`, then `gentle-shell` on `PATH`) before the Pi lines are built, and warn only when no package is found; (b) keep the warning and add a note when a package exists | **Proposed (auto):** (a). Reason: the issue asks to check before warning, and a note would still show the refusal. The rule is the one `gabo-runtime-pi` step 2 already uses, so the two stay the same. | `pi list` runs once per `gabo init` with `pi` in `runtimes.*`. A missing `pi`, or a `pi list` that fails or times out, counts as no package (S2). |

## 3. UX and app flow (optional)

Not applicable: `gabo init` is a CLI with text output and no screen. The lines this change keeps or drops are in S1 and S2; their wording is today's.

## 4. DB migration steps (optional)

Not applicable: the change stores no data.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R32 | R32 says `gabo init` never touches `gabo.yaml` or the user config. | none: S3 keeps both untouched and the Pi check only reads. |
| TRD 9.4 line 490 and decision 14 (line 716): a same-version `gabo init` "Changes nothing" | Stale since 0015 T14 (PR #526) made `gabo init` compare the catalog content; owner words of 2026-11-11 ("gabo init solo compara versiones, no contenido"). | **This spec keeps the amendment** (S4), because the 0015 plan and spec do not carry it and PR #526 did not edit the TRD. Accepting this spec accepts the TRD change; gabo-dev-lead writes the text in this intent's plan. A TRD convention change, decided by the owner's acceptance. If a later 0015 task carries the TRD text, this row and S4 are dropped. |
| TRD 9.4 lines 413, 674 and 675 (the user-level reviewer is replaced when its own `VERSION` differs) | A content comparison for the reviewer would change them. | none now: the reviewer is not compared by content, so those lines stay true; the question is open question 2 of the intent. |
| `gabo-runtime-pi` step 2 | The package rule exists in Go (S1) and in the runtime skill. | none: the same rule. A later change to the rule must update both; this is a maintenance note, not a conflict. |
| PRD R33 | `gabo check` stays read-only and independent of the engine. | none: `gabo check` is not changed. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Do the P2a findings (1), (2), (3) and (5) of the #463 comment join this intent? | open (owner), see intent 0016 open question 1. **Proposed (auto):** separate intents; the owner decides. |
| 2 | Where does a content comparison for the user-level reviewer go (P2a finding 4; T14 compares the catalog only)? | open (owner), see intent 0016 open question 2. Removed from this spec by the narrowing, carried so it is not lost. |
| 3 | `pi list` exits non-zero or times out: is the package present or absent? | answered in S2: absent, so today's warning stays. |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
