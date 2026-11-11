# Intent 0016 gabo init: catalog content and Pi subagent check

| field | value |
|---|---|
| intent id | 0016-gabo-init-catalog-content-and-pi-subagent-check |
| status | Specified |
| owner | gabo-pm |
| product card | none: source issue [#463](https://github.com/testeos-z/gabo-ai/issues/463) has no parent card (checked 2026-11-11) |
| date | 2026-11-11 |

## Problem

`gabo init` decides that the copied catalog and the user-level reviewer are current from the version stamp alone. When the binary has the same version as the stamp but different embedded content (a dev build, or a change made without a version bump), `gabo init` prints `catalog is current (<version>); nothing changed` and leaves the old `.gabo/skills/` and reviewer scripts in place.

Probe P2a found the same root cause for the user-level reviewer. It was replaced only when `VERSION` differed, so a dev build with the same stamp kept a stale script, which failed with `REVIEWER_ENGINE … got 'session'` and `verdict=error` ([#414 comment](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366)).

Separately, on a project with Pi in `runtimes.*`, `gabo init` always prints `gabo refuses to build on Pi without a subagent package:` with the refusal text and the step "install a subagent package", even when gentle-pi is installed. The check is not made before the warning is built.

Owner, verbatim (2026-11-11): "gabo init solo compara versiones, no contenido. Sigue vigente y el probe P2a en #414 le suma hallazgos nuevos."

## Who is affected

Owners and contributors who run `gabo init` on a build whose version matches the project stamp: dev builds, and any build whose catalog changed without a version bump. Every such `gabo init` run leaves the stale catalog in place. Projects with Pi in `runtimes.*` see the subagent warning on every `gabo init` run, including those where gentle-pi is installed (seen on the owner's machine with PR [#460](https://github.com/testeos-z/gabo-ai/pull/460), Pi 1.4.0 with gentle-pi, and in probe P2a on the server).

## Desired outcome

- A changed catalog or user-level reviewer is refreshed and reported by `gabo init` even when the version stamp is equal. An equal catalog keeps today's output.
- On a Pi project with a subagent package installed, `gabo init` prints no refusal. Without a package, or without `pi` on `PATH`, today's warning and step stay.
- The P2a findings that are not about catalog content or the Pi package check are not added silently; the owner decides their scope (open question 1).

## Constraints

- PRD R32: `gabo init` refreshes the copied skill catalog and installs the user-level reviewer, and it never touches `gabo.yaml` or the user config. The Pi package check only reads (`pi list` writes nothing).
- TRD 9.4 states the version-only rule today (line 479 "changes nothing" when the stamp matches; lines 402 and 661 for the reviewer; decision 14). This intent changes that rule for content differences; the TRD lines are amended when the owner accepts the spec.
- PRD R33 and `gabo check` are out of scope (issue #463: `gabo check` impact none).
- Output for every case this intent does not name stays byte-identical to the golden output in `apps/cli/internal/slices/initcmd/`.
- Release: the next release after 0.2.10 (issue #463 label `release:next`).
- Not in scope: runtime skill contents, and intent 0014 ([#444](https://github.com/testeos-z/gabo-ai/issues/444)).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Do the P2a findings in the [#463 comment](https://github.com/testeos-z/gabo-ai/issues/463#issuecomment-6703389898) that are not init behavior join this intent? They are: (1) Pi waves under `pi --print` end after one turn; (2) gabo-cycle on Pi must register each task worktree with `session_worktree_register`; (3) gentle-pi asks before `git push` unless autonomous mode is on, and `gabo init` would have to write that mode; (5) small closure and PR-body bugs. | open. **Proposed (auto):** no; each gets its own intent. Findings (1), (2) and (5) change runtime skills, docs or gabo-cycle, and (3) turns off a gentle-pi confirmation guard for `git push`, which is a safety decision for the owner. |
| 2 | On a same-version refresh, a local edit inside `.gabo/skills/` is replaced by the embedded file. Is that acceptable? | answered by the owner's decision of 2026-10-27 (PR #75, merged): a refresh replaces the catalog wholesale and overwrites local edits, because customization belongs in adapters. The spec applies it to same-version refreshes too. |

## Links

| kind | reference |
|---|---|
| product card | none (#463 has no parent) |
| source issue | [#463](https://github.com/testeos-z/gabo-ai/issues/463) (stays open) |
| probe | [#414 comment, P2a rerun](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366) |
| PRD | [PRD](../../PRD.md), R32 and R33 |
| spec | [spec](spec.md) |
| plan | — (written by gabo-dev-lead after the owner accepts the spec) |
| intent issue | [#489 I0016: gabo init: catalog content and Pi subagent check](https://github.com/testeos-z/gabo-ai/issues/489) (no parent card) |
| related intent | 0013-several-engines-in-parallel-in-one-project (probe of PR #460, issue #393) |
