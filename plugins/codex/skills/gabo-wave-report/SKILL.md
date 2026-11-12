---
name: gabo-wave-report
description: "Trigger: gabo wave report, build wave report, save wave report, wave summary, wave close report. Formats the TRD 9.2 wave report and its summary and saves it through the memory slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.2"
---
## Activation Contract

Load in the main session when it runs the `gabo-cycle` skill and builds the wave report (J3) and in the assistant when it presents the summary at Wave Close. This skill owns the TRD 9.2 wave report format; the cycle and assistant skills reference it instead of repeating it. It uses only the `memory` slot adapter.

## Hard Rules

- Every section below appears, in this order. A section with nothing to report says `none`; never drop it.
- Report only what the outcomes, the tracker, the host and the `memory` slot show. Never guess a status, verdict or answer; a missing value is `unknown` with the source that lacked it.
- Never include secrets, tokens, credentials or sensitive personal data (R21).
- Save it through `memory` (`save wave trace`) as the last entry of `gabo/trace/<wave>`, then read it back (`read wave trace`). A save that does not show up after one retry is listed under Unverified writes in the report you return, never claimed as saved.

## Report Format (TRD 9.2)

1. **Wave**: wave id, mode, start time from `gabo/trace/<wave>`, and the line `session engine: <engine>`, the engine the wave ran on (intent 0013 S7; `unknown (rebuild the agents)` when the main-session entry has no engine line). When `runtimes.coder` is a fixed engine other than the session engine, also the line `coders ran on <session engine> (runtimes.coder is <engine>; subagents inherit the session engine)`. A wave that stopped at a launch cap (intent 0020) adds the paused line `status: paused`: the cap that fired (`max_coders`, `max_fix_rounds` or `max_coder_turns`), the affected task ids and the question "continue or stop". With any cap set, one line per cap in effect with its enforcement: `enforced by the engine` (`max_coder_turns` on Claude Code and OpenCode, only when the built `gabo-coder` agent file carries the engine key `maxTurns` or `steps`; read it, and when it does not, say `advisory`), `enforced by the cycle` (`max_coders` and `max_fix_rounds`) or `advisory` (`max_coder_turns` on Codex and Pi), and the line `spend is not capped by gabo`. No section is added or reordered.
2. **Tasks**: one line per task: task id, status (DONE, HANDED, BLOCKED), PR, last verdict, fix rounds, coder handle. A BLOCKED task adds its reason.
3. **Questions**: each question raised with its answer and resolver (R5), and whether it was a manual answer or an auto decision record.
4. **Interactions**: interaction count with the reasons for each interaction with the owner (R29).
5. **Unverified writes**: every tracker, host or memory write whose re-read did not show it (R24).
6. **Resume disagreements**: the resume disagreements the cycle left alone, where tracker, host and trace differ (D17).
7. **Leftovers**: REFUSED removals and leftovers from coder task closes and the wave-close sweep: paths, branches, containers, test databases; merged tasks still open because no coder could close them (R13).
8. **Plan-change proposals**: the plan-change proposals of the wave, each with issue or PR, scope, affected tasks and outcome.
9. **Denied board moves**: denied board moves as the exact list (`issue -> Ready`, `issue -> Backlog`) for the assistant to apply.
10. **Comfort answer**: the owner's comfort answer to apply next wave (R17). The cycle writes `pending`; the assistant fills it at Wave Close and saves the report again with `save wave trace`, read back.

## Summary (UX 1.5)

The assistant shows the owner this summary, not the full report:

- `Wave <n>: <done> done, <blocked> blocked`
- questions with answer and resolver; decisions taken in auto mode (`list decisions` filtered by `mode: auto` and the wave);
- unverified writes; cleanup leftovers; interaction count;
- where the full report lives: the last entry of `gabo/trace/<wave>`.

Escalations use `<task>: BLOCKED — <reason>` · what was tried · the one decision needed.

## Output Contract

Return the full report in the format above, the `gabo/trace/<wave>` key written, and the save as `verified` or `unverified`.
