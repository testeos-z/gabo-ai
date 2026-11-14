package check

// ShellVerdict is what the shell hook mode prints and exits with. Stub: RED.
type ShellVerdict struct {
	// Exit is 0 (the call runs), ExitFailed (blocked) or ExitAsk.
	Exit int
	// Out is the text to print, empty when nothing is printed.
	Out string
}

// JudgeShell judges one shell call under the shell_guard lock. Stub: RED.
func (u *UseCase) JudgeShell(call ShellCall) (ShellVerdict, error) {
	return ShellVerdict{}, nil
}
