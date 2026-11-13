# Spec 0029 Branch protection on main and dev

| field | value |
|---|---|
| intent | [0029-branch-protection-on-main-and-dev](intent.md) |
| status | Draft |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-13 |

Mode is `auto`. The owner's decisions of 2026-11-13 (the intent's Decided table, decisions 1 to 10) are encoded here as decided and are not reopened. Every choice this spec adds carries a **Proposed (auto)** mark with its reason. No agent accepts a spec: section 7 is the owner's.

Evidence base (read 2026-11-13 unless a line says otherwise):

- **Host state, `gh` read-only as the owner.** `repos/testeos-z/gabo-ai` is public and owned by an organization; `orgs/testeos-z` reports plan `free`. `repos/testeos-z/gabo-ai/branches/main/protection` returns HTTP 404 "Branch not protected". `repos/testeos-z/gabo-ai/rulesets` returns HTTP 200 with an empty list. The labels `status:in-progress`, `status:blocked` and `needs-user` exist.
- **Not verified.** Whether this plan accepts a ruleset with a bypass for the admin role, and what the host does when the bypass login merges without the required statuses. Both are section 6, question 1. The empty ruleset list shows the read endpoint answers; it does not show a write is accepted.
- **The contract.** `.gabo/skills/gabo-contract-repository-host/SKILL.md:15`: an adapter covers an operation only with a `### <operation>` heading under `## Operations`, and `gabo check` lists every operation an adapter misses. It lists five operations today (lines 23-41), as TRD 9.1 does (`docs/product/TRD.md:373`). The catalog example adapter is `examples/host-github.md`; the project adapter is `.claude/skills/gabo-repository-host-github/SKILL.md`.
- **Setup.** `.gabo/skills/gabo-setup/SKILL.md:32` asks about the `repository_host` slot; line 34 writes config through `gabo-config-writer`; line 41 saves a decision through the `memory` slot.
- **The retired command.** `docs/cloud-agent-flow.md:85` ("Labels and the integration ruleset", `gabo repo setup [--dry-run]`); `odd/tasks/gbai-cli-mvp.md:40` (T6: "`dev` ruleset (bot approval + checks, no bypass) via gh"); commits `f15d326`, `94e3f48` and `1d59a61` as the intent records them.
- **The reviewer's merge.** `~/.claude/skills/gabo-reviewer/scripts/merge-head.sh`: it requires the `reviewer/tests` and `reviewer/review` statuses by the bot login on the reviewed SHA (line 9), refuses a base of `main` or `master` (line 49), and merges with `gh pr merge <pr> --repo <repo> --merge --match-head-commit <sha>`, "never --admin" (lines 15-16).
- **Config.** `gabo.yaml` has no protection key. `schemas/gabo.project.schema.json:12-13` closes the top-level keys (`additionalProperties: false`), and `repo` is closed too (line 30). `gabo.yaml` is one of `review.human_only_paths` (`gabo.yaml:53`).
- **Self-approval.** `docs/cloud-agent-flow.md:17`: "GitHub forbids self-approval". The agents run with the owner's login (`gabo.yaml:38`).
- **Playbook.** Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21. Verbatim from the text copy of the page, wrapped lines joined: "Branch protection turns anything the agent writes into a PR, with no direct path to main." (lines 1494-1495); "Anything the agent writes arrives as a PR through branch protection, and the agent has no route to push to main." (lines 1461-1462); "Branch protection policies that require a code owner's approval are also worthwhile." (lines 1165-1166); "Approval comes from a human through branch protection, informed by the findings." (lines 1233-1234).

## 1. Requirements

