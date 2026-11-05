package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeIndexInput(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestModuleDiscoveryBindingTracksNestedManifestNamesOnly(t *testing.T) {
	root := t.TempDir()
	writeIndexInput(t, root, "services/api/go.mod", "module example.test/api\n")
	writeIndexInput(t, root, ".qwen/worktrees/check/services/ignored/go.mod", "module ignored\n")
	writeIndexInput(t, root, "testdata/fixtures/sample/go.mod", "module fixture\n")

	before, known := IndexInputBindingIn(NativeFiles(), root, "module-discovery")
	if !known || before == "" {
		t.Fatal("module discovery binding was unknown")
	}
	inventory, err := ScanModuleDiscoveryIn(NativeFiles(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Manifests) != 1 || inventory.Manifests[0].Path != "services/api/go.mod" {
		t.Fatalf("inventory included filtered paths or missed the real module: %+v", inventory.Manifests)
	}

	// The scanner classifies by names and existence. Manifest content is not a
	// repo-map input, while a nested recognized manifest changes the inventory.
	writeIndexInput(t, root, "services/api/go.mod", "module example.test/api/v2\n")
	contentOnly, known := IndexInputBindingIn(NativeFiles(), root, "module-discovery")
	if !known || contentOnly != before {
		t.Fatalf("manifest contents changed the names-only scanner binding: %q -> %q", before, contentOnly)
	}
	writeIndexInput(t, root, "services/new-api/go.mod", "module example.test/new-api\n")
	after, known := IndexInputBindingIn(NativeFiles(), root, "module-discovery")
	if !known || after == before {
		t.Fatalf("new nested module did not change the discovery binding: before=%q after=%q known=%v", before, after, known)
	}
}

func TestValidationResultsBindingTracksOutcome(t *testing.T) {
	root := t.TempDir()
	writeIndexInput(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"results_path":".pose/results/delivery.json","roots":[]}`)
	result := `{"schema_version":1,"generated_at":"2026-11-04T00:00:00Z","checks":[{"id":"api-test","module":"services/api","name":"go-test","severity":"required","evidence_class":"unit","outcome":"pass"}]}`
	writeIndexInput(t, root, ".pose/results/delivery.json", result)
	before, known := IndexInputBindingIn(NativeFiles(), root, "validation-results")
	if !known || before == "" {
		t.Fatal("valid delivery results did not produce a known binding")
	}
	changed := strings.Replace(result, `"pass"`, `"fail"`, 1)
	writeIndexInput(t, root, ".pose/results/delivery.json", changed)
	after, known := IndexInputBindingIn(NativeFiles(), root, "validation-results")
	if !known || after == before {
		t.Fatalf("validation outcome did not change its binding: before=%q after=%q known=%v", before, after, known)
	}
}

func TestSmokeResultsBindingTracksSourceInventoryWithoutIndexRead(t *testing.T) {
	root := t.TempDir()
	before, known := IndexInputBindingIn(NativeFiles(), root, "smoke-results")
	if !known || before == "" {
		t.Fatal("absent smoke results were not represented as a known empty source projection")
	}
	writeIndexInput(t, root, SmokeResultsDir+"/run.json", `{}`)
	after, known := IndexInputBindingIn(NativeFiles(), root, "smoke-results")
	if !known || after == before {
		t.Fatalf("new smoke source document did not change the binding: before=%q after=%q known=%v", before, after, known)
	}
}

func TestReleaseStatusBindingTracksTagMovement(t *testing.T) {
	repo := newReleaseRepo(t, "v1.0.0")
	beforeTag, known := IndexInputBindingIn(NativeFiles(), repo.root, "release-status")
	if !known || beforeTag == "" {
		t.Fatal("release-status binding was unknown before the tag")
	}
	oldCommit := relGit(t, repo.root, "rev-parse", "HEAD")
	relGit(t, repo.root, "tag", "v1.0.0", oldCommit)
	afterTag, known := IndexInputBindingIn(NativeFiles(), repo.root, "release-status")
	if !known || afterTag == beforeTag {
		t.Fatalf("tag appearance did not change release-status binding: before=%q after=%q known=%v", beforeTag, afterTag, known)
	}
	repo.addVersion(t, "v1.1.0")
	newCommit := relGit(t, repo.root, "rev-parse", "HEAD")
	relGit(t, repo.root, "tag", "-f", "v1.0.0", newCommit)
	afterMove, known := IndexInputBindingIn(NativeFiles(), repo.root, "release-status")
	if !known || afterMove == afterTag {
		t.Fatalf("tag movement did not change release-status binding: tagged=%q moved=%q known=%v", afterTag, afterMove, known)
	}
}
