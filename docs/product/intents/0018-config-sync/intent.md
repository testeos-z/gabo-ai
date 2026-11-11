# Intent 0018 Config sync

| field | value |
|---|---|
| intent id | 0018-config-sync |
| status | Draft (spec written when the owner picks this intent) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-11 |

## Problem

Setting up gabo on a new repository takes too long. `gabo-setup` asks every work-style question again in each repository (mode, `plan_change`, runtimes, where agents run, slot adapters), although the owner works the same way everywhere. Only the personal keys (`alias`, `tone`, `interaction_level`, `language`) live in the global user file. Every other key is a project key in each repository's `gabo.yaml` (TRD 9.3), so nothing carries the owner's way of working from one repository to the next.

Owner, verbatim (2026-11-11): "una forma mas rapida de configurar tardamos mucho en la configuracion inicial que podamos decirle que revise otro repo y clone la configuracion apnostica al opjetivo del repo solo de como me gusta trabajar"

Owner, verbatim (2026-11-11): "me gustan los dos cosas que gabo pueda hacer poll-config o 'sync config' y que por defecto tome local pero que te pregunta si local o remoto"

## Who is affected

The owner, and anyone who sets up gabo on a new repository. It happens once per repository, at setup, and each setup repeats the same work-style answers.

## Desired outcome

- A new repository gets the owner's way of working without answering every setup question again.
- The owner is asked where to take it from, local or remote, and local is the default (owner's words).
- Only work-style settings are carried over; settings tied to the repository's own goal are not copied blindly (owner: "agnostica al objetivo del repo").

## Constraints

- No secrets are copied, asked for or written; credentials are named by env var only (PRD R21).
- Personal keys stay in the global user file and project keys in `gabo.yaml` (TRD 9.3, D11, D22).
- `gabo init` never touches `gabo.yaml` or the user config (PRD R32).
- After a sync, the agents are still built and `gabo check` still passes before work starts (PRD R33).

## Assistant's analysis (input for the spec, not confirmed by the owner)

- One "sync config" operation. The local source is a user-level work-style profile (the default); the remote source is another repository's `gabo.yaml`.
- It copies only work-style keys: `mode`, `runtimes`, slot adapter choices, `parallelism`, `wave.after`, `plan_change`, `review.auto_merge`, `reviewer.placement` and generic `human_only_paths`. It detects or asks the repository-specific keys: `repo.name`, `review.checks`, `developers`, `worktree.keep` and adapter ids.
- The agent build, `gabo check` and the slot smoke test of `gabo-setup` stay after the sync.
- Recommendation: the same operation can save a configured repository's style as the local profile.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does "remoto" mean another repository or another machine? | open (owner) |
| 2 | Which keys count as work style, and which are repository-specific? | carried to spec (assistant's list above is a proposal) |
| 3 | Should the same operation also save a configured repository's style as the local profile? | open (owner); recommended by the assistant |
| 4 | Where does the local profile live: in the global user file, or in a file of its own? | carried to spec |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#493 I0018: Config sync](https://github.com/testeos-z/gabo-ai/issues/493) (sub-issue of #492) |
| PRD | [PRD](../../PRD.md), R21, R32, R33 |
| TRD | [TRD](../../TRD.md), 9.3 (config layers), D11, D22 |
| spec | — (written when the owner picks this intent) |
| plan | — |
| related intent | — |
