# Intent 0029 Branch protection on main and dev

| field | value |
|---|---|
| intent id | 0029-branch-protection-on-main-and-dev |
| status | Draft. The owner accepts the spec, not this intent (owner decision D-6, 2026-11-13) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#766 I0029: Branch protection on main and dev](https://github.com/testeos-z/gabo-ai/issues/766) |
| date | 2026-11-13 (revised 2026-11-13 after the owner's decisions, see the Decided table) |

## Problem

gabo had a repository setup that configured branch protection, and it was retired with the Go command that ran it. The owner decided to recover it, as a service-agnostic operation and not as a command: owner, relayed by the owner's local assistant on 2026-11-13, "sí bien, recuperémoslo" [yes, let's recover it], after the assistant offered it as cut C of the playbook review.

What the retired command did, from git history (`origin/dev`, read 2026-11-13):

- `f15d326` (2026-10-20, "feat(cli): add idempotent repo setup for labels and integration ruleset"), `apps/cli/internal/slices/repo/usecase.go`: created or updated three labels, `mode:cloud`, `mode:local` and `needs-user`, and a repository ruleset named `gbai-integration` on the integration branch. The ruleset had a `deletion` rule, a `non_fast_forward` rule, a `pull_request` rule and a `required_status_checks` rule with two contexts, `reviewer/tests` and `reviewer/review`. It printed a warning that repository and organization admins can still bypass rulesets.
- `94e3f48` (2026-10-23, "feat(repo,audit): drop required approval and audit the reviewer marker"): the `pull_request` rule's `required_approving_review_count` went from 1 to 0.
- `1d59a61` (2026-10-27, "refactor(cli): retire the work-time Go slices (#77)", refs #20): removed the `repo` command and its slice. `docs/product/TRD.md:600` (D18) listed "repo setup" in the Go CLI scope; D18 is now "agent compiler only". `docs/product/UX_UI.md:161` says `gbai repo setup` (removed) is replaced by the `repository_host` adapter (gh).

Today, read-only, 2026-11-13, `gh` as the owner:

- `repos/testeos-z/gabo-ai/rulesets`: an empty list. `repos/testeos-z/gabo-ai/rules/branches/main` and `.../rules/branches/dev`: no rules.
- `repos/testeos-z/gabo-ai/branches/main/protection` and `.../branches/dev/protection`: HTTP 404, "Branch not protected".
- The repository is public and owned by an organization; the owner's token has admin permission.

The worktree does not show when or why the ruleset went. The result: the server does not refuse a push to `main` or `dev`, a force-push, or a deletion of either branch. The reviewer's script waits for its two statuses before it merges (`~/.local/share/gabo/skills/gabo-reviewer/scripts/merge-head.sh:55-59`, and it posts them with `common.sh:163-164` and `:630`), but nothing on the server requires them.

Why it was not configured before: intent 0011 recorded that branch protection and GitHub Environments "are not available on this private repository's plan" (`docs/product/intents/0011-docs-for-the-public-repository/intent.md:38`). On 2026-11-06 it checked the public facts: "available but not configured" (`0011/spec.md:33`, S4). The repository is now public, so that reason no longer holds.

Plan and visibility facts, from GitHub and GitLab documentation, read 2026-11-13 through a page extraction (not byte-checked):

- GitHub, "About protected branches": "You can enable branch restrictions in public repositories owned by a GitHub Free organization". Bypass lists: "Actors may only be added to bypass lists when the repository belongs to an organization." The page does not say whether rulesets are available on Free for public repositories; that is not stated, so this intent does not assume it.
- GitLab, protected branches: the page lists "Tier: Free, Premium, Ultimate". Push, merge and force-push settings have no tier label; code owner approval is "Premium and Ultimate". The page does not state the tier of merge request approval rules.

One fact shapes every option. The agents run with the owner's `gh` login (`gabo.yaml`, reviewer comment: "the scripts run as the owner with the owner's gh login"), and the owner's account has admin rights on the repository. A bypass for admins therefore also bypasses the rule for every agent.

Playbook, Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21, https://claude.com/resources/articles/the-ai-native-sdlc-playbook. Byte-checked on 2026-11-13 against the page text:

- "Branch protection turns anything the agent writes into a PR, with no direct path to main."
- "Branch protection policies that require a code owner's approval are also worthwhile."

## Owner decisions, 2026-11-13

- General criterion, verbatim: "si está en el documento de Anthropic, úsalo. Vamos a cumplir ese documento estrictamente" [if it is in Anthropic's document, use it; we will follow that document strictly]. What the playbook prescribes is in.
- D-2 The reviewer keeps merging only into `dev`; a person approves the spec and `main`. Agents reach only `dev`; the owner does `main` and the versions.
- D-1 CI on PRs stays out. The two required statuses come from the reviewer's scripts, not from a CI run.
- D-6 The owner accepts the spec, not the intent nor the plan.
- D-1, D-2 and D-6 are the owner's numbered choices as relayed by the assistant; the owner's own wording of them is not in the record, so they are not quoted verbatim.

## Who is affected

- The owner, who merges, releases, and is the only human who can override.
- The reviewer, whose merge gate depends on its two statuses being required on the server to mean anything.
- Every coder PR into `dev`, and every release PR into `main`.
- Every new project that runs initial setup, which is offered the protection at that point.

## Scope

In scope:

- A new operation of the `repository_host` contract (TRD 9.1, `docs/product/TRD.md:373`), service-agnostic, named for the protection of the integration branch and of `main`. It sets the rules and reads them back. The GitHub adapter implements it with the rulesets API. A GitLab adapter is not built by this intent; until one exists, the operation reports `unsupported` for GitLab.
- The rules, decided: a pull request before a merge, no force-push and no deletion on both `main` and `dev`; the statuses `reviewer/tests` and `reviewer/review` required on `dev` only (see the Decided table).
- Initial setup recommends the operation and offers it as optional. When the service or the plan lacks it, setup warns and names what is missing.
- Recover the label set the tracker adapter uses (open question 5).

Non-goals:

- No new CLI command (PRD R31, `docs/product/PRD.md:170`). The operation lives in the adapter and in the setup prompt, not in `gabo repo setup`.
- No change to the reviewer's gate logic or its scripts.
- No protection or secret change for `pi/pkg-up` or `NPM_TOKEN`. Intent 0011 (D2) left that as a later change. Package publishing is out of all four intents of this batch (owner, 2026-11-13), so it is not taken here.
- No CI change (D-1) and no code-owner file. The repository has no `CODEOWNERS` file, and the owner did not ask for one.
- No enforcement of a hook. Server rules and the local hooks of intent 0028 are separate layers.

## Desired outcome

- At initial setup, the owner is offered the protection. If accepted, the rules in the Decided table exist on the server and the setup reads them back and reports what it sees. If declined, the setup records the decision in memory and does not ask again in that project.
- After setup, a direct push to `dev` or `main`, a force-push to either, and a deletion of either are refused by the server, whoever holds the token.
- A pull request into `dev` cannot merge until the reviewer's two statuses are green on its head.
- When the service or plan lacks a rule, setup says so in one line per missing rule and continues. Nothing fails silently.
- Measure: the read-back shows the rules set on each branch. Target set by the owner at acceptance.

## Constraints

- PRD R31: no new command; the CLI offers `gabo init`, `gabo check`, `gabo version` and `gabo prompt`. `gabo check` is offline and does not call the service.
- TRD D13 (`docs/product/TRD.md:595`): agents merge only into `dev`, never into `main`. Rules on `main` restrict the owner's path too.
- PRD R9: the reviewer is the mandatory merge gate; server rules make its statuses required and do not replace it.
- TRD 9.1 (`docs/product/TRD.md:373`): the `repository_host` contract already lists open PR, comment, status, merge and delete branch. A protection operation is one more operation of that contract, which the owner must accept as a contract change.
- The single-user model: agents and the owner share one login and admin rights (see the Problem), so no rule may depend on telling them apart.
- Task Size item 5 (delivery or environment): repository settings are an environment change, so the spec and its tasks carry the matching risk line.
- `docs/cloud-agent-flow.md:17` says GitHub forbids self-approval, so a required approval that the owner's account must give cannot be met by the reviewer's own PR.

## Decided (owner, 2026-11-13)

| # | was question | decision | basis |
|---|---|---|---|
| 1 | 2, which branches | Both `main` and `dev` are protected. Agents reach only `dev`. | D-2. The playbook: "Branch protection turns anything the agent writes into a PR, with no direct path to main." |
| 2 | 4, which statuses, and where | `reviewer/tests` and `reviewer/review` required on `dev`; none on `main`. | D-2: the reviewer merges only into `dev`. The reviewer posts both on PRs into `dev` (`common.sh:163-164`). |
| 3 | 1, the rules part | A pull request before merge, no force-push, no deletion, on both branches. | D-2 and the playbook's "no direct path to main". The coder rules already forbid force-push and deletion (`.claude/skills/gabo-repository-host-github/SKILL.md:15-17`). |

## Open questions

Each question has a **Proposed (auto)** option with a one-line reason. None is accepted. Answers go into the spec, which the owner accepts (D-6).

| # | question | status |
|---|---|---|
| 1 | Required approving reviews on `main` and `dev`: zero or one? | open. **Proposed (auto): zero.** Reason: the owner's own merge of a PR into `main` is the human approval (D-2); GitHub forbids approving one's own PR (`docs/cloud-agent-flow.md:17`), and every agent PR uses the owner's login, so one required approval blocks every merge until a second human exists. Option: one approval (needs a second human); code-owner review with a `CODEOWNERS` file (the playbook calls it "worthwhile", not required; none exists today). |
| 2 | Is there a bypass list, and does it include the owner's admin role? | open. **Proposed (auto): no bypass.** Reason: agents use the owner's admin login, so an admin bypass is a bypass for every agent. Option: a bypass for the admin role, which keeps the owner's direct pushes and gives agents the same path; a bypass with a named GitHub App or team, which needs a second identity that does not exist today. |
| 3 | What happens on a plan or service that lacks a rule, such as GitLab or a private repository without it? | open. **Proposed (auto): setup warns with one line per missing rule, names the local hook layer (intent 0028) as the only enforcement left, and continues.** Reason: a failed setup would block the repository's first wave. Option: refuse setup until the rule exists. |
| 4 | Is the protection offered at initial setup, and what if it is declined? | open. **Proposed (auto): offered as optional, with the rule list shown before any write; a decline is recorded in memory and setup does not ask again in that project.** Reason: the owner asked to recommend and offer it as optional. Option: recommend only, with no memory record. |
| 5 | Which labels are recovered: the set the tracker adapter reads, the retired set, or both? | open. **Proposed (auto): the labels the tracker adapter reads today (`status:in-progress`, `status:blocked`, `needs-user`); `mode:cloud` and `mode:local` are not recreated.** Reason: nothing reads the `mode:*` labels in the repository today (searched 2026-11-13). Option: recreate `mode:cloud` and `mode:local` too, with no reader. |
| 6 | Does `gabo check` read the protection? | open. **Proposed (auto): no. `gabo check` stays offline; setup and the adapter's read-back are the only checks.** Reason: `gabo check` validates files offline by design (`docs/product/intents/README.md`, Checks). Option: `gabo check` calls the service and warns when a rule is missing. |
| 7 | Where is the owner's decision on each rule recorded after setup? | open. **Proposed (auto): a decision record in memory (`gabo/decision/global/branch-protection`, per the gabo-memory records) holding the rule list and the read-back result; the rules themselves stay on the server.** Reason: the decision must survive a new session, and the server is the only place the rules are enforced. Option: re-read the rules at every wave start, which adds a service call per wave. |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#766 I0029: Branch protection on main and dev](https://github.com/testeos-z/gabo-ai/issues/766) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) |
| sibling intent, B | [0028 Deterministic hooks as locks](../0028-deterministic-hooks-as-locks/intent.md), [#765](https://github.com/testeos-z/gabo-ai/issues/765) (the local layer; this intent is the server layer) |
| sibling intent, D | [0030 Agent evals when skills change](../0030-agent-evals-when-skills-change/intent.md), [#767](https://github.com/testeos-z/gabo-ai/issues/767) |
| related intents | [0011](../0011-docs-for-the-public-repository/intent.md) (its D2 deferred the `pi/pkg-up` and `NPM_TOKEN` change; its S4 records the public facts) |
| PRD | [PRD](../../PRD.md) (R9, R31) |
| TRD | [TRD](../../TRD.md) section 9.1 (`repository_host`, line 373), D13 (line 595), D18 (line 600) |
| spec | none yet; written only after the owner reviews this intent |
| plan | none yet |
| sources | commits `f15d326`, `94e3f48`, `1d59a61` in this repository; `gh api` read-only checks of 2026-11-13; [GitHub, About protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) (extraction); [GitHub, About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) (extraction, plan not stated); [GitLab, Protected branches](https://docs.gitlab.com/user/project/repository/branches/protected/) (extraction); [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook) (quotes byte-checked 2026-11-13) |
