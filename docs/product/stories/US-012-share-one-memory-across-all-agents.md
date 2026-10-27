# US-012 Share one memory across all agents

| field | value |
|---|---|
| status | Proposed (auto) (owner request 2026-10-27, issue #107) |
| user | tech lead |
| priority | must |
| requirements | R37, R38 |
| depends on | US-001 |

## Story

As a tech lead, I want every agent of my project, on every developer's machine and in the cloud, to use the same memory, so that they all work from the same context.

## Acceptance criteria

- Given I configure gabo-ai for a project, When setup reaches the memory slot, Then it tells me the project needs a memory shared by all its agents and recommends Engram with Engram Cloud sync, with its reason.
- Given I only use Claude Code, When setup lists the memory options, Then it also shows claude-mem and Mem0 and says for each whether it shares memory across machines and across developers.
- Given I choose a backend, When setup writes the config, Then it records the backend, its endpoint and the project name, and no credential value.
- Given the memory slot is configured, When setup verifies it, Then a test write and read succeed from the agent being configured, or I see the failure and its likely cause (for example HTTP 403 from a missing project grant).
- Given two agents on different machines use the project, When one saves a decision, Then the other finds it after sync.

## Out of scope

- Hosting or operating the memory server for the user.
- Shipped adapters for claude-mem and Mem0 (open question 13, section 12 Later).

## Notes

- App flow journey: to be linked when APP_FLOW covers the memory step.
- Setup gotcha: see the Engram Cloud note in the PRD plan change section (issue #107).
