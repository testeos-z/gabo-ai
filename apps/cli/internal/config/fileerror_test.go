package config

import (
	"errors"
	"strings"
	"testing"
)

// gabo check reports config problems per file (T047), so LoadLayers names the
// failing file in a typed error and keeps the message it always had.
func TestLoadLayersReportsAFileError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    files
		wantFile string
		wantLine string
	}{
		{
			name:     "invalid value in the project file",
			files:    files{"/home/o/.config/gabo/config.yaml": minimal, "/src/p/gabo.yaml": "parallelism: 0\n"},
			wantFile: "/src/p/gabo.yaml",
			wantLine: "parallelism",
		},
		{
			name:     "former cloud key in the user file",
			files:    files{"/home/o/.config/gabo/config.yaml": "server:\n  host: x\n", "/src/p/gabo.yaml": minimal},
			wantFile: "/home/o/.config/gabo/config.yaml",
			wantLine: "slots.agent_host",
		},
		{
			name:     "missing key in the merged result",
			files:    files{"/src/p/gabo.yaml": string(without(t, "repo.name"))},
			wantFile: "merged from /src/p/gabo.yaml",
			wantLine: "repo.name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadLayers(tt.files, twoLayers())
			var fe *FileError
			if !errors.As(err, &fe) {
				t.Fatalf("error = %v (%T), want *FileError", err, err)
			}
			if fe.File != tt.wantFile || !strings.Contains(fe.Err.Error(), tt.wantLine) {
				t.Fatalf("FileError = {%q, %v}, want file %q naming %q", fe.File, fe.Err, tt.wantFile, tt.wantLine)
			}
			if want := "config " + tt.wantFile + ": " + fe.Err.Error(); err.Error() != want {
				t.Fatalf("message = %q, want the unchanged %q", err.Error(), want)
			}
		})
	}
}

// A read error is not a config problem: it stays an untyped error.
func TestLoadLayersReadErrorIsNoFileError(t *testing.T) {
	t.Parallel()
	_, err := LoadLayers(failingReader{}, twoLayers())
	var fe *FileError
	if err == nil || errors.As(err, &fe) {
		t.Fatalf("error = %v, want a plain read error", err)
	}
}
