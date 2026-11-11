# Intent 0017 Any runtime engine without a Go change

| field | value |
|---|---|
| intent id | 0017-any-runtime-engine-without-a-go-change |
| status | Specified |
| owner | gabo-pm |
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| date | 2026-11-11 |

## Problem

`apps/cli/internal/config/config.go` holds a fixed list of engines: `var runtimes = []Runtime{claude, codex, pi, opencode}`, and the error text names the same four. The two copies of the project JSON schema (`apps/cli/internal/config/schemas/gabo.project.schema.json` and `schemas/gabo.project.schema.json`) enumerate the same four names.

An engine that `gabo-runtime-builder` writes as `.gabo/runtimes/gabo-runtime-<engine>/SKILL.md` is therefore refused in `gabo.yaml` until gabo-ai ships a Go change and a release. The builder stops at that point (its step 7: "the installed gabo accepts only the runtimes its message lists"). This contradicts PRD R35 and TRD 2, which say a new engine needs no Go change and no gabo-ai release.

## Who is affected

Owners who run gabo with an engine outside the four shipped ones, for example one that their agent adds through the runtime builder. It happens once per such engine, at setup, and it blocks every later `gabo check` and wave until a gabo-ai release accepts the name.

## Desired outcome

- An engine with a valid runtime skill is accepted in `runtimes.coder` and `runtimes.assistant` without a Go change or a gabo-ai release.
- An engine name with no valid runtime skill still fails `gabo check`, with a next step that names `gabo-runtime-builder`.
- The reviewer roles stay limited to the engines the reviewer scripts run.

## Constraints

- PRD R35 and TRD 2 promise no Go change and no release for a new engine. This intent implements that promise; until it lands, the conflict stays open and R35 and TRD 2 stay as written.
- The reviewer scripts run only the engines they support (TRD D27). `runtimes.reviewer` keeps the four named engines.
- An accepted builder engine is in the flexible compatibility tier (PRD section 6, TRD D28): skills and configuration only, no plugin and no hooks, and `gabo init` writes nothing engine-specific for it.
- `gabo check` stays read-only (PRD R33).
- The two schema copies must stay equal.
- The four named engines stay in the full tier with unchanged output.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does the owner want engines accepted without a Go change? | answered: yes (owner decision 2026-11-11, see Log) |
| 2 | What makes a runtime skill valid enough to accept its engine name (carried from intent 0005, question 1)? | open. Spec D1 proposes: the skill exists, its frontmatter parses and it declares `metadata.agent_dir`. |
| 3 | How does the schema express an open set of engine names while editors still suggest the known four (carried from intent 0005, question 2)? | open. Spec D2 proposes a pattern with the four names as `examples`. |
| 4 | Should `runtimes.reviewer` follow the same rule once the scripts report the engines they support (carried from intent 0005, question 3)? | open. Spec D3 proposes: no, it stays the four engines. |
| 5 | Should `gabo check` report a builder-written engine as unverified until its probe result is recorded (carried from intent 0005, question 4)? | open. |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#490 I0017: Any runtime engine without a Go change](https://github.com/testeos-z/gabo-ai/issues/490) (sub-issue of #171) |
| source issue | [#176 I0005: Any runtime engine](https://github.com/testeos-z/gabo-ai/issues/176) (stays open) |
| PRD | [PRD](../../PRD.md), R35; flexible tier in section 6 |
| TRD | [TRD](../../TRD.md), 2 (supported runtimes), D27 (reviewer engines), D28 (compatibility tiers) |
| spec | [spec](spec.md) |
| plan | — (written by gabo-dev-lead after the owner accepts the spec) |
| related intent | [0005 Any runtime engine](../0005-any-runtime-engine/intent.md) (earlier draft, no spec, not changed here); 0006-engine-integration-at-init (flexible tier, S8 and S11) |

## Log

- 2026-11-11, owner decision: yes, engines must be accepted without a Go change. This closes the question of need. Requirements S1 to S6 of spec 0017 are firm. The spec is still awaiting the owner's acceptance, and section 7 of the spec stays empty.
