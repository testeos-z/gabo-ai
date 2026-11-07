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
