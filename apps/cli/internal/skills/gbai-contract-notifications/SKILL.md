---
name: gbai-contract-notifications
description: "Trigger: notifications contract, notifications slot, notify owner, notify developer, alias, adapter coverage. Operations every gbai notifications adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `notifications` slot (`slots.notifications.adapter`, TRD 9.1). This contract states WHO agents must be able to reach; the adapter states HOW with the team's own channel (D2, D4). gbai ships no messaging code.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: the planned coverage check in `gbai generate` (T011) compares them; today only the skills test enforces them, for the bundled example.
- A developer is addressed by alias, resolved through `developers` in `gbai.yaml` (git email to alias, D10); an alias that resolves to no developer is reported, never guessed (R16).
- A notification never carries secrets (R21) and never the reviewer marker.
- A notification is one short message with a link to where the details live (issue, PR, wave report); it is not a question channel that blocks the agent.
- Every sent notification is read back where the channel allows it; otherwise it is reported as unverified.

## Operations

### notify owner

Send one message to the project owner. Return where it was delivered.

### notify developer by alias

Resolve the alias to the developer's identity on the channel and send one message. Return where it was delivered, or `unknown alias`.

## Output Contract

An adapter answers every operation with the exact command, how the recipient is addressed and how to read the message back. An agent using it returns each notification as `verified`, `unverified` or `unknown alias`.

Example adapter: `examples/notify-task-comment.md` (a mention in a task comment).
