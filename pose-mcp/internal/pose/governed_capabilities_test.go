package pose

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Spec pose-governed-capabilities-default-on-new-instances.

func governedTestPolicy(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(`{
  "schema_version": 2, "enabled": true, "adopted_at": "2026-01-01",
  "profiles": {"spec": "spec-closeout@1"},
  "component_aware": true, "component_aware_adopted_at": "2026-01-01",
  "review_bundles": true, "review_bundles_adopted_at": "2026-01-01",
  "overlay_profiles": ["backend-review@1"]
}`), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestGovernedCapabilitiesAdoptAndRetireThroughTheReader(t *testing.T) {
	ids := []string{}
	for _, capability := range GovernedCapabilities() {
		ids = append(ids, capability.ID)
	}
	if !reflect.DeepEqual(ids, []string{"agency-readiness", "atomic-start", "causality-closeout", "contract-nodes"}) {
		t.Fatalf("governed capabilities = %v", ids)
	}
	store := Store{Root: t.TempDir()}
	doc := governedTestPolicy(t)
	before, _ := json.Marshal(doc)
	for _, capability := range GovernedCapabilities() {
		if changes := AdoptGovernedCapability(doc, capability, "2026-10-06"); len(changes) == 0 {
			t.Fatalf("adopting %s changed nothing", capability.ID)
		}
		if again := AdoptGovernedCapability(doc, capability, "2026-10-07"); len(again) != 0 {
			t.Fatalf("adopting %s twice changed %v; an adopted capability keeps its date", capability.ID, again)
		}
	}
	raw, _ := json.Marshal(doc)
	policy, err := store.ParseReviewPolicyDocument(raw)
	if err != nil {
		t.Fatalf("the adopted policy is refused by the reader: %v\n%s", err, raw)
	}
	if policy.AgencyReadinessVersion != 1 || policy.ContractNodesVersion != 1 || policy.AtomicStartVersion != 1 || policy.CausalityCloseoutVersion != 1 {
		t.Fatalf("versions not adopted: %+v", policy)
	}
	if policy.AtomicStartAdoptedAt != "2026-10-06" || policy.CausalityCloseoutAdoptedAt != "2026-10-06" || policy.OverlayAdoptedAt["structural-materiality@1"] != "2026-10-06" {
		t.Fatalf("adoption dates not written: %+v", policy)
	}
	if !reflect.DeepEqual(policy.OverlayProfiles, []string{"backend-review@1", "structural-materiality@1"}) {
		t.Fatalf("overlay not adopted beside the existing one: %v", policy.OverlayProfiles)
	}
	for _, capability := range GovernedCapabilities() {
		if changes := RetireGovernedCapability(doc, capability); len(changes) == 0 {
			t.Fatalf("retiring %s changed nothing", capability.ID)
		}
	}
	after, _ := json.Marshal(doc)
	if string(after) != string(before) {
		t.Fatalf("adopt then retire did not restore the policy:\n%s\n%s", before, after)
	}
}

func TestGovernedCapabilityLookupNamesTheKnownOnes(t *testing.T) {
	if _, ok := LookupGovernedCapability("causality-closeout"); !ok {
		t.Fatal("a governed capability was not found")
	}
	if _, ok := LookupGovernedCapability("time-travel"); ok {
		t.Fatal("an unknown capability was found")
	}
	if names := strings.Join(GovernedCapabilityIDs(), ", "); names != "agency-readiness, atomic-start, causality-closeout, contract-nodes" {
		t.Fatalf("ids = %s", names)
	}
}

func TestGovernedCapabilitiesNameTheToggleWhenNotAdopted(t *testing.T) {
	root, store := reviewBundleFixture(t)
	setCausalityCutoffPolicy(t, root, func(p map[string]any) {
		p["enabled"], p["review_bundles"], p["review_bundles_adopted_at"] = true, true, "2026-08-01"
	})
	report, err := store.EffectiveGovernance("")
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range GovernedCapabilities() {
		found := false
		for _, entry := range report.Entries {
			if entry.ID == capability.ID {
				found = true
				if !strings.Contains(entry.Explanation, "pose adopt "+capability.ID+" --apply") {
					t.Fatalf("%s does not name its toggle: %s", capability.ID, entry.Explanation)
				}
			}
		}
		if !found {
			t.Fatalf("%s is missing from effective governance", capability.ID)
		}
	}
}
