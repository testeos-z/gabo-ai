//go:build e2e

package e2e

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// TestMiniWaveInterruptedAndResumed runs wave 1 of a sample repository with
// two dependent tasks, interrupts it once the second coder is about to start,
// and resumes it with a fresh cycle that sees only the tracker and memory.
func TestMiniWaveInterruptedAndResumed(t *testing.T) {
	w := newWorld(t)

	if out := w.run(w.main, nil, w.bin, "check"); !strings.Contains(out, "check passed") {
		t.Fatalf("gabo check on the sample repository:\n%s", out)
	}

	runs := map[string]int{}
	type review struct {
		pr         int
		head, line string
	}
	var reviews []review
	recordReviews := func(c *cycle) {
		c.onVerdict = func(pr int, head, v string) {
			reviews = append(reviews, review{pr, head, v})
			// The same head again: one verdict per head (R26).
			if again := w.review(pr, head); !strings.Contains(again, "verdict=skipped") {
				t.Errorf("second review of PR #%d head %s: %s", pr, head, again)
			}
		}
	}

	// First run: one claim write is lost once, the re-read catches it (R24).
	w.tasks.drop["claim 1"] = true
	first := newCycle(w, "wave-1", runs)
	recordReviews(first)
	first.interrupt = func(task string) bool { return task == "T002" }
	if _, err := first.run(); !errors.Is(err, errInterrupted) {
		t.Fatalf("first run: want the interruption, got %v", err)
	}
	if t1, _ := w.tasks.readOne(1); t1.State != "closed" {
		t.Fatalf("T001 not closed before the interruption: %+v", t1)
	}
	if t2, _ := w.tasks.readOne(2); !slices.Contains(t2.Labels, "status:in-progress") {
		t.Fatalf("T002 not claimed before the interruption: %+v", t2)
	}

	// A worktree someone else made, outside the ledger.
	stray := filepath.Join(w.wts, "stray")
	w.run(w.main, nil, "git", "worktree", "add", "-q", stray, "-b", "stray", "origin/dev")

	second := newCycle(w, "wave-1", runs)
	recordReviews(second)
	report, err := second.run()
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	t.Run("resume does not rework DONE tasks", func(t *testing.T) {
		if runs["T001"] != 1 || runs["T002"] != 1 {
			t.Fatalf("coder launches: %v, want one per task", runs)
		}
	})

	t.Run("installed reviewer scripts give one verdict per head", func(t *testing.T) {
		if len(reviews) != 2 {
			t.Fatalf("reviews: %+v", reviews)
		}
		log, err := os.ReadFile(w.fx("calls.log"))
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range reviews {
			want := "VERDICT PR #" + itoa(r.pr) + " head " + r.head + " verdict=clean merge=merged"
			if r.line != want {
				t.Errorf("got %q, want %q", r.line, want)
			}
			if n := w.markers(r.pr, r.head); n != 1 {
				t.Errorf("PR #%d has %d verdict comments for %s", r.pr, n, r.head)
			}
			pinned := "gh pr merge " + itoa(r.pr) + " --repo example-org/sample --merge --match-head-commit " + r.head
			if !strings.Contains(string(log), pinned) {
				t.Errorf("no merge pinned to the reviewed head: %s", pinned)
			}
			// review.checks is `gabo check`, run with the real binary.
			if !regexp.MustCompile(`STATUS ` + r.head + ` reviewer/tests success`).Match(log) {
				t.Errorf("gabo check did not pass in the reviewer for %s", r.head)
			}
		}
	})

	t.Run("statuses updated and verified", func(t *testing.T) {
		for n, pr := range map[int]int{1: 101, 2: 102} {
			i, err := w.tasks.readOne(n)
			if err != nil {
				t.Fatal(err)
			}
			if i.State != "closed" || !slices.Contains(i.Comments, "DONE via PR #"+itoa(pr)) ||
				!slices.Contains(i.Comments, "claimed by gabo-cycle (wave wave-1)") || !slices.Contains(i.Comments, "PR #"+itoa(pr)) {
				t.Errorf("issue #%d: %+v", n, i)
			}
		}
		if len(first.unverified)+len(second.unverified) != 0 {
			t.Errorf("unverified writes: %v %v", first.unverified, second.unverified)
		}
		got := w.run(w.origin, nil, "git", "ls-tree", "-r", "--name-only", "dev", "src")
		if got != "src/t001.txt\nsrc/t002.txt\n" {
			t.Errorf("dev holds %q", got)
		}
	})

	t.Run("the ledger lists every created resource and no ledger worktree is left", func(t *testing.T) {
		ledger := map[string]string{}
		for _, e := range second.resources() {
			ledger[e.kind+" "+e.name] = e.state
		}
		for _, cmd := range w.gitLog {
			f := strings.Fields(cmd)
			if len(f) == 6 && f[0] == "worktree" && f[1] == "add" && f[3] != stray {
				for _, res := range []string{"worktree " + f[3], "branch " + f[5]} {
					if ledger[res] != "removed" {
						t.Errorf("%s: ledger state %q, want removed", res, ledger[res])
					}
				}
				if exists(f[3]) {
					t.Errorf("ledger worktree %s still exists", f[3])
				}
			}
		}
		if wts := w.worktrees(); !slices.Equal(wts, []string{stray}) {
			t.Errorf("worktrees left: %v", wts)
		}
		if heads := w.run(w.origin, nil, "git", "for-each-ref", "--format=%(refname:short)", "refs/heads"); heads != "dev\n" {
			t.Errorf("remote branches left: %q", heads)
		}
	})

	t.Run("a resource outside the ledger is reported and not deleted", func(t *testing.T) {
		if !exists(stray) {
			t.Fatal("the stray worktree was deleted")
		}
		if !strings.Contains(report, "worktree "+stray+" (not in the ledger, left in place)") {
			t.Errorf("report misses the stray worktree:\n%s", report)
		}
	})

	t.Run("report has the R29 fields and is the last trace entry", func(t *testing.T) {
		at := 0
		for _, h := range []string{"Wave", "Tasks", "Questions", "Interactions", "Unverified writes",
			"Resume disagreements", "Leftovers", "Plan-change proposals", "Denied board moves", "Comfort answer"} {
			i := strings.Index(report[at:], "## "+h+"\n")
			if i < 0 {
				t.Fatalf("section %q missing or out of order:\n%s", h, report)
			}
			at += i
		}
		for _, want := range []string{
			"T001 DONE PR #101 verdict=clean", "T002 DONE PR #102 verdict=clean",
			"## Interactions\n0\n", "## Unverified writes\nnone", "## Comfort answer\npending",
			"T001: where-does-the-feature-live -> src/t001.txt (resolver gabo-coder, auto decision record)",
		} {
			if !strings.Contains(report, want) {
				t.Errorf("report misses %q:\n%s", want, report)
			}
		}
		trace, err := w.mem.read("gabo/trace/wave-1")
		if err != nil || trace[len(trace)-1] != "report\n"+report {
			t.Errorf("report is not the last trace entry")
		}
	})

	t.Run("auto mode emits no prompt and records each decision", func(t *testing.T) {
		if first.prompts+second.prompts != 0 {
			t.Errorf("prompts: %d", first.prompts+second.prompts)
		}
		for _, id := range []string{"T001", "T002"} {
			d, err := w.mem.read("gabo/decision/wave-1/" + id + "/file-name")
			if err != nil || len(d) != 1 {
				t.Fatalf("%s decisions: %v %v", id, d, err)
			}
			for _, k := range []string{"what", "question", "choice", "why", "mode", "wave", "task", "resolver"} {
				if fields(d[0])[k] == "" {
					t.Errorf("%s decision lacks %s: %s", id, k, d[0])
				}
			}
		}
	})
}

// TestSkillsCarryTheRulesTheWaveFollows ties the steps played above to the
// skill text the real agents follow, so the two cannot drift apart silently.
func TestSkillsCarryTheRulesTheWaveFollows(t *testing.T) {
	read := func(name string) string {
		b, err := fs.ReadFile(skills.Catalog(), name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for skill, rules := range map[string][]string{
		"gabo-cycle": {
			"skips DONE tasks",
			"Never delete anything outside the ledger; a resource you find that is not in it is reported, never removed.",
			"never `--force`",
			"Every write is re-read (R24)",
			"the cycle never writes DONE",
			"`gabo/ledger/<wave>`",
			"`gabo/trace/<wave>`",
		},
		"gabo-coder": {
			"`git worktree remove <wt>` succeeds, never `--force`",
			"You own your task to the end (D23)",
		},
		"gabo-wave-report": {"**Interactions**", "**Leftovers**", "**Unverified writes**", "**Comfort answer**"},
	} {
		text := read(skill)
		for _, r := range rules {
			if !strings.Contains(text, r) {
				t.Errorf("%s no longer says %q", skill, r)
			}
		}
	}
}
