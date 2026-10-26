---
name: gbai-reviewer-notes
description: "Trigger: how the reviewer works, review verdict, VERDICT line, reviewer marker comment, why blocked, merge gate. Explains the gbai reviewer side for other agents."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Use when an agent needs to understand or explain a review result. Agents never run or influence the review themselves; they only ask the `gbai-reviewer` subagent for one.

## Facts

- The `gbai-reviewer` subagent only validates `PR #<n> head <40-hex sha>`, runs the scripts of the user-level `gbai-reviewer` skill (`~/.claude/skills/gbai-reviewer/scripts/`) and relays their final `VERDICT` line (D20). It never reviews, edits, comments or merges itself.
- The scripts live outside every repository, so PR code cannot modify the code that reviews it. They refuse to run when they are not under the user-level skills dir or are writable by others.
- Single-user mode: the scripts run as the owner with the owner's `gh` login; no extra users.
- The scripts work in their own state dir (`~/.local/state/gbai/reviewer`) with their own clone; never edit or build inside it.
- They re-read the PR and skip a head that moved; they test-merge the pinned head into a fresh `dev`. A merge conflict is blocking.
- Checks (`review.checks`) and the review model run in a credential-free copy of the merged tree; the model runs read-only. This is trust-based, not an OS-user boundary.
- The verdict comes from local evidence (checks, conflict, findings), never from an agent's message. One verdict per pushed head (R26).
- The verdict comment carries the marker `<!-- gbai-reviewer-bot sha=<head> -->` and the footer `🤖 gbai reviewer`. It never approves: GitHub forbids approving your own PR.
- Merge gate: the newest marker comment by the owner's login must sign the current head and be clean; merges go only into `dev`, pinned to the reviewed SHA, never `main`, never PRs touching `review.human_only_paths`.

## Hard Rules

- Do not argue with a verdict; fix the findings and push a new head.
- Read the marker comment on the PR for findings; the `VERDICT` line only states the outcome.
- Never post a comment, PR body or review containing the marker, even quoted; a forged marker could sign a head the reviewer never checked.
