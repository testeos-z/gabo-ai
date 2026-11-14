package check

import "io/fs"

// UserFS reads user-level paths outside the project, by absolute OS path,
// without writing. Each error for a missing entry matches fs.ErrNotExist.
type UserFS interface {
	// Lstat returns the mode of name without following a final symlink.
	Lstat(name string) (fs.FileMode, error)
	// Stat returns the mode of name, following every symlink.
	Stat(name string) (fs.FileMode, error)
	// ReadLink returns the raw target of the symlink name.
	ReadLink(name string) (string, error)
}

// OutcomeAppender appends one outcome record line to the file the cycle names
// for the run (intent 0028 S16). The file is created when it is missing and
// is only appended to: a record is never rewritten.
type OutcomeAppender interface {
	Append(name string, line []byte) error
}

// PRBaseReader reads the base branch of a pull request on the repository
// host (intent 0028 D4). repo is the -R value of the command ("" for the
// current repository) and ref the pull request as the command names it: a
// number, a URL or a branch ("" is the current branch's pull request).
type PRBaseReader interface {
	PRBase(repo, ref string) (string, error)
}
