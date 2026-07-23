package pose

// Typed artifact pointers ("spec:my-slug", "report:2026-01-01-x.md"): the
// shared reference resolver used by capability evidence (spec
// pose-capability-mechanism) and by project-state sections (spec
// pose-project-state-artifact R6) so both validate the same nine reference
// kinds the same way instead of drifting apart.

import (
	"fmt"
	"strings"
)

// PointerKinds are the recognized "<kind>:<value>" prefixes. component is
// syntactically valid but never existence-checked locally — GraphForge
// component identity lives outside the POSE artifact tree.
var PointerKinds = []string{"spec", "report", "adr", "knowledge", "doc", "commit", "check", "url", "component"}

// ResolvePointer validates one typed reference against local artifacts.
// ok is true when the reference is well-formed and (for locally-checkable
// kinds) resolves; reason explains a false result.
func (s Store) ResolvePointer(ref string) (ok bool, reason string) {
	kind, value, found := strings.Cut(ref, ":")
	if !found || value == "" {
		return false, fmt.Sprintf("%q is not a typed reference (<type>:<value>)", ref)
	}
	switch kind {
	case "spec":
		if err := ValidateSlug(value); err != nil {
			return false, fmt.Sprintf("spec reference %q has an invalid slug", ref)
		}
		if _, err := s.GetSpec(value); err != nil {
			return false, fmt.Sprintf("spec %q not found in .pose/specs", value)
		}
		return true, ""
	case "report":
		if !localArtifactExists(s.Root, ".pose/reports", value) {
			return false, fmt.Sprintf("report %q not found in .pose/reports", value)
		}
		return true, ""
	case "adr":
		if !localArtifactExists(s.Root, ".pose/adr", value) {
			return false, fmt.Sprintf("adr %q not found in .pose/adr", value)
		}
		return true, ""
	case "knowledge":
		entries, err := s.ListKnowledge()
		if err != nil {
			return false, fmt.Sprintf("knowledge listing unavailable: %v", err)
		}
		for _, entry := range entries {
			if entry.Slug == value {
				return true, ""
			}
		}
		return false, fmt.Sprintf("knowledge %q not found in .pose/knowledge", value)
	case "doc":
		if !localArtifactExists(s.Root, ".", value) {
			return false, fmt.Sprintf("doc %q not found under the project root", value)
		}
		return true, ""
	case "commit":
		if !commitRefPattern.MatchString(value) {
			return false, fmt.Sprintf("commit %q is not a 7-40 char lowercase hex hash", ref)
		}
		return true, ""
	case "check":
		return true, "" // syntactic: any non-empty command string is acceptable
	case "url":
		if !strings.HasPrefix(value, "https://") {
			return false, fmt.Sprintf("url reference %q must start with https://", ref)
		}
		return true, ""
	case "component":
		return true, "" // external identity (GraphForge); never checked locally
	default:
		return false, fmt.Sprintf("reference type %q is not one of %s", kind, strings.Join(PointerKinds, "/"))
	}
}
