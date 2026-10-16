package cli

import (
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// The binary keeps recording its own version in assessment bindings after the
// engine package stopped importing internal/version
// (spec pose-engine-package-stays-mirrorable).
func TestBinaryRecordsItsVersionInAssessmentBindings(t *testing.T) {
	if posemodel.EngineVersion != version.Version {
		t.Fatalf("pose.EngineVersion = %q, want the binary's %q", posemodel.EngineVersion, version.Version)
	}
	binding, _ := (posemodel.Store{Root: t.TempDir()}).CurrentAssessmentBinding(".")
	if binding.EngineVersion != version.Version {
		t.Fatalf("binding engine_version = %q, want %q", binding.EngineVersion, version.Version)
	}
}
