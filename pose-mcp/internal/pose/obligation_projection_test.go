package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-obligation-projection.

func obligationFixture(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"}}`)
	write(".pose/specs/2026-10-01-base.md", "---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n\n## 2. Requirements\n- R1: x\n")
	write(".pose/specs/2026-10-02-consumer.md", "---\nslug: consumer\nstatus: draft\ndepends_on: base\n---\n\n# Spec: consumer\n\n## 2. Requirements\n- R1: y\n")
	write(".pose/specs/2026-10-03-shipped.md", "---\nslug: shipped\nstatus: done\n---\n\n# Spec: shipped\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Revisit the cache key once real usage exists (owner:@core crit:low review:2027-01-01)\n- [done] Already handled.\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=f", "-c", "user.email=f@example.invalid", "commit", "-qm", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return Store{Root: root}
}

func treeHash(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && !strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			raw, _ := os.ReadFile(path)
			h.Write([]byte(path))
			h.Write(raw)
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

func findObligation(r ObligationReport, category, reason string) *Obligation {
	for i := range r.Obligations {
		if r.Obligations[i].Category == category && r.Obligations[i].ReasonCode == reason {
			return &r.Obligations[i]
		}
	}
	return nil
}

func TestProjectionIsReadOnlyAndNavigableToItsSource(t *testing.T) {
	s := obligationFixture(t)
	before := treeHash(t, s.Root)
	r, err := s.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if treeHash(t, s.Root) != before {
		t.Fatal("projecting obligations changed the tree")
	}
	dep := findObligation(r, ObligationDependency, "dependency-not-done")
	if dep == nil {
		t.Fatalf("the unmet prerequisite is not projected: %+v", r.Obligations)
	}
	if dep.Source.Producer != "readiness" || dep.Source.Ref.Artifact != "xref:"+r.Snapshot.Project+"/spec:consumer" || dep.Source.Detail != "base" || dep.Rule != "depends_on" {
		t.Fatalf("source not navigable: %+v", dep.Source)
	}
	if !dep.Restricts(PhaseStart) || dep.Restricts(PhaseCloseout) {
		t.Fatalf("a spec-level prerequisite restricts start and execution only: %+v", dep.Effects)
	}
	debt := findObligation(r, ObligationResidualDebt, "followup-open")
	if debt == nil || debt.Recipient.Principal != "@core" || debt.Restricts(PhaseExecution) || debt.Restricts(PhaseCloseout) {
		t.Fatalf("an open follow-up must be advisory residual debt with its owner: %+v", debt)
	}
	if r.Counts[ObligationResidualDebt] != 1 {
		t.Fatalf("only open follow-ups are projected: %+v", r.Counts)
	}
}

func TestResolvingTheSourceRemovesTheObligationWithoutAnotherWrite(t *testing.T) {
	s := obligationFixture(t)
	path := filepath.Join(s.Root, ".pose/specs/2026-10-01-base.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "status: draft", "status: done", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := s.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if findObligation(r, ObligationDependency, "dependency-not-done") != nil {
		t.Fatal("the dependency is done at its source and still projected")
	}
	if !r.Snapshot.WorktreeDirty || len(r.Snapshot.Limitations) == 0 {
		t.Fatalf("a working tree that differs from the commit is not identified: %+v", r.Snapshot)
	}
}

func TestAFailingProducerIsCoverageNotZeroObligations(t *testing.T) {
	s := obligationFixture(t)
	// An unreadable review policy makes the closeout producer fail for an
	// in-progress spec; readiness keeps working.
	consumer := filepath.Join(s.Root, ".pose/specs/2026-10-02-consumer.md")
	raw, _ := os.ReadFile(consumer)
	_ = os.WriteFile(consumer, []byte(strings.Replace(string(raw), "status: draft", "status: in-progress", 1)), 0o644)
	_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), []byte("{not json"), 0o644)
	r, err := s.ProjectObligations(ObligationQuery{Category: ObligationJudgment})
	if err != nil {
		t.Fatal(err)
	}
	if r.Complete {
		t.Fatal("a failed producer reported a complete answer")
	}
	var closeout ProducerCoverage
	for _, c := range r.Coverage {
		if c.Producer == "closeout" {
			closeout = c
		}
	}
	if closeout.State != CoverageStateUnavailable || closeout.Detail == "" {
		t.Fatalf("closeout failure not in coverage: %+v", r.Coverage)
	}
	if len(r.Obligations) != 0 {
		t.Fatalf("filtered judgment list should be empty here: %+v", r.Obligations)
	}
}

func TestUnintegratedProducersAreAlwaysListed(t *testing.T) {
	r, err := obligationFixture(t).ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]string{}
	for _, c := range r.Coverage {
		listed[c.Producer] = c.State
	}
	for producer := range obligationProducersPending {
		if listed[producer] != CoverageStateUnsupported {
			t.Errorf("%s not listed as unsupported: %v", producer, listed)
		}
	}
	if r.Complete {
		t.Fatal("an answer with unsupported producers is not complete")
	}
}

func TestCorrelationKeepsEveryDistinctRestriction(t *testing.T) {
	a := sampleObligation(ObligationDependency)
	b := a
	b.Source.Producer = "closeout"
	b.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
	merged := correlateObligations([]Obligation{a, b})
	if len(merged) != 1 || len(merged[0].Effects) != 2 || len(merged[0].CorrelatedWith) != 1 || merged[0].CorrelatedWith[0] != "closeout" {
		t.Fatalf("correlation dropped a restriction: %+v", merged)
	}
}

func TestSnapshotChangesInvalidateAnEarlierAnswer(t *testing.T) {
	s := obligationFixture(t)
	snap := s.CurrentObligationSnapshot()
	if snap.SourceRevision == "" || snap.PolicyDigest == "" || snap.Digest == "" || !snap.Coherent {
		t.Fatalf("snapshot incomplete: %+v", snap)
	}
	if changed := s.ObligationSnapshotChanges(snap); len(changed) != 0 {
		t.Fatalf("nothing changed, yet: %v", changed)
	}
	_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), []byte(`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"}}`), 0o644)
	changed := s.ObligationSnapshotChanges(snap)
	if len(changed) == 0 || changed[0] != "policy" {
		t.Fatalf("a policy change was not detected: %v", changed)
	}
}

func TestQueryFiltersByScopeActorPhaseAndCategory(t *testing.T) {
	s := obligationFixture(t)
	scoped, err := s.ProjectObligations(ObligationQuery{Scope: "spec:shipped"})
	if err != nil {
		t.Fatal(err)
	}
	if findObligation(scoped, ObligationDependency, "dependency-not-done") != nil || findObligation(scoped, ObligationResidualDebt, "followup-open") == nil {
		t.Fatalf("scope filter: %+v", scoped.Obligations)
	}
	byActor, _ := s.ProjectObligations(ObligationQuery{Actor: "@core"})
	if len(byActor.Obligations) != 1 || byActor.Obligations[0].Category != ObligationResidualDebt {
		t.Fatalf("actor filter: %+v", byActor.Obligations)
	}
	byPhase, _ := s.ProjectObligations(ObligationQuery{Phase: PhaseStart})
	for _, o := range byPhase.Obligations {
		if !o.Restricts(PhaseStart) && len(o.Effects) == 0 {
			t.Fatalf("phase filter returned %+v", o)
		}
	}
	if _, err := s.ProjectObligations(ObligationQuery{Scope: "spec:nope"}); err == nil {
		t.Fatal("an unknown spec scope must be an error, not an empty answer")
	}
}
