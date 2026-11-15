package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// ErrPreWriteUsage marks a usage error of the pre-write hook mode.
var ErrPreWriteUsage = errors.New("usage: gabo check --pre-write <path>... (or no path and the hook's tool-call JSON on stdin)")

// PreWrite is the hook mode (intent 0001 S11): it judges pending writes to
// paths with the intent stage gate before they happen, and writes nothing.
// Each path is relative to the working directory or absolute inside the
// project root; a path outside the root (its root prefix compared without
// case only on a case-insensitive filesystem, Settings.FoldCase), or one
// that is not docs/product/intents/<dir>/plan.md (compared without case),
// passes. A path is judged as written and at its symlink-resolved target,
// each ".." applied after the symlinks before it (a dangling link at the
// target it would create), and blocked when
// either is a plan.md whose folder's spec.md is missing or not accepted. A
// spec.md that cannot be read, or a path whose symlinks cannot be resolved,
// counts as not accepted (fail closed). The report holds one
// "✗ <path>: <reason>" line per blocked path, in argument order.
func (u *UseCase) PreWrite(paths []string) (Report, error) {
	var r Report
	if err := checkPreWritePaths(paths); err != nil {
		return r, err
	}
	for _, p := range paths {
		if filepath.IsAbs(p) && u.s.Root == "" {
			return Report{}, fmt.Errorf("%w: absolute path %s with no project root", ErrPreWriteUsage, p)
		}
		plans, err := u.plansOf(p)
		if err != nil {
			shown := p
			if rel, ok := relInside(u.s.Root, u.absOf(p), u.s.FoldCase); ok {
				shown = rel
			}
			r.fail("✗ " + shown + ": its symlinks cannot be resolved (" + err.Error() + "); it counts as not accepted")
			continue
		}
		for _, plan := range plans {
			reason, err := u.specGate(path.Dir(plan))
			if err != nil {
				reason = "spec.md cannot be read (" + err.Error() + "); it counts as not accepted"
			}
			if reason != "" {
				r.fail("✗ " + plan + ": " + reason)
				break
			}
		}
	}
	return r, nil
}

// checkPreWritePaths rejects an empty path list or an empty path.
func checkPreWritePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("%w: no path given", ErrPreWriteUsage)
	}
	for _, p := range paths {
		if p == "" {
			return fmt.Errorf("%w: empty path", ErrPreWriteUsage)
		}
	}
	return nil
}

// maxLinks bounds how many dangling symlinks one path may follow, like the
// kernel's limit on a symlink chain.
const maxLinks = 40

// absOf returns p made absolute against the working directory (Settings
// WorkDir, else Root) and cleaned lexically; with neither it returns p
// unchanged. It is the path as written: plansOf resolves the raw path.
func (u *UseCase) absOf(p string) string {
	return filepath.Clean(u.rawAbsOf(p))
}

// rawAbsOf returns p made absolute like absOf but not cleaned, so its ".."
// components can be applied after the symlinks before them.
func (u *UseCase) rawAbsOf(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	base := u.s.WorkDir
	if base == "" {
		base = u.s.Root
	}
	if base == "" {
		return p
	}
	return base + string(filepath.Separator) + p
}

// plansOf returns the project-relative plan.md files a write to p would
// create or change: p as written and p with its symlinks resolved. An error
// means the symlinks could not be resolved.
func (u *UseCase) plansOf(p string) ([]string, error) {
	targets, err := u.targetsOf(p)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(targets, func(rel string) bool { return !isPlan(rel) }), nil
}

// targetsOf returns the project-relative paths a write to p would create or
// change: p as written and p with its symlinks resolved, each once and only
// when inside the project root. An error means the symlinks could not be
// resolved.
func (u *UseCase) targetsOf(p string) ([]string, error) {
	var targets []string
	if rel, ok := relInside(u.s.Root, u.absOf(p), u.s.FoldCase); ok {
		targets = append(targets, rel)
	}
	if u.s.Resolve == nil || u.s.Root == "" {
		return targets, nil
	}
	root, err := u.s.Resolve(u.s.Root)
	if err != nil {
		return nil, err
	}
	target, ok, err := u.resolveRaw(u.rawAbsOf(p))
	if err != nil {
		return nil, err
	}
	if !ok {
		return targets, nil
	}
	if rel, ok := relInside(root, target, u.s.FoldCase); ok && !slices.Contains(targets, rel) {
		targets = append(targets, rel)
	}
	return targets, nil
}

