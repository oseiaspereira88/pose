package cli

import (
	"bytes"
	"strings"
	"testing"
)

// Spec pose-index-recovery-and-close-scope-consistency.

// R1: the plan modes of `pose close` are a spec's; for any other scope the refusal says so, says how
// that scope does close, and never proposes the invocation that was just refused.
func TestClosePlanModesOnANonSpecScopeNameTheRightInvocation(t *testing.T) {
	for _, tc := range []struct{ ref, kind string }{
		{"milestone:pose-shared-runtime/optional-worktree-runtime", "milestone"},
		{"roadmap:pose-shared-runtime", "roadmap"},
	} {
		for _, mode := range []string{"--plan", "--apply", "--resume"} {
			var out, errb bytes.Buffer
			code := cmdClose(t.TempDir(), []string{tc.ref, mode}, &out, &errb)
			msg := errb.String() + out.String()
			if code != 2 {
				t.Errorf("%s %s exited %d, want the usage error 2: %s", tc.ref, mode, code, msg)
			}
			for _, want := range []string{mode + " is the closeout plan of a spec", "a " + tc.kind + " scope closes with `pose close " + tc.ref + "`", "`pose closeout-check " + tc.ref + "`"} {
				if !strings.Contains(msg, want) {
					t.Errorf("%s %s: the refusal lacks %q: %s", tc.ref, mode, want, msg)
				}
			}
			if strings.Contains(msg, "pose close "+tc.ref+" "+mode) || strings.Contains(msg, "use `pose close "+tc.ref+"` for") {
				t.Errorf("%s %s: the refusal proposes the invocation it refused: %s", tc.ref, mode, msg)
			}
		}
	}
}
