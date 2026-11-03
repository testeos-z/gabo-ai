# Per-change AI-Native SDLC model for gabo

Objective: replace gabo's upfront global planning with the per-change intent → spec → plan model (AI-Native SDLC playbook), with one wave per intent. Branch: `feat/per-change-sdlc` (from `origin/dev` a7118a1). Delivery: ask-on-risk. Test runner: `gabo check` plus the repo's existing checks (see T1 map).

## Specs

Owner quotes are verbatim (Spanish), with an English gloss after each.

- **S1. Use SDLC terms, not bashebr names.** "usaremos bashebr como guia del modelo pero no adoptaremos sus nombres ni terminos, pero si los terminos de SDLC." Artifacts are named `intent.md`, `spec.md`, `plan.md` and review findings.
- **S2. Conceptual PRD.** "PRD: ahora sera un documetno conceptual de que ofrece el producto y que optiene el usuario." (The PRD only describes what the product offers and what the user gets.)
- **S3. Conventions-only TRD.** "TRD: solo contendra convenciones globales" (The TRD only holds global conventions.)
- **S4. No upfront DB schema.** "el schema de la base dedatos no lo tendremos al iniciar,porque chocaria mucho en cada iteracion, es mejor que si hacemos cambios en la db con migraciones o scripts quedan documetnado por pasos como migraciones." (No DB schema at the start; DB changes are documented step by step as migrations or scripts.)
- **S5. UX per intent.** "el tenma de UX/UI sera opcional si la aplicaicon lo amerita, los estilos los podemos definir globales pero el UX y app flow seria por intent." (UX/UI is optional. Styles may be global, but UX and app flow are per intent.)
- **S6. User picks the stack.** "mantendremos la flexibilidad de que el usuario elija sus tecnologuias y el asistentele ayude a configurarlo." (The user chooses the technologies and the assistant helps configure them.)
- **S7. Intent template and control folder.** "necesitamos un template para los intent.md y la carpeta que lleve el control".
  - Assumption (assistant recommendation, not contested): the folder is `docs/product/intents/<NNNN-slug>/` and holds `intent.md`, `spec.md`, `plan.md` and review findings.
  - Ownership: `gabo-pm` writes `intent.md` and validates `spec.md`. `gabo-frontend` and `gabo-db-master` contribute UX/flow and migrations to the spec. `gabo-dev-lead` writes `plan.md` and the tasks.
  - Stage control is deterministic (`gabo check` or a hook), not an agent.
- **S8. Tracker hierarchy.** "el tablero puede tener varios intents asociados porque unatarea a nivel de producto puede tener varios cambios de software o tecnicos asociados y al definir el intent lo documetnamos como una sub atera o items en checklist con referencia donde se guarda y documenta en el repo"
  - A product card has N intents. An intent has 1 spec, 1 plan and N tasks.
  - A task belongs to one intent. `blocked-by` links may cross intents.
  - The `tasks` contract gains a parent.
- **S9. No global implementation plan.** "me parece bien, y asi ebitamos modelo cascada". This answers the proposal to drop `IMPLEMENTATION_PLAN.md` and make the tracker the global view.
  - `plan_change.scope`: `task` means one task in the intent's `plan.md`; `group` means a whole intent's spec and plan; `global` means several intents or the PRD/TRD.
  - Rewrite R7's "global plan" wording to match.
- **S10. No waterfall.** "no ya no manejaros un modelo de cascada". There is no batch of accepted intents. Specs are accepted one at a time.
- **S11. One wave per intent; chaining is configurable.** "el modelo de ola en cascada es opcional puedes dejar una sola ola en la nube mientras regresas y cuadn esa ola termine segun que configures toma la siguietne o espera. la ola sera un intent". (Chaining waves is optional. One wave can run in the cloud while the owner is away. When it ends, it either takes the next intent or waits, depending on configuration. Each wave is one intent.)
- **S12. Methodology slot stays open.** "gabo no te obliga a usar openspac o SDD". `spec.md` is per change, and the methodology slot (ODD or another) still governs how a coder builds a task.
- **S13. Gabo helps configure projects.** "que gabo ayude a configurar el proyecto apra esto". Setup or configuration guides a project into this model.
- **S14. Spec acceptance and production gate stay human.** Playbook: "A human team mate always makes this call".
- **S15. Pi refusal recommends Gentle AI.**
  - Owner, verbatim (voice transcript): "Ok, recomendaremos \"Piellen\" como subagentes y me parece bien que se llega a trabajar si no tiene un paquete subagentes. Si no lo tienen, recomendaremos todo el marco de \"Genteel AI\"."
  - Correction, verbatim: "no gavos sino tiene un paquete para su pagente se va a negar a trabajar en pie y en por lo menos le va a decir mira te recomiendo no tiene su agentes no puede trabajar si necesitas te recomiendo usar este"
  - Meaning: there is NO fallback. Without a subagent package, gabo refuses to build on Pi; this is already the behavior. The refusal recommends gentle-pi for subagents and the full Gentle AI framework.
