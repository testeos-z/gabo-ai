---
name: gabo-agent-host-local
description: "Trigger: agent host local, agent_host slot, agents on my machine, local agents, start agent session, stop agent session, where agents run. Local adapter for the gbai agent_host slot."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use when an agent starts, stops or reports agent sessions that run on the owner's own machine. This adapter fills the `agent_host` slot contract (`gabo-contract-agent-host`, TRD 9.1; R8, R11, US-007) with the agent CLI of the role's runtime (`runtimes.{coder, reviewer, assistant}`: `claude` or `pi`) run as a local process inside the task worktree. Point `slots.agent_host.adapter` at this skill's `SKILL.md`. Verified against Claude Code 2.1.312 (`claude --help`) and pi 1.4.0 (`pi --help`).

Subagents launched inside the owner's runtime session (for example `gabo-cycle` launching `gabo-coder`) need no agent host; use this adapter only when a role must run as its own session process.

## Hard Rules

- Agents run as the owner's OS user on the owner's machine; nothing runs on a server and no remote host is reached.
- Every former cloud field of the contract is `not used` (see Fields); the reviewer runs `single-user`.
- Start a session only in an existing task worktree; never create, move or remove a worktree here, and never touch another task's worktree.
- Never pass an API key on the command line and never read or copy credentials; the agent CLI uses the owner's existing login. A key the runtime needs is named only by its environment variable (R21).
- Write the prompt and the session log in a `mktemp -d` dir, outside the worktree.
- A session is a resource: record its handle (process id, session id, worktree, log path) in the wave ledger when it starts, and stop it or report it as still running.
- Stop only a process whose handle is in the ledger, and only after its live command line still carries `--session-id <sid>`: a process id can be reused after the session exits. Never kill by name or pattern.
- Never put the prompt on the command line: it would leak into `ps` output and can exceed the argument length limit.

## Fields

| former key | value |
|---|---|
| `server.railway.{project_id, environment_id, service_id}` | `not used` |
| `users.orca.{name, home, uid, extra_path}` | `not used` |
| `bot.login` | `not used` |
| `sessions.{coder, reviewer, assistant}.prompt_file` | `not used` |
| `sessions.local_orca` | `not used` |
| `/etc/gbai/gbai.yaml` | `not used` |

The role's prompt comes from its built agent and the brief of the caller; the OS user is the owner.

## Operations

### start agent session

A session runs headless (`--print`), so nobody can answer a permission prompt. The adapter never relies on one: it pins on the command line the tools the session may use, and anything else is denied at once instead of waiting.

- `claude`: `--permission-mode dontAsk --permission-prompts none` denies every tool that is not pre-approved, and `--allowedTools "$tools"` pre-approves exactly the tools of the role's built agent (its `tools:` line, for example `Read, Grep, Glob, Edit, Write, Bash, Skill, Agent` for `gabo-coder`). The owner's settings `deny` rules still apply on top.
- `pi`: it has no tool approval prompt; every enabled tool runs. `--tools read,bash,edit,write,grep,find,ls` pins the built-in tools, and `--approve` trusts the worktree's project-local files (its built agents and skills) for this run, so no trust question is left open.

Check before starting, and start nothing when a check fails; report which one failed instead of a handle:

```bash
# claude: the role's built agent file (for example .claude/agents/gabo-coder.md) must name its tools
tools=$(sed -n 's/^tools: *//p' <role agent file>)
[ -n "$tools" ] && claude --help | grep -qF -- --permission-prompts && claude --help | grep -qF -- --allowedTools
# pi: the installed version must support the pinned flags
pi --help | grep -qF -- --approve && pi --help | grep -qF -- --tools
```

Then generate a session id, write the role's prompt to a file in a scratch dir, and start the agent CLI in the worktree as a background process with the prompt read from that file:

```bash
dir=$(mktemp -d)
sid=$(uuidgen)
# claude: the role is the built agent of the same name (for example gabo-coder); --print reads the prompt from stdin
(cd <worktree> && { nohup claude --print --agent <role agent> --session-id "$sid" --permission-mode dontAsk --permission-prompts none --allowedTools "$tools" <"$dir/prompt.md" >"$dir/session.log" 2>&1 & echo $! >"$dir/pid"; })
# pi: append the role's built agent file to the system prompt and attach the prompt file with @
(cd <worktree> && { nohup pi --print --approve --tools read,bash,edit,write,grep,find,ls --session-id "$sid" --append-system-prompt <role agent file> @"$dir/prompt.md" "Follow the attached task prompt." </dev/null >"$dir/session.log" 2>&1 & echo $! >"$dir/pid"; })
```

Only the agent command is backgrounded, so `$!` is the process `nohup` runs the agent in, not a subshell. The session handle is the process id in `$dir/pid` plus `$sid`. Confirm after a short wait, so an early exit (bad flag, missing login) is caught:

```bash
sleep 5
ps -o args= -p "$(cat "$dir/pid")" | grep -qF -- "--session-id $sid" && echo running || echo "failed to start"
```

`running` means it started. `failed to start` means it exited; report the tail of `$dir/session.log` instead of a handle. A tool denied later shows in the session log as a denied tool call, never as a stalled session.

### stop agent session

Signal only when the session id still matches:

```bash
ps -o args= -p <pid> | grep -qF -- "--session-id <sid>" && kill -TERM <pid>
sleep 10
ps -o args= -p <pid> | grep -qF -- "--session-id <sid>" && echo running || echo stopped
```

No match before `kill` means the session already exited and no signal is sent: report it as `finished` with the tail of its log. `stopped` after the wait means it stopped. `running` after the wait means it is still running; report that and never escalate to `kill -KILL` without the owner's request.

### report where agents run

```bash
hostname
id -un
ps -o pid=,etime= -p <pid>
ps -o args= -p <pid> | grep -qF -- "--session-id <sid>" && echo running || echo finished
```

Host: the owner's machine (`hostname`); OS user: the owner (`id -un`); reviewer mode: `single-user`. Sessions: a ledger handle is `running` only when its process still carries its `--session-id <sid>`; otherwise it is `finished`. Report the role, worktree, session id and elapsed time, never the process arguments.

## Output Contract

Return the host, the OS user, the reviewer mode, and per session its role, worktree, process id, session id, log path, the commands run and the final state (`running`, `stopped`, `finished` or `failed to start`).
