package pose

// Explicit remediation lineage (spec pose-abm-remediation-lineage). This is
// integrity validation, not a claim that a repair worked or caused an outcome.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	remediationMaxSpecs           = 256
	remediationMaxLinks           = 1024
	remediationMaxRegistryEntries = 8192
)

var remediationFindingRef = regexp.MustCompile(`^finding:(rva-[a-f0-9]{16})/([A-Za-z0-9][A-Za-z0-9._-]*)$`)

// RemediationLink identifies a declared relation, independent of dependencies.
type RemediationLink struct {
	Ref      string
	Category string
}

// RemediationValues preserves empty members so malformed lists cannot silently
// lose edges. An empty field/list remains the backwards-compatible opt-out.
func RemediationValues(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	if value == "" {
		return nil
	}
	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

func lineageError(code, detail string) error {
	return fmt.Errorf("remediation-lineage/%s: %s", code, detail)
}

func ParseRemediationLinks(values []string) ([]RemediationLink, error) {
	if len(values) > remediationMaxLinks {
		return nil, lineageError("limit", "too many remediation links")
	}
	links := make([]RemediationLink, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		ref, category, ok := strings.Cut(value, "@")
		if !ok || ref == "" || category == "" {
			return nil, lineageError("syntax", "expected reference@category")
		}
		switch category {
		case "defect-fix", "simplification", "revert", "requirement-change", "planned-evolution":
		default:
			return nil, lineageError("category", "unknown remediation category")
		}
		if strings.HasPrefix(ref, "spec:") {
			if err := ValidateSlug(strings.TrimPrefix(ref, "spec:")); err != nil {
				return nil, lineageError("reference", "invalid spec reference")
			}
		} else if !remediationFindingRef.MatchString(ref) || strings.Contains(ref, "..") {
			return nil, lineageError("reference", "expected spec:slug or finding:rva-<16hex>/id")
		}
		if seen[ref] {
			return nil, lineageError("duplicate", "a target is declared more than once")
		}
		seen[ref] = true
		links = append(links, RemediationLink{Ref: ref, Category: category})
	}
	return links, nil
}

// ValidateRemediationLineage follows only explicit links from the supplied spec.
// No opt-in means no traversal and no dependency on the surrounding corpus.
func (s Store) ValidateRemediationLineage(spec Spec) error {
	if len(spec.Remediates) == 0 {
		return nil
	}
	if _, err := ParseRemediationLinks(spec.Remediates); err != nil {
		return err
	}
	if err := s.confineRemediationSpecs(); err != nil {
		return err
	}
	states := map[string]int{}
	edges := 0
	var visit func(Spec) error
	visit = func(current Spec) error {
		if ValidateSlug(current.Slug) != nil {
			return lineageError("reference", "invalid spec identity")
		}
		if states[current.Slug] == 1 {
			return lineageError("cycle", "remediation graph contains a cycle")
		}
		if states[current.Slug] == 2 {
			return nil
		}
		if len(states) >= remediationMaxSpecs {
			return lineageError("limit", "remediation graph exceeds 256 specs")
		}
		states[current.Slug] = 1
		links, err := ParseRemediationLinks(current.Remediates)
		if err != nil {
			return err
		}
		for _, link := range links {
			edges++
			if edges > remediationMaxLinks {
				return lineageError("limit", "remediation graph exceeds 1024 links")
			}
			slug := ""
			if strings.HasPrefix(link.Ref, "spec:") {
				slug = strings.TrimPrefix(link.Ref, "spec:")
			} else {
				slug, err = s.remediationFindingSpec(link.Ref)
				if err != nil {
					return err
				}
			}
			// Roadmap/milestone findings have no spec node to traverse.
			if slug == "" {
				continue
			}
			if states[slug] == 1 {
				return lineageError("cycle", "remediation graph contains a cycle")
			}
			if states[slug] == 2 {
				continue
			}
			target, err := s.GetSpec(slug)
			if err != nil || target.Slug != slug {
				return lineageError("orphan", "target spec cannot be resolved with its declared identity")
			}
			if err := visit(*target); err != nil {
				return err
			}
		}
		states[current.Slug] = 2
		return nil
	}
	return visit(spec)
}

// Preflight the canonical resolver's registry without following symlinks before
// it reads any linked spec. Fail closed on nonregular/oversized files and bound
// the scan. GetSpec remains the single layout and slug resolver.
func (s Store) confineRemediationSpecs() error {
	dir, err := ensureReviewArtifactDir(s.Root, ".pose/specs", false)
	if err != nil {
		return lineageError("path", "spec registry is unavailable or unconfined")
	}
	entries := 0
	return filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return lineageError("path", "spec registry cannot be inspected")
		}
		entries++
		if entries > remediationMaxRegistryEntries {
			return lineageError("limit", "spec registry exceeds 8192 entries")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return lineageError("path", "spec registry contains a symlink")
		}
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return lineageError("path", "spec registry contains an unreadable or nonregular file")
			}
			if info.Size() > maxReviewBundleBytes {
				return lineageError("limit", "spec registry contains an oversized file")
			}
		}
		return nil
	})
}

