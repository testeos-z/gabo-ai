// Package check validates a project read-only (`gabo check`, PRD R33, TRD
// 9.4), engine-neutral except for the full-tier engine files: the config files, every agent of the neutral
// agent spec in each runtime's declared agent directory, each agent's
// version line, the skill and reviewer links a runtime skill declares
// (metadata.skill_dir, metadata.reviewer_link), OpenCode's project-level
// experimental.subagent_depth when a role runs on OpenCode, the engine
// integration keys gabo init writes for claude, codex and pi, every slot adapter's
// contract coverage, the catalog version stamp and the stage of each
// intent folder under docs/product/intents/.
//
// The spec and the contracts come from the project's catalog copy (.gabo/skills/), never from
// the binary, so a newer CLI checks an older project against its own version.
package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// Project-relative locations (TRD 9.4).
const (
	CatalogDir  = ".gabo/skills"
	RuntimesDir = ".gabo/runtimes"
	StampFile   = ".gabo/VERSION"
	// installPath is the module path of the binary for the upgrade command
	// (TRD 9.4), as in initcmd.
	installPath = "github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo"
	// runtimeBuilder writes a runtime skill for an engine without one.
	runtimeBuilder = "gabo-runtime-builder"
)

// Output lines other packages and tests rely on.
const (
	PassedLine         = "check passed: config, agents, slot coverage and catalog version"
	UpgradePRLine      = "upgrade PR: only config validated, human merge required"
	NextFixConfig      = "next: fix the keys above, then rerun gabo check"
	NextBuildAgent     = "next: ask your agent to build it with the runtime skill, then rerun gabo check"
	NextRebuild        = "next: ask your agent to rebuild the agents, then rerun gabo check"
	NextRuntimeBuilder = "next: ask your agent to write it with " + runtimeBuilder + ", then rerun gabo check"
)

// Settings carry the CLI version, the config layers and the upgrade-PR mode.
type Settings struct {
	// Version is the CLI version, with or without a leading v.
	Version string
	// Layers are the user and project config layers (config.LayerPaths).
	Layers []config.Layer
	// UpgradePR (GABO_UPGRADE_PR=1) runs only the config validation.
	UpgradePR bool
	// Former are the config locations from before the gabo-ai rename
	// (config.FormerNames); any that exists fails and is never read.
	Former []config.FormerName
	// Getenv expands the reviewer link a runtime skill declares.
	Getenv func(string) string
	// ReviewerDir is the user-level reviewer skill dir gabo init installs,
	// ${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer; empty when
	// it cannot be known.
	ReviewerDir string
	// Root is the absolute project directory; the pre-write hook mode
	// resolves absolute paths against it.
	Root string
	// Resolve returns a path with every symlink resolved
	// (filepath.EvalSymlinks); nil compares paths as written.
	Resolve func(string) (string, error)
	// Readlink reports whether a path is a symlink and its target as
	// stored (os.Lstat plus os.Readlink), so the pre-write hook mode can
	// follow a dangling link; nil follows none.
	Readlink func(string) (target string, isLink bool, err error)
	// WorkDir is the absolute working directory; the pre-write hook mode
	// takes relative paths from it. Empty means Root.
	WorkDir string
	// FoldCase reports a case-insensitive filesystem at Root: the pre-write
	// hook mode then matches the root in an absolute path without case.
	FoldCase bool
	// CI (CI set) skips the user-level reviewer link: the reviewer scripts
	// and CI run with their own HOME.
	CI bool
}

// Report is the human-readable result. Failed is true when any check failed.
type Report struct {
	Lines  []string
	Failed bool
}

// UseCase runs the checks. It only reads.
type UseCase struct {
	project fs.FS
	config  config.Reader
	user    UserFS
	s       Settings
}

// NewUseCase wires the use case: project is rooted at the project directory
// (symlinks are read through fs.ReadLinkFS), cfg reads the config layers by
// their paths and user reads the user-level reviewer link.
func NewUseCase(project fs.FS, cfg config.Reader, user UserFS, s Settings) *UseCase {
	return &UseCase{project: project, config: cfg, user: user, s: s}
}

