package config

import (
	"fmt"
	"regexp"
)

// MaxVersion is the newest config schema version this loader reads. A file
// without `version` is version 0, the current shape (DB_SCHEMA migration
// step 1).
const MaxVersion = 1

// PersonalKeys belong to one person, not the project: they resolve from the
// global layers only, and a value in the project file is ignored with a
// warning (DB_SCHEMA key classes, D11). Every other key is a project key and
// keeps the user -> project deep merge.
var PersonalKeys = []string{"alias", "interaction_level", "tone", "language"}

// Interaction levels (R17).
const (
	InteractionLow      = "low"
	InteractionBalanced = "balanced"
	InteractionHigh     = "high"
)

// languagePattern is a loose BCP 47 tag check: a primary subtag plus optional
// alphanumeric subtags.
var languagePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8})*$`)

// dropPersonalKeys deletes every personal key from a project layer and
// returns one warning per key removed, in PersonalKeys order.
func dropPersonalKeys(doc map[string]any, file string) []string {
	var warnings []string
	for _, k := range PersonalKeys {
		if _, ok := doc[k]; !ok {
			continue
		}
		delete(doc, k)
		warnings = append(warnings, fmt.Sprintf(
			"%s: personal key %q is ignored in the project file; set it in the global user config", file, k))
	}
	return warnings
}

func (c Config) validateKeyClasses() []error {
	var errs []error
	if c.Version < 0 || c.Version > MaxVersion {
		errs = append(errs, fmt.Errorf("version: must be between 0 and %d, got %d", MaxVersion, c.Version))
	}
	switch c.InteractionLevel {
	case "", InteractionLow, InteractionBalanced, InteractionHigh:
	default:
		errs = append(errs, fmt.Errorf("interaction_level: must be %q, %q or %q, got %q",
			InteractionLow, InteractionBalanced, InteractionHigh, c.InteractionLevel))
	}
	if c.Language != "" && !languagePattern.MatchString(c.Language) {
		errs = append(errs, fmt.Errorf("language: must be a BCP 47 language tag, got %q", c.Language))
	}
	return errs
}
