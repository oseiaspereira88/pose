package pose

import (
	"testing"
	"time"
)

// Spec pose-governance-wait-rework-observability.
func TestWaitsSeparateAgeAttributedWaitAndKnownBlocking(t *testing.T) {
	s := actionFixture(t)
	t0 := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	whole := func(question string) ActionRequest {
		r := decisionRequest()
		r.Question = question
		r.Effects = []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock}}
		return r
	}
	// Two simultaneous whole-spec restrictions overlapping for an hour, and
	// one scoped restriction that does not stop the rest of the spec.
	a, err := s.OpenActionRequest(whole("A?"), t0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.OpenActionRequest(whole("B?"), t0.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenActionRequest(decisionRequest(), t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveActionRequest(answer(a, "human:maintainer", "preserve-v1", "ka"), t0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveActionRequest(answer(b, "human:maintainer", "preserve-v1", "kb"), t0.Add(90*time.Minute)); err != nil {
		t.Fatal(err)
	}
	report, err := s.GovernanceWaits(t0.Add(2 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if report.Requests != 3 || report.Resolved != 2 || report.Open != 1 {
		t.Fatalf("population: %+v", report)
	}
	// Overlap counts once: 12:00–13:30, not 60+60 minutes.
	if report.KnownBlockingSeconds != 90*60 {
		t.Fatalf("simultaneous restrictions were summed: %v", report.KnownBlockingSeconds)
	}
	// Attributed waiting includes the scoped and the open request: 60+60+120 min.
	if report.ByCause["actor"] != 240*60 {
		t.Fatalf("attributed wait: %+v", report.ByCause)
	}
	if report.AgeSeconds.Count != 3 || report.AgeSeconds.Max != 120*60 {
		t.Fatalf("age: %+v", report.AgeSeconds)
	}
	if len(report.Limitations) == 0 {
		t.Fatal("the report does not say what it does not measure")
	}
}

func TestReworkClassifiesByWhatChangedAndKeepsUnknown(t *testing.T) {
	root, store := reviewBundleFixture(t)
	if _, err := store.SealReviewBundle("spec:backend", time.Now()); err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return false }\n")
	if _, err := store.SealReviewBundle("spec:backend", time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	report, err := store.GovernanceRework()
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || len(report.Limitations) == 0 || report.Bundles != 2 {
		t.Fatalf("report: %+v", report)
	}
	if report.BundleSupersessions["subject-changed"] != 1 {
		t.Fatalf("a code change was not classified as subject-changed: %+v", report.BundleSupersessions)
	}
	for cause := range report.BundleSupersessions {
		switch cause {
		case "subject-changed", "intent-or-plan-changed", "review-plan-changed", "evidence-changed", "unknown":
		default:
			t.Fatalf("a cause outside the evidence-backed set: %s", cause)
		}
	}
}
