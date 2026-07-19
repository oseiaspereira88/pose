package cli

// Harne8 control-plane composition (spec pose-harne8-control-plane-integration):
// Compatibility — the open core completes local governed workflows fully
// when every Harne8 component (Conductor, Harness, GraphForge, Portal) is
// absent. Every env var those components would configure is deliberately
// unset for this test.

import (
	"bytes"
	"os"
	"testing"
)

func TestOpenCoreCompletesLocalWorkflowsWithoutHarne8(t *testing.T) {
	for _, key := range []string{
		"CONDUCTOR_URL", "CONDUCTOR_PROJECT_ID", "CONDUCTOR_RUN_TOKEN",
		"POSE_MCP_IDENTITY_SECRET", "POSE_MCP_OPA_URL", "HARNE8_PROJECTS_DIR",
		"POSE_PROJECT_ROOTS", "POSE_OTEL_ENABLED", "OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		old, had := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		if had {
			t.Cleanup(func() { _ = os.Setenv(key, old) })
		}
	}

	root := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}

	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"new-spec", "local-only-feature"}, &out, &errB); code != 0 {
			t.Fatalf("new-spec exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		// Lint-spec on a freshly-scaffolded, still-placeholder spec is
		// expected to report unfilled sections — that's the DoR gate
		// working correctly, not a Harne8-related failure, so it's
		// exercised separately from the "must succeed" step list below.
		out.Reset()
		errB.Reset()
		_ = Main([]string{"lint-spec", "local-only-feature"}, &out, &errB)

		steps := [][]string{
			{"check", "--strict"},
			{"validate", "--tolerant"},
			{"followups", "--all"},
			{"doctor"},
			{"portfolio-projection"},
		}
		for _, step := range steps {
			out.Reset()
			errB.Reset()
			if code := Main(step, &out, &errB); code != 0 {
				t.Fatalf("%v exit=%d out=%s err=%s (open core must complete every local workflow without Harne8)", step, code, out.String(), errB.String())
			}
		}
	})
}
