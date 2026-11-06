package check

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// NextLink follows a missing or wrong skill or reviewer link.
const NextLink = "next: ask your agent to make the links with the runtime skill, then rerun gabo check"

// reviewerScript is the script the gabo-reviewer agent runs through the link.
const reviewerScript = "scripts/reviewer.sh"

// runtimeMeta is the metadata a runtime skill declares in its frontmatter.
type runtimeMeta struct {
	// AgentDir is where the runtime's agent files live (repository-relative).
	AgentDir string `yaml:"agent_dir"`
	// SkillDir is where each catalog skill and fixed-name slot link lives
	// (repository-relative); empty means the runtime makes no skill links.
	SkillDir string `yaml:"skill_dir"`
	// FormerSkillDir is where the runtime made its skill links before the
	// move to .agents/skills (repository-relative); empty means none
	// (intent 0008 S4).
	FormerSkillDir string `yaml:"former_skill_dir"`
	// ReviewerLink is the user-level link to the reviewer skill, with
	// ${NAME}, ${NAME:-default} and $NAME expanded; empty means none.
	ReviewerLink string `yaml:"reviewer_link"`
	// AgentSkills is where an agent file lists its skills (intent 0008 D1).
	AgentSkills agentPlace `yaml:"-"`
	// AgentName is where an agent file declares its name; not declared when
	// the engine names an agent by its file name (intent 0008 D1).
	AgentName agentPlace `yaml:"-"`
}

// parseRuntimeMeta reads metadata from a SKILL.md frontmatter.
func parseRuntimeMeta(b []byte) (runtimeMeta, error) {
	rest, ok := bytes.CutPrefix(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), []byte("---\n"))
	if !ok {
		return runtimeMeta{}, errors.New("missing --- frontmatter")
	}
	front, _, ok := bytes.Cut(rest, []byte("\n---"))
	if !ok {
		return runtimeMeta{}, errors.New("unterminated frontmatter")
	}
	var fm struct {
		Metadata struct {
			runtimeMeta `yaml:",inline"`
			AgentSkills string `yaml:"agent_skills"`
			AgentName   string `yaml:"agent_name"`
		} `yaml:"metadata"`
	}
	err := yaml.Unmarshal(front, &fm)
	if err != nil {
		return runtimeMeta{}, err
	}
	m := fm.Metadata.runtimeMeta
	m.AgentDir = strings.TrimSpace(m.AgentDir)
	m.SkillDir = strings.TrimSpace(m.SkillDir)
	m.FormerSkillDir = strings.TrimSpace(m.FormerSkillDir)
	m.ReviewerLink = strings.TrimSpace(m.ReviewerLink)
	if m.AgentSkills, err = parseAgentPlace("agent_skills", fm.Metadata.AgentSkills); err != nil {
		return runtimeMeta{}, err
	}
	if m.AgentName, err = parseAgentPlace("agent_name", fm.Metadata.AgentName); err != nil {
		return runtimeMeta{}, err
	}
	return m, nil
}

// insideRepo reports whether dir is a repository-relative path that stays
// inside the repository.
func insideRepo(dir string) bool {
	return !path.IsAbs(dir) && !slices.Contains(strings.Split(dir, "/"), "..") && fs.ValidPath(path.Clean(dir))
}

