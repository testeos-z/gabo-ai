# Probe 0028: hooks in coder subagents (S1)

| field | value |
|---|---|
| intent | [0028-deterministic-hooks-as-locks](intent.md) |
| spec | [spec S1](spec.md) |
| task | I0028/T1, issue #802 |
| run | 2026-11-14, one machine, every engine headless, default model of each engine except where a run names one |

## Reading

No person has read this record. The assistant (session claude-c9bdd54d, wave I0028-20261114T183645Z) read it under the owner's standing orders of 2026-11-14, quoted verbatim:

- "si doy permiso quiero que cierren todos los issues pronto"
- "habla con gabo-b para que paralelicen todo lo posible y aprueben solos lo necesario"

Those orders are the assistant's basis for that read. They are not recorded here as a waiver of S1's human read. Spec S1 and plan 0028 still require a person's read before the remaining tasks start, unless the owner waives it in writing. T2 has not started.

## Record

Caveat on the pi row: its (a) and (b) are `not testable`, because the only run was an emulated child process, not a delegated gentle-agents subagent (see the pi notes below). Codex (d) rests on the bypass run alone (see the codex notes).

Items: (a) a pre-command hook fires for a shell call inside a coder subagent and can deny it; (b) the subagent's tool calls reach the hook with an agent or session id; (c) the hand-off event exists and can block; (d) the engine's yes/no permission prompt appears when the session runs with permissions bypassed.

| engine | version | (a) | (b) | (c) | (d) |
|---|---|---|---|---|---|
| claude | 2.1.302 (Claude Code) | yes | yes | yes | no |
| codex | codex-cli 0.167.2 | yes | yes | yes | no |
| pi | 1.5.2 | not testable: the delegated gentle-agents runner was not driven; only an emulated child process ran, so no subagent call was observed | not testable: no call inside a delegated subagent was observed; the session id was seen only on the emulated child's call | not testable: no delegated hand-off was observed; the installed typings suggest the handler cannot block it, but that was not run | no |
| opencode | opencode v2.2.6 | yes | yes | yes | no |

Consequence for the plan: item (a) is `yes` for claude, codex and opencode in a delegated run, and `not testable` for pi. No engine has item (a) `no` on this record, so S14 is not triggered by it. Pi's (a), (b) and (c) are `not testable`, so the Pi relay (T12) rests on unobserved assumptions: T12 and T16 must observe a delegated pi run, hand-off included, before relying on them. Codex (d) rests on the bypass run alone.

## How each cell was observed

Every run used a `mktemp -d` scratch project and hook scripts in a second `mktemp -d` directory. Nothing was written to the worktree or to a user config.

### claude (2.1.302)

- Scratch project with `.claude/settings.json`: a `PreToolUse` command hook on `Bash` that logs its stdin and returns `permissionDecision: deny` when the command contains `probe-deny`, and a `SubagentStop` command hook that returns `decision: block` once.
- Command: `claude -p "<launch one general-purpose subagent that runs echo probe-deny-1 with Bash>" --permission-mode bypassPermissions --output-format stream-json --verbose`.
- Seen: the `PreToolUse` event fired for the subagent's Bash call with `agent_id` and `agent_type` in its stdin JSON. The subagent reported `PreToolUse:Bash hook error: probe: denied by pre-command hook`, so the command did not run. `SubagentStop` fired, was blocked once, and fired again with `stop_hook_active: true`, which shows the block was honoured.
- (d): under `bypassPermissions`, `echo probe-ok > <file>` ran, the file was written, and `permission_denials` was empty. The same write under `default` mode was not written and listed one `Bash` permission denial, which is the request that a prompt would have shown.

### codex (0.167.2)

