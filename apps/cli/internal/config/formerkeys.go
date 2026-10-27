package config

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// FormerCloudKeys are the top-level keys of the retired cloud config
// (removed from Config by T042). A file still carrying them fails as unknown
// keys that point to slots.agent_host; gbai never rewrites or migrates it.
var FormerCloudKeys = []string{"server", "users", "bot", "sessions"}

// formerCloudKeyErrors names every former cloud key path in the document
// (top key, or top.child for a mapping). A document that is not a mapping
// returns nil and is left to the regular decoder.
func formerCloudKeyErrors(b []byte) error {
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil
	}
	var errs []error
	for _, top := range FormerCloudKeys {
		v, ok := doc[top]
		if !ok {
			continue
		}
		paths := []string{top}
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			paths = paths[:0]
			for _, k := range sortedKeys(m) {
				paths = append(paths, top+"."+k)
			}
		}
		for _, p := range paths {
			errs = append(errs, fmt.Errorf("%s: unknown key (the cloud config was removed; agent hosts are set with slots.agent_host)", p))
		}
	}
	return errors.Join(errs...)
}
