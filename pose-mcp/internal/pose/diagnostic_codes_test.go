package pose

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Spec pose-typed-producer-diagnostics.

// Every code a producer emits is in the catalog; a code typed only at its
// call site would be a second, undocumented vocabulary.
func TestEveryEmittedDiagnosticCodeIsCatalogued(t *testing.T) {
	emit := regexp.MustCompile(`(?:NewDiagnostic|OpaqueDiagnostic)\("([a-z0-9-]+)"`)
	files, _ := filepath.Glob("*.go")
	nextSteps := map[string]bool{"none": true, "continue-child": true, "resolve-federated-blockers": true, "record-fresh-review": true, "apply-lifecycle-transition": true, "resolve-closeout-blockers": true, "answer-action-request": true}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, _ := os.ReadFile(file)
		for _, m := range emit.FindAllStringSubmatch(string(raw), -1) {
			code := m[1]
			if _, ok := DiagnosticCatalog[code]; !ok && !nextSteps[code] && !strings.Contains(file, "obligation") {
				t.Errorf("%s emits uncatalogued diagnostic code %q", file, code)
			}
		}
	}
}

func TestReadinessWaitingRefsCarryTypedCodes(t *testing.T) {
	s := readinessStore(t)
	r, err := s.SpecReadiness("waiting")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]string{}
	for _, w := range r.WaitingOn {
		codes[w.Ref] = w.Code
		if w.Code == "" || w.Reason == "" {
			t.Errorf("waiting ref %q lacks code or reason: %+v", w.Ref, w)
		}
	}
	if codes["pending"] != "dependency-not-done" {
		t.Errorf("an in-progress dependency: %q", codes["pending"])
	}
	if codes["missing-spec"] != "dependency-unresolved" {
		t.Errorf("a missing dependency: %q", codes["missing-spec"])
	}
}

func TestJudgmentPendencyNamesItsSourceAndCondition(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}],
  "tools":[{"id":"review-check","requiredness":"required","criteria":["security"]}]
}`)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := store.PrepareReviewAttestation(bundle.BundleID, "agent:audit", now)
	if err != nil || len(prepared.Pending) != 1 {
		t.Fatalf("%v %+v", err, prepared.Pending)
	}
	p := prepared.Pending[0]
	if p.Code != "judgment-unanswered" || p.Source != bundle.BundleID || p.Condition == "" || p.Criterion != "security" {
		t.Fatalf("judgment pendency not typed: %+v", p)
	}
}

func TestCloseoutDiagnosticsTypeTheBlockersAndTheNextStep(t *testing.T) {
	_, store := reviewBundleFixture(t)
	state, err := store.GetCloseoutState("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]Diagnostic{}
	for _, d := range state.Diagnostics {
		codes[d.Code] = d
		if d.Domain == "" || d.Message == "" {
			t.Errorf("diagnostic without domain or message: %+v", d)
		}
	}
	lifecycle, ok := codes["lifecycle-not-done"]
	if !ok || len(lifecycle.Refs) != 1 || lifecycle.Refs[0] != "spec:backend" {
		t.Fatalf("lifecycle not typed: %+v", state.Diagnostics)
	}
	rendered := false
	for _, b := range state.Blockers {
		rendered = rendered || b == lifecycle.Message
	}
	if !rendered {
		t.Fatalf("the legacy string is not the typed item's rendering: %v", state.Blockers)
	}
	if _, ok := codes["review-not-approved"]; !ok {
		t.Fatalf("an unapproved required review is not typed: %+v", state.Diagnostics)
	}
	if state.NextStep == nil || state.NextStep.Code != "record-fresh-review" || state.NextStep.Message != state.NextAction {
		t.Fatalf("next step not typed: %+v (next_action %q)", state.NextStep, state.NextAction)
	}
	for _, d := range state.Diagnostics {
		if d.Code == "review-blocker" && (!d.Opaque || len(d.Refs) > 0) {
			t.Fatalf("a text review blocker gained structure it does not have: %+v", d)
		}
	}
}

func TestDiagnosticCodeDoesNotDependOnWording(t *testing.T) {
	a := NewDiagnostic("child-scope-open", "child spec:x is not closed", "spec:x")
	b := NewDiagnostic("child-scope-open", "o escopo filho spec:x não foi fechado", "spec:x")
	if a.Code != b.Code || a.Condition != b.Condition || strings.Join(a.Refs, ",") != strings.Join(b.Refs, ",") {
		t.Fatal("identity or condition changed with wording")
	}
}
