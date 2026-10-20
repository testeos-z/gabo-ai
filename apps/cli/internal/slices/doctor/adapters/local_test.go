package adapters

import "testing"

func TestPathLookup(t *testing.T) {
	t.Parallel()
	if _, err := (Path{}).LookPath("sh"); err != nil {
		t.Fatalf("LookPath(sh) error = %v", err)
	}
	if _, err := (Path{}).LookPath("gbai-definitely-missing-tool"); err == nil {
		t.Fatal("LookPath(missing) error = nil")
	}
}
