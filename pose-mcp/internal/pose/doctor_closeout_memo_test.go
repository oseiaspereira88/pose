package pose

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Real sealed member review and published receipts; only the topology is
// synthetic. Reusing one approved member in the same roadmap isolates the
// predecessor cost from member-review cost; it is not a production doctor
// benchmark. These existing fixtures have a legacy index: their outer audit
// may refuse currentness. Equivalence below concerns the diagnostic evaluation
// inside one capture, never an assertion that a legacy audit was certified.
func doctorMemoFixture(t *testing.T, n int) (Store, []string) {
	t.Helper()
	root, s := closeoutReviewedFixture(t, "never", "never")
	resolver := *s.FederatedResolver
	resolver.Authorize = nil // exercise the trusted CLI boundary
	s.FederatedResolver = &resolver
	text := "---\nslug: program\nstatus: active\n---\n"
	refs := make([]string, n)
	for i := range refs {
		name := "core"
		if i > 0 {
			name = fmt.Sprintf("step%d", i)
		}
		refs[i] = "milestone:program/" + name
		text += "\n## Milestone: " + name + "\n"
		if i > 0 {
			text += "- after: " + strings.TrimPrefix(refs[i-1], "milestone:program/") + "\n"
		}
		text += "- specs: backend\n"
	}
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", text)
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "local predecessor fixture")
	approveMember(t, s, reviewFixtureNow.Add(10*time.Minute))
	s = s.atReviewTime(reviewFixtureNow.Add(24 * time.Hour))
	for _, ref := range refs {
		state, err := s.GetCloseoutState(ref)
		if err != nil || state.Review.CloseoutReview == nil {
			t.Fatalf("receipt fixture %s: %+v %v", ref, state, err)
		}
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.PublishCloseoutReceipt(receipt); err != nil {
			t.Fatal(err)
		}
	}
	return s, refs
}

func doctorMemoOperation(t *testing.T, s Store, run func(Store)) {
	t.Helper()
	err := s.WithReviewOperationView(func(shared Store) { shared.withDoctorCloseoutMemo(run) })
	if err != nil && !errors.Is(err, ErrGovernanceReadViewUnknown) {
		t.Fatal(err)
	}
}

func TestDoctorCloseoutMemoLinearEquivalenceAndCount(t *testing.T) {
	s, refs := doctorMemoFixture(t, 6)
	doctorMemoOperation(t, s, func(query Store) {
		if query.doctorCloseoutMemo == nil {
			t.Fatal("memo not active")
		}
		computes := 0
		query.doctorCloseoutMemo.onCompute = func(string) { computes++ }
		// Ask in reverse order so the first evaluation must traverse the chain.
		for i := len(refs) - 1; i >= 0; i-- {
			ref := refs[i]
			plain := query
			plain.doctorCloseoutMemo = nil
			want, wantErr := plain.closedByReceipt(ref)
			got, err := query.closedByReceipt(ref)
			if wantErr != nil || err != nil || !want || got != want {
				state, _ := plain.getCloseoutState(ref, true)
				t.Fatalf("%s: memo=%v/%v plain=%v/%v blockers=%v", ref, got, err, want, wantErr, state.Blockers)
			}
		}
		if computes != len(refs) {
			t.Fatalf("real verifier calls=%d, want %d", computes, len(refs))
		}
		plain := query
		plain.doctorCloseoutMemo = nil
		want, wantErr := plain.closeoutReviewsOwed()
		got, err := query.closeoutReviewsOwed()
		if !reflect.DeepEqual(got, want) || fmt.Sprint(err) != fmt.Sprint(wantErr) {
			t.Fatalf("owed differs: memo=%+v/%v plain=%+v/%v", got, err, want, wantErr)
		}
	})
}

func TestDoctorCloseoutMemoAncestorIntersection(t *testing.T) {
	s, refs := doctorMemoFixture(t, 3)
	doctorMemoOperation(t, s, func(query Store) {
		if closed, err := query.closedByReceipt(refs[2]); err != nil || !closed {
			t.Fatalf("uncut X must close: %v %v", closed, err)
		}
		computes := 0
		query.doctorCloseoutMemo.onCompute = func(string) { computes++ }
		query.closeoutChain = []string{refs[0]}
		plain := query
		plain.doctorCloseoutMemo = nil
		want, wantErr := plain.closedByReceipt(refs[2])
		got, err := query.closedByReceipt(refs[2])
		if want || got != want || err != nil || wantErr != nil || computes == 0 {
			t.Fatalf("ancestor through cached child: memo=%v/%v plain=%v/%v computes=%d", got, err, want, wantErr, computes)
		}
		// A cut evaluation must not replace the independent, uncut approval.
		query.closeoutChain = nil
		if closed, err := query.closedByReceipt(refs[2]); err != nil || !closed {
			t.Fatalf("cut poisoned uncut result: %v %v", closed, err)
		}
	})
}

