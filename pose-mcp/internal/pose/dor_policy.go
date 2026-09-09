package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DoRPolicy is `.pose/policy/dor.json`
// (spec pose-changelog-and-dor-policy-types).
//
// It was read through two anonymous structs in two commands, neither of which
// described the file: `readiness.go` looked for `adopted_at`, which the shipped
// policy does not contain, and `check.go` for `defaultTaskType` and
// `taskTypes`, which it does. So the gate reads as unadopted on a fresh
// install, and nothing said whether that was the intent or an omission.
//
// It is the intent — the Definition of Ready applies to specs created after the
// date an instance adopts it, and a project that never sets one is not held to
// it retroactively. The shipped policy now says so with an explicit empty value
// instead of leaving the key out.
type DoRPolicy struct {
	SchemaVersion   int                 `json:"schemaVersion"`
	AdoptedAt       string              `json:"adopted_at"`
	DefaultTaskType string              `json:"defaultTaskType"`
	TaskTypes       map[string][]string `json:"taskTypes"`
}

// LoadDoRPolicy reads the policy. An absent file is an unadopted gate, which is
// the same answer as an empty adoption date.
func LoadDoRPolicy(root string) DoRPolicy {
	var policy DoRPolicy
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "dor.json"))
	if err != nil {
		return policy
	}
	_ = json.Unmarshal(raw, &policy)
	return policy
}

// AppliesTo reports whether a spec created on the given date is held to the
// Definition of Ready. An unset adoption date holds nothing.
func (p DoRPolicy) AppliesTo(createdAt string) bool {
	return p.AdoptedAt != "" && createdAt != "" && createdAt >= p.AdoptedAt
}
