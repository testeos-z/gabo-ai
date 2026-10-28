package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	t.Parallel()
	info := func(v string, ok bool) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: v}}, ok }
	}
	tests := []struct {
		name, build string
		read        func() (*debug.BuildInfo, bool)
		want        string
	}{
		{"ldflags wins", "1.2.3", info("v9.9.9", true), "1.2.3"},
		{"go install module version", "", info("v1.4.0", true), "v1.4.0"},
		{"source build without vcs", "", info("(devel)", true), ""},
		{"source build with vcs", "", info("v1.4.1-0.20261027120000-abc123def456+dirty", true), "v1.4.1-0.20261027120000-abc123def456+dirty"},
		{"no build info", "", info("", false), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveVersion(tt.build, tt.read); got != tt.want {
				t.Fatalf("resolveVersion = %q, want %q", got, tt.want)
			}
		})
	}
}

// A binary from `go install …@v<w>` (no ldflags) can run the init it advises.
func TestInitAcceptsGoInstallVersion(t *testing.T) {
	out, err := runInit(t, t.TempDir(), resolveVersion("", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v1.4.0"}}, true
	}))
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
}
