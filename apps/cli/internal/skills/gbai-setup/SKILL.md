---
name: gbai-setup
description: "Trigger: gbai setup, first setup, configure cloud agents, provision server, onboard gbai. Guides first setup through the gbai MCP tools, step by step."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use when the human wants to set up gbai for the first time or repair a broken setup. Drive every step through the `gbai` MCP tools; run nothing on the server by hand.

## Hard Rules

- NEVER ask the human to paste a secret (token, password, API key) into the chat, and never accept one if offered. Secrets enter only through `gbai secrets set <name>` in the human's own terminal.
- Plan first: call every `*_plan` tool (or an apply tool without `confirm`) and show the plan before applying with `confirm: true`.
- Ask before each apply; one question at a time.
- Stop on a failing step; report the tool error verbatim and the fix it suggests.

## Execution Steps

1. `doctor`: fix every `fail` locally first (railway login, gh auth as the repository owner).
2. `provision_plan` → show → `provision_apply` with `confirm: true`. Re-run `doctor`.
3. Cloud browser logins, done by the human in the server's Orca browser (local Orca paired to the server environment):
   - Gmail for the agent mailbox.
   - GitHub: the **owner** account, once, for the `orca` user. Single-user mode: the coder sessions and the reviewer share it; the reviewer never approves, it signs a verdict comment.
   - Railway.
4. `secret_instructions`: show each returned command and ask the human to run it in their own terminal. Wait for "done"; never ask for the value.
5. `repo_setup_plan` → show → `repo_setup_apply` with `confirm: true`.
6. `sessions_start` (mode `cloud` unless the human says local), then `status` to confirm the sessions are alive.

## Output Contract

End with a checklist of the six steps marked done, skipped (with reason) or failed (with the tool error).