func (s Store) remediationFindingSpec(ref string) (string, error) {
	parts := remediationFindingRef.FindStringSubmatch(ref)
	if len(parts) != 3 {
		return "", lineageError("reference", "invalid finding reference")
	}
	if err := s.confineRemediationReviewFile(".pose/review-attestations", parts[1]); err != nil {
		return "", err
	}
	att, err := s.LoadReviewAttestation(parts[1])
	if err != nil {
		return "", lineageError("orphan", "finding attestation is missing or invalid")
	}
	matches := 0
	for _, finding := range att.Findings {
		if finding.ID == parts[2] {
			matches++
		}
	}
	if matches != 1 {
		return "", lineageError("orphan", "finding must resolve exactly once in its attestation")
	}
	if err := s.confineRemediationReviewFile(".pose/review-bundles", att.BundleID); err != nil {
		return "", err
	}
	bundle, err := s.LoadReviewBundle(att.BundleID)
	if err != nil || att.BundleDigest != bundle.BundleDigest {
		return "", lineageError("orphan", "finding bundle is missing or does not match the attestation")
	}
	scope, err := ParseScopeRef(bundle.Payload.Scope.Ref)
	if err != nil {
		return "", lineageError("reference", "finding has an invalid scope")
	}
	if scope.Kind == "spec" {
		return scope.Slug, nil
	}
	return "", nil
}

func (s Store) confineRemediationReviewFile(rel, id string) error {
	if ValidateSlug(id) != nil {
		return lineageError("reference", "invalid review artifact ID")
	}
	dir, err := ensureReviewArtifactDir(s.Root, rel, false)
	if err != nil {
		return lineageError("orphan", "review artifact directory is missing or unconfined")
	}
	info, err := os.Lstat(filepath.Join(dir, id+".json"))
	if err != nil {
		return lineageError("orphan", "review artifact is unavailable")
	}
	if !info.Mode().IsRegular() {
		return lineageError("path", "review artifact is not a regular file")
	}
	if info.Size() > maxReviewBundleBytes {
		return lineageError("limit", "review artifact is oversized")
	}
	return nil
}

// Remediation categories split into the ones that say a delivery had to be
// reworked and the ones that say the world changed around it. Only the first
// group enters the rate: a requirement change is not a defect, and a planned
// evolution is the plan working. Keeping the split explicit in one place is what
// stops the rate from quietly becoming "every link we could find".
var remediationCountedCategories = map[string]bool{
	"defect-fix":     true,
	"revert":         true,
	"simplification": true,
}

func sortedRemediationCountedCategories() []string {
	out := []string{}
	for category := range remediationCountedCategories {
		out = append(out, category)
	}
	sort.Strings(out)
	return out
}

// remediationProjection computes the remediation dimension over the deliveries
// that finished the maturity window.
//
// Three properties it deliberately keeps, because each was a way to publish a
// flattering number:
//
//   - A delivery too recent to have been remediated yet is censored, not counted
//     as unremediated. It appears in Censored and never in the denominator.
//   - A mature delivery with no incoming link is unknown, not clean. Nothing
//     asserts that a delivery nobody linked to was free of defects, so
//     UnlinkedUnknown is reported beside the rate rather than folded into it.
//   - Categories outside the counted set are still reported per category. The
//     reader can see a requirement-change existed; it just does not become a
//     defect.
func (s Store) remediationProjection(now time.Time, sinceDays, maturityDays, minSample int) GovernanceRemediationDimensions {
	dimensions := GovernanceRemediationDimensions{ByCategory: map[string]int{}, CountedCategories: sortedRemediationCountedCategories()}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		dimensions.Reason = "spec registry is unreadable"
		return dimensions
	}
	cutoff := time.Time{}
	if sinceDays > 0 {
		cutoff = now.AddDate(0, 0, -sinceDays)
	}
	matureBefore := now.AddDate(0, 0, -maturityDays)

	// One pass to classify deliveries, one to resolve the links pointing at them.
	// Links are read from every spec, not only the mature ones: a remediation
	// lands after the delivery it repairs, which is the whole reason the window
	// exists.
	mature := map[string]bool{}
	incoming := map[string]map[string]bool{}
	declared := 0
	for i := range specs {
		spec := &specs[i]
		if spec.Status == "done" {
			if at, ok := parseGovernanceTime(spec.CompletedAt); ok && !at.After(now) && (cutoff.IsZero() || !at.Before(cutoff)) {
				if at.Before(matureBefore) {
					mature[spec.Slug] = true
				} else {
					dimensions.Censored++
				}
			}
		}
		links, parseErr := ParseRemediationLinks(spec.Remediates)
		if parseErr != nil {
			dimensions.InvalidLinks += len(spec.Remediates)
			continue
		}
		for _, link := range links {
			declared++
			target := ""
			if rest, ok := strings.CutPrefix(link.Ref, "spec:"); ok {
				target = rest
			} else if resolved, findErr := s.remediationFindingSpec(link.Ref); findErr == nil {
				target = resolved
			} else {
				dimensions.InvalidLinks++
				continue
			}
			// A finding on a roadmap or milestone has no spec node to attribute.
			if target == "" {
				continue
			}
			if incoming[target] == nil {
				incoming[target] = map[string]bool{}
			}
			incoming[target][link.Category] = true
		}
	}

	dimensions.LinksDeclared = declared
	dimensions.MaturePopulation = len(mature)
	if declared == 0 {
		dimensions.Reason = "no delivery declares an explicit remediation link"
		return dimensions
	}
	dimensions.Available = true
	for slug := range mature {
		categories := incoming[slug]
		if len(categories) == 0 {
			dimensions.UnlinkedUnknown++
			continue
		}
		dimensions.LinkedObserved++
		counted := false
		for category := range categories {
			dimensions.ByCategory[category]++
			if remediationCountedCategories[category] {
				counted = true
			}
		}
		if counted {
			dimensions.ObservedRemediated++
		}
	}
	if dimensions.MaturePopulation < minSample {
		dimensions.InsufficientSample = true
	}
	return dimensions
}
