# Plan 0029 Branch protection on main and dev

| field | value |
|---|---|
| intent | [0029-branch-protection-on-main-and-dev](intent.md) |
| spec | [spec](spec.md) (accepted by the owner on 2026-11-14, section 7) |
| status | Draft. Not started: no wave runs until the owner orders it. Tasks are in the tracker under [#766](https://github.com/testeos-z/gabo-ai/issues/766). |
| owner | gabo-dev-lead |
| date | 2026-11-14 |

Mode is `auto` (owner's standing order: apply the recommended option to every doubt and record it; merges to `main`, versions, publishing, secrets, anything that touches `main` and the owner's own values stay with the owner). Every fixed question below is answered by `gabo-dev-lead` and marked **Proposed (auto)** with a one-line reason. The plan is not accepted by any agent: it changes nothing until a human reviews the plan PR.

The spec is not edited. The TRD 9.1 `repository_host` row is amended in this plan PR, as spec section 5 says (`gabo-dev-lead` does it in the plan). Two tasks write rules or probe rules on the real host of `testeos-z/gabo-ai`, so they are owner actions and carry `status:blocked` and `needs-user`: T1 (the probe of spec question 1) and T5 (the live setup and acceptance run), as intent 0027's T14 did.

Human merge: every task that touches `.claude/` or `.gabo/skills/` is merged by a person, not by the reviewer (`gabo.yaml:53`; `.gabo/skills/` by built-in rule, TRD D20 and 9.4). Each such row says so. T3 changes `gabo.yaml`'s schema, not `gabo.yaml`, and `schemas/` is not human-only.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: see Steps. The probe (T1) answers spec question 1 before the adapter text is written; the config key (T3) is independent and starts at once; the setup step (T4) follows the contract and the key; the live run (T5) is last. Reason: the mechanism of D2 is unverified until T1, and the setup step shows values that T3 defines. |
| Q2 | **Proposed (auto)**: five tasks, each one reviewable PR of about 400 changed lines or less, test-first where a runnable test exists. Reason: one behavior per task, split at the shared file that forces an order (the setup skill is edited by one task only). |
| Q3 | **Proposed (auto)**: see the `spec refs` column and the Spec map. Reason: each row names the S# and D# its acceptance proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic. One edge crosses intents: T5 depends on `0028/T<n>` (the shell guard that stops agents on the admin path, intent decision 5; see PD1). Reason: the owner's admin bypass is safe for agents only while that guard runs, so the live rules are not applied before it exists. |
| Q5 | **Proposed (auto)**: see `acceptance`. Each acceptance starts with its failing test. Skill-text rules are pinned by Go text tests in `apps/cli/internal/skills` (the precedent of 0026 and 0027); the config key by a schema test in `apps/cli/internal/config`. Size is `S` (about 100 changed lines), `M` (about 200 to 400) or `L` (more than 400, to split at unblock). Reason: the spec's verification column names these forms. |
| Q6 | **Proposed (auto)**: yes. Every spec requirement S1 to S12 has at least one task: see the Spec map. Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |
| Q9 | **Proposed (auto)**: no `gabo check` update is needed beyond what the change already triggers. `gabo check` compares the contract's `### <operation>` headings with each adapter's (`apps/cli/internal/slices/check/coverage.go`), so the new heading is checked with no code change; the schema gains one key in T3 so the closed `repo` object accepts it; no CLI command is added (S11, PRD R31). Reason: intent 0008 S10 and TRD 8.5. |

## Steps

1. **Probe and key start together.** T1 (probe of spec question 1, owner action) and T3 (config key, S3 and the S11 pin) share no file. T1 needs the owner's go-ahead because it writes host settings; T3 does not wait for it.
2. **Contract and adapters in one task.** T2 follows T1. It adds `### protect branches` to the contract and to both GitHub adapters in one PR, because `gabo check` reports a contract operation that an adapter lacks, so the contract heading must never merge ahead of the adapters (PD3).
3. **Setup step.** T4 follows T2 and T3: it edits `gabo-setup` once (offer, decline, report, labels, decision record).
4. **Live run last.** T5 follows T2, T4 and the 0028 shell guard: the owner sets the rules on `testeos-z/gabo-ai`, reads them back, runs the operation twice, and the reviewer merges one pull request through them (S12).

The spec's end-to-end acceptance and the owner's target (spec question 3) are checked by T5, not by any single code task.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. The tracker titles are `I0029/T<n>: <title>`. Tracker issues: filled in at creation (see the PR body). Labels: `m1-local-core`, the size label, `flow-improvements`; `status:blocked` and `needs-user` on the blocked rows.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Probe: does the host accept a ruleset with an administrator bypass? [blocked-on-owner: writes host settings] | S2, S4 (evidence for D2, D3); spec question 1 | — | T2 | **Status `blocked` and `needs-user` until the owner gives the go-ahead** to write a probe ruleset on `testeos-z/gabo-ai` (Open questions 1). No code change and no PR. Run through the `repository_host` adapter's commands: create one ruleset that targets a branch name that does not exist, with the three rules, the two statuses and an administrator-role bypass; read it back; open a pull request that lacks a required status on a throwaway branch only if the rule can be tested without touching `main` or `dev` (otherwise record `not run: <reason>`); delete the ruleset and confirm the rules list is empty again. The answers (ruleset accepted or refused on this plan, bypass accepted or refused, behavior of a merge by the bypass login without `--admin`) are posted on the task issue and saved as one decision (`gabo/decision/global/branch-protection-probe`), read back. If the host refuses a ruleset or the bypass, the fallback is classic branch protection (spec question 1), and the owner is told before T2 starts. | S |
| T2 | `protect branches` in the repository_host contract and both GitHub adapters | S1, S2, S4, S5; D1, D2, D3, D6 | T1 | T4, T5 | Human merge: `.gabo/skills/` and `.claude/skills/` are human-only; the catalog copies under `apps/cli/internal/skills/` and `plugins/codex/skills/` stay byte-identical with their `.gabo/skills/` copies. One PR carries all three texts (PD3): `gabo-contract-repository-host/SKILL.md`, `examples/host-github.md` and `.claude/skills/gabo-repository-host-github/SKILL.md`. Test first: a Go text test in `apps/cli/internal/skills` fails on today's contract for lack of a `### protect branches` heading that states the rule list (S2), the read-back and the three results `set`, `missing`, `not offered` with the host's reason (S1, S5), and for lack of the same heading and of the idempotence statement in both GitHub adapters (S4); it also pins the statement that the integration branch alone requires `reviewer/tests` and `reviewer/review` (D6) and that the bypass names the administrator role (D3). The adapter text uses the mechanism T1 confirmed (D2, rulesets by default) and reads the approvals count with the key of T3 (PD2), zero when absent. A host with no implementation yet (a GitLab adapter is not built here) states the operation is not offered. `gabo check` exits 0 with the project adapter, and a scratch adapter without the heading is listed as missing the operation. No Go code changes. | L |
| T3 | Config key for the required approvals per branch, and the S11 pin | S3, S11; D4 | — | T4 | Not human-only (`schemas/` and `apps/cli/internal/config`); `gabo.yaml` itself is not edited. Test first: a schema and config test fails until `repo.protection.required_approvals` is accepted with the sub-keys `integration` and `main`, each an integer of zero or more, both optional (PD2); an absent key and a project without it pass `gabo check` unchanged; a negative or non-integer value is rejected with the existing `gabo check` error shape. A CLI test pins that the commands are exactly `init`, `check`, `version` and `prompt`, and a `gabo check` run with no network and no host call gives the same result with and without this change (S11). The embedded copy of the schema is regenerated if the repository keeps one. | S |
| T4 | Setup step: offer protection, decline once, report, labels, decision record | S6, S7, S8, S9, S10; D5 | T2, T3 | T5 | Human merge: `.gabo/skills/gabo-setup/` is human-only; its catalog and codex copies stay byte-identical. Test first: a Go text test in `apps/cli/internal/skills` (next to `setup_test.go`) fails on today's `gabo-setup` until the step, placed after the `repository_host` slot is verified, states: the rule list is shown with the value of each setting read from `gabo.yaml` and one question is asked before any host write (S6); when the approvals value on the integration branch is not zero the list says the reviewer's merge stops (D4); a decline is saved once and later runs do not ask and write nothing to the host (S7); one report line per missing or not-offered rule, setup continues, and the bypass statement (administrator bypass leaves the server open to any session holding the owner's login; the shell guard of intent 0028 stops agents) is always present, with the added line that the guard is not installed yet while 0028 has not shipped (S8, spec question 2); `status:in-progress`, `status:blocked` and `needs-user` are made sure of through the `tasks` adapter's label command whether the owner accepts or declines, `mode:cloud` and `mode:local` are not created, nothing is written when all three exist (S9, D5); one decision record `gabo/decision/global/branch-protection` holds the rule list, the choice and one result per rule and branch, or the decline with its date (S10). `gabo check` exits 0. | M |
| T5 | Live setup and acceptance run on testeos-z/gabo-ai [blocked-on-owner: writes rules on the real host] | S2, S3, S4, S6, S12; spec question 3 | T2, T4, 0028/T<n> | — | **Status `blocked` and `needs-user`: the owner runs it, since it writes rules on `testeos-z/gabo-ai` and on `main` (owner constraint), and only after the shell guard of 0028 exists (PD1).** Not a code PR. Run one setup session: the rule list and one question appear with no host write before the answer (S6); on accept the operation runs and its read-back shows, on both branches, the pull-request, no-force-push and no-deletion rules, only `dev` with `reviewer/tests` and `reviewer/review`, and the bypass naming the administrator role (S2); with no key in `gabo.yaml` the count is zero on both branches, and after the owner sets `repo.protection.required_approvals.main: 1` and asks again `main` shows one and `dev` zero (S3); a second run leaves the read-back identical with no second rule set (S4); the report has a line per missing rule or none, and the bypass statement; the decision record is read back. Then one reviewer run on a pull request into `dev` with both statuses green prints `MERGED`, and one with a status missing does not merge (S12; the reviewer's scripts are unchanged). Results are posted on #766, read back. If a rule is not offered, the target becomes "every offered rule set, each missing one reported" (spec question 3). | M |

## Spec map

How each spec item lands on the tree at `dev`. The spec is not edited.

| spec | what changes | task |
|---|---|---|
| S1 | `### protect branches` in the contract, with rule list, read-back and three results | T2 |
| S2 | the rule list in the text; its live read-back | T2 (text), T1 (probe), T5 (live) |
| S3 | `repo.protection.required_approvals` per branch, zero when absent | T3 (key), T2 (adapter reads it), T5 (live) |
| S4 | GitHub adapter coverage, idempotent re-run | T2 (text), T5 (live, run twice) |
| S5 | not-offered result per rule; GitLab states not offered | T2 |
| S6 | the setup step shows the rule list and asks once | T4, T5 (session) |
| S7 | a decline is recorded once and not asked again | T4 |
| S8 | one line per missing rule, continue, always the bypass statement | T4, T5 |
| S9 | the three tracker labels made sure of | T4 |
| S10 | one decision record under `gabo/decision/global/branch-protection` | T4 |
| S11 | no new command, `gabo check` offline | T3 (pin) |
| S12 | the reviewer merges under the rules, scripts unchanged | T5 |

## Plan decisions

All are **decided by the owner's standing rule** (resolver: agent, owner's rule: apply the recommended option to every doubt and record it).

| id | what | question | choice | reason | task |
|---|---|---|---|---|---|
| PD1 | Dependency on intent 0028 | Intent 0029 depends on 0028 (decision 5), but spec question 2 proposes that the plan does not wait for it. | The code and text tasks (T2, T3, T4) do not wait. The task that writes the rules on the real host (T5) depends on `0028/T<n>`, the shell guard (`gabo check --pre-command` with the S4 locks). | Setting the rules earlier gives only protection against logins without the administrator role and a status record; the owner's bypass is safe for agents only while the guard runs, and applying it is the owner's act. T4's report still says the guard is not installed while 0028 has not shipped (S8). | T5 |
| PD2 | Config key name (D4, the plan fixes it) | `repo.protection.required_approvals` with branch names `main` and `dev`, or role names. | `repo.protection.required_approvals.integration` and `.main`, integers of zero or more, both optional. | The integration branch name is configurable (`repo.integration_branch`), so a key named `dev` would be wrong in a project that uses another name; `main` is fixed by D13. | T3, T2, T4 |
| PD3 | One task for the contract and both adapters | The contract heading makes `gabo check` list the operation as missing for an adapter without it, so a contract PR merging alone breaks `gabo check` for the project adapter. | One PR with the contract, the catalog example adapter and the project adapter. | An adapter heading alone is harmless, but splitting into three PRs forces a merge order and a red intermediate state for no gain; the size is `L` and splits at unblock if it exceeds about 400 changed lines. | T2 |
| PD4 | TRD 9.1 row | The spec's section 5 says the row is amended by `gabo-dev-lead` in the plan. | Amended in this plan PR: the `repository_host` row gains `protect branches`. No other TRD line changes. | `docs/product` is the plan PR's surface; the spec's acceptance accepted the contract change. | none (this PR) |
| PD5 | Where the probe result is recorded | The spec says to record the answers in the spec's section 2 table; the spec is accepted and not edited here. | T1 posts the answers on its task issue and in a memory decision; the assistant returns them to gabo-pm, who may update the spec. | The dev lead does not edit the spec. | T1 |
| PD6 | Human merges | T2 and T4 change `.gabo/skills/` and `.claude/skills/`. | Each such row says "human merge". The reviewer's `merge-head.sh` stops a merge that touches them. | `gabo.yaml:53` and TRD D20. | T2, T4 |
| PD7 | `gabo.yaml` | The key's default is zero, so no edit of `gabo.yaml` is needed. | Not edited by any task. The owner sets a value by hand if wanted (T5 exercises it). | `gabo.yaml` is human-only and the values are the owner's. | T3, T5 |

## Parallel groups

Edit surfaces are disjoint between groups.

| group | tasks | files |
|---|---|---|
| G1 probe | T1 | none (host settings, owner action) |
| G2 config key | T3 | `schemas/gabo.project.schema.json` and its embedded copy if any, `apps/cli/internal/config/` (key and test), one CLI test |
| G3 contract and adapters | T2 | `apps/cli/internal/skills/gabo-contract-repository-host/` (`SKILL.md`, `examples/host-github.md`) and its `.gabo/skills/` and `plugins/codex/skills/` copies, `.claude/skills/gabo-repository-host-github/SKILL.md`, one new test file in `apps/cli/internal/skills` |
| G4 setup | T4 | `apps/cli/internal/skills/gabo-setup/SKILL.md` and its copies, `setup_test.go` or a new test file |
| G5 live run | T5 | none (host settings, owner action) |

Runnable at once (pairwise disjoint): T1 (after the owner's go-ahead) and T3. T2 follows T1. T4 follows T2 and T3. T5 follows T2, T4 and the 0028 shell guard.

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| The plan or the organization refuses a ruleset or the administrator bypass | T2's mechanism changes to classic branch protection, whose behavior for administrators is also unverified | T1 runs first and the owner is told before T2 starts; the contract text does not change (D2) |
| T1 waits for the owner | T2, T4 and T5 wait behind it | T3 starts at once; the owner's go-ahead is open question 1 |
| The contract heading merges before an adapter has it | `gabo check` lists a missing operation and fails | PD3: one PR carries the contract and both adapters |
| Rules set before 0028's guard ships | The owner's bypass is open to agents holding the owner's login | PD1: T5 waits for the guard; T4's report says the guard is not installed yet |
| A required approval above zero on the integration branch | The reviewer cannot merge (self-approval is forbidden) | T4 warns when the value is not zero; zero is the default |
| A rename of `reviewer/tests` or `reviewer/review` in the reviewer scripts | The rule list requires a status nothing posts | D6: the rename must change T2's list in the same change; the Go text test pins both names |
| Text tests pin wording and wording drifts | Tests fail on harmless edits | Tests pin the rule words and the result names, not whole paragraphs (the 0026 and 0027 precedent) |
| Two human-merged PRs queue | T2 and T4 wait for a person | T3 and T1 start at once; the others follow by `depends_on` |

## Open questions

1. **Go-ahead for the probe (T1).** The probe writes and deletes a ruleset on `testeos-z/gabo-ai`; the owner decides when. (owner)
2. **Timing of the live run (T5).** It waits for the 0028 shell guard (PD1). The owner may release it earlier, accepting that the bypass is open to agents until then (spec question 2 allows it). (owner)
3. **Spec table update (PD5).** The probe answers belong in the spec's section 2 table; the plan does not edit the spec. (gabo-pm)