"The operation" is the new `repository_host` operation of S1. "The rule list" is the list of S2.

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The `repository_host` contract shall list one more operation, `protect branches` (name: D1), that sets the rule list on `repo.integration_branch` and on `main`, reads it back from the host, and returns one line per rule and branch: set, missing, or not offered by the host or plan. | Given the catalog contract, When it is read, Then `## Operations` has a `### protect branches` heading that states the rule list, the read-back and the three results. Given an adapter without that heading, When `gabo check` runs, Then it lists the operation as missing for that adapter. | Document check of the contract; the existing adapter-coverage check of `gabo check` |
| S2 | The rule list shall be: on both branches, a pull request before a merge, no force-push and no deletion; on the integration branch only, the statuses `reviewer/tests` and `reviewer/review` required; on each branch, the required approving reviews of S3; and a bypass for the host's administrator role (decisions 1, 2, 3 and 5). | Given the operation ran with every rule offered, When the rules of each branch are read from the host, Then both branches show the three rules, only the integration branch shows the two statuses, `main` shows none, and the bypass names the administrator role. | Live probe through the adapter on a repository the owner names, compared with this row |
| S3 | The required approving reviews shall be read from `gabo.yaml`, one value per branch, zero when the key is absent (decision 4; key: D4). | Given a `gabo.yaml` without the key, When the operation runs, Then the count set and read back is zero on both branches. Given the key sets one for `main`, When the operation runs again, Then `main` shows one and the integration branch still shows zero. | Schema test of the new key; live probe of both cases |
| S4 | The GitHub adapter shall cover the operation (mechanism: D2), and running it again with the same config shall leave the rules unchanged. | Given the rules are already set, When the operation runs again, Then the read-back is identical and no second rule set exists. | Live probe run twice; the adapter's `### protect branches` heading passes `gabo check` |
| S5 | An adapter whose host or plan does not offer a rule shall say so for that rule and shall not fail the operation; an adapter for a host with no implementation yet shall carry the heading and state that the operation is not offered. | Given a host that lacks one rule, When the operation runs, Then that rule returns "not offered" with the host's reason and the other rules are set. Given a GitLab adapter (not built by this intent), Then its heading states the operation is not offered until it is implemented. | Document check of the contract wording; live probe if question 1 finds a missing rule |
| S6 | During initial setup, the setup skill shall recommend the operation, offer it as optional, and show the rule list with the value of each setting before any write to the host (decision 7). | Given a first setup, When the `repository_host` slot is verified, Then the owner sees the rule list and one question, and no host write has happened. Given the owner accepts, Then the operation runs and its read-back is shown. | Document check of `gabo-setup`; one setup session on a repository the owner names |
| S7 | When the owner declines, setup shall record the decline once and shall not ask again in that project (decision 7). | Given a recorded decline, When setup runs again in the same project, Then it does not ask and it writes nothing to the host. | Document check of `gabo-setup`; read of the decision record (S10) |
| S8 | Setup shall report one line per rule that is missing or not offered and continue, and shall always state, with or without a missing rule, that the administrator bypass leaves the server open to any session holding the owner's login and that the local shell guard of intent 0028 is what stops agents on that path (decisions 5 and 6). | Given one rule not offered, When setup finishes the step, Then the report has exactly one line for that rule, the remaining setup steps run, and the report carries the bypass statement. Given every rule set, Then the report still carries the bypass statement. | Document check of `gabo-setup`; the setup session of S6 |
| S9 | Setup shall make sure the labels the tracker adapter reads exist: `status:in-progress`, `status:blocked` and `needs-user`. `mode:cloud` and `mode:local` are not recreated (decision 8; where: D5). | Given a repository without one of the three labels, When the step runs, Then the label exists afterwards and is read back. Given all three exist, Then nothing is written. | Read-back through the `tasks` adapter |
| S10 | After the operation, or after a decline, setup shall save one decision record in memory under `gabo/decision/global/branch-protection` holding the rule list, the choice and the read-back result (decision 10). The rules themselves stay on the host. | Given the operation ran, When the record is read back, Then it holds the rule list and one result per rule and branch. Given a decline, Then it holds the decline and its date. | Read-back through the `memory` adapter |
| S11 | The change shall add no CLI command, and `gabo check` shall stay offline: it does not read the host's rules (PRD R31, decision 9). | Given the change is delivered, When `gabo` lists its commands, Then they are `init`, `check`, `version` and `prompt`. Given no network, When `gabo check` runs, Then its result does not change because of this intent. | Existing CLI tests; `gabo check` |
| S12 | The agreed flow shall keep working under the rule list with the default values: agents open pull requests into the integration branch, the reviewer merges there on its own, and the owner alone merges into `main`. The reviewer's scripts are not changed. | Given the rules are set with zero required approvals and a pull request into the integration branch has both statuses green on its head, When the reviewer's merge script runs, Then it prints `MERGED`. Given the same pull request with a status missing, Then the script does not merge (its own check, unchanged). | One reviewer run on a repository the owner names, after the rules are set |

