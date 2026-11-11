# Intent 0019 Talk to cloud agent sessions

| field | value |
|---|---|
| intent id | 0019-talk-to-cloud-agent-sessions |
| status | Specified |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-11 |

## Problem

The owner has no defined way to talk to an agent session that runs somewhere else, such as a cloud session or a remote host. The `direct_sessions` slot contract today opens a session in a task's worktree, sends it one message, reads its state and closes it, through an adapter like Orca. It does not define how to reach a session that `agent_host` runs remotely, or which tools to use for that.

Owner, verbatim (2026-11-11): "quiero que evaluemos el tema de conversar con el agente de la nube similara las tareas definimos herramientas para eso, como si quieres usar orca o la comunicacion entre sessiones se cloude o ssh si es remoto"

## Who is affected

The owner, whenever agents run off the owner's machine (a cloud session or a remote host). It happens each time the owner wants to steer or ask a running remote agent.

## Desired outcome

- The owner can evaluate, then use, a defined way to send a message to an agent session and read its answer wherever the session runs.
- The tools for it are chosen per setup, as the tracker is chosen for tasks (owner: "similar a las tareas definimos herramientas"), with Orca, sessions between Claude sessions, or SSH for a remote host as candidates.

## Constraints

- Remote work needs explicit owner authorization for each destination, operation and credential; an available connection is not permission.
- No secrets are stored or written (PRD R21).
- A session is not a second coder: while a task's coder runs, messages go to an open session only when the owner asks (`direct_sessions` hard rules).
- Every session opened is closed or reported as still open, and is recorded in the wave ledger (`direct_sessions` hard rules, D15).

## Assistant's analysis (input for the spec, not confirmed by the owner)

- Widen the `direct_sessions` contract instead of adding a slot: it already has open, send message, read state and close. The handle can be any session that `agent_host` reports, a remote one included.
- Candidate adapters: Orca (exists); Claude sessions (a cloud session receives messages but cannot reply, so its answer is read from its transcript; to be verified); SSH with tmux (any engine; needs the authorization above for each destination and credential).
- Messages that must persist stay on the tracker.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does the owner want an evaluation first (a written comparison of the options), or the evaluation and the first adapter together? | answered: the evaluation, the contract widening and the update of the Orca adapter together; Claude-sessions and SSH-with-tmux adapters come later (spec D2). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Widen `direct_sessions`, or add a new slot? | answered: widen `direct_sessions` (spec D1). Recorded decision (resolver: agent, rule of the owner). |
| 3 | Which adapter comes first: Orca, Claude sessions or SSH with tmux? | answered: Orca (spec D3). Recorded decision (resolver: agent, rule of the owner). |
| 4 | Is the Claude cloud-session behavior (receives messages, cannot reply) confirmed by the engine's documentation? | answered: yes; the Claude Code documentation says the command queues the message and exits without waiting for a reply (spec evidence base and S6). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#494 I0019: Talk to cloud agent sessions](https://github.com/testeos-z/gabo-ai/issues/494) (sub-issue of #492) |
| PRD | [PRD](../../PRD.md), R21 |
| contract | `.gabo/skills/gabo-contract-direct-sessions/SKILL.md`, `.gabo/skills/gabo-contract-agent-host/SKILL.md` |
| spec | [spec](spec.md) |
| plan | — |
| related intent | — |
