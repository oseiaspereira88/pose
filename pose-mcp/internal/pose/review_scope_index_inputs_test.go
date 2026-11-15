package pose

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func currentPinFixture() *IndexGenerationView {
	inputs := map[string]string{}
	for _, name := range IndexInputNames {
		inputs[name] = "sha256:bound"
	}
	return &IndexGenerationView{Status: IndexGenerationCurrent, Descriptor: IndexGenerationDescriptor{GenerationID: "gen-fixture", Inputs: inputs}}
}

// ADR governance-reads-use-bounded-operation-scoped-views D12: under one
// coordinator, the consumed index's inputs are checked once, through the
// coordinator's operation, however many scopes consume the index.
func TestScopeIndexInputsAreCheckedOncePerCoordinator(t *testing.T) {
	root, _ := d7ProbeCoordinatorRoot(t)
	var mu sync.Mutex
	computed := map[string]int{}
	original := governanceIndexInputBindingObserved
	governanceIndexInputBindingObserved = func(_, name string) {
		mu.Lock()
		computed[name]++
		mu.Unlock()
	}
	defer func() { governanceIndexInputBindingObserved = original }()
	pin := currentPinFixture()
	err := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
		coordinatorView, _ := governanceReadViewFromFiles(shared.fsys())
		var wg sync.WaitGroup
		errs := make(chan error, 3)
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- shared.WithReviewScopeOperationView(func(scope Store) error {
					view, _ := governanceReadViewFromFiles(scope.fsys())
					if view.operation.indexInputsFrom != coordinatorView.operation {
						return errors.New("the scope does not check its index inputs through the coordinator")
					}
					view.operation.checkIndexInputs(scope.fsys(), scope.Root, "review-operation", pin)
					return nil
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
		t.Fatal(err)
	}
	for _, name := range IndexInputNames {
		if computed[name] != 1 {
			t.Fatalf("input %s was computed %d times for three scopes of one coordinator", name, computed[name])
		}
	}
}

// D12: the reads of the shared check belong to the coordinator's closure, so
// an input changed while the scopes run fails the coordinator's audit.
func TestAScopeIndexInputChangedDuringTheCoordinatorFailsItsAudit(t *testing.T) {
	root, _ := d7ProbeCoordinatorRoot(t)
	policy := filepath.Join(root, ".pose", "policy", "review.json")
	if err := os.MkdirAll(filepath.Dir(policy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policy, []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pin := currentPinFixture()
	err := (Store{Root: root}).WithReviewCoordinator(func(shared Store) error {
		if err := shared.WithReviewScopeOperationView(func(scope Store) error {
			view, _ := governanceReadViewFromFiles(scope.fsys())
			view.operation.checkIndexInputs(scope.fsys(), scope.Root, "review-operation", pin)
			return nil
		}); err != nil {
			return err
		}
		return os.WriteFile(policy, []byte(`{"schema_version":2}`), 0o600)
	})
	if !errors.Is(err, ErrGovernanceReadViewChanged) {
		t.Fatalf("an index input changed under the coordinator passed its audit: %v", err)
	}
}