// resolveRaw resolves the absolute, uncleaned path abs component by
// component as the kernel does: each existing component has its symlinks
// resolved before the next ".." applies, so lnk/../x climbs from lnk's
// target. A dangling symlink is followed to its target, since a write
// through it creates that target; a missing component is kept as written
// (a plan.md being created does not exist yet). It reports false when no
// component exists, and any error but a missing entry, including too many
// links.
func (u *UseCase) resolveRaw(abs string) (string, bool, error) {
	vol := filepath.VolumeName(abs)
	cur := vol + string(filepath.Separator)
	comps := splitPath(abs[len(vol):])
	found := false
	for links := 0; len(comps) > 0; {
		c := comps[0]
		comps = comps[1:]
		switch c {
		case ".":
			continue
		case "..":
			// cur holds no symlink, so its parent is lexical.
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, c)
		real, err := u.s.Resolve(next)
		if err == nil {
			cur, found = real, true
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false, err
		}
		target, isLink, err := u.readlink(next)
		if err != nil {
			return "", false, err
		}
		if !isLink {
			cur = next
			continue
		}
		if links++; links > maxLinks {
			return "", false, fmt.Errorf("%s: too many links", abs)
		}
		if tv := filepath.VolumeName(target); filepath.IsAbs(target) {
			cur = tv + string(filepath.Separator)
			target = target[len(tv):]
		}
		comps = append(splitPath(target), comps...)
	}
	return cur, found, nil
}

// splitPath returns the non-empty components of p, keeping "." and "..".
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == filepath.Separator })
}

