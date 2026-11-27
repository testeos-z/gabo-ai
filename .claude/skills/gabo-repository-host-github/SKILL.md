---
name: gabo-repository-host-github
description: "Trigger: gabo repository host, open PR, PR status, PR checks, merge into intent branch or dev, delete branch, protect branches, branch rules, rulesets. GitHub adapter for the gabo repository_host slot via gh and git."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Use whenever a gabo agent works with branches or pull requests. Fills the `repository_host` slot contract (TRD 9.1). Repository `repo.name`, integration branch `repo.integration_branch` from `gabo.yaml` (here `R=testeos-z/gabo-ai`, `dev`).

## Hard Rules

- Task PRs target their intent branch (`<NNNN-slug>`, the intent folder name); the intent PR targets the integration branch `dev`. Never open a PR to `main`, never merge into `main`, never push to `main`, `dev` or an intent branch directly (TRD D13).
- Merge only into the PR's accepted base (the intent branch or `dev`), only pinned to the head SHA that received a clean verdict, and only when the merge is yours to do (reviewer gate or explicit owner request). Coders never merge.
- Never force-push. Delete only branches the workflow created and that are merged (ledger, D15, D23). Never force-delete a branch (`-D`): delete the local branch with `-d` while the local `origin/<branch>` tracking ref still exists (no `fetch --prune` before it), so git checks it against that upstream ref (the merge commit keeps the pushed head reachable from `dev`), then the remote one. A `-d` refusal means unpushed work: leave it as a leftover.
- Re-read after every write: `gh pr view` must show the expected state.
- Never post the reviewer marker (`gbai-reviewer-bot`) in a PR body, comment or review.

## Operations

Commands come from this adapter's former operation table and the `repository_host` contract; `gh` flags and `--json` fields confirmed with `gh pr view --help` and `gh pr create --help` on gh 2.101.0. `R=testeos-z/gabo-ai` (`repo.name`), integration branch `dev` (`repo.integration_branch`).

### branch

Not a contract operation; kept for the coder. The intent branch is cut once per wave from `dev`:

```bash
git switch -c <NNNN-slug> origin/dev
git push -u origin HEAD
```

In the task worktree, the task branch starts from the intent branch:

```bash
git switch -c <type>/<issue>-<slug> origin/<NNNN-slug>
git push -u origin HEAD
```

### open PR

Body: summary, `Refs #<issue>`, verification run. Base is the intent branch for a task PR and `dev` for the intent PR.

```bash
gh pr create -R $R --base <base> --head <branch> --title "<conventional title>" --body-file <tmp>
```

Read back: `gh pr view <branch> -R $R --json number,state,baseRefName,headRefName,headRefOid` must show `OPEN`, base `<base>`, head `<branch>`; return `number`.

### comment on PR

```bash
gh pr comment <pr> -R $R --body-file <tmp>
```

Read back: `gh pr view <pr> -R $R --json comments --jq '.comments[-1].body'` must match the posted body.

