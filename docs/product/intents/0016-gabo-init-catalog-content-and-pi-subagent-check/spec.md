# Spec 0016 gabo init: catalog content and Pi subagent check

| field | value |
|---|---|
| intent | [0016-gabo-init-catalog-content-and-pi-subagent-check](intent.md) |
| status | Draft (awaits owner acceptance) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`: the decisions and answers marked **Proposed (auto)** are gabo-pm's proposals for the owner's one-pass approval. The owner's own words are quoted, not paraphrased.

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`):

- **Stamp-only branch.** `apps/cli/internal/slices/initcmd/usecase.go` lines 90-91: `case hasStamp && cur.compare(bin) == 0` returns `catalog is current (<version>); nothing changed`, with no content comparison. The version-change path (lines 93-110) copies the catalog and prints `catalog updated <old> → <new>` and `next: ask your agent to rebuild the agents, then run gabo check`.
- **Pi warning.** `apps/cli/internal/engineconfig/pi.go` `piRule` (lines 30-37) appends `gabo refuses to build on Pi without a subagent package:`, `SubagentRefusal` and the step `install a subagent package (see the refusal above)` unconditionally.
- **Package rule.** `.gabo/skills/gabo-runtime-pi/SKILL.md` step 2: `pi list` lists user and project packages and writes nothing; a subagent package is present when it lists `npm:gentle-pi` or a `git:` source of gentle-pi, or when `gentle-shell` is on `PATH`.
- **Documented rule.** `docs/product/TRD.md` 9.4 line 479: "Existing project, stamp equals the binary version. Changes nothing in the project, prints `catalog is current (<version>); nothing changed`". Lines 402 and 661 tie the user-level reviewer to its `VERSION`. Decision 14 (line 702) refreshes the copy "when the stamp is older or missing". The refresh itself is `copyCatalog` (`usecase.go` lines 151-170): it removes the copy folder, writes the embedded files and rewrites the stamp, so it replaces the copy wholesale.
- **Owner decision on overwrite (2026-10-27, PR #75, merged).** A refresh replaces the catalog folder wholesale and overwrites local edits, because customization belongs in adapters. The same-version refresh of S1 uses the same copy.
- **User-level reviewer.** Probe P2a finding 4 ([#414 comment](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366)): `gabo init` replaces the user-level reviewer only when `VERSION` differs; a dev build with the same stamp kept a stale script (`REVIEWER_ENGINE … got 'session'`, `verdict=error`). The [#463 comment](https://github.com/testeos-z/gabo-ai/issues/463#issuecomment-6703389898) names this as the same root cause.
- **Owner, verbatim (2026-11-11):** "gabo init solo compara versiones, no contenido. Sigue vigente y el probe P2a en #414 le suma hallazgos nuevos."

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When `.gabo/VERSION` equals the binary version and the embedded catalog differs from the copied `.gabo/skills/` tree (D1), `gabo init` shall replace the copy wholesale, as a version change does (`copyCatalog`), and print the content-refresh report (D2) with its `next:` line, and exit 0. The same comparison applies to the user-level reviewer skill (`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`), which is replaced and reported the same way. | Given `.gabo/VERSION` = `0.2.10`, a binary `0.2.10` and one file of `.gabo/skills/` that differs from the embedded catalog, When `gabo init` runs, Then that file equals the embedded bytes, a file the copy has but the catalog does not is removed, the output names the content refresh and its `next:` line, and the exit code is 0. Given the user-level reviewer differs at the same version, Then it is replaced and reported. Given `gabo.yaml` and the user config exist, Then neither is written. | Go unit tests in `apps/cli/internal/slices/initcmd` with a fake file system (same version, one changed file, changed reviewer); golden output for the report |
| S2 | When the stamp equals the binary version and the copied catalog and the user-level reviewer equal the embedded ones, `gabo init` shall print `catalog is current (<version>); nothing changed` as today and write nothing. | Given a project whose catalog and reviewer equal the binary's, When `gabo init` runs, Then the catalog line and the rest of the output equal the 0.2.10 golden output, and no file is written. | existing `initcmd` golden tests unchanged; one new test for an equal reviewer |
| S3 | When `gabo init` builds the Pi section (`pi` in `runtimes.*`) and a subagent package is present, it shall print no `gabo refuses to build on Pi without a subagent package:` line, no `SubagentRefusal` text and no `install a subagent package` step. A package is present when `pi list` (read-only) lists `npm:gentle-pi` or a `git:` source of gentle-pi, or `gentle-shell` is on `PATH`. The check runs before those lines are built (D3). | Given a fake `pi` on `PATH` whose `pi list` prints `npm:gentle-pi`, When `gabo init` runs with `pi` in `runtimes.*`, Then no refusal line, refusal text or install step is printed. Given `gentle-shell` on `PATH` and no `pi list` entry, Then the same. Given the fake `pi` writes anything, Then the test fails. | Go unit tests in `apps/cli/internal/engineconfig` and `initcmd` with fake `pi` and `gentle-shell` executables on a temporary `PATH` |
| S4 | When no subagent package is present, or `pi` is not on `PATH`, or `pi list` exits non-zero or times out, `gabo init` shall print the refusal line, the refusal text and the step exactly as today. | Given no package, Then today's three lines. Given `pi` not on `PATH`, Then today's lines (issue #463 acceptance). Given a `pi list` that exits 1, Then today's lines. | Go unit tests, one per case; golden output |
| S5 | Every other `gabo init` and `gabo check` output stays byte-identical, and this intent adds no write outside the copied catalog and the user-level reviewer skill. `gabo check` is unchanged. | Given `go -C apps/cli build ./...`, `go -C apps/cli vet ./...` and `go -C apps/cli test ./...`, Then all pass with only the S1, S3 and S4 tests added and no golden file of `gabo check` changed. Given a Pi project, Then the package check writes no file. | the three `go -C apps/cli` commands; review of the golden diff |

## 2. Design

Affected components: `apps/cli/internal/slices/initcmd` (the catalog and reviewer comparison in `usecase.go`) and `apps/cli/internal/engineconfig` (`piRule` in `pi.go`), with their tests and golden files. No new config key, no new file and no change to `gabo check`.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | How `gabo init` decides that content differs when the versions are equal | (a) compare the embedded catalog with the copied tree file by file (paths and bytes) on every run; (b) store a digest beside `.gabo/VERSION` at copy time and compare digests | **Proposed (auto):** (a). Reason: it needs no new file or format, it works for projects created before this intent, and it matches the owner's "compara contenido". (b) would not see a change made inside the copy after the stamp was written. | The catalog is read on each run; the cost is a file comparison. A local edit inside `.gabo/skills/` is replaced at the same version, as the 2026-10-27 owner decision already states for a refresh (evidence base above). The reviewer uses the same rule. |
| D2 | Report lines for a same-version refresh | (a) reuse the version-change line with a reason; (b) a new wording | **Proposed (auto):** (a), with the reason in brackets: `catalog updated <version> (content changed)` and `reviewer skill updated <version> (content changed)`, each followed by the same `next:` line as the version change. Reason: the owner already reads the version-change lines, and the reason tells them why the same version changed. | The version-change lines stay as they are; only the new reason text is added. |
| D3 | When and how the Pi refusal is decided | (a) run the package check (read-only `pi list`, then `gentle-shell` on `PATH`) before the Pi lines are built, and warn only when no package is found; (b) keep the warning and add a note when a package exists | **Proposed (auto):** (a). Reason: the issue asks to check before warning, and a note would still show the refusal. The rule is the one `gabo-runtime-pi` step 2 already uses, so the two stay the same. | `pi list` runs once per `gabo init` with `pi` in `runtimes.*`. A missing `pi`, or a `pi list` that fails or times out, counts as no package (S4). |

## 3. UX and app flow (optional)

Not applicable: `gabo init` is a CLI with text output and no screen. The lines this change adds or keeps are in S1, S3, S4 and D2; their wording follows the existing `gabo init` report lines.

## 4. DB migration steps (optional)

Not applicable: the change stores no data. The copied catalog and the user-level reviewer skill are files that `gabo init` replaces; no migration is needed.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R32 | R32 says `gabo init` never touches `gabo.yaml` or the user config. | none: S5 keeps both untouched; the reviewer refresh writes only the user-level reviewer skill that R32 already installs. |
| PRD R32 and TRD 2 (the Go CLI is `init`, `check`, `version` and `prompt`) | A content check runs inside `init`. | none: the change stays inside `init`. |
| `gabo-runtime-pi` step 2 | The package rule exists in Go (S3) and in the runtime skill. | none: the same rule. A later change to the rule must update both; this is a maintenance note, not a conflict. |
| PRD R33 | `gabo check` stays read-only and independent of the engine. | none: `gabo check` is not changed. |
| TRD 9.4 line 479, lines 402 and 661, decision 14 (line 702) | They say that a same-version init "changes nothing" and that the user-level reviewer is replaced when its `VERSION` differs. | **conflict with the owner's words of 2026-11-11** ("gabo init solo compara versiones, no contenido"). Resolution: S1 and S2 replace those rules for the content case. When the owner accepts this spec, the same change amends those TRD lines; gabo-dev-lead writes the TRD text in the plan. The TRD is a PRD/TRD convention, so the owner decides the amendment, not directiva alone. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Do the P2a findings (1), (2), (3) and (5) of the #463 comment join this intent? | open, see intent 0016 open question 1. **Proposed (auto):** separate intents; the owner decides. |
| 2 | On a same-version refresh, may a local edit inside `.gabo/skills/` be replaced (D1)? | answered by the owner's decision of 2026-10-27 (PR #75): a refresh replaces the catalog wholesale and overwrites local edits. |
| 3 | `pi list` exits non-zero or times out: is the package present or absent? | answered in S4: absent, so today's warning stays. |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
