package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

// A diagnostic run inside a sealed review checkout leaves the delivery index as
// sealed; outside one it publishes as before (spec
// pose-validation-evidence-material-binding R7).
func TestRefreshDeliveryIndexLeavesASealedCheckoutAlone(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	target := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	t.Setenv(posepkg.SealedCheckoutEnv, root)
	var stderr bytes.Buffer
	refreshDeliveryIndex(root, []byte("{}\n"), &stderr)
	if _, err := os.Stat(target); err == nil {
		t.Fatal("a sealed checkout was written")
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "indexes")); err == nil {
		t.Fatal("a sealed checkout gained a directory")
	}
	// Another project the same process works in is not the sealed checkout.
	refreshDeliveryIndex(other, []byte("{}\n"), &stderr)
	if _, err := os.Stat(filepath.Join(other, ".pose", "indexes", "delivery-integrity.json")); err != nil {
		t.Fatalf("a fixture was mistaken for the sealed checkout: %v", err)
	}
	t.Setenv(posepkg.SealedCheckoutEnv, "")
	refreshDeliveryIndex(root, []byte("{}\n"), &stderr)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("outside a sealed checkout the index is published as before: %v", err)
	}
}
