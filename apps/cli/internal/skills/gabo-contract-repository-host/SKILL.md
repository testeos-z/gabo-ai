---
name: gabo-contract-repository-host
description: "Trigger: repository host contract, repository_host slot, PR adapter, adapter coverage, write a host adapter. Operations every gbai repository host adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `repository_host` slot (`slots.repository_host.adapter`, TRD 9.1). This contract states WHAT agents must be able to do with branches and pull requests; the adapter states HOW with the user's host (D2). gbai ships no host code.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- Pull requests target `repo.integration_branch`. Never open a PR to, merge into or push to `main` (D13).
- Never force-push. Delete only merged branches the workflow created.
- Re-read after every write: the PR must show the expected state.
- Never post the reviewer marker (`reviewer.signing.marker`) in a PR body, comment or review.

## Operations

### open PR

Open a pull request from the task branch into `repo.integration_branch` with a title, a body that references the task, and the verification run. Return the PR number.

### comment on PR

Post one comment on a pull request.

### read PR status and checks

Read state, base branch, head SHA, mergeability, check results and comments of a pull request.

### merge into integration branch

Merge a pull request into `repo.integration_branch` only, pinned to the head SHA that received a clean verdict; refuse any other base. Return the merged SHA.

### delete branch

Delete a merged task branch, without force, only when the PR merged into the integration branch from that branch. Delete the local copy first, while the host copy still exists to check it against, then the host copy only if it still points at the merged head. Any refusal is reported as a leftover, never forced.

## Output Contract

An adapter answers every operation with the exact commands or tool calls and how to read the result back. An agent using it returns PR number, base, head SHA and state, the merged SHA for merges, and any refused operation with its reason.

Example adapter: `examples/host-github.md` (GitHub through `gh` and `git`).
