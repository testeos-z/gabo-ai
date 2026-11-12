---
name: gabo-coder
description: "Trigger: gabo coder, implement issue, worktree coder, open PR, request review, fix review findings. Rules for a coder subagent working one task in its own worktree."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.3"
---
## Activation Contract

Load in a `gabo-coder` subagent launched by the main session (the assistant that runs the `gabo-cycle` skill) for one task in one git worktree. The assistant gives you the wave id, the task, the worktree, the branch and the mode; the worktree is `<repo-parent>/<repo-name>-worktrees/<wave>/<task>`, inside your wave's folder (intent 0013 S9, D6). Before reading code, load the `methodology` slot adapter under its fixed name `gabo-methodology` (`slots.methodology.adapter`) and build the way it says; this skill carries no methodology of its own. Use the `tasks` and `repository_host` adapters for issue and PR operations.

## Hard Rules

- Never merge, approve or set commit statuses; the reviewer does that.
- Never post a comment, PR body or review containing the reviewer marker (`gbai-reviewer-bot`), even quoted: the reviewer posts with the same login and only its marker comment is the verdict.
- Open the PR to `repo.integration_branch` (`dev`), never `main`, through `repository_host` (open PR), with a body that links the task as `Refs #<n>` on every base, never with a closing keyword: a closing keyword that merges into the default branch closes the task before the Task Close removes its worktree and labels, and the Task Close is the only writer of DONE (TRD D23). Never force-push; delete only your own task branch, and only in the Task Close.
- The PR body also holds one line `gabo check impact: <none, with the reason / updated in <file>>` (TRD 8.5): whether the change adds or alters something `gabo check` should verify (an agent file, a runtime skill, the neutral agent spec, a slot contract or a project file gabo writes) and where the check was updated, or why it needs no update.
- After every push, run the user-level reviewer script yourself from the main checkout, never from your worktree (the caller's `gabo.yaml` names the integration branch the review trusts), exactly as your agent file says, and act on its `VERDICT` line; the request is exactly `PR #<n> head <40-hex pushed sha>` and the PR comment holds the findings.
- Review engine (intent 0013 S11): only when `runtimes.reviewer` in `gabo.yaml` is `session`, the request is exactly `PR #<n> head <40-hex pushed sha> engine <engine>`, where `<engine>` is your session engine: the `gabo-engine:` value of your built agent (`{{engine}}`), the engine of the wave that launched you. With a fixed `runtimes.reviewer` never add an `engine` field; the script refuses it.
- A coder whose input carries `turn limit: <n>` stops at it: it commits nothing new and reports `STOPPED at turn limit` with its resources (worktree, branch, containers) and the state of the work; the cycle then pauses the wave. The limit never changes the fix-round cap below.
- A coder whose input carries `fix round limit: <n>` stops when it has used that many fix rounds in total without a clean review, counting from `fix rounds used: <k>` when the input carries it (a resumed or replacement coder; none means zero): it commits nothing new and reports `STOPPED at fix round limit` with its resources (worktree, branch, containers) and the state of the work; the cycle then pauses the wave. The limit never raises the fix-round cap: "Max 4 fix rounds" below is unchanged.
- Questions go back to the assistant in your outcome; never ask the owner directly.
- You own your task to the end (D23): after the merge you close it yourself (Task Close), and you are the only writer of DONE. Between hand-off and merge stay resumable: end your turn, never poll or wait for the merge.

## Clean Worktree and Resources

- Write every scratch, draft, debug or probe file in a `mktemp -d` dir, never inside the worktree.
- Before opening or updating the PR, run `git -C <wt> status --porcelain --ignored=matching`. The worktree may hold only committed work plus rebuildable build output. Report anything else under `leftovers:`; never delete or move it.
- Test containers: reuse one per task. List every container you created under `containers:`. Never remove containers; the cycle does it from the ledger.
- Heavy suites: never run the same long suite twice at once.
- Committed migrations are immutable: fix them with a new migration, never by rewriting history.

## Decision Gates

| VERDICT | Action |
|---|---|
| `verdict=clean merge=merged` | Run the Task Close now. |
| `verdict=clean merge=skipped`, PR `CONFLICTING` | Read the PR status through `repository_host` (read PR status): `mergeable` is `CONFLICTING`. Then merge `origin/dev` into the task branch in the worktree (never a rebase, never a force-push), resolve the conflict, run the task's checks, push, and request a new review for the new head; it counts as one fix round. A merge you cannot resolve ends BLOCKED with the question, never HANDED. |
| `verdict=clean` other merge | Report "needs human merge"; HANDED. You are resumed with `PR #<n> merged` once it merges. |
| `verdict=blocking` | Fix round: read the latest marker comment, fix, push, request review for the new head. |
| `verdict=error` | Request once more; a second error ⇒ BLOCKED. |
| `verdict=skipped` | The head moved; request review for the current head. |

Max 4 fix rounds; then label the issue `needs-user`, report BLOCKED with the question.

## Task Close

Run when resumed with `PR #<n> merged`, or after `verdict=clean merge=merged`. Like the review request, this runs outside the worktree: run it from the main checkout, the parent of `git -C <wt> rev-parse --path-format=absolute --git-common-dir`. Touch only the worktree and branch of this task that you reported in your outcome: they are entries of your wave's ledger, and the close touches only entries of your own wave's ledger (`gabo/ledger/<wave>`), per intent 0013 S9 and D6. A resource of another wave (a worktree under another `<repo-name>-worktrees/<wave>/` folder, or a branch or container of another wave's task) is neither removed nor reported as a leftover of this wave; `leftovers:` lists only resources of this task (its worktree and branch). Other coders share that checkout: on a git lock error (`*.lock` exists), retry once before you report a leftover. If your worktree or branch is already gone (a new coder launched after a session restart), the caller gives you the main checkout path in your input: run steps 1 and 2 from it without `git -C <wt>`, skip steps 3 and 4 for what is gone, and list nothing as a leftover for it. With the PR still open and only the worktree gone, recreate it there with `git worktree add <worktree path> <branch>` and continue; with the branch gone too, return the question.

1. Verify through `repository_host` (read status) that the PR is merged into `repo.integration_branch` from your branch, and note its merged head SHA. Not merged, or another base or head branch ⇒ change nothing, report HANDED "not merged".
2. Close the task issue (`tasks` change status DONE) and re-read it: it must show closed. Retry once, then report it under `unverified:`.
3. Remove the worktree only if all hold, else keep it and its local branch, list them under `leftovers:` and skip the local branch in step 4:
   - it is not in `worktree.keep` (`gabo.yaml`);
   - `git -C <wt> status --porcelain --ignored=matching`, run before any removal, shows only committed work and rebuildable output (removal deletes ignored files);
   - `git worktree remove <wt>` succeeds, never `--force`.
4. Delete the branch through `repository_host` (delete branch). A refused branch goes under `leftovers:`.
5. Containers stay: the cycle removes them from the ledger.

## Output Contract

The outcome to the cycle (TRD 9.2) holds, in this order:

- `status`: `HANDED` (review clean, you stay resumable), `BLOCKED` (with the reason) or, only after the Task Close, `DONE`;
- task id and the PR link (number and URL, the PR that carries `Refs #<n>`), with the last reviewed SHA;
- a one-paragraph summary of the change;
- open questions for the cycle, each with what it blocks;
- resources created: worktrees, branches, containers and test databases, so the cycle can add them to the wave ledger;
- fix rounds used (at most 4);
- the `methodology` adapter's per-task report.

DONE carries the close report: merged SHA, issue closed (verified or not), worktree removed or left, local and remote branch deleted or left. Then always:

```
leftovers: <none | one path per line from status --ignored=matching that is not committed work or rebuildable output>
containers: <none | name and image of every container you created>
unverified: <none | each tracker or host write whose re-read did not show it>
resources: <worktree path, branch, and any test database you created>
```
