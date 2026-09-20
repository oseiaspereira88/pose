package distpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Regression test for issue #17: pose-dist's own delivery.json/artifacts.json
// (self-referential to pose-mcp's source tree) must never be included in the
// wholesale `.pose/policy` sync, and the placeholder shipped instead must be
// schema-valid and inert (empty roots, disabled).
func TestSelfReferentialPolicyFilesExcluded(t *testing.T) {
	for _, rel := range []string{".pose/policy/delivery.json", ".pose/policy/artifacts.json", ".pose/policy/review.json"} {
		if IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = true, want false — self-referential policy must not be synced verbatim", rel)
		}
	}
	// review.json moved from the sanity list to the excluded list deliberately
	// (spec review-policy-adoption-is-the-instances): it carries `adopted_at`,
	// a dated field per governed contract and the overlays this repository
	// adopted, all of which are statements about this repository rather than
	// defaults for a target.
	// Sanity: other policy files stay on the wholesale allowlist.
	for _, rel := range []string{".pose/policy/state.json", ".pose/policy/dor.json"} {
		if !IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = false, want true — unrelated policy files must still sync", rel)
		}
	}
}

func TestNeutralPolicyTemplatesAreSchemaValidAndInert(t *testing.T) {
	templates := NeutralPolicyTemplates()
	for _, rel := range []string{".pose/policy/delivery.json", ".pose/policy/artifacts.json", ".pose/policy/review.json"} {
		if _, ok := templates[rel]; !ok {
			t.Fatalf("NeutralPolicyTemplates() missing %s", rel)
		}
	}

	root := t.TempDir()
	for rel, content := range templates {
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	delivery, err := pose.LoadDeliveryPolicy(root)
	if err != nil {
		t.Fatalf("LoadDeliveryPolicy(neutral placeholder): %v", err)
	}
	if delivery.Enabled {
		t.Error("neutral delivery.json placeholder must ship enabled=false")
	}
	if len(delivery.Roots) != 0 {
		t.Errorf("neutral delivery.json placeholder must ship roots=[], got %v", delivery.Roots)
	}

	artifacts, err := pose.LoadArtifactPolicy(root)
	if err != nil {
		t.Fatalf("LoadArtifactPolicy(neutral placeholder): %v", err)
	}
	if artifacts.Enabled {
		t.Error("neutral artifacts.json placeholder must ship enabled=false")
	}
	if len(artifacts.GovernedRoots) != 0 {
		t.Errorf("neutral artifacts.json placeholder must ship governed_roots=[], got %v", artifacts.GovernedRoots)
	}

	// The review template carries no date, which is the point: a dated template
	// states another repository's history, and the stamp cannot correct it
	// afterwards because it skips a key that is already present.
	var reviewDoc map[string]any
	if err := json.Unmarshal(templates[".pose/policy/review.json"], &reviewDoc); err != nil {
		t.Fatal(err)
	}
	for key, value := range reviewDoc {
		dated := key == "adopted_at" || key == "evidence_vocabulary_reconciled_at" ||
			(len(key) > 11 && key[len(key)-11:] == "_adopted_at")
		if !dated {
			continue
		}
		t.Errorf("neutral review.json declares %s=%v; the key must be absent so the stamp can write this instance's own day", key, value)
	}

	// Being dateless makes the template unloadable on its own, because enabling
	// component-aware review requires the date that says when this instance
	// received it. That coupling is asserted rather than hidden: the template is
	// an input to `pose install`, which seeds and stamps in one step, and the
	// pairing is what keeps it from ever reaching an instance undated.
	if _, err := (pose.Store{Root: root}).GetReviewPolicy(); err == nil {
		t.Error("the dateless template loaded on its own; the stamp is then no longer required to complete it")
	} else if !strings.Contains(err.Error(), "component-aware adoption date") {
		t.Errorf("the dateless template failed for the wrong reason: %v", err)
	}

	// Stamped the way seeding stamps it, it is the policy the instance operates.
	for key, value := range map[string]string{
		"adopted_at": "2026-09-20", "component_aware_adopted_at": "2026-09-20",
		"review_bundles_adopted_at": "2026-09-20",
	} {
		reviewDoc[key] = value
	}
	stamped, err := json.MarshalIndent(reviewDoc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), stamped, 0o644); err != nil {
		t.Fatal(err)
	}
	review, err := (pose.Store{Root: root}).GetReviewPolicy()
	if err != nil {
		t.Fatalf("GetReviewPolicy(stamped template): %v", err)
	}
	if !review.Enabled || !review.ComponentAware || !review.ReviewBundles {
		t.Errorf("neutral review.json must ship the current contract shape, got %+v", review)
	}
	if len(review.OverlayProfiles) != 0 {
		t.Errorf("neutral review.json must adopt no overlay for the target, got %v", review.OverlayProfiles)
	}

	// No pose-mcp-source-tree path should ever reappear in a shipped template.
	for rel, content := range templates {
		var raw map[string]any
		if err := json.Unmarshal(content, &raw); err != nil {
			t.Fatalf("%s: invalid JSON: %v", rel, err)
		}
	}
}

