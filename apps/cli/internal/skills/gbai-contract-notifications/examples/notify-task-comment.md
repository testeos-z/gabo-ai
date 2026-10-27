# notify-task-comment: example adapter for the gbai notifications contract

Fills the `notifications` slot with a GitHub mention in a comment on the task issue, so the recipient gets GitHub's own notification. Copy it to `.gbai/adapters/notify-task-comment.md` and point `slots.notifications.adapter` at it. Repository `R=<owner>/<repo>` (`repo.name` in `gbai.yaml`).

Notes:

- The comment goes on the task issue the message is about; messages about a whole wave go on the wave's parent issue.
- The owner is the repository owner: `gh repo view $R --json owner --jq .owner.login`.
- `developers` maps git emails to aliases. Each developer's GitHub login is the `author.login` of a commit with that email: `gh api "repos/$R/commits?author=<email>&per_page=1" --jq '.[0].author.login'`.

## Operations

### notify owner

```bash
gh issue comment <n> -R $R --body "@<owner login> <one line>, details: <link>"
```

### notify developer by alias

Find the emails whose alias is `<alias>` in `developers`, resolve one to a login as in the notes, then:

```bash
gh issue comment <n> -R $R --body "@<login> <one line>, details: <link>"
```

No email or no login found: report `unknown alias` and send nothing. Read back either comment with `gh issue view <n> -R $R --json comments --jq '.comments[-1].body'`.
