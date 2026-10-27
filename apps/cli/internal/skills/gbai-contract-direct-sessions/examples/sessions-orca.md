# sessions-orca: example adapter for the gbai direct_sessions contract

Fills the `direct_sessions` slot with Orca: an agent launched in an Orca-managed worktree, driven through its terminal handle. Copy it to `.gbai/adapters/sessions-orca.md` and point `slots.direct_sessions.adapter` at it.

Notes:

- The task worktree must be known to Orca: `orca worktree list --json`; select it with `--worktree path:<worktree path>`.
- Every command takes `--json`; the handle is the terminal handle in the result.
- A remote Orca host adds `--environment <name>` to every command; that host belongs to the `agent_host` adapter.

## Operations

### open session

```bash
orca agent launch --agent claude --worktree path:<worktree> --surface terminal --prompt "<brief>" --json
```

Keep the returned terminal handle.

### send message

```bash
orca terminal send --terminal <handle> --text "<message>" --enter --wait-submit 10 --json
```

After an ambiguous failure, reissue the same command with the reported `--retry-request <id>`; never resend blindly.

### read state

```bash
orca terminal show --terminal <handle> --json
orca terminal read --terminal <handle> --screen --json
```

### close session

```bash
orca terminal close --terminal <handle> --tab --json
orca terminal list --json
```

The handle must no longer appear in the list.
