---
name: gabo-plan-reviewer
description: "Trigger: plan change, plan-change proposal, critical question, approval scale, plan reviewer, owner absent. Plan-change flow for directiva and the clean-context plan reviewer."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

The one plan-change skill (TRD D7, D8, D24; J6; R7, R19, R27). Directiva (`gabo-pm`, `gabo-dev-lead`) load it to run the **Plan-Change Flow**; the `gabo-plan-reviewer` agent loads it to run the **Review**. A critical question is one that changes scope, the plan or other tasks; it is never decided silently (R19).

## Hard Rules

- One issue and one PR per proposal (D7), touching only `docs/product` and tasks; never inside a code PR, never two proposals in one PR.
- Apply the approval scale below against `plan_change.scope` (`task`, `group` or `global`) and `plan_change.global_without_owner` in `gabo.yaml`. A missing key is the narrowest setting: scope `task`, global `false`.
- The reviewer is started only when the owner is absent, and never for a proposal debated with the owner.
- The reviewer never edits code or documents, never pushes, comments, labels or merges; its reply is advice that directiva applies per the scale.
- While a proposal is debated or waits, only the affected task is BLOCKED (operating rule 2).

## Approval Scale

| proposal scope | owner present | owner absent |
|---|---|---|
| One task | Debated with the owner | Applies automatically once the plan-change reviewer agrees |
| Group of tasks | Debated with the owner | Applies automatically only if the config allows group scope and the reviewer agrees; otherwise waits for the owner |
| Global plan | Debated with the owner | Waits for the owner unless `plan_change.global_without_owner` is true, and still needs the reviewer's agreement |
| Any scope beyond `plan_change.scope`, or rejected by the reviewer | Debated with the owner | Never applies automatically; waits for the owner |

Scope order is `task` < `group` < `global`: a proposal is within `plan_change.scope` when its scope is at most the configured one. This table is the APP_FLOW "Plan-change approval scale" row for row.

## Plan-Change Flow

Run by directiva for each critical question (J6, TRD 5.3).

1. Write the proposal: trigger question, scope (one task, group of tasks or global plan), affected task ids, proposed change, rationale.
2. Open one issue for it through the `tasks` adapter (re-read it), and mark each affected task BLOCKED (`tasks` change status BLOCKED) with the proposal issue as the reason.
3. Branch from `repo.integration_branch`, edit only `docs/product` and tasks, and open one PR through `repository_host` with `Refs #<proposal issue>` and the proposal in the body. A diff outside `docs/product` is not a plan change: split it out.
4. Owner present: return the proposal to the assistant to debate with the owner; do not start the reviewer.
5. Owner absent and the scope beyond `plan_change.scope` (or global with `plan_change.global_without_owner` false): it waits for the owner; do not start the reviewer.
6. Otherwise (only when the owner is absent and the scale allows it): request the reviewer by returning `PLAN-REVIEW PR #<n>` with the proposal; the assistant starts a fresh `gabo-plan-reviewer` with exactly that input and hands its reply back.
7. `AGREE`: the proposal applies; mark it **Approved (auto, plan reviewer agreed)** in the PR body, record it through `memory` (`save decision`, with the reviewer's evidence as the reason) and hand the PR to the normal merge gate. Unblock the affected tasks only after the PR merges.
8. `COUNTER-PROPOSAL` or anything else: it never applies automatically; it waits for the owner with both proposals. Never open a second round with the reviewer for the same proposal.

## Review

Run by `gabo-plan-reviewer` in a fresh session with a clean context: nothing from the session that wrote the proposal; that independence is the point (D8). You index the repository yourself and agree or counter-propose. Read-only: never edit, push, comment, label or merge.

Input: `PLAN-REVIEW PR #<n>` and the proposal.

1. Index the repository yourself: `docs/product/` (PRD, TRD, IMPLEMENTATION_PLAN, APP_FLOW, stories), `gabo.yaml` (`plan_change.*`), and the PR diff and description.
2. Check: the PR touches only `docs/product` and tasks; its scope is within `plan_change.scope`; the traceability chain story → requirement → decision → task → test still holds; no dependency cycle, dangling id or `depends_on` / `unblocks` mismatch; the rationale answers the trigger question.
3. Decide. Agree only if every check passes and the change is the smallest one that answers the question.

## Output Contract

Directiva returns the proposal issue and PR numbers, the scope, the path taken (owner debate, waits for owner, reviewer agreed, reviewer counter-proposed), the BLOCKED tasks and unverified writes.

The reviewer replies with exactly one of:

- `AGREE` followed by at most 5 lines of evidence (file:line).
- `COUNTER-PROPOSAL` followed by the different improvement, why, and the evidence.
