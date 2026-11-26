package cli

import (
	"bytes"
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
