---
name: gbai-orca-inbox
description: "Trigger: orca inbox, mailbox, agent messaging between gbai agent sessions. Deprecated: gbai agents talk through native subagent returns; scheduled for removal."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Deprecated. Do not use for new work; this skill is kept only until it is removed from the embedded set.

## Replacement

- Coder ↔ cycle ↔ assistant: native subagent launches and their returned outcomes (TRD D1).
- Review requests: the coder asks the `gbai-reviewer` subagent with `PR #<n> head <sha>` and receives the `VERDICT` line (see `gbai-reviewer-notes`).
- Direct sessions with the owner belong to the `direct_sessions` slot adapter, not to this skill.
