# Intent 0006 Engine integration at init

| field | value |
|---|---|
| intent id | 0006-engine-integration-at-init |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-04 |

## Problem

After `gabo init`, the user still has to find out, engine by engine, how to connect gabo to the engines their project uses: add the gabo marketplace and install the gabo plugin in Claude Code or Codex, install the Pi package, check that the OpenCode stage-gate plugin is in place. `gabo init` copies the catalog and prints the setup prompt, but it neither looks at the engines the project declares nor sets up or recommends their integration. The owner, verbatim: "pero gabo debe tener en el init una forma que detecte el motor y ahi te recomiende la integracion" and "pero la idea es que gabo cli configure todo solo ?".

## Who is affected

Owners and teams who set up or upgrade gabo in a project with one or more engines in `runtimes.coder`, `runtimes.reviewer` and `runtimes.assistant` (`claude`, `codex`, `pi`, `opencode`). It happens on every new project and on every `gabo init` after an upgrade, once per engine, and each teammate who clones the project meets it again.

## Desired outcome

- `gabo init` reads the engines the project declares in `gabo.yaml` and, for each one, sets up the gabo integration itself where the engine allows project-level configuration.
- Where the engine does not allow it, or something outside the project is still needed (a package install, a trust prompt), `gabo init` prints the exact command or step.
- `gabo init` ends with one report of what it wrote and what the user must still run.
- The change ships in gabo release 0.2.6 (owner, verbatim, 2026-11-04: "luego me deje un solo PR a main con la version 0.2.6"; first planned as 0.2.4: "levantaremos un nuevo intent y subiremos la version a 0.2.4").
- The engines in scope are the full compatibility group: `claude`, `codex`, `pi` and `opencode` (owner 2026-11-04).

## Constraints

- The engines come only from `gabo.yaml` `runtimes.*`; no detection of what is installed on the machine or on `PATH` (owner scope, 2026-11-04).
- gabo writes only project-level engine configuration, only the keys it needs, merged with the existing content, reported, and never the user's global engine configuration (owner rule 2026-11-03, intent 0004 S8 and D7; runtime skills, #254).
- Every rule lives in Go and is testable in Go; any non-Go layer only calls Go (owner decision 2026-11-03, intent 0003 open question 6).
- `gabo init` never creates or edits `gabo.yaml` or the user config (PRD R32, TRD 9.4, both amended 2026-11-04 to allow the project-level engine keys of this intent), and the CLI keeps its four commands (PRD R31).
- Runtime knowledge lives in runtime skills, not in Go (TRD D24); this intent moves engine integration knowledge into Go, which the spec must reconcile (spec section 5).
- The plugins already exist: the Claude Code marketplace `gabo-ai` with plugin `gabo` (intent 0001), the Codex marketplace `gabo-ai` under `.agents/plugins/` (intent 0002), the Pi package, renamed `@gabo-ai/pi` and published to npm from `main` (intent 0003; owner 2026-11-04, PR #262; `gabo init` installs it through the project's `.pi/settings.json`) and the OpenCode stage-gate plugin built by `gabo-runtime-opencode` (intent 0004).
- Pi without a subagent package stays a refusal with the gentle-pi and Gentle AI recommendation (owner decision 2026-11-03, intent 0003).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does Codex enable a plugin from a project-level config file, or does it need a user-level install? | carried to spec (answered there from the Codex docs) |
| 2 | What does `gabo init` do on a fresh project, where `gabo.yaml` does not exist yet when it runs? | carried to spec |
| 3 | Does `gabo check` verify the keys `gabo init` writes, as the standing engine-config rule says? | carried to spec |
| 4 | Is publishing `@testeos-z/gabo-pi` to npm part of the release (0.2.6; asked as 0.2.4)? | carried to spec (answered there: yes, as `@gabo-ai/pi`) |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#261 I0006: Engine integration at init](https://github.com/testeos-z/gabo-ai/issues/261) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0001-claude-code-plugin, 0002-codex-plugin, 0003-pi-package, 0004-opencode-runtime, 0005-any-runtime-engine |