// checkLinks requires, for each runtime whose skill declares a skill_dir, a
// link <skill_dir>/<name> to the catalog copy for every skill a spec agent
// lists, and a fixed-name link <skill_dir>/gabo-<slot> to every slot adapter
// that is a skill folder. Then it checks the reviewer runtime's user-level
// reviewer link.
func (u *UseCase) checkLinks(r *Report, cfg config.Config) error {
	catalog, err := fs.Sub(u.project, CatalogDir)
	if err != nil {
		return err
	}
	spec, err := agentspec.Load(catalog)
	if err != nil {
		return nil // reported by checkAgents
	}
	var checked []string
	for _, rt := range runtimesOf(cfg) {
		meta, ok := u.runtimeMeta(rt)
		if ok && meta.AgentDir != "" {
			checkAgentMeta(r, rt, meta)
		}
		if !ok || meta.SkillDir == "" {
			continue
		}
		if !insideRepo(meta.SkillDir) {
			r.fail("✗ runtime "+rt+": skill_dir must stay inside the repository",
				"next: set metadata.skill_dir in gabo-runtime-"+rt+" to a repository-relative path, then rerun gabo check")
			continue
		}
		dir := path.Clean(meta.SkillDir)
		if slices.Contains(checked, dir) {
			continue
		}
		checked = append(checked, dir)
		var seen []string
		for _, a := range spec.Agents {
			for _, name := range a.Skills {
				if slices.Contains(seen, name) {
					continue
				}
				seen = append(seen, name)
				want := path.Join(CatalogDir, name)
				u.checkLink(r, path.Join(dir, name), want,
					"why: agent "+a.ID+" lists "+name+"; link it to "+relTo(dir, want)+" (runtime "+rt+")")
			}
		}
		for _, slot := range slotNames(cfg) {
			folder, ok := u.adapterFolder(slot, cfg.Slots[slot].Adapter)
			if !ok {
				continue
			}
			link := path.Join(dir, "gabo-"+strings.ReplaceAll(slot, "_", "-"))
			if link == folder {
				continue
			}
			u.checkLink(r, link, folder,
				"why: slots."+slot+".adapter is "+strings.TrimSpace(cfg.Slots[slot].Adapter)+"; link it to "+relTo(dir, folder)+" (runtime "+rt+")")
		}
	}
	if rt := string(cfg.Runtimes.Reviewer); rt != "" {
		if meta, ok := u.runtimeMeta(rt); ok && meta.ReviewerLink != "" {
			u.checkReviewerLink(r, rt, meta.ReviewerLink)
		}
	}
	return nil
}

// checkLink requires link to be a symlink in the project whose target
// resolves to the folder want, and want to hold a SKILL.md.
func (u *UseCase) checkLink(r *Report, link, want, why string) {
	wantRel := relTo(path.Dir(link), want)
	info, err := fs.Lstat(u.project, link)
	if err != nil {
		r.fail("✗ skill link "+link+" missing", why, NextLink)
		return
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		r.fail("✗ skill link "+link+" is not a link to "+wantRel, why, NextLink)
		return
	}
	target, err := fs.ReadLink(u.project, link)
	if err != nil {
		r.fail("✗ skill link "+link+" cannot be read: "+err.Error(), why, NextLink)
		return
	}
	if path.IsAbs(target) || path.Join(path.Dir(link), target) != want {
		r.fail("✗ skill link "+link+" points to "+target+", not "+wantRel, why, NextLink)
		return
	}
	if _, err := fs.Stat(u.project, path.Join(want, "SKILL.md")); err != nil {
		r.fail("✗ skill link "+link+" is broken: "+want+" has no SKILL.md", why,
			"next: run gabo init to restore the catalog or fix the adapter folder, then rerun gabo check")
	}
}

// checkReviewerLink requires the user-level reviewer link to be a symlink to
// the reviewer skill dir that gabo init installs, holding the script.
func (u *UseCase) checkReviewerLink(r *Report, rt, declared string) {
	if u.s.CI {
		r.warn("reviewer link not checked in CI: it is user-level")
		return
	}
	if u.s.ReviewerDir == "" {
		r.fail("✗ reviewer skill dir unknown: set XDG_DATA_HOME or HOME to an absolute path")
		return
	}
	link, ok := expand(declared, u.getenv)
	if !ok || !filepath.IsAbs(link) {
		r.fail("✗ reviewer link "+declared+" is not an absolute path here",
			"next: set HOME to an absolute path, then rerun gabo check")
		return
	}
	link = filepath.Clean(link)
	dir := filepath.Clean(u.s.ReviewerDir)
	why := "why: the gabo-reviewer agent runs " + filepath.Join(link, reviewerScript) + "; link it to " + dir + " (runtime " + rt + ")"
	mode, err := u.user.Lstat(link)
	if err != nil {
		r.fail("✗ reviewer link "+link+" missing", why, NextLink)
		return
	}
	if mode&fs.ModeSymlink == 0 {
		r.fail("✗ reviewer link "+link+" is not a link to "+dir, why, NextLink)
		return
	}
	target, err := u.user.ReadLink(link)
	if err != nil {
		r.fail("✗ reviewer link "+link+" cannot be read: "+err.Error(), why, NextLink)
		return
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(link), resolved)
	}
	if filepath.Clean(resolved) != dir {
		r.fail("✗ reviewer link "+link+" points to "+target+", not "+dir, why, NextLink)
		return
	}
	if _, err := u.user.Stat(filepath.Join(link, reviewerScript)); err != nil {
		r.fail("✗ reviewer link "+link+" is broken: "+filepath.Join(dir, reviewerScript)+" missing",
			"next: run gabo init to install the reviewer skill, then rerun gabo check")
	}
}

