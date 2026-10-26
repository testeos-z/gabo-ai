---
name: gbai-memory-engram
description: "Trigger: gbai memory, record decision, auto decision, resource ledger, wave trace, user preferences. Engram adapter for the gbai memory slot."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "0.1"
---
## Activation Contract

Use whenever a gbai agent records or reads decisions, the per-wave resource ledger, wave traces or preference observations. Fills the `memory` slot contract (TRD 9.1) with the Engram MCP tools (`mem_save`, `mem_update`, `mem_search`, `mem_get_observation`).

## Hard Rules

- Never store secrets, tokens, credentials or sensitive personal data (R21).
- Project scope for everything about this repository; personal scope only for personal preferences.
- Use the fixed topic keys below so later agents find the records; update an existing topic with `mem_update` instead of duplicating it.
- Save before you report; a record that failed to save is reported as `unverified`.

## Records

| record | topic key | scope | content |
|---|---|---|---|
| Decision | `gbai/decision/<wave>/<task or "global">/<slug>` | project | what, question, choice, why, who (owner, assistant, directiva), mode (manual/auto), wave, task (R6, R30) |
| Resource ledger | `gbai/ledger/<wave>` | project | one line per resource the workflow created: kind (worktree, branch, container, test db), name or path, creator agent, task, state (live, removed, leftover) (D15) |
| Wave trace | `gbai/trace/<wave>` | project | wave tasks with status, PR, verdict, open questions; enough to resume (D17) |
| Preference | `user/preferences` | personal (global) or project | observation, evidence, count; at about 3 repeats propose it for `~/.config/gbai/config.yaml` (D22) |

## Operations

| contract op | how |
|---|---|
| save decision | `mem_save` with the decision topic, type `decision` |
| list decisions | `mem_search` with `gbai/decision/<wave>` (or `/<task>`) and a mode word; open hits with `mem_get_observation` |
| search / read one | `mem_search`, then `mem_get_observation` for the full text |
| append to ledger | read `gbai/ledger/<wave>`, add the lines, `mem_update` (create with `mem_save` the first time) |
| save / read trace | same pattern on `gbai/trace/<wave>` |
| preferences | read `user/preferences` at session start; increment the count of a repeated observation |

## Output Contract

Return the topic keys written and whether each save succeeded.
