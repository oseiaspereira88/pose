package pose

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func continuityAdapter() ReviewerAdapter {
	return ReviewerAdapter{Command: []string{"fake"}, Vendor: "fixture", Model: "fixture", Limit: &ReviewerLimit{ExitCodes: []int{75}, ResetPattern: `reset: (\S+)`}, Session: &ReviewerSession{Pattern: `session: (\S+)`}, Resume: &ReviewerResume{Command: []string{"fake", "{session}"}}}
}
func TestRunContinuityClassificationAndWait(t *testing.T) {
	s := Store{Root: t.TempDir()}
	a := continuityAdapter()
	run := ReviewRun{ExitCode: 75}
	text := []byte("session: private-ref\nreset: 2030-01-01T00:00:00Z\npartial draft")
	status, _, e := s.classifyReviewInterruption(&run, DispatchPlan{Config: a}, text, errors.New("exit75"), nil)
	if e != nil || status != ReviewRunInterrupted || run.Interruption.Cause != "usage-limit" || !run.Interruption.Resumable {
		t.Fatalf("classification: %+v %v", run, e)
	}
	session, e := s.readReviewSession(run.SessionDigest)
	if e != nil || session != "private-ref" {
		t.Fatal(session, e)
	}
	info, e := os.Stat(filepath.Join(s.Root, reviewPrivateDir, strings.TrimPrefix(run.SessionDigest, "sha256:")+".session"))
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, e)
	}
	now := time.Date(2029, 12, 31, 23, 59, 59, 0, time.UTC)
	if d, e := ReviewResumeDelay(run, now, 2*time.Second, time.Second/2); e != nil || d != 1500*time.Millisecond {
		t.Fatal(d, e)
	}
	if _, e := ReviewResumeDelay(run, now, time.Second, time.Second/2); e == nil {
		t.Fatal("reset over bound accepted")
	}
	if _, e := ReviewResumeDelay(ReviewRun{}, now, time.Hour, 0); e == nil {
		t.Fatal("unknown reset accepted")
	}
	for _, ctx := range []error{context.Canceled, context.DeadlineExceeded} {
		r := ReviewRun{ExitCode: -1}
		status, _, e := s.classifyReviewInterruption(&r, DispatchPlan{Config: ReviewerAdapter{}}, nil, ctx, ctx)
		if e != nil || status != ReviewRunInterrupted || r.Interruption.Cause != "unknown" || r.Interruption.Resumable {
			t.Fatal(r, e)
		}
	}
	r := ReviewRun{ExitCode: 75}
	_, _, e = s.classifyReviewInterruption(&r, DispatchPlan{Config: ReviewerAdapter{}}, []byte("usage limit"), errors.New("exit"), nil)
	if e != nil || r.Interruption != nil {
		t.Fatal("undeclared vendor text became limit")
	}
	a.Limit.OutputPatterns = []string{"LIMIT"}
	r = ReviewRun{ExitCode: 0}
	status, _, e = s.classifyReviewInterruption(&r, DispatchPlan{Config: a}, []byte("LIMIT"), nil, nil)
	if e != nil || status != ReviewRunInterrupted {
		t.Fatal("declared zero-exit output limit ignored", e)
	}
}
func TestRunContinuityPolicyAndPartialIntegrity(t *testing.T) {
	a := continuityAdapter()
	a.Session.Pattern = "bad["
	if validateContinuityAdapter(a) == nil {
		t.Fatal("bad regexp accepted")
	}
	a = continuityAdapter()
	a.Session.Pattern = "(a)(b)"
	if validateContinuityAdapter(a) == nil {
		t.Fatal("ambiguous capture accepted")
	}
	a = continuityAdapter()
	a.Resume.Command = []string{"sh", "-c", "{session}"}
	if validateContinuityAdapter(a) == nil {
		t.Fatal("session executable code accepted")
	}
	a = continuityAdapter()
	a.Resume.MaxResumes = 101
	if validateContinuityAdapter(a) == nil {
		t.Fatal("unbounded resumes accepted")
	}
	r := ReviewRun{Status: ReviewRunInterrupted, Interruption: &ReviewInterruption{Cause: "unknown"}, Output: "decision: rejected"}
	r.PartialDigest = digestBytes([]byte(r.Output))
	if e := validateContinuityRun(r); e != nil {
		t.Fatal(e)
	}
	r.Output = "changed"
	if validateContinuityRun(r) == nil {
		t.Fatal("partial rewrite accepted")
	}
	if _, e := captureReviewValue(`session: (\S+)`, "session: a\nsession: b"); e == nil {
		t.Fatal("ambiguous sessions accepted")
	}
}
func TestRunContinuityExclusiveClaimAndRewrittenPredecessor(t *testing.T) {
	s := Store{Root: t.TempDir()}
	prior := ReviewRun{RunID: "rrn-predecessor", AttemptID: "rat-attempt"}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if s.claimReviewContinuation(prior, time.Unix(0, 0)) == nil {
				winners.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatal("claim winners", winners.Load())
	}
	if _, e := s.existingReviewContinuation(prior, nil); e == nil || !strings.Contains(e.Error(), "pending") {
		t.Fatal("pending claim not visible", e)
	}
	prior.Output = "rewritten"
	if _, e := s.existingReviewContinuation(prior, nil); e == nil || !strings.Contains(e.Error(), "changed") {
		t.Fatal("rewritten predecessor accepted", e)
	}
	raw, _ := json.Marshal(prior)
	if strings.Contains(string(raw), "private-ref") {
		t.Fatal("private material in public run")
	}
}
