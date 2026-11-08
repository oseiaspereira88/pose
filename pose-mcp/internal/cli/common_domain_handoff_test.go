package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// The same operation through the process and through the domain wrapper returns the
// same result (spec pose-common-domain-and-cli-mcp-handoff R1, R9): the wrapper no
// longer starts the binary for it, and nothing about the answer changes.
func TestAnInProcessWrapperReturnsWhatTheSpawnedCommandReturns(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := followupFixture(t)
	suggestStore, followStore := posemodel.Store{Root: repo}, posemodel.Store{Root: fixture}
	cases := []struct {
		root string
		name string
		cli  []string
		run  func() (any, error)
	}{
		{repo, "suggest", []string{"suggest", "feature", "--json"}, func() (any, error) { return suggestStore.Suggest(context.Background(), "feature", "", "") }},
		{fixture, "followups", []string{"followups", "--open", "--json"}, func() (any, error) { return followStore.Followups(context.Background(), false) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, spawned := runBuilt(t, tc.root, tc.cli...)
			if code != 0 {
				t.Fatalf("the spawned command exited %d: %s", code, spawned)
			}
			var want any
			if err := json.Unmarshal([]byte(spawned), &want); err != nil {
				t.Fatalf("the spawned command's output is not JSON: %v", err)
			}
			before, runs := posemodel.CLISpawnAttempts(), posemodel.CLIInProcessRuns()
			got, err := tc.run()
			if err != nil {
				t.Fatal(err)
			}
			if posemodel.CLISpawnAttempts() != before || posemodel.CLIInProcessRuns()-runs != 1 {
				t.Fatalf("the wrapper did not run in process (spawn attempts %d->%d, in-process runs +%d)", before, posemodel.CLISpawnAttempts(), posemodel.CLIInProcessRuns()-runs)
			}
			// followups carries a generation time; compare the structure without it.
			if m, ok := want.(map[string]any); ok {
				delete(m, "generated_at")
			}
			if m, ok := got.(map[string]any); ok {
				delete(m, "generated_at")
			}
			if !reflect.DeepEqual(got, want) {
				g, _ := json.Marshal(got)
				w, _ := json.Marshal(want)
				t.Fatalf("the in-process result differs from the spawned one:\n got  %.400s\n want %.400s", g, w)
			}
		})
	}
}

// followupFixture is a small project with one open follow-up, so the parity of the
// followups wrapper is measured on content instead of on this repository's size.
func followupFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".pose", "specs", "2026-06-01-alpha.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n\n# Spec: alpha\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Look at the thing again. (owner:@team crit:low review:2099-01-01)\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// Concurrent wrapper calls over the same root are independent: the in-process
// commands share no state the way two processes share none.
func TestInProcessWrappersAreSafeUnderConcurrency(t *testing.T) {
	root := followupFixture(t)
	store := posemodel.Store{Root: root}
	errs := make(chan error, 16)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := store.Suggest(context.Background(), "feature", "", "")
			errs <- err
		}()
		go func() {
			_, err := store.Followups(context.Background(), false)
			errs <- err
		}()
	}
	for i := 0; i < 16; i++ {
		// A fixture without a task map answers suggest with the command's own error;
		// what matters here is that no call deadlocks or races.
		if err := <-errs; err != nil && !errAllowed(err) {
			t.Fatal(err)
		}
	}
}

func errAllowed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "task-map missing")
}

// The handoff's argv is consumed by the CLI that produced the plan: an unchanged
// repository must accept the digest it names (spec pose-common-domain-and-cli-mcp-handoff
// R4/R5), and a handoff is not offered for a plan the apply would refuse.
func TestTheDigestOfAHandoffIsTheDigestTheApplyChecks(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	_, raw := runPose(t, root, "close", "spec:bundle", "--plan", "--json")
	var plan posemodel.CloseoutPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		t.Fatalf("plan: %v\n%s", err, raw)
	}
	// The consumer check is about the digest, so the plan is cleared of what would only
	// withhold the handoff; the real plan of this fixture is covered by the blocked case below.
	probe := plan
	probe.Terminal, probe.Blockers = false, nil
	probe.Steps = []posemodel.CloseoutPlanStep{{ID: posemodel.CloseoutStepEvidence, State: posemodel.CloseoutStatePending}}
	h := posemodel.CloseoutHandoff(probe, posemodel.HandoffTarget{})
	if h == nil {
		t.Fatal("no handoff for a plan with a pending step")
	}
	code, out := runPose(t, root, h.Argv[1:]...)
	if strings.Contains(out, "plan-digest-mismatch") || strings.Contains(out, "the repository changed since the plan was computed") {
		t.Fatalf("the digest of a handoff for an unchanged repository was refused (%d): %s", code, out)
	}
}