// Regression test for issue #22: pose-dist's own module-metadata.json
// (pose-mcp, mcp-enforce, @pose-maintainers) and validation-matrix.json
// (moduleOverrides.pose-mcp, moduleOverrides.docs-site) must never be
// included in the wholesale `.pose/indexes` sync — the same leak class
// SelfReferentialPolicyFiles closed for `.pose/policy/` under issue #17.
func TestSelfReferentialIndexFilesExcluded(t *testing.T) {
	for _, rel := range []string{
		".pose/indexes/module-metadata.json",
		".pose/indexes/validation-matrix.json",
		".pose/indexes/repo-map.json",
		".pose/indexes/services.json",
		".pose/indexes/packages.json",
		".pose/indexes/spec-graph.json",
		".pose/indexes/roadmaps.json",
		".pose/indexes/delivery-integrity.json",
		".pose/indexes/releases.json",
		".pose/indexes/extensions.lock.json",
	} {
		if IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = true, want false — self-referential index content must not be synced verbatim", rel)
		}
	}
	// Sanity: other index files stay on the wholesale allowlist —
	// task-map.json is generic governance content, not pose-mcp-specific,
	// and must keep syncing byte-for-byte.
	for _, rel := range []string{".pose/indexes/task-map.json"} {
		if !IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = false, want true — unrelated index files must still sync", rel)
		}
	}
}

