// Package plan scaffolds and checks the six core product docs that the
// planning agents fill and the coder agents follow.
package plan

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Placeholder marks template text that still needs a decision.
const Placeholder = "TODO(gbai)"

// DefaultDir is where the docs live by default.
const DefaultDir = "docs/product"

// Docs are the six core docs in their fill order.
var Docs = []string{"PRD.md", "TRD.md", "IMPLEMENTATION_PLAN.md", "DB_SCHEMA.md", "UX_UI.md", "APP_FLOW.md"}

//go:embed templates/*.md
var templates embed.FS

// State is how far a doc is.
type State string

// Doc states.
const (
	StateMissing  State = "missing"
	StateTemplate State = "template" // still holds placeholders
	StateComplete State = "complete"
)

// DocStatus describes one doc.
type DocStatus struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	State        State  `json:"state"`
	Placeholders int    `json:"placeholders"`
}

// Output is the human-readable report.
type Output struct {
	Lines []string
}

// UseCase scaffolds and inspects the docs.
type UseCase struct {
	fs kernel.FileSystem
}

// NewUseCase wires the use case.
func NewUseCase(fsys kernel.FileSystem) *UseCase { return &UseCase{fs: fsys} }

// New writes every doc from its template; existing docs are kept unless force.
func (u *UseCase) New(_ context.Context, dir string, force bool) (Output, error) {
	var out Output
	for _, name := range Docs {
		target := filepath.Join(dir, name)
		_, err := u.fs.ReadFile(target)
		exists := err == nil
		switch {
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return out, fmt.Errorf("read %s: %w", target, err)
		case exists && !force:
			out.Lines = append(out.Lines, "kept "+target+" (exists; use --force)")
			continue
		}
		body, err := templates.ReadFile("templates/" + name)
		if err != nil {
			return out, err
		}
		if err := u.fs.WriteFile(target, body); err != nil {
			return out, fmt.Errorf("write %s: %w", target, err)
		}
		verb := "created "
		if exists {
			verb = "overwrote "
		}
		out.Lines = append(out.Lines, verb+target)
	}
	return out, nil
}

// Status reports each doc's state in fill order.
func (u *UseCase) Status(_ context.Context, dir string) ([]DocStatus, error) {
	out := make([]DocStatus, 0, len(Docs))
	for _, name := range Docs {
		st := DocStatus{Name: name, Path: filepath.Join(dir, name), State: StateMissing}
		b, err := u.fs.ReadFile(st.Path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, fmt.Errorf("read %s: %w", st.Path, err)
		default:
			st.Placeholders = strings.Count(string(b), Placeholder)
			st.State = StateComplete
			if st.Placeholders > 0 {
				st.State = StateTemplate
			}
		}
		out = append(out, st)
	}
	return out, nil
}