func TestDoctorCloseoutMemoLocalCycle(t *testing.T) {
	s, refs := doctorMemoFixture(t, 2)
	editFile(t, filepath.Join(s.Root, ".pose/roadmaps/program.md"), "## Milestone: core\n", "## Milestone: core\n- after: step1\n")
	doctorMemoOperation(t, s, func(query Store) {
		for _, ref := range refs {
			plain := query
			plain.doctorCloseoutMemo = nil
			want, wantErr := plain.closedByReceipt(ref)
			got, err := query.closedByReceipt(ref)
			if want || got != want || fmt.Sprint(err) != fmt.Sprint(wantErr) {
				t.Fatalf("cycle %s: memo=%v/%v plain=%v/%v", ref, got, err, want, wantErr)
			}
		}
		if len(query.doctorCloseoutMemo.entries) != 0 {
			t.Fatal("cycle retained an approval")
		}
	})
}

func TestDoctorCloseoutMemoLifetimeClockAndBinding(t *testing.T) {
	s, refs := doctorMemoFixture(t, 1)
	var escaped Store
	doctorMemoOperation(t, s, func(query Store) {
		escaped = query
		if closed, err := query.closedByReceipt(refs[0]); err != nil || !closed {
			t.Fatal(closed, err)
		}
		computes := 0
		query.doctorCloseoutMemo.onCompute = func(string) { computes++ }
		other := query.atReviewTime(query.reviewClock().Add(time.Second))
		if closed, err := other.closedByReceipt(refs[0]); err != nil || !closed {
			t.Fatal(closed, err)
		}
		if computes != 0 {
			t.Fatal("different clock entered memo")
		}
		if fresh := query.withoutGovernanceReadView(); fresh.doctorCloseoutMemo != nil || fresh.closeoutTrace != nil {
			t.Fatal("new view retained memo")
		}
		_, view := query.beginGovernanceReadView()
		if fresh := query.withGovernanceReadView(view, "test"); fresh.doctorCloseoutMemo != nil || fresh.closeoutTrace != nil {
			t.Fatal("rebound view retained memo")
		}

	})
	if !escaped.doctorCloseoutMemo.closed || len(escaped.doctorCloseoutMemo.entries) != 0 {
		t.Fatal("query retained memo after callback")
	}
	doctorMemoOperation(t, s, func(query Store) {
		if query.doctorCloseoutMemo == escaped.doctorCloseoutMemo || len(query.doctorCloseoutMemo.entries) != 0 {
			t.Fatal("new query borrowed memo")
		}
	})
}

func TestDoctorCloseoutMemoBoundsAndDynamicAuthorization(t *testing.T) {
	s, refs := doctorMemoFixture(t, 1)
	doctorMemoOperation(t, s, func(query Store) {
		m := query.doctorCloseoutMemo
		m.visited = doctorMemoMaxTotalVisited
		if closed, err := query.closedByReceipt(refs[0]); err != nil || !closed {
			t.Fatal(closed, err)
		}
		if len(m.entries) != 0 {
			t.Fatal("visited budget exceeded")
		}
		m.visited = 0
		for i := 0; i < doctorMemoMaxEntries; i++ {
			m.entries[fmt.Sprint(i)] = doctorCloseoutEntry{}
		}
		if closed, err := query.closedByReceipt(refs[0]); err != nil || !closed {
			t.Fatal(closed, err)
		}
		if len(m.entries) != doctorMemoMaxEntries {
			t.Fatal("entry budget exceeded")
		}
		trace := &doctorCloseoutTrace{visited: map[string]bool{}}
		for i := 0; i <= doctorMemoMaxVisited; i++ {
			trace.visit(fmt.Sprint(i), false)
		}
		visited, cut := trace.snapshot()
		if !cut || len(visited) != doctorMemoMaxVisited {
			t.Fatal("incomplete trace reusable")
		}
	})
	resolver := *s.FederatedResolver
	resolver.Authorize = func(string) bool { return true }
	s.FederatedResolver = &resolver
	doctorMemoOperation(t, s, func(query Store) {
		if query.doctorCloseoutMemo != nil {
			t.Fatal("dynamic authorization cached")
		}
	})
}

