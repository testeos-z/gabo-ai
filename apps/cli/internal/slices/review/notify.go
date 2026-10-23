package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
)

const maxNotifyLine = 1 << 20

// NotifyInput names the mailbox, the expected review and the review output.
type NotifyInput struct {
	To    string
	PR    int
	SHA   string
	Stdin io.Reader
	Echo  io.Writer // receives every input line, so the pane stays readable
}

// NotifyUseCase relays the review summary line to the coder's mailbox.
type NotifyUseCase struct {
	orca Orca
}

// NewNotifyUseCase wires the notify use case.
func NewNotifyUseCase(orca Orca) *NotifyUseCase { return &NotifyUseCase{orca: orca} }

// Notify sends the summary line for PR/SHA found on Stdin, or verdict=error
// when there is none or more than one. `review run` quotes every diagnostic
// line with EchoPrefix, so its single unquoted summary is the only line that
// can match; a second match means something forged one. Nothing else from
// Stdin is ever sent.
func (u *NotifyUseCase) Notify(ctx context.Context, in NotifyInput) (Summary, error) {
	if err := errors.Join(ValidateMailbox(in.To), ValidatePR(in.PR), ValidateSHA(in.SHA)); err != nil {
		return Summary{}, err
	}
	failed := Summary{PR: in.PR, SHA: in.SHA, Verdict: VerdictError, Merge: MergeNone}
	summary, matches := failed, 0
	sc := bufio.NewScanner(in.Stdin)
	sc.Buffer(make([]byte, 0, 64<<10), maxNotifyLine)
	for sc.Scan() {
		line := sc.Text()
		if _, err := fmt.Fprintln(in.Echo, line); err != nil {
			return Summary{}, fmt.Errorf("echo: %w", err)
		}
		if s, ok := ParseSummary(line); ok && s.PR == in.PR && s.SHA == in.SHA {
			summary, matches = s, matches+1
		}
	}
	if err := sc.Err(); err != nil || matches != 1 {
		summary = failed
	}
	if err := u.orca.Send(ctx, in.To, summary.String()); err != nil {
		return summary, fmt.Errorf("send to %s: %w", in.To, err)
	}
	return summary, nil
}
