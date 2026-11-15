package pose

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec pose-federated-proof-reads-each-input-once, R8: what a bundle's bytes
// say about their scope is parsed once per content; each file keeps its own
// artifact path, and a changed content is parsed again.
func TestABundleHeaderIsParsedOncePerContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	good := `{"payload":{"scope":{"ref":"spec:alpha"}}}`
	write("rvb-1.json", good)
	write("rvb-2.json", good)
	files := NativeFiles()
	first := readReviewBundleHeader(files, dir, "rvb-1.json")
	second := readReviewBundleHeader(files, dir, "rvb-2.json")
	if first.owner != "spec:alpha" || second.owner != "spec:alpha" || first.loss != nil || second.loss != nil {
		t.Fatalf("owners %q %q losses %v %v", first.owner, second.owner, first.loss, second.loss)
	}
	write("rvb-1.json", `{"payload":{"scope":{"ref":"spec:beta"}}}`)
	if changed := readReviewBundleHeader(files, dir, "rvb-1.json"); changed.owner != "spec:beta" {
		t.Fatalf("a changed content kept the old scope: %q", changed.owner)
	}
	bad := `{"payload":{"scope":{"ref":"spec:alpha"}},"payload":{"scope":{"ref":"spec:beta"}}}`
	write("rvb-3.json", bad)
	write("rvb-4.json", bad)
	a := readReviewBundleHeader(files, dir, "rvb-3.json")
	b := readReviewBundleHeader(files, dir, "rvb-4.json")
	if a.loss == nil || b.loss == nil || a.loss.Kind != b.loss.Kind || a.loss.Detail != b.loss.Detail {
		t.Fatalf("the loss was not kept whole: %+v %+v", a.loss, b.loss)
	}
	if a.loss.Artifact != ".pose/review-bundles/rvb-3.json" || b.loss.Artifact != ".pose/review-bundles/rvb-4.json" {
		t.Fatalf("a kept loss named another file: %q %q", a.loss.Artifact, b.loss.Artifact)
	}
}
