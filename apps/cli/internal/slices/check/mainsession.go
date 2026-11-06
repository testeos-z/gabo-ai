package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// The markers around the scoped main-session block (spec 0009 S3).
const (
	assistantStart = "<!-- GABO_ASSISTANT_START -->"
	assistantEnd   = "<!-- GABO_ASSISTANT_END -->"
)

// The kinds of main-session entry a runtime skill declares (spec 0009 D5).
const (
	// kindAgent is a project default agent: an agent file
	// <agent_dir>/<id>.<ext> and a key whose value is the id in the first
	// existing config file.
	kindAgent = "agent"
	// kindBlock is the scoped block in the first existing instructions file.
	kindBlock = "block"
)

// mainSessionMeta is metadata.main_session of a runtime skill (spec 0009
// D5): what the engine's main-session entry is, so Go names no engine.
type mainSessionMeta struct {
	Kind string `yaml:"kind"`
	// Config are the project config files in the engine's resolution order
	// (kind agent).
	Config []string `yaml:"config"`
	// Key is the top-level config key naming the default agent (kind agent).
	Key string `yaml:"key"`
	// Files are the instructions files in the engine's resolution order
	// (kind block).
	Files []string `yaml:"files"`
}

// normalize trims and validates the declaration.
func (m *mainSessionMeta) normalize() error {
	if m == nil {
		return nil
	}
	m.Kind, m.Key = strings.TrimSpace(m.Kind), strings.TrimSpace(m.Key)
	switch m.Kind {
	case kindAgent:
		if len(m.Config) == 0 || m.Key == "" {
			return errors.New("main_session: kind agent needs config and key")
		}
	case kindBlock:
		if len(m.Files) == 0 {
			return errors.New("main_session: kind block needs files")
		}
	default:
		return fmt.Errorf("main_session: kind %q is not %s or %s", m.Kind, kindAgent, kindBlock)
	}
	for _, list := range [][]string{m.Config, m.Files} {
		for i, p := range list {
			p = strings.TrimSpace(p)
			if p == "" || !insideRepo(p) {
				return fmt.Errorf("main_session: %q must be a repository-relative path", p)
			}
			list[i] = path.Clean(p)
		}
	}
	return nil
}

// files returns every file the entry reads: config files and instructions
// files.
func (m *mainSessionMeta) files() []string {
	if m == nil {
		return nil
	}
	return append(slices.Clone(m.Config), m.Files...)
}

// checkMainSession verifies, read-only, that the main-session entry follows
// gabo.yaml (spec 0009 S5, intent 0013 S5): every engine listed in
// runtimes.assistant has its full entry, and no full-tier runtime skill of
// the catalog copy that is not listed keeps a gabo entry. A catalog without
// a main_session entry checks nothing.
func (u *UseCase) checkMainSession(r *Report, cfg config.Config) error {
	var listed []string
	for _, rt := range assistantsOf(cfg) {
		if rt = strings.TrimSpace(rt); rt != "" && !slices.Contains(listed, rt) {
			listed = append(listed, rt)
		}
	}
	catalog, err := fs.Sub(u.project, CatalogDir)
	if err != nil {
		return err
	}
	spec, err := agentspec.Load(catalog)
	if len(listed) == 0 || err != nil || spec.MainSession == nil {
		return nil // a spec error is reported by checkAgents
	}
	set := engineconfig.Set(runtimesOf(cfg)...)
	var own []runtimeMeta
	for _, rt := range listed {
		meta, ok := u.runtimeMeta(rt)
		if !ok {
			continue // reported by checkAgents
		}
		own = append(own, meta)
		if declaresMainSession(r, rt, meta) {
			if err := u.checkAssistantEntry(r, set, listed, rt, meta, spec); err != nil {
				return err
			}
		}
	}
	if len(own) < len(listed) {
		return nil // a listed engine's own files are unknown, so none is called stale
	}
	return u.checkStaleEntries(r, set, listed, own, *spec.MainSession)
}

// declaresMainSession reports whether the runtime skill declares
// metadata.main_session; a full-tier one without it fails, a flexible-tier
// one gets one warning and no failure (TRD D28).
func declaresMainSession(r *Report, rt string, meta runtimeMeta) bool {
	if meta.MainSession != nil {
		return true
	}
	skill := "gabo-runtime-" + rt
	if fullTier(rt) {
		r.fail("✗ runtime "+rt+": "+skill+" declares no metadata.main_session", NextRestoreCatalog)
		return false
	}
	r.warn("runtime " + rt + ": " + skill + " declares no metadata.main_session; main-session entry not checked for " + rt + " (flexible tier)")
	return false
}

// pieceState is what gabo check found of one piece of an entry.
type pieceState int

const (
	pieceAbsent pieceState = iota
	piecePresent
	piecePartial
	pieceDuplicated
)

