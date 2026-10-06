package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-capability-catalog.

func runAdopt(t *testing.T, repo string, args ...string) (int, string, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdAdopt(repo, args, &out, &errB)
	return code, out.String(), errB.String()
}

func TestAdoptCatalogTogglesEveryCapabilityKind(t *testing.T) {
	repo, _ := installedInstance(t)
	store := posemodel.Store{Root: repo}
	for _, id := range []string{"qualified-artifact-refs", "spec-authority-transfer", "overlay:engineering-judgment"} {
		if code, out, errB := runAdopt(t, repo, id, "--date", "2026-10-06", "--apply"); code != 0 {
			t.Fatalf("adopt %s: code=%d out=%s err=%s", id, code, out, errB)
		}
	}
	policy, err := store.GetReviewPolicy()
	if err != nil || policy.SpecAuthorityTransferVersion != 1 || policy.SchemaVersion != posemodel.SpecAuthorityTransferPolicySchemaVersion || policy.OverlayAdoptedAt["engineering-judgment@1"] != "2026-10-06" {
		t.Fatalf("catalog adoptions did not reach the policy: %+v err=%v", policy, err)
	}
	if code, _, errB := runAdopt(t, repo, "qualified-artifact-refs", "--off", "--apply"); code == 0 || !strings.Contains(errB, "spec-authority-transfer depends on qualified-artifact-refs") {
		t.Fatalf("turning off a required capability was not refused: code=%d err=%s", code, errB)
	}
	for _, id := range []string{"spec-authority-transfer", "qualified-artifact-refs"} {
		if code, _, errB := runAdopt(t, repo, id, "--off", "--apply"); code != 0 {
			t.Fatalf("off %s: %s", id, errB)
		}
	}
	if policy, _ := store.GetReviewPolicy(); policy.SchemaVersion != posemodel.ReviewPolicySchemaVersion {
		t.Fatalf("schema was not restored: %d", policy.SchemaVersion)
	}
	if code, _, errB := runAdopt(t, repo, "signed-attestations", "--apply"); code == 0 || !strings.Contains(errB, "trusted issuer") {
		t.Fatalf("an unmet prerequisite was not refused with what is missing: code=%d err=%s", code, errB)
	}
}

func TestAdoptCatalogRecordsDeclineAndDeferAndAdoptClearsThem(t *testing.T) {
	repo, _ := installedInstance(t)
	if code, _, errB := runAdopt(t, repo, "overlay:high-criticality-review", "--decline", "--apply"); code == 0 || !strings.Contains(errB, "--decline needs --reason") {
		t.Fatalf("a decline without a reason was accepted: code=%d err=%s", code, errB)
	}
	if code, _, errB := runAdopt(t, repo, "overlay:high-criticality-review", "--decline", "--reason", "no critical components", "--apply"); code != 0 {
		t.Fatalf("decline: %s", errB)
	}
	if code, _, errB := runAdopt(t, repo, "atomic-start", "--defer", "--reason", "x", "--apply"); code == 0 || !strings.Contains(errB, "atomic-start is on") {
		t.Fatalf("declaring an adopted capability deferred was not refused: code=%d err=%s", code, errB)
	}
	code, out, _ := runAdopt(t, repo, "--list", "--json")
	if code != 0 {
		t.Fatalf("list --json failed")
	}
	var states []posemodel.CapabilityState
	if err := json.Unmarshal([]byte(out), &states); err != nil {
		t.Fatal(err)
	}
	byID := map[string]posemodel.CapabilityState{}
	for _, state := range states {
		byID[state.ID] = state
	}
	if byID["overlay:high-criticality-review"].State != posemodel.CapabilityDeclined || byID["atomic-start"].State != posemodel.CapabilityOn || byID["definition-of-ready"].State != posemodel.CapabilityOn {
		t.Fatalf("states after install and decline: %+v", byID)
	}
	if code, _, errB := runAdopt(t, repo, "overlay:high-criticality-review", "--apply"); code != 0 {
		t.Fatalf("adopting a declined capability: %s", errB)
	}
	decisions, _ := posemodel.ReadAdoptionDecisions(repo)
	if _, still := decisions.Decisions["overlay:high-criticality-review"]; still {
		t.Fatal("adopting did not clear the recorded decline")
	}
	_, text, _ := runAdopt(t, repo, "--list")
	if !strings.Contains(text, "adopt.overlay:high-criticality-review=on") || !strings.Contains(text, "on in new instances") {
		t.Fatalf("--list does not render states:\n%s", text)
	}
}
