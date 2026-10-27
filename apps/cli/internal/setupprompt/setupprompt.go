// Package setupprompt holds the setup prompt, the only entry text of gabo-ai
// setup (TRD 9.4, D25, PRD R36). It is embedded in the CLI, printed by
// `gabo init` on a fresh project and published unchanged in
// docs/setup-prompt.md between BeginMarker and EndMarker.
package setupprompt

import (
	"bytes"
	_ "embed"
	"errors"
)

// Text is the setup prompt, byte for byte as in setup-prompt.md.
//
//go:embed setup-prompt.md
var Text string

// Marker comments that delimit the published prompt in docs/setup-prompt.md;
// each sits on its own line.
const (
	BeginMarker = "<!-- gabo-setup-prompt:begin -->"
	EndMarker   = "<!-- gabo-setup-prompt:end -->"
)

// Published returns the bytes between the begin marker line and the end
// marker line of a documentation page. Each marker must appear exactly once,
// on its own line, begin before end.
func Published(page []byte) ([]byte, error) {
	begin := []byte(BeginMarker + "\n")
	end := []byte("\n" + EndMarker + "\n")
	if bytes.Count(page, []byte(BeginMarker)) != 1 || bytes.Count(page, []byte(EndMarker)) != 1 {
		return nil, errors.New("page must hold each setup prompt marker exactly once")
	}
	i := bytes.Index(page, begin)
	if i < 0 || (i > 0 && page[i-1] != '\n') {
		return nil, errors.New("begin marker must sit on its own line")
	}
	start := i + len(begin)
	j := bytes.Index(page[start-1:], end)
	if j < 0 {
		return nil, errors.New("end marker must sit on its own line after the begin marker")
	}
	return page[start : start-1+j+1], nil
}
