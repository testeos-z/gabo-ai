package check

import (
	"fmt"
	"io/fs"
)

// checkAgentName fails an agent file whose declared name differs from its
// spec id (intent 0008 S6), where the runtime skill declares
// metadata.agent_name. A flexible-tier engine without agent_skills is
// skipped (S2), and a file that declares no name is not a declared name.
func (u *UseCase) checkAgentName(r *Report, id, rt, file string) {
	meta, ok := u.runtimeMeta(rt)
	if !ok || !meta.AgentName.declared() || (!meta.AgentSkills.declared() && !fullTier(rt)) {
		return
	}
	b, err := fs.ReadFile(u.project, file)
	if err != nil {
		return // read errors are reported by checkAgentVersion
	}
	if name := meta.AgentName.name(b); name != "" && name != id {
		r.fail(fmt.Sprintf("✗ agent %s (%s): declares name %s", id, rt, name), "why: "+file, NextRebuild)
	}
}
