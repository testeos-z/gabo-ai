# Intent 0027 Coders and product QA share one definition of done

| field | value |
|---|---|
| intent id | 0027-coders-follow-gentle-ai-odd-and-rdd |
| status | Narrowed to scope A on 2026-11-13, and changed again the same day by the owner's playbook decisions (second change; spec section 0 and its Log). The owner's one confirmation is for the spec only (decision 6); it does not accept this intent. The earlier acceptance of 2026-11-13 (spec section 7) is kept as the record. No plan, wave or code starts from this text. |
| owner | gabo-pm |
| scope | scope A of the four-intent cut of 2026-11-13: what done means and who checks it |
| sister intents | 0028 deterministic hooks (locks), 0029 branch protection, 0030 agent evals (written by a sister agent in parallel; cited by number, their folders land on dev separately) |
| product card | none open: [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) is closed, so no card was created; the intent issue is linked under #492 as #703 was |
| date | 2026-11-13 (narrowed to scope A; second change the same day with the playbook decisions) |

## Problem

The owner asked that the coders use gentle-ai as closely as possible to its ODD and its RDD. Owner, verbatim, relayed 2026-11-13 by the owner's local assistant: "quiero que los coder usen gentle-ai lo mas apegado a ODD y RDD".

On 2026-11-13 the owner reviewed the gaps between gabo and Anthropic's playbook and cut the work into four intents. Owner, verbatim, relayed by the owner's local assistant: "Me encaja ese corte, me parece bien. Prepara los documentos de intents y los issues" [That cut fits me, I think it is fine. Prepare the intent documents and the issues]. This intent is the first of the four. It keeps what done means and who checks it. The owner did not confirm the 16 open questions as they stood; the open questions are re-derived for this scope in the spec, section 6.

Scope A, in four parts:

- **(a) The coder follows its methodology.** The coder uses gentle-ai ODD and RDD through the methodology slot, and reports its evidence: RED and GREEN or the stated exception, the spec ids it covered, and the RDD outcome.
- **(b) Two gates on the coder's report.** Done gate: the coder runs the checks of `review.checks` and pastes their output. Report gate: a task does not close with an incomplete coder report. Owner, verbatim, relayed 2026-11-13 for the report gate: "esa sí la vamos a meter como una mejora. Que el coder tiene que reportar" [that one we will add as an improvement. The coder has to report]. Their enforcement mechanism and whether they block are open (spec, questions 26 and 27).
- **(c) The reviewer's main job is product QA.** It runs the product as a user would and checks each line of the task's done contract. It still reads the code, after the product QA (owner correction, 2026-11-13; the playbook's review also reads the code, so the criterion settles it).
- **(d) A review policy.** A versioned policy file, read from the integration branch and never from the PR head, sets three review passes, what counts as Important, and a cap of five nits. Decided 2026-11-13 by the owner's criterion: "si está en el documento de Anthropic, úsalo. Vamos a cumplir ese documento estrictamente" [if it is in Anthropic's document, use it; we will comply with that document strictly]. The read from `dev` is the owner's direction. Open: questions 22 and 23 (the missing-file rule and the other rule sources) and question 24 (which findings block) (spec S25 to S27).
- **(e) A fresh-context verifier before the PR.** Before a coder opens its PR, a verifier that starts without the coder's conversation exercises the changed behavior and the two nearest neighboring flows, and reports only to the coder. It never approves or blocks; the reviewer stays the gate (spec S28). Owner, relayed 2026-11-13 (verbatim): "no, ponlo antes, ponlo... no lo dejaremos afuera, porque yo veo que esto cumple opciones diferentes al revisor" [no, put it before, put it... we will not leave it out, because I see that it covers options different from the reviewer's].

