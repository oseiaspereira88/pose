package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	write(".pose/specs/limited/spec.md", spec("\n### Accepted limits\n- L1: an escape. Accepted because it is rare.\n- L2: another. Accepted because it is rarer.\n"))
	write(".pose/adr/2026-11-25-limited.md", "# ADR\n\nFor `limited`. L1 is accepted.\n")
	if code, out := run(); code == 0 || !strings.Contains(out, "accepted limit L2 is not named by the ADR") {
		t.Fatalf("a limit the ADR omits passed: %d %s", code, out)
	}
}
