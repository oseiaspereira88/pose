package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-commit-gate-refuses-index-without-descriptor.

func checkOutput(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := cmdCheck(root, args, &out, &errb)
	return code, out.String() + errb.String()
}

// R1/R2/R3: an index set whose descriptor no longer describes its files is an error of the strict gate
// (a warning of the tolerant one) naming `pose index`; the repaired set and a repository with no
// generation pass that check.
func TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if _, out := checkOutput(t, root); strings.Contains(out, "index set unavailable") || strings.Contains(out, "differ from generation") {
		t.Fatalf("a repository with no generation is reported: %s", out)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	if _, out := checkOutput(t, root); strings.Contains(out, "index set unavailable") || strings.Contains(out, "differ from generation") {
		t.Fatalf("a whole set is reported: %s", out)
	}
	// A hand-edited file while the generation is whole: the divergence branch, strict and tolerant.
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "services.json"), []byte("[{\"edited\":true}]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	strictCode, strictOut := checkOutput(t, root)
	if !strings.Contains(strictOut, "index files differ from generation") || !strings.Contains(strictOut, "services.json") || !strings.Contains(strictOut, "pose index") {
		t.Fatalf("strict check does not name the diverged file: %s", strictOut)
	}
	if strictCode == 0 {
		t.Fatalf("strict check passed with a diverged index file: %s", strictOut)
	}
	_, tolerantDiverged := checkOutput(t, root, "--tolerant")
	if !findingIsAWarning(tolerantDiverged, "index files differ from generation") {
		t.Fatalf("the tolerant gate reports the divergence as something other than a warning: %s", tolerantDiverged)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("pose index over the diverged set: %d %s", code, errOut)
	}
	// What a commit of regenerated indexes without their descriptor leaves in a checkout: the
	// generation directory is not committed and the legacy copy no longer matches.
	if err := os.RemoveAll(filepath.Join(root, ".pose", "indexes", "generations")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "spec-graph.json"), []byte("{\"edited\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := checkOutput(t, root)
	if !strings.Contains(out, "index set unavailable") || !strings.Contains(out, "spec-graph.json") || !strings.Contains(out, "pose index") {
		t.Fatalf("strict check does not name the unavailable set: %s", out)
	}
	if code == 0 {
		t.Fatalf("strict check passed with an unavailable index set: %s", out)
	}
	// The tolerant gate (a commit hook) warns and does not fail on this finding.
	_, tolerant := checkOutput(t, root, "--tolerant")
	if !findingIsAWarning(tolerant, "index set unavailable") {
		t.Fatalf("the tolerant gate reports it as something other than a warning: %s", tolerant)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("pose index over the unavailable set: %d %s", code, errOut)
	}
	if _, out := checkOutput(t, root); strings.Contains(out, "index set unavailable") {
		t.Fatalf("the repaired set is still reported: %s", out)
	}
}

// findingIsAWarning reports whether the line of the check output that carries the finding is a
// warning ("[!] warning ...") and not an error.
func findingIsAWarning(output, finding string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, finding) {
			return strings.HasPrefix(strings.TrimSpace(line), "[!] warning") || strings.HasPrefix(strings.TrimSpace(line), "[!]")
		}
	}
	return false
}