- Hooks come from `hooks` in the session config (`-c`), since project-local hooks are disabled until the project is trusted. Without `--dangerously-bypass-hook-trust` the hooks did not run at all (first two attempts: no hook log, and the subagent's `echo` succeeded). The flag runs enabled hooks for that invocation only; the probe used it on a scratch project. This is a condition for every codex lock, recorded for T9 and T10.
- Command: `codex exec --dangerously-bypass-approvals-and-sandbox --dangerously-bypass-hook-trust --enable multi_agent --json "<spawn one subagent that runs echo probe-deny-1>"`, with a `PreToolUse` hook (no matcher, filtered in the script to `Bash`, `shell` and `exec_command`) and a `SubagentStop` hook.
- Seen: the `PreToolUse` event fired for the subagent's `Bash` call with `agent_id` and `agent_type`; the call was blocked with `Command blocked by PreToolUse hook: probe: denied by pre-command hook`. `SubagentStop` fired and was blocked once, then fired with `stop_hook_active: true`.
- (d): with `--dangerously-bypass-approvals-and-sandbox` the write ran and no `PermissionRequest` hook event fired. The default-sandbox contrast run could not write anything, because bubblewrap failed on this machine (`bwrap: Failed to make / slave`), so it is no positive control for a prompt. The `no` rests on the bypass run alone.

### pi (1.5.2)

- Child emulation: a `pi` process started with `GENTLE_PI_AGENTS_CHILD=1`, `--no-extensions` and one probe extension, which is the argument shape the gentle-agents runner gives its children. The probe did not drive a delegated gentle-agents run, so the subagent is emulated, not observed. Items (a) and (b) are therefore `not testable` in the table; the emulated result is kept here as evidence about the extension API only.
- Extension: a `tool_call` handler that logs each call with `ctx.sessionManager.getSessionId()` and returns `{ block: true, reason }` for a `bash` call containing `probe-deny`.
- Model: `--provider opencode-go --model beam-free`; the default `openai` credential reported `invalid` and was not refreshed.
- Command: `pi --no-extensions -e <probe.ts> --provider opencode-go --model beam-free --mode json -p "<call bash echo probe-deny-1>"`, with `GENTLE_PI_AGENTS_CHILD=1`.
- Seen: `tool_call` fired for the bash call with a session id (`sessionId`); the `tool_execution_end` result was `probe: denied by pre-command hook` with `isError: true`. The event carries no agent id.
- (c): the `agent_end` event fired at the end of the child run. Its handler type in the installed typings (`dist/core/extensions/types.d.ts`) is `ExtensionHandler<AgentEndEvent>` with no result type, which suggests a handler cannot block it. No run tested a block, and the emulated child is not a delegated hand-off, so the cell is `not testable`; the typings reading stays here as the reason to expect no block.
- (d): a primary session (no child marker) ran `echo probe-ok > <file>` with no prompt; the file was written and the stream held no permission event. pi has no permission prompt and no bypass switch for bash.

### opencode (v2.2.6)

- An earlier pass recorded (a) as `no` and (b) and (c) as `not testable`. That pass is withdrawn: its plugin used the `tool.before` shape, and not even a parent-level call logged anything, so its silence was no evidence that the hook was missing. The shape that matters is the one below; the stagegate tests of this repository record that opencode 2.2.6 refuses other plugin shapes. The rerun below uses the shape this version loads: a default export with an `id` and a `setup(ctx)` that registers `ctx.tool.hook("execute.before", ...)` (the shape of the shipped `gabo-stage-gate` plugin). The shipped plugin itself was not run.
- Scratch project with `.opencode/plugins/probe-hook.js`: `execute.before` logs each event and throws `probe: denied by pre-command hook` for a shell command containing `probe-deny`; `execute.after` logs each event and throws `probe: handoff blocked` for a `subagent` tool result. Each log line is written to a file in the scratch directory.
- Positive control, parent level: `opencode run --auto --format json --model opencode/beam-free "Run exactly this shell command with the bash tool and report its output: echo probe-deny-0"`. The `execute.before` hook fired for the `shell` call with the command and the call was denied; the parent reported the denial.
- (a) and (b): `opencode run --auto --format json --model opencode/beam-free "Launch the gentle-ai-worker subagent to run echo probe-deny-1 and report what it returned."`. The subagent's `shell` call reached `execute.before` with `sessionID` and `agent: gentle-ai-worker` (the parent's calls carry `agent: gentle-orchestrator`), and the hook denied it. The subagent reported the denial and no output. So (a) is `yes` and (b) is `yes`.
- (c): the `execute.after` hook fired for the parent's `subagent` tool call with the subagent's `status` and `result`. Throwing there replaced the hand-off: the parent reported `probe: handoff blocked` and no subagent output. So (c) is `yes`.
- The subagent had to be launched as `gentle-ai-worker`: the default agent denies the `subagent` action except for named agents.
- (d): a bash call that the global config asks about (`find ... -delete`) ran under `--auto` with no permission event, and the file was deleted. Under default mode the stream showed `permission requested: shell ...; auto-rejecting` and the file was kept, so a prompt is requested when permissions are not bypassed.

## Open after this record

- The S1 human read is open. T2 waits for a person's read or the owner's written waiver. The assistant's read is recorded in the Reading section and is not the gate.
- The native credential-list formats of codex, pi and opencode (plan PD5 and risk 3) are not covered by this record. T9 must establish them. The plan's T1 row still says T1 verifies them until a plan PR corrects it (D7).
- Pi (a), (b) and (c) need one delegated gentle-agents run to be observed. T12 and T16 run it.
- Codex (d) is weaker evidence than the other engines' (d): the bypass run is the only observation. S12's assumption depends on it, so T2 and T9 treat it as provisional.
- The codex hook trust condition and the opencode hook shape (`execute.before` in a V2 definition, with `execute.after` for the hand-off) are inputs to T9 and T10.
