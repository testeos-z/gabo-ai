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

// A pi reviewer fails in the file that sets it, even when a higher layer
// overrides it, with its key path and its own next step (T072).
func TestLoadLayersRejectsPiReviewerInItsFile(t *testing.T) {
	t.Parallel()
	pi := "runtimes:\n  reviewer: pi\n"
	for name, tt := range map[string]struct {
		files    files
		wantFile string
	}{
		"project file": {files{"/src/p/gabo.yaml": strings.Replace(minimal, "reviewer: claude", "reviewer: pi", 1)}, "/src/p/gabo.yaml"},
		"user file":    {files{"/home/o/.config/gabo/config.yaml": pi, "/src/p/gabo.yaml": minimal}, "/home/o/.config/gabo/config.yaml"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadLayers(tt.files, twoLayers())
			var fe *FileError
			if !errors.As(err, &fe) {
				t.Fatalf("error = %v (%T), want *FileError", err, err)
			}
			msg := fe.Err.Error()
			if fe.File != tt.wantFile || !strings.Contains(msg, `runtimes.reviewer: must be "claude", got "pi"`) || !strings.HasSuffix(msg, NextReviewerClaude) {
				t.Fatalf("FileError = {%q, %q}, want file %q, the key path and %q last", fe.File, msg, tt.wantFile, NextReviewerClaude)
			}
		})
	}
	_, err := LoadFile(files{"/x/gabo.yaml": strings.Replace(minimal, "reviewer: claude", "reviewer: pi", 1)}, "/x/gabo.yaml")
	if err == nil || !strings.Contains(err.Error(), "config /x/gabo.yaml: ") || !strings.Contains(err.Error(), NextReviewerClaude) {
		t.Fatalf("LoadFile() error = %v, want the file and %q", err, NextReviewerClaude)
	}
}
