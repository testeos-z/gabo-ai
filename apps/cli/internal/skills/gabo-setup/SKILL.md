---
name: gabo-setup
description: "Trigger: gbai setup, first setup, configure gbai, onboard gbai, change slot adapter, setup prompt. Guides first setup: config files, slot adapters, verification, generate."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Load in the assistant when there is no config, a key is missing, or the owner wants to change a slot (D21: setup starts from a prompt pasted into Claude Code or Pi; there is no TUI).

## Hard Rules

- NEVER ask for, accept or write a secret. Config holds env var names only (R21).
- No silent defaults (R12): offer a recommendation, write a key only after the owner answers. One question at a time, in a burst the owner can answer quickly.
- Personal keys (`alias`, `interaction_level`, `tone`, `language`) go only in `~/.config/gbai/config.yaml`; project keys only in `gbai.yaml`.
- Edit the YAML directly; `gbai generate` validates it (D18). Read an existing file first and merge; never overwrite blindly.
- gentle-ai (ODD methodology, RDD coder-side review, Engram memory) is the recommended stack, never required; install it only if the owner accepts.

## Execution Steps

1. Identity: `git config user.email`; ask the alias once; add it to `developers` and to the global file.
2. Personal keys → global file.
3. Project keys → `gbai.yaml`: `mode`, `repo.{name, integration_branch}`, `runtimes.*`, `generation.targets`, `reviewer.{placement, signing}`, `review.checks`, `plan_change.*`, `parallelism`.
4. Slots (`tasks`, `repository_host`, `memory`, `methodology`, `reviewer_placement`): pick an existing adapter or co-write one; check it covers every operation of its contract (TRD 9.1).
5. Verify each slot with one harmless round trip (for example read one issue, read one PR, save and read a memory note). A failed slot stays unverified: name the failing operation and ask how to fix the adapter.
6. Ensure the user-level `gbai-reviewer` skill is installed (`~/.claude/skills/gbai-reviewer/`).
7. Run `gbai generate` when the compiler is available; until then the agents and skills in `.claude/` are maintained by hand.

## Output Contract

A checklist per step: done, skipped (reason) or failed (failing operation and next step).
