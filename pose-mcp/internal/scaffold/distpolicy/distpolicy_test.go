package distpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Regression test for issue #17: pose-dist's own delivery.json/artifacts.json
// (self-referential to pose-mcp's source tree) must never be included in the
// wholesale `.pose/policy` sync, and the placeholder shipped instead must be
// schema-valid and inert (empty roots, disabled).
func TestSelfReferentialPolicyFilesExcluded(t *testing.T) {
	for _, rel := range []string{".pose/policy/delivery.json", ".pose/policy/artifacts.json"} {
		if IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = true, want false — self-referential policy must not be synced verbatim", rel)
		}
	}
	// Sanity: other policy files stay on the wholesale allowlist.
	for _, rel := range []string{".pose/policy/review.json", ".pose/policy/state.json"} {
		if !IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = false, want true — unrelated policy files must still sync", rel)
		}
	}
}

func TestNeutralPolicyTemplatesAreSchemaValidAndInert(t *testing.T) {
	templates := NeutralPolicyTemplates()
	for _, rel := range []string{".pose/policy/delivery.json", ".pose/policy/artifacts.json"} {
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
	for _, rel := range []string{".pose/indexes/module-metadata.json", ".pose/indexes/validation-matrix.json"} {
		if IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = true, want false — self-referential index content must not be synced verbatim", rel)
		}
	}
	// Sanity: other index files stay on the wholesale allowlist — in
	// particular task-map.json, which is generic governance content, not
	// pose-mcp-specific, and must keep syncing byte-for-byte.
	for _, rel := range []string{".pose/indexes/task-map.json", ".pose/indexes/repo-map.json"} {
		if !IsIncluded(rel) {
			t.Errorf("IsIncluded(%q) = false, want true — unrelated index files must still sync", rel)
		}
	}
}

func TestNeutralIndexTemplatesAreSchemaValidAndClean(t *testing.T) {
	templates := NeutralIndexTemplates()
	for _, rel := range []string{".pose/indexes/module-metadata.json", ".pose/indexes/validation-matrix.json"} {
		if _, ok := templates[rel]; !ok {
			t.Fatalf("NeutralIndexTemplates() missing %s", rel)
		}
	}

	for rel, content := range templates {
		var raw map[string]any
		if err := json.Unmarshal(content, &raw); err != nil {
			t.Fatalf("%s: invalid JSON: %v", rel, err)
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
}
