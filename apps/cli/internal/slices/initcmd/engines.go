package initcmd

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	projectconfig "github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// noConfigLine is printed instead of the engine step when the project has
// no gabo.yaml yet (intent 0006 S10, D8).
const noConfigLine = "engine integration: no gabo.yaml yet; after your agent writes it, run gabo init again"

// ErrEngineFailures makes init exit 1 after the report when gabo.yaml cannot
// give the engine set or an engine line is a ✗ (intent 0006 S5, S10, D10).
// The catalog and reviewer work of the same run is kept.
var ErrEngineFailures = errors.New("engine integration failed; fix the ✗ lines above and run gabo init again")

// engineFiles is the one project file init writes per full-tier engine
// (S2, S3, S7); OpenCode's config is found by loadOpencodeConfigs.
var engineFiles = map[string]string{
	engineconfig.Claude: engineconfig.ClaudeSettings,
	engineconfig.Codex:  engineconfig.CodexConfigPath,
	engineconfig.Pi:     engineconfig.PiSettings,
}

// engineSet reads the engine set and the assistant engine from the
// runtimes mapping of gabo.yaml (S1; intent 0010 D1): the set is
// engineconfig.Engines of the roles (intent 0013 S2) in the fixed order of
// engineconfig.Set. runtimes.assistant is one engine or a list; only a
// one-engine assistant is the assistant engine here, so a list leaves every
// main-session key alone. It needs only YAML and that mapping, not a fully
// valid config.
func engineSet(b []byte) ([]string, string, error) {
	var doc struct {
		Runtimes yaml.Node `yaml:"runtimes"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, "", fmt.Errorf("cannot be read as YAML (%v)", err)
	}
	if doc.Runtimes.Kind != yaml.MappingNode {
		return nil, "", errors.New("has no runtimes mapping")
	}
	var rt struct {
		Coder     string                    `yaml:"coder"`
		Reviewer  string                    `yaml:"reviewer"`
		Assistant projectconfig.RuntimeList `yaml:"assistant"`
	}
	if err := doc.Runtimes.Decode(&rt); err != nil {
		return nil, "", fmt.Errorf("runtimes cannot be read (%v)", err)
	}
	assistants := make([]string, len(rt.Assistant))
	for i, a := range rt.Assistant {
		assistants[i] = string(a)
	}
	assistant := strings.TrimSpace(string(rt.Assistant.Single()))
	return engineconfig.Set(engineconfig.Engines(rt.Coder, rt.Reviewer, assistants)...), assistant, nil
}

// engineStep runs each engine's rule in set order and writes each changed
// engine file. A refused or failed engine gets its ✗ line and the next
// engine still runs (S5, D10). Then it removes the gabo main-session value
// from the file of each engine outside the set (intent 0010 S4, D2). It
// reads and writes only project files.
func (u *UseCase) engineStep(set []string, assistant, version string) engineconfig.Report {
	var report engineconfig.Report
	for _, engine := range set {
		report.Sections = append(report.Sections, u.integrateEngine(engine, assistant, version))
	}
	report.Sections = append(report.Sections, u.removeMainSession(set, assistant, version)...)
	return report
}

func (u *UseCase) integrateEngine(engine, assistant, version string) engineconfig.Section {
	in := engineconfig.Input{Files: map[string][]byte{}, Version: version, Assistant: assistant}
	file, owns := engineFiles[engine]
	fail := func(reason string) engineconfig.Section {
		return engineconfig.Section{Engine: engine, Lines: []string{engineconfig.FailurePrefix + " " + file + ": " + reason}}
	}
	if owns {
		if reason := u.refuseEngineFile(file); reason != "" {
			return fail(reason)
		}
		b, err := u.store.ReadFile(u.path(file))
		switch {
		case err == nil:
			in.Files[file] = b
		case !errors.Is(err, fs.ErrNotExist):
			return fail(fmt.Sprintf("cannot read it (%v); fix it and run gabo init again", err))
		}
	}
	if engine == engineconfig.Claude {
		// The rule only needs to know whether the main session's agent is built (S8).
		if _, err := u.store.Lstat(u.path(engineconfig.ClaudeAgentFile)); err == nil {
			in.Files[engineconfig.ClaudeAgentFile] = nil
		}
	}
	var writable []string // the already checked files this engine may write
	if owns {
		writable = []string{file}
	}
	if engine == engineconfig.Opencode {
		if b, err := u.store.ReadFile(u.path(engineconfig.OpencodeStageGate)); err == nil {
			in.Files[engineconfig.OpencodeStageGate] = b
		}
		if _, err := u.store.Lstat(u.path(engineconfig.OpencodeAgentFile)); err == nil {
			in.Files[engineconfig.OpencodeAgentFile] = nil
		}
		var reason string
		if writable, file, reason = u.loadOpencodeConfigs(&in, assistant); reason != "" {
			return fail(reason)
		}
	}
	report, writes := engineconfig.Integrate([]string{engine}, in)
	section := report.Sections[0]
	section.Lines = append(section.Lines, u.writeEngineFiles(writes, writable)...)
	return section
}

// loadOpencodeConfigs reads the OpenCode configs the rule may change
// (intent 0010 S2, S3, S4): always the first existing one (else the new
// .opencode/opencode.json), which must not be refused, since it gets
// experimental.subagent_depth and, with OpenCode as the assistant,
// default_agent; with another assistant also every other existing one that
// is not refused, for the default_agent removal. It returns the files that
// may be written, in lookup order, or the refused or unreadable file and
// why.
func (u *UseCase) loadOpencodeConfigs(in *engineconfig.Input, assistant string) ([]string, string, string) {
	target := engineconfig.OpencodeNewConfig
	for _, p := range engineconfig.OpencodeConfigs {
		if _, err := u.store.Lstat(u.path(p)); err == nil || u.refuseEngineFile(p) != "" {
			target = p
			break
		}
	}
	if reason := u.refuseEngineFile(target); reason != "" {
		return nil, target, reason
	}
	b, err := u.store.ReadFile(u.path(target))
	switch {
	case err == nil:
		in.Files[target] = b
	case !errors.Is(err, fs.ErrNotExist):
		return nil, target, fmt.Sprintf("cannot read it (%v); fix it and run gabo init again", err)
	}
	var read []string
	if assistant != "" && assistant != engineconfig.Opencode {
		read = u.readRemovable(in, engineconfig.OpencodeConfigs)
	}
	var writable []string
	for _, p := range engineconfig.OpencodeConfigs {
		if p == target || slices.Contains(read, p) {
			writable = append(writable, p)
		}
	}
	return writable, "", ""
}

// readRemovable reads each of files that is not refused into in and returns
// those it read. A missing, refused or unreadable file is skipped: gabo init
// only clears its own value from them.
func (u *UseCase) readRemovable(in *engineconfig.Input, files []string) []string {
	var read []string
	for _, file := range files {
		if u.refuseEngineFile(file) != "" {
			continue
		}
		if b, err := u.store.ReadFile(u.path(file)); err == nil {
			in.Files[file] = b
			read = append(read, file)
		}
	}
	return read
}

// writeEngineFiles writes each changed file that is in writable, in
// writable order, and returns a ✗ line for each failed write.
func (u *UseCase) writeEngineFiles(writes map[string][]byte, writable []string) []string {
	var lines []string
	for _, file := range writable {
		data, ok := writes[file]
		if !ok {
			continue
		}
		if err := u.store.WriteFile(u.path(file), data); err != nil {
			lines = append(lines, engineconfig.FailurePrefix+" "+file+": cannot write it ("+err.Error()+"); fix it and run gabo init again")
		}
	}
	return lines
}

// removeMainSession reads the files of each engine outside set that has a
// main-session removal and writes the files the removal changes. gabo init
// no longer integrates those engines, so a refused, missing or unreadable
// file is left alone without a line; only a failed write is a ✗ line.
func (u *UseCase) removeMainSession(set []string, assistant, version string) []engineconfig.Section {
	in := engineconfig.Input{Files: map[string][]byte{}, Version: version, Assistant: assistant}
	read := map[string][]string{}
	for _, engine := range engineconfig.Set(engineconfig.Claude, engineconfig.Codex, engineconfig.Opencode, engineconfig.Pi) {
		if engineconfig.Removable(engine) && !slices.Contains(set, engine) {
			read[engine] = u.readRemovable(&in, engineconfig.RemovalFiles(engine))
		}
	}
	report, writes := engineconfig.Remove(set, in)
	for i, section := range report.Sections {
		report.Sections[i].Lines = append(section.Lines, u.writeEngineFiles(writes, read[section.Engine])...)
	}
	return report.Sections
}

// refuseEngineFile returns why file must not be written: its engine
// directory or the file is a symlink or of the wrong kind (S5). Empty means
// the file may be read and written.
func (u *UseCase) refuseEngineFile(file string) string {
	checks := []struct {
		rel string
		dir bool
	}{{path.Dir(file), true}, {file, false}}
	if checks[0].rel == "." { // a file at the project root has no engine directory
		checks = checks[1:]
	}
	for _, p := range checks {
		mode, err := u.store.Lstat(u.path(p.rel))
		name := p.rel
		if !p.dir {
			name = "it"
		}
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return ""
		case err != nil:
			return fmt.Sprintf("cannot stat %s (%v); fix it and run gabo init again", p.rel, err)
		case mode&fs.ModeSymlink != 0:
			return name + " is a symlink; gabo init writes only inside the project, so replace it with a real " + kind(p.dir) + " and run gabo init again"
		case p.dir && !mode.IsDir():
			return name + " is not a directory; move it away and run gabo init again"
		case !p.dir && !mode.IsRegular():
			return "not a regular file; move it away and run gabo init again"
		}
	}
	return ""
}

func kind(dir bool) string {
	if dir {
		return "directory"
	}
	return "file"
}