// Run checks the project. A returned error is a read error (exit 2); check
// failures are in the report.
func (u *UseCase) Run() (Report, error) {
	var r Report
	if err := u.checkFormer(&r); err != nil {
		return Report{}, err
	}
	cfg, ok, err := u.checkConfig(&r)
	if err != nil {
		return Report{}, err
	}
	if u.s.UpgradePR {
		r.Lines = append(r.Lines, UpgradePRLine)
		return r, nil
	}
	stamp, hasStamp := u.checkStamp(&r)
	if ok {
		if err := u.checkAgents(&r, cfg, stamp, hasStamp); err != nil {
			return Report{}, err
		}
		if err := u.checkLinks(&r, cfg); err != nil {
			return Report{}, err
		}
		if err := u.checkSubagentDepth(&r, cfg); err != nil {
			return Report{}, err
		}
		if err := u.checkEngineKeys(&r, cfg); err != nil {
			return Report{}, err
		}
		if err := u.checkSlots(&r, cfg); err != nil {
			return Report{}, err
		}
	}
	if err := u.checkIntents(&r); err != nil {
		return Report{}, err
	}
	if !r.Failed {
		r.Lines = append(r.Lines, PassedLine)
	}
	return r, nil
}

func (r *Report) fail(lines ...string) {
	r.Failed = true
	r.Lines = append(r.Lines, lines...)
}

func (r *Report) warn(line string) { r.Lines = append(r.Lines, "! "+line) }

// checkFormer fails for each config location still under its name from
// before the gabo-ai rename, with the rename to make, once per location. It
// only probes.
func (u *UseCase) checkFormer(r *Report) error {
	var reported []string
	for _, f := range u.s.Former {
		if slices.Contains(reported, f.From) {
			continue
		}
		_, err := u.config.ReadFile(f.Probe)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read config %s: %w", f.Probe, err)
		}
		reported = append(reported, f.From)
		r.fail("✗ "+f.Probe+" uses a name from before the gabo-ai rename and is not read",
			"next: rename "+f.From+" to "+f.To+", then rerun gabo check")
	}
	return nil
}

// checkConfig loads the layers through the config package: each layer and
// the merged result are validated against their schemas, personal keys in
// the project file warn and former cloud keys fail.
func (u *UseCase) checkConfig(r *Report) (config.Config, bool, error) {
	var projectPath string
	for _, l := range u.s.Layers {
		if l.Name == config.LayerProject {
			projectPath = l.Path
		}
	}
	if projectPath != "" {
		_, err := u.config.ReadFile(projectPath)
		if errors.Is(err, fs.ErrNotExist) {
			projectPath = ""
		} else if err != nil {
			return config.Config{}, false, fmt.Errorf("read config %s: %w", projectPath, err)
		}
	}
	if projectPath == "" {
		r.fail("✗ no "+config.ProjectFile+" found", "next: run gabo init, then ask your agent to configure gabo-ai")
		return config.Config{}, false, nil
	}
	res, err := config.LoadLayers(u.config, u.s.Layers)
	for _, w := range res.Warnings {
		r.warn(w)
	}
	var fe *config.FileError
	if errors.As(err, &fe) {
		// A key's own next step (other than "run setup") is kept, before the
		// general one.
		var next []string
		for _, line := range strings.Split(fe.Err.Error(), "\n") {
			line = strings.TrimPrefix(strings.TrimSpace(line), "✗ ")
			if line == "" || line == config.NextSetup {
				continue
			}
			if strings.HasPrefix(line, "next:") {
				next = append(next, line)
				continue
			}
			r.fail("✗ " + fe.File + ": " + line)
		}
		for _, line := range next {
			r.fail(line)
		}
		r.fail(NextFixConfig)
		return config.Config{}, false, nil
	}
	if err != nil {
		return config.Config{}, false, err
	}
	return res.Config, true, nil
}

// checkStamp compares the catalog stamp with the CLI version: older warns,
// missing or newer fails. It returns the stamp when it is a valid version.
func (u *UseCase) checkStamp(r *Report) (string, bool) {
	b, err := fs.ReadFile(u.project, StampFile)
	if errors.Is(err, fs.ErrNotExist) {
		r.fail("✗ no stamp: run gabo init")
		return "", false
	}
	if err != nil {
		r.fail("✗ cannot read "+StampFile+": "+err.Error(), "next: restore it from git or rerun gabo init")
		return "", false
	}
	raw := strings.TrimSpace(string(b))
	stamp, ok := parseVersion(raw)
	if !ok {
		r.fail(fmt.Sprintf("✗ %s holds %q, not a version x.y.z", StampFile, raw), "next: restore it from git or delete it, then rerun gabo init")
		return "", false
	}
	cli, ok := parseVersion(strings.TrimPrefix(u.s.Version, "v"))
	switch {
	case !ok:
		r.warn(fmt.Sprintf("gabo %s is not a release version x.y.z; catalog version %s not compared", u.s.Version, stamp))
	case stamp.compare(cli) < 0:
		r.warn("this project uses gabo " + stamp.String() + "; run gabo init to upgrade it")
	case stamp.compare(cli) > 0:
		r.fail("✗ catalog version "+stamp.String()+" is newer than gabo "+cli.String(),
			"next: run go install "+installPath+"@v"+stamp.String())
	}
	return stamp.String(), true
}

