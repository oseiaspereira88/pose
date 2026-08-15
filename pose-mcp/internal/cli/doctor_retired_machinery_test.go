package cli

// Retired-machinery discoverability (spec pose-domain-rule-extension-migration,
// ADR retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update):
// deliverMachinery only ever walks the engine's *current* machineryRoots, so a
// file the engine used to ship (e.g. backend-go.md/frontend-react.md, now
// extensions) but no longer does is never revisited — it stays on disk,
// untouched and un-updated, with nothing else ever telling the instance so.
// `pose doctor` closes that discoverability gap.

import (
	"path/filepath"
	"strings"
	"testing"
)

func writeMachineryManifest(t *testing.T, root string, paths ...string) {
	t.Helper()
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = `"` + p + `"`
	}
	writeArtifactTestFile(t, root, filepath.Join(".pose", "state", "machinery-manifest.json"),
		`{"schema_version":1,"paths":[`+strings.Join(quoted, ",")+`]}`)
}

func TestDoctorWarnsWhenRetiredMachineryStillOnDisk(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeMachineryManifest(t, root, ".pose/rules/backend-go.md", ".pose/rules/security.md")
	writeArtifactTestFile(t, root, filepath.Join(".pose", "rules", "backend-go.md"), "# Rule: Backend Go\n")
	writeArtifactTestFile(t, root, filepath.Join(".pose", "rules", "security.md"), "# Rule: Security\n")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "machinery.retired-on-disk")
	if !ok {
		t.Fatal("expected a machinery.retired-on-disk finding")
	}
	if f.Level != "warn" || f.RemediationClass != remediationDetectable {
		t.Errorf("machinery.retired-on-disk: level=%q class=%q, want warn/detectable", f.Level, f.RemediationClass)
	}
	if !strings.Contains(f.Message, ".pose/rules/backend-go.md") {
		t.Errorf("machinery.retired-on-disk message does not name the retired path: %q", f.Message)
	}
	if strings.Contains(f.Message, ".pose/rules/security.md") {
		t.Errorf("machinery.retired-on-disk flagged a still-current file as retired: %q", f.Message)
	}
	if !strings.Contains(f.Hint, "pose extension install pose-rule-backend-go") {
		t.Errorf("machinery.retired-on-disk hint does not name the matching extension: %q", f.Hint)
	}
}

func TestDoctorSilentWhenInstanceAlreadyRemovedTheRetiredFile(t *testing.T) {
	root := doctorTrailerFixture(t)
	// Manifest remembers backend-go.md was once delivered, but the instance
	// deleted it on its own — deliverMachinery's own "instance-deleted paths
	// stay deleted" contract means this is not something to flag.
	writeMachineryManifest(t, root, ".pose/rules/backend-go.md")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "machinery.retired-on-disk")
	if !ok {
		t.Fatal("expected a machinery.retired-on-disk finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("machinery.retired-on-disk level=%q, want ok — instance already removed the file itself", f.Level)
	}
}

func TestDoctorSilentWhenNoMachineryManifestExists(t *testing.T) {
	root := doctorTrailerFixture(t)
	// No .pose/state/machinery-manifest.json at all (pre-pose-instance-
	// engine-version-tracking instance, or never updated) — nothing to
	// compare against, must not error or false-positive.
	f, ok := findDoctorFinding(runDoctorJSON(t, root), "machinery.retired-on-disk")
	if ok && f.Level != "ok" {
		t.Errorf("machinery.retired-on-disk level=%q with no manifest, want ok or absent finding", f.Level)
	}
}
