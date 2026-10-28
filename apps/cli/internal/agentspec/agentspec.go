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

// Spec is the validated content of agents.yaml.
type Spec struct {
	Agents []Agent `yaml:"agents"`
}

// IDs lists the agent ids in spec order.
func (s Spec) IDs() []string {
	ids := make([]string, len(s.Agents))
	for i, a := range s.Agents {
		ids[i] = a.ID
	}
	return ids
}

// Load reads File from the catalog root and validates it: required fields,
// unique ids, every launched_by and template resolves, every skill exists in
// the catalog (a directory with a SKILL.md) and launched_by has no cycle. It
// reports every problem found.
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
		switch {
		case a.Template == "":
			fail("agent %s: missing template", a.ID)
		case !fs.ValidPath(a.Template) || a.Template == ".":
			fail("agent %s: template %q must be a relative path inside %s", a.ID, a.Template, Dir)
		case !isFile(root, path.Join(Dir, a.Template)):
			fail("agent %s: template %q not found", a.ID, a.Template)
		}
		for _, s := range a.Skills {
			if s == "" || strings.Contains(s, "/") || !isFile(root, s+"/SKILL.md") {
				fail("agent %s: skill %q not in the catalog", a.ID, s)
			}
		}
	}
	if err := checkCycles(spec, byID); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
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
