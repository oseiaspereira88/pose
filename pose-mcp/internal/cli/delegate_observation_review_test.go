package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// B03 of the first review: a run whose record cannot be published keeps its
// marker, the one trace of the attempt it reserved.
func TestARunWithoutARecordKeepsItsMarker(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	saved := posemodel.PublishFault
	defer func() { posemodel.PublishFault = saved }()
	posemodel.PublishFault = func(path string) error {
		if strings.Contains(filepath.ToSlash(path), "/.pose/review-runs/") && strings.HasSuffix(path, ".json") {
			return os.ErrPermission
		}
		return nil
	}
	runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "unpublished")
	if active, _, _ := (posemodel.Store{Root: root}).ListDelegationActive(); len(active) != 1 {
		t.Fatalf("the marker went with the record unpublished: %+v", active)
	}
}

// B07 of the first review: wait never answers with a run's state over a
// history it could not read whole.
func TestWaitReportsAnIncompleteHistory(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "witnessed"); code != 0 {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	os.Remove(filepath.Join(root, ".pose", "review-run-ledger", runs[0].RunID+".json"))
	if code, out, _ := runPoseSplit(t, root, "delegate", "wait", "witnessed", "--json"); code != 1 || !strings.Contains(out, "could not be read whole") {
		t.Fatalf("wait over a lost witness: %d %s", code, out)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "watch", "witnessed", "--no-follow"); code != 1 || !strings.Contains(errOut, "incomplete") {
		t.Fatalf("watch over a lost witness: %d %s", code, errOut)
	}
}

// B01 and B08 of the first review: the hand-back is a file, not an inherited
// descriptor, and a detached run that does not reserve within the bound is
// stopped, not left to run unannounced.
func TestADetachedRunThatDoesNotHandBackIsStopped(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, slowApprovingAgent)
	wrapper := filepath.Join(t.TempDir(), "delayed-pose")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nsleep 4\nexec "+shellQuote(poseBinary(t))+" \"$@\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	savedExe, savedBound := delegateExecutable, delegateDetachBound
	defer func() { delegateExecutable, delegateDetachBound = savedExe, savedBound }()
	delegateExecutable = func() (string, error) { return wrapper, nil }
	delegateDetachBound = time.Second
	code, out, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--detach", "--json")
	if code != 1 || !strings.Contains(out, "was stopped") {
		t.Fatalf("a hand-back past the bound: %d %s", code, out)
	}
	time.Sleep(6 * time.Second)
	if attempts, _, _ := (posemodel.Store{Root: root}).ListDelegationAttempts(); len(attempts) != 0 {
		t.Fatalf("the stopped run reserved an attempt later: %+v", attempts)
	}
}

// F03 and F04 of the first review: the preview carries -v, and ls reads a
// running run's progress from its live events.
func TestPreviewCarriesVerboseAndLsReadsLiveProgress(t *testing.T) {
	dir := t.TempDir()
	argsPath, fake := filepath.Join(dir, "args"), filepath.Join(dir, "fzf")
	os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > "+shellQuote(argsPath)+"\nhead -n 1\n"), 0o700)
	saved := delegatePicker
	delegatePicker = fake
	defer func() { delegatePicker = saved }()
	var errOut bytes.Buffer
	pickAttempt(dir, []posemodel.DelegationAttempt{{AttemptID: "att-preview", State: "completed"}}, delegateFlags{verbose: true, lines: 3}, &errOut)
	if raw, _ := os.ReadFile(argsPath); !strings.Contains(string(raw), " -v") {
		t.Fatalf("the preview lost -v: %s", raw)
	}
	root := dispatchInstance(t)
	writeDelegation(t, root, streamingAgent)
	code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--json")
	var detached struct{ Attempt string }
	if code != 0 || json.Unmarshal([]byte(out), &detached) != nil {
		t.Fatalf("detach: %d %s", code, out)
	}
	time.Sleep(1500 * time.Millisecond)
	_, ls := runBuilt(t, root, "delegate", "ls", detached.Attempt, "--json")
	runBuilt(t, root, "delegate", "wait", detached.Attempt)
	if !strings.Contains(ls, `"turns": 1`) || !strings.Contains(ls, `"last_step": "message: looking at the bundle"`) {
		t.Fatalf("ls of a running run: %s", ls)
	}
}

// B08 of the second review: a detached run that cannot hand its attempt back
// does not start its agent, and one stopped after the bound records its run
// instead of leaving the agent running.
func TestADetachedRunThatCannotHandBackDoesNotRun(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, slowApprovingAgent)
	// The hand-back file already exists, so its exclusive publication fails.
	dir, err := posemodel.Store{Root: root}.DelegationPrivateDir("supervisors")
	if err != nil {
		t.Fatal(err)
	}
	handback := filepath.Join(dir, "occupied.handback")
	os.WriteFile(handback, []byte("{}\n"), 0o600)
	cmd := exec.Command(poseBinary(t), "delegate", "review", "spec:alpha")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), delegateHandback+"="+handback)
	out, _ := cmd.CombinedOutput()
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Status != posemodel.ReviewRunFailed || !strings.Contains(runs[0].Failure, "could not be handed back") {
		t.Fatalf("a run that could not hand back: %+v %s", runs, out)
	}
}
