package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Intent stage control (S7, S14): each docs/product/intents/<NNNN-slug>/
// folder is checked offline and deterministically, with no tracker call.
const (
	IntentsDir     = "docs/product/intents"
	NextFixIntents = "next: fix the intent folders above, then rerun gabo check"
)

var (
	intentNameRE = regexp.MustCompile(`^\d{4}-[a-z0-9]+(-[a-z0-9]+)*$`)
	taskIDRE     = regexp.MustCompile(`^T[0-9]+$`)
	crossRefRE   = regexp.MustCompile(`^(\d{4})/(T[0-9]+)$`)
)

// planTask is one row of a plan's task table.
type planTask struct {
	id        string
	dependsOn []string
	unblocks  []string
}

// intentFolder is one well-named intent folder and its parsed plan.
type intentFolder struct {
	name, num string
	tasks     []planTask
	ids       map[string]bool
}

func (f intentFolder) dir() string  { return path.Join(IntentsDir, f.name) }
func (f intentFolder) plan() string { return path.Join(f.dir(), "plan.md") }

// checkIntents validates the intent folders. A project without IntentsDir
// is not checked and reports nothing.
func (u *UseCase) checkIntents(r *Report) error {
	entries, err := fs.ReadDir(u.project, IntentsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", IntentsDir, err)
	}
	var fails []string
	var folders []*intentFolder
	byNum := map[string]*intentFolder{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := path.Join(IntentsDir, e.Name())
		if !intentNameRE.MatchString(e.Name()) {
			fails = append(fails, "✗ "+dir+": folder name must be NNNN-slug (4 digits, then lowercase words joined by hyphens)")
			continue
		}
		f := &intentFolder{name: e.Name(), num: e.Name()[:4], ids: map[string]bool{}}
		if first, ok := byNum[f.num]; ok {
			fails = append(fails, "✗ intent number "+f.num+" used twice: "+first.dir()+" and "+dir)
			continue
		}
		byNum[f.num] = f
		folders = append(folders, f)
	}
	for _, f := range folders {
		ff, err := u.checkIntentFiles(f)
		if err != nil {
			return err
		}
		fails = append(fails, ff...)
	}
	for _, f := range folders {
		fails = append(fails, checkPlanRefs(f, byNum)...)
	}
	fails = append(fails, planCycles(folders, byNum)...)
	if len(fails) > 0 {
		r.fail(append(fails, NextFixIntents)...)
	}
	return nil
}

// checkIntentFiles requires intent.md and, for a plan, an accepted spec
// (the human gate) and a task table with unique T<n> ids. It fills f.tasks.
func (u *UseCase) checkIntentFiles(f *intentFolder) ([]string, error) {
	var fails []string
	if _, err := u.readOptional(path.Join(f.dir(), "intent.md")); errors.Is(err, fs.ErrNotExist) {
		fails = append(fails, "✗ "+f.dir()+": no intent.md")
	} else if err != nil {
		return nil, err
	}
	plan, err := u.readOptional(f.plan())
	if errors.Is(err, fs.ErrNotExist) {
		return fails, nil
	}
	if err != nil {
		return nil, err
	}
	reason, err := u.specGate(f.dir())
	if err != nil {
		return nil, err
	}
	if reason != "" {
		fails = append(fails, "✗ "+f.plan()+": "+reason)
	}
	rows, ok := parseTable(plan, "id", "title", "spec refs", "depends_on", "unblocks")
	if !ok {
		return append(fails, "✗ "+f.plan()+": no task table (| id | title | spec refs | depends_on | unblocks | acceptance | size |)"), nil
	}
	for _, row := range rows {
		if strings.Contains(row["title"], "(removed") {
			continue
		}
		id := row["id"]
		switch {
		case !taskIDRE.MatchString(id):
			fails = append(fails, "✗ "+f.plan()+": task id "+fmt.Sprintf("%q", id)+" is not T<n>")
			continue
		case f.ids[id]:
			fails = append(fails, "✗ "+f.plan()+": task "+id+" listed twice")
			continue
		}
		f.ids[id] = true
		f.tasks = append(f.tasks, planTask{id: id, dependsOn: refList(row["depends_on"]), unblocks: refList(row["unblocks"])})
	}
	return fails, nil
}

// checkPlanRefs resolves every depends_on and unblocks reference: a local
// T<n> must be a task of the plan and mirrored by it, a <NNNN>/T<n> must
// name an existing intent and one of its plan's tasks.
func checkPlanRefs(f *intentFolder, byNum map[string]*intentFolder) []string {
	var fails []string
	byID := map[string]planTask{}
	for _, t := range f.tasks {
		byID[t.id] = t
	}
	for _, t := range f.tasks {
		for _, k := range []struct {
			key, mirror string
			refs        []string
			back        func(planTask) []string
		}{
			{"depends_on", "unblocks", t.dependsOn, func(o planTask) []string { return o.unblocks }},
			{"unblocks", "depends_on", t.unblocks, func(o planTask) []string { return o.dependsOn }},
		} {
			for _, ref := range k.refs {
				num, id, ok := splitRef(ref, f.num)
				prefix := "✗ " + f.plan() + ": " + t.id + " " + k.key + " "
				switch {
				case !ok:
					fails = append(fails, prefix+fmt.Sprintf("%q", ref)+", not T<n> or <NNNN>/T<n>")
				case num != f.num:
					target, exists := byNum[num]
					if !exists {
						fails = append(fails, prefix+ref+", but no intent "+num+" exists")
					} else if !target.ids[id] {
						fails = append(fails, prefix+ref+", but "+target.plan()+" has no task "+id)
					}
				case !f.ids[id]:
					fails = append(fails, prefix+id+", which is not a task of this plan")
				case !slices.Contains(k.back(byID[id]), t.id) && !slices.Contains(k.back(byID[id]), f.num+"/"+t.id):
					fails = append(fails, prefix+id+", but "+id+" does not list "+t.id+" in "+k.mirror)
				}
			}
		}
	}
	return fails
}

