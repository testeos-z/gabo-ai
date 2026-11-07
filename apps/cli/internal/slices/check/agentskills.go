package check

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// templateSlotRE finds the template slots that name a fixed-name link:
// {{methodology}} and {{adapter:<slot>}}.
var templateSlotRE = regexp.MustCompile(`\{\{(methodology|adapter:([A-Za-z0-9_-]+))\}\}`)

// templateLink is a fixed-name link an agent template names: the slot and
// the link name gabo-<slot> ("_" written "-").
type templateLink struct{ slot, name string }

// templateLinks lists the fixed-name links a template names, in order of
// first appearance, each once.
func templateLinks(b []byte) []templateLink {
	var out []templateLink
	for _, m := range templateSlotRE.FindAllSubmatch(b, -1) {
		slot := string(m[2])
		if slot == "" {
			slot = config.SlotMethodology
		}
		l := templateLink{slot: slot, name: "gabo-" + strings.ReplaceAll(slot, "_", "-")}
		if !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return out
}

// expectedSkill is one skill an agent file must list once: name, or alt (the
// frontmatter name of a fixed-name link's adapter) in its place.
type expectedSkill struct{ name, alt string }

// compareSkills matches the skills an agent file lists against want as a set
// with no duplicates: missing holds each wanted name listed under neither
// name; duplicate each wanted entry listed more than once (by its name or
// alt), once, as its first extra copy writes it; extra each listed skill that
// matches no wanted entry, every copy (spec 0014 S2). All three keep list
// order.
func compareSkills(want []expectedSkill, got []string) (missing, duplicate, extra []string) {
	matched := make([]bool, len(want))
	reported := make([]bool, len(want))
	for _, s := range got {
		first, free := -1, -1
		for i, w := range want {
			if w.name != s && (w.alt == "" || w.alt != s) {
				continue
			}
			if first < 0 {
				first = i
			}
			if !matched[i] {
				free = i
				break
			}
		}
		switch {
		case free >= 0:
			matched[free] = true
		case first < 0:
			extra = append(extra, s)
		case !reported[first]:
			reported[first] = true
			duplicate = append(duplicate, s)
		}
	}
	for i, w := range want {
		if !matched[i] {
			missing = append(missing, w.name)
		}
	}
	return missing, duplicate, extra
}

// checkAgentSkills requires the agent file to list exactly the spec skills of
// agent id plus the fixed-name links its template in the project catalog copy
// names, each once, in the place the runtime skill declares as
// metadata.agent_skills (intent 0008 S1, D2). Without that declaration it
// checks nothing (checkAgentMeta reports it).
func (u *UseCase) checkAgentSkills(r *Report, cfg config.Config, rt string, spec agentspec.Spec, id, file string) error {
	meta, ok := u.runtimeMeta(rt)
	i := slices.IndexFunc(spec.Agents, func(a agentspec.Agent) bool { return a.ID == id })
	if !ok || !meta.AgentSkills.declared() || i < 0 {
		return nil
	}
	a := spec.Agents[i]
	tmplPath := path.Join(CatalogDir, agentspec.Dir, a.Template)
	tmpl, err := fs.ReadFile(u.project, tmplPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", tmplPath, err)
	}
	b, err := fs.ReadFile(u.project, file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}
	var want []expectedSkill
	add := func(w expectedSkill) {
		if !slices.ContainsFunc(want, func(e expectedSkill) bool { return e.name == w.name }) {
			want = append(want, w)
		}
	}
	for _, s := range a.Skills {
		add(expectedSkill{name: s})
	}
	for _, l := range templateLinks(tmpl) {
		add(expectedSkill{name: l.name, alt: u.adapterName(cfg, l.slot, l.name)})
	}
	missing, duplicate, extra := compareSkills(want, meta.AgentSkills.skills(b))
	for _, s := range missing {
		r.fail(fmt.Sprintf("✗ agent %s (%s): skill %s missing", id, rt, s), "why: "+file, NextRebuild)
	}
	for _, s := range duplicate {
		r.fail(fmt.Sprintf("✗ agent %s (%s): skill %s listed twice", id, rt, s), "why: "+file, NextRebuild)
	}
	for _, s := range extra {
		r.fail(fmt.Sprintf("✗ agent %s (%s): skill %s not in the agent spec", id, rt, s), "why: "+file, NextRebuild)
	}
	return nil
}

// adapterName returns the frontmatter name of the slot's adapter skill when
// it differs from the link name, else "".
func (u *UseCase) adapterName(cfg config.Config, slot, link string) string {
	s, ok := cfg.Slots[slot]
	if !ok {
		return ""
	}
	file := u.adapterPath(slot, s.Adapter)
	if !insideRepo(file) {
		return ""
	}
	b, err := fs.ReadFile(u.project, file)
	if err != nil {
		return "" // reported by checkSlots
	}
	name, _ := frontmatterValue(b, "name").(string)
	if name = strings.TrimSpace(name); name == link {
		return ""
	}
	return name
}
