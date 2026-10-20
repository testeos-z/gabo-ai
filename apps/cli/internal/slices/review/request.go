package review

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// RequestInput identifies the pinned head to review and the coder's mailbox.
type RequestInput struct {
	PR     int
	SHA    string
	Notify string
}

// RequestOutput describes the terminal that runs the review.
type RequestOutput struct {
	Title    string
	Terminal string // raw orca output
}

// RequestUseCase opens a detached Orca terminal that runs the review through
// sudo and pipes its summary to `review notify`. It never waits for the result.
type RequestUseCase struct {
	orca     Orca
	reviewer string
}

// NewRequestUseCase wires the request use case; reviewer is the OS user the
// sudo rule allows.
func NewRequestUseCase(orca Orca, reviewer string) *RequestUseCase {
	return &RequestUseCase{orca: orca, reviewer: reviewer}
}

// Request validates the input and launches the review terminal.
func (u *RequestUseCase) Request(ctx context.Context, in RequestInput) (RequestOutput, error) {
	if err := errors.Join(ValidatePR(in.PR), ValidateSHA(in.SHA), ValidateMailbox(in.Notify)); err != nil {
		return RequestOutput{}, err
	}
	title := fmt.Sprintf("gbai-review-%d-%s", in.PR, in.SHA[:7])
	command := fmt.Sprintf("sudo -n -u %s %s head %d %s | %s review notify --to %s --pr %d --sha %s",
		shellQuote(u.reviewer), kernel.ReviewRunPath, in.PR, in.SHA,
		kernel.ServerBinaryPath, shellQuote(in.Notify), in.PR, in.SHA)
	out, err := u.orca.CreateTerminal(ctx, title, command)
	if err != nil {
		return RequestOutput{}, fmt.Errorf("open review terminal: %w", err)
	}
	return RequestOutput{Title: title, Terminal: out}, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
