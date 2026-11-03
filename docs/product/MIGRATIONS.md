# Migrations

| field | value |
|---|---|
| status | Draft |
| owner | gabo-db-master |
| date | 2026-11-03 |
| storage | config-only |

gabo-ai has no database. Its only persistence is the configuration files, the skill catalog copy with its version stamp, the user-level reviewer skill and the built agent files ([TRD 9.3](TRD.md#93-configuration-shapes)). There is no upfront schema: the config shape is the sum of the steps below, and each new change logs its step here from its intent's `spec.md` section 4.

## Conventions

- **Storage.** YAML files validated with JSON Schema draft 2020-12: the global user file and the project `gabo.yaml` ([TRD 9.3](TRD.md#93-configuration-shapes)).
- **Tool and location.** No migration tool. A config change is a change to the Go loader (`apps/cli/internal/config/`), to both copies of the schemas (`schemas/` at the repository root and the embedded copy refreshed by `go generate ./internal/config`) and to `apps/cli/gabo.example.yaml`. The assistant rewrites an existing config file only with the owner's approval; gabo-ai never rewrites one silently.
- **Versioning.** Each file carries `version`; a missing `version` reads as 0. A breaking change bumps it and ships a new schema file; additive optional keys do not.
- **Forward-only.** Never edit an applied step; fix it with a new row. Previous file versions stay in git history, so a step is reversible by checkout.

## Log

Steps 1 to 7 predate the intent model and were first recorded in the archived [DB_SCHEMA section 6](archive/DB_SCHEMA.md#6-migration); their numbers are unchanged, because code comments cite them (for example "DB_SCHEMA migration step 3").

| order | intent | file | purpose | rollback |
|---|---|---|---|---|
| 1 | — (before intents) | `apps/cli/internal/config/keyclass.go` | Add `version` and the personal and project key classes to the loader, so personal keys can win from the global file without breaking existing files | none: forward-only |
| 2 | — (before intents) | `apps/cli/internal/config/` | Make the cloud keys (`server.*`, `users.*`, `bot.*`, `sessions.*.prompt_file`) optional, so local use works without them | none: forward-only |
| 3 | — (before intents) | `apps/cli/internal/config/projectkeys.go` | Add the project keys `mode`, `slots` (including `slots.methodology` and `slots.agent_host`), `reviewer.{placement, signing}`, `developers` and `plan_change.*`; `generation.targets`, added here, was removed later (TRD D24) | none: forward-only |
| 4 | — (before intents) | `apps/cli/internal/config/` | Move the cloud keys, `sessions.local_orca` and the system layer `/etc/gabo/gabo.yaml` into the `slots.agent_host` adapter, and drop the system layer from the core loader | none: forward-only |
| 5 | — (before intents) | `apps/cli/internal/config/` | Replace silent code defaults with suggestions the assistant asks about (R12); a missing required key fails with `run setup` | none: forward-only |
| 6 | — (before intents) | `schemas/gabo.user.schema.json`, `schemas/gabo.project.schema.json` | Publish the JSON Schemas and write the `$schema` comment in both files for editor validation | none: forward-only |
| 7 | — (before intents) | `apps/cli/internal/config/`, `.gabo/` | Rename (TRD D26): `gbai.yaml` to `gabo.yaml`, the global user directory `gbai` to `gabo`, `.gbai/` to `.gabo/` and the user-level reviewer location to `.../gabo/`; `gabo check` reports a config still under an old name | none: forward-only |
| 8 | — (feature `per-change-sdlc`, commit 1b36a39) | `apps/cli/internal/config/config.go`, both project schemas, `apps/cli/gabo.example.yaml` | Add the optional `wave.after` key (`next` or `wait`, unset means `wait`): one wave runs one intent, and this key decides whether the assistant chains the next intent | remove the key; unset already means `wait` |
| 9 | — (feature `per-change-sdlc`, commit baf47c5) | both project schemas | Redefine the meaning of `plan_change.scope` without changing its values: `task` is one task in one intent's `plan.md`, `group` is one whole intent, `global` is several intents or the PRD and TRD conventions | none: description only, no value changes |

## Open questions

1. Should steps 8 and 9 carry an intent id once the feature that made them is recorded as an intent, or stay attributed to the feature? (gabo-db-master)
