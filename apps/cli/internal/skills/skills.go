// Package skills embeds the agent skills shipped with gabo. Each skill is a
// directory holding a SKILL.md; contract skills also hold example adapters
// under examples/, directiva skills hold plan templates under templates/, and
// gabo-reviewer holds the reviewer scripts and their shell tests under
// scripts/ (TRD D20), and gabo-agent-templates holds the neutral agent spec
// (agents.yaml) with one body template per agent under agents/ (TRD D24).
// gabo-reviewer-config holds the codex and opencode isolation-test script and
// its shell test under scripts/ (intent 0001 plan T5).
// gabo-runtime-opencode holds the OpenCode stage-gate plugin under plugins/
// (intent 0004 plan T6).
package skills

import (
	"archive/tar"
	"bytes"
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

//go:embed */SKILL.md */examples/*.md */templates/*.md
//go:embed gabo-agent-templates/agents.yaml gabo-agent-templates/agents/*.md
//go:embed gabo-runtime-opencode/plugins/*.js
//go:embed gabo-reviewer-config/scripts/*.sh gabo-reviewer-config/scripts/tests/*.sh
//go:embed gabo-reviewer/README.md gabo-reviewer/scripts/*.sh gabo-reviewer/scripts/config.env gabo-reviewer/scripts/tests/*.sh
var content embed.FS

// File is one embedded file; Path is relative, like gabo-setup/SKILL.md or
// gabo-contract-tasks/examples/tasks-gh.md.
type File struct {
	Path string
	Data []byte
}

// reviewerSkill is installed only at user level (TRD D20, T041), never with
// the catalog, so PR code cannot change the scripts that review it.
const reviewerSkill = "gabo-reviewer"

// Catalog returns the embedded catalog as a filesystem rooted like the
// project's .gabo/skills/ copy, so without the reviewer skill.
func Catalog() fs.FS { return catalogFS{} }

type catalogFS struct{}

func (catalogFS) Open(name string) (fs.File, error) {
	if dir, _, _ := strings.Cut(name, "/"); dir == reviewerSkill {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return content.Open(name)
}

// Files returns every embedded catalog file in path order, without the
// reviewer skill.
func Files() ([]File, error) {
	return walk(func(dir string) bool { return dir != reviewerSkill })
}

// ReviewerFiles returns the files of the gabo-reviewer skill (SKILL.md,
// README.md and scripts/) in path order.
func ReviewerFiles() ([]File, error) {
	return walk(func(dir string) bool { return dir == reviewerSkill })
}

func walk(keep func(dir string) bool) ([]File, error) {
	var out []File
	err := fs.WalkDir(content, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if dir, _, _ := strings.Cut(p, "/"); !keep(dir) {
			return nil
		}
		b, err := content.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: p, Data: b})
		return nil
	})
	return out, err
}

// Names lists the catalog skill names in sorted order.
func Names() []string {
	entries, _ := content.ReadDir(".")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() && e.Name() != reviewerSkill {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}

// Tar returns a deterministic tar archive of the skills: directories 0755,
// files 0644, owner 0.
func Tar() ([]byte, error) {
	files, err := Files()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	epoch := time.Unix(0, 0)
	dirs := map[string]bool{}
	for _, f := range files {
		if dir := path.Dir(f.Path); !dirs[dir] {
			dirs[dir] = true
			if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: dir + "/", Mode: 0o755, ModTime: epoch}); err != nil {
				return nil, err
			}
		}
		h := &tar.Header{Typeflag: tar.TypeReg, Name: f.Path, Mode: 0o644, Size: int64(len(f.Data)), ModTime: epoch}
		if err := tw.WriteHeader(h); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
