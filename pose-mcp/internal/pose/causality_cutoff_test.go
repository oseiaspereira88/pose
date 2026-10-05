package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-causality-closeout-adoption-cutoff. The fixture's only spec,
// `backend`, was created on 2026-08-13.

func setCausalityCutoffPolicy(t *testing.T, root string, edit func(map[string]any)) {
	t.Helper()
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	edit(policy)
	raw, _ = json.Marshal(policy)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCausalityCutoffStampsOnlyScopesCreatedOnOrAfterTheDate(t *testing.T) {
	root, store := reviewBundleFixture(t)
	stamp := func(cutoff string) ReviewBundle {
		t.Helper()
		setCausalityCutoffPolicy(t, root, func(p map[string]any) {
			p["causality_closeout_version"] = 1
			if cutoff == "" {
				delete(p, "causality_closeout_adopted_at")
			} else {
				p["causality_closeout_adopted_at"] = cutoff
			}
		})
		bundle, err := store.PrepareReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		return bundle
	}

	older := stamp("2026-08-14")
	if governed, _ := BundleGovernedBy(older, CausalityCloseoutContract); governed || older.Payload.Basis != nil {
		t.Fatalf("a scope created before the cutoff was held to causality closeout: %+v", older.Payload.GoverningContracts)
	}
	if !strings.Contains(strings.Join(older.Warnings, "\n"), "causality_closeout_adopted_at 2026-08-14") {
		t.Fatalf("the exemption is not explained: %v", older.Warnings)
	}
	for _, cutoff := range []string{"2026-08-13", ""} {
		bundle := stamp(cutoff)
		if governed, _ := BundleGovernedBy(bundle, CausalityCloseoutContract); !governed {
			t.Fatalf("cutoff %q: a scope created on or after the date, or with no cutoff, was not stamped", cutoff)
		}
	}

	// An undated spec is never exempted.
	specPath := filepath.Join(root, ".pose", "specs", "backend", "spec.md")
	raw, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, []byte(strings.Replace(string(raw), "created_at: 2026-08-13\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if governed, _ := BundleGovernedBy(stamp("2026-08-14"), CausalityCloseoutContract); !governed {
		t.Fatal("a spec without a creation date escaped the contract")
	}
}

func TestCausalityCutoffSkipsALateOverlayForOlderScopes(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/structural-materiality.json", `{
  "schema_version": 2, "id": "structural-materiality", "version": 1, "scope": "spec",
  "selectors": {"structural_kinds": ["dependency", "component", "delivery-metadata", "governance-contract", "public-contract", "submodule"]},
  "criteria": [{"id": "design-causality", "kind": "judgment", "requires_structural_mapping": true, "description": "Map each material fact."}]
}`)
	plan := func(dates map[string]any) ReviewPlan {
		t.Helper()
		setCausalityCutoffPolicy(t, root, func(p map[string]any) {
			overlays, _ := p["overlay_profiles"].([]any)
			found := false
			for _, ref := range overlays {
				found = found || ref == "structural-materiality@1"
			}
			if !found {
				p["overlay_profiles"] = append(overlays, "structural-materiality@1")
			}
			if dates == nil {
				delete(p, "overlay_adopted_at")
			} else {
				p["overlay_adopted_at"] = dates
			}
		})
		result, err := store.ReviewPlan("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	selected := func(p ReviewPlan, ref string) bool {
		for _, profile := range p.SelectedProfiles {
			if profile.Ref == ref {
				return true
			}
		}
		return false
	}

	undated := plan(nil)
	if undated.Structure == nil {
		t.Fatal("an undated structural overlay did not resolve the structure")
	}
	late := plan(map[string]any{"structural-materiality@1": "2026-08-14"})
	if late.Structure != nil || selected(late, "structural-materiality@1") {
		t.Fatalf("an overlay adopted after the scope was created still applied: structure=%+v", late.Structure)
	}
	if !strings.Contains(strings.Join(late.Explain, "\n"), "structural-materiality@1 not selected: adopted on 2026-08-14") {
		t.Fatalf("the skipped overlay is not explained: %v", late.Explain)
	}
	if onTheDay := plan(map[string]any{"structural-materiality@1": "2026-08-13"}); onTheDay.Structure == nil {
		t.Fatal("an overlay adopted on the scope's creation day was skipped")
	}
}

func TestCausalityCutoffRefusesMalformedOrOrphanedDates(t *testing.T) {
	root, store := reviewBundleFixture(t)
	cases := map[string]func(map[string]any){
		"causality date":   func(p map[string]any) { p["causality_closeout_adopted_at"] = "14/08/2026" },
		"overlay date":     func(p map[string]any) { p["overlay_adopted_at"] = map[string]any{"backend-review@1": "soon"} },
		"orphaned overlay": func(p map[string]any) { p["overlay_adopted_at"] = map[string]any{"not-listed@1": "2026-08-14"} },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			setCausalityCutoffPolicy(t, root, func(p map[string]any) {
				delete(p, "causality_closeout_adopted_at")
				delete(p, "overlay_adopted_at")
				edit(p)
			})
			if _, err := store.GetReviewPolicy(); err == nil {
				t.Fatal("the policy reader accepted it")
			}
		})
	}
}

func TestCausalityCutoffIsStatedByEffectiveGovernance(t *testing.T) {
	root, store := reviewBundleFixture(t)
	setCausalityCutoffPolicy(t, root, func(p map[string]any) {
		p["enabled"], p["review_bundles"], p["review_bundles_adopted_at"] = true, true, "2026-08-01"
		p["causality_closeout_version"] = 1
		p["causality_closeout_adopted_at"] = "2026-08-14"
		p["overlay_adopted_at"] = map[string]any{"backend-review@1": "2026-08-14"}
	})
	report, err := store.EffectiveGovernance("")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(report)
	for _, want := range []string{"specs created before 2026-08-14 are not held to causality closeout", "backend-review@1 applies to specs created on or after 2026-08-14"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("effective governance does not state %q:\n%s", want, raw)
		}
	}
}

func TestCausalityCutoffIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"causality_closeout_adopted_at", "overlay_adopted_at"} {
			if !strings.Contains(string(raw), "`"+key) {
				t.Fatalf("%s does not document %s", rel, key)
			}
		}
	}
}
