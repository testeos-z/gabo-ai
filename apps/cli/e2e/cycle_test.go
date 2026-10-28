//go:build e2e

package e2e

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// This file plays one wave the way the gabo-cycle and gabo-coder skills
// describe it (Execution Steps, Task Close), with deterministic stand-ins for
// the agents. The behavior lives in the skills; the test pins that following
// them on a real repository with the installed reviewer scripts gives the
// outcome the plan asks for (T034).

var errInterrupted = errors.New("wave interrupted")

type result struct {
	id      string
	issue   int
	status  string // DONE | HANDED | BLOCKED
	pr      int
	verdict string
	handle  string
}

// cycle is one run of the gabo-cycle subagent for one wave.
type cycle struct {
	w    *world
	wave string
	mode string
	// coderRuns counts coder launches per task across runs (owned by the test).
	coderRuns map[string]int
	// interrupt stops the run once a task's coder is about to launch.
	interrupt func(task string) bool
	// onVerdict runs after each review, before GitHub carries out a merge.
	onVerdict func(pr int, head, verdict string)
	// prompts counts questions put to the owner.
	prompts int

	results       map[string]*result
	unverified    []string
	disagreements []string
	leftovers     []string
}

func newCycle(w *world, wave string, runs map[string]int) *cycle {
	return &cycle{w: w, wave: wave, mode: "auto", coderRuns: runs, results: map[string]*result{},
		interrupt: func(string) bool { return false }, onVerdict: func(int, string, string) {}}
}

func (c *cycle) ledgerTopic() string { return "gabo/ledger/" + c.wave }
func (c *cycle) traceTopic() string  { return "gabo/trace/" + c.wave }

// save appends a memory entry and reads it back (R24): retried once, then
// listed as unverified.
func (c *cycle) save(topic, entry string) {
	for range 2 {
		c.w.must(c.w.mem.append(topic, entry))
		got, err := c.w.mem.read(topic)
		c.w.must(err)
		if len(got) > 0 && got[len(got)-1] == entry {
			return
		}
	}
	c.unverified = append(c.unverified, "memory "+topic+": "+entry)
}

// tracked runs a tracker write and re-reads the issue until ok holds (R24).
func (c *cycle) tracked(what string, n int, write func() error, ok func(issue) bool) {
	for range 2 {
		c.w.must(write())
		i, err := c.w.tasks.readOne(n)
		c.w.must(err)
		if ok(i) {
			return
		}
	}
	c.unverified = append(c.unverified, "tracker #"+itoa(n)+": "+what)
}

func (c *cycle) ledger(kind, name, task, state string) {
	c.save(c.ledgerTopic(), "kind="+kind+" name="+name+" creator=gabo-cycle task="+task+" state="+state)
}

