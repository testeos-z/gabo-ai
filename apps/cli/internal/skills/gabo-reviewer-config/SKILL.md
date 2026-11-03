---
name: gabo-reviewer-config
description: "Trigger: configure the reviewer, pick the reviewer engine, runtimes.reviewer, set up code review, reviewer engine missing. Configures claude, codex, pi or opencode as the gabo reviewer engine; never runs a review."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Load when the owner picks or changes the engine that reviews pull requests (`runtimes.reviewer`), during `gabo-setup` or later. This is the single catalog source of the reviewer configuration for every engine that ships gabo (intent 0001 S10, D4): it behaves the same whether it runs in Claude Code, Codex, Pi or OpenCode. It configures one of the four engines, `claude`, `codex`, `pi` or `opencode`, as reviewer.

This skill never runs a review. The deterministic reviewer scripts live in the user-level `gabo-reviewer` skill, which `gabo init` installs at `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer` (D20); the `reviewer_placement` adapter and the `gabo-reviewer` agent run them. See `gabo-reviewer-notes` for how a review works.

## Hard Rules

- Never run a review, never call the reviewer scripts, never post a verdict or the reviewer marker.
- Place the reviewer scripts at user level only through `gabo init`. Never copy, edit or link them into a repository, and never ship them in a plugin or package (D20).
- Install nothing yourself: no engine CLI, no package, no login. When the picked engine's CLI is missing, name the engine and its missing CLI, install nothing and write nothing.
- Write `runtimes.reviewer` only through `gabo-config-writer`, at project level, and only after every step before it passed. On any refusal leave `runtimes.reviewer` unchanged.
- Write no secret into any file (R21): a credential is an env var name, never its value.

## Engines

| engine (`runtimes.reviewer`) | CLI on `PATH` | isolation test |
|---|---|---|
| `claude` | `claude` | none |
| `pi` | `pi` | none |
| `codex` | `codex` | required, once, at configuration time |
| `opencode` | `opencode` | required, once, at configuration time |

`claude` and `pi` need no isolation test: they keep working as reviewers exactly as today, with their own clone and home (R22).

## Steps

1. **Pick.** Ask the owner which engine reviews, one question, the four engines of the table as the only answers. `runtimes.reviewer` has no default; in auto mode take the engine the owner already named and record the choice with the `memory` slot (save decision).
2. **Check the CLI.** Run `command -v <cli>` for the engine's CLI from the table, read-only. When it prints nothing, stop: name the engine and its missing CLI (for example "reviewer engine `codex`: CLI `codex` not found on `PATH`"), install nothing and write nothing, and tell the owner to install it and ask again, or to pick another engine.
3. **Isolation test (`codex`, `opencode` only).** Run the one-time isolation test for that engine now, never during a wave: a planted `.opencode/` plugin (`opencode`), or a planted `AGENTS.md` and `.codex/` config (`codex`), in a scratch PR clone must not execute or change the reviewer's behaviour or config. On pass, record the pass marker for that engine and its version. On fail, or when this catalog copy has no isolation-test script yet, name the engine and the failed check, record no pass marker, leave `runtimes.reviewer` unchanged and ask the owner to pick another engine.
4. **Record.** Set `runtimes.reviewer` to the engine through `gabo-config-writer`.
5. **Place the scripts.** Tell the owner to run `gabo init` when `gabo check` reports the user-level reviewer skill missing or outdated; it is the only way the scripts reach `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`. Then rebuild the agents through the runtime skill of each engine in `runtimes.*`, which links the user-level copy into that engine's home.
6. **Check.** Run `gabo check` and fix what it reports through the skill that owns it; never edit around it.

## Output Contract

Report: the engine picked, the CLI check result, the isolation test result (`none` for `claude` and `pi`; pass with marker, or the failed check), whether `runtimes.reviewer` changed, whether `gabo init` is still needed, and the `gabo check` result. On a refusal report exactly what is missing and that nothing was installed or written.
