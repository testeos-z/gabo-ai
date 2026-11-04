# gabo-ai

A CLI and harness that sets up a team of autonomous coding agents in your project: an assistant you talk to, planning agents, a wave runner, coders and an independent reviewer. The team uses your tracker, your methodology and your infrastructure.

## Why

When you build with AI agents, you often end up as the glue between them. You plan with one agent, launch coders with another, review results and answer every question. When you are tired or away, work stops.

gabo-ai gives that coordination to the agents. You keep what only you can do: validate, plan and decide. Work moves forward wave after wave. Each question goes to whoever can answer it, and in auto mode every decision is recorded with its reason. See the [PRD](docs/product/PRD.md) for the full problem statement and goals.

## Core concepts

| concept | what it means |
|---|---|
| **Agents** | The **assistant** is your single point of contact. **Directiva** plans: the PM (PRD, each intent and its spec), the dev lead (TRD, each intent's plan and its task issues with dependencies), the DB master (data changes and the migrations log) and the frontend agent (UX and flow in the spec). **gabo-cycle** runs waves. **Coders** implement one task each in their own worktree. The **reviewer** is an independent process and the merge gate. |
| **Slots and adapters** | Each configurable part of the system is a slot: `tasks`, `repository_host`, `memory`, `methodology`, `reviewer_placement`, `agent_host`, and optionally `notifications` and `direct_sessions`. gabo-ai ships a **contract** for each slot (what the agents must be able to do). An **adapter** is a Markdown skill or document that explains how to do it with your tool. Tool-specific behavior is never compiled code. |
| **Runtimes** | The engine each agent role runs in (`runtimes.coder`, `runtimes.reviewer`, `runtimes.assistant`). gabo accepts `claude` (Claude Code), `codex` (Codex), `pi` (Pi) and `opencode` (OpenCode), each with a shipped runtime skill; see [Compatibility](#compatibility). |
| **Intents** | Each change is an intent in [`docs/product/intents/<NNNN-slug>/`](docs/product/intents/README.md) (the per-change model): `intent.md` states the problem, `spec.md` the requirements and acceptance criteria, and `plan.md` the tasks. A human accepts each spec before its plan is written. There is no global plan written upfront; the tracker is the global view. |
| **Waves** | A wave is one intent: its unblocked tasks. The assistant starts it, gabo-cycle launches coders, each coder opens a PR, the reviewer posts a verdict, and the cycle reports back to the assistant. When the wave closes, `wave.after` decides: `next` starts the next accepted intent, `wait` (the default) stops until you ask. |
| **Reviewer** | Runs the checks in `review.checks` on a credential-free copy of the merged tree and posts a `VERDICT` line on the PR. It never reads or writes the shared memory. Paths in `review.human_only_paths`, the catalog and the version stamp always need a human merge. |
| **Plan change** | Mid-project changes to scope or the plan go through a plan-change proposal in its own issue and PR, touching only `docs/product` and tasks. You debate it if you are present. If you are absent, a clean-context `gabo-plan-reviewer` debates it, within the limits of `plan_change.*`: the scope is one task of an intent's plan, one whole intent, or several intents and the PRD/TRD conventions. |
| **Integration branch** | Agents merge only into `repo.integration_branch` (for example `dev`), never into `main`. Merging to `main` is always your decision. |

Read more in [docs/architecture.md](docs/architecture.md), the [TRD](docs/product/TRD.md) and the [app flow](docs/product/APP_FLOW.md).

## How it works

![gabo-ai team flow: owner, memory, assistant, development wave, repository, directiva and team tasks](docs/product/diagrams/team-flow.png)

*Team flow and configurable points, exported from the [archify diagram](docs/product/diagrams/team-flow.html) (labels in Spanish).*

You set gabo-ai up once with `gabo init`, the setup prompt and `gabo check`; after that you only talk to the assistant. For each change, directiva writes an intent and its spec; once you accept the spec, it writes the plan and the task issues with their dependencies. Each wave runs one intent: `gabo-cycle` takes that intent's unblocked tasks and launches one coder per task in its own worktree; every PR goes through the independent reviewer, the merge gate, which merges only into the integration branch, never into `main`. The reviewer never reads or writes the shared memory.

For details, see the diagrams [team-flow.html](docs/product/diagrams/team-flow.html) and [configuration-flow.html](docs/product/diagrams/configuration-flow.html), and the TRD sections [4.2 Building blocks](docs/product/TRD.md#42-building-blocks-c4-container-view), [5.1 Wave run](docs/product/TRD.md#51-wave-run-r3-r4-r5-r10-r13) and [5.2 Review](docs/product/TRD.md#52-review-r9).

## Requirements

- Go, at the version named in [`apps/cli/go.mod`](apps/cli/go.mod), to install the CLI.
- git, and a git repository for your project.
- A coding agent engine: Claude Code, Codex, Pi or OpenCode (see [Compatibility](#compatibility)). Pi has no subagents, so gabo refuses to build on Pi without a subagent package; [gentle-pi](https://www.npmjs.com/package/gentle-pi) is recommended for subagents, and the full [Gentle AI framework](https://github.com/Gentleman-Programming/gentle-ai) (methodology, RDD and Engram memory) is the recommended setup. See [Your engine](docs/setup-prompt.md#your-engine).
- Whatever tools your adapters use. For example, the GitHub adapters use the `gh` CLI.

## Compatibility

Status for gabo 0.2.6 (tag `apps/cli/v0.2.6`), checked against the code. **Yes** means it ships in 0.2.6; **planned** means it is designed or tracked but does not work yet.

gabo has two compatibility tiers:

- **Full**: Claude Code, Codex, Pi and OpenCode. Each has a runtime skill, a plugin or package with hooks, the project-level integration that `gabo init` sets up, and, for Claude Code, Codex and Pi, the engine files that `gabo check` reads.
- **Flexible**: any other engine with subagents and skills. It gets only the gabo skills and configuration: no plugin and no hooks. `gabo init` writes nothing engine-specific for it and prints one line, `<engine>: flexible tier: skills and configuration only; nothing written`, and `gabo check` stays engine-neutral. In 0.2.6, `gabo.yaml` does not accept such an engine name yet (see **Accepted engine names** below).

| engine | tier | coder | assistant | reviewer | runtime skill | plugin or package | subagents | post-write hook (`gabo check`) | pre-write hook (`gabo check --pre-write`) | `gabo init` writes | `gabo check` reads |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Claude Code | full | yes | yes | yes | `gabo-runtime-claude` | yes: plugin `gabo@gabo-ai` ([Claude Code plugin](#claude-code-plugin)) | built in | yes, in the plugin | yes, in the plugin | `.claude/settings.json`: `extraKnownMarketplaces."gabo-ai"`, `enabledPlugins."gabo@gabo-ai"` | the same keys |
| Codex | full | yes | yes | yes, after the one-time isolation test | `gabo-runtime-codex` | yes: plugin `gabo@gabo-ai` ([Codex plugin](#codex-plugin)) | built in | yes, in the plugin | yes, in the plugin | `.codex/config.toml`: `[marketplaces.gabo-ai]`, `[plugins."gabo@gabo-ai"]` | the same keys |
| Pi | full | yes | yes | yes | `gabo-runtime-pi` | yes: package `@gabo-ai/pi`, `pi install npm:@gabo-ai/pi@<version>` ([Pi package](#pi-package)) | **required**: a subagent package such as gentle-pi; gabo refuses without one | no | yes, in the package extension | `.pi/settings.json`: the `packages` entry `npm:@gabo-ai/pi@<version>` (release builds only) | that entry, any version |
| OpenCode | full | yes | yes | yes, after the one-time isolation test | `gabo-runtime-opencode` | no plugin to install: the runtime skill copies the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js` into the project | built in; `gabo check` requires `experimental.subagent_depth` of 3 or more | no | yes, in the stage-gate plugin | nothing (the runtime skill installs the stage-gate plugin) | no engine key |
| any other engine | flexible | planned | planned | planned | written by `gabo-runtime-builder` | none | required (the builder refuses an engine without subagents and skills) | none | none | nothing; one flexible-tier line | nothing engine-specific |

- **Accepted engine names.** `gabo.yaml` and `gabo check` accept `claude`, `codex`, `pi` and `opencode` in `runtimes.coder`, `runtimes.assistant` and `runtimes.reviewer`. Any other value fails `gabo check`, so a flexible-tier engine, including a runtime skill that `gabo-runtime-builder` writes for it, cannot be used until [I0005 (#176)](https://github.com/testeos-z/gabo-ai/issues/176) opens `runtimes.*`.
- **Engine integration by `gabo init`.** For each engine that `runtimes.*` in `gabo.yaml` names, `gabo init` writes the project-level keys of the table above, never your global engine config. It only adds missing keys and keeps every value you set. It ends with an `engine integration:` report that names each file it wrote and what you still need to run, such as trusting the project folder in the engine. Without a `gabo.yaml` it writes no engine file and asks you to run `gabo init` again once your agent has written it.
- **gabo-ai dogfoods `gabo init`.** This repository runs `gabo init` on itself and commits the engine file it writes for its own `gabo.yaml` `runtimes` (all `claude`): [`.claude/settings.json`](.claude/settings.json) with `extraKnownMarketplaces."gabo-ai"` (the GitHub source `testeos-z/gabo-ai`) and `enabledPlugins."gabo@gabo-ai": true`, so its own `gabo check` verifies those keys.
- **Reviewer configuration.** The catalog skill `gabo-reviewer-config` configures any of the four engines as reviewer. For `codex` and `opencode` it runs a one-time isolation test first; the reviewer runs those engines only while its pass marker matches the installed engine version.
- **Pi needs a subagent package.** Pi core has no subagents. `gabo-runtime-pi` builds nothing until a subagent package is present. It recommends [gentle-pi](https://www.npmjs.com/package/gentle-pi) and, as the full setup, the [Gentle AI framework](https://github.com/Gentleman-Programming/gentle-ai) installed with `gentle-ai install --agent pi`. The gabo Pi package is not a subagent package. See [Your engine](docs/setup-prompt.md#your-engine).
- **Tracking.** All engine work is tracked under the product card [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171).

Engine versions the runtime skills were checked against (each skill rechecks its sources when the installed version differs):

| engine | version | checked on | where it is recorded |
|---|---|---|---|
| Claude Code | 2.1.302 | 2026-10-28 | `gabo-runtime-claude` skill, Sources |
| Codex (`codex-cli`) | 0.163.0 | 2026-11-03 | `gabo-runtime-codex` skill, Sources |
| Pi | 1.4.0 | 2026-11-03 | `gabo-runtime-pi` skill, Sources |
| gentle-pi | 4.5.1 | 2026-11-03 | `gabo-runtime-pi` skill, Sources |
| OpenCode | 2.0.32 | 2026-11-03 | `gabo-runtime-opencode` skill, Sources |

## Install

### Go install

Install the newest release tag. Go needs your own git credentials while the repository is private:

```sh
GOPRIVATE=github.com/testeos-z/* go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest
```

To pin a release, use `@vX.Y.Z` instead of `@latest`. `gabo version` prints the installed release, for example `gabo v0.2.6`. Then, in your project root:

```sh
gabo version
gabo init
gabo check
```

See [Quick start](#quick-start) for the setup between `gabo init` and `gabo check`.

> **Install from a tag.** `gabo init` accepts only a release version (`x.y.z`) and refuses a pseudo-version, such as the one Go installs from a branch or a commit. Because the module lives in `apps/cli`, release tags are named `apps/cli/vX.Y.Z`.

### Build from source

Clone the repository and name the version yourself. If your project already has a `.gabo/VERSION`, use that version:

```sh
git clone https://github.com/testeos-z/gabo-ai.git
cd gabo-ai
go -C apps/cli install -ldflags "-X main.buildVersion=<x.y.z>" ./cmd/gabo
```

Check the install with `gabo version`.

### Claude Code plugin

The repository is its own Claude Code marketplace, `gabo-ai`, with one plugin, `gabo`. The plugin adds the gabo catalog skills (except the skills that run reviews) and two hooks that run `gabo check` around Claude's writes in a gabo project.

**With `gabo init`.** When `gabo.yaml` names `claude` in `runtimes.*`, `gabo init` adds `extraKnownMarketplaces."gabo-ai"` and `enabledPlugins."gabo@gabo-ai"` to the project's `.claude/settings.json`, never to your global Claude Code settings. Its `engine integration:` report then asks you to trust the project folder in Claude Code, so the marketplace and the plugin apply, and, while `testeos-z/gabo-ai` is private, to give git credentials that can read it. Commit `.claude/settings.json`. As the alternative, install the plugin at user level by hand:

```sh
claude plugin marketplace add testeos-z/gabo-ai
claude plugin install gabo@gabo-ai
```

- **Private repository.** While `testeos-z/gabo-ai` is private, Claude Code clones it with your own git credentials. Set them up first: run `gh auth setup-git`, or add an SSH key that can read the repository.
- **The `checks` option.** On by default. Before Claude writes or edits a gabo file (`gabo.yaml`, `.gabo/`, `.claude/agents/`, `.claude/skills/`, `docs/product/`), `gabo check --pre-write` blocks a write that breaks the stage gate; after it, `gabo check` runs and reports failures to Claude. Both hooks do nothing outside a gabo project (no `.gabo/VERSION`) or when `gabo` is not on `PATH`. Turn them off with the plugin's `checks` row in `/config`.
- **`gabo init` stays required.** The plugin does not configure a project and ships no agents; `gabo init` writes only the two plugin keys above. Per-project config (`gabo.yaml`), slots, adapters and agents still come from `gabo init` and the setup prompt (see [Quick start](#quick-start)). Inside a project, the copy in `.gabo/skills/` stays the source of truth: the agents preload it and `gabo check` checks it.
- **Skill names.** Plugin skills appear namespaced as `gabo:<name>`, for example `gabo:gabo-pm`, so they never clash with the project copy. To use only the project copy in one project, disable the plugin there with `"enabledPlugins": { "gabo@gabo-ai": false }` in that project's `.claude/settings.json`.

See [`plugins/claude-code/README.md`](plugins/claude-code/README.md) for the plugin layout, the open owner decisions and the release checklist.

### Codex plugin

The repository is also a Codex marketplace, `gabo-ai` ([`.agents/plugins/marketplace.json`](.agents/plugins/marketplace.json)), with one plugin, `gabo`. It ships the same skills as the Claude Code plugin and two hooks that run `gabo check` around Codex's `apply_patch` edits in a gabo project. These steps are for the Codex CLI.

**With `gabo init`.** When `gabo.yaml` names `codex` in `runtimes.*`, `gabo init` adds `[marketplaces.gabo-ai]` and `[plugins."gabo@gabo-ai"]` to the project's `.codex/config.toml`, never to your global Codex config. Its `engine integration:` report then asks you to trust the project in Codex, so it loads `.codex/config.toml`, to trust both gabo hooks in `/hooks` and, while `testeos-z/gabo-ai` is private, to set up git credentials that can read it. While the fallback is printed, the report also says: if Codex does not list the gabo plugin after trust, run `codex plugin marketplace add testeos-z/gabo-ai` and `codex plugin add gabo@gabo-ai`. Commit `.codex/config.toml`. As the alternative, install the plugin by hand:

```sh
codex plugin marketplace add testeos-z/gabo-ai
codex plugin add gabo@gabo-ai
```

- **Private repository.** While `testeos-z/gabo-ai` is private, Codex clones it with your own git credentials. Set them up first: run `gh auth setup-git`, or add an SSH key that can read the repository.
- **Trust the hooks.** Codex runs a plugin hook only after you trust it. In a Codex session, open `/hooks` and trust both gabo hooks: the `PreToolUse` hook denies a patch to a gabo file that breaks the stage gate (`gabo check --pre-write`), and the `PostToolUse` hook runs `gabo check` after the patch and returns its failures to Codex. A hook that changes in an update stays off until you trust it again. Both hooks do nothing outside a gabo project (no `.gabo/VERSION`) or when `gabo` is not on `PATH`. To turn them off, leave them untrusted.
- **Update.** Run `codex plugin marketplace upgrade gabo-ai`. Codex caches the plugin per version, so you get a new plugin when a CLI release bumps its version; then check `/hooks` again.
- **`gabo init` stays required.** The plugin does not configure a project and builds no agents; `gabo init` writes only the two plugin keys above. Run `gabo init` and the setup prompt (see [Quick start](#quick-start)) with `runtimes.coder: codex`: the `gabo-runtime-codex` skill builds the agents as `.codex/agents/<id>.toml` and links the skills under `.agents/skills/`.

See [`plugins/codex/README.md`](plugins/codex/README.md) for the plugin layout, how its skill and script copies stay equal to their sources, and the release checklist.

### Pi package

The Pi package `@gabo-ai/pi` ([`packages/pi/`](packages/pi/)) ships the `gabo-pi` and `gabo-reviewer-config` skills and an extension that runs `gabo check --pre-write` at session start and before Pi's `write` and `edit` calls in a gabo project.

**With `gabo init`.** When `gabo.yaml` names `pi` in `runtimes.*`, a release build of `gabo init` adds the `packages` entry `npm:@gabo-ai/pi@<version>`, pinned to its own version, to the project's `.pi/settings.json`, never to your global Pi settings; a development build writes nothing and tells you to add the entry by hand. Its `engine integration:` report then asks you to trust the project in Pi, so it installs and loads `@gabo-ai/pi` from `.pi/settings.json`, and to install a subagent package. Commit `.pi/settings.json`. As the alternative, install the package by hand:

```sh
pi install npm:@gabo-ai/pi@<version>
```

Use the gabo CLI version as `<version>`. Before the package is on npm, install it from a checkout with `pi install /path/to/gabo-ai/packages/pi`. Check it with `pi list`.

- **`gabo init` and `gabo-runtime-pi` stay required.** The package itself writes no project file and builds no agents; with `runtimes.coder: pi`, `gabo-runtime-pi` builds them in `.pi/agents/`.
- **It is not a subagent package.** gabo on Pi still needs one, such as gentle-pi; without it the extension warns at session start and blocks writes under `.pi/agents/`.

See [`packages/pi/README.md`](packages/pi/README.md) for the package layout and coexistence rules, and [`packages/pi/VERIFY.md`](packages/pi/VERIFY.md) for the manual verification run before each release.

### Install with curl (planned)

> **Not available yet.** A one-line install script is planned but does not exist. Use one of the methods above.

> **Self-update does not work yet.** When a project's `.gabo/VERSION` is newer than the installed CLI, `gabo init` is designed to update itself from a signed GitHub release. No release with signed assets exists yet and gabo pins no signing key, so `gabo init` prints the upgrade command instead. Install the newer version with `go install` as above.

## Quick start

1. In your project root, run `gabo init`. It copies the skill catalog to `.gabo/skills/`, writes the version stamp `.gabo/VERSION`, installs the `gabo-reviewer` skill at user level, and prints the setup prompt. With no `gabo.yaml` yet, it writes no engine file.
2. Open your coding agent in the project root.
3. Paste the prompt that `gabo init` printed (`gabo prompt` prints it again at any time). You can also ask the agent to configure gabo-ai.
4. Answer the setup questions. The agent asks one at a time and applies no value you did not accept. It writes the config, builds the agents for your engine and checks each slot.
5. Run `gabo init` again now that `gabo.yaml` exists. It writes the project-level integration for each engine in `runtimes.*` (see [Compatibility](#compatibility)) and ends with an `engine integration:` report of what you still need to run, such as trusting the project in your engine.
6. Run `gabo check` to confirm everything passes.
7. Commit the changed files (`gabo.yaml`, `.gabo/`, adapters, agent folders, and the engine files `gabo init` wrote).

You can also skip step 1: paste the [setup prompt](docs/setup-prompt.md#prompt) into your agent. It installs the CLI if `gabo` is missing and runs `gabo init` for you. Both ways are safe to rerun.

## Commands

The CLI has only four commands. Everything else is done by your agent through the skills.

| command | what it does |
|---|---|
| `gabo init` | Copies the skill catalog to `.gabo/skills/`, writes `.gabo/VERSION` and installs the reviewer skill at user level. With a `gabo.yaml`, it also writes the project-level integration of each declared engine and reports what you still need to run. It never touches an existing config or your global engine config. |
| `gabo check` | Validates, read-only, the config files, the agents, slot coverage (each adapter explains every operation of its contract) and the catalog version. Exits non-zero on any failure. |
| `gabo version` | Prints the gabo version. |
| `gabo prompt` | Prints the setup prompt, and nothing else, to paste into your agent. It reads no config and writes nothing, with or without a project. |

## Configuration

gabo merges two layers. Secrets never go in either one; config files hold only the names of the environment variables that carry them.

| file | scope | holds |
|---|---|---|
| `gabo.yaml` (git top-level of the project) | team, committed | `repo`, `runtimes`, `slots`, `review`, `mode`, `plan_change`, `parallelism`, `developers`, `worktree` |
| `~/.config/gabo/config.yaml` (or `$XDG_CONFIG_HOME/gabo/config.yaml`) | personal | `alias`, `language`, `tone`, `interaction_level` |

The project file overrides the user file key by key. Unknown keys fail. The setup agent writes both files through the `gabo-config-writer` skill. See the annotated example [`apps/cli/gabo.example.yaml`](apps/cli/gabo.example.yaml) and the JSON Schemas in [`schemas/`](schemas/).

## Running a wave

After setup, open your agent in the project and talk to the assistant. Ask it to plan the product, then to start a wave. The assistant delegates:

- planning to directiva, which writes `docs/product` and the task issues;
- each wave to `gabo-cycle`, which claims the unblocked tasks, creates worktrees, launches coders up to `parallelism`, and cleans up what it created.

Coder questions travel through the cycle to the assistant. In `manual` mode, the assistant asks you one question at a time. In `auto` mode, it records each decision with its reason. At the end of each wave, you get a wave report. See journey J3 in the [app flow](docs/product/APP_FLOW.md#j3-run-a-wave-without-watching-r4-r5-r15).

## Upgrading

1. Install the newer CLI.
2. Run `gabo init` in the project again. It refreshes `.gabo/skills/` and `.gabo/VERSION`, adds any engine integration key that is missing (it keeps the ones you set) and does not touch your config. Do what its `engine integration:` report asks. An existing Pi entry `npm:@gabo-ai/pi@<old version>` in `.pi/settings.json` is kept as your choice; change it to the new version by hand.
3. Ask your agent to rebuild the agents.
4. Run `gabo check` until it passes.

`gabo check` tells you when a rebuild is needed. See the Rebuild section of `.gabo/skills/gabo-setup/SKILL.md` in your project.

## Documentation

- [Setup prompt and install details](docs/setup-prompt.md)
- [Architecture](docs/architecture.md)
- [Product requirements (PRD)](docs/product/PRD.md)
- [Technical design (TRD)](docs/product/TRD.md)
- [App flow](docs/product/APP_FLOW.md)
- [UX and UI](docs/product/UX_UI.md)
- [Migrations log](docs/product/MIGRATIONS.md)
- [Intents](docs/product/intents/README.md)
- [Archived planning documents](docs/product/archive/README.md)

## License

[MIT](LICENSE)
