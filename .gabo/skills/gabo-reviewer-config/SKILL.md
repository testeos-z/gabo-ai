---
name: gabo-reviewer-config
description: "Trigger: configure the reviewer, pick the reviewer engine, runtimes.reviewer, set up code review, reviewer engine missing. Configures claude, codex, pi or opencode as the gabo reviewer engine; never runs a review."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Load when the owner picks or changes the engine that reviews pull requests (`runtimes.reviewer`), during `gabo-setup` or later. This is the single catalog source of the reviewer configuration for every engine that ships gabo (intent 0001 S10, D4): it behaves the same whether it runs in Claude Code, Codex, Pi or OpenCode. It configures one of the four engines, `claude`, `codex`, `pi` or `opencode`, as reviewer, or `session` (intent 0013 S11).

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

`session` reviews each wave on the engine of the main session that starts it, one of the engines listed in `runtimes.assistant`; the reviewer scripts refuse any other engine. So under `session` steps 2 and 3 run for each engine listed in `runtimes.assistant`, and `runtimes.assistant` must already be set.

## Steps

1. **Pick.** Ask the owner which engine reviews, one question, the four engines of the table or `session` as the only answers. `runtimes.reviewer` has no default; in auto mode take the engine the owner already named and record the choice with the `memory` slot (save decision).
2. **Check the CLI.** Run `command -v <cli>` for the engine's CLI from the table, read-only; under `session`, for each engine listed in `runtimes.assistant`. When it prints nothing, stop: name the engine and its missing CLI (for example "reviewer engine `codex`: CLI `codex` not found on `PATH`"), install nothing and write nothing, and tell the owner to install it and ask again, or to pick another engine.
3. **Isolation test (`codex`, `opencode` only).** Run the one-time isolation test for that engine now, exactly once, at configuration time, never during a wave; under `session`, run it once for each listed `codex` or `opencode` engine, and none when only `claude` or `pi` are listed. If any of them fails, leave `runtimes.reviewer` unchanged. The command: `bash <skill dir>/scripts/isolation-test.sh <engine>`, where `<skill dir>` is the folder holding this `SKILL.md` (`.gabo/skills/gabo-reviewer-config` after `gabo init`). It builds a scratch PR clone (`mktemp -d`) with a planted `.opencode/` plugin (`opencode`), or a planted `AGENTS.md` and `.codex/` config (`codex`), and runs the engine there with the reviewer's own config and home: `codex exec --ignore-user-config --ignore-rules -s read-only --ephemeral` with a `CODEX_HOME` of its own, or `opencode run --standalone` with its own `HOME` and `XDG_*`. The planted files must not execute or change the reviewer's behaviour or config. Exit 0: it passed and wrote the pass marker `${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer-isolation/<engine>` holding the engine version, which a wave checks before it starts. Exit 1: it names the engine and the failed check and wrote no marker; record no pass marker, leave `runtimes.reviewer` unchanged and ask the owner to pick another engine. Never write the marker yourself and never run the script for `claude` or `pi`.
4. **Record.** Set `runtimes.reviewer` to the engine, or to `session`, through `gabo-config-writer`. Under `session`, an engine added to `runtimes.assistant` later needs this skill again before it reviews: the scripts refuse a `codex` or `opencode` engine without its pass marker.
5. **Place the scripts.** Tell the owner to run `gabo init` when `gabo check` reports the user-level reviewer skill missing or outdated; it is the only way the scripts reach `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`. Then rebuild the agents through the runtime skill of each engine of the engine set (every engine listed in `runtimes.assistant`, plus `runtimes.coder` and `runtimes.reviewer` when each names an engine), which links the user-level copy into that engine's home.
6. **Check.** Run `gabo check` and fix what it reports through the skill that owns it; never edit around it.

## Output Contract

Report: the engine picked (or `session` with the listed engines), the CLI check result per engine, the isolation test result per listed engine under `session` (`none` for `claude` and `pi`; pass with the marker path and engine version, or the engine and the failed check the script printed), whether `runtimes.reviewer` changed, whether `gabo init` is still needed, and the `gabo check` result. On a refusal report exactly what is missing and that nothing was installed or written.