// planCycles finds the dependency cycles across every plan: an edge goes
// from a task to each task it depends on, and from each task it unblocks
// to it. Nodes are <NNNN>/T<n>; each back edge reports one cycle.
func planCycles(folders []*intentFolder, byNum map[string]*intentFolder) []string {
	edges := map[string][]string{}
	add := func(from, to string) {
		if !slices.Contains(edges[from], to) {
			edges[from] = append(edges[from], to)
		}
	}
	var nodes []string
	for _, f := range folders {
		for _, t := range f.tasks {
			node := f.num + "/" + t.id
			nodes = append(nodes, node)
			for _, ref := range t.dependsOn {
				if target, ok := resolveRef(ref, f.num, byNum); ok {
					add(node, target)
				}
			}
			for _, ref := range t.unblocks {
				if target, ok := resolveRef(ref, f.num, byNum); ok {
					add(target, node)
				}
			}
		}
	}
	slices.Sort(nodes)
	for _, n := range nodes {
		slices.Sort(edges[n])
	}
	const (
		unseen = iota
		open
		done
	)
	state := map[string]int{}
	var stack, fails []string
	var visit func(n string)
	visit = func(n string) {
		state[n] = open
		stack = append(stack, n)
		for _, m := range edges[n] {
			switch state[m] {
			case unseen:
				visit(m)
			case open:
				cycle := append(slices.Clone(stack[slices.Index(stack, m):]), m)
				fails = append(fails, "✗ dependency cycle: "+strings.Join(cycle, " -> "))
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = done
	}
	for _, n := range nodes {
		if state[n] == unseen {
			visit(n)
		}
	}
	return fails
}

// resolveRef returns the <NNNN>/T<n> node of a reference to an existing task.
func resolveRef(ref, own string, byNum map[string]*intentFolder) (string, bool) {
	num, id, ok := splitRef(ref, own)
	if !ok {
		return "", false
	}
	if f, exists := byNum[num]; !exists || !f.ids[id] {
		return "", false
	}
	return num + "/" + id, true
}

// splitRef splits T<n> (in intent own) or <NNNN>/T<n>.
func splitRef(ref, own string) (num, id string, ok bool) {
	if taskIDRE.MatchString(ref) {
		return own, ref, true
	}
	if m := crossRefRE.FindStringSubmatch(ref); m != nil {
		return m[1], m[2], true
	}
	return "", "", false
}

// refList splits a depends_on or unblocks cell; "—", "-" and "" are none.
func refList(cell string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(cell, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
		if f = strings.Trim(f, "`"); f != "" && f != "—" && f != "-" {
			out = append(out, f)
		}
	}
	return out
}

// specGate is the intent stage gate (S7) for a plan.md in dir, shared by
// the folder check and the pre-write hook mode: it returns why a plan is
// blocked there, or "" when dir's spec.md is accepted. A read error other
// than a missing spec.md is returned, with no reason.
func (u *UseCase) specGate(dir string) (string, error) {
	spec, err := u.readOptional(path.Join(dir, "spec.md"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "no spec.md; a plan starts only after a human accepts the spec", nil
	case err != nil:
		return "", err
	case !specAccepted(spec):
		return "spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan", nil
	}
	return "", nil
}

// specAccepted reports whether the spec's Acceptance table has a row with
// both "accepted by" and "date" filled (not empty, a dash or a <placeholder>).
func specAccepted(spec []byte) bool {
	rows, ok := parseTable(spec, "accepted by", "date")
	if !ok {
		return false
	}
	for _, row := range rows {
		if filled(row["accepted by"]) && filled(row["date"]) {
			return true
		}
	}
	return false
}

func filled(cell string) bool {
	return cell != "" && cell != "—" && cell != "-" && !(strings.HasPrefix(cell, "<") && strings.HasSuffix(cell, ">"))
}

// parseTable returns the rows of the first Markdown table whose header holds
// every column in want (case-insensitive), keyed by lower-cased header.
func parseTable(b []byte, want ...string) ([]map[string]string, bool) {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		header := tableCells(line)
		if header == nil {
			continue
		}
		for j := range header {
			header[j] = strings.ToLower(header[j])
		}
		if !containsAll(header, want) {
			continue
		}
		var rows []map[string]string
		for _, l := range lines[i+1:] {
			cells := tableCells(l)
			if cells == nil {
				break
			}
			if isSeparator(cells) {
				continue
			}
			row := map[string]string{}
			for j, h := range header {
				if j < len(cells) {
					row[h] = cells[j]
				}
			}
			rows = append(rows, row)
		}
		return rows, true
	}
	return nil, false
}

func tableCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, ":-") != "" {
			return false
		}
	}
	return true
}

func containsAll(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// readOptional reads a project file; a missing one matches fs.ErrNotExist.
func (u *UseCase) readOptional(name string) ([]byte, error) {
	b, err := fs.ReadFile(u.project, name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return b, err
}
