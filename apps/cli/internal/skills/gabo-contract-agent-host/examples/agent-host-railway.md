# agent-host-railway: example adapter for the gbai agent_host contract

Fills the cloud slot `agent_host` with one Railway service that runs an Orca server, reached from the owner's machine through a saved Orca environment. Copy it to `.gbai/adapters/agent-host-railway.md`, fill the fields and point `slots.agent_host.adapter` at it.

## Fields

| field | value |
|---|---|
| `server.railway.project_id` | `<railway project id>` |
| `server.railway.environment_id` | `<railway environment id>` |
| `server.railway.service_id` | `<railway service id>` |
| `users.orca` | name `orca`, home `/home/orca`, uid `1001`, extra_path `/home/orca/.local/bin` |
| `bot.login` | `<bot GitHub login>` for the isolated reviewer, or `not used` in single-user mode |
| `sessions.coder.prompt_file` | `prompts/coder.md` |
| `sessions.reviewer.prompt_file` | `prompts/reviewer.md` |
| `sessions.assistant.prompt_file` | `prompts/assistant.md` |
| `sessions.local_orca` | `orca` |
| `/etc/gbai/gbai.yaml` | `not used`: its values are the fields above |
| Railway token | env var `RAILWAY_TOKEN`, never its value |
| Orca environment | saved name `gbai-railway` (`orca environment add` with the server's pairing code) |

## Operations

### start agent session

Check the service is up, then launch the role's agent with the content of its prompt file:

```bash
railway status --json
orca agent launch --environment gbai-railway --agent claude --worktree path:<worktree> --surface terminal --prompt "<prompt file content>" --json
```

Keep the returned terminal handle.

### stop agent session

```bash
orca terminal close --environment gbai-railway --terminal <handle> --tab --json
orca terminal list --environment gbai-railway --json
```

The handle must no longer appear in the list.

### report where agents run

Host: Railway service `<service_id>` in project `<project_id>`; OS user `orca`; reviewer mode `isolated` when `bot.login` is set, otherwise `single-user`. Sessions: `orca terminal list --environment gbai-railway --json`.
