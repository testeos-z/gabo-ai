package config

import (
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

// The cloud keys removed by T042 fail as unknown keys naming the key path
// and slots.agent_host, in a single file and in any layer (T006).
func TestFormerCloudKeysFailAsUnknownKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		extra string
		paths []string
	}{
		{"server", "server:\n  host: orca.example.com\n  user: gbai\n", []string{"server.host", "server.user"}},
		{"users", "users:\n  leo:\n    ssh_key: k\n", []string{"users.leo"}},
		{"bot", "bot: true\n", []string{"bot"}},
		{"sessions", "sessions:\n  local_orca:\n    enabled: true\n", []string{"sessions.local_orca"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(minimal + tt.extra))
			if err == nil {
				t.Fatalf("Parse() accepted former cloud key %s", tt.name)
			}
			for _, want := range append([]string{"unknown key", "slots.agent_host"}, tt.paths...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Parse() error = %q, want it to contain %q", err, want)
				}
			}
			_, err = LoadLayers(files{
				"/home/o/.config/gbai/config.yaml": minimal,
				"/src/p/gbai.yaml":                 tt.extra,
			}, twoLayers())
			if err == nil || !strings.Contains(err.Error(), "/src/p/gbai.yaml") || !strings.Contains(err.Error(), tt.paths[0]) ||
				!strings.Contains(err.Error(), "slots.agent_host") {
				t.Errorf("LoadLayers() error = %v, want the project file, %s and slots.agent_host", err, tt.paths[0])
			}
		})
	}
}

// generation was removed by the owner (TRD D24): runtimes come from runtimes.*.
func TestGenerationIsAnUnknownKey(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(minimal + "generation:\n  targets: [claude]\n"))
	if err == nil || !strings.Contains(err.Error(), "generation") {
		t.Fatalf("Parse() error = %v, want generation rejected as an unknown key", err)
	}
}

// The loader reads only the user and project layers: a file at the former
// system path is never read, and loading never writes a config file.
func TestLoaderIgnoresTheSystemFileAndNeverWrites(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{Files: map[string]string{
		"/etc/gbai/gbai.yaml":              "server:\n  host: orca\nparallelism: 9\n",
		"/home/o/.config/gbai/config.yaml": minimal,
		"/src/p/gbai.yaml":                 "parallelism: 4\n",
	}}
	env := map[string]string{"HOME": "/home/o"}
	res, err := LoadLayers(fsys, LayerPaths(func(k string) string { return env[k] }, "/src/p"))
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if res.Config.Parallelism != 4 {
		t.Errorf("parallelism = %d, want the project's 4", res.Config.Parallelism)
	}
	for _, l := range res.Layers {
		if strings.HasPrefix(l.Path, "/etc/") {
			t.Errorf("layer %+v reads a system path", l)
		}
	}
	if len(fsys.Writes) != 0 {
		t.Errorf("loading wrote %q, want no config file rewritten", fsys.Writes)
	}
}