// readlink reports whether p is a symlink and its target; a missing p is no
// link. Without Settings.Readlink nothing is a link.
func (u *UseCase) readlink(p string) (string, bool, error) {
	if u.s.Readlink == nil {
		return "", false, nil
	}
	target, isLink, err := u.s.Readlink(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	return target, isLink, err
}

// relInside returns p as a clean slash-separated path relative to root,
// and whether it lies inside root. With fold (a case-insensitive
// filesystem) the root prefix is compared without case. A relative p is
// taken as relative to root already; callers pass absolute paths
// (UseCase.absOf).
func relInside(root, p string, fold bool) (string, bool) {
	rel := filepath.Clean(p)
	if filepath.IsAbs(rel) {
		root = filepath.Clean(root)
		abs := rel
		var err error
		if rel, err = filepath.Rel(root, abs); err != nil {
			return "", false
		}
		prefix := strings.TrimSuffix(root, string(filepath.Separator)) + string(filepath.Separator)
		if fold && len(abs) > len(prefix) && strings.EqualFold(abs[:len(prefix)], prefix) {
			rel = abs[len(prefix):]
		}
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// isPlan reports whether rel is docs/product/intents/<dir>/plan.md,
// without case.
func isPlan(rel string) bool {
	return strings.EqualFold(path.Base(rel), "plan.md") && strings.EqualFold(path.Dir(path.Dir(rel)), IntentsDir)
}

// humanOnlyRoute is the route a coder reads for a write to a human-only path
// (intent 0028 S7, S11).
const humanOnlyRoute = "a human edits this path"

// WriteVerdict is what the file-tool hook mode prints and exits with: Exit
// is 0 (the write runs), ExitFailed (blocked) or ExitAsk (the engine asks),
// and Out is the text to print, empty when nothing is printed.
type WriteVerdict struct {
	Exit int
	Out  string
}

// WriteCall is the part of a file-tool call the human-only lock judges.
type WriteCall struct {
	// AgentID is the subagent's id of the call, "" for the main session.
	AgentID string
	// Engine is the engine the call came from, "" when the hook does not name it.
	Engine string
	// CanAsk reports whether the engine can show its yes/no prompt for an ask.
	CanAsk bool
	// Gated reports that the stage gate already blocks the call, so no prompt
	// is shown for it whatever the lock says.
	Gated bool
}

// writeSession reports whether a file-tool call belongs to a coder session
// (intent 0028 S7, decision 13). A call with an agent id is a subagent's. A
// call without one whose engine is not named is the main session: no hook
// names its engine yet, and the human's own edit of gabo.yaml must pass. A
// named engine is judged as CoderSession judges it: claude and codex give an
// id to subagents only, so their id-less calls are the main session, and every
// other engine (pi, opencode) is a coder's.
func writeSession(agentID, engine string) bool {
	if agentID != "" {
		return true
	}
	if engine == "" {
		return false
	}
	return CoderSession(agentID, engine)
}

// JudgeWrites judges the paths one file-tool call writes under the
// human_only_paths lock (intent 0028 S7, S10 to S12, S16). Only a coder session
// is judged (writeSession). A path whose target is gabo.yaml or lies under
// review.human_only_paths is a violation: its state gives Decide's outcome, one
// record is written per violating path, and each violation prints one Explain
// block. An ask counts as blocked where no prompt can be shown (CanAsk false)
// or the stage gate blocks the call anyway (Gated). A config that cannot be read
// fails the coder's writes closed. It writes no project file. The caller runs
// PreWrite (the stage gate) on the same paths, which also blocks a path whose
// symlinks cannot be resolved, so such a path is not judged here. The test_files
// lock judges the same paths after this one (judgeTestFiles).
func (u *UseCase) JudgeWrites(paths []string, call WriteCall) WriteVerdict {
	if !writeSession(call.AgentID, call.Engine) {
		return WriteVerdict{}
	}
	cfg, err := u.lockConfig()
	if err != nil {
		return WriteVerdict{Exit: ExitFailed, Out: Explain(config.LockHumanOnlyPaths, "the project config cannot be read ("+err.Error()+")", "fix the project, its gabo.yaml or the working directory, then retry", Decision{Exit: ExitFailed, Outcome: OutcomeBlocked}) + "\n"}
	}
	var b strings.Builder
	exit := 0
	for _, p := range paths {
		target, ok := u.humanOnlyTarget(p, cfg.Review.HumanOnlyPaths)
		if !ok {
			continue
		}
		rule := "a coder write to the human-only path " + target + " is blocked"
		d := decideFor(cfg, config.LockHumanOnlyPaths, call)
		if err := u.RecordDecision(config.LockHumanOnlyPaths, rule, d); err != nil {
			return WriteVerdict{Exit: ExitFailed, Out: Explain(config.LockHumanOnlyPaths, rule, "fix the outcome file named by GABO_OUTCOME_FILE, then retry", Decision{}) + "\n"}
		}
		if d.Exit == 0 {
			continue
		}
		b.WriteString(Explain(config.LockHumanOnlyPaths, rule, humanOnlyRoute, d) + "\n")
		// A block outranks an ask: the strictest outcome of the call wins.
		if exit == 0 || d.Exit == ExitFailed {
			exit = d.Exit
		}
	}
	return WriteVerdict{Exit: exit, Out: b.String() + u.judgeTestFiles(paths, cfg, call)}
}

// decideFor judges one violating write under lock at its state in cfg, with the
// engine's prompt limits applied: an ask the engine cannot show, or one the stage
// gate blocks anyway (call.Gated), counts as blocked.
func decideFor(cfg config.Config, lock config.LockName, call WriteCall) Decision {
	d := Decide(cfg.Lock(lock), cfg.Mode)
	switch {
	case d.Exit == ExitAsk && !call.CanAsk:
		d = Decision{Exit: ExitFailed, Outcome: OutcomeBlocked, Note: "the engine cannot show its prompt here; it counts as blocked"}
	case d.Exit == ExitAsk && call.Gated:
		d = Decision{Exit: ExitFailed, Outcome: OutcomeBlocked, Note: "the stage gate blocks this write too; no prompt is shown"}
	}
	return d
}

// lockConfig reads gabo.yaml's layers for the write locks: no config file
// reads every key as its default (intent 0028 PD2); any other error is
// returned, and the caller fails closed.
func (u *UseCase) lockConfig() (config.Config, error) {
	if u.config == nil {
		return config.Config{}, nil
	}
	res, err := config.LoadLayers(u.config, u.s.Layers)
	switch {
	case err == nil:
		return res.Config, nil
	case errors.Is(err, config.ErrNoConfig):
		return config.Config{}, nil
	default:
		return config.Config{}, err
	}
}

// humanOnlyTarget returns the first project-relative target of a write to p
// that is human-only, as written or with its symlinks resolved.
func (u *UseCase) humanOnlyTarget(p string, entries []string) (string, bool) {
	targets, err := u.targetsOf(p)
	if err != nil {
		return "", false
	}
	for _, t := range targets {
		if humanOnlyPath(t, entries, u.s.FoldCase) {
			return t, true
		}
	}
	return "", false
}

// humanOnlyPath reports whether rel, a project-relative target, is human-only:
// gabo.yaml, or a path under an entry of review.human_only_paths. The match is
// the reviewer's rule (is_human_only in the gabo-reviewer scripts): a
// directory entry (trailing /) matches by prefix, a file entry by its exact
// path or as a trailing path element. With fold the comparison ignores case.
func humanOnlyPath(rel string, entries []string, fold bool) bool {
	norm := func(s string) string {
		if fold {
			return strings.ToLower(s)
		}
		return s
	}
	r := norm(rel)
	if r == norm("gabo.yaml") {
		return true
	}
	for _, e := range entries {
		e = norm(e)
		if strings.HasSuffix(e, "/") {
			if strings.HasPrefix(r, e) {
				return true
			}
		} else if r == e || strings.HasSuffix(r, "/"+e) {
			return true
		}
	}
	return false
}