- **Out of scope.** "los hook y del layout <slug>/. de ghiaog123 podemops evaluarlos". The ghiaog123 hooks are evaluated only, not implemented here.

## Tasks

| id | specs | route | status | commit |
|---|---|---|---|---|
| T1 | all | delegated explorer (mapping trigger) | done (map in L4) | — |
| T2 (A) | S1 S4 S5 S7 S9 | Intent folder templates: intent/spec/plan, drop STORY and IMPLEMENTATION_PLAN, DB_SCHEMA → migrations log, APP_FLOW/UX_UI per intent and optional. `SRC/gabo-{pm,dev-lead,db-master,frontend}/templates/*`, `SRC/templates_test.go` (templates part). Route: delegated writer, parallel with T4 and T5. | done; DESIGN.md replaces UX_UI, and per-intent UX/flow is spec section 3 | 2489a54 |
| T3 (B) | S7 S8 S9 | Planning skills ownership, new traceability gap list (task→intent), dev-lead parent = intent. Files: four planning SKILL.md, gabo-doc-guide, `templates_test.go:330-357`, `skills_test.go:237`. Depends on T2. Route: delegated writer (parallel with T6 and T7); also owns the PRD/TRD templates and the four agent templates. | done; parent applied the skills_test doc-guide pin, gabo-assistant gates and agents.yaml roles; full suite, vet, gofmt and e2e pass | c01e10b |
| T4 (C) | S8 | Tasks contract: parent intent and intent filter. `SRC/gabo-contract-tasks/*`, `.claude/skills/gabo-tasks-github/SKILL.md`, coverage tests. Risk: item 3 (contract consumed by adapters). Route: delegated writer, parallel. | done; no new heading; check tests green | e3949ef |
| T5 (D) | S9 | plan_change scopes redefined: plan-reviewer skill and agent template, `.claude/agents` rebuild, APP_FLOW scale table, `skills_test.go:715-740`, schema description. Risk: item 3. Route: delegated writer, parallel. | done; RED→GREEN for the scale and process tests | baf47c5 |
| T6 (E) | S11 | One wave = one intent, plus an after-wave key (next or wait): both schemas, config Go, the example, gabo-cycle, gabo-assistant, skills_test wave part, e2e wave test, `gabo.yaml`. Depends on T4. Risk: item 3 (config format). Route: delegated writer (runs parallel with T2). | done; RED→GREEN in config and skills, e2e Wave passes, default wait | 1b36a39 |
| T7 (F) | S7 | Deterministic intent-stage check in `gabo check`. `slices/check/{intents.go, usecase.go, command.go, tests}`. Depends on T2. Route: delegated writer (parallel). | done; RED→GREEN on 13 fixtures; differential identical against a control build | df6fcd2 |
| T8 (G) | S6 S13 | Setup guides stack choice and the intent model. `SRC/gabo-setup/SKILL.md`, `setup_test.go`, both setup-prompt copies. Depends on T2 and T6. Route: delegated writer (parallel with T9); also updates the gabo-cycle agent role. | done; RED→GREEN on setup and prompt tests; full suite ok | 3fc1017 |
| T10 | S15 | The Pi refusal message also recommends the full Gentle AI framework. No fallback. Route: delegated writer. Files: the gabo-runtime-pi skill, runtimepi_test, the README and the setup prompts. | done; RED→GREEN; full suite ok; gentle-pi 4.5.1 tools verified | see the T10 commit |
| T9 (H) | S2 S3 S4 S10 S14 | Product docs: conceptual PRD (R3/R4/R7/R19/R27), conventions TRD (5.1, D7, D19), retire IMPLEMENTATION_PLAN, DB_SCHEMA and stories, README. May need two slices. Depends on T3, T5 and T6. Route: delegated writer (parallel with T8). Retired docs are archived under docs/product/archive (assumption). | done; 0 broken links; `gabo check` passes with an intents README and rejects a bad folder | 37a4047 |

