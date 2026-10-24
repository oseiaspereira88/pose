package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPolicyScopeRejectsUnsupportedMapKeys(t *testing.T) {
	for _, version := range []int{1, 2, 3, 4} {
		for _, field := range []string{"identity_assurance", "reviewer_independence"} {
			for _, kind := range []string{"", "spce", " spec", "spec ", "SPEC", "spec\n", "future", "spec:alpha"} {
				t.Run(fmt.Sprintf("%s/%q/v%d", field, kind, version), func(t *testing.T) {
					policy := map[string]any{"schema_version": version, "enabled": true}
					if version >= 3 {
						policy["qualified_artifact_refs_version"] = 1
					}
					if version == 4 {
						policy["spec_authority_transfer_version"] = 1
					}
					value := "declared"
					if field == "reviewer_independence" {
						value = "different-actor"
					}
					policy[field] = map[string]string{"spec": value, kind: value}
					raw, _ := json.Marshal(policy)
					s := governanceFixture(t, string(raw), "")
					_, err := s.GetReviewPolicy()
					if err == nil || !strings.Contains(err.Error(), field) || !strings.Contains(err.Error(), "scope") {
						t.Fatalf("expected localized invalid scope: %v", err)
					}
					projection, err := s.EffectiveGovernance("")
					if err == nil || len(projection.Entries) != 0 {
						t.Fatalf("invalid policy projected protection: %+v %v", projection, err)
					}
					_, err = s.PrepareReviewBundle("spec:alpha")
					if err == nil || !strings.Contains(err.Error(), field) {
						t.Fatalf("preparation bypassed validation: %v", err)
					}
					after, _ := os.ReadFile(filepath.Join(s.Root, ".pose/policy/review.json"))
					if string(after) != string(raw) {
						t.Fatal("operator policy rewritten")
					}
				})
			}
		}
	}
}

func TestPolicyScopeKnownKindsAndDefaults(t *testing.T) {
	for _, kind := range []string{"spec", "milestone", "roadmap"} {
		for _, version := range []int{1, 2, 3, 4} {
			p := map[string]any{"schema_version": version, "enabled": true, "identity_assurance": map[string]string{kind: "declared"}, "reviewer_independence": map[string]string{kind: "different-actor"}, "future_metadata": map[string]any{"custom": true}}
			if version >= 3 {
				p["qualified_artifact_refs_version"] = 1
			}
			if version == 4 {
				p["spec_authority_transfer_version"] = 1
			}
			raw, _ := json.Marshal(p)
			s := governanceFixture(t, string(raw), "")
			policy, err := s.GetReviewPolicy()
			if err != nil {
				t.Fatal(err)
			}
			if policy.ReviewIdentityAssurance(kind) != "declared" || policy.ReviewIdentityAssurance("spec") != "declared" {
				t.Fatal("known-kind default changed")
			}
		}
	}
	for _, kind := range []string{"spce", "", " spec", "SPEC", "future"} {
		p := ReviewPolicy{IdentityAssurance: map[string]string{kind: "verified"}}
		if got := p.ReviewIdentityAssurance(kind); got != "" {
			t.Fatalf("unsupported kind %q gained assurance %q", kind, got)
		}
	}
	s := Store{Root: t.TempDir()}
	p, err := s.GetReviewPolicy()
	if err != nil || p.ReviewIdentityAssurance("spec") != "declared" {
		t.Fatalf("absent policy default: %+v %v", p, err)
	}
	s = governanceFixture(t, `{"schema_version":99,"identity_assurance":{"future":"verified"}}`, "")
	if _, err = s.GetReviewPolicy(); err == nil {
		t.Fatal("unsupported schema admitted")
	}
}

func TestPolicyScopeInvalidCurrentPolicyPreservesSealedHistory(t *testing.T) {
	root, s := reviewBundleFixture(t)
	bundle, err := s.SealReviewBundle("spec:backend", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(bundle)
	writeReviewFixture(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"identity_assurance":{"spce":"declared"}}`)
	loaded, err := s.LoadReviewBundle(bundle.BundleID)
	if err != nil {
		t.Fatalf("historical bundle unreadable: %v", err)
	}
	after, _ := json.Marshal(loaded)
	if string(after) != string(before) {
		t.Fatal("sealed history changed")
	}
	history, err := s.ReadReviewBundles("spec:backend")
	if err != nil || len(history.Bundles) != 1 {
		t.Fatalf("history lost: %+v %v", history, err)
	}
	if _, err = s.PrepareReviewBundle("spec:backend"); err == nil {
		t.Fatal("new preparation accepted invalid current policy")
	}
	if projection, err := s.EffectiveGovernance("spec:backend"); err == nil || len(projection.Entries) > 0 {
		t.Fatalf("invalid policy became effective: %+v %v", projection, err)
	}
}

func TestPolicyScopeInvalidProtectedBaselineCannotDowngradeToWarning(t *testing.T) {
	root, s := protectedBaselineFixture(t)
	writeReviewFixture(t, root, ".pose/policy/review.json", strings.Replace(protectedBaselinePolicy, `"component_aware": true`, `"identity_assurance":{"spce":"declared"},"component_aware":true`, 1))
	designDeltaGit(t, root, "add", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "legacy invalid selector")
	revision := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	graph := DeliveryIntegrityGraph{SchemaVersion: DeliveryIntegritySchemaVersion, ChangeSets: []ChangeSet{{ID: "cs-governance", Spec: "governance", ResolvedBase: revision, ResolvedHead: revision, Paths: []ObservedPath{{Action: "modified", Path: ".pose/policy/review.json"}}}}}
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	writeReviewFixture(t, root, ".pose/policy/review.json", protectedBaselinePolicy)
	_, err := s.ReviewPlan("spec:governance")
	if err == nil || !strings.Contains(err.Error(), "protected review policy baseline") || !strings.Contains(err.Error(), `identity_assurance["spce"]`) {
		t.Fatalf("invalid baseline became unprotected plan: %v", err)
	}
}
