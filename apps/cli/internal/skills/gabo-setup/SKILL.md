---
name: gabo-setup
description: "Trigger: gabo setup, first setup, configure gabo, setup prompt, change slot adapter or runtime, rebuild agents after gabo upgrade. Guides setup: ask, adapters, config, build agents, gabo check, slot smoke test, per-change product model."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "3.0"
---
## Activation Contract

Load in the owner's agent (Claude Code, Pi or another engine) after the setup prompt (the only entry text, printed by `gabo init`) or when the owner asks it to configure gabo-ai, change a slot or a runtime (J1, J7), and after a gabo upgrade (Rebuild). This skill orders the work; the writing is done by the skills that own it: `gabo-adapter-writer` (adapters), `gabo-config-writer` (both config files), `gabo-agent-templates` with the runtime skill of each engine in use (`gabo-runtime-claude`, `gabo-runtime-codex`, `gabo-runtime-opencode`, `gabo-runtime-pi`, or one written first through `gabo-runtime-builder`). All of them are read from `.gabo/skills/`.

## Hard Rules

- **Owner answers only (R12).** Apply no value without the owner's answer. Offer a recommendation with its reason, then wait. One question at a time, in a quick burst. In auto mode, only `gabo-config-writer`'s recorded-decision path may choose a value.
- **No secrets (R21).** Never ask for, accept or write a secret; a credential is configured by its env var name only.
- **Four commands.** The CLI has only `gabo init`, `gabo check`, `gabo version` and `gabo prompt`. You may run `gabo prompt` to reprint the setup prompt (read-only, with or without a project). Use no other gabo command and no gabo MCP tool; config is still edited as YAML by `gabo-config-writer` and validated by `gabo check`.
- **Recommended, not required.** Offer the gentle-ai stack (gentle-ai ODD and RDD for `methodology`, Engram for `memory`), and for Pi a subagent package (gentle-pi: `npm i -g gentle-pi` or `pi install npm:gentle-pi`; another package is acceptable), each as recommended, not required, with why; install it only if the owner accepts. Never edit gentle-ai's or gentle-pi's own configuration.
- **Idempotent.** Read `gabo.yaml` and the global user file first; skip every question whose answer is already there, so rerunning setup changes only what the owner changes.
- Never place `gabo-reviewer` in the project; `gabo init` installs it at user level and the runtime skill only links it (D20).

## Steps

1. **Ask.** Note the setup start time (UTC ISO 8601) in the session. If `.gabo/skills/` is missing, run `gabo init`. Then ask, skipping what is already set:
   - Identity: read `git config user.email`; if `developers` has no alias for it, ask once how to call the owner (R16).
   - Personal keys, asked once for every repository: `language`, then `tone` and `interaction_level` if the owner wants them.
   - `repo.name`, `repo.integration_branch` (never `main`, D13), `review.checks`, `parallelism`.
   - If a check needs a secret or external service (for example an integration test database URL), either list the variable names in `review.check_env` and have the owner export the values where the reviewer runs, or use a check script that provisions its own ephemeral database. Never write a value to the config (R21); a shell guard like `test -n "$X" && ...` does not work, checks run as argv without a shell. Every listed variable is handed to the PR code the checks run, so list only test-scoped values (a throwaway test database), never the owner's personal tokens or loader variables such as `LD_PRELOAD`, `BASH_ENV` or `GIT_*`.
   - `mode` (`manual` or `auto`), `plan_change.scope` (`task`, `group` or `global`) and `plan_change.global_without_owner`.
   - The runtimes: `runtimes.coder`, `runtimes.reviewer` and `runtimes.assistant`. `runtimes.reviewer` has no default: ask it explicitly even when the others are answered. Each accepts `claude` or `pi`. A `pi` reviewer reuses the owner's existing Pi login and default model; ask for `review.pi_credential_env` (an env var name, never its value) only if the owner wants the review to use a separate credential instead.
   - Ask where each agent runs (`slots.agent_host`: this machine or a remote host) and which tracker holds the tasks (`slots.tasks`); then explain how the agents communicate: through the tracker (issues, PRs and their comments), the shared `memory` slot and, only if configured, `direct_sessions`; agents on different hosts never talk directly.
   - Each other slot (`repository_host`, `memory`, `methodology`, `reviewer_placement`, and `notifications` or `direct_sessions` if wanted): how it is handled and with what, suggesting options from what is installed. For `memory`, explain **Shared memory** first (below) and ask its questions there. For Pi, check the subagent package now (Hard Rules).
