package config

import (
	"strings"
	"testing"
)

func TestFileWithoutVersionReadsAsVersionZero(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Version != 0 {
		t.Fatalf("version = %d, want 0", cfg.Version)
	}
	res, err := LoadLayers(files{
		"/etc/gbai/gbai.yaml":              minimal,
		"/home/o/.config/gbai/config.yaml": "version: 1\nalias: Leo\n",
	}, threeLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if got := []int{res.Layers[0].Version, res.Layers[1].Version}; got[0] != 0 || got[1] != 1 {
		t.Fatalf("layer versions = %v, want [0 1]", got)
	}
}

func TestVersionMustBeSupported(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"-1", "2"} {
		if _, err := Parse([]byte(minimal + "version: " + v + "\n")); err == nil || !strings.Contains(err.Error(), "version") {
			t.Errorf("version %s: error = %v, want a version error", v, err)
		}
	}
	if _, err := Parse([]byte(minimal + "version: 1\n")); err != nil {
		t.Errorf("version 1: error = %v", err)
	}
}

func TestPersonalKeysResolveGlobalFirst(t *testing.T) {
	t.Parallel()
	r := files{
		"/etc/gbai/gbai.yaml":              minimal,
		"/home/o/.config/gbai/config.yaml": "alias: Leo\ninteraction_level: balanced\ntone: direct, brief\nlanguage: es\n",
		"/src/p/gbai.yaml":                 "alias: Other\ninteraction_level: high\ntone: chatty\nlanguage: en\nparallelism: 5\n",
	}
	res, err := LoadLayers(r, threeLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	c := res.Config
	if c.Alias != "Leo" || c.InteractionLevel != "balanced" || c.Tone != "direct, brief" || c.Language != "es" {
		t.Errorf("personal keys = %q %q %q %q, want the global values", c.Alias, c.InteractionLevel, c.Tone, c.Language)
	}
	if c.Parallelism != 5 {
		t.Errorf("parallelism = %d, want project 5: project keys keep user -> project merge", c.Parallelism)
	}
	if len(res.Warnings) != 4 {
		t.Fatalf("warnings = %q, want one per ignored personal key", res.Warnings)
	}
	for i, key := range PersonalKeys {
		if w := res.Warnings[i]; !strings.Contains(w, key) || !strings.Contains(w, "/src/p/gbai.yaml") {
			t.Errorf("warning %d = %q, want it to name %s and the project file", i, w, key)
		}
	}
}

func TestProjectPersonalKeyIsIgnoredEvenWithoutAGlobalValue(t *testing.T) {
	t.Parallel()
	res, err := LoadLayers(files{"/src/p/gbai.yaml": minimal + "alias: Other\n"}, threeLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if res.Config.Alias != "" {
		t.Errorf("alias = %q, want the project value ignored", res.Config.Alias)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "alias") {
		t.Errorf("warnings = %q", res.Warnings)
	}
}

func TestNoWarningsWithoutProjectPersonalKeys(t *testing.T) {
	t.Parallel()
	res, err := LoadLayers(files{
		"/etc/gbai/gbai.yaml":              minimal,
		"/home/o/.config/gbai/config.yaml": "alias: Leo\n",
	}, threeLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if res.Config.Alias != "Leo" || len(res.Warnings) != 0 {
		t.Fatalf("alias = %q, warnings = %q", res.Config.Alias, res.Warnings)
	}
}

func TestPersonalKeyValidation(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{"interaction_level: low", "interaction_level: high", "language: es", "language: pt-BR"} {
		if _, err := Parse([]byte(minimal + ok + "\n")); err != nil {
			t.Errorf("%s: error = %v", ok, err)
		}
	}
	for doc, field := range map[string]string{
		"interaction_level: many": "interaction_level",
		"language: spanish!":      "language",
	} {
		if _, err := Parse([]byte(minimal + doc + "\n")); err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: error = %v, want it to name %s", doc, err, field)
		}
	}
}