## 2. Design

The operation is text in three places, as every contract operation is: the catalog contract (`gabo-contract-repository-host`, with its embedded copy under `apps/cli/internal/skills/`), the catalog example adapter (`examples/host-github.md`) and the project adapter (`.claude/skills/gabo-repository-host-github/SKILL.md`). The setup skill gains one step after the `repository_host` slot is verified. `gabo.yaml` and its schema gain one key. No Go command is added; `gabo check` changes only through the heading it already compares.

Flow at setup: show the rule list with the values read from `gabo.yaml`; ask once; on accept, run the operation, show the read-back, report missing rules and the bypass statement (S8), make sure the labels exist (S9), save the record (S10); on decline, save the record and go on.

Assumptions, not owner decisions: (a) a recorded decline stops the question, not the operation, so the owner can still ask an agent to run it later; (b) a change of the approvals value takes effect when the owner asks for the operation again, since nothing watches the config.

What is protected, by actor, once the rules are set:

| actor | on the host | by the local shell guard of intent 0028 |
|---|---|---|
| A login without the administrator role | Stopped: no direct push, force-push or deletion on either branch; no merge into the integration branch without the two statuses | Not needed |
| The owner, by hand | Not stopped, by decision 5: the bypass keeps the direct step | Not judged (not an agent session) |
| An agent holding the owner's login, with 0028 shipped and its hook firing | Not stopped (same login, same bypass) | Stopped |
| The same agent before 0028 ships, on a machine without gabo, or on an engine where the hook does not fire | Not stopped | Not stopped: only the rule text in the skills applies, as today |

