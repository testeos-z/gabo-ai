package check

import (
	"fmt"
	"io/fs"
)

// Agent version lines (intent 0008 S7, D3): never skipped silently.
const (
	notComparedLine = "✗ agent versions not compared: no valid catalog stamp"
	nextInitRebuild = "next: run gabo init, then ask your agent to rebuild the agents"
)

// checkVersionsComparable reports once that the agent versions cannot be
// compared when there is no valid catalog stamp.
func checkVersionsComparable(r *Report, hasStamp bool) {
	if !hasStamp {
		r.fail(notComparedLine, nextInitRebuild)
	}
}

// checkAgentVersion requires the agent file to carry a gabo-version line
// holding x.y.z and, with a valid stamp, to equal it.
func (u *UseCase) checkAgentVersion(r *Report, id, rt, file, stamp string, hasStamp bool) error {
	b, err := fs.ReadFile(u.project, file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}
	v := versionLine(b)
	if _, ok := parseVersion(v); !ok {
		r.fail(fmt.Sprintf("✗ agent %s (%s): no gabo-version line", id, rt), "why: "+file, NextRebuild)
		return nil
	}
	if hasStamp && v != stamp {
		r.fail(fmt.Sprintf("✗ agent %s built with %s, catalog is %s: ask your agent to rebuild", id, v, stamp),
			"why: "+file, NextRebuild)
	}
	return nil
}
