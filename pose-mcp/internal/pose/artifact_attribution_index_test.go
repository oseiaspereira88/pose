package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRetroactiveAttributionIndexBinding(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := NativeFiles()
	changeSets := recordedChangeSetsDigest(files, root)
	binding := func() string {
		v, ok := IndexInputBindingIn(files, root, "policy")
		if !ok {
			t.Fatal("could not bind policy inputs")
		}
		return v
	}
	original := binding()
	manifest := filepath.Join(root, ArtifactAttributionManifest)
	for _, path := range []string{manifest, manifest + ".sig"} {
		previous := binding()
		if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
			t.Fatal(err)
		}
		first := binding()
		if err := os.WriteFile(path, []byte("second"), 0o600); err != nil {
			t.Fatal(err)
		}
		if first == previous || binding() == first {
			t.Fatalf("appearance or modification of %s did not change the binding", filepath.Base(path))
		}
	}
	for _, path := range []string{manifest, manifest + ".sig"} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	if binding() != original || recordedChangeSetsDigest(files, root) != changeSets {
		t.Fatal("removing the manifest and signature failed to restore the bindings")
	}
}

func TestRetroactiveAttributionReviewSubjectClassification(t *testing.T) {
	scope := ScopeRef{Kind: "spec", Slug: "historical-attribution"}
	for _, path := range []string{ArtifactAttributionManifest, ArtifactAttributionManifest + ".sig"} {
		if class, include := reviewBundlePathClass(path, scope, nil); !include || class != "governance" {
			t.Errorf("%s was not retained in the review subject: %q, %v", path, class, include)
		}
	}
}
