package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Intent 0026 S2 (plan T5, decided D4 and PD4): gabo pins no model in the
// built Pi coder or in the Pi runtime text. A Pi coder takes the model of the
// owner's Pi configuration, and a launch the account refuses for its model
// stops the task with the engine's error on its ledger line, with no other
// model tried.
func TestPiBuiltCoderCarriesNoModelKey(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)

	// The built agent file is the fenced markdown block under "### Agents":
	// its frontmatter names the keys the built agent carries.
	_, build, ok := strings.Cut(doc, "\n### Agents\n")
	if !ok {
		t.Fatalf("%s: no ### Agents section", runtimePi)
	}
	_, fence, ok := strings.Cut(build, "```markdown\n---\n")
	if !ok {
		t.Fatalf("%s: no built agent frontmatter block", runtimePi)
	}
	front, _, ok := strings.Cut(fence, "\n---\n")
	if !ok {
		t.Fatalf("%s: built agent frontmatter is not closed", runtimePi)
	}
	keys := map[string]bool{}
	for _, line := range strings.Split(front, "\n") {
		if m := topLevelKey.FindStringSubmatch(line); m != nil {
			keys[m[1]] = true
		}
	}
	if !keys["name"] || !keys["tools"] {
		t.Fatalf("%s: built agent frontmatter keys %v, want name and tools (the block was not read)", runtimePi, keys)
	}
	if keys["model"] {
		t.Errorf("%s: built agent frontmatter carries a model key (S2)", runtimePi)
	}

	// The text states that no model is named and where a Pi coder's model comes from.
	requireIn(t, runtimePi, doc, []string{
		"The agent file carries no `model` key",
		"a Pi coder takes the model of the owner's Pi configuration",
		"coder launch the account refuses for its model stops the task with the engine's error on its ledger line",
		"no other model is tried",
	})
	if m := modelPin.FindString(doc); m != "" {
		t.Errorf("%s: names a model in its text (S2): %q", runtimePi, m)
	}
}

// Intent 0026 S2: the coder text carries the same refusal rule, so a coder
// reading only its own skill knows that no other model is tried.
func TestPiCoderRefusedModelStopsTheTaskOnTheLedger(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", catalog, []string{
		"coder launch the account refuses for its model stops the task with the engine's error on its ledger line",
		"no other model is tried",
	})
	if m := modelPin.FindString(catalog); m != "" {
		t.Errorf("gabo-coder/SKILL.md names a model in its text (S2): %q", m)
	}
}

// Intent 0026 S3, PD4: the coder contract has no approval-label gate. A gabo
// coder opens its PR without the `status:approved` label and stops on no
// approval gate, in the coder text and in the Pi runtime text.
func TestCoderOpensThePRWithoutTheApprovalGate(t *testing.T) {
	t.Parallel()
	const gate = "opens its PR without the `status:approved` label and stops on no approval gate"
	requireIn(t, "gabo-coder/SKILL.md", readCatalog(t, "gabo-coder/SKILL.md"), []string{gate})
	requireIn(t, runtimePi, runtimePiDoc(t), []string{gate})
}

// Intent 0026 S3, PD5: no gabo file, catalog skill, agent template or Codex
// copy, creates, adds or requires `status:approved`. A sentence that names the
// label is allowed only when it denies the gate, for example "never adds".
func TestNoGaboFileCreatesAddsOrRequiresTheApprovalLabel(t *testing.T) {
	t.Parallel()

	// The classifier must bite: a positive sentence is flagged, a denial is not.
	if len(approvalViolations("Add the `status:approved` label before the PR.")) == 0 {
		t.Fatal("approvalViolations misses a sentence that adds the label")
	}
	if len(approvalViolations("The approval gate needs status:approved on the issue.")) == 0 {
		t.Fatal("approvalViolations misses a sentence that requires the label")
	}
	if got := approvalViolations("A gabo coder opens its PR without the `status:approved` label. gabo never creates, adds or requires that label."); len(got) != 0 {
		t.Fatalf("approvalViolations flags a denial: %q", got)
	}

	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".md") && !strings.HasSuffix(f.Path, ".yaml") {
			continue
		}
		for _, s := range approvalViolations(string(f.Data)) {
			t.Errorf("catalog %s: %s", f.Path, s)
		}
	}

	codex := filepath.Join(repoRoot, codexPluginDir, "skills")
	err = filepath.WalkDir(codex, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, s := range approvalViolations(string(b)) {
			t.Errorf("codex copy %s: %s", p, s)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// topLevelKey matches a frontmatter key at the start of a line.
var topLevelKey = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*):`)

// modelPin matches a model named in text: a --model flag, a model key or
// assignment, or a sentence that pins a model.
var modelPin = regexp.MustCompile(`(?i)--model\b|(?m)^\s*model\s*:|\bmodel\s*[:=]\s*\S|\bpin\w*\s+(a\s+|the\s+)?model\b`)

// approvalSentence splits text into sentences on a full stop, a semicolon or
// a newline; it does not split on the colon of `status:approved`.
var approvalSentence = regexp.MustCompile(`\.\s|;\s|\n`)

var (
	approvalPositive = regexp.MustCompile(`(?i)\b(creat\w*|add\w*|requir\w*|wait\w*|needs?|must have|until)\b`)
	approvalDenial   = regexp.MustCompile(`(?i)\b(never|without|not|no|neither|nor|forbid\w*|cannot|don't|does not)\b`)
)

// approvalViolations returns each sentence that names `status:approved` and
// tells a coder to create, add, wait for or require it, without a denial.
func approvalViolations(doc string) []string {
	var out []string
	for _, s := range approvalSentence.Split(doc, -1) {
		if strings.Contains(s, "status:approved") && approvalPositive.MatchString(s) && !approvalDenial.MatchString(s) {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