Primary reference: Anthropic, [The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, August 21, 2026. Its verifier exercises the changed behavior and the two nearest neighboring flows without fixing anything, its builder pastes the output of build, test and lint before reporting done, and its REVIEW.md sets the review passes. Each fact in the spec's evidence base is marked verbatim (checked byte for byte against the fetched page) or summary.

Secondary reference: Anthropic, [Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps), the citation spec 0022 uses. The evaluator uses the running app as a user, and generator and evaluator agree beforehand on what done means.

### The coder side

Today a coder loads the methodology adapter, but the adapter is not followed. The evidence is in the repository and in the coder runs reported by the assistant:

- The adapter is wired in: `gabo.yaml:27-28` sets `slots.methodology.adapter`, and `.claude/agents/gabo-coder.md:8` preloads `gabo-methodology`. The coder skill says to load that adapter before reading code (`apps/cli/internal/skills/gabo-coder/SKILL.md:11`).
- The adapter already states the RDD rule: it runs only when `gentle-ai review mode status` is on, and the coder then runs assess on the commit and follows its transitions (`.claude/skills/gabo-methodology-gentle-ai/SKILL.md`).
- Reported by the assistant from session claude-b2e28fa5, not re-checked against transcripts: in 13 coder runs of waves I0023, I0025 and I0026 the adapter text was preloaded in all 13. One run ran `review start` and `status` and stopped on the per-candidate consent envelope, which a coder cannot answer. One run ran only `review mode status`. The other eleven ran no gentle-ai review command and reported no RDD outcome. No coder created an ODD feature document (`odd/tasks/`).
- The adapter's assess command does not work on committed work as written: on 2026-11-13, with gentle-ai 4.4.0 in a clean worktree, it returned `risk: high`, `reason: unassessable`, and an error that names `--base-ref` and `--committed-only`.

The result: the owner's RDD and ODD are not visible in any wave. A coder's report does not say whether RDD ran, and an incomplete report can close a task.

### The reviewer side

The gabo reviewer is the merge gate today, and its checks cover the diff, not the product:

- Its prompt asks for correctness bugs, security issues, architecture boundary violations and missing tests for changed behavior, and names `AGENTS.md` and `docs/product/` as the rules (`apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh:324-328`).
- Verified 2026-11-13 (spec, evidence base): those rules are read from the checkout of the PR head, not from the integration branch. The head checkout is created at `review-head.sh:164-168` and named at `:450-455`. A PR that edits `AGENTS.md` or `docs/product/` is reviewed against its own edited copy.
- It never runs the product, and it does not check any requirement's acceptance criteria. Accepted specs already state them, Given/When/Then, for example spec 0026 section 1; nothing in the review step reads them against a running product.
- Its mechanical gates are `review.checks` (`gabo.yaml:45-48`: `go -C apps/cli build`, `vet` and `test`, run on the merge result) and `review.auto_merge: true` (`gabo.yaml:52`).

Result: a clean verdict says the diff has no visible defect and the build passes. It does not say that a user can do what the spec says, and it does not say which rules it applied.

## Who is affected

- The owner, who wants the coders to build the way gentle-ai builds, who wants the reviewer to test the product the way a user does, and who needs to see both in each wave report.
- The assistant that runs each wave. It relays consent envelopes, reads the RDD outcome and the gate outcomes of every task, and relays the reviewer's product QA result.
- Every coder task of every wave. The gaps appear in each of the 13 runs reported above.
- The gabo reviewer, which stays the only merge gate (PRD R9). Its main job becomes product QA on the PR head, and it keeps a code read of every head under a policy the PR cannot edit.
- Every product change of gabo-ai, because a clean verdict will mean the product was run and not only read.

## Desired outcome

- Every coder task reports its ODD evidence (RED and GREEN, or the stated exception, with the spec ids it covered) and its RDD outcome: `granted`, `declined`, `passive`, `off`, `unavailable`, or gentle-ai's reason when no review was due. A missing item reads `not run: <reason>`, never as a pass.
- Every coder task writes its done contract before its first commit: one observable check per linked requirement (S12). The contract is the definition of done the reviewer tests against.
- Done gate and report gate: a coder's hand-off shows the pasted output of each `review.checks` command, and a task does not close with an incomplete report (S18, S19). Enforcement and blocking are the owner's decisions (questions 26 and 27).
- The coder's code is its own review's responsibility, with unit tests for each changed rule and integration tests where a task crosses a boundary (S15).
- For every pushed head that changes product behavior, the reviewer runs the build under test as a user would and reports each contract line as `pass`, `fail: <observed>` or `not run: <reason>` (S11). A head with no product change reads `not applicable`.
- The reviewer's comment shows the verdict, the product QA result, the checks result and the code read, in that order (S13). The code read applies the review policy: three passes, the Important definition and the nit cap (S14, S25 to S27). Every rule source it used is named.
- Every wave report shows the RDD outcome, the ODD line and the gate outcomes for each task, so the owner can count them without reading the coder transcripts (S9).
- A coder never answers a consent envelope, never toggles RDD, and never lowers a risk tier. A consent envelope reaches the owner through a route the owner decides (S4).
- The gabo reviewer stays the only merge gate (PRD R9). RDD, product QA and the gates add outside views and never merge anything (S5).
- Before a coder opens its PR, a fresh-context verifier runs the changed behavior and the two nearest neighboring flows and reports to the coder only (S28). It never approves or blocks; the reviewer stays the gate.
- Measure: the share of tasks whose wave report line carries an RDD outcome, per wave; and the share of product PRs whose reviewer comment carries a product QA result. Cost per PR rises against today; it is not yet measured and has no target (question 31).

## What moved out (2026-11-13)

| item | where it goes | spec ids (old) | note |
|---|---|---|---|
| Hook 1, shell guard | intent 0028 (deterministic hooks, locks) | S16, questions 12, 13, 15, 16 | |
| Hook 2, release gate | intent 0028 | S17, questions 17, 20 | |
| Hook 5, folder boundary | dropped by the owner | S20, question 21 | Owner, verbatim, relayed 2026-11-13: "La carpeta es la única que dejaremos fuera... cada coder debe trabajar en su propio work tree, entonces no veo el choque". Not in 0027 and not in 0028 by this decision. |
| The engine probe, the hook design rule, the engine events, the failure mapping, hook observability | intent 0028 | S21 to S24, D8 to D13 | Their full text and the engine evidence stay in git at `63b36df` (this file's history and the spec's), for the sister agent to read. |
| CI and package publishing | out of this cut; CI on PRs stays out as a separate item | none | Owner decisions 2026-11-13: out of scope. Question 9 option (b), moving the checks to CI, is therefore not offered. |
| Maintain stage: a small version to intent 0030 (owner decision 3); the rest (permissions and isolation, CLAUDE.md practices, metrics and traces, Design-stage security skills) is not in 0027 | 0030 (small part); none (the rest) | Not yet discussed with the owner for the rest. The fresh-context verifier moved in: S28 |
| Hook 3, coder done gate | stays in 0027 as a gate | S18 (rewritten), questions 18, 26, 27 | Its mechanism (cycle check, engine hook or reviewer) and whether it blocks are open. |
| Hook 4, methodology-report gate | stays in 0027 as a gate | S19 (rewritten), questions 19, 26, 27 | Same open questions as hook 3. |
| Test-file hook (owner decision 4) | intent 0028; it warns and never blocks | S29 reads its record | The playbook's primary mechanism blocks; the owner chose warn. Flagged in spec section 5 |

## Constraints

- PRD R8: the coder uses the methodology declared in the methodology slot. R9: two layers, an optional coder-side review per methodology and a mandatory reviewer-side merge gate, which never merges into main, and PR code cannot change the scripts that review it. R5: every question a coder raises reaches the assistant and the wave report. R15: in auto mode the human receives no prompts, only the final answer. R25: an interrupted wave resumes from tracker state and traces. R26: the reviewer reviews each pushed head at most once. R29: every wave produces a report. R31: the CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt`, so this intent adds no CLI command.
- gentle-ai rules, as the adapter and the orchestration rules of gentle-ai state them: global RDD on never grants consent for a candidate, consent is candidate-scoped; a coder never runs `gentle-ai review mode enable|disable`; an unavailable or failed assessment never lowers the risk tier.
- `gabo.yaml:53` lists `.claude/`, `AGENTS.md`, `CLAUDE.md` and `gabo.yaml` in `review.human_only_paths`, and the user-level reviewer scripts merge a change to such a path only by hand. The adapter and the coder agent live under `.claude/`, so their change merges by a human.
- The reviewer runs in a clean, credential-free environment (`review.checks` run with a scrubbed environment; TRD D20). A product QA that drives an engine session needs credentials, which this rule does not allow in that environment.
- The reviewer's binary is the installed one, at least the base branch's `.gabo/VERSION`, and it never self-updates (TRD D24). A product QA run on the PR's build uses a different binary than the installed one.
- The review policy is read from the integration branch (owner direction, 2026-11-13). The reviewer's other rule sources are open (question 23).
- The methodology contract (`apps/cli/internal/skills/gabo-contract-methodology/SKILL.md`) requires an adapter to declare its coder-side review explicitly and never to weaken the coder's hard rules.
- Accepted specs 0023, 0025 and 0026 are not reopened. Their behavior changes only through this intent's spec.
- Owner preference (intent 0026, #700 comment): few custom scripts; prefer the engine's own features.
- Assumption, not decided: this intent binds the gentle-ai adapter that gabo-ai ships, the coders that load it, and the gabo reviewer's user-level scripts. Other methodology adapters keep their own contract.

## Open questions

The spec, section 6, is the list. Status in short:

- Decided on 2026-11-13 and still applying: questions 1, 2, 3, 4, 5, 7, 9, 11, 18, 25, 28 and 29 (spec section 6). Question 3 is settled by the owner's criterion; question 28 by owner decision 2; question 29 by owner decision 5.
- Open, each with a Proposed (auto) option: question 8 (option (b) and its cost cap); questions 19 and 26 (the gates' mechanism; 19 is merged into 26); questions 22 and 23 (the rule sources read from dev, and the missing-file rule); question 24 (which findings block the verdict); question 27 (gates 3 and 4 block or warn); question 31 (cost per PR); questions 32 to 34 (the form of the test-edit rule, and the verifier's placement and model).
- Partly superseded: question 6 (visibility only), where the owner's gates instruction changes its scope for gates 3 and 4; question 10, superseded by the question 3 correction. Question 30 is dropped.
- Moved out with the hooks: questions 12, 13, 15, 16, 17 and 20; question 14 is split (its hook part moved, its gate part is question 27); question 21 was dropped by the owner.
## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it) |
| intent issue | [#758 I0027: Coders and product QA share one definition of done](https://github.com/testeos-z/gabo-ai/issues/758) |
| PRD | [PRD](../../PRD.md) |
| TRD | [TRD](../../TRD.md) sections 9.1 (methodology slot), 9.2 (coder outcome, wave report), D6, D20 (reviewer) |
| spec | [spec](spec.md) |
| sister intents | 0028 deterministic hooks (locks), 0029 branch protection, 0030 agent evals (cited by number; written by a sister agent) |
| plan | none yet: the plan is written only after the owner accepts the narrowed spec |
| related intent | [0022](../0022-measure-roles/intent.md) (the Anthropic harness citation); [0026](../0026-pi-probe-follow-ups/intent.md) (same probe evidence, accepted spec untouched) |
| sources | coder runs of I0023, I0025 (PR #706) and I0026 (PR #739), reported by the assistant, session claude-b2e28fa5; reviewer source `apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh` at `63b36df`; [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook) (primary, page checked 2026-11-13); [Anthropic, Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps) (secondary) |
