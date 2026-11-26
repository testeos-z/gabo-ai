# Intent landing branches, #908 fix, release 0.6.0

Each intent gets its own landing branch that collects its task PRs; finished intents merge into `dev`.
Order: local workflow first, then two intents (fix + improvement), then release 0.6.0.

## Specs

- S1: "ahora cada Intent tendra su propia rama de aterrizale, los agentes podran hacer auto merge a la rama de aterrizaje principal."
- S2: "1- la de desarrollo principal ejemplo dev. ahi los intent verteran sus cambios al terminar."
- S3: "2- rama por Intent donde esta rama funcionara como funciona la rama de trabajo ahora sera donde se acumularan los cmabios de un intent."
- S4: "1. primeor cambviaremos en local la forma en que gabo trabaja las ramas para usar esa forma de trabajo con una rama por intent."
- S5: "2. levantarems dos intnet yno para el fix y uno para la mejroa." Fix = issue #908 "Automerge does not resume after a clean review check changes from pending to success".
- S6: "3. subiremos una version mas yo manejo las versiones de 2 en 2 y esta seran mejoras menores" — version confirmed by user: "0.6.0" (previous tag apps/cli/v0.4.2).

## Tasks

- T1 [S1-S4] Map current branch flow (skills, reviewer scripts, gabo.yaml, cycle) — explore — commit: n/a
- T2 [S1-S4] Change local repo workflow to one landing branch per intent (`intent/<NNNN>-<slug>` from `dev`, task PRs into it, intent PR into `dev`) — inline — commit: pending
  - T2.1 [S1-S4] `.agents/skills/gabo-repository-host-github/SKILL.md`: branch from the intent branch; open PR with base = intent branch (task) or `dev` (intent PR); merge gate accepts `dev` or an existing intent folder name, refuses main/master; delete branch for task and intent branches; protect branches also on each intent branch (and lift deletion rule at close); new `sync` reverse-merge steps (0032 S9).
  - T2.2 [S1-S4] `.agents/skills/gabo-tasks-github/SKILL.md`: drop "PRs target `dev`"; new op `read parent intent folder` (0032 S6) from the intent issue body line.
  - T2.3 [S1-S4] `.gabo/skills/gabo-cycle/SKILL.md`: wave open cuts `<NNNN-slug>` from `dev`, protects it, passes it in each handoff; worktrees from `origin/<NNNN-slug>`; wave close = sync branch reverse merge, intent PR into `dev`, review request, close intent, delete branch (0032 S7, S9, S10).
  - T2.4 [S1-S4] `.gabo/skills/gabo-coder/SKILL.md`, `gabo-assistant`, `gabo-reviewer-placement-local`, `gabo-reviewer-notes`, `gabo-contract-repository-host`, `gabo-contract-tasks`, `gabo-agent-templates/agents/gabo-coder.md`: base = intent branch from handoff, verify against parent intent folder, CONFLICTING merges `origin/<intent branch>`.
  - T2.5 [S1-S4] User-level reviewer scripts `~/.local/share/gabo/skills/gabo-reviewer/scripts/{common,reviewer,review-head,merge-head}.sh`: accept base `dev` or an intent folder existing on `dev`; test-merge against the PR's own base; list PRs of all accepted bases. Outside the project: needs owner permission per path.
  - T2.6 Try it: dry probe with a throwaway intent branch + one task PR (no merge to `dev`); record what worked and what broke in the Log as input for 0032 implementation.
- T3 [S5] Create intent 0031 (folder + issue): fix #908 — worker — commit: pending
- T4 [S5,S1-S3] Create intent 0032 (folder + issue): intent landing branches in the gabo product (L4 decisions) — worker — commit: pending
- T5 [S6] Release 0.6.0 (after T3/T4 delivered; push/tag are user decisions) — inline — commit: pending

## Log

