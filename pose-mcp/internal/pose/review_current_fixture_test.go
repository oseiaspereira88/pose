package pose

import (
	"os"
	"path/filepath"
	"testing"
)

// bindCurrentReviewFixture upgrades synthetic legacy indexes without rebuilding
// their deliberately chosen evidence. Call after editing indexed inputs and
// before evaluating a gate that asserts current approval rather than history.
func bindCurrentReviewFixture(t *testing.T, store Store) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(store.Root, ".git")); os.IsNotExist(err) {
		federatedTestGit(t, store.Root, "init", "-q")
		federatedTestGit(t, store.Root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "review fixture source")
	}
	outputs := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		raw, err := os.ReadFile(filepath.Join(indexDir(store.Root), name))
		if os.IsNotExist(err) {
			raw = []byte("{}\n")
		} else if err != nil {
			t.Fatal(err)
		}
		outputs[name] = raw
	}
	bindingStore := store
	if bindingStore.FederatedResolver == nil {
		resolver, project, err := EnvironmentArtifactResolverIn(store.fsys(), store.Root, "")
		if err != nil {
			t.Fatal(err)
		}
		bindingStore.FederatedResolver, bindingStore.FederatedProjectID = &resolver, project
	}
	inputs, unknown, _ := IndexInputBindingsForStoreIn(bindingStore)
	if len(unknown) != 0 {
		t.Fatalf("current fixture has unknown index inputs: %v", unknown)
	}
	if _, err := store.PublishIndexGeneration(outputs, inputs); err != nil {
		t.Fatal(err)
	}
}
