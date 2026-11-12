package pose

import (
	"errors"
	"strings"
	"testing"
)

func TestDelegationCaptureFailureWithholdsPrivateErrorDetails(t *testing.T) {
	const private = "/private/synthetic-session/credential-value"
	for _, stage := range []string{"session/interrupt recognition", "argv", "stdout", "final output", "normalized events", "stderr"} {
		got := delegationCaptureFailure("worker", "claude-stream-json", stage, errors.New("open "+private+": denied"))
		if strings.Contains(got, private) || !strings.Contains(got, "private error details withheld") || !strings.Contains(got, stage) {
			t.Fatalf("unsafe or unhelpful error: %s", got)
		}
	}
	for _, reason := range []string{"conflicting adapter capture channels", "adapter changed the captured session", "redacted capture exceeds byte budget"} {
		got := delegationCaptureFailure("worker", "codex-jsonl", "stdout", errors.New(reason))
		if !strings.Contains(got, reason) || strings.Contains(got, "system/init") {
			t.Fatalf("wrong diagnostic: %s", got)
		}
	}
}