// piece is one part of the assistant's entry and the file it lives in;
// next is the step that clears it, the rebuild when empty.
type piece struct {
	file  string
	state pieceState
	next  string
}

// checkAssistantEntry fails for each piece of the assistant engine's entry
// that is not complete: the link of each main-session skill in skill_dir,
// then the agent file and key (kind agent) or exactly one complete block in
// the first existing instructions file (kind block). A piece is "missing"
// when nothing of the entry is there, else "incomplete"; two blocks are
// "duplicated".
// set is the engine set of runtimes.* and listed the engines of
// runtimes.assistant, for the gabo init hints.
func (u *UseCase) checkAssistantEntry(r *Report, set, listed []string, rt string, meta runtimeMeta, spec agentspec.Spec) error {
	m := spec.MainSession
	var pieces []piece
	if dir := path.Clean(meta.SkillDir); meta.SkillDir != "" && insideRepo(dir) {
		for _, skill := range m.Skills {
			link, state := path.Join(dir, skill), pieceAbsent
			if u.linksTo(link, path.Join(CatalogDir, skill)) {
				state = piecePresent
			}
			pieces = append(pieces, piece{file: link, state: state})
		}
	}
	ms := meta.MainSession
	switch ms.Kind {
	case kindAgent:
		if dir := path.Clean(meta.AgentDir); meta.AgentDir != "" && insideRepo(dir) {
			p, err := u.agentFilePiece(dir, m.ID, spec)
			if err != nil {
				return err
			}
			pieces = append(pieces, p)
		}
		file, b, found, err := u.firstExisting(ms.Config)
		if err != nil {
			return err
		}
		v, isSet := keyValue(b, found, ms.Key)
		switch {
		case isSet && v == any(m.ID):
			pieces = append(pieces, piece{file: file, state: piecePresent})
		case !fullTier(rt): // a flexible-tier runtime skill writes its own key (intent 0010 D4)
			pieces = append(pieces, piece{file: file, state: pieceAbsent})
		case isSet:
			// The user's own value: gabo init keeps it (intent 0010 S6).
			r.fail(fmt.Sprintf("✗ %s: %s is %s, not %s", file, ms.Key, jsonText(v), jsonText(m.ID)),
				fmt.Sprintf("next: change %s to %s in %s, or remove it and run gabo init, then rerun gabo check", ms.Key, jsonText(m.ID), file))
		default:
			next, err := u.initMergeHint(set, listed, rt, ms, m.ID, file)
			if err != nil {
				return err
			}
			pieces = append(pieces, piece{file: file, state: pieceAbsent, next: next})
		}
	case kindBlock:
		file, b, found, err := u.firstExisting(ms.Files)
		if err != nil {
			return err
		}
		state := pieceAbsent
		if found {
			state = blockState(string(b))
		}
		pieces = append(pieces, piece{file: file, state: state})
	}
	anyThere := slices.ContainsFunc(pieces, func(p piece) bool { return p.state != pieceAbsent })
	for _, p := range pieces {
		word := ""
		switch {
		case p.state == pieceDuplicated:
			word = "duplicated"
		case p.state == piecePartial, p.state == pieceAbsent && anyThere:
			word = "incomplete"
		case p.state == pieceAbsent:
			word = "missing"
		default:
			continue
		}
		next := p.next
		if next == "" {
			next = NextRebuild
		}
		r.fail("✗ "+p.file+": gabo assistant entry "+word+" for "+rt, next)
	}
	return nil
}

// initMergeHint is the step that clears a full-tier engine's missing
// main-session key (intent 0010 S6): gabo init writes it, so the hint names
// gabo init and the one file it writes, a new one when none exists; when
// gabo init would refuse a file, the hint is its reason, so following it
// never loops (D7).
func (u *UseCase) initMergeHint(set, listed []string, rt string, ms *mainSessionMeta, id, file string) (string, error) {
	run, err := u.initDryRun(set, rt, listed, ms.Config)
	if err != nil {
		return "", err
	}
	if name, reason, ok := run.refusal(file, ms.Config); ok {
		return initRefusalHint(name, reason), nil
	}
	for _, name := range ms.Config {
		if keyNames(run.after(name), ms.Key, id) {
			return fmt.Sprintf("next: run gabo init (it merges %s: %s into %s), then rerun gabo check", jsonText(ms.Key), jsonText(id), name), nil
		}
	}
	return fmt.Sprintf("next: set %s to %s in %s, then rerun gabo check", ms.Key, jsonText(id), file), nil
}

// initRun is what gabo init's engine step would do, run in memory: the
// files it read, the bytes it would write and the files it refuses, each
// with gabo init's reason.
type initRun struct {
	read, writes map[string][]byte
	refusals     map[string]string
}

