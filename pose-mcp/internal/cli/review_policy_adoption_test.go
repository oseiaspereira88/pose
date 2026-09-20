package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/scaffold"
)

// reviewPolicyDatedKey names the policy fields that assert something about a
// repository's own history, which is what must never travel in a distribution.
func reviewPolicyDatedKey(key string) bool {
	return key == "adopted_at" || key == "evidence_vocabulary_reconciled_at" ||
		(len(key) > 11 && key[len(key)-11:] == "_adopted_at")
}

// An adoption date is a statement about one repository's own history: work
// completed before it keeps its approval under the contract that arrived on that
// day. A date that travelled from the distribution is a statement about someone
// else's history, and `stampContractAdoption` skips any contract already
// recorded — so an instance that receives one never earns its own.
func readInstalledReviewPolicy(t *testing.T, root string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func installFreshInstance(t *testing.T) (string, map[string]any) {
	t.Helper()
	root := newGitRepo(t)
	var out, errOut bytes.Buffer
	if code := Main([]string{"install", root, "--skip-mcp"}, &out, &errOut); code != 0 {
		t.Fatalf("install code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	return root, readInstalledReviewPolicy(t, root)
}

func TestInstallStampsItsOwnReviewAdoption(t *testing.T) {
	root, doc := installFreshInstance(t)
	today := time.Now().UTC().Format(time.DateOnly)

	// Every dated field in the policy must be either absent or today's date.
	// A date from before this install is a date from another repository.
	for key, value := range doc {
		text, ok := value.(string)
		if !ok {
			continue
		}
		if !reviewPolicyDatedKey(key) {
			continue
		}
		if text != "" && text != today {
			t.Errorf("fresh install inherited %s=%s, which it did not earn (today is %s)", key, text, today)
		}
	}

	// Each contract in the registry must carry this instance's own date, so the
	// exemption boundary is the day this repository received the contract.
	for _, contract := range posemodel.ReviewContracts() {
		legacy := posemodel.LegacyContractField(contract.ID)
		if legacy == "" {
			continue
		}
		got, _ := doc[legacy].(string)
		if got != today {
			t.Errorf("contract %s: %s=%q, want today's date %q", contract.ID, legacy, got, today)
		}
	}

	// And the policy has to be operable: the engine must resolve a plan from it.
	writeCloseoutCLIFile(t, root, ".pose/specs/fresh.md",
		"---\nslug: fresh\nstatus: in-progress\ncreated_at: 2026-09-20\n---\n# Spec\n\n## 2. Requirements\n\n- R1: exist.\n")
	if _, err := (posemodel.Store{Root: root}).ReviewPlan("spec:fresh"); err != nil {
		t.Fatalf("the seeded policy cannot resolve a review plan: %v", err)
	}
}

// The distribution must not carry a dated review policy at all.
//
// Measured against the embedded template rather than against an installed
// instance: this repository stamped its own adoption today, and a fresh install
// stamps today too, so comparing the two results cannot tell inheritance from
// coincidence. The template is the thing that either carries a date or does not.
func TestShippedReviewPolicyCarriesNoAdoptionDate(t *testing.T) {
	raw, err := fs.ReadFile(scaffold.Dist(), ".pose/policy/review.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("the shipped review policy is not valid JSON: %v", err)
	}
	for key, value := range doc {
		if !reviewPolicyDatedKey(key) {
			continue
		}
		if text, ok := value.(string); ok && text != "" {
			t.Errorf("the shipped template carries %s=%s, a date no installing project earned", key, text)
		}
	}
	// An overlay is an adoption decision. Shipping one adopts it for every
	// project that installs, which is the thing explicit adoption exists to stop.
	if overlays, ok := doc["overlay_profiles"].([]any); !ok || len(overlays) != 0 {
		t.Errorf("the shipped template adopts overlay profiles for the target: %v", doc["overlay_profiles"])
	}
	// It still has to be the contract shape, or a fresh instance silently starts
	// on the legacy path.
	for key, want := range map[string]any{"enabled": true, "component_aware": true, "review_bundles": true, "schema_version": float64(2)} {
		if doc[key] != want {
			t.Errorf("the shipped template changed the contract shape: %s=%v, want %v", key, doc[key], want)
		}
	}
}

// An instance that already recorded its dates keeps them: update is additive,
// and the stamp never overwrites a decision the instance already made.
func TestUpdateKeepsRecordedAdoption(t *testing.T) {
	root, _ := installFreshInstance(t)
	path := filepath.Join(root, ".pose/policy/review.json")
	doc := readInstalledReviewPolicy(t, root)
	doc["explicit_judgment_adopted_at"] = "2026-01-02"
	doc["component_aware_adopted_at"] = "2026-01-03"
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	inDir(t, root, func() {
		if code := Main([]string{"update", "--no-self"}, &out, &errOut); code != 0 {
			t.Fatalf("update code=%d err=%s", code, errOut.String())
		}
	})
	after := readInstalledReviewPolicy(t, root)
	if got, _ := after["explicit_judgment_adopted_at"].(string); got != "2026-01-02" {
		t.Errorf("update rewrote a recorded adoption: explicit_judgment_adopted_at=%q", got)
	}
	if got, _ := after["component_aware_adopted_at"].(string); got != "2026-01-03" {
		t.Errorf("update rewrote a recorded adoption: component_aware_adopted_at=%q", got)
	}

	// `--force` is the dangerous one: it resets managed manuals wholesale, and
	// if it reached the policy the same way, an instance would silently lose the
	// dates and overlays it chose and receive the template's instead. Policy is
	// seeded absent-only regardless of force, and that is asserted here rather
	// than read off the code, because the blast radius is every adopter.
	after["overlay_profiles"] = []string{"backend-review@1"}
	raw, err = json.MarshalIndent(after, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"install", root, "--force", "--skip-mcp"}, &out, &errOut); code != 0 {
		t.Fatalf("install --force code=%d err=%s", code, errOut.String())
	}
	forced := readInstalledReviewPolicy(t, root)
	if got, _ := forced["explicit_judgment_adopted_at"].(string); got != "2026-01-02" {
		t.Errorf("install --force reset a recorded adoption: explicit_judgment_adopted_at=%q", got)
	}
	overlays, _ := forced["overlay_profiles"].([]any)
	if len(overlays) != 1 {
		t.Errorf("install --force reset the instance's adopted overlays: %v", forced["overlay_profiles"])
	}
}
