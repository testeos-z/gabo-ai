// Package adapters holds the driven adapters of the doctor slice.
package adapters

import (
	"os/exec"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/doctor"
)

// Path implements doctor.Local with the process PATH.
type Path struct{}

var _ doctor.Local = Path{}

// LookPath resolves name in PATH.
func (Path) LookPath(name string) (string, error) { return exec.LookPath(name) }