- L1: "1. revisa issues que tenemos una mejra que hacer 2. ahora haremos un cambio sera una mejora sobre la rama de aterrizaje, ahora cada Intent tendra su propia rama de aterrizale, los agentes podran hacer auto merge a la rama de aterrizaje principal. osea tendremos 2 tipos de rama de tarriaje, 1- la de desarrollo principal ejemplo dev. ahi los intent verteran sus cambios al terminar. 2- rama por Intent donde esta rama funcionara como funciona la rama de trabajo ahora sera donde se acumularan los cmabios de un intent. ahora mismo solo listame los cambios que llevaremos"
- L2: "seran 3 intent ademas de eso 1. primeor cambviaremos en local la forma en que gabo trabaja las ramas para usar esa forma de trabajo con una rama por intent. 2. levantarems dos intnet yno para el fix y uno para la mejroa. 3. subiremos una version mas yo manejo las versiones de 2 en 2 y esta seran mejoras menores asi que pasaremos a 0.8.0"
- L3: User corrected version: "0.6.0".
- L4: Decisions: intent PR into dev is merged by the automatic reviewer (same gate). No new gabo.yaml key: "tengo duda se supone quie el gabo.yaml esta recomendado como no tocar eso causaria que el flujo se bloquee, creo que lo mejor seria algo homonimo que la rama se llame igual que el intent" -> landing branch name = intent folder name (e.g. `0032-intent-landing-branches`). gabo-cycle creates it from dev when the wave opens. Intent branches get the same ruleset as dev.
- L5: "espera levanta dos sub agentes y crea los dons intents" -> T3 and T4 now (intent 0031 fix #908, intent 0032 landing branches), via two parallel writers; T2 waits.
- L6: 0032 spec must be host/tracker agnostic: "recuerda que debe serconfigruable" (GitHub, GitLab, Trello mix) -> S6/S7 rewritten with slot contracts.
- L7: Owner answers: "1. que el agente cree esa estructura de carpetas." "2. ... siemrpe se debe hacer un merge reverso, primero se crea una rama nueva basada de intent, luego enviar merge a la rama basada en intent desde dev ... ahi se arregla el conflicto, luego se para a la rama del intent y de ahi directo a dev." "3. ahi debe ajustarce al stack del usuario" "4. ... 3 intnet, cada uno con su rama y su wave, luego se van fucionando a dev. seria solo resolver conflictos y ya." "5. no puede ser automatico siemrpe que cumpla las regla y no deje conflcitos." -> 0032 S8-S10 added; S5 human-only handling pending.
- L8: Owner: "sobre la 3 si proteger cada rama de intent, con esto clarfo crea los intent y specs" -> 0032 D3 updated (protect each intent branch, stack-adapted), S5 auto-merge per owner point 5, D5 human-only proposed.
- L9: Owner: "capto los spec," -> 0031 and 0032 specs Accepted (section 7 filled, 2026-11-26).
- L10: T2.1 done on dev (uncommitted): .agents/skills/gabo-repository-host-github/SKILL.md now targets task PRs at the intent branch, intent PR at dev; merge gate accepts dev or an existing intent folder, refuses main/master; sync reverse-merge op added; intent branches protected, deletion rule released at close.
- L11: T2.2 done on dev (uncommitted): .agents/skills/gabo-tasks-github/SKILL.md now says task PRs target the intent branch, intent PRs target dev; link PR carries one Refs per poured task; intent DONE only after its intent PR merged.
- L12: T2.3 done on dev (uncommitted): .gabo/skills/gabo-cycle/SKILL.md cuts the intent branch from dev at wave start, task branches from origin/<intent-branch>, handoff carries intent branch, CONFLICTING merges intent branch, drift via sync reverse merge; folder/branch mismatch stops the wave.
- L13: T2.4 done on dev (uncommitted): gabo-coder CONFLICTING merges origin/<intent-branch> re-derived via parent intent; contract host/examples accept intent branch base; reviewer-notes policy/test-merge/preflight read the PR base; assistant auto-mode merges only into the accepted base.
- L14: T2.5 done: user-level reviewer scripts need no change (fully parameterized by REVIEWER_BASE_BRANCH, only main/master refused); convention documented in gabo-reviewer-notes: coder exports REVIEWER_BASE_BRANCH=<intent-branch> for task PRs, default dev for the intent PR.
- L15: T2.6 probe done, all cleaned. Cut 0000-probe-landing from dev, task branch probe/task-1, PR #911 base=intent branch. REVIEWER_BASE_BRANCH=0000-probe-landing reviewer.sh head -> tests=success review=success VERDICT clean merge=merged (auto_merge true). Intent PR #912 to dev opened MERGEABLE, closed unmerged. Remote+local branches and worktree removed. T2.1-T2.5 edits stay uncommitted on dev for owner decision (direct commit or PR).
- L16: T6 round 2. PR #913 verdict blocking: mirror test failures fixed by syncing all 3 skill trees (Go pinned tests updated to intent-branch wording, host-github example). S6 gap closed in the user-level scripts: common.sh validate_base_branch (dev or NNNN-slug folder existing under docs/product/intents/ on dev, checked via gh api contents; else skip without verdict or merge), wired into review-head.sh + merge-head.sh; tests/base-branch.test.sh (10 checks: unit x4, merge x3, review x2) and pinned digests updated in merge-head.test.sh; whole script suite green. 0031 intent.md now links #910.
