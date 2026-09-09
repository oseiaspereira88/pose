package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChangelogPolicy is `.pose/policy/changelog.json`
// (spec pose-changelog-and-dor-policy-types).
//
// `categories` shipped in that file from the start and nothing read it. The set
// of valid fragment categories was written out three times instead — in the
// fragment loader, in `pose check`, and implicitly in the notes renderer's
// order — so the file promised a setting it did not have and the three copies
// could disagree.
type ChangelogPolicy struct {
	SchemaVersion int      `json:"schema_version"`
	AdoptedAt     string   `json:"adopted_at"`
	Categories    []string `json:"categories,omitempty"`
}

// DefaultChangelogCategories is what applies when the policy is absent or
// declares none. It is also what the shipped policy contains, so an instance
// that never touches the file behaves exactly as before.
func DefaultChangelogCategories() []string {
	return []string{"added", "changed", "fixed", "removed", "deprecated", "security"}
}

// changelogRenderOrder is presentation, not configuration: the categories a
// reader most needs to see come first, which is not the order anyone would
// write them in a list. A category the policy adds is rendered after these,
// sorted, rather than dropped.
var changelogRenderOrder = []string{"security", "removed", "deprecated", "added", "changed", "fixed"}

var changelogLabels = map[string]string{
	"security": "Security", "removed": "Removed", "deprecated": "Deprecated",
	"added": "Added", "changed": "Changed", "fixed": "Fixed",
}

// LoadChangelogPolicy reads the policy, falling back to the defaults. A missing
// or unreadable file is not an error here: every caller wants the categories,
// and the gate that requires the file to exist is `pose check`.
func LoadChangelogPolicy(root string) ChangelogPolicy {
	policy := ChangelogPolicy{SchemaVersion: 1}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "changelog.json"))
	if err == nil {
		_ = json.Unmarshal(raw, &policy)
	}
	if len(policy.Categories) == 0 {
		policy.Categories = DefaultChangelogCategories()
	}
	return policy
}

// ValidCategories reports the set a fragment may declare.
func (p ChangelogPolicy) ValidCategories() map[string]bool {
	valid := map[string]bool{}
	for _, category := range p.Categories {
		if category = strings.TrimSpace(category); category != "" {
			valid[category] = true
		}
	}
	if len(valid) == 0 {
		for _, category := range DefaultChangelogCategories() {
			valid[category] = true
		}
	}
	return valid
}

// CategoryList renders the accepted categories the way an error message needs
// them, in the order the policy declares.
func (p ChangelogPolicy) CategoryList() string {
	accepted := []string{}
	for _, category := range p.Categories {
		if category = strings.TrimSpace(category); category != "" {
			accepted = append(accepted, category)
		}
	}
	if len(accepted) == 0 {
		accepted = DefaultChangelogCategories()
	}
	return strings.Join(accepted, "|")
}

// RenderOrder returns the categories in the order the notes present them: the
// known ones first, then anything the policy adds, sorted so the output is
// stable.
func (p ChangelogPolicy) RenderOrder() []string {
	valid := p.ValidCategories()
	order := []string{}
	for _, category := range changelogRenderOrder {
		if valid[category] {
			order = append(order, category)
			delete(valid, category)
		}
	}
	extra := make([]string, 0, len(valid))
	for category := range valid {
		extra = append(extra, category)
	}
	sort.Strings(extra)
	return append(order, extra...)
}

// CategoryLabel is the heading for a category. One the policy adds gets its own
// name capitalised rather than being rendered under someone else's heading.
func CategoryLabel(category string) string {
	if label, ok := changelogLabels[category]; ok {
		return label
	}
	if category == "" {
		return ""
	}
	return strings.ToUpper(category[:1]) + category[1:]
}
