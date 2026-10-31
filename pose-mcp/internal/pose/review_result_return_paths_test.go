package pose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// R1–R2 on every return path of a dispatch (adopted from the independent
// review's probe, round 1 F02): the result is published at most once, with
// the marker present unless marking itself failed, and the marker is kept
// exactly when the record or the result is missing.
func TestARunsReturnPathsPublishOnceAndKeepTheirMarker(t *testing.T) {
	for _, mode := range []string{"onstart", "panic", "record-fails", "receipt-fails", "mark-fails", "cancelled-before-reserve", "cancel-after-mark"} {
		t.Run(mode, func(t *testing.T) {
			f := provenanceRunFixture(t, "openai", "probe-model")
			plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "probe")
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			saved := beforeRunResultPublished
			defer func() { beforeRunResultPublished = saved }()
			beforeRunResultPublished = func(r ReviewRun) {
				count++
				if mode != "mark-fails" && !f.store.DelegationActiveMarkerExists(r.AttemptID) {
					t.Error("marker absent at publication")
				}
				if mode == "receipt-fails" {
					attempts, e := f.store.readDelegatedReviewAttempts(f.bundle.BundleID)
					if e != nil {
						t.Fatal(e)
					}
					for _, a := range attempts {
						if a.AttemptID == r.AttemptID {
							if e := os.MkdirAll(attemptReceiptPath(f.store.Root, a), 0700); e != nil {
								t.Fatal(e)
							}
						}
					}
				}
			}
			attempt := ""
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			plan.OnStart = func(a string) error {
				attempt = a
				switch mode {
				case "panic":
					panic("probe")
				case "record-fails":
					p := filepath.Join(f.store.Root, reviewRunsDir)
					if e := os.Rename(p, p+".saved"); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(p, []byte("blocked"), 0600); e != nil {
						t.Fatal(e)
					}
				case "cancel-after-mark":
					cancel()
				}
				return errors.New("probe onstart")
			}
			if mode == "mark-fails" {
				p, e := f.store.delegationActiveDir(true)
				if e != nil {
					t.Fatal(e)
				}
				if e = os.RemoveAll(p); e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(p, []byte("blocked"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "cancelled-before-reserve" {
				cancel()
			}
			panicked := false
			var r ReviewRun
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				r, err = f.store.RunReviewDispatchContext(ctx, plan, time.Now)
			}()
			want := 1
			if mode == "panic" || mode == "record-fails" || mode == "cancelled-before-reserve" {
				want = 0
			}
			if count != want {
				t.Errorf("publication calls %d want %d", count, want)
			}
			if panicked != (mode == "panic") {
				t.Errorf("panic=%v", panicked)
			}
			if mode == "receipt-fails" || mode == "record-fails" {
				if err == nil {
					t.Error("missing operational error")
				}
			}
			if attempt != "" {
				kept := f.store.DelegationActiveMarkerExists(attempt)
				wantKeep := mode == "panic" || mode == "record-fails" || mode == "receipt-fails"
				if kept != wantKeep {
					t.Errorf("marker kept=%v want=%v", kept, wantKeep)
				}
			}
			if want == 1 && r.RunID == "" {
				t.Error("expected recorded result")
			}
			t.Logf("mode=%s calls=%d run=%s error=%v", mode, count, r.RunID, err)
		})
	}
}

// The same for a continuation (a resume).
func TestAContinuationsReturnPathsPublishOnceAndKeepTheirMarker(t *testing.T) {
	for _, mode := range []string{"onstart", "receipt-fails", "record-fails", "panic"} {
		t.Run(mode, func(t *testing.T) {
			f := provenanceRunFixture(t, "openai", "probe-model", func(a *ReviewerAdapter) {
				a.Command = []string{"sh", "-c", "cat >/dev/null; echo session: probe-session; exit 75"}
				a.Limit = &ReviewerLimit{ExitCodes: []int{75}}
				a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
				a.Resume = &ReviewerResume{Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved", "sh", "{session}"}}
			})
			plan, err := f.store.PlanReviewResume(f.run.RunID)
			if err != nil {
				t.Fatal(err)
			}
			saved := beforeRunResultPublished
			defer func() { beforeRunResultPublished = saved }()
			count := 0
			beforeRunResultPublished = func(r ReviewRun) {
				count++
				if !f.store.DelegationActiveMarkerExists(r.AttemptID) {
					t.Error("marker missing before continuation receipt")
				}
				if mode == "receipt-fails" {
					if e := os.MkdirAll(filepath.Join(f.store.Root, continuationPath(f.run.RunID, true)), 0700); e != nil {
						t.Fatal(e)
					}
				}
			}
			plan.OnStart = func(a string) error {
				if mode == "panic" {
					panic("probe")
				}
				if mode == "record-fails" {
					p := filepath.Join(f.store.Root, reviewRunsDir)
					if e := os.Rename(p, p+".saved"); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(p, []byte("blocked"), 0600); e != nil {
						t.Fatal(e)
					}
				}
				return errors.New("onstart probe")
			}
			var r ReviewRun
			func() { defer func() { recover() }(); r, err = f.store.RunReviewDispatch(plan, time.Now) }()
			want := 1
			if mode == "record-fails" || mode == "panic" {
				want = 0
			}
			if count != want {
				t.Errorf("calls=%d want=%d", count, want)
			}
			kept := f.store.DelegationActiveMarkerExists(f.run.AttemptID)
			if kept != (mode != "onstart") {
				t.Errorf("marker kept=%v", kept)
			}
			if mode == "onstart" {
				next, e := f.store.PlanReviewResume(f.run.RunID)
				if e != nil || next.Existing == nil || next.Existing.RunID != r.RunID {
					t.Fatalf("replay lost: %+v %v", next, e)
				}
			}
			t.Logf("mode=%s calls=%d kept=%v error=%v", mode, count, kept, err)
		})
	}
}
