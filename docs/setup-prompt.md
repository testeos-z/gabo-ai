# Setup prompt

The setup prompt is the only entry text of gabo-ai setup (TRD 9.4, D25, PRD R36). Your coding agent follows it, in your project, to install the CLI when it is missing, run `gabo init`, ask you the setup questions, write the config, build the gabo agents, run `gabo check`, verify each slot and guide the project into the per-change model (a short global context, then one intent at a time).

The block under [Prompt](#prompt) is the prompt, byte for byte the file embedded in the CLI (`apps/cli/internal/setupprompt/setup-prompt.md`). Do not edit it here: change the embedded file and copy it over; a Go test fails when the two differ.

## Two ways to start

Both entries reach the same setup; pick the one that fits you.

| entry | you do | your agent does |
|---|---|---|
| (a) Terminal | [Install the CLI](#install-the-cli), run `gabo init` in the project root, then open your agent there and ask it to configure gabo-ai (or paste what `gabo init` printed) | loads the setup skills from `.gabo/skills/` and continues with the questions |
| (b) Pasted prompt | open your agent in the project root and paste the [prompt](#prompt) | installs the CLI if `gabo` is missing, runs `gabo init`, then continues with the questions |

`gabo init` copies the skill catalog to `.gabo/skills/`, writes the version stamp `.gabo/VERSION` and the catalog content hash `.gabo/CATALOG_HASH`, installs the `gabo-reviewer` skill at user level (never in the project), replacing a copy whose `VERSION` differs or whose content hash `REVIEWER_HASH` differs from the embedded reviewer at the same version, and, on a fresh project, prints the prompt after its `next:` line. It never touches an existing config, so rerunning it, or the prompt, is safe.

`gabo init` also configures the engines declared in `gabo.yaml` (`runtimes.coder`, `runtimes.reviewer` and `runtimes.assistant`): it merges the gabo plugin or package keys into the project settings of Claude Code (`.claude/settings.json`), Codex (`.codex/config.toml`) and Pi (`.pi/settings.json`), keeping every other key, writes only `default_agent` for OpenCode in its project config (an existing `experimental.subagent_depth` is left exactly as it is) and reports whether the stage-gate plugin that `gabo-runtime-opencode` installs is in place, writes the main-session key of each engine listed in `runtimes.assistant` (`agent` or `default_agent`, see below), and ends with an `engine integration:` report that lists what it wrote and what you still need to do. On a fresh project there is no `gabo.yaml` yet, so it writes no engine file and says so; once your agent has written `gabo.yaml`, `gabo init` runs again (step 5 of the prompt does it) and configures the declared engines. It never creates or edits `gabo.yaml`.

`gabo prompt` prints the same prompt, byte for byte and nothing else, at any time, with or without a project; it reads no config and writes nothing.

You answer every question; the agent applies no value you did not accept and asks one question at a time. Config files hold only the names of environment variables, never a secret value.

## Install the CLI

The CLI needs Go (the version named in `apps/cli/go.mod`). Install the newest release:

```sh
go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest
```

`gabo init` accepts only a release version (`x.y.z`). `@latest` resolves to the newest release tag; while the repository has no release tag, Go installs a pseudo-version and `gabo init` refuses it. In that case build from a clone and name the version yourself, using the project's `.gabo/VERSION` when it already has one:

```sh
git clone https://github.com/testeos-z/gabo-ai.git
cd gabo-ai
go -C apps/cli install -ldflags "-X main.buildVersion=<x.y.z>" ./cmd/gabo
```

Check the install with `gabo version`.

## Your engine

The prompt names no engine; the agent loads the runtime skill of the engine it runs in from `.gabo/skills/`. During setup `gabo-setup` asks for `runtimes.assistant` first: one engine, or a list without duplicates such as `[claude, pi]`, so you can open your main session on any listed engine. `runtimes.coder` and `runtimes.reviewer` each take one engine or `session`, the engine of the main session that starts the wave. The agent then builds the agents of every engine of the engine set (each listed assistant engine, plus `runtimes.coder` and `runtimes.reviewer` when each names an engine), loading each engine's runtime skill; every agent file names the engine it was built for, and `gabo check` fails one found in another engine's agent directory with `✗ <file>: built for <value>, not <engine>`.

| engine | runtime skill | agents built in | prerequisite |
|---|---|---|---|
| Claude Code | `gabo-runtime-claude` | `.claude/agents/` | none |
| Pi | `gabo-runtime-pi` | `.pi/agents/` | a subagent package (below) |
| Codex | `gabo-runtime-codex` | `.codex/agents/` (TOML) | none |
| OpenCode | `gabo-runtime-opencode` | `.opencode/agents/` | none beyond OpenCode itself |
| another engine | written first with `gabo-runtime-builder` | the engine's own agent folder | subagents and skills; the builder refuses an engine without them and says why |

**Main-session entry.** Your own session starts as the gabo assistant; coder and reviewer sessions keep their role. `gabo init` writes the config key, `agent` or `default_agent`, for each engine listed in `runtimes.assistant` and removes it from every engine that is not listed; when the agent builds the agents, the runtime skill of each listed engine builds the agent file, the skill link and, for Codex and Pi, the block. Claude Code: `.claude/agents/gabo-assistant.md` and `agent: "gabo-assistant"` in `.claude/settings.json`. OpenCode: `.opencode/agents/gabo-assistant.md` and `default_agent: "gabo-assistant"` in its project config. Codex: a scoped `<!-- GABO_ASSISTANT_START -->` block in `AGENTS.override.md`, else `AGENTS.md`. Pi: the same block in the first existing of `AGENTS.override.md`, `AGENTS.md`, `CLAUDE.md`; Codex and Pi listed together share one block, whose fixed line ``Session engine: loaded by Codex, this session's engine is `codex`; loaded by Pi, it is `pi`.`` tells each session its engine. The prompt itself does not change; `gabo check` verifies the entry: a key gap names `next: run gabo init (it merges "<key>": "gabo-assistant" into <file>), then rerun gabo check` (or `it removes`), any other piece the rebuild. An entry left for an engine that is not listed fails with `✗ <file>: gabo assistant entry for <engine>, but runtimes.assistant is <engines joined with ", ">`. After a change of `runtimes.assistant`, run `gabo init`, then rebuild the agents. See [Engine compatibility](compatibility.md#notes).

**Several sessions.** With a list, a main session on each listed engine can run its own wave at the same time, on different intents. A wave claims its intent before any task; a second session asked for the same intent claims nothing and returns `intent <I####> is claimed by session <session id> (engine <engine>) since <UTC time>; it is released when that wave closes`. Each wave cleans only its own worktrees and branches.

**Pi needs a subagent package.** Pi core has no subagents, so `gabo-runtime-pi` builds nothing until one is present; there is no fallback, and the agent refuses and tells you what to install. Recommended for subagents: gentle-pi, installed with `npm i -g gentle-pi` (adds the `gentle-shell` launcher) or `pi install npm:gentle-pi` (inside plain Pi; npm is used only because gentle-pi's postinstall needs it); it needs Pi >= 1.2.0 and Node >= 22.19.0. Another subagent package you already use is acceptable once the agent confirms it discovers project agents in `.pi/agents/`. As the recommended setup, the refusal also points to the full [Gentle AI framework](https://github.com/Gentleman-Programming/gentle-ai) (its methodology with RDD and the Engram memory stack), set up for Pi with `gentle-ai install --agent pi`. The agent explains why the package is needed and installs it only after you accept; gabo-ai never edits Pi's or gentle-pi's own configuration.

## Recommended stack

The agent offers the gentle-ai stack as recommended, not required, and explains why: gentle-ai with ODD for the `methodology` slot, its RDD as coder-side validation inside that adapter, and Engram for the `memory` slot. It installs it only if you accept; any other methodology or memory adapter works. RDD never replaces the gabo reviewer, which stays the merge gate.

## Shared memory

Before it configures the memory slot, the agent explains that one memory backend and one memory project name should be shared by every agent of this project, local and cloud, for every developer, except the reviewer, which never reads or writes it; a local-only memory is fine, for example for a solo developer. It offers Engram with Engram Cloud sync first as the recommendation, because one self-hostable server shares a whole project across machines and developers through autosync. For teams that only use Claude Code, `gabo-setup` also lists:

| backend | shares across machines | shares across developers |
|---|---|---|
| claude-mem | only with its paid Cloud Sync (per user) | not through Cloud Sync; only its paid TeamBrain plan or export and import |
| Mem0 (hosted MCP) | yes, memories live in the Mem0 account | yes, through a shared Mem0 organization and project |

Both are user-supplied adapters: gabo-ai ships none for them. Setup records only the memory adapter and the memory project name, and never sets up the backend, its credentials or its access; that stays in your own setup outside gabo. The memory smoke test writes only under a scratch topic and deletes the probe afterwards, or writes nothing when the backend cannot delete.

## When setup is done

`gabo check` passes and each slot is reported as verified, or unverified with the failing operation and a question for you. Your project then holds a short global context under `docs/product/` (a conceptual PRD, a conventions-only TRD with the stack you chose, an optional `DESIGN.md` and `MIGRATIONS.md`, and an empty `intents/` folder), and `wave.after` says whether a finished wave takes the next accepted intent or waits (the default). Each change starts as an intent, created through `gabo-pm`, and you accept its spec before a wave runs it. The agent lists the files it changed (`gabo.yaml`, `.gabo/`, adapters, agent folders, `docs/product/`) for you to commit. After a gabo upgrade, run `gabo init` again and ask your agent to rebuild the agents, then rerun `gabo check`.

## Sync config

Once a project is set up, you can ask your agent to sync the work-style config, by saying "sync config" (or "poll-config"), instead of answering every setup question again. The agent applies no value you did not confirm.

1. **Source.** The agent asks "local or remote?" and pre-selects local: it says "I will use the local profile; say remote to use another repository; Enter or ok keeps local." Enter or ok means local.
2. **Local.** It reads `profile.yaml` in your user config folder (`$XDG_CONFIG_HOME/gabo/profile.yaml` when that variable is an absolute path, else `~/.config/gabo/profile.yaml`). If there is none, it says so and offers the remote source or the normal setup.
3. **Remote.** You name another repository, by local path or by GitHub URL. The agent reads only its `gabo.yaml`, read-only, without cloning, and stores nothing from it.
4. **Report and propose.** It shows one diff of the work-style keys against your `gabo.yaml`: `mode`, the runtimes, the adapter names, `parallelism`, `wave.after`, the plan-change and review keys, `reviewer.placement` and the generic human-only paths. The agent never copies repository-specific keys (`repo.name`, `repo.integration_branch`, `review.checks`, `developers`, `worktree.keep`) or personal keys; it asks for them. A secret-like key or value stops the sync and is named by its key path only.
5. **Apply.** You confirm the diff, drop any key, or pick the source value for a key that differs. Only then does the agent write, through `gabo-config-writer`, one key at a time with `gabo check` after each.
6. **Finish setup.** After a write, the agent runs `gabo init` when `.gabo/skills/` is missing or the runtimes changed, then builds the agents, runs `gabo check` until it passes and runs the slot smoke test. None is skipped; a slot whose smoke test did not run is reported as unverified.
7. **Save the profile.** When you ask, the agent saves the work-style keys of this repository as `profile.yaml`, the same file step 2 reads. Only the work-style keys are saved, with adapter names instead of paths and never a secret. If a profile already exists, the agent shows the difference first and writes nothing without your yes.

## Prompt

<!-- gabo-setup-prompt:begin -->
Set up gabo-ai in this project. Work in this project's root directory, follow the steps in order, and skip any step whose result already holds, so running this prompt again is safe. Ask me before every choice and apply no value I did not accept. Never write a secret value anywhere; config files hold only the names of environment variables.

1. If the `gabo` command is missing, install the gabo CLI from https://github.com/testeos-z/gabo-ai/blob/main/docs/setup-prompt.md#install-the-cli and check that it runs.
2. If `.gabo/skills/` is missing, run `gabo init` here.
3. Load these skills from `.gabo/skills/`: `gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, and the runtime skill for the engine you are running in; if `.gabo/skills/` holds none for your engine, load `gabo-runtime-builder` and write that runtime skill first.
4. Following `gabo-setup` and `gabo-config-writer`, ask me the setup questions one at a time, then write the global user file and the project config `gabo.yaml`.
5. Run `gabo init` again here, so it writes this engine's project integration now that `gabo.yaml` exists.
6. With `gabo-agent-templates` and the runtime skill, build every agent of the neutral agent spec in this engine's format.
7. Run `gabo check`, fix every failure it reports and rerun it until it passes.
8. Run the per-slot smoke test from `gabo-setup` and report each slot as verified or unverified with the reason.
9. Following `gabo-setup`, guide this project into the per-change model. Ask me which technologies to use (language, framework, database and hosting): I choose, you only propose options and help configure them, and the choice goes into the TRD. Scaffold a short global context: a conceptual PRD, a conventions-only TRD, a `DESIGN.md` only if the app has a UI, a `MIGRATIONS.md` only if it has a database, and an empty `docs/product/intents/`, with no upfront schema, plan or stories. Ask me about `wave.after`: `next` chains to the next accepted intent when a wave ends, `wait` stops after the wave and is the default; I accept each spec one at a time. Then point me to creating the first intent.

About memory: before you configure the memory slot, explain that one memory backend and one memory project name should be shared by every agent of this project, local and cloud, for every developer, except the reviewer, which never reads or writes it; a local-only memory is fine, for example for a solo developer. Offer Engram with Engram Cloud sync first as the recommendation, list the other backends `gabo-setup` names, record only the memory adapter and the memory project name, and never set up the backend or its credentials. The memory smoke test writes only under a scratch topic and deletes the probe afterwards, or writes nothing when the backend cannot delete.
<!-- gabo-setup-prompt:end -->