### read PR status and checks

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefOid,mergeable,mergeStateStatus,statusCheckRollup,comments
```

### merge into integration branch

The accepted bases are `dev` and any intent folder name existing under `docs/product/intents/` on `dev` (read from the repository, never from the tracker). Check the PR's `baseRefName` first (`gh pr view <pr> -R $R --json baseRefName --jq .baseRefName`); refuse `main`, `master` and any other base. Then:

```bash
gh pr merge <pr> -R $R --merge --match-head-commit <sha>
```

Read back: `gh pr view <pr> -R $R --json state,mergeCommit --jq '[.state,.mergeCommit.oid]'` must show `MERGED`; return the merged SHA.

### sync from the integration branch (reverse merge)

Never merge `dev` directly into the intent branch and never touch `dev` from a task branch. When `dev` moved after the intent branch was cut, resolve the drift on a throwaway branch first (0032 S9):

```bash
git switch -c <NNNN-slug>-sync origin/<NNNN-slug>
git merge origin/dev
```

Resolve conflicts here, run the task's checks, push, and open a PR with base `<NNNN-slug>` through `open PR`; it merges under the same reviewer gate. When the sync PR is merged, the intent PR into `dev` follows. Never rebase, never force-push. When the intent branch already contains `dev` (no divergence), skip this operation.

### delete branch

After merge only: `gh pr view <pr> -R $R --json state,baseRefName,headRefName,headRefOid` must show `MERGED`, an accepted base (the intent branch or `dev`) and head `<branch>`; the worktree is already removed. This covers task branches and, at wave close, the intent branch itself. In this order:

1. Local, while the local `origin/<branch>` tracking ref still exists (a refusal is a leftover; never `-D`):

   ```bash
   git branch -d <branch>
   ```

2. Remote, only if `git ls-remote --heads origin <branch>` shows exactly `<headRefOid>`; absent means nothing to do; another SHA means refuse:

   ```bash
   git push origin --delete <branch>
   ```

Read back: `git branch --list <branch>` and `git ls-remote --heads origin <branch>` are empty.

### protect branches

Uses repository rulesets (not classic branch protection): one ruleset per branch, named `gabo-protect-<branch>`, so a rerun finds it by name. Branches: `dev` (`repo.integration_branch`), `main`, and each intent branch (`<NNNN-slug>`). The rule list, as the contract states it:

- On every protected branch: a pull request before a merge, no force-push and no deletion.
- On `repo.integration_branch` (`dev`) and on each intent branch: the statuses `reviewer/tests` and `reviewer/review` required. `main` requires no status.
- On each branch: the approving reviews count from `repo.protection.required_approvals.integration` (`dev` and intent branches) or `repo.protection.required_approvals.main`; zero when the key is absent.
- On every protected branch: a bypass for the administrator role, which names that role and no user.

1. Approving reviews per branch, from `gabo.yaml` with a YAML reader (for example `yq -r '.repo.protection.required_approvals.integration // 0' gabo.yaml` and the same with `.main`); a missing key reads as `0`.
   The key is accepted by `gabo check` once intent 0029 T3 lands; until then a value set by hand is rejected, and an absent key reads as `0`.
2. Find the ruleset of the branch by name; no output means none exists:

   ```bash
   gh api --paginate "repos/$R/rulesets?includes_parents=false" --jq '.[]|select(.name=="gabo-protect-<branch>")|.id'
   ```

3. Write the body to a file in `mktemp -d`, one per branch:

   ```json
   {
     "name": "gabo-protect-<branch>",
     "target": "branch",
     "enforcement": "active",
     "conditions": {"ref_name": {"include": ["refs/heads/<branch>"], "exclude": []}},
     "bypass_actors": [{"actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "always"}],
     "rules": [
       {"type": "pull_request", "parameters": {"required_approving_review_count": <count>, "dismiss_stale_reviews_on_push": false, "require_code_owner_review": false, "require_last_push_approval": false, "required_review_thread_resolution": false}},
       {"type": "non_fast_forward"},
       {"type": "deletion"}
     ]
   }
   ```

   On `dev` and on each intent branch, add this rule to `rules`; `main` has no `required_status_checks` rule, so it requires no status:

   ```json
   {"type": "required_status_checks", "parameters": {"strict_required_status_checks_policy": false, "required_status_checks": [{"context": "reviewer/tests"}, {"context": "reviewer/review"}]}}
   ```

   `actor_id` 5 is the repository administrator role (`RepositoryRole`); the bypass names that role only. The required statuses carry no `integration_id`, so any login that posts `reviewer/tests` or `reviewer/review` satisfies them; agents run with the owner's login, so that gap is accepted; the required contexts are D6.

4. Write: no ruleset found, `gh api -X POST repos/$R/rulesets --input <file>`; one found, `gh api -X PUT repos/$R/rulesets/<id> --input <file>` with the same body; more than one found, write nothing and report `missing` with the ids.
5. Not offered: when the host refuses the write and names one rule type or the bypass, remove that rule from the body, resend, and report that rule `not offered` with the host's reason. Repeat for each rule the host names, at most once per rule in the list. A refusal that names no rule is `missing` with the host's reason, with no retry.
6. Read back and compare with the rule list, one result per rule and branch (`set` or `missing`); a rule already reported `not offered` in step 5 keeps that result and is not read back:

   ```bash
   gh api repos/$R/rulesets/<id> --jq '{name,enforcement,conditions,bypass_actors,rules:[.rules[]|{type,parameters}]}'
   ```

Running it again with the same config: the ruleset is found by name, the PUT replaces it with the same body, and the read-back is identical; no second ruleset is created.

At wave close, before deleting the intent branch: PUT the intent branch ruleset without the `deletion` rule (same body otherwise), read it back, then delete the branch through `delete branch`. The `dev` and `main` rulesets are never released.

## Output Contract

Return PR number, base, head SHA, state, and for merges the merged SHA; report any refused operation with its reason.
