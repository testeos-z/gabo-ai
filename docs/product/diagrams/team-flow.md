# Team flow: one subagent layer

The assistant runs the wave in the main session. Its only subagents are the directiva agents, the plan reviewer and one `gabo-coder` per task. No gabo agent launches another agent. Each coder starts the user-level reviewer script as a separate process in a clean clone of the repository. Intent 0015 (spec in PR #475, not yet on `dev`) defines this graph.

```mermaid
flowchart TD
  owner([Owner])

  subgraph main["Main session (assistant)"]
    assistant["gabo-assistant"]
    cycle["gabo-cycle skill<br/>reads unblocked tasks, launches coders,<br/>updates the tracker and ledger, cleans up"]
  end

  subgraph directiva["Directiva subagents"]
    pm["gabo-pm"]
    lead["gabo-dev-lead"]
    db["gabo-db-master"]
    fe["gabo-frontend"]
    planrev["gabo-plan-reviewer<br/>only on a plan change"]
  end

  subgraph wave["Wave subagents: one per task, in its worktree"]
    coder["gabo-coder x N"]
  end

  script["Reviewer script<br/>separate process, clean clone"]
  tracker[("Tracker<br/>issues or cards")]
  memory[("Memory<br/>decisions, ledger, trace")]
  pr["Pull request to dev"]

  owner -->|goal, answers| assistant
  assistant -->|loads and runs| cycle
  assistant -->|plan and spec work| pm
  assistant --> lead
  assistant --> db
  assistant --> fe
  assistant -->|plan change, owner absent| planrev
  lead -->|creates tasks| tracker
  cycle -->|reads unblocked, updates state| tracker
  cycle -->|launches one per task| coder
  coder -->|outcome: HANDED, DONE, BLOCKED, questions| cycle
  coder -->|opens| pr
  coder -->|bash, PR number and head SHA| script
  script -->|VERDICT line| coder
  script -->|verdict comment| pr
  assistant -->|records decisions| memory
  assistant -->|wave report| owner
```

## Reading the graph

- **Main session.** `gabo-cycle` is a skill, not an agent. It runs inside the assistant's session on every engine, so a Claude Code wave works with `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1`.
- **Coders.** Each coder is a first-level subagent. It works in its own worktree, opens the PR and closes its task after the merge.
- **Review.** The coder runs the reviewer script through its shell, reads the last `VERDICT` line and acts on it. There is no reviewer relay agent.
- **Questions.** A coder returns questions in its outcome. The main session relays them to the directiva agents or the owner; coders never message another agent.
- **Configurable points.** The tracker, the memory, the repository host, the methodology and the reviewer placement and engine are slots set in `gabo.yaml`. See [configuration-flow.html](configuration-flow.html).
