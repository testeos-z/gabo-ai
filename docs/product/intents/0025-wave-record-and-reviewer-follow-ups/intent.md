# Intent 0025 Wave record and reviewer follow-ups

| field | value |
|---|---|
| intent id | 0025-wave-record-and-reviewer-follow-ups |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-13 |

## Problem

Intents 0020, 0022 and 0024 left findings that no intent covers. Owner request, 2026-11-13: "quiero que sigan trabajando faltan muchos issues". Each finding is lettered (a) to (j) and quotes its evidence.

**From intent 0024 (mechanical wave steps), [#664](https://github.com/testeos-z/gabo-ai/issues/664), `docs/product/intents/0024-mechanical-wave-steps-and-follow-up-fixes/plan.md` ("Findings for the owner", 2 to 8), the wave report on #664 and the closing PR [#693](https://github.com/testeos-z/gabo-ai/pull/693) with PR [#694](https://github.com/testeos-z/gabo-ai/pull/694):**

- (a) **The wave id equals the intent id**, so a second wave of one intent appends to the first wave's stamps file and `wave-record.sh measures` mixes both.
- (b) **The reviewer's prune judges `pr-<n>` worktrees by the caller's repository.** 74 of the 361 real worktrees belong to PRs that do not exist in `gabo-ai`, and a number that exists in two repositories is judged by the caller's; the age rule is the only guard.
- (c) **The `saves` ledger covers only the report entry.** The start entry and the ledger entry have no `saves add`, so `saves unverified` cannot cover them (the part of PD15 that did not pass).
- (d) **The stamps for `pr-opened`, `pr-merged` and `wave-reported` are written late**, when the cycle writes them and not when they happened (same second); the wall time is right, the per-event times are not.
- (e) **`wave-record.sh caps` prints `enforced by the engine` for `max_coder_turns` from the engine name alone**, even when the built coder agent has no turn key (the script's table is `claude` and `opencode`).
- (f) **Sentences are duplicated in `gabo-cycle`.** Checked on `origin/dev` at `b16d4d2`: three sentences appear twice (the `session started` line saved with its stamp, the report entry holding the whole `cat` of the stamps file, and the start entry followed by `saves add` and `saves read`), not two.
- (g) **The real reviewer caches sit just under the limit**: 1.99G against the default `REVIEWER_CACHE_MAX=2G`, so the trim never triggers on real data; with a smaller limit the fresh `check-home` files and `cache/go-mod` keep the total over it and only a WARN says so.
- (h) **Codex `mem_save` with the topic key `gabo/trace/I0001` overwrote earlier probe observations** (the earlier trace and the ledger): a topic upsert replaces the previous wave's trace. Same cause as (a): the topic key is the intent id.
- (i) **The `gabo` on PATH, an older build with the same version number as the build under test, rejects `wave.max_coders`, `max_coder_turns` and `max_fix_rounds` in `gabo check`.** A probe environment fact, not a defect of `dev`; the version stamp cannot tell the two builds apart.

**From intent 0022 (measure roles), [#497](https://github.com/testeos-z/gabo-ai/issues/497), probe [#637](https://github.com/testeos-z/gabo-ai/issues/637), `docs/product/intents/0022-measure-roles/plan.md` ("Probe results", third run):**

- (j) **The planning record's `time` line carries the date only** (`time: 2026-11-12 (UTC).`), not the UTC time of day; intent 0024 fixed the proposals field of the same record but not the time. The assistant text still says only "and the UTC time".

**Checked and left out.** Each leftover of intents 0020 and 0022 was checked against `origin/dev`:

| source | leftover | verdict |
|---|---|---|
| I0020 findings 3, 4, 5, 6 | label, `gabo.yaml` rewrite, clean head at the limit, "continue or stop" | fixed by intent 0024 (S3, S5, S6; probe PASS on #677) |
| I0020 finding 7 | OpenCode lists caps not in effect | fixed by intent 0024 (S3: no key, no line) |
| I0020 finding 7 | Codex children fork the parent context and spend about twice the tokens; `claude -p` wrote `~/.config/gabo/config.yaml` | engine and probe-side facts, not a gabo defect; recorded in the spec as an open question, not a requirement |
| I0020 T5 | Pi cap probe not run (usage limit) | carried to the probe requirement, optional |
| I0022 follow-up items 1 to 6 | trace lines, wall time, read-back, proposals, memory adapter test, TRD 9.2 | fixed by intent 0024 (S2, S4, S7, S8, S9) |
| I0022 T5 | token reporting on Codex, OpenCode, Pi, and the `read wave trace` check after planning saves not run | carried to the probe requirement |
| I0022 T5 | role trial (d) not run | needs the owner to name an intent; carried as an open question |
| I0024 finding 9 | on PR #198 the reviewer skipped the merge and the assistant merged it itself | cause unknown; carried as an open question, no requirement |

**Common cause behind (a), (c), (d), (e) and (f).** The wave's bookkeeping is keyed or timed by what the assistant or the script knows at write time (the intent id, the engine name, the write clock) and not by what happened. Intent 0024 made the steps mechanical; these are the gaps its probe found in that mechanism.

## Who is affected

The owner, who reads the wave report and its `Measures` section and cannot trust a mixed wall time, a label the built agent does not honor or a "none unverified" line that skipped three saves; and whoever runs the reviewer on a machine, because it can remove or keep the wrong worktree and its caches stay near the disk limit. (a), (c) to (e) and (h) affect every wave on every engine; (b) and (g) every reviewed PR; (f), (i) and (j) every maintainer of the skills and every probe.

## Desired outcome

- Two waves of one intent keep separate stamps, traces and ledgers, and no save overwrites a previous wave; every memory write of a wave is in the save ledger; the per-event stamp times are the times the events happened.
- A cap label states what the built agent really does; `gabo-cycle` says each thing once; the planning record carries a full UTC time.
- The reviewer never judges a worktree by a repository it does not belong to, and its caches end a prune under the limit or the log lists their sizes.
- A probe runs the build under test, and says which build it ran.

## Constraints

- The owner's preference (2026-11-12): keep custom scripts and functions to a minimum and prefer engine-native features. No new script; an existing script is extended only when no native or skill-text fix exists.
- The CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt` (PRD R31; TRD D24); a new command is the owner's change, not an agent's.
- `gabo.yaml` and `.claude/` are human-only paths for agents; no requirement edits them.
- The reviewer deletes only what it created (PRD R23) and its state is disposable (TRD 9.4, reviewer state row).
- No secrets in any script or trace line (PRD R21).
- Same files as the open intent 0023 (#575: the cycle, the coder skill, the assistant) and as the closed 0024: order the plan tasks after 0023's where they edit the same text.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does the open intent 0023 (#575) cover any of (a) to (j)? | answered: no. Checked against its spec on `origin/dev`: 0023 covers Pi sessions and `--print`, worktree registration, the Pi guardrails file, status labels on closed tasks, the `Closes`/`Refs` keyword and merged remote branches. None of (a) to (j). Only the files overlap (spec section 5). |
| 2 | Is the intent id the right key for a wave's trace, ledger and stamps, or should each wave have a run id of its own? | decided by the owner's standing rule (2026-11-13): a wave run id for the trace, the ledger and the stamps (spec D1). |
| 3 | One intent for ten findings? | decided by the owner's standing rule (2026-11-13): one intent, as the owner asked for the follow-ups together; each finding keeps its own requirement (spec D2). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#695 I0025: Wave record and reviewer follow-ups](https://github.com/testeos-z/gabo-ai/issues/695) (sub-issue of #492) |
| sources | I0024 [#664](https://github.com/testeos-z/gabo-ai/issues/664), [#677](https://github.com/testeos-z/gabo-ai/issues/677), PR [#693](https://github.com/testeos-z/gabo-ai/pull/693), PR [#694](https://github.com/testeos-z/gabo-ai/pull/694); I0022 [#497](https://github.com/testeos-z/gabo-ai/issues/497), [#637](https://github.com/testeos-z/gabo-ai/issues/637); I0020 [#495](https://github.com/testeos-z/gabo-ai/issues/495), [#581](https://github.com/testeos-z/gabo-ai/issues/581) |
| PRD | [PRD](../../PRD.md), R13, R21, R23, R24, R25, R31 |
| TRD | [TRD](../../TRD.md), 5.1, 9.2, 9.4, D17, D24 |
| skill | `.gabo/skills/gabo-cycle/SKILL.md`, `.gabo/skills/gabo-assistant/SKILL.md`, `.gabo/skills/gabo-wave-report/SKILL.md`, `.gabo/skills/gabo-reviewer/SKILL.md` |
| code | `apps/cli/internal/skills/gabo-reviewer/scripts/`, `apps/cli/internal/skills/gabo-wave-report/scripts/wave-record.sh` |
| spec | [spec](spec.md) |
| related intent | 0020-spend-or-turn-cap-for-auto-mode; 0022-measure-roles; 0024-mechanical-wave-steps-and-follow-up-fixes; 0023-unattended-pi-waves-and-flow-hygiene (open, same skill files, no overlap in findings) |
