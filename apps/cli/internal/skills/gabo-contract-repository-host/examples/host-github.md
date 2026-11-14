# host-github: example adapter for the gabo repository_host contract

Fills the `repository_host` slot with GitHub through `gh` and `git`. Copy it to `.gabo/adapters/host-github.md` and point `slots.repository_host.adapter` at it. Repository `R=<owner>/<repo>`, integration branch `B=<repo.integration_branch>` (for example `dev`).

Branches are created in the task worktree with `git switch -c <type>/<issue>-<slug> origin/$B` and pushed with `git push -u origin HEAD`.

## Operations

### open PR

```bash
gh pr create -R $R --base $B --head <branch> --title "<conventional title>" --body-file <tmp>
```

The body holds a summary, `Refs #<issue>` and the verification run.

### comment on PR

```bash
gh pr comment <pr> -R $R --body-file <tmp>
```

### read PR status and checks

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefOid,mergeable,mergeStateStatus,statusCheckRollup,comments
```

### merge into integration branch

Check `baseRefName` equals `$B` first and refuse otherwise, then:

```bash
gh pr merge <pr> -R $R --merge --match-head-commit <sha>
gh pr view <pr> -R $R --json state,mergeCommit --jq '[.state,.mergeCommit.oid]|@tsv'
```

The read-back must show `MERGED`; its `mergeCommit.oid` is the merged SHA to return.

### delete branch

After the merge only, with the task worktree already removed:

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefName,headRefOid
git branch -d <branch>
git ls-remote --heads origin <branch>
git push origin --delete <branch>
```

1. The read must show `MERGED`, base `$B` and head `<branch>`; otherwise refuse.
2. Local first, with lower-case `-d` while the local `origin/<branch>` tracking ref still exists (no `fetch --prune` before it): git checks the branch against that upstream ref; the merge commit keeps the pushed head reachable from `dev`, and unpushed commits still refuse. A refusal is a leftover; never force it.
3. Remote only if `ls-remote` shows exactly `headRefOid`; no line means nothing to delete; any other SHA is a refusal and a leftover.

### protect branches

Uses repository rulesets (not classic branch protection): one ruleset per branch, named `gabo-protect-<branch>`, so a rerun finds it by name. Branches: `<branch>` is `$B` and `main`. The rule list, as the contract states it:

- On both branches: a pull request before a merge, no force-push and no deletion.
- On `repo.integration_branch` only (`$B`): the statuses `reviewer/tests` and `reviewer/review` required. `main` requires no status.
- On each branch: the approving reviews count from `repo.protection.required_approvals.integration` (`$B`) or `repo.protection.required_approvals.main`; zero when the key is absent.
- On both branches: a bypass for the administrator role, which names that role and no user.

1. Approving reviews per branch, from `gabo.yaml` with a YAML reader (for example `yq -r '.repo.protection.required_approvals.integration // 0' gabo.yaml` and the same with `.main`); a missing key reads as `0`.
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

   On `$B` only, add this rule to `rules`; `main` has no `required_status_checks` rule, so it requires no status:

   ```json
   {"type": "required_status_checks", "parameters": {"strict_required_status_checks_policy": false, "required_status_checks": [{"context": "reviewer/tests"}, {"context": "reviewer/review"}]}}
   ```

   `actor_id` 5 is the repository administrator role (`RepositoryRole`); the bypass names that role only. The required statuses carry no `integration_id`, so any login that posts `reviewer/tests` or `reviewer/review` satisfies them; agents use the owner's login (D6), so that gap is accepted.

4. Write: no ruleset found, `gh api -X POST repos/$R/rulesets --input <file>`; one found, `gh api -X PUT repos/$R/rulesets/<id> --input <file>` with the same body; more than one found, write nothing and report `missing` with the ids.
5. Not offered: when the host refuses the write and names one rule type or the bypass, remove that rule from the body, resend, and report that rule `not offered` with the host's reason. Repeat for each rule the host names, at most once per rule in the list. A refusal that names no rule is `missing` with the host's reason, with no retry.
6. Read back and compare with the rule list, one result per rule and branch (`set` or `missing`):

   ```bash
   gh api repos/$R/rulesets/<id> --jq '{name,enforcement,conditions,bypass_actors,rules:[.rules[]|{type,parameters}]}'
   ```

Running it again with the same config: the ruleset is found by name, the PUT replaces it with the same body, and the read-back is identical; no second ruleset is created.
