package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// NextCoverAdapter follows a gapped adapter.
const NextCoverAdapter = "next: add a ### <operation> heading with its commands under ## Operations for each, then rerun gabo check"

// Coverage compares the operation headings of a contract skill with those of
// an adapter file: an operation is a "### <name>" heading, spelled exactly,
// under the "## Operations" section. It returns the contract's operations the
// adapter covers and those it misses, both in contract order.
func Coverage(contract, adapter []byte) (covered, missing []string) {
	have := operations(adapter)
	for _, op := range operations(contract) {
		if slices.Contains(have, op) {
			covered = append(covered, op)
		} else {
			missing = append(missing, op)
		}
	}
	return covered, missing
}

// operations lists the "### " headings of the "## Operations" section,
// skipping fenced code blocks.
func operations(doc []byte) []string {
	var out []string
	in, fence := false, false
	for _, line := range strings.Split(string(doc), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~"):
			fence = !fence
		case fence:
		case strings.HasPrefix(line, "## "):
			in = strings.TrimSpace(line) == "## Operations"
		case in && strings.HasPrefix(line, "### "):
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "### ")))
		}
	}
	return out
}

// checkSlots requires every configured slot adapter to exist and to cover
// every operation of its contract skill, read from the catalog copy.
func (u *UseCase) checkSlots(r *Report, cfg config.Config) error {
	for _, name := range slotNames(cfg) {
		skill := "gabo-contract-" + strings.ReplaceAll(name, "_", "-")
		contractPath := path.Join(CatalogDir, skill, "SKILL.md")
		contract, err := fs.ReadFile(u.project, contractPath)
		if errors.Is(err, fs.ErrNotExist) {
			r.fail("✗ slot "+name+": contract "+skill+" missing from "+CatalogDir,
				"next: run gabo init to restore the catalog, then rerun gabo check")
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", contractPath, err)
		}
		file := u.adapterPath(name, cfg.Slots[name].Adapter)
		adapter, err := fs.ReadFile(u.project, file)
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
			r.fail("✗ slot "+name+": adapter "+file+" not found",
				"next: write it with gabo-adapter-writer or fix slots."+name+".adapter, then rerun gabo check")
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}
		if _, missing := Coverage(contract, adapter); len(missing) > 0 {
			r.fail("✗ adapter "+file+" does not cover: "+strings.Join(missing, ", "),
				"why: slots."+name+".adapter, contract "+contractPath, NextCoverAdapter)
		}
	}
	return nil
}

// adapterPath resolves slots.<name>.adapter to a project file. A methodology
// adapter may be a bare skill name, read from the catalog copy (config.Slot).
func (u *UseCase) adapterPath(slot, adapter string) string {
	adapter = path.Clean(strings.TrimSpace(adapter))
	if slot == config.SlotMethodology && !strings.Contains(adapter, "/") && path.Ext(adapter) == "" {
		return path.Join(CatalogDir, adapter, "SKILL.md")
	}
	return adapter
}
