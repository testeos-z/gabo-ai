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