2. **Adapter.** For each slot, the owner points to an adapter (a catalog adapter such as `gabo-tasks-github`, or a skill or document they have), or co-writes one with you through `gabo-adapter-writer`. Check that every operation of `gabo-contract-<slot>` is explained; ask about each gap, one at a time.
3. **Write the config.** Through `gabo-config-writer`: personal keys to the global user file, team keys and `slots.<slot>.adapter` to `gabo.yaml`, one key at a time, with its `gabo check` after each write.
4. **Build the agents.** For each engine named in `runtimes.*`, load its runtime skill and build every agent of the neutral agent spec from `gabo-agent-templates`. An engine without a runtime skill in `.gabo/skills/`: run `gabo-runtime-builder` first; if it refuses the engine (no subagents or skills), report that runtime as not built.
   - **Main-session entry.** The owner's main-session entry is defined by the Main-Session Entry table of `gabo-agent-templates`. For Claude Code and OpenCode, `gabo init` writes the main-session key (`agent` in `.claude/settings.json`, `default_agent` in the OpenCode project config) and the runtime skill of `runtimes.assistant` builds the agent file and link of the default agent `gabo-assistant`; for Codex and Pi that runtime skill writes the scoped block in `AGENTS.override.md`, `AGENTS.md` (or, for Pi, an existing `CLAUDE.md`). `gabo init` removes the key of every other engine and every runtime skill removes the agent file, link and block of every other full-tier engine it finds, so the entry follows `gabo.yaml`.
   - **After you change `runtimes.*`, run `gabo init`, then rebuild every agent through the runtime skill of each engine in `runtimes.*`**, the new assistant engine included, before step 5, so the new engine gets its entry and the old one loses it. Never write or remove the entry yourself; a runtime skill that reports `run gabo init` means the key is missing: run `gabo init` and rebuild.
5. **Check.** Run `gabo check`, fix every failure it reports (a config key with `gabo-config-writer` and the owner, a coverage line with `gabo-adapter-writer`, a missing or stale agent with the runtime skill) and rerun `gabo check` until it passes. A failure you cannot fix without the owner is one question.
6. **Smoke test.** For each configured slot, launch one subagent of the working kind (for example `gabo-coder`) that runs one harmless operation through the slot's adapter and reads the result back: `tasks` read one issue, `repository_host` read one PR status, `memory` see the memory probe below, `methodology` report its coder instructions, `reviewer_placement` report the placement, `agent_host` report where agents run. A slot whose operation fails stays unverified (APP_FLOW A3): the report names the failing operation, and you ask the owner how to fix the adapter; after the fix rerun `gabo check` and that slot's smoke test, nothing else.
   - **Memory probe.** The memory contract has no delete operation and no save under an arbitrary topic, so the probe uses the backend's own memory tool that the adapter names, and the report says so. With a backend that can delete, write a note only under a dedicated scratch topic `gabo/setup-probe/<timestamp>` (UTC), read it back, then delete it and check that the topic reads back nothing, so nothing stays in the project memory; a probe still there after one retry stays unverified and is reported with its key for the owner to remove. With a backend that has no delete operation, the probe writes nothing: it runs one read-only `search` through the adapter, and the R28 save in step 7, read back, is the write check.
7. **Record the start (R28).** Right after the `memory` slot is verified, `save decision` through it: `what: setup start <UTC ISO 8601>`, `question: when did setup start (R28)`, `choice:` the timestamp, `reason: time to first wave`, `wave: none`, `task: global`. Read it back. An unverified `memory` slot means nothing is saved: report it.
8. **Product model.** Guide the project into the per-change model: a short global context, then one intent at a time, each with its own `spec.md` and `plan.md` under `docs/product/intents/<NNNN-slug>/`. Skip what `docs/product/` already holds; ask one question at a time.
   - **Stack.** Ask which technologies to use: language, framework, database and hosting. The owner chooses; you only propose options, each with its reason, and helps configure the one chosen, never imposing one. Record the choice in the stack and tooling table of `docs/product/TRD.md` through `gabo-dev-lead`.
   - **Global context.** Scaffold it short, each document through the agent that owns it: a conceptual PRD (what the product offers and what the user gets) through `gabo-pm`; a TRD with conventions only through `gabo-dev-lead`; a `DESIGN.md` only if the app has a UI, through `gabo-frontend`; a `MIGRATIONS.md` only if the app has a database, through `gabo-db-master`; and an empty `docs/product/intents/`. There is no upfront schema, plan or stories: schema changes, UX, flow and tasks come per intent.
   - **Waves.** Ask about `wave.after` and write the answer through `gabo-config-writer`, with its `gabo check`: `next` chains to the next accepted intent when a wave ends, `wait` stops after the wave. Explain that the default is `wait`, that each wave runs one intent, and that there is no batch of accepted intents: the owner accepts each spec one at a time, and the production gate stays the owner's too.
   - **First intent.** Point the owner to creating the first intent through `gabo-pm`; setup writes no intent itself.