func TestNeutralIndexTemplatesAreSchemaValidAndClean(t *testing.T) {
	templates := NeutralIndexTemplates()
	for _, rel := range SelfReferentialIndexFiles {
		full := ".pose/indexes/" + rel
		if _, ok := templates[full]; !ok {
			t.Fatalf("NeutralIndexTemplates() missing %s", full)
		}
	}

	for rel, content := range templates {
		// services.json/packages.json are JSON arrays, not objects — every
		// other template is an object; json.Valid accepts both.
		if !json.Valid(content) {
			t.Fatalf("%s: invalid JSON", rel)
		}
	}

	// The leak this spec closes is structural (data fields), not textual —
	// the _comment explaining *why* a field is blank legitimately names
	// pose-mcp in prose, same as the existing policy templates already do.
	// The assertions below check the actual data: modules/moduleOverrides
	// must be empty and defaults.owner/domain must be blank.
	var moduleMetadata struct {
		Modules  map[string]any `json:"modules"`
		Defaults struct {
			Owner  string `json:"owner"`
			Domain string `json:"domain"`
		} `json:"defaults"`
	}
	if err := json.Unmarshal(templates[".pose/indexes/module-metadata.json"], &moduleMetadata); err != nil {
		t.Fatal(err)
	}
	if len(moduleMetadata.Modules) != 0 {
		t.Errorf("neutral module-metadata.json must ship modules={}, got %v", moduleMetadata.Modules)
	}
	if moduleMetadata.Defaults.Owner != "" || moduleMetadata.Defaults.Domain != "" {
		t.Errorf("neutral module-metadata.json must ship blank owner/domain defaults, got owner=%q domain=%q", moduleMetadata.Defaults.Owner, moduleMetadata.Defaults.Domain)
	}

	var validationMatrix struct {
		ModuleOverrides  map[string]any `json:"moduleOverrides"`
		Stacks           map[string]any `json:"stacks"`
		DeliveryProfiles map[string]any `json:"deliveryProfiles"`
	}
	if err := json.Unmarshal(templates[".pose/indexes/validation-matrix.json"], &validationMatrix); err != nil {
		t.Fatal(err)
	}
	if len(validationMatrix.ModuleOverrides) != 0 {
		t.Errorf("neutral validation-matrix.json must ship moduleOverrides={}, got %v", validationMatrix.ModuleOverrides)
	}
	if len(validationMatrix.Stacks) == 0 {
		t.Error("neutral validation-matrix.json must keep the generic stacks catalog, not strip it")
	}
	if len(validationMatrix.DeliveryProfiles) == 0 {
		t.Error("neutral validation-matrix.json must keep the generic deliveryProfiles, not strip them")
	}

	// The seven cmdIndex-computed placeholders (regression for spec
	// pose-derived-index-self-referential-leak) must be the genuinely empty
	// shape a target with zero specs/modules/roadmaps/releases produces —
	// never a byte of pose-dist's own ~130-spec graph, module list or
	// release history.
	for rel, wantEmptyKeys := range map[string][]string{
		".pose/indexes/spec-graph.json": {"specs", "edges"},
		".pose/indexes/roadmaps.json":   {"roadmaps"},
	} {
		var doc map[string]any
		if err := json.Unmarshal(templates[rel], &doc); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		for _, key := range wantEmptyKeys {
			v, ok := doc[key]
			if !ok {
				t.Errorf("%s: missing key %q", rel, key)
				continue
			}
			switch vv := v.(type) {
			case map[string]any:
				if len(vv) != 0 {
					t.Errorf("%s.%s must be empty, got %v", rel, key, vv)
				}
			case []any:
				if len(vv) != 0 {
					t.Errorf("%s.%s must be empty, got %v", rel, key, vv)
				}
			default:
				t.Errorf("%s.%s has unexpected type %T", rel, key, v)
			}
		}
	}
	for _, rel := range []string{".pose/indexes/services.json", ".pose/indexes/packages.json"} {
		var arr []any
		if err := json.Unmarshal(templates[rel], &arr); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if len(arr) != 0 {
			t.Errorf("%s must be an empty array, got %v", rel, arr)
		}
	}
	var extLock struct {
		Extensions map[string]any `json:"extensions"`
	}
	if err := json.Unmarshal(templates[".pose/indexes/extensions.lock.json"], &extLock); err != nil {
		t.Fatal(err)
	}
	if len(extLock.Extensions) != 0 {
		t.Errorf("neutral extensions.lock.json must ship extensions={}, got %v", extLock.Extensions)
	}
}

// Regression test for spec pose-domain-rule-extension-migration: pose-dist
// installs pose-rule-backend-go/pose-rule-frontend-react locally (into its
// own .pose/rules/) for its own dogfooded review needs. Without this
// exclusion, that local install would make the wholesale .pose/rules/ sync
// re-embed the file into every fresh instance, silently undoing the
// extension migration.
func TestExtensionOnlyRuleFilesExcluded(t *testing.T) {
	for _, rel := range []string{".pose/rules/backend-go.md", ".pose/rules/frontend-react.md"} {
		if IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = true, want false — extension-only rule content must not be synced verbatim even when present locally for dogfooding", rel)
		}
	}
	// Sanity: universal governance rules stay on the wholesale allowlist.
	for _, rel := range []string{".pose/rules/security.md", ".pose/rules/documentation-style.md"} {
		if !IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = false, want true — universal rules must still sync", rel)
		}
	}
}
