# Spec 0020 Spend or turn cap for auto mode

| field | value |
|---|---|
| intent | [0020-spend-or-turn-cap-for-auto-mode](intent.md) |
| status | Accepted (owner, 2026-11-12) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`. Owner's standing rule (2026-11-11): the agent applies its recommended answer to every open question and records it as a decision; only merging to main, tags and secrets stay the owner's. Each decision below is marked **Recorded decision (resolver: agent, rule of the owner)** and is saved through the `memory` slot. The source is the owner's approval of this intent on 2026-11-11 ("si" to creating an intent for each pending idea) and the evidence below; on which intent gets its spec next: "vamos en orden numerico".

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`, and the engines' current documentation):

- **The idea.** Anthropic, Best practices for Claude Code (https://code.claude.com/docs/en/best-practices, the paragraph on unattended runs): "Before a run goes unattended, make sure it knows what done looks like, can check its own work, and can only touch what it should. Give it a cap on turns or spend, and have a person approve the result at the end."
- **Today's bounds.** `.gabo/skills/gabo-cycle/SKILL.md`: `parallelism` bounds the coders in parallel; rule 3: a coder BLOCKED after 4 fix rounds leaves its task BLOCKED and "Never launch a fifth round". `.gabo/skills/gabo-coder/SKILL.md`: "Max 4 fix rounds". Nothing bounds the turns a coder takes, the coders launched in one wave or the spend.
- **Config today.** `schemas/gabo.project.schema.json` line 189: `wave` has the one key `after`, with `additionalProperties: false` (the copy in `apps/cli/internal/config/schemas/` is byte-equal). `apps/cli/internal/config/projectkeys.go` lines 90-93: `type Wave struct { After string }`, validated at line 157. Files are decoded strictly (TRD 9.3), so a new `wave` key needs a loader and schema change.
- **Pausing and resuming.** `gabo-cycle` receives `resume <session id>`, rereads `gabo/trace/<wave>` and `gabo/ledger/<wave>`, and recomputes the wave from tracker state, ledger and trace (steps 2 and 3); the trace holds each task's status, PR, verdict, fix rounds and open questions (D17). PRD R15: during an auto-mode run the human receives no prompts, only the final answer. PRD R29: every wave produces a report.
- **Built agents.** `gabo-runtime-claude` and the agent templates fill only the named slots and "invent nothing: no frontmatter key, tool, skill, model or rule the spec and template do not give".

What each engine documents (sources fetched 2026-11-11):

| engine | per-coder turn cap | what happens at the cap | spend |
|---|---|---|---|
| Claude Code | `maxTurns` in the subagent frontmatter: "Maximum number of agentic turns before the subagent stops" ([sub-agents](https://code.claude.com/docs/en/sub-agents)) | the subagent's output comes back marked partial (needs Claude Code v2.1.246 or later) and Claude can resume it | `--max-budget-usd`, "print mode only", an estimate, subagent spend counts toward it, spend can pass it; `--max-turns` is also print mode only ([CLI reference](https://code.claude.com/docs/en/cli-reference)). No spend cap is documented for an interactive session |
| OpenCode | `steps` in the agent config: a positive number of iterations ([agents](https://opencode.ai/docs/agents/)) | on the last step OpenCode "removes tools and asks the model to summarize in text"; a new user message resets the allowance | no cost limit documented on that page |
| Codex | none documented for a custom agent (its fields: `name`, `description`, `developer_instructions`, `model`, reasoning and verbosity); `agents.max_concurrent_threads_per_session` caps open spawned-agent threads, which is concurrency, not turns ([subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents.md)) | not applicable | none documented |
| Pi with gentle-pi | none found in the gentle-pi documents read (`docs/readme-reference.md` and `docs/gentle-shell.md`, the first 100,000 characters of each): "Subagents have no automatic total execution timeout"; `stall_timeout_ms` is a silence watchdog, not a cap. Third-party Pi subagent packages document a `max_turns`, but gentle-pi is the recommended package. Unconfirmed: the probe did not run Pi (`not run: openai subscription usage limit`, [#581](https://github.com/testeos-z/gabo-ai/issues/581)) | not applicable | none documented |

Summary of the table: a per-coder turn cap is enforceable by the engine on Claude Code and OpenCode and advisory on Codex and Pi; spend is not capped from inside a wave on any of the four. The reading of the Pi documents is partial, so "none" for Pi means none found in the parts read; the probe (S9) did not run Pi (`not run: openai subscription usage limit`), so the Pi row stays unconfirmed. The probe observed no difference in the Claude Code, OpenCode and Codex rows.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | `gabo.yaml` shall accept three optional integer keys under `wave` (team layer): `wave.max_coder_turns` (at least 1), `wave.max_coders` (at least 1, the most coders launched in one wave in total, a resume not counting) and `wave.max_fix_rounds` (1 to 4). With none set, behavior is today's: no turn cap, no coder limit, and fix rounds at 4. | Given `wave.max_coders: 0`, Then `gabo check` fails on that key in the existing failure shape and exit 1. Given `wave.max_fix_rounds: 5`, Then the same. Given none of the keys, Then the output of `gabo check` and every golden file is byte-identical to today's. Given the two schema copies, Then they are byte-equal. | Go unit tests in `apps/cli/internal/config` (valid, invalid, unset); schema validation test; a test that the two schema copies are equal; golden output |
| S2 | When `wave.max_coder_turns` is set, the runtime skill of an engine that documents a turn cap shall write that engine's key into the built `gabo-coder` agent (`maxTurns` for Claude Code, `steps` for OpenCode) and omit it when the key is unset. For Codex and Pi, which document none, the cycle shall pass the limit in the coder's launch input and the coder shall stop and report at it, as an advisory cap. | Given `wave.max_coder_turns: 30` and Claude Code, Then `.claude/agents/gabo-coder.md` has `maxTurns: 30`. Given OpenCode, Then its coder agent has `steps: 30`. Given the key unset, Then neither key is written. Given Codex or Pi, Then the built coder has no engine key and the cycle's launch input carries the limit. | skill review; probe (S9) on Claude Code, with the built agent read back |
| S3 | The cycle shall launch no more than `wave.max_coders` coders in a wave, and shall pause (S4) before a launch that would exceed it. It shall pause when a task reaches `wave.max_fix_rounds` rounds without a clean review, when that value is below 4; at 4 rounds rule 3 applies as today (BLOCKED, never a fifth round). | Given `wave.max_coders: 2` and three unblocked tasks, Then two coders launch, the third does not, and the wave pauses naming `max_coders`. Given `wave.max_fix_rounds: 2` and a coder needing a third round, Then the wave pauses naming `max_fix_rounds`, and the task is not BLOCKED. Given `wave.max_fix_rounds` unset, Then rule 3 at 4 rounds is unchanged. | skill review; probe (S9) |
| S4 | When a cap fires, the wave shall pause, not abort: the cycle launches nothing new, lets the coders already running return, and returns a wave report marked paused that names the cap that fired, the tasks it affected, and asks the owner to continue or stop. No task is marked DONE or BLOCKED because of the pause, and the state stays in the tracker, the ledger and the trace. In auto mode this report is the final answer (R15). | Given a cap fires in auto mode, Then no prompt reaches the owner during the wave and the final answer is the paused report. Given the report, Then it carries the cap name, the task ids and the question "continue or stop". Given the tracker, ledger and trace after the pause, Then they hold every task's state. | skill review; the `gabo-wave-report` format carries the paused line; probe (S9) |
| S5 | A coder that the engine stopped at its turn cap shall not count as BLOCKED or DONE: a coder that returns partial output (Claude Code) or returns without an outcome status after a wrap-up summary (OpenCode) is treated as stopped at the turn cap, its task stays claimed with its ledger handle, and the wave pauses with the cap `max_coder_turns`. | Given a Claude Code coder that stops at `maxTurns`, Then the cycle reports the cap `max_coder_turns`, keeps the task claimed with its ledger handle, and does not write BLOCKED. Given an OpenCode coder that ends without an outcome status, Then the same. | skill review; probe (S9) on Claude Code |
| S6 | When the owner answers "continue", the assistant shall resume the wave through the existing `resume <session id>` rule, and the trace shall record that the counter of the cap that fired restarted, with its time. When the owner answers "stop", the wave closes through the normal close, with the unfinished tasks left as they are. | Given "continue", Then the cycle runs with `resume`, appends one trace line for the reset, and counts that cap from zero. Given "stop", Then the wave closes with the report and no new coder launches. | skill review; probe (S9) |
| S7 | The wave report shall state, for each cap in effect, whether it was enforced by the engine, enforced by the cycle or advisory, and shall state that spend is not capped by gabo. The caps shall never change `review.auto_merge`, never let auto mode merge anything more than it merges today, and never skip a review. | Given `wave.max_coder_turns` on Codex, Then the report says "max_coder_turns: advisory". Given Claude Code, Then "enforced by the engine". Given `wave.max_coders`, Then "enforced by the cycle". Given any run, Then the report has a line that spend is not capped by gabo. Given a capped wave, Then `review.auto_merge` and the merge behavior are unchanged. | skill review; review of the diff for `review.*` and merge text |
| S8 | The Go change shall be limited to the three keys: the `Wave` struct, its validation, both schema copies, and their tests; it adds no command. The key table in TRD 9.3, the setup questions of `gabo-setup`, `docs/` and the help text that lists keys shall name the keys and their defaults. | Given `go -C apps/cli build ./...`, `vet` and `test`, Then they pass with only the S1 tests added. Given the diff, Then no command is added. Given the docs, Then each key appears with "unset keeps today's behavior". | `go -C apps/cli build ./...`, `vet`, `test`; review of the diff and the docs |
| S9 | The change shall be probed on a disposable repository on Claude Code: a wave with `wave.max_coder_turns` low enough that a coder stops, `wave.max_coders: 1` with two tasks, and `wave.max_fix_rounds: 1`, each reaching the paused report, then a "continue" that resumes. The same probe on OpenCode, Codex and Pi shall record what each engine really did, and the table above is corrected where the probe differs. | Given each probe, Then the paused report names the right cap and a resume continues the wave. Given an engine whose result differs from the table, Then the table is corrected in the spec. | probe; results recorded in the intent's review findings; the plan assigns it to a task |

## 2. Design

Affected files: `apps/cli/internal/config` (the `Wave` struct, its validation, both schema copies, tests), the `gabo-cycle` and `gabo-coder` skills (S3 to S6), the `gabo-wave-report` format (the paused line), `gabo-agent-templates` (an optional turn-limit slot in the neutral spec and the coder template), the runtime skills of Claude Code and OpenCode (S2), `gabo-setup` (the questions), the TRD 9.3 key table and the docs.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | What is capped | (a) per-coder turns, coders launched per wave and fix rounds, and no spend key; (b) also a spend key; (c) only a wall-clock limit | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: turns, coders and rounds are counted by gabo or documented by the engine, while no engine lets gabo cap spend from inside a wave (Claude Code's `--max-budget-usd` is print mode only and is set by whoever starts the process); a key gabo cannot enforce would promise a limit it does not give. | The owner who wants a spend cap on an unattended Claude Code run starts it with `--max-budget-usd` (documented, an estimate). The wave report says spend is not capped by gabo (S7). |
| D2 | Honest enforcement per engine | (a) engine-enforced where the engine documents a cap, advisory elsewhere, and say which; (b) advisory everywhere; (c) refuse a cap on engines without one | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: it uses the real limit where it exists (Claude Code `maxTurns`, OpenCode `steps`) and never claims enforcement on Codex or Pi. | The report names the kind of enforcement (S7). A later engine release that adds a cap changes the table and the runtime skill, not the keys. |
| D3 | What happens at a cap | (a) pause: launch nothing new, let running coders return, report, ask continue or stop; (b) abort the wave; (c) stop only the coder that hit it | **Recorded decision (resolver: agent, rule of the owner):** (a), the assistant's recommendation. Reason: an abort loses live work, and state is already in the tracker, ledger and trace, so a pause costs nothing and resumes through the existing rule. | A paused wave is the final answer in auto mode (R15). Resume needs no new mechanism (S6). |
| D4 | Where the caps live and their default | (a) three optional keys under `wave`, team layer, unset keeps today's behavior; (b) a single `wave.cap` value; (c) in the user file | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: `wave` already holds the wave settings, the team agrees on how much an unattended wave may do, and unset must change nothing. | The Go loader and both schemas change (S8). The caps never touch `review.auto_merge` (S7). |
| D5 | Which agent gets the turn cap, and when | (a) only `gabo-coder`, written into its built agent at build time and passed in its launch input; (b) every agent; (c) the launch input only | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: the coder is the agent that loops (fix rounds); the cycle and the reviewer are bounded by their scripts and the wave; the engine key is the only enforcement some engines have. | Changing `wave.max_coder_turns` needs a rebuild of the agents, as any config change does (TRD risks table). The neutral spec and the coder template get one optional slot. |
| D6 | What "continue" means | (a) restart the counter of the cap that fired and record it in the trace; (b) lift the cap for the rest of the wave; (c) ask the owner for a new value | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: the owner chose the cap, so a continue grants one more allowance of the same size; lifting it would hide the next runaway. | One trace line per reset; "stop" uses the normal close. |

## 3. UX and app flow (optional)

Not applicable: no screen or component. The journey is the wave report: a paused wave ends with the report that names the cap and asks "continue or stop"; the owner answers in the conversation. Wording follows the existing wave report lines.

## 4. DB migration steps (optional)

Not applicable: no stored data changes. The new keys are optional, so an existing `gabo.yaml` stays valid.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R15 (in auto mode the human receives no prompts, only the final answer) | A cap asks the owner to continue or stop. | resolved by S4 and D3: the pause is the wave's final answer; nothing is asked during the wave. |
| `gabo-agent-templates` and the runtime skills ("invent nothing: no frontmatter key the spec and template do not give") | S2 writes `maxTurns` and `steps` into built agents. | resolved by D5: the neutral spec and the coder template get one optional slot, and the runtime skills map it to the engine key; nothing else is invented. A catalog convention change, decided with the spec's acceptance. |
| TRD 9.3 key table | It lists `wave.after` only. | amended with the spec's acceptance: three rows for the new keys. |
| `gabo-cycle` rule 3 and `gabo-coder` ("Max 4 fix rounds") | `wave.max_fix_rounds` could be read as a new limit. | none: 4 stays the ceiling; the key can only lower it (S1, S3). |
| `review.auto_merge` and the merge gate (PRD R9, D13) | A cap could be read as touching merges. | none: S7 states that caps never change merge behavior. |
| Go loader strictness (TRD 9.3: unknown keys fail) | New keys need Go. | resolved by S8: a Go change limited to the three keys, with no new command. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | What is capped: turns, spend, wall-clock time, or several? | answered: coder turns, coders launched and fix rounds; spend is not capped by gabo (D1). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Which engines report turns or spend to a running session? | answered by the table in the evidence base: Claude Code and OpenCode document a turn cap; Codex and Pi document none; none documents a spend cap from inside a wave (Claude Code's is print mode only). The Pi reading is partial; S9 confirms it. |
| 3 | At the cap, does the wave stop and report, or pause and ask the owner? | answered: pause, report and ask continue or stop (D3). Recorded decision (resolver: agent, rule of the owner). |
| 4 | Does "a person approves the result at the end" change `review.auto_merge` for capped waves? | answered: no; the caps never change `review.auto_merge` or let auto mode merge more (S7). The owner still decides the merge to main, which no wave does. Recorded decision (resolver: agent, rule of the owner). |

Nothing is left that needs the owner except the acceptance in section 7.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner, alias Leo) | 2026-11-12 | Accepted by the owner as the spec stands on dev, with its recorded decisions (resolver: agent, rule of the owner). Owner, verbatim: "los que ya tienes spec estan aprobados" (this sentence accepts the specs of intents 0020, 0021 and 0022). Relayed unedited by the assistant from the owner's own message in the session of 2026-11-12; the status row was set in PR #573. |
