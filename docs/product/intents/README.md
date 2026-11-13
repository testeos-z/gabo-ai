# Intents

One folder per change to gabo-ai. A change starts as an intent, gets one spec, and once a human accepts the spec it gets one plan and its tasks. The tracker is the global view of the work; there is no global plan.

## Layout

```text
docs/product/intents/
  README.md            this file
  <NNNN-slug>/
    intent.md          the problem, the affected users and the desired outcome (gabo-pm)
    spec.md            requirements, acceptance criteria, UX and flow, data changes, and the human acceptance record
    plan.md            ordered steps and the task table (gabo-dev-lead)
    <review findings>  findings recorded while the change is reviewed
```

- `<NNNN>` is the next free four-digit number. `<slug>` is the title in lower case, with every run of non-alphanumeric characters replaced by one `-`. An intent id is never reused or renumbered.
- The documents are written from the templates of the directiva skills: `gabo-pm` (`intent.md`, `spec.md`), `gabo-dev-lead` (`plan.md`), with UX and flow from `gabo-frontend` and migrations from `gabo-db-master` in the spec.
- A tracker product card can hold several intents. Each task belongs to exactly one intent, and its `blocked-by` links may cross intents (written `<NNNN>/T<n>` in a plan).
- One wave runs one intent ([TRD 5.1](../TRD.md#51-wave-run-r3-r4-r5-r10-r13)). The spec is accepted by a human, one intent at a time.

## Checks

`gabo check` validates every folder here offline: the folder name and a unique number, an `intent.md` in each folder, the stage order (no `plan.md` without a `spec.md` whose acceptance table a human filled) and the plan task table (ids, cross-intent references, `depends_on` mirrored by `unblocks`, no dependency cycle). Plain files in this folder, such as this README, are skipped.

## Index

Directiva opens each intent when the owner asks for a change; never create one by hand to fill the folder. The status is the `intent.md` status; acceptance lives in each spec's section 7.

| intent | title | intent issue | spec | target release |
|---|---|---|---|---|
| [0001](0001-claude-code-plugin/intent.md) | Claude Code plugin | [#172](https://github.com/testeos-z/gabo-ai/issues/172) | [spec](0001-claude-code-plugin/spec.md) | — |
| [0002](0002-codex-plugin/intent.md) | Codex plugin | [#173](https://github.com/testeos-z/gabo-ai/issues/173) | [spec](0002-codex-plugin/spec.md) | — |
| [0003](0003-pi-package/intent.md) | Pi package | [#174](https://github.com/testeos-z/gabo-ai/issues/174) | [spec](0003-pi-package/spec.md) | — |
| [0004](0004-opencode-runtime/intent.md) | OpenCode runtime | [#175](https://github.com/testeos-z/gabo-ai/issues/175) | [spec](0004-opencode-runtime/spec.md) | — |
| [0005](0005-any-runtime-engine/intent.md) | Any runtime engine | [#176](https://github.com/testeos-z/gabo-ai/issues/176) | — (no spec yet) | — |
| [0006](0006-engine-integration-at-init/intent.md) | Engine integration at init | [#261](https://github.com/testeos-z/gabo-ai/issues/261) | [spec](0006-engine-integration-at-init/spec.md) | 0.2.6 |
| [0007](0007-update-documentation-for-0-2-6/intent.md) | Update documentation for 0.2.6 | [#263](https://github.com/testeos-z/gabo-ai/issues/263) | [spec](0007-update-documentation-for-0-2-6/spec.md) | 0.2.6 |
| [0008](0008-stricter-gabo-check/intent.md) | Stricter gabo check | [#304](https://github.com/testeos-z/gabo-ai/issues/304) | [spec](0008-stricter-gabo-check/spec.md) (accepted 2026-11-06) | 0.2.8 |
| [0009](0009-main-session-defaults-to-the-gabo-assistant/intent.md) | Main session defaults to the gabo assistant | [#305](https://github.com/testeos-z/gabo-ai/issues/305) | [spec](0009-main-session-defaults-to-the-gabo-assistant/spec.md) (accepted 2026-11-06) | 0.2.8 |
| [0010](0010-gabo-init-writes-the-main-session-key/intent.md) | gabo init writes the main-session key | [#354](https://github.com/testeos-z/gabo-ai/issues/354) | [spec](0010-gabo-init-writes-the-main-session-key/spec.md) (accepted 2026-11-06) | 0.2.10 |
| [0011](0011-docs-for-the-public-repository/intent.md) | Docs for the public repository | [#355](https://github.com/testeos-z/gabo-ai/issues/355) | [spec](0011-docs-for-the-public-repository/spec.md) (accepted 2026-11-06) | 0.2.10 |
| [0012](0012-design-norms-template/intent.md) | Design norms template | [#374](https://github.com/testeos-z/gabo-ai/issues/374) | [spec](0012-design-norms-template/spec.md) (accepted 2026-11-06) | 0.2.10 |
| [0013](0013-several-engines-in-parallel-in-one-project/intent.md) | Several engines in parallel in one project | [#393](https://github.com/testeos-z/gabo-ai/issues/393) | [spec](0013-several-engines-in-parallel-in-one-project/spec.md) (accepted 2026-11-06) | 0.2.10 |
| [0014](0014-agent-name-and-duplicate-skill-checks/intent.md) | Agent name and duplicate skill checks | [#444](https://github.com/testeos-z/gabo-ai/issues/444) | [spec](0014-agent-name-and-duplicate-skill-checks/spec.md) (accepted 2026-11-07) | 0.2.10 |
| [0015](0015-one-subagent-layer-on-every-engine/intent.md) | One subagent layer on every engine | [#474](https://github.com/testeos-z/gabo-ai/issues/474) | [spec](0015-one-subagent-layer-on-every-engine/spec.md) (accepted 2026-11-11) | next (after 0.2.10) |
| [0016](0016-gabo-init-pi-subagent-package-check-and-reviewer-content-check/intent.md) | gabo init: Pi subagent-package check and reviewer content check | [#489](https://github.com/testeos-z/gabo-ai/issues/489) | [spec](0016-gabo-init-pi-subagent-package-check-and-reviewer-content-check/spec.md) (accepted 2026-11-11) | next after 0.2.10 |
| [0017](0017-any-runtime-engine-without-a-go-change/intent.md) | Any runtime engine without a Go change | [#490](https://github.com/testeos-z/gabo-ai/issues/490) | [spec](0017-any-runtime-engine-without-a-go-change/spec.md) (accepted 2026-11-11) | — |
| [0018](0018-config-sync/intent.md) | Config sync | [#493](https://github.com/testeos-z/gabo-ai/issues/493) | [spec](0018-config-sync/spec.md) (accepted 2026-11-11) | — |
| [0019](0019-talk-to-cloud-agent-sessions/intent.md) | Talk to cloud agent sessions | [#494](https://github.com/testeos-z/gabo-ai/issues/494) | [spec](0019-talk-to-cloud-agent-sessions/spec.md) (accepted 2026-11-11) | — |
| [0020](0020-spend-or-turn-cap-for-auto-mode/intent.md) | Spend or turn cap for auto mode | [#495](https://github.com/testeos-z/gabo-ai/issues/495) | [spec](0020-spend-or-turn-cap-for-auto-mode/spec.md) (accepted 2026-11-12) | — |
| [0021](0021-delegate-by-reason-not-by-default/intent.md) | Delegate by reason, not by default | [#496](https://github.com/testeos-z/gabo-ai/issues/496) | [spec](0021-delegate-by-reason-not-by-default/spec.md) (accepted 2026-11-12) | — |
| [0022](0022-measure-roles/intent.md) | Measure roles | [#497](https://github.com/testeos-z/gabo-ai/issues/497) | [spec](0022-measure-roles/spec.md) (accepted 2026-11-12) | — |
| [0023](0023-unattended-pi-waves-and-flow-hygiene/intent.md) | Unattended Pi waves and flow hygiene | [#575](https://github.com/testeos-z/gabo-ai/issues/575) | [spec](0023-unattended-pi-waves-and-flow-hygiene/spec.md) (accepted 2026-11-12) | — |
| [0024](0024-mechanical-wave-steps-and-follow-up-fixes/intent.md) | Mechanical wave steps and follow-up fixes | [#664](https://github.com/testeos-z/gabo-ai/issues/664) | [spec](0024-mechanical-wave-steps-and-follow-up-fixes/spec.md) (accepted 2026-11-12) | — |
| [0025](0025-wave-record-and-reviewer-follow-ups/intent.md) | Wave record and reviewer follow-ups | [#695](https://github.com/testeos-z/gabo-ai/issues/695) | [spec](0025-wave-record-and-reviewer-follow-ups/spec.md) (accepted 2026-11-13) | — |
| [0026](0026-pi-probe-follow-ups/intent.md) | Pi probe follow-ups | [#703](https://github.com/testeos-z/gabo-ai/issues/703) | [spec](0026-pi-probe-follow-ups/spec.md) (accepted 2026-11-13) | — |
| [0027](0027-coders-follow-gentle-ai-odd-and-rdd/intent.md) | Coders and product QA share one definition of done | [#758](https://github.com/testeos-z/gabo-ai/issues/758) | [spec](0027-coders-follow-gentle-ai-odd-and-rdd/spec.md) (scope narrowed 2026-11-13, awaiting the owner's confirmation) | — |
| [0028](0028-deterministic-hooks-as-locks/intent.md) | Deterministic hooks as locks | [#765](https://github.com/testeos-z/gabo-ai/issues/765) | — (no spec yet) | — |
| [0029](0029-branch-protection-on-main-and-dev/intent.md) | Branch protection on main and dev | [#766](https://github.com/testeos-z/gabo-ai/issues/766) | — (no spec yet) | — |
| [0030](0030-agent-evals-when-skills-change/intent.md) | Agent evals when skills change | [#767](https://github.com/testeos-z/gabo-ai/issues/767) | — (no spec yet) | — |
