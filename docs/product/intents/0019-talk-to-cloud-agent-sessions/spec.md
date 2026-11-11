# Spec 0019 Talk to cloud agent sessions

| field | value |
|---|---|
| intent | [0019-talk-to-cloud-agent-sessions](intent.md) |
| status | Draft (awaits owner acceptance) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`. Owner's standing rule (2026-11-11): the agent applies its recommended answer to every open question and records it as a decision; only merging to main, tags and secrets stay the owner's. Each decision below is marked **Recorded decision (resolver: agent, rule of the owner)** and is saved through the `memory` slot. The owner's own words are the source and are quoted verbatim.

Source, owner verbatim (2026-11-11): "quiero que evaluemos el tema de conversar con el agente de la nube similara las tareas definimos herramientas para eso, como si quieres usar orca o la comunicacion entre sessiones se cloude o ssh si es remoto". On which intent gets its spec next: "vamos en orden numerico".

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`, and the Claude Code documentation):

- **The slot contract today.** `.gabo/skills/gabo-contract-direct-sessions/SKILL.md`: four operations, `open session` (starts an agent session in one task worktree with a brief and returns a handle), `send message`, `read state` and `close session`. Its hard rules: a session opens only in an existing task worktree; it is not a second coder, so while the task's coder runs, messages go to it only when the owner asked; every session the adapter opens is closed or reported, and goes into the wave ledger. `gabo check` compares the `### <operation>` headings of the adapter with the contract.
- **The host contract.** `.gabo/skills/gabo-contract-agent-host/SKILL.md`: `start agent session` (returns a handle), `stop agent session` and `report where agents run` (lists the sessions running there). Hard rules: starting or stopping a remote session needs the owner's explicit authorization of the host, the operation and the credential; an adapter never discovers or reuses ambient credentials; secrets are named by env var only (R21).
- **The shipped Orca adapters.** `.gabo/skills/gabo-direct-sessions-orca/SKILL.md` drives one Orca terminal per session and opens `claude` and `pi` sessions only (a resolved `codex` or `opencode` is refused). It states that a remote Orca host adds `--environment <name>` to every command and that the host belongs to the `agent_host` adapter. `.gabo/skills/gabo-agent-host-railway-orca/SKILL.md` runs every `orca` command with `--environment "$GABO_ORCA_ENVIRONMENT"` and names the environment by env var.
- **Recorded rules.** PRD R14 (the assistant mediates by default and offers a named Orca session in the task's worktree) and R18; TRD 9.1 row for `direct_sessions`; TRD line 104 (direct sessions depend on Orca in the first release). `gabo-setup` step 1 says agents communicate through the tracker, the `memory` slot and, only if configured, `direct_sessions`, and that agents on different hosts never talk directly.
- **Claude cloud sessions.** [Use Claude Code in the cloud](https://code.claude.com/docs/en/claude-code-on-the-web), section "Send follow-ups from the CLI" (checked 2026-11-11): `claude -p "your message" --cloud <session-id>` "posts one message and exits"; "The CLI queues the message into the session and exits without waiting for a reply." It needs an Anthropic account (not Amazon Bedrock, Google Vertex or another third-party provider) and the organization policy `allow_remote_sessions`; an archived session answers `cloud session <id> is archived and cannot accept new messages`. The session itself is viewed at claude.ai/code.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The `direct_sessions` contract shall let `send message` and `read state` take the handle of any session that `agent_host` reports (`start agent session` or `report where agents run`), on this machine or on a remote host. The four operation headings and the meaning of `open session` stay as they are; `close session` closes only a session this slot opened. | Given a session started by the `agent_host` adapter on a server, When the assistant sends it a message by its handle, Then the adapter delivers it and returns the result. Given the same handle and `read state`, Then the state and latest output come back. Given `close session` on a handle that this slot did not open, Then the adapter refuses and names `stop agent session` of the host. Given the contract file, Then it still has the four headings `open session`, `send message`, `read state` and `close session`. | skill review against S1; `gabo check` passes on this repository, which compares the adapter's headings with the contract; probe (S8) |
| S2 | A handle shall name its host, so that the adapter picks the right host: the host as `agent_host` reports it plus the session handle that host returns. A local handle names the local host. | Given a handle for a remote host, Then the adapter addresses that host and no other. Given a handle for the local host, Then no remote argument is added. Given a handle that names no known host, Then the adapter refuses and sends nothing. | skill review; probe (S8) |
| S3 | The Orca adapter shall accept such a handle: for a remote host it adds `--environment <name>` to every command, taking the name from the env var that the host's adapter declares; for a local host it adds nothing. It shall keep opening `claude` and `pi` sessions only, and `send message` and `read state` shall work for a session of any engine, since they drive a terminal. | Given a remote handle, Then every `orca terminal send` and `orca terminal read` of the adapter carries `--environment` with the declared name, and never a value of a secret. Given a session running `codex` that `agent_host` started, Then `send message` and `read state` work, and `open session` for `codex` is still refused as today. | skill review; probe (S8) |
| S4 | Any remote use of the slot shall need the owner's explicit authorization naming the host, the operation and the credential or session to use, and the contract shall say so for every remote adapter. An adapter never discovers or reuses ambient credentials, SSH agents, ControlMaster sockets, Orca pairings or authenticated sessions. | Given a remote handle and no authorization naming that host, operation and credential in this session, Then the adapter sends and reads nothing and asks the caller for the authorization. Given an authorization for host A, Then host B is not reached. Given the contract file, Then this rule is in its hard rules. | skill review; probe (S8) with a refused attempt |
| S5 | The contract shall state that a direct session is live conversation only: a question, an answer or a decision that must persist goes to the tracker as an issue comment through the `tasks` adapter, and a decision is also saved through `memory`. Nothing that matters is left only in a session. | Given a decision reached in a direct session, Then the assistant posts it as a comment on the task issue and saves it through `memory` before it closes the session. Given a session that drops, Then nothing the team needs is lost, because it was posted. | skill review; the `gabo-assistant` text for Intervene in a Task names the step; probe (S8) |
| S6 | An adapter that cannot return a session's reply to the caller shall say so in `read state`, naming where the answer is read, and never return an empty output as if the agent were silent. | Given a cloud Claude session reached through `claude -p ... --cloud`, Then `read state` reports "reply not returned by this command; read it in the session at claude.ai/code" and the send result. | skill review of the contract rule; applies to the later Claude-sessions adapter |
| S7 | The hard rules of the contract stay: a session is not a second coder (while a task's coder runs, messages go to a session only when the owner asked) and every session the slot opens is closed or reported and recorded in the wave ledger. The change adds no Go command, touches no Go file and changes no `gabo check` output or exit code. | Given a coder running and no owner request, Then no message is sent. Given `go -C apps/cli test ./...` and `gabo check`, Then both pass with no golden file changed. Given the diff, Then it touches only skill files and docs. | skill review; `go -C apps/cli build ./...`, `vet`, `test`; `gabo check`; review of the diff for Go files |
| S8 | The change shall be probed on a disposable repository: one session on the local host and one on a remote Orca host, each with `open session`, `send message`, `read state`, `close session`, and a refused attempt without authorization. The remote probe shall run only with the owner's authorization of that host at probe time. | Given the local probe, Then the four operations pass. Given the owner's authorization for a remote host, Then the same four operations pass there. Given no authorization, Then the remote probe does not start. | probe, with its result read back and recorded in the intent's review findings; the plan assigns it to a task |

## 2. Design

Affected files: the `gabo-contract-direct-sessions` skill (S1, S2, S4, S5, S6), the `gabo-direct-sessions-orca` adapter (S3), the Intervene in a Task text of the `gabo-assistant` skill (S5), the `gabo-setup` sentence on agent communication (section 5), the TRD 9.1 row for `direct_sessions`, and the catalog copies of these skills. No Go file.

Evaluation of the three candidate tools (the owner's "orca o la comunicacion entre sessiones se cloude o ssh"):

| tool | reach | can the answer return to the caller | engines | authorization | status in this intent |
|---|---|---|---|---|---|
| Orca terminals | local and remote: the shipped adapters add `--environment <name>` for a remote host | yes: `terminal read` returns the screen and new output | the adapter opens `claude` and `pi`; send and read work on any terminal | the remote host needs the owner's explicit authorization (S4) | first adapter, built here (D3) |
| Claude cloud sessions | any Anthropic cloud session, from the CLI of any machine logged in to the same account | no: the command queues the message and exits without waiting; the reply is read in the session at claude.ai/code | Claude Code only | an Anthropic account (not a third-party provider) and the organization policy `allow_remote_sessions`; an archived session refuses | named, built later (D3); limit stated in S6 |
| SSH with tmux | any host the owner can reach | not verified in this intent: expected through the tmux pane, to be verified when it is built | any engine that runs in a terminal | per destination, operation and credential or session; never an ambient agent or socket (S4) | named, built later (D3) |

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Contract shape | (a) widen `direct_sessions`: the handle may be any session `agent_host` reports; (b) a new slot for remote conversations; (c) put "talk to a session" into `agent_host` | **Recorded decision (resolver: agent, rule of the owner):** (a), the assistant's recommendation. Reason: the four operations already exist, and `agent_host` already returns handles; a second slot would overlap `direct_sessions`, and (c) mixes where agents run with how the owner talks to them. | Existing adapters keep their four headings, so `gabo check` coverage does not change. TRD 9.1 gets one amended sentence. |
| D2 | Scope of this intent | (a) the evaluation, the contract widening and the update of the Orca adapter; (b) also build the Claude-sessions and SSH adapters | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: Orca exists and already covers local and remote; the other two have their own limits and authorization needs and are better judged one at a time. | The Claude-sessions adapter and the SSH-with-tmux adapter are later intents or tasks, created when the owner picks them; this spec names them and states their known limits only. |
| D3 | First adapter | (a) Orca; (b) Claude sessions; (c) SSH with tmux | **Recorded decision (resolver: agent, rule of the owner):** (a), because it exists and covers local and remote. | Engines other than `claude` and `pi` still cannot be opened by the Orca adapter, but can be messaged and read when `agent_host` started them (S3). |
| D4 | Durable versus live messages | (a) the tracker holds anything that must persist and a direct session is live only; (b) treat the session transcript as the record | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: a session is lost if it drops, and the tracker is where agents on different hosts already meet. | One extra step for the assistant before it closes a session (S5). |
| D5 | Remote authorization | (a) the owner's explicit authorization of host, operation and credential for any remote use, never ambient access; (b) trust a configured host | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: it is the rule of the `agent_host` contract (R21) and the owner's stated rule for SSH; a configured host is not permission. | Remote adapters of every kind inherit it from the contract (S4). |
| D6 | Closing a session this slot did not open | (a) refuse and name `stop agent session`; (b) allow it | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: the host that started a coder session owns its lifecycle, and closing it from a conversation tool could stop a running coder. | The slot stays a conversation tool; stopping stays with `agent_host`. |

## 3. UX and app flow (optional)

Not applicable: no screen or component. The journey is the existing Intervene in a Task flow of the assistant, with one new destination: after three mediated exchanges the assistant offers a direct session (APP_FLOW A5), and a remote session is reached by its handle once the owner authorizes the host. Wording follows the assistant's existing prompts.

## 4. DB migration steps (optional)

Not applicable: no stored data changes.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| `gabo-setup` step 1: "agents on different hosts never talk directly" | A session on another host is reached through `direct_sessions`. | resolved by scope: the rule is about agents' work traffic between themselves; this intent adds the owner's conversation with one agent through the assistant, which stays the single point of contact. When the owner accepts the spec, the sentence is amended to say agents never talk to each other directly across hosts, and the owner can reach a remote session through `direct_sessions` after authorizing it. |
| TRD 9.1 row for `direct_sessions` ("open session in a worktree with a brief, send message, read state, close") | It does not say the handle may be a remote session reported by `agent_host`. | amended with the spec's acceptance: the row says send message and read state take any handle `agent_host` reports. A TRD convention change, decided by the owner's acceptance. |
| PRD R14 and R18 (the assistant mediates by default; one Orca terminal opens by default) | A session on a server is a further terminal. | none: S7 keeps mediation by default and the owner's request as the trigger. |
| `gabo-contract-agent-host` hard rule on authorization and R21 | `direct_sessions` reaches remote hosts. | none: S4 applies the same rule to the slot. |
| TRD line 104 (direct sessions depend on Orca in the first release) | Other tools are named. | none: D3 keeps Orca as the first and only adapter built. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Does the owner want an evaluation first, or the evaluation and the first adapter together? | answered: the evaluation, the contract widening and the Orca update together (D2). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Widen `direct_sessions`, or add a new slot? | answered: widen (D1). Recorded decision (resolver: agent, rule of the owner). |
| 3 | Which adapter comes first? | answered: Orca (D3). Recorded decision (resolver: agent, rule of the owner). |
| 4 | Is the Claude cloud-session behavior (receives messages, cannot reply to the sender) confirmed by the engine's documentation? | answered: yes, by the Claude Code documentation cited in the evidence base: the command queues the message and exits without waiting for a reply (S6). How a later adapter reads the answer back, beyond the session at claude.ai/code, is a question for that adapter's own intent. |
| 5 | How does an SSH-with-tmux adapter return a reply, and under which authorization? | carried to the later SSH intent; the authorization rule is S4. |

The remote probe (S8) needs the owner's authorization of the host at probe time; nothing remote is contacted before then. Nothing else needs the owner except the acceptance in section 7.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
