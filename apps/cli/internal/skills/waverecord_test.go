package skills

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const waveRecordPath = "gabo-wave-report/scripts/wave-record.sh"

// Intent 0024 S1, PD3: the script ships with the catalog and the Codex plugin
// carries a byte-identical copy.
func TestWaveRecordScriptShips(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var data []byte
	for _, f := range files {
		if f.Path == waveRecordPath {
			data = f.Data
		}
	}
	if data == nil {
		t.Fatalf("%s is not in Files(): add it to the //go:embed lines of skills.go", waveRecordPath)
	}
	cp, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills", waveRecordPath))
	if err != nil {
		t.Fatalf("Codex plugin copy: %v", err)
	}
	if !bytes.Equal(cp, data) {
		t.Fatalf("plugins/codex/skills/%s differs from the catalog; refresh it with cp", waveRecordPath)
	}
}

func TestWaveRecordShellTests(t *testing.T) {
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	for _, tool := range []string{"bash", "sh"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found: %v", tool, err)
		}
	}
	out, err := exec.Command("bash", "gabo-wave-report/scripts/tests/wave-record.test.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("wave-record shell tests: %v\n%s", err, out)
	}
}
