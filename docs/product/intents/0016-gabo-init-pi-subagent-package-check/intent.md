# Intent 0016 gabo init: Pi subagent-package check

| field | value |
|---|---|
| intent id | 0016-gabo-init-pi-subagent-package-check |
| status | Specified |
| owner | gabo-pm |
| product card | none: source issue [#463](https://github.com/testeos-z/gabo-ai/issues/463) has no parent card (checked 2026-11-11); #463 is a sub-issue of the intent issue #489 |
| date | 2026-11-11 |

## Problem

On a project with Pi in `runtimes.*`, `gabo init` always prints `gabo refuses to build on Pi without a subagent package:` with the refusal text and the step "install a subagent package", even when gentle-pi is installed. `apps/cli/internal/engineconfig/pi.go` `piRule` appends them unconditionally, and the check is not made before the warning is built. The package rule already exists in the runtime skill `gabo-runtime-pi` (step 2: `pi list`). This is item 2 of issue #463.

Narrowing, owner verbatim (2026-11-11, by voice), answering "should intent 0016 keep only the Pi package check?": "si ... no me gusta tener nada duplicado". Item 1 of #463, `gabo init` comparing the catalog content and not only the version, is now intent 0015 task T14 ([#524](https://github.com/testeos-z/gabo-ai/issues/524), plan decision PD9; implemented by PR [#526](https://github.com/testeos-z/gabo-ai/pull/526), merged). This intent no longer carries it.

## Who is affected

Owners and contributors whose projects have Pi in `runtimes.*`: they see the subagent warning on every `gabo init` run, including those where gentle-pi is installed (seen on the owner's machine with PR [#460](https://github.com/testeos-z/gabo-ai/pull/460), Pi 1.4.0 with gentle-pi, and in probe P2a on the server).

## Desired outcome

- On a Pi project with a subagent package installed, `gabo init` prints no refusal. Without a package, or without `pi` on `PATH`, today's warning and step stay.
- The P2a findings that are not about the Pi package check are not added silently; the owner decides their scope (open question 1).
- The user-level reviewer is a gap that neither this intent nor task T14 covers; the owner decides where it goes (open question 2).

## Constraints

- PRD R32: `gabo init` never touches `gabo.yaml` or the user config. The Pi package check only reads (`pi list` writes nothing).
- PRD R33 and `gabo check` are out of scope (issue #463: `gabo check` impact none).
- Output for every case this intent does not name stays byte-identical to the golden output in `apps/cli/internal/engineconfig/` and `apps/cli/internal/slices/initcmd/`.
- Release: the next release after 0.2.10 (issue #463 label `release:next`).
- TRD 9.4 still describes the old same-version rule; that amendment belongs to the spec's section 5 (neither the 0015 plan nor spec carries it).
- Not in scope: the catalog content comparison (0015 T14), runtime skill contents, and intent 0014 ([#444](https://github.com/testeos-z/gabo-ai/issues/444)).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Do the P2a findings in the [#463 comment](https://github.com/testeos-z/gabo-ai/issues/463#issuecomment-6703389898) that are not init behavior join this intent? They are: (1) Pi waves under `pi --print` end after one turn; (2) gabo-cycle on Pi must register each task worktree with `session_worktree_register`; (3) gentle-pi asks before `git push` unless autonomous mode is on, and `gabo init` would have to write that mode; (5) small closure and PR-body bugs. | open (owner). **Proposed (auto):** no; each gets its own intent. Findings (1), (2) and (5) change runtime skills, docs or gabo-cycle, and (3) turns off a gentle-pi confirmation guard for `git push`, which is a safety decision for the owner. |
| 2 | The user-level reviewer is still replaced only when its own `VERSION` differs (P2a finding 4, TRD 9.4). Task T14 and PR #526 compare the catalog copy only. Where does a content comparison for the reviewer go: a further task of intent 0015, a later intent, or nowhere? | open (owner). Removed from this intent's requirements by the narrowing, so it is carried here rather than lost. |

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
| intent issue | [#489 I0016: gabo init: Pi subagent-package check](https://github.com/testeos-z/gabo-ai/issues/489) (no parent card) |
| related intent | 0015-one-subagent-layer-on-every-engine (T14); 0013-several-engines-in-parallel-in-one-project (probe of PR #460, issue #393) |
