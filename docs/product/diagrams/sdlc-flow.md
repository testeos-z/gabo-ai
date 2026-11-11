# Per-change SDLC flow

Each change is one intent. The tracker is the global view. This chart shows who does what from the idea to production, with one subagent layer in the wave (intent 0015, spec S1, S10): the main session runs the `gabo-cycle` skill and launches the coders; each coder starts the reviewer script as a separate process in a clean clone.

```mermaid
flowchart TD
  subgraph spec["Context and spec per change"]
    idea([Owner: product idea])
    ctx["Assistant: short context and stack<br/>PRD, TRD"]
    pm["gabo-pm: intent.md to spec.md"]
    contrib["gabo-frontend + gabo-db-master<br/>UX flow and migrations"]
    accept{{"Owner answers and accepts spec.md<br/>gate before the plan"}}
  end

  subgraph plan["Plan"]
    lead["gabo-dev-lead: writes plan.md"]
    card[("Product card")]
    tasks[("Task issues T###<br/>children of the intent, blocked-by")]
  end

  subgraph wave["Wave (repeats): main session runs the gabo-cycle skill"]
    launch["Assistant: launches the wave, does not block"]
    cycle["gabo-cycle skill: claims unblocked tasks<br/>launches one coder per task"]
    coder["gabo-coder x N: own worktree<br/>methodology from gabo.yaml"]
    script["Reviewer script: separate process<br/>clean clone, checks and agent review of spec.md and plan.md"]
    report["Assistant: receives the wave report"]
    memory[("Memory: decisions, reports, ledger")]
  end

  subgraph ship["Review, production and questions"]
    merge["PR to dev, merge<br/>CI/CD and deterministic hooks"]
    gate{{"Production gate<br/>only the owner crosses it"}}
    prod([Ready for production])
  end

  subgraph float["Floating questions"]
    qlead["gabo-dev-lead receives the question"]
    qpm["gabo-pm: product question"]
    qowner(["Owner answers or debates"])
    block["Critical question: plan change<br/>1 issue + 1 PR, only that T### BLOCKED"]
    planrev["gabo-plan-reviewer<br/>agrees or counter-proposes, owner absent only"]
  end

  idea -->|idea| ctx
  ctx -->|new change| pm
  pm -->|asks for input| contrib
  pm -->|questions and spec.md| accept
  accept -->|spec accepted| lead
  lead -->|creates T###| tasks
  card -->|children of the intent| tasks
  tasks -->|unblocked| cycle
  launch -->|starts the wave| cycle
  cycle -->|launches| coder
  coder -->|opens PR, runs bash| script
  script -->|VERDICT| coder
  script -->|VERDICT approved| merge
  coder -->|outcome| cycle
  cycle -->|wave report| report
  report -->|records| memory
  merge -->|ready for prod| gate
  gate --> prod
  prod -->|prod signal: new intent.md| idea

  coder -.->|1. question, in its outcome| cycle
  cycle -.->|relays| qlead
  qlead -.->|2. touches product| qpm
  qpm -.->|2. to the owner| qowner
  qlead -.->|3. critical| block
  block -.->|4. owner present| qpm
  block -.->|4. owner absent| planrev
  planrev -.->|5. adjusts the plan| qlead
  qlead -.->|5. task unblocks| cycle
```

## Reading the graph

- The only subagents are `gabo-pm`, `gabo-dev-lead`, `gabo-db-master`, `gabo-frontend`, `gabo-plan-reviewer` and one `gabo-coder` per task. `gabo-cycle` is a skill run by the main session, and the reviewer is a script, not an agent.
- No edge goes from one gabo agent to another. A coder question travels in its outcome to the main session, which relays it (dotted path).
- Floating questions never stop the wave: only the task that depends on the answer is BLOCKED.
- The production gate belongs to the owner; the agent never crosses it.
