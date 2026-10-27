---
name: gbai-contract-methodology
description: "Trigger: methodology contract, methodology slot, gbai-methodology, how coders build, coder-side review, adapter coverage. Operations every gbai methodology adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `methodology` slot (`slots.methodology.adapter`, TRD 9.1). Every coder loads the adapter under the fixed name `gbai-methodology`. This contract states WHAT a methodology must tell the coder; the adapter states HOW the team builds (D2, D5). Recommended, not required: gentle-ai ODD with RDD as the coder-side review layer.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: the planned coverage check in `gbai generate` (T011) compares them; today only the skills test enforces them, for the bundled example.
- The adapter must declare coder-side review explicitly: either `none`, or the tool (for example GGA or RDD), when it runs and what the coder does with its result. Silence is not a declaration.
- Coder-side review never replaces the gbai reviewer, which stays the only merge gate (R9); a coder never merges or approves.
- The methodology never weakens the coder's hard rules: clean worktree, no force-push, PR to `repo.integration_branch`, at most 4 reviewer fix rounds.

## Operations

### coder instructions

How the coder builds one task: what it reads first, the test-first rule or its exception, how it commits work units, and which checks it runs before pushing (at least `review.checks`).

### coder-side review

Whether a review runs on the coder's side before the push, and if so which tool, when, how it is invoked and how its outcome changes the coder's next step. State `none` when there is none.

### report per task

What the coder adds to its outcome for the methodology: test evidence or the stated exception, commits, checks run with results, and the coder-side review outcome.

## Output Contract

An adapter answers every operation with concrete steps and commands. A coder using it returns the per-task report from `report per task` inside its normal outcome.

Example adapter: `examples/methodology-tdd.md` (plain test-driven development, no coder-side review tool).
