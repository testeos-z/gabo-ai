---
name: gabo-contract-reviewer-placement
description: "Trigger: reviewer placement contract, reviewer_placement slot, where the reviewer runs, VERDICT line, adapter coverage. Operations every gabo reviewer placement adapter must cover."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `reviewer_placement` slot (`slots.reviewer_placement.adapter`, `reviewer.placement`, TRD 9.1). This contract states WHAT must happen to get one review of a pushed PR head; the adapter states WHERE and HOW the scripts of the user-level `gbai-reviewer` skill run (D6, D20). Only placement `local` is implemented in the first release.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- A review request is exactly `PR #<n> head <40-hex sha>`, or, with the optional `engine` field, `PR #<n> head <40-hex sha> engine <engine>` only when `runtimes.reviewer` is `session` (intent 0013 S11, D5): the engine of the wave that requests the review. An `engine` field under a fixed `runtimes.reviewer`, a missing `engine` under `session`, extra words or any other shape is refused before any command runs.
- The adapter passes the engine to the scripts as an argument, never as environment; the scripts check it again against the integration branch and refuse with `PREFLIGHT-FAILED <reason>`.
- The review runs the installed scripts of the user-level `gbai-reviewer` skill, never a copy from the repository under review (D20).
- One verdict per pushed head (R26): a head that moved or already has a verdict is skipped.
- The agent relaying the review never reviews, edits, comments, approves or merges itself, and never posts the reviewer marker.

## Operations

### trigger review

Start one review run for a PR number pinned to its head SHA, on the engine of the optional `engine` field under `session`, else on the fixed `runtimes.reviewer`.

### report placement

State where the run happened (for example `local`) and in which mode (`single-user` or `isolated`).

### return verdict

Return the `VERDICT` line unchanged and where the full verdict lives (the reviewer's PR comment). The line has the form `VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.

## Output Contract

An adapter answers every operation with the exact command and the location of its output. An agent using it returns the placement, the mode and the `VERDICT` line verbatim, preceded by the preflight failure reason when the scripts refused to run.

Example adapter: `examples/reviewer-local.md` (local placement, single-user mode).
