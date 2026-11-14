package config

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// agentsFile is the repository instruction file that every engine reads.
const agentsFile = repoRoot + "AGENTS.md"

// section returns the lines under a level-2 heading, up to the next one.
func section(t *testing.T, text, heading string) []string {
	t.Helper()
	lines := strings.Split(text, "\n")
	start := slices.Index(lines, heading)
	if start < 0 {
		t.Fatalf("AGENTS.md has no %q section", heading)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return lines[start+1 : end]
}

// healthyOutput reports whether the lines after a "$ " command, up to the
// next command or fence, show output and end with the line "exit: 0".
func healthyOutput(after []string) bool {
	var out []string
	for _, line := range after {
		if strings.HasPrefix(line, "$ ") || strings.HasPrefix(line, "```") {
			break
		}
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return len(out) >= 2 && out[len(out)-1] == "exit: 0"
}

// commandProblems compares the commands block with the check commands and
// returns one problem per missing command, per unknown command and per command
// without its healthy output.
func commandProblems(block, checks []string) []string {
	var problems []string
	listed := map[string]bool{}
	for i, line := range block {
		cmd, ok := strings.CutPrefix(line, "$ ")
		if !ok {
			continue
		}
		listed[cmd] = true
		if !slices.Contains(checks, cmd) {
			problems = append(problems, "block lists a command that is not in review.checks: "+cmd)
		}
		if !healthyOutput(block[i+1:]) {
			problems = append(problems, "no healthy output ending in exit: 0 under: "+cmd)
		}
	}
	for _, c := range checks {
		if !listed[c] {
			problems = append(problems, "review.checks command missing from the block: "+c)
		}
	}
	return problems
}

func TestAgentsMdCommands(t *testing.T) {
	t.Parallel()
	cfg, err := Load(repoRoot + ProjectFile)
	if err != nil {
		t.Fatalf("load gabo.yaml: %v", err)
	}
	b, err := os.ReadFile(agentsFile)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	block := section(t, string(b), "## Commands")
	if problems := commandProblems(block, cfg.Review.Checks); len(problems) > 0 {
		t.Fatalf("AGENTS.md commands block: %s", strings.Join(problems, "; "))
	}
	// A command added to review.checks must fail until the block lists it.
	added := append(slices.Clone(cfg.Review.Checks), "go -C apps/cli fmt ./...")
	if len(commandProblems(block, added)) == 0 {
		t.Fatal("a command added to review.checks is not reported as missing from the block")
	}
}

func TestAgentsMdRepeatedMistakes(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(agentsFile)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	block := section(t, string(b), "## Repeated mistakes")
	text := strings.ToLower(strings.Join(block, "\n"))
	if !strings.Contains(text, "two-times rule") {
		t.Error("repeated-mistakes section does not state the two-times rule")
	}
	if !strings.Contains(text, "second time") {
		t.Error("repeated-mistakes section does not say when the rule applies (a second time)")
	}
	entry := regexp.MustCompile(`^\s*([-*]|\d+\.)\s`)
	for _, line := range block {
		if entry.MatchString(line) {
			t.Errorf("repeated-mistakes section starts with an entry, want none: %q", line)
		}
	}
}
