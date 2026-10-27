package config

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Project keys added by DB_SCHEMA migration step 3. They are optional here;
// making the required ones fail when missing is migration step 5 (T005).

// Mode is who resolves questions (R6).
type Mode string

// Modes.
const (
	ModeManual Mode = "manual"
	ModeAuto   Mode = "auto"
)

// Slot names of the TRD 9.1 contracts; the slot name is the key under
// slots.<name>.adapter.
const (
	SlotTasks             = "tasks"
	SlotMemory            = "memory"
	SlotNotifications     = "notifications"
	SlotDirectSessions    = "direct_sessions"
	SlotMethodology       = "methodology"
	SlotRepositoryHost    = "repository_host"
	SlotReviewerPlacement = "reviewer_placement"
	SlotAgentHost         = "agent_host"
)

// SlotNames lists every slot a project may configure, in TRD 9.1 order.
var SlotNames = []string{
	SlotTasks, SlotMemory, SlotNotifications, SlotDirectSessions,
	SlotMethodology, SlotRepositoryHost, SlotReviewerPlacement, SlotAgentHost,
}

// Slot points at the adapter Markdown file filling one slot's contract (D2).
type Slot struct {
	// Adapter is a repository-relative path; for methodology it may also be
	// a skill name.
	Adapter string `yaml:"adapter"`
}

// Reviewer placements (D6). Only PlacementLocal is implemented.
const (
	PlacementLocal         = "local"
	PlacementServer        = "server"
	PlacementGitHubActions = "github-actions"
	PlacementGitLabCI      = "gitlab-ci"
)

// laterPlacements are planned placements not available in this release.
var laterPlacements = []string{PlacementServer, PlacementGitHubActions, PlacementGitLabCI}

// Reviewer configures where the reviewer runs and how it signs its verdict.
type Reviewer struct {
	Placement string  `yaml:"placement,omitempty"`
	Signing   Signing `yaml:"signing,omitempty"`
}

// Signing is how the reviewer marks its verdict comment (PRD 10.1).
type Signing struct {
	Marker string `yaml:"marker,omitempty"`
}

// Plan change scopes (R19, D7), narrowest first.
const (
	ScopeTask   = "task"
	ScopeGroup  = "group"
	ScopeGlobal = "global"
)

// PlanChange bounds which plan changes agents may make (R19, D7).
type PlanChange struct {
	Scope              string `yaml:"scope,omitempty"`
	GlobalWithoutOwner bool   `yaml:"global_without_owner,omitempty"`
}

// Generation selects the runtimes to generate agent files for (D3).
type Generation struct {
	Targets []Runtime `yaml:"targets,omitempty"`
}

var (
	markerPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	emailPattern  = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

func (c Config) validateProjectKeys() []error {
	var errs []error
	fail := func(field, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", field, fmt.Sprintf(format, args...)))
	}

	switch c.Mode {
	case "", ModeManual, ModeAuto:
	default:
		fail("mode", "must be %q or %q, got %q", ModeManual, ModeAuto, c.Mode)
	}

	for _, name := range sortedKeys(c.Slots) {
		if !slices.Contains(SlotNames, name) {
			fail("slots."+name, "unknown slot; must be one of %s", strings.Join(SlotNames, ", "))
			continue
		}
		if !relativePath(c.Slots[name].Adapter) {
			fail("slots."+name+".adapter", "must be a repository-relative path, got %q", c.Slots[name].Adapter)
		}
	}

	switch p := c.Reviewer.Placement; {
	case p == "" || p == PlacementLocal:
	case slices.Contains(laterPlacements, p):
		fail("reviewer.placement", "%q is planned for a later release; only %q is available", p, PlacementLocal)
	default:
		fail("reviewer.placement", "must be %q (later: %s), got %q", PlacementLocal, strings.Join(laterPlacements, ", "), p)
	}
	if m := c.Reviewer.Signing.Marker; m != "" && !markerPattern.MatchString(m) {
		fail("reviewer.signing.marker", "must use letters, digits, '.', '_' or '-', got %q", m)
	}

	for _, email := range sortedKeys(c.Developers) {
		if !emailPattern.MatchString(email) {
			fail("developers", "key %q must be a git email", email)
		}
		if strings.TrimSpace(c.Developers[email]) == "" {
			fail("developers."+email, "alias must not be blank")
		}
	}

	switch c.PlanChange.Scope {
	case "", ScopeTask, ScopeGroup, ScopeGlobal:
	default:
		fail("plan_change.scope", "must be %q, %q or %q, got %q", ScopeTask, ScopeGroup, ScopeGlobal, c.PlanChange.Scope)
	}

	for i, t := range c.Generation.Targets {
		if t != RuntimeClaude && t != RuntimePi {
			fail("generation.targets", "must list %q or %q, got %q", RuntimeClaude, RuntimePi, t)
		} else if slices.Contains(c.Generation.Targets[:i], t) {
			fail("generation.targets", "duplicate target %q", t)
		}
	}
	return errs
}

// relativePath reports whether p is a non-empty repository-relative path
// that does not escape the repository.
func relativePath(p string) bool {
	return strings.TrimSpace(p) != "" && !strings.HasPrefix(p, "/") &&
		!slices.Contains(strings.Split(p, "/"), "..")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
