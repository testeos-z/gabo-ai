# Probe 0028: hooks in coder subagents (S1)

| field | value |
|---|---|
| intent | [0028-deterministic-hooks-as-locks](intent.md) |
| spec | [spec S1](spec.md) |
| task | I0028/T1, issue #802 |
| run | 2026-11-14, one machine, every engine headless, default model of each engine except where a run names one |

## Record

Caveat on the pi row: its (a) and (b) come from an emulated child process, not from a delegated gentle-agents subagent (see the pi notes below). Codex (d) rests on the bypass run alone (see the codex notes).

Items: (a) a pre-command hook fires for a shell call inside a coder subagent and can deny it; (b) the subagent's tool calls reach the hook with an agent or session id; (c) the hand-off event exists and can block; (d) the engine's yes/no permission prompt appears when the session runs with permissions bypassed.

| engine | version | (a) | (b) | (c) | (d) |
|---|---|---|---|---|---|
| claude | 2.1.302 (Claude Code) | yes | yes | yes | no |
| codex | codex-cli 0.167.2 | yes | yes | yes | no |
| pi | 1.5.2 | yes | yes | no | no |
| opencode | opencode v2.2.6 | no | not testable: no hook was invoked for any call, so no id reached one | not testable: no subagent hand-off hook fired, so blocking cannot be observed | no |

Consequence for the plan: item (a) is `no` for opencode. S14 applies to opencode. Every lock that needs a pre-command hook on opencode shows `not run: <reason>` in the wave report, and no task stops for it. Claude, codex and pi show (a) `yes`, so no S14 fallback is stated for them on this record.

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

- Child emulation: a `pi` process started with `GENTLE_PI_AGENTS_CHILD=1`, `--no-extensions` and one probe extension, which is the argument shape the gentle-agents runner gives its children. The probe did not drive a delegated gentle-agents run, so the subagent is emulated, not observed.
- Extension: a `tool_call` handler that logs each call with `ctx.sessionManager.getSessionId()` and returns `{ block: true, reason }` for a `bash` call containing `probe-deny`.
- Model: `--provider opencode-go --model beam-free`; the default `openai` credential reported `invalid` and was not refreshed.
- Command: `pi --no-extensions -e <probe.ts> --provider opencode-go --model beam-free --mode json -p "<call bash echo probe-deny-1>"`, with `GENTLE_PI_AGENTS_CHILD=1`.
- Seen: `tool_call` fired for the bash call with a session id (`sessionId`); the `tool_execution_end` result was `probe: denied by pre-command hook` with `isError: true`. The event carries no agent id.
- (c): the `agent_end` event fired at the end of the child run, but its handler type is `ExtensionHandler<AgentEndEvent>` with no result type (installed typings `dist/core/extensions/types.d.ts`), so a handler cannot block it. The hand-off cannot block.
- (d): a primary session (no child marker) ran `echo probe-ok > <file>` with no prompt; the file was written and the stream held no permission event. pi has no permission prompt and no bypass switch for bash.

### opencode (v2.2.6)

- Two configurations were tried, both in a scratch project, and neither invoked a hook on any call:
  - `.claude/settings.json` with the same `PreToolUse` and `SubagentStop` command hooks as claude. The subagent completed, `echo probe-deny-1` ran, and no hook log was written.
  - A project plugin under `.opencode/plugin/` and `.opencode/plugins/`, listed by `opencode plugin list` and referenced from `opencode.json`, with a `tool.before` hook in both the plain and the `hooks` object shape. It logged nothing, and it logged nothing for a parent-level bash call either, so the hook was not invoked.
- The subagent had to be launched as `gentle-ai-worker`: the default agent denies the `subagent` action except for named agents.
- Command: `opencode run --auto --format json --model opencode/beam-free "<launch the gentle-ai-worker subagent to run echo probe-deny-1>"`.
- Seen: the subagent ran the command and returned `probe-deny-1`; no pre-command hook fired, so (a) is `no`. Because no hook received any call, (b) cannot be observed. No `SubagentStop` hook fired, so the hand-off cannot be observed, and no typed hand-off hook for subagents was found.
- (d): a bash call that the global config asks about (`find ... -delete`) ran under `--auto` with no permission event, and the file was deleted. Under default mode the stream showed `permission requested: shell ...; auto-rejecting` and the file was kept, so a prompt is requested when permissions are not bypassed.

## Open after this record

- The human read of this record that gates T2 is done by the assistant under the owner's order of 2026-11-14 ("aprueben solos lo necesario"), as the PR body says.
- The native credential-list formats of codex, pi and opencode (plan PD5 and risk 3) are not covered by this record. T9 must establish them.
- Codex (d) is weaker evidence than the other engines' (d): the bypass run is the only observation. S12's assumption depends on it, so T2 and T9 treat it as provisional.
- The codex hook trust condition and the opencode hook result are inputs to T9 and T10 and to S14 for opencode.
