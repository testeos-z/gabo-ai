# memory-engram: example adapter for the gabo memory contract

Fills the `memory` slot with the Engram MCP tools (`mem_save`, `mem_update`, `mem_search`, `mem_get_observation`). Copy it to `.gabo/adapters/memory-engram.md` and point `slots.memory.adapter` at it.

Notes:

- Records live under fixed topic keys so later agents find them. Update an existing topic with `mem_update` instead of saving a duplicate.
- Project scope (`scope: project`) for everything but personal preferences.
- Read back every save with `mem_get_observation` on the returned id.

| record | topic key |
|---|---|
| decision | `gabo/decision/<wave>/<task or global>/<slug>` |
| resource ledger | `gabo/ledger/<wave>` |
| wave trace | `gabo/trace/<wave>` |
| planning record | `gabo/trace/<I####>/planning` |
| preference | `user/preferences` |

## Operations

### save decision

`mem_save` with `type: decision`, `topic_key: gabo/decision/<wave>/<task>/<slug>`, title `<what>` and content lines `what:`, `question:`, `choice:`, `reason:`, `mode:`, `wave:`, `task:`, `resolver:`.

### list decisions

`mem_search` with query `gabo/decision/<wave>` (add `/<task>` to narrow) plus `mode: <manual|auto>` when filtering by mode; open each hit with `mem_get_observation` and drop records whose fields do not match the filter.

### search

`mem_search` with the query; return the hit ids and titles.

### read one

`mem_get_observation` with the id.

### save ledger

`mem_search` for `gabo/ledger/<wave>`; if found, `mem_get_observation`, append the new lines and `mem_update`; otherwise `mem_save` with `topic_key: gabo/ledger/<wave>`. Line format: `<kind> <name or path> creator=<agent> task=<T###> state=<live|removed|leftover>`.

### read ledger

`mem_search` for `gabo/ledger/<wave>`, then `mem_get_observation`.

### save wave trace

Same pattern as `save ledger` on `gabo/trace/<wave>`; the content is one line per task: `<T###> status=<s> pr=#<n> verdict=<v> questions=<n>`. The observation whose topic key is exactly `gabo/trace/<wave>` is the wave's trace; the planning topic `gabo/trace/<I####>/planning` is a separate record saved with the same operation, which a wave never reads or overwrites.

### read wave trace

`mem_search` for `gabo/trace/<wave>` (or `gabo/trace/` for the latest), then `mem_get_observation`. Select the observation whose topic key is exactly `gabo/trace/<wave>`; the "latest" fallback ignores keys with a segment after the wave id, so it never returns a `.../planning` record.

### save preference

`mem_search` for `user/preferences` in the chosen scope; increment the count of a repeated observation with `mem_update`, or add it with `mem_save` (`topic_key: user/preferences`, `scope: personal` for global).

### read preferences

`mem_search` for `user/preferences` in the personal and the project scope; open both with `mem_get_observation`.
