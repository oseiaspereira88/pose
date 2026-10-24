package pose

import (
	"fmt"
	"sort"
)

// reviewScopeKindSupported is the closed security-selector taxonomy. Adding a
// kind requires an explicit versioned contract, including ParseScopeRef grammar.
func reviewScopeKindSupported(kind string) bool {
	switch kind {
	case "spec", "milestone", "roadmap":
		return true
	default:
		return false
	}
}

// reviewPolicyScopeError distinguishes an invalid selector from an unavailable
// historical baseline. It must never downgrade protection to a warning.
type reviewPolicyScopeError struct{ Field, Kind string }

func (e *reviewPolicyScopeError) Error() string {
	return fmt.Sprintf("pose: invalid review policy %s[%q]: unsupported scope kind (supported: spec, milestone, roadmap)", e.Field, e.Kind)
}

func validateReviewPolicyScopes(field string, values map[string]string) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, kind := range keys {
		if !reviewScopeKindSupported(kind) {
			return &reviewPolicyScopeError{Field: field, Kind: kind}
		}
	}
	return nil
}
