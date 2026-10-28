---
name: gabo-agent-host-local
description: "Trigger: agent host local, agent_host slot, agents on my machine, local agents, start agent session, stop agent session, where agents run. Local adapter for the gabo agent_host slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Use when an agent starts, stops or reports agent sessions that run on the owner's own machine. This adapter fills the `agent_host` slot contract (`gabo-contract-agent-host`, TRD 9.1; R8, R11, US-007) with the agent CLI of the role's runtime (`runtimes.coder` or `runtimes.assistant`: `claude` or `pi`) run as a local process inside the task worktree. Point `slots.agent_host.adapter` at this skill's `SKILL.md`. Verified against Claude Code 2.1.312 (`claude --help`) and pi 1.4.0 (`pi --help`).

Subagents launched inside the owner's runtime session (for example `gabo-cycle` launching `gabo-coder`) need no agent host; use this adapter only when a role must run as its own session process.

## Hard Rules

- Agents run as the owner's OS user on the owner's machine; nothing runs on a server and no remote host is reached.
- Every former cloud field of the contract is `not used` (see Fields); the reviewer runs `single-user`.
- Never start a `reviewer` session, on either runtime: its worktree holds the branch under review, and the agent CLI would load that branch's settings, hooks, agents and skills, so the code under review could widen the reviewer's tools. The reviewer gate runs through the `reviewer_placement` slot. Report the refusal instead of a handle.
- A `coder` session starts only in its existing task worktree. An `assistant` session starts in the repository's main checkout (the parent of `git rev-parse --path-format=absolute --git-common-dir`), the owner's own trusted copy, since the assistant works on no single task. Never create, move or remove a worktree here, and never touch another task's worktree. `<worktree>` below is that directory.
- Never pass an API key on the command line and never read or copy credentials; the agent CLI uses the owner's existing login. A key the runtime needs is named only by its environment variable (R21).
- Write the prompt and the session log in a `mktemp -d` dir, outside the worktree.
- A session is a resource: record its handle (process id, session id, worktree, scratch dir) in the wave ledger when it starts, and stop it or report it as still running.
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
| `/etc/gabo/gabo.yaml` | `not used` |

The role's prompt comes from its built agent and the brief of the caller; the OS user is the owner.

## Operations

### start agent session

A session runs headless (`--print`), so nobody can answer a permission prompt. The adapter never relies on one: it pins on the command line the tools the session may use, and anything else is denied at once instead of waiting.

- `claude`: `--permission-mode dontAsk --permission-prompts none` denies every tool that is not pre-approved, and `--allowedTools "$tools"` pre-approves exactly the tools of the role's built agent (the one `tools:` line of its frontmatter, for example `Read, Grep, Glob, Edit, Write, Bash, Skill, Agent` for `gabo-coder`). The owner's settings `deny` rules still apply on top. The session also reads the worktree's project settings; that is safe only because the worktree holds the task's own branch, written by the same role, which is why a `reviewer` session is never started.
- `pi`: it has no tool approval prompt; every enabled tool runs, and the adapter has no per-role tool list for it. It starts only the `coder` and `assistant` roles, which need edit and shell tools anyway: `--tools read,bash,edit,write,grep,find,ls` pins the built-in tools, `--append-system-prompt <role agent file>` reads the file (pi 1.4.0 `--help`: "Append text or file contents to the system prompt"), and `--approve` trusts the worktree's project-local files (its built agents and skills) for this run, so no trust question is left open. The worktree holds the task's own branch, written by the same role.

Check before starting, and start nothing when a check fails; report which one failed instead of a handle. When the `tools:` check fails, say that the built agent needs exactly one one-line `tools: A, B, C` entry in its frontmatter (a YAML list under `tools:` is refused, never guessed):

```bash
# both runtimes: only the coder and assistant roles; a reviewer is refused before anything else
# claude: the role's built agent file (for example .claude/agents/gabo-coder.md) must have exactly one
# tools: line inside its frontmatter, never a body line
case "<runtime role>" in coder|assistant) ;; *) false ;; esac &&
  tools=$(awk '/^---$/ { if (++n == 2) exit; next } n == 1 && sub(/^tools: */, "") { print; c++ } END { exit c != 1 }' <role agent file>) &&
  [ -n "$tools" ] && claude --help | grep -qF -- --permission-prompts && claude --help | grep -qF -- --allowedTools
# pi: the installed version must support every flag the start command relies on
case "<runtime role>" in coder|assistant) ;; *) false ;; esac &&
  pi --help | grep -qF -- --approve && pi --help | grep -qF -- --tools &&
  pi --help | grep -qF -- --session-id && pi --help | grep -qF -- --append-system-prompt
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

Once a session is `stopped`, `finished` or `failed to start` and the tail of its log is in the report, remove its scratch dir (prompt, log and pid file, recorded in the ledger handle) and drop the handle from the ledger; a `running` session keeps both:

```bash
rm -r -- <scratch dir>
```

### report where agents run

```bash
hostname
id -un
ps -o args= -p <pid> | grep -qF -- "--session-id <sid>" && ps -o etime= -p <pid> || echo finished
```

Host: the owner's machine (`hostname`); OS user: the owner (`id -un`); reviewer mode: `single-user`. Sessions: a ledger handle is `running` only when its process still carries its `--session-id <sid>`; otherwise it is `finished`. The elapsed time is read only after the session id matched, so a reused process id never reports another process's time. Report the role, worktree, session id and elapsed time, never the process arguments.

## Output Contract

Return the host, the OS user, the reviewer mode, and per session its role, worktree, process id, session id, scratch dir and log path, the commands run and the final state (`running`, `stopped`, `finished`, `failed to start` or `refused`).