This is the accepted cost of decision 5, stated plainly: until intent 0028 ships, this intent adds protection only against logins without the administrator role (`packages/pi/README.md:89`, dated 2026-11-04, says only the owner's account has write access, so those logins cannot push today either; not re-checked), plus a server-side record of the rules. It is not a stand-alone guard for agents.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Shape and name of the operation | (a) one operation that sets and reads back / (b) two operations, set and read | **Proposed (auto):** (a), named `protect branches`. Reason: every write in this contract already ends in its own read-back, and one heading keeps adapter coverage simple. | TRD 9.1's `repository_host` row gains the operation (section 5). Every adapter needs the heading, including ones that only state "not offered". |
| D2 | Mechanism on GitHub | (a) repository rulesets / (b) classic branch protection | (a), as the intent's scope states and as the retired command did. Pending the probe of question 1. | If the plan refuses a ruleset or its bypass, question 1 decides the fallback; the contract text does not change. |
| D3 | Administrator bypass | (a) no bypass (the intent's earlier recommended option, and T6 of the retired command) / (b) bypass for the administrator role | (b), **owner decision 5**. It wins over the earlier recommended option. | The table above. The rule list shown at setup names the bypass. This intent depends on intent 0028. |
| D4 | Config key for the required approvals | (a) a map by branch under `repo`, for example `repo.protection.required_approvals: {main: 0, dev: 0}` / (b) a new top-level section | **Proposed (auto):** (a); the plan fixes the exact key name. Reason: the branches are already named under `repo`. The file is the owner's decision (`gabo.yaml`). | The project schema is closed, so it needs the key before `gabo check` accepts it. `gabo.yaml` is a human-only path: the reviewer does not merge that change, the owner does. A value above zero on the integration branch stops the reviewer's merge, because the host forbids approving one's own pull request and agents use the owner's login; the rule list shown at setup says so when the value is not zero. |
| D5 | Where the labels are made sure of | (a) in the new operation / (b) through the `tasks` adapter, which already has a label-create command | **Proposed (auto):** (b), called from the setup step. Reason: labels belong to the tracker, and the host contract stays about branches and pull requests. | No new `tasks` operation. Setup runs the step whether the protection is accepted or declined. |
| D6 | Required statuses | (a) the reviewer's two contexts / (b) a CI context | (a), decisions 2 and D-1: `reviewer/tests` and `reviewer/review`, the names the reviewer's scripts post. CI on pull requests is out of scope. | A rename of either context in the reviewer's scripts must change the rule list in the same change. |

## 3. UX and app flow (optional)

Not applicable: the change has no screen. Its only user-facing surface is the setup conversation, covered by S6 to S8.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data. The rules live on the host, and the decision record is a memory entry (S10).

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R31 (four CLI commands) | The retired feature was a command. | None left: it returns as a contract operation and a setup step (S11). |
| TRD 9.1, `repository_host` row | The row lists five operations; this adds a sixth. | Accepting this spec accepts the contract change (intent, Constraints). The TRD row is amended by `gabo-dev-lead` in the plan. |
| TRD D13 and the contract's rule "never push to `main`" | The operation writes settings that concern `main`. | No conflict: it writes host settings, not commits, and only after the owner accepts at setup (S6). The agents' rules are unchanged. |
| PRD R9 (the reviewer is the mandatory merge gate) | Server rules could be read as a second gate. | They make the reviewer's statuses required and replace nothing (S12, D6). |
| Playbook, "no direct path to main" (lines 1494-1495) | The administrator bypass leaves a direct path on the host for the owner's login, which agents share. | Owner decision 5: the path is closed for agents locally by intent 0028. The cost is in section 2 and in the setup report (S8). |
| Playbook, "Approval comes from a human through branch protection" (lines 1233-1234) | Required approvals default to zero, so the host does not enforce a human approval. | Owner decision 4. The owner's own merge into `main` is the human approval (D-2). The playbook's code-owner sentence says "also worthwhile" (lines 1165-1166); the intent's non-goals exclude a code-owner file. The value is configurable for a project with more than one person (S3). |
| `review.human_only_paths` | The config key and any change to `.claude/` adapters touch human-only paths. | The reviewer leaves those pull requests for the owner to merge, as today. |
| Task Size item 5 (delivery or environment) | Repository settings are an environment change. | The plan's tasks carry that risk line (intent, Constraints). |

## 6. Open questions

The intent carries no open question. These are raised by this spec.

| # | question | status |
|---|---|---|
| 1 | Does the host, on this plan (public repository, organization on the free plan), accept a ruleset with a bypass for the administrator role, and does a merge by the bypass login without `--admin` get refused when a required status is missing? Neither is verified. | Carried to the plan as its first task, and to the owner for the go-ahead, since the probe writes host settings. **Proposed (auto):** probe with a ruleset that targets a branch name that does not exist, read it back, delete it, and record the answers in the table of section 2. If the plan refuses rulesets or the bypass, fall back to classic branch protection without enforcement on administrators. Tradeoff: the fallback keeps the owner's decisions but uses another API, and its exact behavior for administrators is also unverified, so it needs the same probe. |
| 2 | May the rules be set on a repository before intent 0028 ships? | Carried to the owner at acceptance. **Proposed (auto):** yes; the plan of this intent does not wait for 0028, and until 0028 ships the setup report says that the local guard is not installed yet (S8). Tradeoff: sooner protection against non-administrator logins and a required-status record, against the risk that the rules are taken for an agent brake they are not. |
| 3 | The measure's target (intent, Desired outcome: "Target set by the owner at acceptance"). | Carried to the owner at acceptance. **Proposed (auto):** on `testeos-z/gabo-ai`, the read-back shows every rule of S2 on both branches and the report has no missing-rule line. Tradeoff: if question 1 finds a rule the plan does not offer, the target cannot be met as written and becomes "every offered rule set, each missing one reported". |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| | | |
