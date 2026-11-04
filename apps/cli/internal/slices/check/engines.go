package check

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// keyFiles are the engines whose keys gabo check verifies (intent 0006 S11)
// and the one project file each reads: Claude Code (S2), Codex (S3) and Pi
// (S7). OpenCode, also in the full tier, has no such key (S6); any other
// engine is in the flexible tier, never checked and never read.
var keyFiles = map[string]string{
	engineconfig.Claude: engineconfig.ClaudeSettings,
	engineconfig.Codex:  engineconfig.CodexConfigPath,
	engineconfig.Pi:     engineconfig.PiSettings,
}

// keyProbeVersion is the release version given to the gabo init rules when
// they only decide what is missing: the Pi rule pins nothing for a source
// build, and any version of the Pi entry passes (S11).
const keyProbeVersion = "0.0.0"

// piMissingLine names both fixes, because a source build of gabo init writes
// nothing for Pi (S7, S11).
var piMissingLine = engineconfig.FailurePrefix + " " + engineconfig.PiSettings + ": packages entry npm:" + engineconfig.PiPackage +
	` missing; run gabo init (a release build) or add "npm:` + engineconfig.PiPackage + `@<version>" to packages`

// checkEngineKeys fails for each engine integration key gabo init writes
// that the project lacks, for the engines the runtime roles name (S11). It
// only reads.
func (u *UseCase) checkEngineKeys(r *Report, cfg config.Config) error {
	lines, err := engineKeyFailures(u.project, engineconfig.Set(runtimesOf(cfg)...))
	if err != nil {
		return err
	}
	if len(lines) > 0 {
		r.fail(lines...)
	}
	return nil
}

// engineKeyFailures runs the gabo init rule of each checked engine in set on
// the project's current file and turns what it would add into failure
// lines: `✗ <path>: <key> missing; run gabo init`, one per key, or the Pi
// line. A key present with another value passes (S4, D6); a file the rule
// cannot read fails with the rule's own ✗ line. It reads only the files of
// the checked engines and writes nothing.
func engineKeyFailures(project fs.FS, set []string) ([]string, error) {
	var out []string
	for _, engine := range set {
		file, ok := keyFiles[engine]
		if !ok {
			continue
		}
		in := engineconfig.Input{Files: map[string][]byte{}, Version: keyProbeVersion}
		b, err := fs.ReadFile(project, file)
		switch {
		case err == nil:
			in.Files[file] = b
		case !errors.Is(err, fs.ErrNotExist):
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		report, writes := engineconfig.Integrate([]string{engine}, in)
		for _, s := range report.Sections {
			for _, l := range s.Lines {
				if strings.HasPrefix(l, engineconfig.FailurePrefix) {
					out = append(out, l)
				}
			}
		}
		if _, changes := writes[file]; !changes {
			continue
		}
		if engine == engineconfig.Pi {
			out = append(out, piMissingLine)
			continue
		}
		keys := addedKeys(report, file)
		if len(keys) == 0 {
			// The rule would write without naming its keys: still a failure.
			keys = []string{"gabo integration keys"}
		}
		for _, key := range keys {
			out = append(out, engineconfig.FailurePrefix+" "+file+": "+key+" missing; run gabo init")
		}
	}
	return out, nil
}

// addedKeys returns the keys named by the rule's `wrote <file>: added k1,
// k2` line, so the key names come from engineconfig alone.
func addedKeys(report engineconfig.Report, file string) []string {
	prefix := "wrote " + file + ": added "
	for _, s := range report.Sections {
		for _, l := range s.Lines {
			if rest, ok := strings.CutPrefix(l, prefix); ok {
				return strings.Split(rest, ", ")
			}
		}
	}
	return nil
}
