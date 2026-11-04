# Spec 0007 Update documentation for 0.2.6

| field | value |
|---|---|
| intent | [0007-update-documentation-for-0-2-6](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — |
| date | 2026-11-04 |

Evidence base (checked 2026-11-04 on `dev` after PR #232 merged): `README.md` (Compatibility "Status for gabo 0.2.2", Pi row and Pi package section naming `@testeos-z/gabo-pi`), `docs/setup-prompt.md` ("Your engine" table without OpenCode), `docs/cloud-agent-flow.md` lines 37 and 56 ("`claude` or `pi`") and its "what the CLI must automate" table, `plugins/claude-code/README.md`, `plugins/codex/README.md`, `packages/pi/README.md`; PR #262 (rename to `@gabo-ai/pi`, publish from `pi/pkg-up`, open on 2026-11-04); intent 0006 spec as accepted on 2026-11-04. Amended on 2026-11-04 by owner decision (section 7): S1 adds the two compatibility tiers to the README Compatibility section (PRD section 6, TRD D28) and S8 documents that gabo-ai dogfoods `gabo init` (intent 0006 T12, #282).

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The root `README.md` shall describe gabo 0.2.6 for the four engines `claude`, `codex`, `pi` and `opencode`: the Compatibility section states the status of 0.2.6 (tag `apps/cli/v0.2.6`) with one row per engine, the Pi row names the npm package `@gabo-ai/pi`, and the install sections (Claude Code plugin, Codex plugin, Pi package) say that `gabo init` writes the project-level integration for each engine `gabo.yaml` declares (`.claude/settings.json`, `.codex/config.toml`, `.pi/settings.json`; nothing for OpenCode), never the user's global engine config, and ends with a report of what the user must still run; the manual install commands stay as the alternative, and the Codex fallback commands are shown while intent 0006 D4 holds. Quick start and Upgrading say to run `gabo init` again after the agent writes `gabo.yaml` and after each upgrade. The Compatibility section shows both compatibility tiers (amended 2026-11-04, owner decision, PRD section 6, TRD D28): **full**, the four engine rows, each with its plugin or package, hooks, the project-level integration `gabo init` sets up and the engine files `gabo check` reads; **flexible**, any other engine with subagents and skills, which gets only the gabo skills and configuration (no plugin, no hooks), for which `gabo init` writes nothing engine-specific and prints one flexible-tier line, and for which `gabo check` stays engine-neutral; the flexible row states whether `gabo.yaml` accepts such a name in 0.2.6 (today it does not; it depends on intent 0005, #176). | Given the README Then "Status for gabo 0.2.6" replaces 0.2.2, the table has the four engine rows, the Pi install line reads `pi install npm:@gabo-ai/pi@<version>` and the text names the three files `gabo init` writes; Given the Compatibility section Then it names the full tier with the four engines and the flexible tier as skills and configuration only, no plugin and no hooks, and says whether a flexible engine name is accepted by `gabo.yaml` in 0.2.6, checked against `apps/cli/internal/config` on `dev`; Given `grep -n "testeos-z/gabo-pi\|0\.2\.2" README.md` Then no match outside a dated history note. | doc review; grep check |
| S2 | `docs/setup-prompt.md` shall list the four engines in "Your engine" (adding OpenCode: `gabo-runtime-opencode`, `.opencode/agents/`, no prerequisite beyond OpenCode itself) and say, outside the marked prompt block, that `gabo init` configures the declared engines and is run again once `gabo.yaml` exists (intent 0006 S10, S15). The marked prompt block is changed only by intent 0006 S15. | Given the page Then the table has rows for Claude Code, Codex, Pi and OpenCode and the page names the second `gabo init`; Given `go -C apps/cli test ./internal/setupprompt/...` Then the byte-equality test passes. | doc review; Go test |
| S3 | `docs/cloud-agent-flow.md` shall name the four engines wherever it lists the runtime per role or the review engine (`claude`, `codex`, `pi` or `opencode`), and its table of what the CLI automates shall add the project-level engine integration that `gabo init` writes (intent 0006). | Given `grep -n "\`claude\` or \`pi\`" docs/cloud-agent-flow.md` Then no match; Given the `gabo init` row Then it names the engine integration keys and that user-level engine config is never written. | doc review; grep check |
| S4 | `plugins/claude-code/README.md` and `plugins/codex/README.md` shall say, in their Install sections, that `gabo init` writes the plugin's project keys when `gabo.yaml` declares the engine (Claude Code: `extraKnownMarketplaces` and `enabledPlugins` in `.claude/settings.json`; Codex: `[marketplaces.gabo-ai]` and `[plugins."gabo@gabo-ai"]` in `.codex/config.toml`), what the user still does (trust the folder; Codex: trust the hooks in `/hooks`), and keep the manual commands as the alternative; their release checklists name 0.2.6 where they name a version and `@gabo-ai/pi` where they name the Pi package. | Given each plugin README Then its Install section holds the `gabo init` paragraph and the manual commands; Given `grep -rn "testeos-z/gabo-pi" plugins/` Then no match. | doc review; grep check |
| S5 | `packages/pi/README.md` shall install the package from npm as `@gabo-ai/pi` (`pi install npm:@gabo-ai/pi@<version>`, or the project `.pi/settings.json` `packages` entry that `gabo init` writes when `gabo.yaml` declares `pi`, loaded after the project is trusted), keep the checkout install as a development path, and state that its version follows the gabo release (0.2.6). | Given the README Then the first install command is the npm one with `@gabo-ai/pi`, the `gabo init` paragraph names `.pi/settings.json`, and `grep -n "testeos-z/gabo-pi" packages/pi/README.md` has no match. | doc review; grep check |
| S6 | After the change, the documents of S1 to S5 shall agree with each other and with the code of release 0.2.6: the four engines, the package name `@gabo-ai/pi`, the files `gabo init` writes, and the version 0.2.6. | Given `grep -rn "testeos-z/gabo-pi\|\`claude\` or \`pi\`" README.md docs/setup-prompt.md docs/cloud-agent-flow.md plugins/*/README.md packages/pi/README.md` Then no match; Given `gabo check` and `go -C apps/cli test ./...` Then both pass. | grep check; `gabo check`; Go tests |
| S7 | The change shall alter documentation only: no Go, script, manifest or skill file. | Given the diff of each intent 0007 PR Then it touches only the files named in S1 to S5 and S8. | review of the diff |
| S8 | The root `README.md` shall say that gabo-ai dogfoods `gabo init`: the repository runs the released or `dev` build of `gabo init` itself and commits the project-level engine files it writes for its own `gabo.yaml` `runtimes` (today `.claude/settings.json` with `extraKnownMarketplaces."gabo-ai"` and `enabledPlugins."gabo@gabo-ai"`), so its own `gabo check` verifies them (intent 0006 T12, #282). (Added 2026-11-04, owner decision, section 7.) | Given the README Then one paragraph names the dogfood, `.claude/settings.json` and the two keys, and matches the file committed by intent 0006 T12 on `dev`; Given `gabo check` at the repository root Then it reports no engine-key failure. | doc review; `gabo check` |

## 2. Design

Five documents, one task each, each written after the intent 0006 task whose behavior it describes is merged, then one consistency sweep. The README Compatibility section keeps the shape PR #232 gave it (one dated status per release, engine rows, verified versions) and moves its status line to 0.2.6.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Compatibility section shape | A: keep a dated status of one release, updated each release / B: a per-release history table | A, **Proposed (auto)**, accepted (owner 2026-11-04): PR #232's shape, smallest change, one place to read. | Each release updates the status line; old status lives in git history. |
| D2 | When each doc task runs | A: before intent 0006 code, from the spec / B: after the intent 0006 task it documents is merged | B, **Proposed (auto)**, accepted (owner 2026-11-04): documents describe merged behavior, never planned behavior. | The intent 0007 wave runs after the intent 0006 wave. |
| D3 | Pi package name in docs | A: `@testeos-z/gabo-pi` / B: `@gabo-ai/pi` | B (owner 2026-11-04, npm org `gabo-ai`, PR #262). | The docs are correct once PR #262 is merged and 0.2.6 is published. |

## 3. UX and app flow (optional)

Not applicable: documentation pages only; no screen or designed journey.

## 4. DB migration steps (optional)

Not applicable: no stored data changes.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| `review.human_only_paths` (`plugins/`) | S4 edits `plugins/claude-code/README.md` and `plugins/codex/README.md`. | The coder opens the PR; the coordinator merges it ([owner authorization, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/261#issuecomment-6633243675): merge PRs into `dev` including human-only paths). |
| TRD 9.4, D25 (setup prompt single source) | none: S2 edits only the prose outside the marked block. | — |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent Q1: is the Compatibility table a dated status of one release? | answered: yes (D1). |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner) | 2026-11-04 | Accepted by the owner in session, with every **Proposed (auto)** answer: "apruebo el spec 3 sobre lo que falta. dile al agente en la nube que lo haga, que complete los 3 intent y el solo se apruebe los PR a dev y luego me deje un solo PR a main con la version 0.2.6. ademas que sume un 4 intent que actualice la documentacion" |
| leomerida15 (owner) | 2026-11-04 | Intent 0006 T12 (#282, gabo-ai runs `gabo init` and commits `.claude/settings.json`) approved and to be documented here (S8), owner ([owner record, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/263#issuecomment-6633521254)): "1. si esa tarea nueva tambien al de documentar." Recorded by gabo-pm on the owner's instruction "Record the owner quote where acceptance is recorded." in the linked [owner record, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/263#issuecomment-6633521254). |
| leomerida15 (owner) | 2026-11-04 | Compatibility tiers in the README (S1), owner ([owner record, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/263#issuecomment-6633521254)): "2. si cambie el PRD ahora buscaremos dos tipos de compatibilidad - full con los motores soportados y flexible con cualquier otro que solo seran skills y configuraciones" Recorded by gabo-pm on the owner's instruction "Record the owner quote where acceptance is recorded." in the linked [owner record, 2026-11-04](https://github.com/testeos-z/gabo-ai/issues/263#issuecomment-6633521254). |
