---
name: gabo-memory-engram
description: "Trigger: gabo memory, record decision, auto decision, resource ledger, wave trace, user preferences. Engram adapter for the gabo memory slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Use whenever a gabo agent records or reads decisions, the per-wave resource ledger, wave traces or preference observations. Fills the `memory` slot contract (TRD 9.1) with the Engram MCP tools (`mem_save`, `mem_update`, `mem_search`, `mem_get_observation`).

## Hard Rules

- Never store secrets, tokens, credentials or sensitive personal data (R21).
- Project scope for everything about this repository; personal scope only for personal preferences.
- Use the fixed topic keys below so later agents find the records; update an existing topic with `mem_update` instead of duplicating it.
- Save before you report; a record that failed to save is reported as `unverified`.

## Records

| record | topic key | scope | content |
|---|---|---|---|
| Decision | `gabo/decision/<wave>/<task or "global">/<slug>` | project | what, question, choice, why, who (owner, assistant, directiva), mode (manual/auto), wave, task (R6, R30) |
| Resource ledger | `gabo/ledger/<wave>` | project | one line per resource the workflow created: kind (worktree, branch, container, test db), name or path, creator agent, task, state (live, removed, leftover) (D15) |
| Wave trace | `gabo/trace/<wave>` | project | wave tasks with status, PR, verdict, open questions; enough to resume (D17) |
| Planning record | `gabo/trace/<I####>/planning` | project | the lines the assistant saves after a planning step: agent, `roles: separate`, `gaps: <n> (<ids>)` or `gaps: none`, `proposals: <n>` with issue or PR and scope each or `proposals: none`, and the UTC time |
| Preference | `user/preferences` | personal (global) or project | observation, evidence, count; at about 3 repeats propose it for `~/.config/gabo/config.yaml` (D22) |

## Operations

Tool calls come from this adapter's former operation table and Records, and the `memory` contract with its Engram example adapter; parameter names (`title`, `content`, `type`, `topic_key`, `scope`; `mem_update` needs `id` and `expected_project`; `mem_search` takes `query`, optional `scope`, `type`) confirmed from the Engram MCP tool schemas on 2026-10-28. Every save is read back with `mem_get_observation` on the returned id; a record that does not show up after one retry is `unverified`.

### save decision

`mem_save` with `type: decision`, `topic_key: gabo/decision/<wave>/<task or "global">/<slug>`, `scope: project`, `title: <what>`, `content` lines `what:`, `question:`, `choice:`, `why:`, `who:`, `mode:`, `wave:`, `task:`. Return the observation id. Read back: `mem_get_observation` with that id.

### list decisions

`mem_search` with `query: gabo/decision/<wave>` (add `/<task>` to narrow) and `type: decision`, plus the mode word (`manual` or `auto`) when filtering by mode; open each hit with `mem_get_observation` and drop records whose fields do not match the filter.

### search

`mem_search` with the query; return the hit ids and titles.

### read one

`mem_search` for the topic key when only the key is known, then `mem_get_observation` with the id for the full text.

### save ledger

`mem_search` for `gabo/ledger/<wave>`; if found, `mem_get_observation`, add the lines and `mem_update` (`id`, `expected_project`, `content`); otherwise `mem_save` with `topic_key: gabo/ledger/<wave>`, `scope: project`. One line per resource: kind, name or path, creator agent, task, state (`live`, `removed`, `leftover`). Read back: `mem_get_observation` with the id.

### read ledger

`mem_search` for `gabo/ledger/<wave>`, then `mem_get_observation`.

### save wave trace

Same pattern as `save ledger` on `gabo/trace/<wave>`: wave tasks with status, PR, verdict, open questions. Select the observation whose topic key is exactly `gabo/trace/<wave>` for the update (the search is full text, so a hit with another key, such as the planning record, is never updated); if none matches, `mem_save`. That observation is the wave's trace; the planning topic `gabo/trace/<I####>/planning` is a separate record saved with the same operation, which a wave never reads or overwrites; its content is a separate format (the lines of the Planning record row), not the wave tasks above.

### read wave trace

`mem_search` for `gabo/trace/<wave>` (or `gabo/trace/` for the latest), then `mem_get_observation`. Select the observation whose topic key is exactly `gabo/trace/<wave>`; the "latest" fallback picks only the newest observation whose key has no segment after the wave id, so it never returns a `.../planning` record.

### save preference

`mem_search` for `user/preferences` in the chosen scope (`scope: personal` for global, `scope: project` otherwise); increment the count of a repeated observation with `mem_update`, or add it with `mem_save` (`topic_key: user/preferences`, same scope). Content: observation, evidence, count.

### read preferences

Read `user/preferences` at session start: `mem_search` for `user/preferences` with `scope: personal` and with `scope: project`; open both with `mem_get_observation`; project overrides global.

## Output Contract

Return the topic keys written and whether each save succeeded.
