package config

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestMerge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		lower, higher map[string]any
		want          map[string]any
	}{
		{
			name:   "higher scalar overrides",
			lower:  map[string]any{"parallelism": 3},
			higher: map[string]any{"parallelism": 5},
			want:   map[string]any{"parallelism": 5},
		},
		{
			name:   "higher list replaces the whole list",
			lower:  map[string]any{"review": map[string]any{"checks": []any{"a", "b"}}},
			higher: map[string]any{"review": map[string]any{"checks": []any{"c"}}},
			want:   map[string]any{"review": map[string]any{"checks": []any{"c"}}},
		},
		{
			name:   "key absent in higher keeps the lower value",
			lower:  map[string]any{"repo": map[string]any{"name": "o/r"}, "parallelism": 2},
			higher: map[string]any{"parallelism": 4},
			want:   map[string]any{"repo": map[string]any{"name": "o/r"}, "parallelism": 4},
		},
		{
			name: "nested mappings merge key by key",
			lower: map[string]any{"repo": map[string]any{
				"name": "o/r", "integration_branch": "dev"}},
			higher: map[string]any{"repo": map[string]any{"integration_branch": "main"}},
			want: map[string]any{"repo": map[string]any{
				"name": "o/r", "integration_branch": "main"}},
		},
		{
			name:   "a scalar in higher replaces a lower mapping",
			lower:  map[string]any{"coder": map[string]any{"login": "x"}},
			higher: map[string]any{"coder": nil},
			want:   map[string]any{"coder": nil},
		},
		{
			name:   "nil lower",
			higher: map[string]any{"parallelism": 1},
			want:   map[string]any{"parallelism": 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Merge(tt.lower, tt.higher); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Merge() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMergeDoesNotMutateItsInputs(t *testing.T) {
	t.Parallel()
	lower := map[string]any{"review": map[string]any{"auto_merge": false}}
	Merge(lower, map[string]any{"review": map[string]any{"auto_merge": true}})
	if lower["review"].(map[string]any)["auto_merge"] != false {
		t.Fatalf("lower mutated: %#v", lower)
	}
}

// files is an in-memory Reader; a missing name reports fs.ErrNotExist.
type files map[string]string

func (f files) ReadFile(name string) ([]byte, error) {
	b, ok := f[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return []byte(b), nil
}

func twoLayers() []Layer {
	return []Layer{
		{Name: LayerUser, Path: "/home/o/.config/gbai/config.yaml"},
		{Name: LayerProject, Path: "/src/p/gbai.yaml"},
	}
}

func TestLoadLayersHigherLayersWin(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gbai/config.yaml": with("parallelism: 4\ncoder:\n  login: helper\nreview:\n  checks: [make test]\n  auto_merge: true\n"),
		"/src/p/gbai.yaml":                 "parallelism: 6\nreview:\n  checks: [go test, go vet]\n",
	}
	res, err := LoadLayers(r, twoLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	c := res.Config
	if c.Parallelism != 6 {
		t.Errorf("parallelism = %d, want project 6", c.Parallelism)
	}
	if c.Coder.Login != "helper" {
		t.Errorf("coder.login = %q, want user layer value", c.Coder.Login)
	}
	if !reflect.DeepEqual(c.Review.Checks, []string{"go test", "go vet"}) {
		t.Errorf("checks = %q, want the project list only", c.Review.Checks)
	}
	if !c.Review.AutoMerge {
		t.Errorf("auto_merge lost: a key absent in higher layers must keep the user value")
	}
	if c.Repo.Name != "cappy-lab/vulcam" || c.Repo.IntegrationBranch != "dev" {
		t.Errorf("user values lost: %+v", c.Repo)
	}
	for i, l := range res.Layers {
		if !l.Found || l.Name != twoLayers()[i].Name {
			t.Errorf("layer %d = %+v", i, l)
		}
	}
}

func TestLoadLayersSkipsMissingFiles(t *testing.T) {
	t.Parallel()
	res, err := LoadLayers(files{"/src/p/gbai.yaml": minimal}, twoLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	got := []bool{res.Layers[0].Found, res.Layers[1].Found}
	if !reflect.DeepEqual(got, []bool{false, true}) {
		t.Fatalf("found = %v", got)
	}
	if res.Config.Parallelism != 3 {
		t.Fatalf("parallelism = %d, want the project file's 3", res.Config.Parallelism)
	}
}

func TestLoadLayersEmptyPathAndEmptyFile(t *testing.T) {
	t.Parallel()
	layers := twoLayers()
	layers[0].Path = ""
	res, err := LoadLayers(files{"/src/p/gbai.yaml": minimal}, layers)
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if res.Layers[0].Found || !res.Layers[1].Found {
		t.Fatalf("layers = %+v", res.Layers)
	}
	res, err = LoadLayers(files{"/home/o/.config/gbai/config.yaml": minimal, "/src/p/gbai.yaml": "# nothing yet\n"}, twoLayers())
	if err != nil {
		t.Fatalf("LoadLayers() with an empty project file error = %v", err)
	}
	if !res.Layers[0].Found || !res.Layers[1].Found {
		t.Fatalf("layers = %+v", res.Layers)
	}
}

func TestLoadLayersUnknownKeyNamesTheFile(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gbai/config.yaml": minimal,
		"/src/p/gbai.yaml":                 "parallelizm: 4\n",
	}
	res, err := LoadLayers(r, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "/src/p/gbai.yaml") || !strings.Contains(err.Error(), "parallelizm") {
		t.Fatalf("error = %v, want unknown key naming the project file", err)
	}
	if len(res.Layers) != 2 {
		t.Fatalf("layers not reported on error: %+v", res.Layers)
	}
}

// Required keys are checked only on the merged result; values are also
// checked per layer (see TestLoadLayersValidatesEveryLayerAgainstItsSchema).
func TestLoadLayersRequiresKeysOnlyOnTheMergedResult(t *testing.T) {
	t.Parallel()
	// Neither file is complete alone; merged they are.
	r := files{
		"/home/o/.config/gbai/config.yaml": string(without(t, "review.checks")),
		"/src/p/gbai.yaml":                 "review:\n  checks: [go test ./...]\n",
	}
	if _, err := LoadLayers(r, twoLayers()); err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if _, err := LoadLayers(files{"/src/p/gbai.yaml": "review:\n  checks: [go test ./...]\n"}, twoLayers()); err == nil ||
		!strings.Contains(err.Error(), "repo.name") {
		t.Fatalf("merged validation error = %v", err)
	}
}

func TestLoadLayersNoFileFound(t *testing.T) {
	t.Parallel()
	_, err := LoadLayers(files{}, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "no config file found") || !strings.Contains(err.Error(), "--config") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadLayersReadErrorIsNotSkipped(t *testing.T) {
	t.Parallel()
	_, err := LoadLayers(failingReader{}, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "/home/o/.config/gbai/config.yaml") {
		t.Fatalf("error = %v", err)
	}
}

type failingReader struct{}

func (failingReader) ReadFile(string) ([]byte, error) { return nil, errors.New("permission denied") }

func TestLoadFileIsASingleLayer(t *testing.T) {
	t.Parallel()
	res, err := LoadFile(files{"/x/custom.yaml": minimal}, "/x/custom.yaml")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if len(res.Layers) != 1 || res.Layers[0] != (Layer{Name: LayerFlag, Path: "/x/custom.yaml", Found: true}) {
		t.Fatalf("layers = %+v", res.Layers)
	}
	if _, err := LoadFile(files{}, "/x/missing.yaml"); err == nil || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing file error = %v", err)
	}
}

func TestLayerPathsOrder(t *testing.T) {
	t.Parallel()
	env := map[string]string{"HOME": "/home/o"}
	got := LayerPaths(func(k string) string { return env[k] }, "/src/p")
	want := []Layer{
		{Name: LayerUser, Path: "/home/o/.config/gbai/config.yaml"},
		{Name: LayerProject, Path: "/src/p/gbai.yaml"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LayerPaths() = %+v, want %+v", got, want)
	}
}

func TestUserPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"xdg wins", map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/home/o"}, "/x/gbai/config.yaml"},
		{"relative xdg is ignored", map[string]string{"XDG_CONFIG_HOME": "rel", "HOME": "/home/o"}, "/home/o/.config/gbai/config.yaml"},
		{"home fallback", map[string]string{"HOME": "/home/o"}, "/home/o/.config/gbai/config.yaml"},
		{"nothing", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := UserPath(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Fatalf("UserPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		runner kernel.Runner
		want   string
	}{
		{"git top-level", &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("/src/p\n")}}, "/src/p"},
		{"not a repository", &kerneltest.Runner{Result: kernel.Result{ExitCode: 128, Stderr: []byte("fatal: not a git repository")}}, "/src/p/sub"},
		{"git missing", &kerneltest.Runner{Err: errors.New("exec: git not found")}, "/src/p/sub"},
		{"empty output", &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("\n")}}, "/src/p/sub"},
		{"relative output", &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("src\n")}}, "/src/p/sub"},
		{"no runner", nil, "/src/p/sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ProjectRoot(context.Background(), tt.runner, "/src/p/sub"); got != tt.want {
				t.Fatalf("ProjectRoot() = %q, want %q", got, tt.want)
			}
		})
	}
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("/src/p\n")}}
	ProjectRoot(context.Background(), r, "/src/p/sub")
	if c := r.Calls[0].Command; c.Name != "git" || strings.Join(c.Args, " ") != "rev-parse --show-toplevel" || c.Dir != "/src/p/sub" {
		t.Fatalf("command = %+v", c)
	}
}

func TestLoadLayersRejectsUnsupportedVersionInAnyLayer(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gbai/config.yaml": "version: 99\n",
		"/src/p/gbai.yaml":                 "version: 1\n",
	}
	_, err := LoadLayers(r, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "/home/o/.config/gbai/config.yaml") || !strings.Contains(err.Error(), "version") {
		t.Fatalf("error = %v, want unsupported version naming the user file", err)
	}
}

func TestLoadLayersKeepsVersionOutOfTheMergedConfig(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gbai/config.yaml": minimal + "version: 1\n",
		"/src/p/gbai.yaml":                 "parallelism: 4\n",
	}
	res, err := LoadLayers(r, twoLayers())
	if err != nil {
		t.Fatalf("LoadLayers() error = %v", err)
	}
	if res.Config.Version != 0 {
		t.Fatalf("merged version = %d, want 0: version is per file", res.Config.Version)
	}
	if res.Layers[0].Version != 1 || res.Layers[1].Version != 0 {
		t.Fatalf("layer versions = %d, %d", res.Layers[0].Version, res.Layers[1].Version)
	}
}
