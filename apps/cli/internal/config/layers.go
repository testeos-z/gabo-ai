package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Layer names, lowest to highest precedence. LayerFlag is the single file
// given with --config, which replaces the layered lookup.
const (
	LayerSystem  = "system"
	LayerUser    = "user"
	LayerProject = "project"
	LayerFlag    = "flag"
)

// ProjectFile is the project layer's file name at the project root.
const ProjectFile = "gbai.yaml"

// Layer is one config file in the resolution order. An empty Path means the
// location could not be resolved (for example no HOME) and is skipped.
type Layer struct {
	Name  string
	Path  string
	Found bool
	// Version is the file's schema version; 0 when the file has none.
	Version int
}

// Resolution is the merged, defaulted and validated config together with the
// layers it was looked up in.
type Resolution struct {
	Layers []Layer
	Config Config
	// Warnings are non-fatal problems, such as a personal key set in the
	// project file and ignored.
	Warnings []string
}

// Reader reads a file; a missing file is reported with an error matching
// fs.ErrNotExist (kernel.FileSystem satisfies it).
type Reader interface {
	ReadFile(name string) ([]byte, error)
}

// LayerPaths returns the system, user and project layers in precedence order.
func LayerPaths(getenv func(string) string, projectRoot string) []Layer {
	project := ""
	if projectRoot != "" {
		project = path.Join(projectRoot, ProjectFile)
	}
	return []Layer{
		{Name: LayerSystem, Path: kernel.ServerConfigPath},
		{Name: LayerUser, Path: UserPath(getenv)},
		{Name: LayerProject, Path: project},
	}
}

// UserPath is $XDG_CONFIG_HOME/gbai/config.yaml, else ~/.config/gbai/config.yaml;
// relative values are ignored, as the XDG spec requires. Empty when neither resolves.
func UserPath(getenv func(string) string) string {
	if xdg := getenv("XDG_CONFIG_HOME"); path.IsAbs(xdg) {
		return path.Join(xdg, "gbai", "config.yaml")
	}
	if home := getenv("HOME"); path.IsAbs(home) {
		return path.Join(home, ".config", "gbai", "config.yaml")
	}
	return ""
}

// ProjectRoot is the git top-level of cwd, or cwd itself when cwd is not in a
// git repository, git cannot run or runner is nil.
func ProjectRoot(ctx context.Context, runner kernel.Runner, cwd string) string {
	if runner == nil {
		return cwd
	}
	res, err := runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}, Dir: cwd})
	if err != nil || res.ExitCode != 0 {
		return cwd
	}
	top := strings.TrimSpace(string(res.Stdout))
	if !path.IsAbs(top) || strings.ContainsAny(top, "\n\r") {
		return cwd
	}
	return top
}

// LoadFile loads exactly one file (--config): no layering.
func LoadFile(r Reader, name string) (Resolution, error) {
	res := Resolution{Layers: []Layer{{Name: LayerFlag, Path: name}}}
	b, err := r.ReadFile(name)
	if err != nil {
		return res, fmt.Errorf("read config: %w", err)
	}
	res.Layers[0].Found = true
	cfg, err := Parse(b)
	if err != nil {
		return res, fmt.Errorf("config %s: %w", name, err)
	}
	res.Layers[0].Version = cfg.Version
	res.Config = cfg
	return res, nil
}

// LoadLayers reads every existing layer in order and merges them: a higher
// layer overrides a lower one field by field (see Merge). Missing files are
// skipped; unknown keys fail per file. Defaults and validation run once on
// the merged result. Layers are returned with Found set, even on error.
func LoadLayers(r Reader, layers []Layer) (Resolution, error) {
	res := Resolution{Layers: append([]Layer(nil), layers...)}
	var merged map[string]any
	var used []string
	for i, l := range res.Layers {
		if l.Path == "" {
			continue
		}
		b, err := r.ReadFile(l.Path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return res, fmt.Errorf("read config %s: %w", l.Path, err)
		}
		res.Layers[i].Found = true
		doc, version, err := decodeLayer(b)
		if err != nil {
			return res, fmt.Errorf("config %s: %w", l.Path, err)
		}
		res.Layers[i].Version = version
		if l.Name == LayerProject {
			res.Warnings = append(res.Warnings, dropPersonalKeys(doc, l.Path)...)
		}
		merged = Merge(merged, doc)
		used = append(used, l.Path)
	}
	if len(used) == 0 {
		var looked []string
		for _, l := range res.Layers {
			if l.Path != "" {
				looked = append(looked, l.Path)
			}
		}
		return res, fmt.Errorf("no config file found (looked in %s); create one or pass --config <file>", strings.Join(looked, ", "))
	}
	b, err := yaml.Marshal(merged)
	if err != nil {
		return res, fmt.Errorf("encode merged config: %w", err)
	}
	cfg, err := Parse(b)
	if err != nil {
		return res, fmt.Errorf("config merged from %s: %w", strings.Join(used, ", "), err)
	}
	res.Config = cfg
	return res, nil
}

// decodeLayer checks a layer's keys against Config (unknown keys fail) and
// returns it as a generic mapping with its schema version. An empty document
// is an empty layer of version 0.
func decodeLayer(b []byte) (map[string]any, int, error) {
	var probe Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&probe); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("decode yaml: %w", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, 0, fmt.Errorf("decode yaml: %w", err)
	}
	return doc, probe.Version, nil
}

// Merge returns lower overridden by higher: mappings present in both merge
// key by key; any other value in higher (scalar, list or null) replaces the
// lower one; a key absent in higher keeps the lower value. Inputs are not
// modified.
func Merge(lower, higher map[string]any) map[string]any {
	out := make(map[string]any, len(lower)+len(higher))
	for k, v := range lower {
		out[k] = v
	}
	for k, hv := range higher {
		lm, lok := out[k].(map[string]any)
		hm, hok := hv.(map[string]any)
		if lok && hok {
			out[k] = Merge(lm, hm)
			continue
		}
		out[k] = hv
	}
	return out
}
