// Package skills embeds the agent skills shipped with gbai. Each skill is a
// directory holding a SKILL.md; contract skills also hold example adapters
// under examples/.
package skills

import (
	"archive/tar"
	"bytes"
	"embed"
	"io/fs"
	"path"
	"slices"
	"time"
)

//go:embed */SKILL.md */examples/*.md
var content embed.FS

// File is one embedded file; Path is relative, like gbai-setup/SKILL.md or
// gbai-contract-tasks/examples/tasks-gh.md.
type File struct {
	Path string
	Data []byte
}

// Files returns every embedded file in path order.
func Files() ([]File, error) {
	var out []File
	err := fs.WalkDir(content, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
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

// Names lists the skill names in sorted order.
func Names() []string {
	entries, _ := content.ReadDir(".")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
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
