---
name: gbai-coder
description: "Trigger: gbai coder, implement issue, worktree coder, open PR, fix review findings. Rules for a coder session working one issue in its own worktree."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use in a coder session that implements one issue in its own git worktree.

## Hard Rules

- Never merge, approve, or set commit statuses. Only the reviewer does.
- Never post a comment, PR body or review that contains the reviewer's verdict marker (`<!-- gbai-reviewer-bot`), even quoted; the reviewer posts with the same login and only its marker comment is the verdict.
- Test-first (ODD): observe a failing test, implement, refactor green. Work-unit commits, Conventional Commits, no AI attribution. Use gentle-ai receipt-driven review when it is enabled.
- Open the PR to the integration branch (never `main`) with label `mode:cloud` when `GBAI_MODE=cloud`, else `mode:local`.
- After every push to the PR, run once: `gbai review request --pr <n> --sha <pushed head> --notify <your own durable mailbox>`. Never wait or poll for the result.
- React only to verdict messages that arrive in your Orca inbox. A message is a notification; the bot's comment on GitHub is the source of truth.
- Never force-push; never delete branches you did not create.

## Clean Worktree and Resources

- Write every scratch, draft, debug or probe file in a `mktemp -d` dir, never inside the worktree.
- Before opening or updating the PR, run `git -C <wt> status --porcelain --ignored=matching`. The worktree may hold only committed work plus rebuildable build output. Report any other leftover under `leftovers:`; never delete or move it.
- Test containers: reuse one per issue (start the existing one). Never create a second one to dodge a polluted one without reporting it. List every container you created under `containers:`. Never remove containers; the owner does.
- Heavy suites: never run the same long suite twice at once; kill your own older runs first; respect the configured `parallelism`.
- Committed migrations are immutable: fix them with a new migration, never by rewriting history.

## Decision Gates

| Verdict message | Action |
|---|---|
| `verdict=clean merge=merged` | Done; report and end. |
| `verdict=clean` other merge | Report "needs human merge" and end. |
| `verdict=blocking` | Fix round: read the bot's latest PR comment, fix, push, request review again. |
| `verdict=error` | Request review again once; on a second error label `needs-user`. |
| `verdict=skipped` | The head moved; request review for the current head. |

Max 4 fix rounds; then label the issue `needs-user`, report BLOCKED and end.

## Output Contract

End with `HANDED`, `DONE` or `BLOCKED`, the PR number and the last reviewed SHA, then always:

```
leftovers: <none | one path per line from status --ignored=matching that is not committed work or rebuildable output>
containers: <none | name and image of every container you created>
```
