# Intent 0027 Coders and product QA share one definition of done

| field | value |
|---|---|
| intent id | 0027-coders-follow-gentle-ai-odd-and-rdd |
| status | Accepted (spec, owner, 2026-11-13) |
| owner | gabo-pm |
| scope | one intent for the whole quality flow: the coder's engineering with gentle-ai ODD and RDD, and the gabo reviewer as product QA (owner, 2026-11-13) |
| product card | none open: [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) is closed, so no card was created; the intent issue is linked under #492 as #703 was |
| date | 2026-11-13 (rewritten for the wider scope) |

## Problem

The owner asked that the coders use gentle-ai as closely as possible to its ODD and its RDD. Owner, verbatim, relayed 2026-11-13 by the owner's local assistant: "quiero que los coder usen gentle-ai lo mas apegado a ODD y RDD".

The owner then widened the scope, relayed 2026-11-13 by the owner's local assistant (approximate voice transcription, quoted as relayed, with an English gloss in brackets):

- "yo creo que podemos hacerlo en un solo intent, que organice bien el tema" [I think we can do it in one intent that organizes the topic well].
- The gabo reviewer "no valida código. Valida es cómo interactúa el usuario con ese código, o sea, el producto" [does not validate code; it validates how the user interacts with that code, that is, the product].
- It must be well defined "qué debe cumplir el código, o sea, cuál es la funcionalidad final" [what the code must meet, that is, the final functionality].

So this intent has two roles, one spec:

- **(a) The coder** does the engineering and its own code review with gentle-ai ODD and RDD: well-written code, unit and integration tests.
- **(b) The gabo reviewer** becomes a product QA. It uses the product as a user would and checks that it works as expected against an agreed definition of done, instead of reading the diff for code defects.

Reference the owner cited: Anthropic, [Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps). The evaluator uses the running app as a user, the generator self-evaluates before handing off, and generator and evaluator agree beforehand on what done means. The same citation is used in spec 0022.

### The coder side

Today a coder loads the methodology adapter, but the adapter is not followed. The evidence is in the repository and in the coder runs reported by the assistant:

