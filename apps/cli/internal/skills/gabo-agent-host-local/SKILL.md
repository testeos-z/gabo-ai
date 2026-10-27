---
name: gabo-agent-host-local
description: "Trigger: agent host local, agent_host slot, agents on my machine, local agents, start agent session, stop agent session, where agents run. Local adapter for the gbai agent_host slot."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use when an agent starts, stops or reports agent sessions that run on the owner's own machine. This adapter fills the `agent_host` slot contract (`gabo-contract-agent-host`, TRD 9.1; R8, R11, US-007) with the agent CLI of the role's runtime (`runtimes.{coder, reviewer, assistant}`: `claude` or `pi`) run as a local process inside the task worktree. Point `slots.agent_host.adapter` at this skill's `SKILL.md`. Verified against Claude Code 2.1.302 (`claude --help`) and pi 1.4.0 (`pi --help`).

Subagents launched inside the owner's runtime session (for example `gabo-cycle` launching `gabo-coder`) need no agent host; use this adapter only when a role must run as its own session process.

## Hard Rules

- Agents run as the owner's OS user on the owner's machine; nothing runs on a server and no remote host is reached.
- Every former cloud field of the contract is `not used`: no server ids, no separate OS user, no bot identity (the reviewer runs `single-user`), no session prompt files beyond the role's built agent, no Orca CLI name and no system config layer.
- Start a session only in an existing task worktree; never create, move or remove a worktree here, and never touch another task's worktree.
- Never pass an API key on the command line and never read or copy credentials; the agent CLI uses the owner's existing login. A key the runtime needs is named only by its environment variable (R21).
- Write the prompt and the session log in a `mktemp -d` dir, outside the worktree.
- A session is a resource: record its handle (process id, session id, worktree, log path) in the wave ledger when it starts, and stop it or report it as still running.
- Stop only a process whose handle is in the ledger, and only after its live command line still carries `--session-id <sid>`: a process id can be reused after the session exits. Never kill by name or pattern.
- Never put the prompt on the command line: it would leak into `ps` output and can exceed the argument length limit.

## Operations

### start agent session

Generate a session id, write the role's prompt to a file in a scratch dir, then start the agent CLI in the worktree as a background process with the prompt read from that file:

```bash
dir=$(mktemp -d)
sid=$(uuidgen)
# claude: the role is the built agent of the same name (for example gabo-coder); --print reads the prompt from stdin
(cd <worktree> && nohup claude --print --agent <role agent> --session-id "$sid" <"$dir/prompt.md" >"$dir/session.log" 2>&1 & echo $! >"$dir/pid")
# pi: append the role's built agent file to the system prompt and attach the prompt file with @
(cd <worktree> && nohup pi --print --session-id "$sid" --append-system-prompt <role agent file> @"$dir/prompt.md" "Follow the attached task prompt." </dev/null >"$dir/session.log" 2>&1 & echo $! >"$dir/pid")
```

The session handle is the process id in `$dir/pid` plus `$sid`. Confirm after a short wait, so an early exit (bad flag, missing login) is caught:

```bash
sleep 5
ps -o args= -p "$(cat "$dir/pid")" | grep -F -- "--session-id $sid"
```

A match means it is running. No match means it failed to start; report the tail of `$dir/session.log` instead of a handle.

### stop agent session

Match the session id first, then stop:

```bash
ps -o args= -p <pid> | grep -F -- "--session-id <sid>"
kill -TERM <pid>
sleep 10
ps -o args= -p <pid> | grep -F -- "--session-id <sid>"
```

No match before `kill` means the session already exited: report it as `finished` with the tail of its log and send no signal. No match after the wait means it stopped. A match after the wait means it is still running; report that and never escalate to `kill -KILL` without the owner's request.

### report where agents run

```bash
hostname
id -un
ps -o pid=,etime= -p <pid>
ps -o args= -p <pid> | grep -F -- "--session-id <sid>"
```

Host: the owner's machine (`hostname`); OS user: the owner (`id -un`); reviewer mode: `single-user`. Sessions: a ledger handle is `running` only when its process still carries its `--session-id <sid>`; otherwise it is `finished`. Report the role, worktree, session id and elapsed time, never the process arguments.

## Output Contract

Return the host, the OS user, the reviewer mode, and per session its role, worktree, process id, session id, log path, the commands run and the final state (`running`, `stopped`, `finished` or `failed to start`).
