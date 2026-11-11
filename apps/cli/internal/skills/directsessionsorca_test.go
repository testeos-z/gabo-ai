package skills

import (
	"strings"
	"testing"
)

// The Orca direct-sessions adapter (intent 0019, plan T2): it takes the two
// handle fields of the contract, adds --environment for a remote host only,
// refuses unknown hosts, and needs the owner's authorization for remote use.

// remoteFlag is the only form of --environment the adapter may write: the name
// of the env var that the agent_host adapter declares, never a secret value.
const remoteFlag = `--environment "$GABO_ORCA_ENVIRONMENT"`

// Intent 0019 S2, PD3 (plan T2): the adapter takes the host and handle fields;
// the literal local names the local host; a remote host is the Orca environment
// that agent_host reports, named by GABO_ORCA_ENVIRONMENT.
func TestDirectSessionsOrcaTakesTheTwoHandleFields(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	sectionRequire(t, "## Handle", contractSection(doc, "## Handle"), []string{
		"A handle has two named fields, `host` and `handle`, never one joined string.",
		"`host` is `local` for the local host, the literal that `gabo-contract-direct-sessions` names; it adds no `--environment`.",
		"A remote `host` is the Orca environment name that `agent_host` reports for where the session runs (`report where agents run`: reached through Orca environment)",
		"The adapter reads that name from the env var `GABO_ORCA_ENVIRONMENT` and never a secret value",
		"`handle` is the terminal handle that Orca returned for the session.",
	})
}

// Intent 0019 S2, S4 (plan T2): a host that is neither local nor the Orca
// environment in GABO_ORCA_ENVIRONMENT is refused before any orca command runs;
// the host of a remote handle selects the target, so a mismatch or an unset or
// empty variable sends and reads nothing. The adapter is prose that an agent
// follows, so these tests pin the rule text.
func TestDirectSessionsOrcaRefusesUnknownHosts(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	sectionRequire(t, "## Handle", contractSection(doc, "## Handle"), []string{
		"A `host` that is neither `local` nor the Orca environment in `GABO_ORCA_ENVIRONMENT` is an unknown host: it is refused before any `orca` command, and nothing is sent or read.",
		"A remote `host` must equal the value of `GABO_ORCA_ENVIRONMENT`; a mismatched `host` is refused before any `orca` command, so a handle of host B never runs against host A.",
		"If `GABO_ORCA_ENVIRONMENT` is unset or empty, refuse before any `orca` command, name the missing variable, and send and read nothing.",
		"This adapter drives Orca terminals only: a session that `gabo-agent-host-local` started is a background process with a pid and a session id, not an Orca terminal, so its handle is not reachable through this adapter and is refused before any `orca` command.",
	})
}

// Intent 0019 S3 (plan T2): every orca terminal send, read and show of a remote
// handle carries --environment with the declared env var name.
func TestDirectSessionsOrcaRemoteCommandsCarryTheEnvironment(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	ops := contractSection(doc, "## Operations")
	requireIn(t, directSessions, ops, []string{
		"orca terminal send " + remoteFlag + " --terminal <handle> --text \"<message>\" --enter --wait-submit 10 --json",
		"orca terminal show " + remoteFlag + " --terminal <handle> --json",
		"orca terminal read " + remoteFlag + " --terminal <handle> --screen --json",
	})
	if got, want := strings.Count(ops, "--environment"), strings.Count(ops, remoteFlag); got != want {
		t.Errorf("%s: %d --environment flags in Operations, %d of them name $GABO_ORCA_ENVIRONMENT; every flag must be that env var name, never a value", directSessions, got, want)
	}
}

// Intent 0019 S3 (plan T2), existing behavior: the local form of open session,
// send message, read state and close session stays exactly as it is, with no
// --environment added.
func TestDirectSessionsOrcaLocalOperationsKeepTheirCommands(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	requireIn(t, directSessions, contractSection(doc, "## Operations"), []string{
		"orca terminal create --worktree path:<worktree> --title \"gabo-<task id>\" --command \"<agent cli>\" --json",
		"orca terminal send --terminal <handle> --text \"$(cat <brief file>)\" --enter --wait-submit 10 --json",
		"orca terminal send --terminal <handle> --text \"<message>\" --enter --wait-submit 10 --json",
		"orca terminal show --terminal <handle> --json",
		"orca terminal read --terminal <handle> --screen --json",
		"orca terminal close --terminal <handle> --tab --json",
		"orca terminal list --worktree path:<worktree> --json",
	})
}

// Intent 0019 S4, D5, PD4 (plan T2): remote use needs the owner's explicit
// authorization naming host, operation and credential or session; without it
// nothing is sent or read; host A never reaches host B; no ambient access.
func TestDirectSessionsOrcaRemoteUseNeedsAuthorization(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	sectionRequire(t, "## Remote Use", contractSection(doc, "## Remote Use"), []string{
		"Before any command to a remote `host`, the owner's explicit authorization, given in this session, must name the host, the operation (`send message` or `read state`) and the credential or session to use.",
		"With no authorization that names all three, send and read nothing and ask the caller for it.",
		"An authorization for host A never reaches host B.",
		"Never discover, reuse or create an Orca pairing, a credential, an SSH agent, a ControlMaster socket or an authenticated session; their presence is not permission.",
		"The `local` host needs no authorization.",
	})
	for _, op := range []string{"### send message", "### read state"} {
		sectionRequire(t, op, contractSection(doc, op), []string{
			"Check the handle under `## Handle` and, for a remote `host`, the authorization under `## Remote Use` before any command.",
			"For a remote `host`, check that it equals `GABO_ORCA_ENVIRONMENT` and that the variable is set and non-empty; otherwise refuse and run no command.",
		})
	}
}

// Intent 0019 S3 (plan T2): send message and read state work for a session of
// any engine, since they drive a terminal; open session stays limited to claude
// and pi with today's refusal line.
func TestDirectSessionsOrcaSendAndReadWorkForAnyEngine(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	for _, op := range []string{"### send message", "### read state"} {
		sectionRequire(t, op, contractSection(doc, op), []string{
			"It drives an Orca terminal, so it works for an Orca terminal session of any engine, including a `codex` session that `agent_host` started on a remote Orca host.",
		})
	}
	requireIn(t, directSessions, doc, []string{
		"This adapter opens `claude` and `pi` sessions only: a resolved `codex` or `opencode` is refused with `refused: <engine> has no Orca session command in gabo-direct-sessions-orca`.",
	})
	sectionRequire(t, "### open session", contractSection(doc, "### open session"), []string{
		"Opens on the local Orca only; a session on a remote host is started by `agent_host`.",
	})
}

// Intent 0019 S1, D6 (plan T2): close session closes only a handle this slot
// opened on the local host; any other handle is refused, naming stop agent
// session of the host that reported it.
func TestDirectSessionsOrcaCloseRefusesSessionsItDidNotOpen(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, directSessions)
	sectionRequire(t, "### close session", contractSection(doc, "### close session"), []string{
		"Closes only a handle this slot opened on the local host and recorded in the wave ledger.",
		"For any other handle, including every remote `host`, refuse and name `stop agent session` of the host that reported it",
	})
}