func (c *cycle) trace(r *result) {
	c.save(c.traceTopic(), "task="+r.id+" issue="+itoa(r.issue)+" status="+r.status+" pr="+itoa(r.pr)+
		" verdict="+orNone(r.verdict)+" rounds=0 questions=none")
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// entry is one ledger resource with its latest state.
type entry struct{ kind, name, task, state string }

func fields(line string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Fields(line) {
		if k, v, ok := strings.Cut(f, "="); ok {
			m[k] = v
		}
	}
	return m
}

// resources folds the ledger into the latest state per resource, in order.
func (c *cycle) resources() []entry {
	lines, err := c.w.mem.read(c.ledgerTopic())
	c.w.must(err)
	var out []entry
	at := map[string]int{}
	for _, l := range lines {
		f := fields(l)
		key := f["kind"] + " " + f["name"]
		if i, ok := at[key]; ok {
			out[i].state = f["state"]
			continue
		}
		at[key] = len(out)
		out = append(out, entry{f["kind"], f["name"], f["task"], f["state"]})
	}
	return out
}

// run is one cycle run: resume, the wave, the fallback sweep and the report.
func (c *cycle) run() (string, error) {
	trace, err := c.w.mem.read(c.traceTopic())
	c.w.must(err)
	if len(trace) == 0 {
		c.save(c.traceTopic(), "wave="+c.wave+" mode="+c.mode+" start="+time.Now().UTC().Format(time.RFC3339))
	}
	// Step 1, resume (D17): tracker state plus ledger and trace.
	last := map[string]map[string]string{}
	for _, l := range trace {
		if f := fields(l); f["task"] != "" {
			last[f["task"]] = f
		}
	}
	all, err := c.w.tasks.readAll()
	c.w.must(err)
	for _, i := range all {
		t := last[i.id()]
		if i.State == "closed" {
			r := &result{id: i.id(), issue: i.Number, status: "DONE", handle: "coder-" + i.id()}
			if t != nil {
				r.pr, r.verdict = atoi(t["pr"]), t["verdict"]
			}
			c.results[r.id] = r
			continue
		}
		if t != nil && t["status"] != "LAUNCHED" && c.w.readPR(atoi(t["pr"])).Number == 0 {
			c.disagreements = append(c.disagreements, i.id()+": trace says "+t["status"]+" but the host has no PR")
			continue
		}
		for _, e := range c.resources() {
			if e.kind == "worktree" && e.task == i.id() && e.state == "live" {
				branch := c.ledgerBranch(i.id())
				if err := c.launch(i, e.name, branch); err != nil {
					return "", err
				}
			}
		}
	}
	// Steps 2-4: unblocked tasks, one at a time (parallelism 1).
	for {
		i, ok := c.nextUnblocked()
		if !ok {
			break
		}
		c.tracked("claim", i.Number, func() error { return c.w.tasks.claim(i.Number, "gabo-cycle", c.wave) },
			func(got issue) bool { return slices.Contains(got.Labels, "status:in-progress") })
		i, err = c.w.tasks.readOne(i.Number) // rule 1: current text
		c.w.must(err)
		wt := filepath.Join(c.w.wts, strings.ToLower(i.id()))
		branch := "feat/" + itoa(i.Number) + "-" + strings.ToLower(i.id())
		c.w.git(c.w.main, "worktree", "add", "-q", wt, "-b", branch, "origin/dev")
		c.ledger("worktree", wt, i.id(), "live")
		c.ledger("branch", branch, i.id(), "live")
		c.ledger("handle", "coder-"+i.id(), i.id(), "live")
		c.trace(&result{id: i.id(), issue: i.Number, status: "LAUNCHED"})
		if err := c.launch(i, wt, branch); err != nil {
			return "", err
		}
	}
	c.sweep()
	return c.report(), nil
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func (c *cycle) ledgerBranch(task string) string {
	for _, e := range c.resources() {
		if e.kind == "branch" && e.task == task {
			return e.name
		}
	}
	return ""
}

// nextUnblocked is the tasks op "read unblocked": open, no status label and
// every blocker closed, in plan order.
func (c *cycle) nextUnblocked() (issue, bool) {
	all, err := c.w.tasks.readAll()
	c.w.must(err)
	closed := map[int]bool{}
	for _, i := range all {
		closed[i.Number] = i.State == "closed"
	}
	sort.Slice(all, func(a, b int) bool { return all[a].Title < all[b].Title })
	for _, i := range all {
		labeled := slices.ContainsFunc(i.Labels, func(l string) bool { return strings.HasPrefix(l, "status:") || l == "needs-user" })
		if i.State != "open" || labeled || !all1(i.BlockedBy, closed) {
			continue
		}
		return i, true
	}
	return issue{}, false
}

func all1(ns []int, ok map[int]bool) bool {
	for _, n := range ns {
		if !ok[n] {
			return false
		}
	}
	return true
}

// launch runs the coder for one task and records its outcome (step 4).
func (c *cycle) launch(i issue, wt, branch string) error {
	if c.interrupt(i.id()) {
		return errInterrupted
	}
	r := c.coder(i, wt, branch)
	c.results[r.id] = r
	if r.status == "DONE" {
		c.ledger("worktree", wt, r.id, "removed")
		c.ledger("branch", branch, r.id, "removed")
		c.ledger("handle", r.handle, r.id, "removed")
	}
	c.trace(r)
	return nil
}

// coder is the gabo-coder subagent: one change, push, PR, review, and the
// Task Close after a clean merged verdict (D23).
func (c *cycle) coder(i issue, wt, branch string) *result {
	c.coderRuns[i.id()]++
	r := &result{id: i.id(), issue: i.Number, status: "HANDED", pr: 100 + i.Number, handle: "coder-" + i.id()}
	// A choice the coder cannot settle from the task: auto mode records a
	// decision instead of asking the owner.
	if c.mode == "auto" {
		c.save("gabo/decision/"+c.wave+"/"+i.id()+"/file-name",
			"what=file-name question=where-does-the-feature-live choice=src/"+strings.ToLower(i.id())+".txt "+
				"why=one-file-per-task mode=auto wave="+c.wave+" task="+i.id()+" resolver=gabo-coder")
	} else {
		c.prompts++
	}
	c.w.must(os.MkdirAll(filepath.Join(wt, "src"), 0o755))
	c.w.must(os.WriteFile(filepath.Join(wt, "src", strings.ToLower(i.id())+".txt"), []byte(i.Title+"\n"), 0o644))
	c.w.git(wt, "add", "-A")
	c.w.git(wt, "commit", "-qm", "feat: "+strings.SplitN(i.Title, ": ", 2)[1])
	c.w.git(wt, "push", "-q", "-u", "origin", "HEAD")
	head := c.w.git(wt, "rev-parse", "HEAD")
	c.w.openPR(r.pr, branch, head)
	c.tracked("link PR", i.Number, func() error { return c.w.tasks.comment(i.Number, "PR #"+itoa(r.pr)) },
		func(got issue) bool { return slices.Contains(got.Comments, "PR #"+itoa(r.pr)) })
	v := c.w.review(r.pr, head)
	c.onVerdict(r.pr, head, v)
	r.verdict = fields(v)["verdict"]
	if r.verdict == "" {
		c.w.t.Fatalf("VERDICT line without verdict=: %s", v)
	}
	if !strings.HasSuffix(v, "verdict=clean merge=merged") {
		return r
	}
	c.w.mergePR(r.pr) // GitHub carries out the reviewer's pinned merge
	if c.taskClose(i, r, wt, branch, head) {
		r.status = "DONE"
	}
	return r
}

// taskClose follows the coder skill's Task Close from the main checkout.
func (c *cycle) taskClose(i issue, r *result, wt, branch, head string) bool {
	pr := c.w.readPR(r.pr)
	if pr.State != "MERGED" || pr.BaseRef != "dev" || pr.HeadRef != branch {
		return false
	}
	c.tracked("close", i.Number, func() error { return c.w.tasks.close(i.Number, r.pr) },
		func(got issue) bool { return got.State == "closed" })
	if strings.TrimSpace(c.w.git(wt, "status", "--porcelain", "--ignored=matching")) != "" {
		c.leftovers = append(c.leftovers, "worktree "+wt+" (uncommitted files)")
		return true
	}
	c.w.git(c.w.main, "fetch", "-q", "origin", "dev")
	c.w.git(c.w.main, "worktree", "remove", wt)
	c.w.git(c.w.main, "branch", "-d", branch)
	// The remote branch goes only while it still points at the merged head.
	if remote, _, _ := strings.Cut(c.w.git(c.w.main, "ls-remote", "--heads", "origin", branch), "\t"); remote == head {
		c.w.git(c.w.main, "push", "-q", "origin", "--delete", branch)
	}
	return true
}

// sweep is the wave-close fallback sweep: only ledger entries no coder
// closed, only after the PR merged; anything outside the ledger is reported
// and left in place.
func (c *cycle) sweep() {
	known := map[string]bool{}
	for _, e := range c.resources() {
		known[e.kind+" "+e.name] = true
		if e.state != "live" || e.kind == "handle" {
			continue
		}
		r := c.results[e.task]
		if r == nil || c.w.readPR(r.pr).State != "MERGED" {
			continue
		}
		var err error
		switch e.kind {
		case "worktree":
			_, err = c.w.try(c.w.main, nil, "git", "worktree", "remove", e.name)
		case "branch":
			_, err = c.w.try(c.w.main, nil, "git", "branch", "-d", e.name)
		}
		state := "removed"
		if err != nil {
			state = "leftover"
			c.leftovers = append(c.leftovers, e.kind+" "+e.name+" (REFUSED)")
		}
		c.ledger(e.kind, e.name, e.task, state)
	}
	for _, p := range c.w.worktrees() {
		if !known["worktree "+p] {
			c.leftovers = append(c.leftovers, "worktree "+p+" (not in the ledger, left in place)")
		}
	}
}

// report builds the TRD 9.2 wave report in the gabo-wave-report format and
// saves it as the last trace entry.
func (c *cycle) report() string {
	trace, err := c.w.mem.read(c.traceTopic())
	c.w.must(err)
	var b strings.Builder
	section := func(title string, lines []string) {
		b.WriteString("## " + title + "\n")
		if len(lines) == 0 {
			lines = []string{"none"}
		}
		b.WriteString(strings.Join(lines, "\n") + "\n\n")
	}
	start := fields(trace[0])
	section("Wave", []string{start["wave"] + ", mode " + start["mode"] + ", start " + start["start"]})
	var tasks, questions []string
	ids := make([]string, 0, len(c.results))
	for id := range c.results {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		r := c.results[id]
		tasks = append(tasks, r.id+" "+r.status+" PR #"+itoa(r.pr)+" verdict="+orNone(r.verdict)+" rounds=0 handle="+r.handle)
		d, err := c.w.mem.read("gabo/decision/" + c.wave + "/" + id + "/file-name")
		c.w.must(err)
		for _, rec := range d {
			f := fields(rec)
			questions = append(questions, id+": "+f["question"]+" -> "+f["choice"]+" (resolver "+f["resolver"]+", auto decision record)")
		}
	}
	section("Tasks", tasks)
	section("Questions", questions)
	section("Interactions", []string{itoa(c.prompts)})
	section("Unverified writes", c.unverified)
	section("Resume disagreements", c.disagreements)
	section("Leftovers", c.leftovers)
	section("Plan-change proposals", nil)
	section("Denied board moves", nil)
	section("Comfort answer", []string{"pending"})
	out := strings.TrimSpace(b.String())
	c.save(c.traceTopic(), "report\n"+out)
	return out
}
