package check

import (
	"fmt"
	"io/fs"
)

// checkAgentName fails an agent file whose declared name differs from its
// spec id (intent 0008 S6) or that declares no name (spec 0014 S1: absent,
// blank, non-string, or in a frontmatter or TOML that does not parse), where
// the runtime skill declares metadata.agent_name. A flexible-tier engine
// without agent_skills is skipped (S2).
func (u *UseCase) checkAgentName(r *Report, id, rt, file string) {
	meta, ok := u.runtimeMeta(rt)
	if !ok || !meta.AgentName.declared() || (!meta.AgentSkills.declared() && !fullTier(rt)) {
		return
	}
	b, err := fs.ReadFile(u.project, file)
	if err != nil {
		return // read errors are reported by checkAgentVersion
	}
	switch name := meta.AgentName.name(b); {
	case name == "":
		r.fail(fmt.Sprintf("✗ agent %s (%s): declares no name", id, rt), "why: "+file, NextRebuild)
	case name != id:
		r.fail(fmt.Sprintf("✗ agent %s (%s): declares name %s", id, rt, name), "why: "+file, NextRebuild)
	}
}
