package pose

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGovernanceReadViewRoadmapChildrenKeepAuthorizedBinding(t *testing.T) {
	store, _, _ := retentionFixture(t)
	writeReviewFixture(t, store.Root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n# Program\n\n## Milestone: core\n- specs: backend\n")
	bindCurrentReviewFixture(t, store)
	err := store.WithReviewCoordinator(func(shared Store) error {
		return shared.WithReviewScopeOperationView(func(scope Store) error {
			state, err := scope.GetCloseoutState("roadmap:program")
			if err == nil && (len(state.Children) != 1 || len(state.Children[0].Children) != 1) {
				t.Fatalf("children were not evaluated: %+v", state)
			}
			return err
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceReadViewScopeCoordinatorRetainsCallbackAndAuditFailure(t *testing.T) {
	root, _ := d7ProbeCoordinatorRoot(t)
	path := filepath.Join(root, "policy.json")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	callbackFailure := errors.New("domain evaluation failed")
	err := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
		if _, err := shared.fsys().ReadFile(path); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte("after!"), 0600); err != nil {
			return err
		}
		return callbackFailure
	})
	if !errors.Is(err, callbackFailure) || !errors.Is(err, ErrGovernanceReadViewChanged) {
		t.Fatalf("callback or audit cause was lost: %v", err)
	}
}

func TestGovernanceReadViewScopeCoordinatorRefusesRetiredOrChangedBinding(t *testing.T) {
	for _, kind := range []string{"root", "project", "resolver", "files", "retired"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := d7ProbeCoordinatorRoot(t)
			var retained Store
			var scopeErr error
			called := false
			outerErr := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
				retained = shared
				if kind == "retired" {
					return nil
				}
				bad := shared
				switch kind {
				case "root":
					bad.Root = t.TempDir()
				case "project":
					bad.FederatedProjectID = "proj.foreign"
				case "resolver":
					bad.FederatedResolver = &ArtifactResolver{}
				case "files":
					bad.FS = NativeFiles()
				}
				scopeErr = bad.WithReviewScopeOperationView(func(Store) error { called = true; return nil })
				return scopeErr
			})
			if kind == "retired" {
				if outerErr != nil {
					t.Fatal(outerErr)
				}
				scopeErr = retained.WithReviewScopeOperationView(func(Store) error { called = true; return nil })
			}
			if scopeErr == nil || called {
				t.Fatalf("scope borrowed an unauthorized binding: callback=%t err=%v", called, scopeErr)
			}
		})
	}
}

func TestGovernanceReadViewScopeCoordinatorParallelClosuresStayIndependent(t *testing.T) {
	root, path := d7ProbeCoordinatorRoot(t)
	err := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
		const count = 4
		errs := make(chan error, count)
		var wg sync.WaitGroup
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- shared.WithReviewScopeOperationView(func(scope Store) error {
					rv, _ := governanceReadViewFromFiles(scope.fsys())
					rv.operation.budgetMu.Lock()
					rv.operation.inputBytes = governanceReadViewMaxInputBytes - 40
					rv.operation.budgetMu.Unlock()
					_, err := scope.fsys().ReadFile(path)
					return err
				})
			}()
		}
		wg.Wait()
		close(errs)
		var combined error
		for err := range errs {
			combined = errors.Join(combined, err)
		}
		return combined
	})
	if err != nil {
		t.Fatalf("independently fitting parallel scopes shared consumption: %v", err)
	}
}

func TestGovernanceReadViewScopeCoordinatorHistoricalApprovalAndLaterRejection(t *testing.T) {
	for _, reject := range []bool{false, true} {
		label := "retained"
		if reject {
			label = "later-rejection"
		}
		t.Run(label, func(t *testing.T) {
			store, _, now := retentionFixture(t)
			if reject {
				addMatrixCheck(t, store)
				next, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
				att := approvedBundleAttestation(next, "agent:later")
				att.Decision = "rejected"
				if _, err := store.RecordReviewAttestation(att, now.Add(3*time.Minute)); err != nil {
					t.Fatal(err)
				}
			}
			closeBackendSpec(t, store)
			addMatrixCheck(t, store)
			bindCurrentReviewFixture(t, store)
			var state CloseoutState
			err := store.WithReviewCoordinator(func(shared Store) error {
				return shared.WithReviewScopeOperationView(func(scope Store) error {
					var err error
					state, err = scope.GetCloseoutState("spec:backend")
					return err
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			if state.Terminal == reject || state.Review.Approved == reject {
				t.Fatalf("historical decision lost under scoped coordinator: reject=%t state=%+v", reject, state)
			}
		})
	}
}

func TestGovernanceReadViewScopeCoordinatorAuditsCatalogueAfterScopes(t *testing.T) {
	root, path := d7ProbeCoordinatorRoot(t)
	sp := filepath.Join(root, ".pose/specs/alpha.md")
	original := "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-13\n---\nbody\n"
	if err := os.WriteFile(sp, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	err := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
		if _, err := shared.ListSpecs("", ""); err != nil {
			return err
		}
		if err := shared.WithReviewScopeOperationView(func(scope Store) error { _, err := scope.fsys().ReadFile(path); return err }); err != nil {
			return err
		}
		return os.WriteFile(sp, []byte(strings.Replace(original, "status: done", "status: draft", 1)), 0600)
	})
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("selection changed after successful scope and escaped coordinator audit: %v", err)
	}
}

func TestGovernanceReadViewScopeAuditPreservesChangeAndIndependentUnknown(t *testing.T) {
	audit := governanceReadViewAudit{ChangedPaths: []string{"changed.json"}, ChangedProducers: map[string]bool{"review-operation": true}, UnknownProducers: map[string][]string{"review-operation": {"a different input exceeded its final-audit budget"}}}
	err := reviewOperationGovernanceAuditError(audit, nil, nil)
	if !errors.Is(err, ErrGovernanceReadViewChanged) || !errors.Is(err, ErrGovernanceReadViewUnknown) {
		t.Fatalf("independent changed/unknown causes were collapsed: %v", err)
	}
}
