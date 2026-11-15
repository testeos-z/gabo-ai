package check

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
)

// formerAgentIDs are the agents gabo 0.2.10 built and the neutral spec no
// longer lists (intent 0015 S7, D3, D4). They are fixed here, naming no
// engine: the rule runs in every declared agent directory.
var formerAgentIDs = []string{"gabo-cycle", "gabo-reviewer"}

// checkFormerAgents fails each file in dir named after a former agent that
// carries a gabo-version line, so gabo wrote it. It stays silent when the
// project's own catalog copy still lists the id (the agents-exist rule then
// asks for the file until gabo init upgrades the copy, plan decision PD5),
// and for a file without that line, which is the user's own.
func (u *UseCase) checkFormerAgents(r *Report, dir string, files map[string][]string, spec agentspec.Spec) error {
	for _, id := range formerAgentIDs {
		if slices.Contains(spec.IDs(), id) {
			continue
		}
		for _, name := range files[id] {
			file := path.Join(dir, name)
			b, err := fs.ReadFile(u.project, file)
			if err != nil {
				return fmt.Errorf("read %s: %w", file, err)
			}
			if versionLine(b) == "none" {
				continue
			}
			r.fail(fmt.Sprintf("✗ %s: agent %s is no longer in the agent spec", file, id), NextRebuild)
		}
	}
	return nil
}
