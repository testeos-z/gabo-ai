package check

import (
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// checkLeftovers fails, for each runtime declared in runtimes.* whose skill
// declares metadata.former_skill_dir, on every gabo-* symlink left in that
// directory that points into the catalog copy or to a slot adapter folder
// of gabo.yaml (intent 0008 S4). Any other entry is the user's and is not
// reported; a missing directory prints nothing.
func (u *UseCase) checkLeftovers(r *Report, cfg config.Config) {
	var current, former []string
	for _, rt := range runtimesOf(cfg) {
		meta, ok := u.runtimeMeta(rt)
		if !ok {
			continue
		}
		if meta.SkillDir != "" {
			current = append(current, path.Clean(meta.SkillDir))
		}
		if meta.FormerSkillDir == "" {
			continue
		}
		if !insideRepo(meta.FormerSkillDir) {
			r.fail("✗ runtime "+rt+": former_skill_dir must stay inside the repository",
				"next: set metadata.former_skill_dir in gabo-runtime-"+rt+" to a repository-relative path, then rerun gabo check")
			continue
		}
		dir := path.Clean(meta.FormerSkillDir)
		if !slices.Contains(former, dir) {
			former = append(former, dir)
		}
	}
	var folders []string
	for _, slot := range slotNames(cfg) {
		if folder, ok := u.adapterFolder(slot, cfg.Slots[slot].Adapter); ok {
			folders = append(folders, folder)
		}
	}
	for _, dir := range former {
		if slices.Contains(current, dir) {
			continue // still a skill_dir: its links are current
		}
		entries, err := fs.ReadDir(u.project, dir)
		if err != nil {
			continue // missing or unreadable: nothing left to report
		}
		for _, e := range entries {
			link := path.Join(dir, e.Name())
			if strings.HasPrefix(e.Name(), "gabo-") && u.gaboLink(link, folders) {
				r.fail("✗ "+link+": gabo link left after the move to .agents/skills", NextRebuild)
			}
		}
	}
}

// gaboLink reports whether link is a symlink whose relative target resolves
// into the catalog copy or to one of the slot adapter folders.
func (u *UseCase) gaboLink(link string, folders []string) bool {
	info, err := fs.Lstat(u.project, link)
	if err != nil || info.Mode()&fs.ModeSymlink == 0 {
		return false
	}
	target, err := fs.ReadLink(u.project, link)
	if err != nil || path.IsAbs(target) {
		return false
	}
	resolved := path.Join(path.Dir(link), target)
	return strings.HasPrefix(resolved, CatalogDir+"/") || slices.Contains(folders, resolved)
}
