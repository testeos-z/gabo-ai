// Package agentspec loads and validates the neutral agent spec (TRD 8.1,
// D24): the engine-independent list of gabo agents that runtime skills build
// into agent files and `gabo check` verifies.
//
// The loader reads from a catalog root: the embedded catalog
// (skills.Catalog()) or the project's .gabo/skills/ copy.
package agentspec

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Dir is the catalog skill that holds the spec and the agent templates.
	Dir = "gabo-agent-templates"
	// File is the spec path relative to the catalog root.
	File = Dir + "/agents.yaml"
	// MainSession is the launched_by value of an agent the main session (the
	// assistant) launches; it is not an agent file itself.
	MainSession = "main-session"
)

// Agent is one gabo agent that runs from an agent file.
type Agent struct {
	ID   string `yaml:"id"`
	Role string `yaml:"role"`
	// Skills are catalog skill names the agent loads.
	Skills []string `yaml:"skills"`
	// Tools are engine-neutral capabilities, for example tracker, git, memory.
	Tools []string `yaml:"tools"`
	// LaunchedBy is another agent id or MainSession.
	LaunchedBy string `yaml:"launched_by"`
	// Template is the body template path, relative to Dir.
	Template string `yaml:"template"`
}

// MainSessionEntry is the owner's main-session entry (spec 0009 D4): the
// assistant agent template and the scoped block a runtime skill builds only
// for runtimes.assistant. It is no spec agent: IDs and Agents never hold it.
type MainSessionEntry struct {
	ID   string `yaml:"id"`
	Role string `yaml:"role"`
	// Skills are catalog skill names the main session loads.
	Skills []string `yaml:"skills"`
	// Template is the agent body template path, relative to Dir.
	Template string `yaml:"template"`
	// Block is the scoped instructions-file block path, relative to Dir.
	Block string `yaml:"block"`
}

// Spec is the validated content of agents.yaml.
type Spec struct {
	Agents []Agent `yaml:"agents"`
	// MainSession is nil when agents.yaml has no main_session entry.
	MainSession *MainSessionEntry `yaml:"main_session"`
}

// IDs lists the agent ids in spec order; the main-session entry is not one.
func (s Spec) IDs() []string {
	ids := make([]string, len(s.Agents))
	for i, a := range s.Agents {
		ids[i] = a.ID
	}
	return ids
}

// Load reads File from the catalog root and validates it: required fields,
// unique ids, every launched_by and template resolves, every skill exists in
// the catalog (a directory with a SKILL.md) and launched_by has no cycle; an
// optional main_session entry needs an id that is no spec agent, a role, and
// a template and block that resolve. It reports every problem found.
func Load(root fs.FS) (Spec, error) {
	b, err := fs.ReadFile(root, File)
	if err != nil {
		return Spec{}, fmt.Errorf("read %s: %w", File, err)
	}
	var spec Spec
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("parse %s: %w", File, err)
	}
	if err := validate(root, spec); err != nil {
		return Spec{}, fmt.Errorf("%s: %w", File, err)
	}
	return spec, nil
}

func validate(root fs.FS, spec Spec) error {
	if len(spec.Agents) == 0 {
		return errors.New("no agents")
	}
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	byID := map[string]Agent{}
	for i, a := range spec.Agents {
		if a.ID == "" {
			fail("agent %d: missing id", i+1)
			continue
		}
		if _, dup := byID[a.ID]; dup {
			fail("agent %s: duplicate id", a.ID)
			continue
		}
		byID[a.ID] = a
	}
	for _, a := range spec.Agents {
		if a.ID == "" {
			continue
		}
		if a.Role == "" {
			fail("agent %s: missing role", a.ID)
		}
		if len(a.Tools) == 0 {
			fail("agent %s: missing tools", a.ID)
		}
		switch _, known := byID[a.LaunchedBy]; {
		case a.LaunchedBy == "":
			fail("agent %s: missing launched_by", a.ID)
		case a.LaunchedBy != MainSession && !known:
			fail("agent %s: launched_by %q is not a spec agent or %s", a.ID, a.LaunchedBy, MainSession)
		}
		checkFile(root, fail, "agent "+a.ID, "template", a.Template)
		checkSkills(root, fail, "agent "+a.ID, a.Skills)
	}
	if m := spec.MainSession; m != nil {
		validateMainSession(root, fail, *m, byID)
	}
	if err := checkCycles(spec, byID); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateMainSession(root fs.FS, fail func(string, ...any), m MainSessionEntry, byID map[string]Agent) {
	const who = "main_session"
	if m.ID == "" {
		fail("%s: missing id", who)
	} else if _, clash := byID[m.ID]; clash {
		fail("%s: id %s is a spec agent", who, m.ID)
	}
	if m.Role == "" {
		fail("%s: missing role", who)
	}
	checkSkills(root, fail, who, m.Skills)
	checkFile(root, fail, who, "template", m.Template)
	checkFile(root, fail, who, "block", m.Block)
}

// checkFile requires field p to name a file inside Dir.
func checkFile(root fs.FS, fail func(string, ...any), who, field, p string) {
	switch {
	case p == "":
		fail("%s: missing %s", who, field)
	case !fs.ValidPath(p) || p == ".":
		fail("%s: %s %q must be a relative path inside %s", who, field, p, Dir)
	case !isFile(root, path.Join(Dir, p)):
		fail("%s: %s %q not found", who, field, p)
	}
}

// checkSkills requires every name to be a catalog skill folder.
func checkSkills(root fs.FS, fail func(string, ...any), who string, names []string) {
	for _, s := range names {
		if s == "" || strings.Contains(s, "/") || !isFile(root, s+"/SKILL.md") {
			fail("%s: skill %q not in the catalog", who, s)
		}
	}
}

// checkCycles follows each launched_by chain and reports the first cycle; with
// one launcher per agent a chain revisits an agent only through a cycle.
func checkCycles(spec Spec, byID map[string]Agent) error {
	for _, a := range spec.Agents {
		if a.ID == "" {
			continue
		}
		var chain []string
		at := map[string]int{}
		for id := a.ID; ; {
			if i, seen := at[id]; seen {
				loop := append(chain[i:], id)
				return fmt.Errorf("launched_by cycle: %s", strings.Join(loop, " -> "))
			}
			next, ok := byID[id]
			if !ok {
				break
			}
			at[id] = len(chain)
			chain = append(chain, id)
			id = next.LaunchedBy
		}
	}
	return nil
}

func isFile(root fs.FS, p string) bool {
	info, err := fs.Stat(root, p)
	return err == nil && info.Mode().IsRegular()
}
