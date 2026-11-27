package cli

import (
	"bytes"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-declared-threat-model R4: the ready check refuses an
// accepted limit without its reason, and one an ADR citing the spec omits.
func TestReadyCheckJudgesAcceptedLimits(t *testing.T) {
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
	spec := func(limits string) string {
		return "---\nslug: limited\nstatus: draft\ncreated_at: 2026-11-25\n---\n\n# Spec: limited\n\n## 1. Intent\n\n### Goal\nA goal worth stating.\n\n## 2. Requirements\n\n### Functional\n- R1: It works.\n\n## 3. Technical Plan\n\n### Affected areas\nThe engine.\n" + limits + "\n## 4. Tasks\n- [ ] Do it.\n"
	}
	path := filepath.Join(root, ".pose/specs/limited/spec.md")
	run := func() (int, string) {
		var out, errOut bytes.Buffer
		code := lintOneSpec(path, false, true, &out, &errOut)
		return code, out.String() + errOut.String()
	}
	write(".pose/specs/limited/spec.md", spec("\n### Accepted limits\n- L1: an escape. Accepted because it is rare.\n"))
	if code, out := run(); code != 0 {
		t.Fatalf("a justified limit failed the ready check: %d %s", code, out)
	}
	// Spec pose-review-declared-limits-followups, review round 1, N1: the
	// ready check reads the raw spec, as the plan does, so a comment in the
	// section fails it instead of passing a section the plan refuses.
	write(".pose/specs/limited/spec.md", spec("\n### Accepted limits\n- L1: an escape. Accepted because <!-- x --> it is rare.\n"))
	if code, out := run(); code == 0 || !strings.Contains(out, "comment marker") {
		t.Fatalf("a comment in the limits passed the ready check: %d %s", code, out)
	}
	write(".pose/specs/limited/spec.md", spec("\n### Accepted limits\n- L1: an escape with no reason.\n"))
	if code, out := run(); code == 0 || !strings.Contains(out, "does not say why it is accepted") {
		t.Fatalf("an unjustified limit passed: %d %s", code, out)
	}
	write(".pose/specs/limited/spec.md", spec("\n### Accepted limits\n- L1: an escape. Accepted because it is rare.\n- L2: another escape. Accepted because it is rarer.\n"))
	// Review round 1, N3: L10 does not name L1.
	write(".pose/adr/2026-11-25-limited.md", "# ADR\n\nFor `limited`. L10 and L2 are accepted.\n")
	if code, out := run(); code == 0 || !strings.Contains(out, "accepted limit L1 is not named by the ADR") {
		t.Fatalf("L10 was taken as naming L1: %d %s", code, out)
	}
	write(".pose/adr/2026-11-25-limited.md", "# ADR\n\nFor `limited`. L1 is accepted.\n")
	if code, out := run(); code == 0 || !strings.Contains(out, "accepted limit L2 is not named by the ADR") {
		t.Fatalf("a limit the ADR omits passed: %d %s", code, out)
	}
	// Review round 5, N2: an ADR that cannot be read fails the check closed.
	if os.Geteuid() != 0 {
		adr := filepath.Join(root, ".pose/adr/2026-11-25-limited.md")
		if err := os.Chmod(adr, 0o000); err != nil {
			t.Fatal(err)
		}
		if code, out := run(); code == 0 || !strings.Contains(out, "cannot be read to check the accepted limits") {
			t.Fatalf("an unreadable ADR was read as absent: %d %s", code, out)
		}
		if err := os.Chmod(adr, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Review round 8, N4: an ADR about a longer slug does not cite this one.
	write(".pose/adr/2026-11-25-limited.md", "# ADR\n\nFor `limited-other`. Nothing about this spec.\n")
	if code, out := run(); code != 0 {
		t.Fatalf("an ADR citing a longer slug was taken to cite this spec: %d %s", code, out)
	}
	// Review round 9, N3: an ADR that links the spec's dated file cites it.
	write(".pose/adr/2026-11-25-limited.md", "# ADR\n\nSee [the spec](../specs/2026-11-25-limited.md).\n")
	if code, out := run(); code == 0 || !strings.Contains(out, "L1") {
		t.Fatalf("an ADR linking the spec file was not taken to cite it: %d %s", code, out)
	}
	// Review round 7, N2: an ADR that is a link is not followed.
	adr := filepath.Join(root, ".pose/adr/2026-11-25-limited.md")
	_ = os.Chmod(adr, 0o644)
	_ = os.Remove(adr)
	if err := os.Symlink("/dev/zero", adr); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var code int
	var out string
	go func() { code, out = run(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the ready check followed an ADR link to an endless source")
	}
	if code == 0 || !strings.Contains(out, "cannot be read to check the accepted limits") {
		t.Fatalf("an ADR link was read: %d %s", code, out)
	}
}

// Spec pose-review-declared-limits-followups R3, R6: a within-limit finding
// can be typed with its limit, and the checks' text output shows it.
func TestWithinLimitFindingsAreTypedAndShown(t *testing.T) {
	findings, err := parseCLIReviewFindings([]string{"n1|low|within-limit|-|-||why it is inside||L1"})
	if err != nil || len(findings) != 1 || findings[0].Limit != "L1" || findings[0].Rationale != "why it is inside" {
		t.Fatalf("a typed within-limit finding: %+v %v", findings, err)
	}
	var out bytes.Buffer
	renderWithinLimitFindings(render(&out, &out), "review", &posemodel.ReviewAttempt{Findings: []posemodel.ReviewFinding{{ID: "N1", Severity: "low", Disposition: "within-limit", Limit: "L1", Rationale: "inside"}, {ID: "N2", Disposition: "accepted-risk"}}})
	if !strings.Contains(out.String(), "N1 limit:L1") || strings.Contains(out.String(), "N2") {
		t.Fatalf("the text output: %q", out.String())
	}
}
