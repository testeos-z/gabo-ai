package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
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
// gabo.yaml (spec 0009 S5): the engine in runtimes.assistant has its full
// entry, and no other full-tier runtime skill of the catalog copy keeps a
// gabo entry. A catalog without a main_session entry checks nothing.
func (u *UseCase) checkMainSession(r *Report, cfg config.Config) error {
	assistant := string(cfg.Runtimes.Assistant)
	catalog, err := fs.Sub(u.project, CatalogDir)
	if err != nil {
		return err
	}
	spec, err := agentspec.Load(catalog)
	if assistant == "" || err != nil || spec.MainSession == nil {
		return nil // a spec error is reported by checkAgents
	}
	own, ok := u.runtimeMeta(assistant)
	if !ok {
		return nil // reported by checkAgents
	}
	if declaresMainSession(r, assistant, own) {
		if err := u.checkAssistantEntry(r, assistant, own, spec); err != nil {
			return err
		}
	}
	return u.checkStaleEntries(r, assistant, own, *spec.MainSession)
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

// piece is one part of the assistant's entry and the file it lives in.
type piece struct {
	file  string
	state pieceState
}

// checkAssistantEntry fails for each piece of the assistant engine's entry
// that is not complete: the link of each main-session skill in skill_dir,
// then the agent file and key (kind agent) or exactly one complete block in
// the first existing instructions file (kind block). A piece is "missing"
// when nothing of the entry is there, else "incomplete"; two blocks are
// "duplicated".
func (u *UseCase) checkAssistantEntry(r *Report, rt string, meta runtimeMeta, spec agentspec.Spec) error {
	m := spec.MainSession
	var pieces []piece
	if dir := path.Clean(meta.SkillDir); meta.SkillDir != "" && insideRepo(dir) {
		for _, skill := range m.Skills {
			link, state := path.Join(dir, skill), pieceAbsent
			if u.linksTo(link, path.Join(CatalogDir, skill)) {
				state = piecePresent
			}
			pieces = append(pieces, piece{link, state})
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
		state := pieceAbsent
		if found && keyNames(b, ms.Key, m.ID) {
			state = piecePresent
		}
		pieces = append(pieces, piece{file, state})
	case kindBlock:
		file, b, found, err := u.firstExisting(ms.Files)
		if err != nil {
			return err
		}
		state := pieceAbsent
		if found {
			state = blockState(string(b))
		}
		pieces = append(pieces, piece{file, state})
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
		r.fail("✗ "+p.file+": gabo assistant entry "+word+" for "+rt, NextRebuild)
	}
	return nil
}

// agentFilePiece finds the main-session agent file <dir>/<id>.<ext>; a
// missing one is named with the extension of the spec agents built there.
func (u *UseCase) agentFilePiece(dir, id string, spec agentspec.Spec) (piece, error) {
	files, err := u.agentFiles(dir)
	if err != nil {
		return piece{}, err
	}
	if names := files[id]; len(names) > 0 {
		return piece{path.Join(dir, names[0]), piecePresent}, nil
	}
	ext := ".md"
	for _, a := range spec.IDs() {
		if names := files[a]; len(names) > 0 {
			ext = path.Ext(names[0])
			break
		}
	}
	return piece{path.Join(dir, id+ext), pieceAbsent}, nil
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
	doc, err := parseJSONC(b)
	if err != nil {
		return false
	}
	v, ok := doc[key].(string)
	return ok && v == id
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

// checkStaleEntries fails for every gabo main-session entry of another
// full-tier runtime skill in the catalog copy, read from its
// metadata.main_session and recognized only by its gabo identity: the link
// of a main-session skill into the catalog, an agent file <id> carrying a
// gabo-version line, a key whose value is exactly <id>, or the block
// markers. A link in the assistant's own skill_dir, an agent file in its
// agent_dir and a file its own entry uses are the assistant's, never stale
// (Codex and Pi share AGENTS.md). Each path is reported once, for the first
// runtime skill in catalog order that declares it.
func (u *UseCase) checkStaleEntries(r *Report, assistant string, own runtimeMeta, m agentspec.MainSessionEntry) error {
	entries, err := fs.ReadDir(u.project, CatalogDir)
	if err != nil {
		return nil // reported by checkAgents
	}
	ownFiles := own.MainSession.files()
	ownSkillDir, ownAgentDir := path.Clean(own.SkillDir), path.Clean(own.AgentDir)
	var reported []string
	stale := func(file, rt string) {
		if slices.Contains(reported, file) {
			return
		}
		reported = append(reported, file)
		r.fail("✗ "+file+": gabo assistant entry for "+rt+", but runtimes.assistant is "+assistant, NextRebuild)
	}
	for _, e := range entries {
		rt, ok := strings.CutPrefix(e.Name(), "gabo-runtime-")
		if !ok || rt == assistant || !fullTier(rt) {
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
		if dir := path.Clean(meta.SkillDir); meta.SkillDir != "" && insideRepo(dir) && dir != ownSkillDir {
			for _, skill := range m.Skills {
				if link := path.Join(dir, skill); u.linksTo(link, path.Join(CatalogDir, skill)) {
					stale(link, rt)
				}
			}
		}
		ms := meta.MainSession
		if dir := path.Clean(meta.AgentDir); ms.Kind == kindAgent && meta.AgentDir != "" && insideRepo(dir) && dir != ownAgentDir {
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
					stale(file, rt)
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
			if ms.Kind == kindAgent && slices.Contains(ms.Config, file) && keyNames(b, ms.Key, m.ID) ||
				ms.Kind == kindBlock && blockState(string(b)) != pieceAbsent {
				stale(file, rt)
			}
		}
	}
	return nil
}
