package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-followup-reconciliation-candidates.
func TestCandidatesAreRaisedWithReasonsAndNothingIsDispositioned(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/2026-10-01-target.md", "---\nslug: target-spec\nstatus: done\n---\n\n# Spec: target\n")
	spec := "---\nslug: origin\nstatus: done\n---\n\n# Spec: origin\n\n## 7. Final Report\n\n### Follow-ups\n\n" +
		"- [open] Fold this into `target-spec` once it lands (owner:@core crit:low review:2099-01-01)\n" +
		"- [open] Harden TestExistingThing against flakes (owner:@core crit:low review:2099-01-01)\n" +
		"- [open] Revisit the cache (owner:@core crit:low review:2020-01-01)\n" +
		"- [open] Nothing to reconcile here (owner:@core crit:low review:2099-01-01)\n"
	write(".pose/specs/2026-10-02-origin.md", spec)
	write("pkg/thing_test.go", "package pkg\n\nimport \"testing\"\n\nfunc TestExistingThing(t *testing.T) {}\n")
	candidates, err := Store{Root: root}.FollowupCandidates("2026-10-04")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[int]string{}
	for _, c := range candidates {
		kinds[c.Ordinal] = strings.Join(c.Kinds, ",")
		if c.Limits == "" || len(c.Evidence) == 0 {
			t.Fatalf("a candidate without its reason or limits: %+v", c)
		}
	}
	if kinds[1] != CandidateTargetTerminal || kinds[2] != CandidateEvidencePresent || kinds[3] != CandidateOverdue || kinds[4] != "" {
		t.Fatalf("candidate kinds: %+v", kinds)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".pose/specs/2026-10-02-origin.md"))
	if string(raw) != spec {
		t.Fatal("raising candidates changed a disposition")
	}
	r, _ := Store{Root: root}.ProjectObligations(ObligationQuery{Category: ObligationResidualDebt})
	if len(r.Obligations) != 4 {
		t.Fatalf("an open follow-up disappeared from the projection: %+v", r.Obligations)
	}
	flagged := 0
	for _, o := range r.Obligations {
		if strings.Contains(o.Message, "reconciliation candidate") {
			flagged++
		}
	}
	if flagged != 3 {
		t.Fatalf("Attention should show the reason on the three candidates: %d", flagged)
	}
}
