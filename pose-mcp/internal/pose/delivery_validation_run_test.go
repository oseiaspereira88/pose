package pose

import "testing"

// A carried check is read as an observation of the head it ran at (spec
// pose-validate-carries-unchanged-checks R7); an executed one keeps the run's.
func TestDeliveryValidationRunReadsTheCarriedHead(t *testing.T) {
	raw := []byte(`{"schema_version":1,"generated_at":"2026-11-22T10:00:00Z","git_head":"bbbb","checks":[` +
		`{"id":"a/go/ok","module":"a","name":"ok","severity":"required","evidence_class":"unit","outcome":"pass"},` +
		`{"id":"b/go/ok","module":"b","name":"ok","severity":"required","evidence_class":"unit","outcome":"pass","carried_from":"aaaa","carried_generated_at":"2026-11-21T09:00:00Z"}]}`)
	results, err := ParseDeliveryValidationRun(raw, ".pose/results/delivery-validation.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	for _, r := range results {
		switch r.ID {
		case "a/go/ok":
			if r.GitHead != "bbbb" || r.GeneratedAt != "2026-11-22T10:00:00Z" {
				t.Fatalf("executed check read as %s at %s", r.GitHead, r.GeneratedAt)
			}
		case "b/go/ok":
			if r.GitHead != "aaaa" || r.GeneratedAt != "2026-11-21T09:00:00Z" {
				t.Fatalf("carried check read as %s at %s, want the head it ran at", r.GitHead, r.GeneratedAt)
			}
		}
	}
}
