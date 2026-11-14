package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// Spec 0030 S19: the catalog carries a template of the three instruction-file
// blocks, and setup offers it for a project's instruction file. Setup never
// overwrites text the project already has. These tests read the real template
// and the real gabo.yaml, and use a fixture project directory for the file
// states (absent, complete, partial).

const (
	instructionCommands = "## Commands"
	instructionMistakes = "## Repeated mistakes"
	instructionFindings = "## Review findings"
)

// fixtureProjectChecks returns the review.checks commands of this repository's
// gabo.yaml, the project's own check commands for the fixture project.
func fixtureProjectChecks(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("../../../../gabo.yaml")
	if err != nil {
		t.Fatalf("read gabo.yaml: %v", err)
	}
	cfg, err := config.Parse(b)
	if err != nil {
		t.Fatalf("parse gabo.yaml: %v", err)
	}
	if len(cfg.Review.Checks) == 0 {
		t.Fatal("gabo.yaml has no review.checks")
	}
	return cfg.Review.Checks
}

// fixtureInstructionFile writes the project's instruction file into a fixture
// project directory when text is not empty, and returns what setup reads.
func fixtureInstructionFile(t *testing.T, text string) (string, bool) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if text == "" {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("fixture project unexpectedly has an instruction file: %v", err)
		}
		return "", false
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b), true
}

// offerSection returns the lines of the offer under one level-2 heading, up to
// the next one.
func offerSection(t *testing.T, offer, heading string) string {
	t.Helper()
	lines := strings.Split(offer, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " ") == heading {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("offer has no %q section:\n%s", heading, offer)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func TestInstructionTemplateHasTheThreeBlocks(t *testing.T) {
	t.Parallel()
	tpl, err := InstructionTemplate()
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	var at int
	for _, heading := range []string{instructionCommands, instructionMistakes, instructionFindings} {
		i := strings.Index(tpl[at:], heading+"\n")
		if i < 0 {
			t.Fatalf("template has no %q section, in that order", heading)
		}
		at += i + len(heading)
	}
	if !strings.Contains(tpl, "{{commands}}") {
		t.Error("commands block has no {{commands}} placeholder for the project's check commands")
	}
	mistakes := offerSection(t, tpl, instructionMistakes)
	for _, want := range []string{"two-times rule", "second time", "No entries yet."} {
		if !strings.Contains(strings.ToLower(mistakes), strings.ToLower(want)) {
			t.Errorf("repeated-mistakes section does not contain %q", want)
		}
	}
	findings := strings.ToLower(offerSection(t, tpl, instructionFindings))
	for _, want := range []string{"second time", "proposed", "no agent merges"} {
		if !strings.Contains(findings, want) {
			t.Errorf("review-findings rule does not contain %q", want)
		}
	}
}

func TestInstructionOfferAbsentFileOffersTemplateWithProjectCommands(t *testing.T) {
	t.Parallel()
	checks := fixtureProjectChecks(t)
	existing, exists := fixtureInstructionFile(t, "")
	offer, err := InstructionOffer(existing, exists, checks)
	if err != nil {
		t.Fatal(err)
	}
	for _, heading := range []string{instructionCommands, instructionMistakes, instructionFindings} {
		if !strings.Contains(offer, heading+"\n") {
			t.Errorf("offer for an absent file misses %q", heading)
		}
	}
	block := offerSection(t, offer, instructionCommands)
	if strings.Contains(offer, "{{commands}}") {
		t.Error("offer still holds the {{commands}} placeholder")
	}
	var listed []string
	for _, l := range strings.Split(block, "\n") {
		if cmd, ok := strings.CutPrefix(l, "$ "); ok {
			listed = append(listed, cmd)
		}
	}
	if strings.Join(listed, "\n") != strings.Join(checks, "\n") {
		t.Errorf("commands block lists %q, want the project's review.checks %q", listed, checks)
	}
}

func TestInstructionOfferCompleteFileIsUnchanged(t *testing.T) {
	t.Parallel()
	checks := fixtureProjectChecks(t)
	full, err := InstructionOffer("", false, checks)
	if err != nil {
		t.Fatal(err)
	}
	// A project file that already holds the three sections, with its own text.
	text := "# Project\n\nOwn rules.\n\n" + full
	existing, exists := fixtureInstructionFile(t, text)
	offer, err := InstructionOffer(existing, exists, checks)
	if err != nil {
		t.Fatal(err)
	}
	if offer != "" {
		t.Errorf("offer for a complete file is %q, want none so the file stays unchanged", offer)
	}
}

func TestInstructionOfferPartialFileOffersOnlyMissingSections(t *testing.T) {
	t.Parallel()
	checks := fixtureProjectChecks(t)
	full, err := InstructionOffer("", false, checks)
	if err != nil {
		t.Fatal(err)
	}
	// The project has its commands block already; only the other two are missing.
	text := "# Project\n\nOwn text, kept as it is.\n\n" + offerSection(t, full, instructionCommands) + "\n"
	existing, exists := fixtureInstructionFile(t, text)
	offer, err := InstructionOffer(existing, exists, checks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(offer, instructionCommands+"\n") {
		t.Error("offer repeats the commands block the file already has")
	}
	for _, heading := range []string{instructionMistakes, instructionFindings} {
		if !strings.Contains(offer, heading+"\n") {
			t.Errorf("offer misses the missing section %q", heading)
		}
	}
	if !strings.HasPrefix(existing, "# Project\n\nOwn text, kept as it is.") {
		t.Error("fixture file text was altered before setup")
	}
}

func TestSetupOffersTheInstructionBlocksWithoutOverwriting(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		"instruction-blocks.md", "`instruction-blocks.md`",
		"never overwrites", "project's own check commands",
	})
}
