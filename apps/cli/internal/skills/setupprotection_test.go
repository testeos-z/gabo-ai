package skills

import (
	"strings"
	"testing"
)

// Intent 0029 T4 (spec S6 to S10; decision D5): gabo-setup offers branch
// protection after the repository_host slot is verified. It shows the rule
// list with the values from gabo.yaml before any host write, asks one
// question, records a decline once, reports each missing or not offered rule
// with the bypass statement, makes sure the three status labels exist, and
// saves one decision record. The live host run is owner work (T5), not pinned here.

// protectionStep returns the **Branch protection.** step of the setup skill.
func protectionStep(t *testing.T) string {
	t.Helper()
	return setupStep(t, "**Branch protection.**", "**Product model.**")
}

// S6: the step follows the smoke test that verifies the repository_host slot
// and comes before the product model; it shows the rule list with the values
// from gabo.yaml before any host write, recommends the operation as optional
// and asks one question.
func TestSetupProtectionIsOfferedAfterTheHostSlotIsVerified(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	_, steps, _ := strings.Cut(doc, "## Steps")
	smoke := strings.Index(steps, "**Smoke test.**")
	protect := strings.Index(steps, "**Branch protection.**")
	model := strings.Index(steps, "**Product model.**")
	if smoke < 0 || protect < smoke || model < protect {
		t.Fatalf("%s: the **Branch protection.** step must follow the smoke test and precede **Product model.**", setupSkill)
	}
	requireAll(t, protectionStep(t), []string{
		"after the `repository_host` slot is verified",
		"`protect branches`",
		"`repo.integration_branch`",
		"`repo.protection.required_approvals.integration`",
		"`repo.protection.required_approvals.main`",
		"read from `gabo.yaml`",
		"zero when the key is absent",
		"before any write to the host",
		"recommend",
		"optional",
		"one question",
		"accept or decline",
	})
}

// D4: when the approvals value on the integration branch is not zero, the
// rule list says the reviewer's merge stops, and why.
func TestSetupProtectionSaysTheReviewerMergeStopsOnApprovals(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"not zero",
		"reviewer's merge stops",
		"cannot approve its own pull request",
		"owner's login",
	})
}

// The rule list states the four rules of the protect branches contract (T2).
func TestSetupProtectionShowsTheRuleList(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"a pull request before a merge",
		"no force-push",
		"no deletion",
		"`reviewer/tests`",
		"`reviewer/review`",
		"`main` requires no status",
		"administrator role",
	})
}

// The start record (R28) is saved right after the memory slot is verified, so
// the owner's answer to the protection question does not delay it.
func TestSetupProtectionRunsAfterTheStartRecord(t *testing.T) {
	t.Parallel()
	_, steps, _ := strings.Cut(setupDoc(t), "## Steps")
	start := strings.Index(steps, "**Record the start (R28).**")
	protect := strings.Index(steps, "**Branch protection.**")
	if start < 0 || protect < start {
		t.Fatalf("%s: the **Branch protection.** step must come after **Record the start (R28).**", setupSkill)
	}
}

// S7: a decline is saved once, with its date; later runs do not ask and
// write nothing to the host.
func TestSetupProtectionRecordsADeclineOnce(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"decline",
		"`list decisions`", "`wave: none`", "`task: global`", "whose `what` starts with `branch protection`", "the most recent",
		"with its date",
		"does not ask again",
		"writes nothing to the host",
	})
}

// S8: one report line per missing or not offered rule, the remaining setup
// steps still run, and the bypass statement is always present, with the
// added line that the shell guard is not installed yet while 0028 has not shipped.
func TestSetupProtectionReportsMissingRulesAndTheBypass(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"one report line",
		"`missing`",
		"`not offered`",
		"remaining setup steps",
		"administrator bypass leaves the server open to any session holding the owner's login",
		"the local shell guard of intent 0028 is what stops agents on that path",
		"always",
		"not installed yet",
		"until intent 0028 has shipped",
	})
}

// S9: the three status labels are made sure of through the tasks adapter's
// label command, whether the owner accepts or declines; mode:cloud and
// mode:local are not created; nothing is written when all three exist.
func TestSetupProtectionMakesSureTheStatusLabelsExist(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"`status:in-progress`", "`status:blocked`", "`needs-user`",
		"through the `tasks` adapter's label command",
		"whether the owner accepts or declines",
		"`mode:cloud`", "`mode:local`", "not created",
		"nothing is written when all three exist",
	})
}

// S10: one decision record holds the rule list, the choice and one result per
// rule and branch, or the decline with its date; the lookup uses the same fields.
func TestSetupProtectionSavesOneDecisionRecord(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"`save decision`",
		"`memory`",
		"the rule list, the choice and one result per rule and branch",
		"the decline and its date",
		"`set`", "`missing`", "`not offered`",
		"`reason:`", "`mode:`", "`wave: none`", "`task: global`", "`resolver: owner`",
		"unverified",
	})
}

// S9 is a tracker write, not a host write: S7's "writes nothing to the host"
// does not stop the label check on a later run.
func TestSetupProtectionLabelsAreTrackerWrites(t *testing.T) {
	t.Parallel()
	requireAll(t, protectionStep(t), []string{
		"tracker labels, not a host write",
		"report `branch protection: not offered (<reason>)`, still run the labels check below, and go on",
		"report the labels as not run and go on",
	})
}

// The R28 step keeps its number and its text; the protection step follows it.
// The memory probe names the record step as step 7.
func TestSetupMemoryProbeNamesTheStartRecordAsStep7(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{"the R28 save in step 7, read back"})
}

// The step leaves the output contract with the branch protection result.
func TestSetupOutputContractNamesTheBranchProtectionResult(t *testing.T) {
	t.Parallel()
	_, out, ok := strings.Cut(setupDoc(t), "## Output Contract")
	if !ok {
		t.Fatalf("%s: no ## Output Contract section", setupSkill)
	}
	requireAll(t, out, []string{"branch protection"})
}
