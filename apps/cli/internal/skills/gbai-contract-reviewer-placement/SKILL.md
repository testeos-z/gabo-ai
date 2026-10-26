---
name: gbai-contract-reviewer-placement
description: "Trigger: reviewer placement contract, reviewer_placement slot, where the reviewer runs, VERDICT line, adapter coverage. Operations every gbai reviewer placement adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `reviewer_placement` slot (`slots.reviewer_placement.adapter`, `reviewer.placement`, TRD 9.1). This contract states WHAT must happen to get one review of a pushed PR head; the adapter states WHERE and HOW the scripts of the user-level `gbai-reviewer` skill run (D6, D20). Only placement `local` is implemented in the first release.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: the coverage check in `gbai generate` compares them.
- A review request is exactly `PR #<n> head <40-hex sha>`; anything else is refused before any command runs.
- The review runs the installed scripts of the user-level `gbai-reviewer` skill, never a copy from the repository under review (D20).
- One verdict per pushed head (R26): a head that moved or already has a verdict is skipped.
- The agent relaying the review never reviews, edits, comments, approves or merges itself, and never posts the reviewer marker.

## Operations

### trigger review

Start one review run for a PR number pinned to its head SHA.

### report placement

State where the run happened (for example `local`) and in which mode (`single-user` or `isolated`).

### return verdict

Return the `VERDICT` line unchanged and where the full verdict lives (the reviewer's PR comment). The line has the form `VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.

## Output Contract

An adapter answers every operation with the exact command and the location of its output. An agent using it returns the placement, the mode and the `VERDICT` line verbatim, preceded by the preflight failure reason when the scripts refused to run.

Example adapter: `examples/reviewer-local.md` (local placement, single-user mode).
