package pose

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// B30's generation, policy, result and history readers must all consume the
// operation's view. A native reader in any one axis would incorrectly call this
// virtually updated index fresh or silently ignore a newly staged history file.
func TestUpdateOperationFactsReadProjectedPolicyResultsAndHistory(t *testing.T) {
	store, _ := factsWithResultsFile(t, []DeliveryValidationResult{{ID: "api/test", Module: "api", Check: "test", EvidenceClass: "unit", Outcome: "pass"}})
	p := NewPreviewFiles(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	projected := Store{Root: store.Root, FS: p}
	if f := statusOf(t, projected); f.Status != SpecFactsFresh {
		t.Fatalf("fixture: %+v", f)
	}
	graph, err := projected.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, filepath.FromSlash(factsResultsPath))
	raw, err := p.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var run map[string]any
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	run["git_head"] = "projected-head"
	changed, _ := json.Marshal(run)
	if err := p.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if reason := projected.resultsFileDisagrees(graph); !strings.Contains(reason, "written again") {
		t.Fatalf("projected results lost: %s", reason)
	}
	if reason := store.resultsFileDisagrees(graph); reason != "" {
		t.Fatalf("physical results changed: %s", reason)
	}
	// Point the staged policy at another staged result. The error must name it.
	policy := filepath.Join(store.Root, ".pose/policy/delivery.json")
	if err := p.WriteFile(policy, []byte(`{"schema_version":1,"enabled":true,"results_path":".pose/results/projected.json","roots":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteFile(filepath.Join(store.Root, ".pose/results/projected.json"), changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if reason := projected.resultsFileDisagrees(graph); !strings.Contains(reason, "projected.json") {
		t.Fatalf("projected policy lost: %s", reason)
	}
	history := filepath.Join(store.Root, ".pose/reports/history/new.jsonl")
	if err := p.MkdirAll(filepath.Dir(history), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteFile(history, []byte("{\"spec\":\"alpha\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if found, err := hasExplicitAttributionIn(p, store.Root, "alpha"); err != nil || !found {
		t.Fatalf("projected history lost: %v %v", found, err)
	}
	if found, err := hasExplicitAttribution(store.Root, "alpha"); err != nil || found {
		t.Fatalf("physical history changed: %v %v", found, err)
	}
	if f := statusOf(t, projected); f.Status == SpecFactsFresh {
		t.Fatalf("projected bindings incorrectly fresh: %+v", f)
	}
}
