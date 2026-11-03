# Intent 0005 Any runtime engine

| field | value |
|---|---|
| intent id | 0005-any-runtime-engine |
| status | Draft |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-03 |

## Problem

`gabo.yaml` accepts only a fixed list of engines for `runtimes.coder`, `runtimes.reviewer` and `runtimes.assistant` (today `claude` and `pi`; intents 0002 and 0004 add `codex` and `opencode` to coder and assistant). An engine that the user's agent adds through `gabo-runtime-builder` fails `gabo check` until gabo-ai ships a Go and schema change and a release. This contradicts the promise that a new engine needs no gabo-ai release (PRD R35) and that `gabo-runtime-builder` adds an engine "without a Go change or a release" (TRD 2, Supported runtimes).

## Who is affected

Owners and teams who run gabo on an engine that gabo-ai does not ship a runtime skill for. It happens once per new engine, at setup, and it blocks every later `gabo check` and wave in that project until a gabo-ai release accepts the engine name.

## Desired outcome

- An engine that has a valid runtime skill (shipped in the catalog or written by `gabo-runtime-builder`) is accepted in `runtimes.*` without a Go change or a gabo-ai release.
- `gabo check` still rejects an engine name that has no valid runtime skill, with a next step that names `gabo-runtime-builder`.

## Constraints

- The reviewer scripts run only the engines they support (`REVIEWER_ENGINE` `claude` or `pi` today, `gabo-reviewer/scripts/common.sh`; TRD D27). `runtimes.reviewer` must not accept an engine the scripts cannot run (owner decision 2026-11-03).
- `gabo check` stays read-only and keeps validating `gabo.yaml`, the agent directories and the links it reads from each runtime skill (TRD 9.4; PRD R31).
- Builder-written runtime skills live in `.gabo/runtimes/<name>/`, outside the catalog copy (TRD 9.4).
- The JSON Schema exists in two copies (`apps/cli/internal/config/schemas/gabo.project.schema.json`, `schemas/gabo.project.schema.json`) that must stay equal.
- Intents 0002 and 0004 add `codex` and `opencode` to `runtimes.coder` and `runtimes.assistant` in one enum edit (owner decision 2026-11-03); this intent must not undo that edit.

## Open questions

| # | question | status |
|---|---|---|
| 1 | What makes a runtime skill "valid" enough to accept its engine name (for example: present in `.gabo/skills/` or `.gabo/runtimes/`, declares `metadata.agent_dir`, has passed its probe)? | open |
| 2 | How does the JSON Schema express an open set of engine names while editors still suggest the known ones? | open |
| 3 | Does `runtimes.reviewer` stay a fixed list tied to the reviewer scripts, or follow the same rule once the scripts report which engines they support? | open |
| 4 | Should `gabo check` validate a builder-written engine the same way as a shipped one, or report it as unverified until its probe result is recorded? | open |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#176 I0005: Any runtime engine](https://github.com/testeos-z/gabo-ai/issues/176) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0002-codex-plugin, 0004-opencode-runtime |