## Log

- **L1 (2026-11-03), owner request, verbatim:** "apliquemos manual estos cambios a este repo, que luego usaremos para esta iudea para implementarla que gabo ayude a configurar el proyecto apra esto, el modelo de ola en cascada es opcional puedes dejar una sola ola en la nube mientras regresas y cuadn esa ola termine segun que configures toma la siguietne o espera. la ola sera un intent"
- **L2.** "manual" means the owner authorized applying the changes directly, outside gabo's own plan-change and wave flow. The S1–S13 decisions come from the 2026-11-03 conversation, and the owner is present.
- **L3.** The branch was created from `origin/dev`. Local `dev` was 2 commits behind and was fast-forwarded. Untracked `.gabo/` and `.vulcam/` and the modified `.atl/*` files are preserved and not part of this feature.
- **L4. T1 map findings.**
  - The tracked source for the skills is `apps/cli/internal/skills/` (SRC). `.gabo/skills` is an untracked copy made by `gabo init`.
  - `.claude/agents` are built from `SRC/gabo-agent-templates`.
  - Traceability is skill prose pinned by `templates_test.go`. `gabo check` does not validate docs today.
  - The plan-reviewer scale table must match APP_FLOW row for row.
  - The schema exists twice (`schemas/` and `config/schemas/`).
  - There is no CI. Checks are `go -C apps/cli build/vet/test ./...` plus gofmt and e2e (`-tags e2e`).
  - No Plane or Trello adapter exists.
- **L5. Size forecast and delivery.** The forecast is about 2,450 authored lines, which is over 400. Delivery is ask-on-risk, so the chain strategy is asked once.
- **L6 (2026-11-03). Owner chose "Feature branch chain".**
  - Each slice PR targets `feat/per-change-sdlc`, and one final PR goes from feat to `dev`.
  - The parent owns git and commits each unit.
  - `skills_test.go` is owned by T5 during the parallel phase. T4 reports any `skills_test.go` change it needs to the parent.
- **L7.** T5 is committed (baf47c5). `.claude/agents` has no CLI rebuild, so it was edited by hand to match the template. T4 is committed (e3949ef): an optional `parent` on `create` and an `intent` filter on `read unblocked`, with no new heading so adapters stay covered. T6 started, using `wave.after` (next|wait, default wait as an assumption).
- **L8.** T2 is committed (2489a54). T3 and T7 started alongside T6. T6 owns `skills_test.go` and `gabo-assistant`; T3 reports its edits to those files to the parent.
- **L9.** T6 is committed (1b36a39). Follow-up for T8: `gabo-agent-templates/agents.yaml:47`, `agents/gabo-cycle.md:5` and `.claude/agents/gabo-cycle.md` should name the intent.
- **L10.** T7 is committed (df6fcd2). The check is skipped when `docs/product/intents/` is absent. The writer's scratch directory was removed.
- **L11.** T3 is committed (c01e10b). Seam check after T2–T7: `go test ./...`, vet, gofmt and e2e all green. T8 and T9 started.
- **L12.** T8 is committed (3fc1017) and T9 is committed (37a4047). The final seam check passes: `go test -count=1 ./...`, e2e, vet and gofmt. RDD is on: assess origin/dev..HEAD (untracked excluded) gives risk medium, review_due false (under_budget), so the writers' self-verification stands.
- **Follow-ups (not in scope):**
  - `gabo.yaml:3`, the schema descriptions, the config Go comments and the agent-host skills still cite DB_SCHEMA; repoint them to TRD 9.3 or MIGRATIONS.
  - `docs/product/diagrams/*` still shows the old global-plan flow.
  - The ghiaog123 hooks are still to be evaluated.
- **Next:** the owner decides on push and on the PR feat/per-change-sdlc → dev. Old next note: apply T3's `skills_test.go` edits after T6, run the seam check (the full suite), then T8 and T9.