func (u *UseCase) getenv(name string) string {
	if u.s.Getenv == nil {
		return ""
	}
	return u.s.Getenv(name)
}

// runtimeMeta reads the runtime skill's metadata; a missing or unreadable
// skill is reported by checkAgents.
func (u *UseCase) runtimeMeta(rt string) (runtimeMeta, bool) {
	b, err := u.runtimeSkill(rt)
	if err != nil {
		return runtimeMeta{}, false
	}
	meta, err := parseRuntimeMeta(b)
	return meta, err == nil
}

// runtimeSkill reads gabo-runtime-<rt>/SKILL.md from the catalog copy, else
// from the project's own runtimes.
func (u *UseCase) runtimeSkill(rt string) ([]byte, error) {
	skill := "gabo-runtime-" + rt
	var b []byte
	var err error
	for _, dir := range []string{CatalogDir, RuntimesDir} {
		b, err = fs.ReadFile(u.project, path.Join(dir, skill, "SKILL.md"))
		if !errors.Is(err, fs.ErrNotExist) {
			break
		}
	}
	return b, err
}

// adapterFolder returns the project folder of a slot adapter that is a skill
// (<folder>/SKILL.md); other adapters need no fixed-name link.
func (u *UseCase) adapterFolder(slot, adapter string) (string, bool) {
	file := u.adapterPath(slot, adapter)
	if path.Base(file) != "SKILL.md" || !insideRepo(file) {
		return "", false
	}
	return path.Dir(file), true
}

func runtimesOf(cfg config.Config) []string {
	var out []string
	for _, rt := range []config.Runtime{cfg.Runtimes.Coder, cfg.Runtimes.Reviewer, cfg.Runtimes.Assistant} {
		if rt != "" && !slices.Contains(out, string(rt)) {
			out = append(out, string(rt))
		}
	}
	return out
}

func slotNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Slots))
	for name := range cfg.Slots {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// relTo returns the relative slash path from dir to target, both clean and
// repository-relative.
func relTo(dir, target string) string {
	d := strings.Split(dir, "/")
	t := strings.Split(target, "/")
	i := 0
	for i < len(d) && i < len(t) && d[i] == t[i] {
		i++
	}
	parts := make([]string, 0, len(d)-i+len(t)-i)
	for range d[i:] {
		parts = append(parts, "..")
	}
	parts = append(parts, t[i:]...)
	if len(parts) == 0 {
		return "."
	}
	return strings.Join(parts, "/")
}

// expand replaces ${NAME:-default}, ${NAME} and $NAME with getenv; an unset
// or empty NAME takes the (expanded) default. It reports false when a NAME
// without a default is unset or empty.
func expand(s string, getenv func(string) string) (string, bool) {
	ok := true
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		if s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				b.WriteString(s[i:])
				break
			}
			body := s[i+2 : i+2+end]
			name, def, hasDef := strings.Cut(body, ":-")
			v := getenv(name)
			switch {
			case v == "" && hasDef:
				var defOK bool
				v, defOK = expand(def, getenv)
				ok = ok && defOK
			case v == "":
				ok = false
			}
			b.WriteString(v)
			i += 2 + end
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] == '_' || s[j] >= 'A' && s[j] <= 'Z' || s[j] >= 'a' && s[j] <= 'z' || j > i+1 && s[j] >= '0' && s[j] <= '9') {
			j++
		}
		if j == i+1 {
			b.WriteByte('$')
			continue
		}
		v := getenv(s[i+1 : j])
		ok = ok && v != ""
		b.WriteString(v)
		i = j - 1
	}
	return b.String(), ok
}
