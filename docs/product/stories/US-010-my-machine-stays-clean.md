# US-010 My machine stays clean

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | should |
| requirements | R13 |
| depends on | US-003 |

## Story

As a solo developer, I want each wave to remove the worktrees and merged branches it created, so that leftovers do not fill my disk or need manual cleanup.

## Acceptance criteria

- Given a wave created worktrees and branches, When the wave closes, Then the worktrees and merged branches that wave created are removed.
- Given something created by the wave could not be cleaned, When the wave report is produced, Then the leftover is listed in it.
- Given several waves have run, When I check my machine, Then I do not have to clean up wave worktrees or merged branches by hand.

## Out of scope

- Removing resources the wave did not create (auto-mode limits, 10.2).
- Providing its own infrastructure (non-goal).

## Notes

- App flow journey: [J10 in APP_FLOW](../APP_FLOW.md).
- Evidence: the vulcam disk-full incident of 2026-10-22 (PRD section 2, Evidence), where loose worktrees, reviewer copies and stopped containers filled the disk and the owner cleaned up by hand.
