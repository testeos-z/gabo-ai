# Intent 0032 Intent landing branches

| field | value |
|---|---|
| intent id | 0032-intent-landing-branches |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#909 I0032: Intent landing branches](https://github.com/testeos-z/gabo-ai/issues/909) |
| date | 2026-11-26 |

## Problem

Today every task PR targets the same working branch (the flow that `gabo-cycle` and the reviewer scripts implement around `origin/dev`), so the work of different intents accumulates in one place with no per-intent boundary. There is no branch where one intent's task PRs gather on their own before the finished intent pours into development.

The owner asked for one landing branch per intent, in the owner's words (L1, Spanish verbatim, kept as stated including spelling):

- S1, verbatim: "ahora cada Intent tendra su propia rama de aterrizale, los agentes podran hacer auto merge a la rama de aterrizaje principal."
- S2, verbatim: "1- la de desarrollo principal ejemplo dev. ahi los intent verteran sus cambios al terminar."
- S3, verbatim: "2- rama por Intent donde esta rama funcionara como funciona la rama de trabajo ahora sera donde se acumularan los cmabios de un intent."

The owner also ordered (L2, S5) two intents in parallel — one for the #908 fix and one for this landing-branch improvement — so this intent is the improvement half of that pair; the fix half is intent 0031.

## Who is affected

- The owner, who today cannot see one intent's accumulated work apart from every other intent's work on the same branch.
- Every coder and the assistant, whose task PRs all pour into the same branch with no per-intent landing place.
- The reviewer, whose auto-merge gate must now serve two landing-branch kinds (the main development branch and one branch per intent) instead of one.

## Desired outcome

- Each intent has its own landing branch where its task PRs accumulate, and finished intents pour into the main development branch (`dev`).
- Agents can auto-merge to the main landing branch, and the intent PR into `dev` is merged by the automatic reviewer under the same gate (L4).
- Reverse merge contra `dev` siempre antes del intent PR: se corta `<NNNN-slug>-sync` desde la rama del intent, se fusiona `dev` en ella, se resuelven los conflictos ahí, se lleva a la rama del intent por la revisión normal, y de ahí el intent PR directo a `dev`.
- No new `gabo.yaml` key is needed: the landing branch is named exactly like the intent folder (L4).

## Constraints

- No new `gabo.yaml` key (L4: the owner rejected a new key as breaking the recommended flow; the branch name is homonymous with the intent folder, e.g. `0032-intent-landing-branches`).
- `main`/`master` stay refused as landing targets (intent 0029 branch protection; reviewer `common.sh` refusal).
- Intent branches get the same ruleset as `dev` (L4).
- Planning only: this change writes `intent.md` and `spec.md`; no code, no task issues. The plan (`plan.md`) and the D6 hard-rule amendment to `docs/cloud-agent-flow.md` land together with the spec amendment A1 in PR #914.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which fixed questions does the spec answer, and which stay open? | carried to spec |
| 2 | Ordering with the local-workflow change (T2) and the parallel fix intent 0031: does either block this spec? | carried to spec |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (parents I0030 [#767](https://github.com/testeos-z/gabo-ai/issues/767); this intent's issue goes under the same card) |
| sibling intent (fix half of S5) | intent 0031 (fix for [#908](https://github.com/testeos-z/gabo-ai/issues/908)), created in parallel by another worker |
| sibling intent, prior art | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md) (reviewer gate), [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md) (`main`/`master` refusal) |
| related intent | [0030 Agent evals when skills change](../0030-agent-evals-when-skills-change/intent.md) (structure copied from this folder: `intent.md` + `spec.md`) |
| PRD | [PRD](../../PRD.md) |
| TRD | [TRD](../../TRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) (landed together with spec amendment A1 in PR #914) |
| source | `odd/tasks/intent-landing-branches.md` (S1–S6, T1–T5, L1–L5) |
