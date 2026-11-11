# Intent 0016 gabo init: Pi subagent-package check and reviewer content check

| field | value |
|---|---|
| intent id | 0016-gabo-init-pi-subagent-package-check-and-reviewer-content-check |
| status | Specified |
| owner | gabo-pm |
| product card | none: source issue [#463](https://github.com/testeos-z/gabo-ai/issues/463) has no parent card (checked 2026-11-11); #463 is a sub-issue of the intent issue #489 |
| date | 2026-11-11 |

## Problem

Two things in `gabo init` are wrong today.

1. On a project with Pi in `runtimes.*`, `gabo init` always prints `gabo refuses to build on Pi without a subagent package:` with the refusal text and the step "install a subagent package", even when gentle-pi is installed. `apps/cli/internal/engineconfig/pi.go` `piRule` appends them unconditionally, and the check is not made before the warning is built. The package rule already exists in the runtime skill `gabo-runtime-pi` (step 2: `pi list`). This is item 2 of issue #463.
2. `gabo init` replaces the user-level reviewer (the copy under `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`) only when its own `VERSION` differs from the binary version. A dev build that shares the version stamp keeps stale scripts: in probe P2a the installed reviewer was stamped `0.2.10` but predated the `session` engine support, and the run failed with `REVIEWER_ENGINE … got 'session'` and `verdict=error` ([#414 comment](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366), finding 4, repeated in the [#463 comment](https://github.com/testeos-z/gabo-ai/issues/463#issuecomment-6703389898)).

Narrowing, owner verbatim (2026-11-11, by voice), answering "should intent 0016 keep only the Pi package check?": "si ... no me gusta tener nada duplicado". Item 1 of #463, `gabo init` comparing the catalog content and not only the version, is intent 0015 task T14 ([#524](https://github.com/testeos-z/gabo-ai/issues/524), plan decision PD9; PR [#526](https://github.com/testeos-z/gabo-ai/pull/526) merged into `dev` as d06b951). This intent does not carry it.

The reviewer content check, owner verbatim (2026-11-11, by voice), answering where it goes: "Sí, en el intent 16". T14 compares the catalog copy only, so the reviewer part is this intent's.

## Who is affected

Owners and contributors whose projects have Pi in `runtimes.*` see the subagent warning on every `gabo init` run, including those where gentle-pi is installed (seen on the owner's machine with PR [#460](https://github.com/testeos-z/gabo-ai/pull/460), Pi 1.4.0 with gentle-pi, and in probe P2a on the server). Anyone who runs `gabo init` from a build whose version equals the installed reviewer's, with different reviewer content, keeps stale reviewer scripts and gets a failed or wrong review.

## Desired outcome

- On a Pi project with a subagent package installed, `gabo init` prints no refusal. Without a package, or without `pi` on `PATH`, today's warning and step stay.
- When the installed reviewer's version equals the binary's but its content differs from the embedded reviewer, `gabo init` replaces it and reports it, the same way task T14 does for the catalog copy; with equal content the output is unchanged.
- The P2a findings that are not about these two checks are not added silently; the owner decides their scope (open question 1).

## Constraints

- PRD R32: `gabo init` never touches `gabo.yaml` or the user config, and installs the user-level reviewer. The Pi package check only reads (`pi list` writes nothing).
- The reviewer content check reuses the hash approach of task T14 (PR #526) and its wording; it does not define a second approach (spec D2).
- The refusal rules of the reviewer install stay as they are: a relative directory, a directory inside the project, a symlink, a path that is not a directory and a write error.
- PRD R33 and `gabo check` are out of scope: `gabo check` does not look at the user copy (TRD 9.4), and issue #463 states a `gabo check` impact of none.
- Output for every case this intent does not name stays byte-identical to the golden output in `apps/cli/internal/engineconfig/` and `apps/cli/internal/slices/initcmd/`.
- Release: the next release after 0.2.10 (issue #463 label `release:next`).
- TRD 9.4 still describes the version-only rules for the catalog and the reviewer; the amendment is in the spec's section 5 (neither the 0015 plan or spec nor PR #526 carries it).
- Not in scope: the catalog content comparison (0015 T14), runtime skill contents, and intent 0014 ([#444](https://github.com/testeos-z/gabo-ai/issues/444)).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Do the P2a findings in the [#463 comment](https://github.com/testeos-z/gabo-ai/issues/463#issuecomment-6703389898) that are not init behavior join this intent? They are: (1) Pi waves under `pi --print` end after one turn; (2) gabo-cycle on Pi must register each task worktree with `session_worktree_register`; (3) gentle-pi asks before `git push` unless autonomous mode is on, and `gabo init` would have to write that mode; (5) small closure and PR-body bugs. | open (owner). **Proposed (auto):** no; each gets its own intent. Findings (1), (2) and (5) change runtime skills, docs or gabo-cycle, and (3) turns off a gentle-pi confirmation guard for `git push`, which is a safety decision for the owner. |
| 2 | Where does a content comparison for the user-level reviewer go (P2a finding 4; T14 compares the catalog copy only)? | answered: in this intent (spec S4, D2). Owner, verbatim, 2026-11-11, by voice: "Sí, en el intent 16". Not in 0015 and not dropped. |

## Links

| kind | reference |
|---|---|
| product card | none (#463 has no parent) |
| source issue | [#463](https://github.com/testeos-z/gabo-ai/issues/463), a sub-issue of #489; stays open until both halves are done: item 1 by 0015 T14 (#524, done) and item 2 by this intent |
| owner of item 1 | intent 0015 task T14, [#524](https://github.com/testeos-z/gabo-ai/issues/524), PR [#526](https://github.com/testeos-z/gabo-ai/pull/526); plan decision PD9 in the 0015 plan |
| probe | [#414 comment, P2a rerun](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366) |
| PRD | [PRD](../../PRD.md), R32 and R33 |
| spec | [spec](spec.md) |
| plan | — (written by gabo-dev-lead after the owner accepts the spec) |
| intent issue | [#489 I0016: gabo init: Pi subagent-package check and reviewer content check](https://github.com/testeos-z/gabo-ai/issues/489) (no parent card) |
| related intent | 0015-one-subagent-layer-on-every-engine (T14); 0013-several-engines-in-parallel-in-one-project (probe of PR #460, issue #393) |
