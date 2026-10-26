# US-008 Trust what gets merged

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | tech lead |
| priority | must |
| requirements | R9, R10 |
| depends on | US-003 |

## Story

As a tech lead, I want every PR validated by an isolated reviewer and linked to its task, so that I can trust what gets merged and see its status in the tracker.

## Acceptance criteria

- Given a coder opens a PR, When the reviewer validates it, Then the reviewer scripts installed in the repository run tests, build and lint as in CI plus an independent agent review, and this validation is a mandatory merge gate that never merges into main.
- Given the reviewer runs, When it validates a PR, Then it uses its own clone, home and cache, never the coder's, comments the verdict on the PR, and the reviewer agent relays the verdict to the coder.
- Given a coder reports DONE or BLOCKED, When the cycle processes the report, Then the task status in the tracker reflects it.
- Given a PR exists, When I look at its task, Then the PR is linked to it (for example "Closes #n" for GitHub issues).

## Out of scope

- Reviewer on a separate server, GitHub Actions or GitLab CI in the first release (Later).
- Additional reviewer engines (Later).
- Replacing the human in final validation (non-goal).

## Notes

- App flow journey: [J8 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
