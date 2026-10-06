package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-capability-catalog.

func catalogFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/policy/review.json", `{
  "schema_version": 2, "enabled": true, "adopted_at": "2026-01-01",
  "profiles": {"spec": "spec-closeout@1"},
  "component_aware": true, "component_aware_adopted_at": "2026-01-01",
  "review_bundles": true, "review_bundles_adopted_at": "2026-01-01",
  "allow_criterion_reuse": true,
  "overlay_profiles": []
}`)
	write(".pose/policy/dor.json", `{"schemaVersion":1,"adopted_at":"","defaultTaskType":"feature","taskTypes":{"feature":["Intent"]}}`)
	for _, profile := range []string{"structural-materiality", "engineering-judgment", "high-criticality-review", "backend-review", "frontend-review"} {
		write(".pose/review-profiles/"+profile+".json", `{"schema_version":2,"id":"`+profile+`","version":1,"scope":"spec","selectors":{"languages":["go"]},"criteria":[{"id":"c","kind":"judgment","description":"d"}]}`)
	}
	return root
}

func TestCapabilityCatalogDescribesEveryEntry(t *testing.T) {
	want := []string{"agency-readiness", "atomic-start", "causality-closeout", "contract-nodes", "criterion-reuse", "definition-of-ready",
		"overlay:backend-review", "overlay:engineering-judgment", "overlay:frontend-review", "overlay:high-criticality-review", "overlay:structural-materiality",
		"qualified-artifact-refs", "signed-attestations", "spec-authority-transfer", "verified-identity"}
	got := []string{}
	for _, entry := range CapabilityCatalog() {
		got = append(got, entry.ID)
		if entry.Summary == "" || entry.Effect == "" || entry.IntroducedIn == "" {
			t.Fatalf("%s is not fully described: %+v", entry.ID, entry)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("catalog = %v\nwant    %v", got, want)
	}
	for _, capability := range GovernedCapabilities() {
		entry, ok := LookupCatalogEntry(capability.ID)
		if !ok || !entry.DefaultForNew {
			t.Fatalf("%s is adopted by a new instance but the catalog does not say so", capability.ID)
		}
	}
}

func TestCapabilityCatalogAdoptsAndRetiresEveryToggleThroughTheReader(t *testing.T) {
	root := catalogFixture(t)
	store := Store{Root: root}
	order := []string{"agency-readiness", "atomic-start", "causality-closeout", "contract-nodes", "definition-of-ready",
		"overlay:backend-review", "overlay:engineering-judgment", "overlay:frontend-review", "overlay:high-criticality-review",
		"qualified-artifact-refs", "spec-authority-transfer"}
	for _, id := range order {
		docs, err := LoadPolicyDocs(root)
		if err != nil {
			t.Fatal(err)
		}
		entry, _ := LookupCatalogEntry(id)
		if blocker := CatalogAdoptBlocker(root, docs, entry); blocker != "" {
			t.Fatalf("%s blocked: %s", id, blocker)
		}
		if changes := entry.Adopt(docs, "2026-10-06"); len(changes) == 0 {
			t.Fatalf("adopting %s changed nothing", id)
		}
		if err := docs.Write(root, store); err != nil {
			t.Fatalf("adopting %s produced a policy the reader refuses: %v", id, err)
		}
		reloaded, _ := LoadPolicyDocs(root)
		if !entry.Adopted(reloaded) {
			t.Fatalf("%s does not read back as adopted", id)
		}
	}
	policy, err := store.GetReviewPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if policy.SchemaVersion != SpecAuthorityTransferPolicySchemaVersion || policy.QualifiedArtifactRefsVersion != 1 || policy.SpecAuthorityTransferVersion != 1 {
		t.Fatalf("schema-bound capabilities did not raise the schema: %+v", policy)
	}
	if LoadDoRPolicy(root).AdoptedAt != "2026-10-06" {
		t.Fatal("the Definition of Ready was not dated")
	}
	if policy.OverlayAdoptedAt["engineering-judgment@1"] != "2026-10-06" {
		t.Fatalf("an overlay adopted later is not dated: %v", policy.OverlayAdoptedAt)
	}
	for i := len(order) - 1; i >= 0; i-- {
		docs, _ := LoadPolicyDocs(root)
		entry, _ := LookupCatalogEntry(order[i])
		if blocker := CatalogRetireBlocker(docs, entry); blocker != "" {
			t.Fatalf("retiring %s blocked: %s", order[i], blocker)
		}
		entry.Retire(docs)
		if err := docs.Write(root, store); err != nil {
			t.Fatalf("retiring %s produced a policy the reader refuses: %v", order[i], err)
		}
	}
	policy, _ = store.GetReviewPolicy()
	if policy.SchemaVersion != ReviewPolicySchemaVersion || len(policy.OverlayProfiles) != 0 || policy.AgencyReadinessVersion != 0 {
		t.Fatalf("retiring everything did not restore the original contract: %+v", policy)
	}
}

func TestCapabilityCatalogRefusesMissingRequirementsAndPrerequisites(t *testing.T) {
	root := catalogFixture(t)
	docs, _ := LoadPolicyDocs(root)
	transfer, _ := LookupCatalogEntry("spec-authority-transfer")
	if blocker := CatalogAdoptBlocker(root, docs, transfer); !strings.Contains(blocker, "qualified-artifact-refs") || !strings.Contains(blocker, "pose adopt qualified-artifact-refs") {
		t.Fatalf("a missing required capability was not named: %q", blocker)
	}
	signed, _ := LookupCatalogEntry("signed-attestations")
	if blocker := CatalogAdoptBlocker(root, docs, signed); !strings.Contains(blocker, "trusted") {
		t.Fatalf("an unmet prerequisite was not named: %q", blocker)
	}
	refs, _ := LookupCatalogEntry("qualified-artifact-refs")
	refs.Adopt(docs, "2026-10-06")
	transfer.Adopt(docs, "2026-10-06")
	if blocker := CatalogRetireBlocker(docs, refs); !strings.Contains(blocker, "spec-authority-transfer") {
		t.Fatalf("turning off a capability another one requires was not refused: %q", blocker)
	}
}

func TestCapabilityCatalogRecordsDeclineAndDefer(t *testing.T) {
	root := catalogFixture(t)
	if err := RecordAdoptionDecision(root, "overlay:engineering-judgment", AdoptionDeclined, "judgment is reviewed by people here", "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if err := RecordAdoptionDecision(root, "contract-nodes", AdoptionDeferred, "revisit after 7.1", "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if err := RecordAdoptionDecision(root, "time-travel", AdoptionDeclined, "x", "2026-10-05"); err == nil {
		t.Fatal("a decision for an unknown capability was recorded")
	}
	states, err := CapabilityStates(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]CapabilityState{}
	for _, state := range states {
		byID[state.ID] = state
	}
	if byID["overlay:engineering-judgment"].State != CapabilityDeclined || byID["overlay:engineering-judgment"].Decision.Reason == "" {
		t.Fatalf("decline not reported: %+v", byID["overlay:engineering-judgment"])
	}
	if byID["contract-nodes"].State != CapabilityDeferred {
		t.Fatalf("defer not reported: %+v", byID["contract-nodes"])
	}
	if byID["agency-readiness"].State != CapabilityOff || byID["signed-attestations"].State != CapabilityNeedsSetup || byID["criterion-reuse"].State != CapabilityOn {
		t.Fatalf("states: agency=%s signed=%s reuse=%s", byID["agency-readiness"].State, byID["signed-attestations"].State, byID["criterion-reuse"].State)
	}
	if err := ClearAdoptionDecision(root, "contract-nodes"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".pose", "policy", "adoption-decisions.json"))
	var doc AdoptionDecisions
	if err := json.Unmarshal(raw, &doc); err != nil || doc.SchemaVersion != 1 || len(doc.Decisions) != 1 {
		t.Fatalf("decision file after clearing: %s (%v)", raw, err)
	}
}

func TestCapabilityCatalogIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "pose adopt --list") || !strings.Contains(string(raw), "`.pose/policy/adoption-decisions.json`") {
			t.Fatalf("%s does not document the catalog and its decision record", rel)
		}
	}
}

// Verified identity names the project its claims must bind; adopting it where
// .pose/project.json declares the id writes authority_project, so the policy
// the reader then loads is valid.
func TestCapabilityCatalogVerifiedIdentityWritesTheDeclaredProject(t *testing.T) {
	root := catalogFixture(t)
	if _, err := WriteProjectFile(root, "proj.catalog", ""); err != nil {
		t.Fatal(err)
	}
	docs, _ := LoadPolicyDocs(root)
	docs.Review["trusted_attestation_issuers"] = []any{"test:issuer#sha256:" + strings.Repeat("a", 64)}
	docs.Review["authority_audience"] = "harne8:tenant-a"
	entry, _ := LookupCatalogEntry("verified-identity")
	if blocker := CatalogAdoptBlocker(root, docs, entry); blocker != "" {
		t.Fatalf("verified identity is blocked: %s", blocker)
	}
	changes := entry.Adopt(docs, "2026-10-06")
	if err := docs.Write(root, Store{Root: root}); err != nil {
		t.Fatalf("adopting verified identity produced a policy the reader refuses: %v (changes %v)", err, changes)
	}
	policy, err := Store{Root: root}.GetReviewPolicy()
	if err != nil || policy.AuthorityProject != "proj.catalog" {
		t.Fatalf("authority_project = %q (%v)", policy.AuthorityProject, err)
	}
}
