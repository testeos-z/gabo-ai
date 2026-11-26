# Spec 0032 Intent landing branches

| field | value |
|---|---|
| intent | [0032-intent-landing-branches](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-26 |

Mode is `auto`. The owner's words below are quoted verbatim from `odd/tasks/intent-landing-branches.md` (S1–S3, Spanish, spelling kept as stated). The L4 decisions are the owner's, relayed in that file's Log; every other decision carries a **Proposed (auto)** mark with a one-line reason. No agent accepts a spec: section 7 is the owner's.

## 0. Terms and evidence

Terms used below:

- **Main development branch:** the branch in `repo.integration_branch` (`gabo.yaml:12`, `dev` today) where finished intents pour in.
- **Intent branch:** the landing branch of one intent, named exactly like the intent folder (e.g. `0032-intent-landing-branches`), cut from `dev` by `gabo-cycle` when the wave opens.
- **Reviewer gate:** the automatic reviewer's verdict + auto-merge path (reviewer scripts + `review.auto_merge`) that merges clean heads today.

Owner statements, verbatim (the requirements in section 1 encode them and reopen none):

- S1, verbatim: "ahora cada Intent tendra su propia rama de aterrizale, los agentes podran hacer auto merge a la rama de aterrizaje principal."
- S2, verbatim: "1- la de desarrollo principal ejemplo dev. ahi los intent verteran sus cambios al terminar."
- S3, verbatim: "2- rama por Intent donde esta rama funcionara como funciona la rama de trabajo ahora sera donde se acumularan los cmabios de un intent."
- S5 context, summary (not a quote): two intents are raised in parallel, one for the fix (#908) and one for this improvement; the fix half is intent 0031, created by another worker.
- L4 decisions, summary (owner's, from the Log): the intent PR into `dev` is merged by the automatic reviewer under the same gate; no new `gabo.yaml` key ("tengo duda se supone quie el gabo.yaml esta recomendado como no tocar eso causaria que el flujo se bloquee, creo que lo mejor seria algo homonimo que la rama se llame igual que el intent"); the landing branch name is the intent folder name; `gabo-cycle` creates it from `dev` when the wave opens; intent branches get the same ruleset as `dev`.

Evidence and impact map (from exploration, paths as found):

- Embedded catalog: `apps/cli/internal/skills/gabo-{cycle,coder,assistant,reviewer,reviewer-placement-local,contract-repository-host,reviewer-notes,agent-templates,agent-host-railway-orca}`.
- Reviewer scripts (`gabo-reviewer/scripts/`): `common.sh` (`BASE_BRANCH` 121–122; `main`/`master` refusal 379–381), `reviewer.sh` (115, `pr list --base`), `review-head.sh` (72, `baseRefName` check; 97–165, test merge), `merge-head.sh` (78–83, base gate).
- Driven adapters: `repository_host` and `tasks`.
- Pinned branch expectations: Go tests pinning `origin/dev` (`coder_test.go`, `cycle_test.go`, `agenttemplates_test.go`, `protectbranches_test.go`, `setupprotection_test.go`) and script-test fixtures.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The flow shall offer two landing-branch kinds: the main development branch (`repo.integration_branch`, e.g. `dev`) and one branch per intent. Owner S1, verbatim: "ahora cada Intent tendra su propia rama de aterrizale, los agentes podran hacer auto merge a la rama de aterrizaje principal." | Given an open intent with task PRs, When its work accumulates, Then the PRs target that intent's branch, not `dev` directly. Given `repo.integration_branch: dev`, Then `dev` remains the main landing branch. | text test of the branch-kind rule in the owning skill; read-back of one intent branch and its base |
| S2 | Finished intents shall pour into the main development branch. Owner S2, verbatim: "1- la de desarrollo principal ejemplo dev. ahi los intent verteran sus cambios al terminar." | Given an intent whose tasks are done, When it finishes, Then one intent PR targets `dev`. Given an unfinished intent, Then no intent PR into `dev` exists. | skill text test of the intent-PR rule; read-back of the intent PR base on the first intent that uses this flow |
| S3 | Each intent branch shall work as the working branch works today: it is where that intent's task changes accumulate. Owner S3, verbatim: "2- rama por Intent donde esta rama funcionara como funciona la rama de trabajo ahora sera donde se acumularan los cmabios de un intent." | Given an intent branch, When task PRs for that intent merge, Then they merge into the intent branch and the reviewer auto-merges clean heads there. Given a task of another intent, Then its PR does not target this branch. | reviewer script test with fixture PRs per base; probe on one intent branch with two task PRs |
| S4 | The intent branch shall be named exactly like the intent folder (e.g. `0032-intent-landing-branches`); `gabo-cycle` creates it from `dev` when the wave opens; it gets the same ruleset as `dev`; no new `gabo.yaml` key is added; `main`/`master` stay refused (L4). | Given intent `0032-intent-landing-branches` and a wave opening, When the wave opens, Then a branch named `0032-intent-landing-branches` exists cut from `dev` and no `gabo.yaml` key was added for it. Given the ruleset on `dev`, Then the same ruleset covers the intent branch. Given a PR targeting `main` or `master`, Then it is refused as today. | text test that no new key exists in `gabo.yaml` and its schemas; test of the branch-name rule (folder name = branch name); ruleset read-back on the first intent branch |
| S5 | The intent PR into `repo.integration_branch` shall be merged automatically by the reviewer whenever it meets the gate rules and leaves no conflicts (owner: "no puede ser automatico siemrpe que cumpla las regla y no deje conflcitos"). Human-only paths were already approved by a human when each task PR merged into the intent branch, so the intent PR does not hold on them again (Proposed, see D6). | Given an intent PR with a clean verdict, no conflicts and no eval hold, When the merge step runs, Then it merges pinned to the head. Given a failing verdict or a conflict, Then it does not merge. Given human-only paths already merged into the intent branch through a human-merged task PR, Then the intent PR still merges automatically. | merge-step script tests with intent-PR fixtures (clean, failing, conflict, human-only already approved) |
| S6 | Every agent shall resolve the intent branch without depending on one host or tracker (owner: "recuerda que debe ser configurable", GitHub, GitLab, Trello or any mix). The source of the name is the intent folder in the repository, `docs/product/intents/<NNNN-slug>/`, whose name is the branch. `gabo-cycle` passes the folder in each coder handoff. The coder checks it through the `tasks` slot contract operation that returns the parent intent folder of a task (new contract obligation; each adapter implements it: GitHub sub-issues, Trello card link or label, etc.) and opens no PR when they differ. The reviewer accepts a PR base only when it is `repo.integration_branch` or an intent folder name that exists under `docs/product/intents/` on `repo.integration_branch`, read from the repository; any other base, `main` and `master` are refused. Skills name contract operations, never `gh`. | Given a wave for intent folder `0032-intent-landing-branches`, When a coder resolves its base, Then it is `0032-intent-landing-branches`. Given a `tasks` adapter whose parent-intent read returns another folder, Then the coder opens no PR and reports the mismatch. Given a PR base `0099-unknown` with no such folder on `dev`, Then the reviewer skips it without verdict or merge. Given base `main`, Then it is refused. | skill text tests (`gabo-cycle`, `gabo-coder`, `gabo-contract-tasks`: parent-intent-folder operation; no `gh` in those skills); reviewer script tests with bases `dev`, existing folder, unknown, `main` |
| S7 | When an intent's tasks are all done and merged into its intent branch, `gabo-cycle` shall pour it into `repo.integration_branch` through `repository_host` contract operations only: open PR (MR on GitLab) from the intent branch to `repo.integration_branch` referencing the intent, request review (same request form as task PRs), and on a merged verdict mark the intent done through the `tasks` contract and delete the intent branch under the merged-only rule. A blocking verdict opens a fix round on the intent branch within the existing fix-round cap. Coders never open the intent PR. | Given an intent with one open task, When the wave closes, Then no intent PR exists. Given all tasks merged, Then exactly one intent PR targets `dev`. Given a clean verdict and merge, Then the intent is done and its branch deleted. Given a blocking verdict, Then a fix round runs and the branch is kept. | `gabo-cycle` skill text test (trigger, contract operations, no host CLI); probe on the first intent using this flow |
| S8 | When a project has no intent folder for the work, the agent that plans the intent (`gabo-pm`) shall create `docs/product/intents/<NNNN-slug>/` before the wave opens (owner: "que el agente cree esa estructura de carpetas"), so every intent has a folder and so a branch name. | Given a new intent and no folder, When it is planned, Then the folder exists on `repo.integration_branch` before `gabo-cycle` cuts the branch. | `gabo-pm` and `gabo-cycle` skill text tests (folder before branch) |
| S9 | Before the intent PR into `repo.integration_branch`, conflicts shall be resolved by a reverse merge, never by a direct change to `repo.integration_branch` (owner: "primero se crea una rama nueva basada de intent, luego enviar merge a la rama basada en intent desde dev, ... ahi se arregla el conflicto, luego se pasa a la rama del intent y de ahi directo a dev"): cut `<NNNN-slug>-sync` from the intent branch, merge `repo.integration_branch` into it, resolve conflicts there, PR `<NNNN-slug>-sync` into the intent branch through the normal review, then the intent PR into `repo.integration_branch`. Never rebase or force-push. | Given `dev` moved after the intent branch was cut, When the intent finishes, Then a `-sync` branch with a merge of `dev` is reviewed into the intent branch before the intent PR. Given no divergence, Then the sync step is skipped. | `gabo-cycle` skill text test (order, no rebase/force-push); probe with a fixture conflict |
| S10 | Several intents may run at once, each with its own branch and its own wave; they pour into `repo.integration_branch` one by one, each after its own S9 reverse merge (owner: "3 intent, cada uno con su rama y su wave, luego se van fusionando a dev"). | Given three open intents, Then three intent branches and three waves exist; When one merges into `dev`, Then the next runs S9 against the new `dev` before its intent PR. | `gabo-cycle` skill text test; probe with two intents |

## 2. Design

The change repoints the existing flow at two branch kinds instead of one: `gabo-cycle` cuts the intent branch from `dev` at wave open; task PRs target it and the reviewer auto-merges them there; the intent PR targets `dev` and the same reviewer gate merges it. No configuration key is added; the branch name carries the intent identity.

Affected components (final paths are the plan's): `gabo-cycle` (cut the intent branch from `dev` at wave open); `gabo-coder` and `gabo-assistant` (target task PRs at the intent branch); `gabo-reviewer`, `gabo-reviewer-placement-local`, `gabo-reviewer-notes` and the reviewer scripts (`common.sh`, `reviewer.sh`, `review-head.sh`, `merge-head.sh`: base-branch handling per kind, `main`/`master` refusal kept); `gabo-contract-repository-host` plus the `repository_host` and `tasks` adapters (branch and PR operations per kind); `gabo-agent-templates` and `gabo-agent-host-railway-orca` (prompts/host paths that name the working branch); Go tests pinning `origin/dev` and script-test fixtures (updated to cover both kinds).

Every decision below is **Proposed (auto)** unless it names an owner decision; the owner's acceptance of this spec confirms or changes each.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | How the intent branch is named | (a) exactly the intent folder name; (b) a new `gabo.yaml` key holding the branch name | (a), owner decision L4: no new key, homonymous with the intent. Reason: the owner judged a new key as breaking the recommended flow | The plan changes no config schema; the folder name is the single source of the branch name |
| D2 | Who creates the intent branch and from where | (a) `gabo-cycle` from `dev` when the wave opens; (b) the first coder on demand; (c) a human by hand | (a), owner decision L4. Reason: one deterministic creation point per wave | The wave record can cite the cut commit; no task waits for a branch |
| D3 | Protection of intent branches | (a) protect each intent branch like `dev`; (b) lighter rules | (a), owner decision: "si proteger cada rama de intent". Each intent branch gets: changes only through PRs, no force-push, the `reviewer/tests` and `reviewer/review` statuses required; applied through the `repository_host` contract operation `protect branches`, adapted to the user's stack (GitHub rulesets, GitLab protected branches, ...). Deletion is allowed only by S7 after the merge into `repo.integration_branch`, so the deletion rule is lifted for the intent branch at that step | `protect branches` covers intent branches at creation and releases them at deletion |
| D4 | Who merges the intent PR into `dev` | (a) the automatic reviewer under the same gate; (b) a human only | (a), owner decision L4. Reason: the finished intent pours like any clean head | No new merge path; the verdict rules are unchanged |
| D6 | Human-only paths on the intent PR | (a) already approved at task-PR level, intent PR merges automatically; (b) the intent PR holds again for a human | (a) Proposed (auto), from owner point 5. Reason: the human approval already happened on the task PR; holding again would make almost every intent merge manual | Owner confirms or changes at acceptance |
| D5 | `main`/`master` handling | (a) stay refused as landing targets; (b) allow them for intent branches | (a), following intent 0029 and the existing `common.sh` refusal. Reason: the protection intent already decided this | The refusal tests stay green and gain intent-branch cases |

## 3. UX and app flow (optional)

Not applicable: the change has no user-facing surface. The owner sees branches and PRs on the repository host as today, one branch per intent plus `dev`.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data. Branches and PRs live on the repository host, not in a migrated store.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| `gabo.yaml` recommended-flow guidance (owner: no new key) | A branch-per-intent flow could suggest a new key | No conflict: D1 adds no key (S4) |
| Intent 0029 (branch protection on `main` and `dev`) | Intent branches need a ruleset | Resolved by D3: same ruleset as `dev`; `main`/`master` stay refused |
| Reviewer as merge gate (PRD R9) | Two branch kinds could suggest two gates | No conflict: D4 keeps one gate for both kinds |
| Test-first for behavior changes (`AGENTS.md`) | Branch targeting is behavior | S1–S10 each name their test layer; pinned `origin/dev` tests are updated, not deleted |
| Reviewer scripts use `gh` directly | S6/S7 are host-agnostic, the bundled reviewer is GitHub-only | Not changed here: the local reviewer stays a GitHub adapter; other hosts need their own `reviewer_placement` adapter (recorded limitation) |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent open question 1 (which questions the spec answers) | answered: S1–S3 encode the owner's verbatim statements; S4–S5 encode the L4 decisions; the S5 two-intent ordering is context, not a requirement of this intent |
| 2 | Intent open question 2 (ordering with T2 local-workflow change and sibling intent 0031) | carried to plan: T2 changes the local workflow first; 0031 runs in parallel by another worker; neither blocks this spec |
| 3 | Do the reviewer script line numbers cited in section 0 drift before the plan lands? | carried to plan: the plan re-checks `common.sh`, `reviewer.sh`, `review-head.sh` and `merge-head.sh` against the cited lines before editing |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner, alias Leo) | 2026-11-26 | Acceptance of the spec as written in the main session. Owner, verbatim (Spanish, English gloss in brackets): "capto los spec," [I accept the specs]. Context: the assistant asked for acceptance of the 0031 and 0032 specs, including the proposed D6 of 0032. Not covered: the intent and the plan. |

## A1. Amendment (recorded 2026-11-26, implements the accepted D6)

Reviewer finding on PR #914: `docs/cloud-agent-flow.md` hard rule 5 says
merges happen "never for human-only paths", which contradicts the accepted
D6 (the intent PR does not hold again on human-only paths already approved
at task-PR level). Resolution, faithful to the owner's acceptance of D6:

- Hard rule 5 is amended with the D6 exception (see the flow doc): a
  proven intent pour into `dev` may merge despite configured
  `review.human_only_paths`; the reviewer's built-in paths still hold.
- The proof rule refines S5/D5 without changing them: every commit in the
  pour diff must be reachable from `dev`, or associated with a merged PR
  whose base is the intent branch. Unproven commits keep the hold.
- The built-in paths are exactly `.gabo/VERSION`, the `.gabo/skills/`
  subtree, and the reviewer's own source under `skills/gabo-reviewer/`
  (the `is_builtin_human_only` set); the flow doc names them verbatim.
- Transitivity: a merged PR based on the intent branch that contains
  human-only paths can only have been merged by a human, because the
  gate holds such PRs for a human merge. Proof by merged PR therefore
  transitively proves human approval of those paths.
- Acceptance of the exact text: the hard-rule-5 wording above ships in
  PR #914, which touches a human-only path and can therefore only be
  merged by a human; that human merge is the acceptance of the exact
  text required by the reviewer finding.
