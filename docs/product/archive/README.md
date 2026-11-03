# Archive

Retired product documents of gabo-ai, kept for history. Nothing here is maintained, and no agent plans from these files.

## Why they were retired

On 2026-11-03 gabo-ai moved from upfront global planning to the per-change model that gabo itself ships (feature `per-change-sdlc`, `odd/tasks/per-change-sdlc.md`). Each change is now an intent in [`../intents/`](../intents/README.md), with its own `intent.md`, `spec.md`, `plan.md` and review findings. The tracker is the global view of the work. With that model, these upfront documents no longer apply:

| file | replaced by |
|---|---|
| [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) | One `plan.md` per intent. There is no global plan, roadmap or milestone list. |
| [DB_SCHEMA.md](DB_SCHEMA.md) | No upfront schema. The configuration file shapes now live in [TRD 9.3](../TRD.md#93-configuration-shapes), and data or config changes are logged step by step in [MIGRATIONS.md](../MIGRATIONS.md). |
| [stories/](stories/README.md) | The requirements and acceptance criteria of each change, in its `spec.md`. |
| [PRD-0.10.md](PRD-0.10.md) | The conceptual [PRD](../PRD.md). This copy keeps the full acceptance criteria, the stories column and the dated amendment history. |
| [TRD-0.9.md](TRD-0.9.md) | The conventions-only [TRD](../TRD.md). This copy keeps the requirement coverage table. |
| [APP_FLOW-0.7.md](APP_FLOW-0.7.md) | The trimmed [APP_FLOW](../APP_FLOW.md). This copy keeps the auto-mode proposal history and the superseded steps. |

Links inside the archived files point to the archived copies, so section references (for example `DB_SCHEMA 3.2` or `PRD 10.1` as they were) still resolve.

## Assumption

The files are archived rather than deleted, so the history stays readable without git and the references that code comments and schemas still make (for example `DB_SCHEMA 3.2` or `DB_SCHEMA 4.2`) can be followed. This is an assumption of the `per-change-sdlc` feature, not an owner decision; deleting the folder later changes no behavior.