// initDryRun runs gabo init's pure rule for engine on the project's files,
// in memory, as gabo init runs it: the engine's rule when it is in set,
// else the removals of the engines not in set. It writes nothing.
func (u *UseCase) initDryRun(set []string, engine string, assistants, files []string) (initRun, error) {
	in := engineconfig.Input{Files: map[string][]byte{}, Version: u.s.Version, Assistants: assistants}
	for _, name := range files {
		b, err := fs.ReadFile(u.project, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return initRun{}, fmt.Errorf("read %s: %w", name, err)
		}
		in.Files[name] = b
	}
	report, writes := engineconfig.Remove(set, in)
	if slices.Contains(set, engine) {
		report, writes = engineconfig.Integrate([]string{engine}, in)
	}
	run := initRun{read: in.Files, writes: writes, refusals: map[string]string{}}
	for _, s := range report.Sections {
		for _, l := range s.Lines {
			for _, name := range files {
				if reason, ok := strings.CutPrefix(l, engineconfig.FailurePrefix+" "+name+": "); ok {
					reason, _, _ = strings.Cut(reason, "; fix it and run gabo init again")
					run.refusals[name] = reason
				}
			}
		}
	}
	return run, nil
}

// after returns the bytes file would hold after the run, nil when absent.
func (r initRun) after(file string) []byte {
	if b, ok := r.writes[file]; ok {
		return b
	}
	return r.read[file]
}

// refusal returns the file gabo init refuses, file itself first, then the
// first of order, and gabo init's reason.
func (r initRun) refusal(file string, order []string) (string, string, bool) {
	for _, name := range append([]string{file}, order...) {
		if reason, ok := r.refusals[name]; ok {
			return name, reason, true
		}
	}
	return "", "", false
}

// initRefusalHint is the step for a file gabo init refuses to merge.
func initRefusalHint(file, reason string) string {
	return "next: fix " + file + " (" + reason + "), then run gabo init and rerun gabo check"
}

// initRemoveHint is the step that clears another full-tier engine's gabo
// main-session key (intent 0010 S6): gabo init removes it, so the hint names
// gabo init; when gabo init would refuse a file it names gabo init's
// reason, and when gabo init would leave the key (a file it does not parse
// and does not own) it names removing the key by hand, so no hint loops (D7).
func (u *UseCase) initRemoveHint(set, listed []string, rt string, ms *mainSessionMeta, id, file string) (string, error) {
	run, err := u.initDryRun(set, rt, listed, ms.Config)
	if err != nil {
		return "", err
	}
	if name, reason, ok := run.refusal(file, ms.Config); ok {
		return initRefusalHint(name, reason), nil
	}
	if !keyNames(run.after(file), ms.Key, id) {
		return fmt.Sprintf("next: run gabo init (it removes %s: %s from %s), then rerun gabo check", jsonText(ms.Key), jsonText(id), file), nil
	}
	return fmt.Sprintf("next: remove %s: %s from %s, then rerun gabo check", jsonText(ms.Key), jsonText(id), file), nil
}

