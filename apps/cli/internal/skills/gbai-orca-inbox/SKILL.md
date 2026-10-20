---
name: gbai-orca-inbox
description: "Trigger: orca inbox, mailbox, agent messaging, verdict message, notify coder, send message between sessions. Messaging protocol between gbai sessions via Orca."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use whenever a gbai session sends or receives a message through Orca.

## Hard Rules

- Address durable mailboxes only: `run:<id>` or `dispatch:<id>`. Never address terminal handles; they change on restart.
- A message is a notification only. It never decides anything: re-read the source of truth (GitHub PR, comment, status) before acting.
- Never put secrets, tokens or diffs in a message.
- Do not poll the inbox; wait for the next message event.

## Who Sends What

| From | To | Message |
|---|---|---|
| coder (via `gbai review request --notify`) | reviewer terminal | starts the review of a pinned head |
| reviewer terminal (`gbai review notify`) | coder mailbox | the summary line |
| coordinator | coder sessions | the issue to work on |
| coder | coordinator | `HANDED`, `DONE` or `BLOCKED` with PR number |

## Summary Line

Exactly: `PR #<n> head <40-hex sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`. Ignore a summary whose SHA is not your latest pushed head.
