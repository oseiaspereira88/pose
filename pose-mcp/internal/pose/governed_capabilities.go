package pose

// Governed capabilities are the opt-in review-policy capabilities a project
// turns on as a whole: a version key, the date it applies from where the
// capability has one, and the overlay that carries its obligation where it
// needs one. A new instance adopts them at install; an existing one turns each
// on or off with `pose adopt` (spec
// pose-governed-capabilities-default-on-new-instances). Keeping the keys that
// go together in one place is what lets neither path forget one.

import (
	"sort"
	"strconv"
)

type GovernedCapability struct {
	ID         string
	VersionKey string
	Version    int
	// DateKey names the adoption date the capability reads, or "" when it
	// applies from the moment it is adopted.
	DateKey string
	// Overlay is the review overlay adopted with the capability, dated through
	// overlay_adopted_at, or "".
	Overlay string
	Summary string
}

var governedCapabilities = []GovernedCapability{
	{ID: "agency-readiness", VersionKey: "agency_readiness_version", Version: AgencyReadinessPolicyVersion,
		Summary: "unsatisfied action requests refuse the start, close and release they restrict"},
	{ID: "atomic-start", VersionKey: "atomic_start_version", Version: AtomicStartPolicyVersion, DateKey: "atomic_start_adopted_at",
		Summary: "`pose start --apply` records the R/A/D baseline a spec is reconciled against"},
	{ID: "causality-closeout", VersionKey: "causality_closeout_version", Version: CausalityCloseoutPolicyVersion, DateKey: "causality_closeout_adopted_at", Overlay: "structural-materiality@1",
		Summary: "reviews answer for observed structure with a causal map to the decision basis"},
	{ID: "contract-nodes", VersionKey: "contract_nodes_version", Version: ContractNodesPolicyVersion,
		Summary: "amendments track requirements, assumptions and decisions as versioned nodes"},
}

// GovernedCapabilities returns the registry, sorted by id.
func GovernedCapabilities() []GovernedCapability {
	out := append([]GovernedCapability{}, governedCapabilities...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// GovernedCapabilityIDs lists the ids, sorted.
func GovernedCapabilityIDs() []string {
	ids := []string{}
	for _, capability := range GovernedCapabilities() {
		ids = append(ids, capability.ID)
	}
	return ids
}

// LookupGovernedCapability finds one capability by id.
func LookupGovernedCapability(id string) (GovernedCapability, bool) {
	for _, capability := range governedCapabilities {
		if capability.ID == id {
			return capability, true
		}
	}
	return GovernedCapability{}, false
}

// AdoptGovernedCapability sets the capability's keys in a raw review-policy
// document, dated date, and returns what it changed. An adopted capability
// keeps the date it was adopted on: adopting it again changes nothing, so a
// rerun can never move a cutoff and re-judge work.
func AdoptGovernedCapability(doc map[string]any, capability GovernedCapability, date string) []string {
	changes := []string{}
	if !governedVersionSet(doc, capability) {
		doc[capability.VersionKey] = capability.Version
		changes = append(changes, "set "+capability.VersionKey+"="+strconv.Itoa(capability.Version))
		if capability.DateKey != "" {
			doc[capability.DateKey] = date
			changes = append(changes, "set "+capability.DateKey+"="+date)
		}
	}
	if capability.Overlay != "" {
		overlays := governedStringList(doc["overlay_profiles"])
		if !containsString(overlays, capability.Overlay) {
			doc["overlay_profiles"] = toAnyList(append(overlays, capability.Overlay))
			dates, _ := doc["overlay_adopted_at"].(map[string]any)
			if dates == nil {
				dates = map[string]any{}
			}
			dates[capability.Overlay] = date
			doc["overlay_adopted_at"] = dates
			changes = append(changes, "add overlay_profiles "+capability.Overlay, "set overlay_adopted_at."+capability.Overlay+"="+date)
		}
	}
	return changes
}

// RetireGovernedCapability removes the keys AdoptGovernedCapability sets and
// returns what it changed.
func RetireGovernedCapability(doc map[string]any, capability GovernedCapability) []string {
	changes := []string{}
	for _, key := range []string{capability.VersionKey, capability.DateKey} {
		if key == "" {
			continue
		}
		if _, present := doc[key]; present {
			delete(doc, key)
			changes = append(changes, "remove "+key)
		}
	}
	if capability.Overlay != "" {
		overlays := governedStringList(doc["overlay_profiles"])
		if containsString(overlays, capability.Overlay) {
			kept := []string{}
			for _, ref := range overlays {
				if ref != capability.Overlay {
					kept = append(kept, ref)
				}
			}
			doc["overlay_profiles"] = toAnyList(kept)
			changes = append(changes, "remove overlay_profiles "+capability.Overlay)
		}
		if dates, _ := doc["overlay_adopted_at"].(map[string]any); dates != nil {
			if _, present := dates[capability.Overlay]; present {
				delete(dates, capability.Overlay)
				changes = append(changes, "remove overlay_adopted_at."+capability.Overlay)
			}
			if len(dates) == 0 {
				delete(doc, "overlay_adopted_at")
			}
		}
	}
	return changes
}

// GovernedCapabilityAdopted reports whether the document adopts it.
func GovernedCapabilityAdopted(doc map[string]any, capability GovernedCapability) bool {
	return governedVersionSet(doc, capability)
}

// ParseReviewPolicyDocument runs the review-policy reader over raw bytes, so a
// writer can refuse what the reader would refuse before writing it.
func (s Store) ParseReviewPolicyDocument(raw []byte) (ReviewPolicy, error) {
	return s.parseReviewPolicy(raw)
}

func governedVersionSet(doc map[string]any, capability GovernedCapability) bool {
	switch value := doc[capability.VersionKey].(type) {
	case float64:
		return int(value) == capability.Version
	case int:
		return value == capability.Version
	}
	return false
}

func governedStringList(value any) []string {
	out := []string{}
	if items, ok := value.([]any); ok {
		for _, item := range items {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
	}
	return out
}

func toAnyList(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
