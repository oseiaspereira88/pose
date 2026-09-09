// The shipped stacks catalog and this repository's own must agree
// (spec pose-emittable-analysis-evidence-classes).
//
// `.pose/indexes/validation-matrix.json` is excluded from the byte-for-byte
// scaffold sync because its `moduleOverrides` describe pose-mcp's own module
// graph (issue #22). What ships instead is the literal in this file, whose
// `stacks` and `deliveryProfiles` are meant to be the same generic catalog.
//
// Meant to be, and kept so by hand. Nothing compared them, so declaring an
// evidence class on a stack check in one place and not the other would ship a
// default that disagrees with the catalog this repository validates itself
// against — and it would ship silently, which is how the shellcheck list, the
// docs-parity list and the workflow checkout depths all went wrong.

package distpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestShippedStacksCatalogMatchesThisRepositorys(t *testing.T) {
	var shipped map[string]any
	raw := NeutralIndexTemplates()[".pose/indexes/validation-matrix.json"]
	if raw == nil {
		t.Fatal("no shipped validation-matrix.json template")
	}
	if err := json.Unmarshal(raw, &shipped); err != nil {
		t.Fatalf("the shipped template is not valid JSON: %v", err)
	}

	// Four levels up from internal/scaffold/distpolicy is the distribution root.
	livePath := filepath.Join("..", "..", "..", "..", ".pose", "indexes", "validation-matrix.json")
	liveRaw, err := os.ReadFile(livePath)
	if err != nil {
		t.Skipf("this repository's own matrix is not present at %s: %v", livePath, err)
	}
	var live map[string]any
	if err := json.Unmarshal(liveRaw, &live); err != nil {
		t.Fatalf("this repository's matrix is not valid JSON: %v", err)
	}

	// `moduleOverrides` and `defaults` are deliberately different: one describes
	// this repository, the other is the neutral shell. `stacks` and
	// `deliveryProfiles` are the generic catalog, and must not diverge.
	for _, key := range []string{"stacks", "deliveryProfiles"} {
		if !reflect.DeepEqual(shipped[key], live[key]) {
			t.Errorf("%q differs between the shipped template in distpolicy.go and .pose/indexes/validation-matrix.json — an instance would receive a catalog this repository does not use; update both", key)
		}
	}
}
