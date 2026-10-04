package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-effective-governance-projection: reachable through Main.
func TestStateGovernanceIsReachableAndReadOnly(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	mustWrite(t, filepath.Join(root, ".pose/policy/dor.json"), `{"schema_version":1,"adopted_at":""}`)
	var out, errB bytes.Buffer
	inDir(t, root, func() { Main([]string{"state", "--governance"}, &out, &errB) })
	text := out.String()
	for _, want := range []string{"governance.atomic-start=kind=capability supported=true configured=false applicable=false effective=false reasons=not-adopted", "governance.definition-of-ready=", "reasons=no-readiness-cutoff"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s%s", want, text, errB.String())
		}
	}
	out.Reset()
	inDir(t, root, func() { Main([]string{"state", "--governance", "--json"}, &out, &errB) })
	var projection pose.GovernanceProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil || len(projection.Entries) == 0 {
		t.Fatalf("json projection: %v %s", err, out.String())
	}
}