// jsonText renders v as compact JSON, without HTML escaping.
func jsonText(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprint(v)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// agentFilePiece finds the main-session agent file <dir>/<id>.<ext>; a
// missing one is named with the extension of the spec agents built there.
func (u *UseCase) agentFilePiece(dir, id string, spec agentspec.Spec) (piece, error) {
	files, err := u.agentFiles(dir)
	if err != nil {
		return piece{}, err
	}
	if names := files[id]; len(names) > 0 {
		return piece{file: path.Join(dir, names[0]), state: piecePresent}, nil
	}
	ext := ".md"
	for _, a := range spec.IDs() {
		if names := files[a]; len(names) > 0 {
			ext = path.Ext(names[0])
			break
		}
	}
	return piece{file: path.Join(dir, id+ext), state: pieceAbsent}, nil
}

// firstExisting reads the first existing file of order; when none exists it
// returns the order joined with " or " as the file to name.
func (u *UseCase) firstExisting(order []string) (string, []byte, bool, error) {
	for _, name := range order {
		b, err := fs.ReadFile(u.project, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", nil, false, fmt.Errorf("read %s: %w", name, err)
		}
		return name, b, true, nil
	}
	return strings.Join(order, " or "), nil, false, nil
}

// keyNames reports whether the JSON (or JSONC) config b sets the top-level
// key to exactly id.
func keyNames(b []byte, key, id string) bool {
	v, set := keyValue(b, true, key)
	return set && v == any(id)
}

// keyValue returns the top-level key of the JSON (or JSONC) config b and
// whether it is set; a missing or unparsable file sets nothing.
func keyValue(b []byte, found bool, key string) (any, bool) {
	if !found {
		return nil, false
	}
	doc, err := parseJSONC(b)
	if err != nil {
		return nil, false
	}
	v, ok := doc[key]
	return v, ok
}

// blockState classifies the scoped blocks in an instructions file.
func blockState(doc string) pieceState {
	starts, ends := strings.Count(doc, assistantStart), strings.Count(doc, assistantEnd)
	switch {
	case starts == 0 && ends == 0:
		return pieceAbsent
	case starts == 1 && ends == 1 && strings.Index(doc, assistantStart) < strings.Index(doc, assistantEnd):
		return piecePresent
	case starts > 1 && ends > 1:
		return pieceDuplicated
	default:
		return piecePartial
	}
}

// linksTo reports whether link is a symlink whose relative target resolves
// to want.
func (u *UseCase) linksTo(link, want string) bool {
	info, err := fs.Lstat(u.project, link)
	if err != nil || info.Mode()&fs.ModeSymlink == 0 {
		return false
	}
	target, err := fs.ReadLink(u.project, link)
	return err == nil && !path.IsAbs(target) && path.Join(path.Dir(link), target) == want
}

// checkStaleEntries fails for every gabo main-session entry of a full-tier
// runtime skill in the catalog copy that runtimes.assistant does not list, read from its
// metadata.main_session and recognized only by its gabo identity: the link
// of a main-session skill into the catalog, an agent file <id> carrying a
// gabo-version line, a key whose value is exactly <id>, or the block
// markers. A link in a listed engine's skill_dir, an agent file in its
// agent_dir and a file its entry uses are that engine's, never stale
// (Codex and Pi share AGENTS.md). The stale line names the listed engines
// joined with ", " (intent 0013 S5). Each path is reported once, for the first
// runtime skill in catalog order that declares it.
func (u *UseCase) checkStaleEntries(r *Report, set, listed []string, own []runtimeMeta, m agentspec.MainSessionEntry) error {
	entries, err := fs.ReadDir(u.project, CatalogDir)
	if err != nil {
		return nil // reported by checkAgents
	}
	var ownFiles, ownSkillDirs, ownAgentDirs []string
	for _, meta := range own {
		ownFiles = append(ownFiles, meta.MainSession.files()...)
		ownSkillDirs = append(ownSkillDirs, path.Clean(meta.SkillDir))
		ownAgentDirs = append(ownAgentDirs, path.Clean(meta.AgentDir))
	}
	assistant := strings.Join(listed, ", ")
	var reported []string
	stale := func(file, rt, next string) {
		if slices.Contains(reported, file) {
			return
		}
		reported = append(reported, file)
		r.fail("✗ "+file+": gabo assistant entry for "+rt+", but runtimes.assistant is "+assistant, next)
	}
	for _, e := range entries {
		rt, ok := strings.CutPrefix(e.Name(), "gabo-runtime-")
		if !ok || slices.Contains(listed, rt) || !fullTier(rt) {
			continue
		}
		b, err := fs.ReadFile(u.project, path.Join(CatalogDir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		meta, err := parseRuntimeMeta(b)
		if err != nil || meta.MainSession == nil {
			continue
		}
		if dir := path.Clean(meta.SkillDir); meta.SkillDir != "" && insideRepo(dir) && !slices.Contains(ownSkillDirs, dir) {
			for _, skill := range m.Skills {
				if link := path.Join(dir, skill); u.linksTo(link, path.Join(CatalogDir, skill)) {
					stale(link, rt, NextRebuild)
				}
			}
		}
		ms := meta.MainSession
		if dir := path.Clean(meta.AgentDir); ms.Kind == kindAgent && meta.AgentDir != "" && insideRepo(dir) && !slices.Contains(ownAgentDirs, dir) {
			files, err := u.agentFiles(dir)
			if err != nil {
				return err
			}
			for _, name := range files[m.ID] {
				file := path.Join(dir, name)
				b, err := fs.ReadFile(u.project, file)
				if err != nil {
					return fmt.Errorf("read %s: %w", file, err)
				}
				if versionLine(b) != "none" {
					stale(file, rt, NextRebuild)
				}
			}
		}
		for _, file := range ms.files() {
			if slices.Contains(ownFiles, file) {
				continue
			}
			b, err := fs.ReadFile(u.project, file)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("read %s: %w", file, err)
			}
			switch {
			case ms.Kind == kindAgent && slices.Contains(ms.Config, file) && keyNames(b, ms.Key, m.ID):
				// Only full-tier engines reach here: gabo init removes the key (intent 0010 S6).
				next, err := u.initRemoveHint(set, listed, rt, ms, m.ID, file)
				if err != nil {
					return err
				}
				stale(file, rt, next)
			case ms.Kind == kindBlock && blockState(string(b)) != pieceAbsent:
				stale(file, rt, NextRebuild)
			}
		}
	}
	return nil
}
