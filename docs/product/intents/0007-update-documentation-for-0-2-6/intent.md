# Intent 0007 Update documentation for 0.2.6

| field | value |
|---|---|
| intent id | 0007-update-documentation-for-0-2-6 |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-04 |

## Problem

The documentation does not match what gabo 0.2.6 will ship. The README's Compatibility section (PR #232) states the status of 0.2.2; `docs/cloud-agent-flow.md` still says the runtime per role is "`claude` or `pi`"; the setup page's "Your engine" table has no OpenCode row; the root README, `plugins/claude-code/README.md`, `plugins/codex/README.md` and `packages/pi/README.md` install the Pi package as `@testeos-z/gabo-pi`, which was never published, instead of `@gabo-ai/pi` from npm; and none of them says that `gabo init` now configures the declared engines itself (intent 0006). The owner, verbatim (2026-11-04): "ademas que sume un 4 intent que actualice la documentacion".

## Who is affected

Every new user who installs gabo from the README or the setup page, and every owner upgrading to 0.2.6, with any of the four engines `claude`, `codex`, `pi` and `opencode`. It happens on each install and each upgrade until the docs are aligned.

## Desired outcome

- The README, `docs/setup-prompt.md`, `docs/cloud-agent-flow.md`, `plugins/*/README.md` and `packages/pi/README.md` describe gabo 0.2.6 for the four engines `claude`, `codex`, `pi` and `opencode`.
- The Pi package is installed from npm as `@gabo-ai/pi`.
- The docs say what `gabo init` writes for each declared engine (intent 0006) and what the user must still run.
- The change ships in gabo release 0.2.6, with intent 0006 (owner, verbatim: "luego me deje un solo PR a main con la version 0.2.6").

## Constraints

- Documentation only: no behavior change; every documented behavior is the merged behavior of intent 0006 or earlier, so each doc task waits for the task it documents.
- The marked prompt block of `docs/setup-prompt.md` stays byte-equal to the embedded setup prompt (TRD 9.4, D25); only intent 0006 S15 changes it.
- English, written for a reader new to gabo (`gabo-doc-guide`).
- `plugins/` is a `review.human_only_paths` entry in `gabo.yaml`, so the plugin README changes need a human-only merge, which the owner delegated to the coordinator on 2026-11-04 ([owner authorization, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/261#issuecomment-6633243675)).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Is the README Compatibility table kept as a dated status of one release, now 0.2.6? | carried to spec |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#263 I0007: Update documentation for 0.2.6](https://github.com/testeos-z/gabo-ai/issues/263) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0006-engine-integration-at-init, 0003-pi-package, PR #232 (README Compatibility), PR #262 (`@gabo-ai/pi` on npm) |
