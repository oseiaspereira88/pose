package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCheckDeliveryGraphFailureReportedOnce(t *testing.T) {
	for _, mode := range []string{"strict", "tolerant"} {
		t.Run(mode, func(t *testing.T) {
			root := surfaceFixture(t)
			for i := 0; i < 3; i++ {
				slug := fmt.Sprintf("done-%d", i)
				writeArtifactTestFile(t, root, ".pose/specs/"+slug+"/spec.md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-08-03\ncompleted_at: 2026-08-04\n---\n# Done\n")
			}
			writeArtifactTestFile(t, root, ".pose/specs/broken/spec.md", "---\nslug: broken\nstatus: draft\ncreated_at: 2026-08-03\ndelivers: surface:missing\n---\n# Broken\n### Delivery targets\n- invalid target\n")
			var out, errOut bytes.Buffer
			checker := &nativeChecker{root: root, mode: mode, stdout: &out, out: render(&out, &errOut)}
			checker.checkDeliveryContracts()
			if checker.errors+checker.warnings != 1 {
				t.Fatalf("got %d errors and %d warnings, want one shared finding: %s%s", checker.errors, checker.warnings, &out, &errOut)
			}
			output := out.String() + errOut.String()
			if !strings.Contains(output, "spec:broken") || strings.Contains(output, "spec:done-") {
				t.Fatalf("finding must name responsible draft: %s", output)
			}
			if mode == "strict" && checker.errors != 1 || mode == "tolerant" && checker.warnings != 1 {
				t.Fatalf("mode severity changed: %+v", checker)
			}
		})
	}
}
