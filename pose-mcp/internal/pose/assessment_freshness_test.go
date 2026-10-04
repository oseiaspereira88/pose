package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-adaptive-assessment-freshness.

func freshnessFixture(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("svc/main.go", "package main\n\nfunc main() {}\n")
	write("other/README.md", "# other\n")
	write(".pose/specs/2026-10-04-work.md", "---\nslug: work\nstatus: in-progress\ncomponents: svc\n---\n\n# Spec: work\n")
	git := func(args ...string) {
		if out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=f", "-c", "user.email=f@example.invalid"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "fixture")
	return Store{Root: root}
}

func commitAll(t *testing.T, root, message string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=f", "-c", "user.email=f@example.invalid", "commit", "-qm", message}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

func TestAssessmentFreshnessFollowsTheBindingNotTheClock(t *testing.T) {
	s := freshnessFixture(t)
	if f := s.ComponentAssessmentFreshness("svc"); f.State != AssessmentMissing {
		t.Fatalf("no assessment yet: %+v", f)
	}
	state, err := s.DiscoverComponent("svc")
	if err != nil || state.Binding == nil {
		t.Fatalf("discovery did not bind: %+v %v", state, err)
	}
	if err := s.SaveComponentState(state); err != nil {
		t.Fatal(err)
	}
	commitAll(t, s.Root, "assessment")
	if f := s.ComponentAssessmentFreshness("svc"); f.State != AssessmentFresh {
		t.Fatalf("an unchanged component must be reused: %+v", f)
	}
	// A change in an unrelated area keeps it fresh.
	_ = os.WriteFile(filepath.Join(s.Root, "other/README.md"), []byte("# other, edited\n"), 0o644)
	commitAll(t, s.Root, "unrelated")
	if f := s.ComponentAssessmentFreshness("svc"); f.State != AssessmentFresh {
		t.Fatalf("an unrelated change staled the assessment: %+v", f)
	}
	_ = os.WriteFile(filepath.Join(s.Root, "svc/main.go"), []byte("package main\n\nfunc main() { println() }\n"), 0o644)
	if f := s.ComponentAssessmentFreshness("svc"); f.State != AssessmentStale || !strings.Contains(strings.Join(f.Reasons, ";"), "uncommitted") {
		t.Fatalf("an uncommitted change must stale it: %+v", f)
	}
	commitAll(t, s.Root, "change")
	if f := s.ComponentAssessmentFreshness("svc"); f.State != AssessmentStale || !strings.Contains(strings.Join(f.Reasons, ";"), "content changed") {
		t.Fatalf("a committed content change must stale it: %+v", f)
	}
}

func TestAStaleAssessmentIsAnAdvisoryObligation(t *testing.T) {
	s := freshnessFixture(t)
	r, err := s.ProjectObligations(ObligationQuery{Category: ObligationEvidence})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Obligations) != 1 || r.Obligations[0].ReasonCode != "assessment-missing" || r.Obligations[0].Restricts(PhaseExecution) {
		t.Fatalf("a missing assessment of a declared component: %+v", r.Obligations)
	}
}
