# memory-engram: example adapter for the gabo memory contract

Fills the `memory` slot with the Engram MCP tools (`mem_save`, `mem_update`, `mem_search`, `mem_get_observation`). Copy it to `.gabo/adapters/memory-engram.md` and point `slots.memory.adapter` at it.

Notes:

- Records live under fixed topic keys so later agents find them. Update an existing topic with `mem_update` instead of saving a duplicate.
- Project scope (`scope: project`) for everything but personal preferences.
- Read back every save with `mem_get_observation` on the returned id.
- In the ledger and trace rows below `<run id>` is the wave's run id `<I####>-<YYYYMMDDTHHMMSSZ>`, as in the operations. The decision row keeps `<wave>`, the wave id (the intent id), so a resumed run finds the decisions of earlier runs; the planning record keeps the intent id too.

| record | topic key |
|---|---|
| decision | `gabo/decision/<wave>/<task or global>/<slug>` |
| resource ledger | `gabo/ledger/<run id>` |
| wave trace | `gabo/trace/<run id>` |
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

`mem_search` for `gabo/ledger/<run id>`; if found, `mem_get_observation`, append the new lines and `mem_update`; otherwise `mem_save` with `topic_key: gabo/ledger/<run id>`. Line format: `<kind> <name or path> creator=<agent> task=<T###> state=<live|removed|leftover>`.

### read ledger

`mem_search` for `gabo/ledger/<run id>`, then `mem_get_observation`.

### save wave trace

Same pattern as `save ledger` on `gabo/trace/<run id>`; the content is one line per task: `<T###> status=<s> pr=#<n> verdict=<v> questions=<n>`. The observation whose topic key is exactly `gabo/trace/<run id>` is the wave's trace; the planning topic `gabo/trace/<I####>/planning` is a separate record saved with the same operation, which a wave never reads or overwrites; its content is a separate format (the lines the assistant defines), not the one line per task above.

### read wave trace

`mem_search` for `gabo/trace/<run id>` (for the newest run of an intent, `gabo/trace/<I####>-`, asking for 20 hits, the tool's maximum, and consider only topic keys of the form `gabo/trace/<I####>-<8 digits>T<6 digits>Z` (`<8 digits>` the UTC date YYYYMMDD, `<6 digits>` the UTC time HHMMSS), compared case-insensitively because Engram lowercases topic keys; take the one that sorts last, the latest run: with the legacy key `gabo/trace/i0001-w8-t001` and the run keys `gabo/trace/i0001-20261113t145700z` and `gabo/trace/i0001-20261113t150111z` it takes `gabo/trace/i0001-20261113t150111z`, though the legacy key sorts last; when the page holds fewer than 20 hits and none is such a key, read `gabo/trace/<I####>` (a trace written before the run id); a full page of 20 hits with no run-id key is an incomplete lookup, never no run), then `mem_get_observation`. Select the observation whose topic key is exactly `gabo/trace/<run id>` (or the one the prefix lookup selected); the reader must treat the "latest" fallback as ignoring keys with a segment after the wave id, so that it never returns a `.../planning` record.

### save preference

`mem_search` for `user/preferences` in the chosen scope; increment the count of a repeated observation with `mem_update`, or add it with `mem_save` (`topic_key: user/preferences`, `scope: personal` for global).

### read preferences

`mem_search` for `user/preferences` in the personal and the project scope; open both with `mem_get_observation`.
