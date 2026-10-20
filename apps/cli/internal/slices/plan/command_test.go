package plan

import (
	"bytes"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestCommandNewThenStatus(t *testing.T) {
	t.Parallel()
	uc := NewUseCase(&kerneltest.FS{})
	var out bytes.Buffer
	cmd := NewCommand(uc)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"new", "--dir", "x"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	cmd.SetArgs([]string{"status", "--dir", "x"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "template x/") != 6 {
		t.Fatalf("status:\n%s", out.String())
	}
}