9. Tell the owner which files changed (`gabo.yaml`, `.gabo/`, adapters, agent folders, the main-session entry such as `AGENTS.md`, and the engine config keys `gabo init` writes such as `.claude/settings.json`, `docs/product/`) so they can commit them; commit only if the owner asks.

## Shared memory

Explain this in plain words before configuring the `memory` slot (R37, R38), then ask; apply nothing without the owner's answer (R12).

- **The recommendation.** Use one memory backend and one memory project name shared by every agent of the project, local and cloud, for every developer, except the reviewer, which never reads or writes it (R9), so all of them coordinate from the same context. The reviewer agent and its scripts get no memory slot (D6). A local-only memory is acceptable, for example for a solo developer.
- **Backends.** Offer first, as recommended: Engram with Engram Cloud sync, because one self-hostable server shares a whole project across machines and developers through autosync, and it is the memory of the gentle-ai stack. If the owner picks Engram Cloud, mention that each user's credential needs an explicit grant for the project on that server; granting it is the owner's backend setup. For users who only use Claude Code, also list:

  | backend | shares across machines | shares across developers |
  |---|---|---|
  | claude-mem | only with its paid Cloud Sync (per user) | not through Cloud Sync; only its paid TeamBrain plan or export and import |
  | Mem0 (hosted MCP) | yes, memories live in the Mem0 account | yes, through a shared Mem0 organization and project |

  Both are user-supplied adapters: the owner points `slots.memory.adapter` at an adapter they have or co-write with `gabo-adapter-writer`; gabo ships none for them until PRD open question 13 is answered.
- **What setup records.** Only `slots.memory.adapter` (through `gabo-config-writer`) and the memory project name, written in the project-owned memory adapter file that `slots.memory.adapter` points to; never in the catalog copy under `.gabo/skills/`, which `gabo init` replaces and which is human-only. There is no new config key. If `slots.memory.adapter` points into the catalog copy (a catalog adapter), ask the owner to make a project-owned copy first, through `gabo-adapter-writer`, which points `slots.memory.adapter` at the copy; then write the name there. If that adapter file lives under a `review.human_only_paths` entry (for example `.claude/`), tell the owner the write lands in a human-only path: its PR needs a human merge and the reviewer never auto-merges it.
- **Not setup's job.** Setup never sets up the backend, its credentials or its access, and writes no secret (R21); the owner does that in their own setup outside gabo.

## Rebuild

Run after a gabo upgrade (TRD 9.4), or when `gabo check` prints one of these lines:

| `gabo check` line | action |
|---|---|
| `this project uses gabo <v>; run gabo init to upgrade it` (catalog older than the CLI) | run `gabo init`, then rebuild every agent |
| `✗ no stamp: run gabo init` | run `gabo init`, then rebuild every agent |
| `✗ agent <id> built with <v>, catalog is <w>: ask your agent to rebuild` | rebuild every agent |
| `✗ catalog version <w> is newer than gabo <v>` | rebuild nothing: report its `next:` upgrade command to the owner |
| `✗ <file>: gabo assistant entry missing for <engine>` (or `incomplete`, or `duplicated`) | when its `next:` names `gabo init`, run `gabo init` (it writes the main-session key); otherwise rebuild every agent, and the runtime skill of `runtimes.assistant` rebuilds the agent file, link or block |
| `✗ <file>: gabo assistant entry for <engine>, but runtimes.assistant is <value>` | run `gabo init` (it removes the other engine's key), then rebuild every agent; each runtime skill removes the gabo entry of every other full-tier engine |

1. `gabo init` refreshes the catalog copy `.gabo/skills/` and the stamp `.gabo/VERSION` and reinstalls the user-level `gabo-reviewer` skill, and writes the engine config keys for `runtimes.*` (the main-session key included); it never touches the config files `gabo.yaml` and the global user file, so ask no setup question again.
2. For each engine in `runtimes.*`, rebuild every agent through its runtime skill, which rewrites each agent's `gabo-version` line, and the runtime skill of `runtimes.assistant` builds the agent file and link of the main-session entry (or its block); `gabo init` writes the main-session key.
3. Then rerun `gabo check` and fix what fails until it passes, as in step 5.

## Output Contract

A checklist per step: done, skipped (reason) or failed (failing operation and next step). Then each slot as `verified` or `unverified: <failing operation>`, the last `gabo check` result, the setup start record (saved or why not), the product model (stack recorded, global docs scaffolded, `wave.after`), the open questions for the owner, and any unverified write.
