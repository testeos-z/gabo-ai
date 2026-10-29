# US-012 Share one memory across all agents

| field | value |
|---|---|
| status | Proposed (auto) (owner request 2026-10-27, issue #107) |
| user | tech lead |
| priority | should |
| requirements | R37, R38 |
| depends on | US-001 |

## Story

As a tech lead, I want every agent of my project, on every developer's machine and in the cloud, to use the same memory, so that they all work from the same context.

## Acceptance criteria

- Given I configure gabo-ai for a project, When setup reaches the memory slot, Then it recommends a memory shared by all the project's agents except the reviewer, and recommends Engram with Engram Cloud sync, with its reason.
- Given I only use Claude Code, When setup lists the memory options, Then it also shows claude-mem and Mem0 and says for each whether it shares memory across machines and across developers.
- Given I am a solo developer, When I keep a local-only memory, Then setup accepts it.
- Given I choose a backend, When setup writes the config, Then it records only the memory adapter and the project name; setting up the backend, its credentials and its access stays with me.
- Given the memory slot is configured, When setup checks it, Then it writes and reads only under a dedicated scratch topic and deletes that probe afterwards, and on failure I see the error (for example HTTP 403 from a missing project grant).
- Given two agents on different machines share the project memory, When one saves a decision, Then the other finds it after sync.
- Given the reviewer runs on a PR, When it reviews, Then it does not read or write the shared project memory (R9).

## Out of scope

- Hosting or operating the memory server, its credentials or its access for the user.
- Shipped adapters for claude-mem and Mem0 (open question 13, section 12 Later).

## Notes

- App flow journey: to be linked when APP_FLOW covers the memory step.
- Backend setup note: see the Engram Cloud note in the PRD plan change section (issue #107).
