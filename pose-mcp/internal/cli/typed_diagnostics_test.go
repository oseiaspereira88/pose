package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-typed-producer-diagnostics: closeout-check JSON carries typed
// diagnostics and a typed next step through Main.
func TestCloseoutCheckJSONCarriesTypedDiagnostics(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	var out, errB bytes.Buffer
	inDir(t, root, func() { Main([]string{"closeout-check", "spec:bundle", "--json"}, &out, &errB) })
	var state posemodel.CloseoutState
	if err := json.Unmarshal(out.Bytes(), &state); err != nil {
		t.Fatalf("%v: %s %s", err, out.String(), errB.String())
	}
	found := false
	for _, d := range state.Diagnostics {
		found = found || d.Code == "lifecycle-not-done"
	}
	if !found || state.NextStep == nil || state.NextStep.Code == "" {
		t.Fatalf("closeout-check JSON lacks typed diagnostics: %+v next=%+v", state.Diagnostics, state.NextStep)
	}
}