func TestDoctorCloseoutMemoErrorsAndRemoteBoundary(t *testing.T) {
	s, _ := doctorMemoFixture(t, 1)
	doctorMemoOperation(t, s, func(query Store) {
		computes := 0
		query.doctorCloseoutMemo.onCompute = func(string) { computes++ }
		for i := 0; i < 2; i++ {
			if closed, err := query.closedByReceipt("not-a-scope"); closed || err == nil {
				t.Fatalf("invalid scope: %v %v", closed, err)
			}
			if closed, err := query.closedByReceipt("milestone:program/missing"); closed || err != nil {
				t.Fatalf("missing receipt: %v %v", closed, err)
			}
		}
		if computes != 4 || len(query.doctorCloseoutMemo.entries) != 0 {
			t.Fatal("errors or negative results were cached", computes)
		}
		trace := &doctorCloseoutTrace{visited: map[string]bool{}}
		query.closeoutTrace = trace
		resolved := query.FederatedResolver.Resolve("coordinator", "xref:source/roadmap:component")
		if !resolved.Resolved {
			t.Fatal("remote fixture did not resolve", resolved)
		}
		plain := query
		plain.doctorCloseoutMemo, plain.closeoutTrace = nil, nil
		want, wantReason := plain.AggregateDependencyClosed(*plain.FederatedResolver, "coordinator", resolved)
		got, reason := query.AggregateDependencyClosed(*query.FederatedResolver, "coordinator", resolved)
		if got != want || reason != wantReason {
			t.Fatal("remote semantics changed", got, reason, want, wantReason)
		}
		if _, cut := trace.snapshot(); !cut {
			t.Fatal("remote boundary left reusable parent trace")
		}
	})
}

func TestDoctorCloseoutMemoBindingRefused(t *testing.T) {
	s, refs := doctorMemoFixture(t, 1)
	err := s.WithReviewOperationView(func(shared Store) {
		shared.withDoctorCloseoutMemo(func(query Store) {
			if closed, err := query.closedByReceipt(refs[0]); err != nil || !closed {
				t.Fatal(closed, err)
			}
			computes := 0
			query.doctorCloseoutMemo.onCompute = func(string) { computes++ }
			resolver := *query.FederatedResolver
			query.FederatedResolver = &resolver
			if closed, _ := query.closedByReceipt(refs[0]); closed {
				t.Fatal("approval borrowed another resolver binding")
			}
			if computes != 0 {
				t.Fatal("incompatible binding entered memo")
			}
		})
	})
	if err == nil || !strings.Contains(err.Error(), "Store context") {
		t.Fatal("binding conflict escaped final audit", err)
	}
}

func TestDoctorCloseoutMemoFallbackKeepsCallerClock(t *testing.T) {
	s, _ := doctorMemoFixture(t, 1)
	resolver := *s.FederatedResolver
	resolver.Authorize = func(string) bool { return true }
	s.FederatedResolver = &resolver
	for _, explicit := range []bool{false, true} {
		input := s
		if !explicit {
			input.reviewTime = nil
		}
		err := input.WithReviewOperationView(func(shared Store) {
			before := shared.reviewTime
			shared.withDoctorCloseoutMemo(func(query Store) {
				if query.doctorCloseoutMemo != nil || query.reviewTime != before {
					t.Fatal("fallback changed caller clock or enabled memo")
				}
			})
		})
		if err != nil && !errors.Is(err, ErrGovernanceReadViewUnknown) {
			t.Fatal(err)
		}
	}
}

// This is a real terminal scope with a foreign spec proof, not just a
// fabricated manifest. Removing traceDoctorCloseoutSources must retain the
// approval and fail this test; the conservative cut is intentional.
func TestDoctorCloseoutMemoForeignSpecProofIsNotRetained(t *testing.T) {
	_, s, _, _ := federatedTestMilestoneConsumer(t)
	setCloseoutPolicy(t, s.Root, "never", "never")
	approveMember(t, s, reviewFixtureNow.Add(10*time.Minute))
	resolver := *s.FederatedResolver
	resolver.Authorize = nil
	s.FederatedResolver = &resolver
	s = s.atReviewTime(reviewFixtureNow.Add(24 * time.Hour))
	ref := "milestone:program/core"
	state, err := s.GetCloseoutState(ref)
	if err != nil || state.Review.CloseoutReview == nil {
		t.Fatal(state, err)
	}
	receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.PublishCloseoutReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	doctorMemoOperation(t, s, func(query Store) {
		calls := 0
		query.doctorCloseoutMemo.onCompute = func(string) { calls++ }
		for i := 0; i < 2; i++ {
			closed, err := query.closedByReceipt(ref)
			if err != nil || !closed {
				state, _ := query.getCloseoutState(ref, true)
				t.Fatalf("foreign fixture not terminal: %v %v %v", closed, err, state.Blockers)
			}
		}
		if calls != 2 || len(query.doctorCloseoutMemo.entries) != 0 {
			t.Fatal("foreign proof retained", calls)
		}
	})
}