- The adapter is wired in: `gabo.yaml:27-28` sets `slots.methodology.adapter`, and `.claude/skills/gabo-methodology` is a symlink to the gentle-ai adapter. The Claude coder agent preloads `gabo-methodology` (`.claude/agents/gabo-coder.md:8`) and is told to follow the methodology slot adapter (`.claude/agents/gabo-coder.md:14`). The coder skill tells the coder to load that adapter before reading code and says the skill carries no methodology of its own (`apps/cli/internal/skills/gabo-coder/SKILL.md:11`).
- The adapter already states the RDD rule: it runs only when `gentle-ai review mode status` is on, and the coder then runs assess on the commit and follows its transitions (`.claude/skills/gabo-methodology-gentle-ai/SKILL.md`, lines 17 and 24, and the coder-side review operation at lines 38-52).
- Reported by the assistant from session claude-b2e28fa5, not re-checked against the transcripts from this run: in 13 coder runs of waves I0023, I0025 and I0026 the adapter text was preloaded in all 13. Only one run (I0025 T3, PR #706) ran `gentle-ai review start` and `status`, and it stopped on the per-candidate consent envelope (`gentle-ai.review-integration.consent/v3`), which a coder cannot answer. The owner's assistant relayed it, the owner granted it, and the review ran after merge. One run (I0026 T8, PR #739) ran only `review mode status` and reported skipping assess as a process miss. The other eleven ran no gentle-ai review command and reported no RDD outcome. No coder created an ODD feature document (`odd/tasks/`).
- The adapter's assess command does not work on committed work as written. Checked on 2026-11-13 with gentle-ai 4.4.0 in a clean worktree at `origin/dev` (nothing pending): `gentle-ai review assess --cwd <wt> --json` returned `risk: high`, `reason: unassessable`, `review_due: true`, and an error that names `--base-ref` and `--committed-only` as the way to assess committed work.

The result: the owner's RDD and ODD are not visible in any wave. A coder's report does not say whether RDD ran, the wave report cannot show it, and a consent envelope has no route to the owner.

### The reviewer side

The gabo reviewer is the merge gate today, and its checks cover the code, not the product:

- Its input is the diff. The review step runs `git diff --stat` and `git diff` from the merge base to the head, and tells the reviewer to read other checkout files for context (`apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh:299-306`).
- Its prompt asks for correctness bugs, security issues, architecture boundary violations and missing tests for changed behavior, and says to ignore formatting (`review-head.sh:324-328`). It names `AGENTS.md` and `docs/product/` as the rules to review against. It does not ask for a check of each spec requirement's acceptance criteria, it does not pass the task issue, and it never runs the product. Its verdict contract is at `review-head.sh:335-343`.
- Its mechanical gates are `review.checks` (`go -C apps/cli build`, `vet` and `test` on the merge result, `gabo.yaml:45-48`) and `review.auto_merge: true` (`gabo.yaml:52`). The merge itself runs only when `REVIEWER_AUTO_MERGE=true` (`merge-head.sh:42`).
- Accepted specs already state acceptance criteria per requirement, Given/When/Then, for example spec 0026 section 1. Nothing in the review step reads them against a running product.

Result: a clean verdict says the code has no visible defect in the diff and the build passes. It does not say that a user can do what the spec says.

## Who is affected

- The owner, who wants the coders to build the way gentle-ai builds, who wants the reviewer to test the product the way a user does, and who needs to see both in each wave report.
- The assistant that runs each wave. It relays consent envelopes, reads the RDD outcome of every task, and relays the reviewer's product QA result.
- Every coder task of every wave. The gap appears in each one of the 13 runs reported above, so it is the normal case, not an exception.
- The gabo reviewer, which stays the only merge gate (PRD R9) and changes from a code reader to a product QA on the PR head.
- Every product change of gabo-ai, because a clean verdict will mean the product was run and not only read.

## Desired outcome

- Every coder task reports its ODD evidence (RED and GREEN, or the stated exception, with the spec ids it covered) and its RDD outcome. The outcome is one of `granted`, `declined`, `passive`, `off`, `unavailable`, or the gentle-ai reason when no review was due. A missing item reads `not run: <reason>`, never as a pass.
- Every coder task writes its done contract before its first commit: one observable check per linked requirement, agreed as OQ 7 decides. The contract is the definition of done the reviewer tests against.
- The coder's code is its own review's responsibility: well-written code with unit tests for each changed rule and integration tests where a task crosses a boundary (a CLI command, a file, a skill that runs a command).
- For every pushed head that changes product behavior, the gabo reviewer runs the build under test as a user would and reports each contract line as `pass`, `fail: <observed>` or `not run: <reason>`. A head with no product change reads `not applicable`.
- The reviewer's verdict shows the product QA result and the checks result separately, so the coder knows which one to fix.
- Every wave report shows the RDD outcome and the ODD line for each task, so the owner can count how many tasks ran RDD, and with what result, without reading the coder transcripts.
- A coder never answers a consent envelope, never toggles RDD, and never lowers a risk tier. A consent envelope reaches the owner through a route the owner decides.
- The gabo reviewer stays the only merge gate (PRD R9). RDD and product QA add outside views and never merge anything.
- Measure: the share of tasks whose wave report line carries an RDD outcome, per wave; and the share of product PRs whose reviewer comment carries a product QA result. The targets are set by the owner at acceptance.

## Constraints

- PRD R8: the coder uses the methodology declared in the methodology slot. R9: two layers, an optional coder-side review per methodology and a mandatory reviewer-side merge gate, which never merges into main, and PR code cannot change the scripts that review it (separation: the PRD does not use the phrase "separation of duties"; R9 is the rule in substance). R5: every question a coder raises reaches the assistant and the wave report. R15: in auto mode the human receives no prompts, only the final answer. R25: an interrupted wave resumes from tracker state and traces. R26: the reviewer reviews each pushed head at most once. R29: every wave produces a report. R31: the CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt`, so this intent adds no CLI command.
- gentle-ai rules, as the adapter and the orchestration rules of gentle-ai state them: global RDD on never grants consent for a candidate, consent is candidate-scoped; a coder never runs `gentle-ai review mode enable|disable`; an unavailable or failed assessment never lowers the risk tier.
- `gabo.yaml:53` lists `.claude/` in `review.human_only_paths`, and the user-level reviewer scripts merge a change to such a path only by hand (`common.sh:256-257`). The adapter and the coder agent live under `.claude/`, so their change merges by a human, not by the reviewer.
- The reviewer runs in a clean, credential-free environment (`review.checks` run with a scrubbed environment; TRD D20). A product QA that drives an engine session needs credentials, which this rule does not allow in the reviewer's environment.
- The reviewer's binary is the installed one, at least the base branch's `.gabo/VERSION`, and it never self-updates (TRD D24). A product QA run on the PR's build uses a different binary than the installed one.
- The methodology contract (`apps/cli/internal/skills/gabo-contract-methodology/SKILL.md`) requires an adapter to declare its coder-side review explicitly and never to weaken the coder's hard rules.
- Accepted specs 0023, 0025 and 0026 are not reopened. Their behavior changes only through this intent's spec.
- Owner preference (intent 0026, #700 comment): few custom scripts; prefer the engine's own features.
- Assumption, not decided: this intent binds the gentle-ai adapter that gabo-ai ships, the coders that load it, and the gabo reviewer's user-level scripts. Other methodology adapters keep their own contract; the methodology contract changes only if the owner asks for it.

## Open questions

The owner accepted the spec on 2026-11-13 with the recommended option of each question (spec, section 7). Each question is decided in the spec, section 6, with its option record. Question 8 option (b) is the one open question: the cost cap is not set.

| # | question | status |
|---|---|---|
| 1 | How does a per-candidate RDD consent reach the owner when a coder cannot give it, and what happens to the task meanwhile? | decided (owner): option (a), spec section 6.1 |
| 2 | Does the ODD feature document (`odd/tasks/<feature>.md` and its Engram mirror) get created per task, or do the gabo plan row and the task issue act as it? | decided (owner): option (b), spec section 6.2 |
| 3 | Who reads the code: does coder-side RDD own the code read while the reviewer runs product QA only, or do both still read the code? | decided (owner): option (a), spec section 6.3 |
| 4 | Is the RDD candidate each work-unit commit before the push, or the PR head? | decided (owner): option (a), spec section 6.4 |
| 5 | What happens when RDD is unavailable or its assessment fails? | decided (owner): option (c), spec section 6.5 |
| 6 | Beyond visibility, does a gate enforce the coder's ODD, RDD and done contract? | decided (owner): option (a), revisit (b) after two waves, spec section 6.6 |
| 7 | Where does the product QA get what the product must do: the spec's acceptance criteria, the task issue, a done contract agreed before coding, or a combination? | decided (owner): option (d), spec section 6.7 |
| 8 | How does QA test a product that is a CLI plus skills with no web UI, and at what cost per PR? | decided in part (owner): (a) on every product PR and (c) once per wave; **open: option (b) and its cost cap**, spec section 6.8 |
| 9 | Who gates the merge when QA does not read the code, and what happens to `review.checks` (build, vet, test)? | decided (owner): option (a), spec section 6.9 |
| 10 | What happens when RDD did not run or was not due, and nobody reviewed the code? | decided (owner), narrowed: (c) for high-risk changes, (a) for the rest, spec section 6.10 |
| 11 | The intent title no longer names both roles. Keep "Coders follow gentle-ai ODD and RDD", or propose a new title with the folder unchanged? | decided (owner): option (b), title "Coders and product QA share one definition of done", spec section 6.11 |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it) |
| intent issue | [#758 I0027: Coders and product QA share one definition of done](https://github.com/testeos-z/gabo-ai/issues/758) |
| PRD | [PRD](../../PRD.md) |
| TRD | [TRD](../../TRD.md) sections 9.1 (methodology slot), 9.2 (coder outcome, wave report), D6, D20 (reviewer) |
| spec | [spec](spec.md) |
| plan | none yet: the plan is written only after the owner accepts the spec |
| related intent | [0022](../0022-measure-roles/intent.md) (the Anthropic harness citation); [0026](../0026-pi-probe-follow-ups/intent.md) (same probe evidence, accepted spec untouched) |
| sources | coder runs of I0023, I0025 (PR #706) and I0026 (PR #739), reported by the assistant, session claude-b2e28fa5; gentle-ai 4.4.0 `review assess --help` and the 2026-11-13 assess check above; reviewer source `apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh` at `406015f`; [Anthropic, Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps) |