// checkAgents requires, for each distinct runtime, every spec agent in the
// directory its runtime skill declares, with a version line x.y.z equal to
// the stamp when the stamp is valid (checkAgentVersion).
func (u *UseCase) checkAgents(r *Report, cfg config.Config, stamp string, hasStamp bool) error {
	checkVersionsComparable(r, hasStamp)
	catalog, err := fs.Sub(u.project, CatalogDir)
	if err != nil {
		return err
	}
	spec, err := agentspec.Load(catalog)
	if err != nil {
		r.fail("✗ agent spec: "+err.Error(), "next: run gabo init to restore the catalog, then rerun gabo check")
		return nil
	}
	var checked []string
	for _, rt := range runtimesOf(cfg) {
		dir, ok := u.agentDir(r, rt)
		if !ok || slices.Contains(checked, dir) {
			continue
		}
		checked = append(checked, dir)
		files, err := u.agentFiles(dir)
		if err != nil {
			return err
		}
		for _, id := range spec.IDs() {
			names := files[id]
			if len(names) == 0 {
				r.fail("✗ agent "+id+" missing", fmt.Sprintf("why: no file named %s in %s (runtime %s)", id, dir, rt), NextBuildAgent)
				continue
			}
			for _, name := range names {
				if err := u.checkAgentVersion(r, id, rt, path.Join(dir, name), stamp, hasStamp); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// agentDir returns the agent directory the runtime skill declares; it reports
// a failure and returns false when there is none or it leaves the repository.
func (u *UseCase) agentDir(r *Report, rt string) (string, bool) {
	skill := "gabo-runtime-" + rt
	b, err := u.runtimeSkill(rt)
	if err != nil {
		r.fail("✗ runtime "+rt+" has no runtime skill "+skill, NextRuntimeBuilder)
		return "", false
	}
	meta, err := parseRuntimeMeta(b)
	dir := meta.AgentDir
	if err != nil {
		r.fail("✗ runtime "+rt+": "+skill+" frontmatter: "+err.Error(), NextRuntimeBuilder)
		return "", false
	}
	if dir == "" {
		r.fail("✗ runtime "+rt+": "+skill+" declares no metadata.agent_dir", NextRuntimeBuilder)
		return "", false
	}
	if !insideRepo(dir) {
		r.fail("✗ runtime "+rt+": agent_dir must stay inside the repository",
			"next: set metadata.agent_dir in "+skill+" to a repository-relative path, then rerun gabo check")
		return "", false
	}
	return path.Clean(dir), true
}

// agentFiles lists the files directly in dir (not recursive) by base name
// without extension. A missing directory has no files.
func (u *UseCase) agentFiles(dir string) (map[string][]string, error) {
	entries, err := fs.ReadDir(u.project, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	files := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		id := strings.TrimSuffix(e.Name(), path.Ext(e.Name()))
		files[id] = append(files[id], e.Name())
	}
	return files, nil
}

// versionRE finds the agent version line in a frontmatter or a comment.
var versionLineRE = regexp.MustCompile(`(?m)gabo-version:[ \t]*([0-9A-Za-z.+-]+)`)

// versionLine returns the agent's gabo-version, or "none".
func versionLine(b []byte) string {
	if m := versionLineRE.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return "none"
}

// version is a bare semantic version x.y.z, as the stamp holds it. It
// mirrors initcmd's own parser; slices do not import each other.
type version [3]int

var versionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func parseVersion(s string) (version, bool) {
	m := versionRE.FindStringSubmatch(s)
	if m == nil {
		return version{}, false
	}
	var v version
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return version{}, false
		}
		v[i] = n
	}
	return v, true
}

func (v version) compare(o version) int {
	for i := range v {
		if v[i] != o[i] {
			if v[i] < o[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (v version) String() string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }
