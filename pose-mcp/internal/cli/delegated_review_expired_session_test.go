package cli

import (
	"encoding/json"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// expiredSessionPolicy is the agent of the round tests whose provider refuses
// every resume, as an expired session does.
func expiredSessionPolicy(t *testing.T) string {
	t.Helper()
	var policy map[string]any
	if err := json.Unmarshal([]byte(delegationPolicy(roundsAgent("rounder", "other", false))), &policy); err != nil {
		t.Fatal(err)
	}
	agent := policy["agents"].(map[string]any)["rounder"].(map[string]any)
	agent["resume"].(map[string]any)["command"] = []string{"sh", "-c", "cat >/dev/null; echo 'session expired: session not found' >&2; exit 1", "x", "{session}"}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestFailedContinuationOpensANewSession(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, expiredSessionPolicy(t))
	roundRun(t, root, "rounder")
	nextReviewRound(t, root, "change after rejection")
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "rounder", "--json", "--reason", "next round")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(code, out, errOut)
	}
	failed := runRecord(t, root, doc["run"].(string))
	if code != 1 || failed.Status != posemodel.ReviewRunFailed || failed.Thread == nil || failed.Thread.Mode != posemodel.ReviewThreadContinued {
		t.Fatalf("the failing attempt: %d %+v", code, failed.Thread)
	}
	next := roundRun(t, root, "rounder")
	run := runRecord(t, root, next["run"].(string))
	if run.Thread == nil || run.Thread.Mode != posemodel.ReviewThreadNew {
		t.Fatalf("the next attempt continued: %+v", run.Thread)
	}
	if !strings.Contains(run.Thread.Reason, failed.RunID) || !strings.Contains(run.Thread.Reason, "could not be continued") {
		t.Fatalf("the reason: %q", run.Thread.Reason)
	}
	if run.Status != posemodel.ReviewRunCompleted || run.Round != failed.Round {
		t.Fatalf("%s round %d, failed in round %d", run.Status, run.Round, failed.Round)
	}
}

// A continuation that works is not touched by a failure of another round.
func TestContinuationAcrossRoundsKeepsItsSession(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(roundsAgent("rounder", "other", false)))
	roundRun(t, root, "rounder")
	nextReviewRound(t, root, "control change")
	second := runRecord(t, root, roundRun(t, root, "rounder")["run"].(string))
	if second.Thread == nil || second.Thread.Mode != posemodel.ReviewThreadContinued {
		t.Fatalf("the control: %+v", second.Thread)
	}
}

// Only a run that failed while continuing counts (R2), for its own bundle.
func TestOnlyAFailedContinuationGivesWay(t *testing.T) {
	continued := &posemodel.ReviewThread{Mode: posemodel.ReviewThreadContinued}
	cold := &posemodel.ReviewThread{Mode: posemodel.ReviewThreadNew}
	for name, c := range map[string]struct {
		run  posemodel.ReviewRun
		gave bool
	}{
		"failed continuing": {posemodel.ReviewRun{BundleID: "b", Status: posemodel.ReviewRunFailed, Thread: continued}, true},
		"failed cold":       {posemodel.ReviewRun{BundleID: "b", Status: posemodel.ReviewRunFailed, Thread: cold}, false},
		"first round":       {posemodel.ReviewRun{BundleID: "b", Status: posemodel.ReviewRunFailed}, false},
		"interrupted":       {posemodel.ReviewRun{BundleID: "b", Status: posemodel.ReviewRunInterrupted, Thread: continued}, false},
		"completed":         {posemodel.ReviewRun{BundleID: "b", Status: posemodel.ReviewRunCompleted, Thread: continued}, false},
		"another bundle":    {posemodel.ReviewRun{BundleID: "other", Status: posemodel.ReviewRunFailed, Thread: continued}, false},
	} {
		if got := posemodel.FailedContinuation([]posemodel.ReviewRun{c.run}, "b") != nil; got != c.gave {
			t.Errorf("%s: %v", name, got)
		}
	}
}

func TestFailedContinuationReasonIsBounded(t *testing.T) {
	reason := posemodel.FailedContinuationReason(&posemodel.ReviewRun{RunID: "run-1", Failure: strings.Repeat("x", 4096)})
	if !strings.Contains(reason, "run-1") || len(reason) > 400 {
		t.Fatalf("%d bytes: %s", len(reason), reason[:80])
	}
}
