---
name: gabo-adapter-writer
description: "Trigger: write an adapter, co-write a slot adapter, missing adapter, adapter does not cover, new tracker or tool for a slot. Co-writes a slot adapter with the owner from its contract skill, records it and runs gabo check."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load when a slot has no adapter for the owner's tool: J1 steps 5 and 6 (receive the adapter, check coverage) when the owner chooses to co-write one, J7 when the owner switches a slot to another tool, and whenever `gabo check` reports `✗ slot <slot>: adapter <file> not found` or `✗ adapter <file> does not cover: <operations>`. The contract skill says WHAT agents must do; the adapter says HOW with the team's own tool (D2, D4). gabo ships no tool code: the adapter is a Markdown file of exact commands.

Inputs: the slot name (`tasks`, `repository_host`, `memory`, `methodology`, `reviewer_placement`, `notifications`, `direct_sessions`, `agent_host`) and the tool the owner named. Use `gabo-config-writer` for the config write and the `memory` slot adapter only in auto mode.

## Hard Rules

- **The contract is the outline.** Read `gabo-contract-<slot>` from `.gabo/skills/` (the slot name with `_` as `-`). Its `## Operations` section lists the operations; the adapter has one `### <operation>` heading per contract operation, spelled exactly as in the contract, under its own `## Operations` section. A heading inside a fenced code block does not count; never rename, merge or skip an operation. Use the contract's `examples/` adapter as the shape, never as the content for another tool.
- **Exact commands only.** Each operation gives the exact commands or tool calls for the owner's tool: the command line with its placeholders (`<n>`, `<branch>`), the identifiers it needs and how to read the result back (the contract's re-read rule, R24). Take them from the owner, a document or skill the owner points to, or the tool's own documentation or `--help` for the installed version, and name that source under the heading. You may run a read-only command (`--help`, a list or view) to confirm a command exists; never run a write to test it.
- **Never invent a command.** If you cannot fill an operation from those sources, do not guess a flag, endpoint or subcommand. Ask the owner about that operation, one question at a time, naming the operation and what you already know. A subagent returns the open operations as questions in its outcome instead of asking.
- **Owner answers only (R12).** Write an operation only from the owner's answer or a cited source. In auto mode (`mode: auto` in `gabo.yaml`) with the owner not asked, an operation you still cannot fill stays open: record the gap through `memory` (`save decision`: what, question, choice, reason, mode, wave, task), leave the slot unverified and report it; never fill it to make `gabo check` pass.
- **No secrets (R21).** An adapter never holds a token, password or API key. A command that needs a credential reads it from an env var name the owner gives (for example `$LINEAR_API_KEY`); if the owner pastes a secret, do not store it and ask for the env var name instead.
- **Location.** Save the adapter in the repository as `.gabo/adapters/<slot>-<tool>.md` (for example `.gabo/adapters/tasks-linear.md`), or where the owner names, never under `.gabo/skills/`: `gabo init` refreshes that catalog copy. Keep an existing adapter's path; edit it in place.
- **Record, then check.** Point `slots.<slot>.adapter` at the file only through `gabo-config-writer`, never by editing `gabo.yaml` here. Then run `gabo check` and fix every coverage line it reports for this adapter; rerun `gabo check` until it passes for this slot.

## Steps

1. Read the contract `gabo-contract-<slot>` and its example adapter; list its operations in order.
2. Ask the owner which tool and version they use and where its commands are documented; read an adapter or document the owner already has before writing.
3. Draft the file: a first line `# <slot> adapter: <tool>`, a short paragraph naming the tool, its version and the sources, then `## Operations` with one `### <operation>` heading per contract operation in contract order, each with the exact commands and the read-back.
4. For each operation you cannot fill, ask the owner (one question at a time), then write the answer. Repeat until every heading has commands, or record the gap as above.
5. Save it at its location and read it back.
6. Record it: through `gabo-config-writer`, set `slots.<slot>.adapter` to the file path (relative to the repository root).
7. Run `gabo check`. For `✗ adapter <file> does not cover: <operations>`, add or fix each named heading (spelling, a missing `## Operations` section, a heading in a fence) with the owner's commands, then rerun `gabo check` until it passes for this slot. A failure on another slot or key is reported, not fixed here.
8. The caller then verifies the slot with one harmless operation (J1 step 10); a failed verify leaves the slot unverified and names the failing operation.

## Output Contract

Return: slot, tool, adapter path, each operation with its source (`owner`, `<doc or skill>`, `--help`) or `open`, the config write from `gabo-config-writer`, the last `gabo check` result for this slot (`pass` or its `✗` and `next:` lines), the questions for the owner (one per open operation), decisions recorded in auto mode, and any unverified write.
