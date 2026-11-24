package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// treeDigest names every file under root with its content, so any write,
// creation or removal changes it.
func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		rel, _ := filepath.Rel(root, path)
		lines = append(lines, rel+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func assessSealedRoot(t *testing.T) string {
	t.Helper()
	root := assessTestRoot(t)
	writeCloseoutCLIFile(t, root, "svc/main.go", "package main\n\n// TODO: a marker\nfunc main() {}\n")
	writeCloseoutCLIFile(t, root, "svc/go.mod", "module example.com/svc\n\ngo 1.22\n")
	writeCloseoutCLIFile(t, root, ".pose/state/project-state.md", "# Project state\n")
	return root
}

// Spec pose-assess-leaves-sealed-checkouts-alone R2, R4: inside the sealed
// checkout of a delegated review every assessment command leaves the tree
// as it was, and the ones that only write refuse.
func TestAssessLeavesTheSealedCheckoutUnchanged(t *testing.T) {
	root := assessSealedRoot(t)
	t.Setenv(pose.SealedCheckoutEnv, root)
	before := treeDigest(t, root)
	for _, args := range [][]string{
		{"discover"}, {"discover", "--update-state"}, {"discover", "--component", "svc", "--if-stale"},
		{"integrate"}, {"integrate", "--update-state", "--json"},
		{"tech-debt"}, {"tech-debt", "--update-state"},
	} {
		code, out, errOut := runAssess(t, root, args...)
		if code != 0 || !strings.Contains(errOut, "sealed checkout of a delegated review") {
			t.Fatalf("assess %v: %d %s %s", args, code, out, errOut)
		}
		if strings.Contains(out, "Artifacts written") {
			t.Fatalf("assess %v claims it wrote: %s", args, out)
		}
		if after := treeDigest(t, root); after != before {
			t.Fatalf("assess %v changed the sealed checkout:\n%s\n---\n%s", args, before, after)
		}
	}
	for _, sub := range []string{"init", "snapshot", "request"} {
		if code, _, errOut := runAssess(t, root, sub); code == 0 || !strings.Contains(errOut, "sealed review checkout") {
			t.Fatalf("assess %s ran in the sealed checkout: %d %s", sub, code, errOut)
		}
	}
	if after := treeDigest(t, root); after != before {
		t.Fatal("a refused assess command changed the sealed checkout")
	}
	// The writers refuse on their own, for any other caller.
	if err := (pose.Store{Root: root}).SaveTechDebtReport(&pose.TechDebtReportState{}); err != pose.ErrSealedAssessmentWrite {
		t.Fatalf("the tech-debt writer wrote into the sealed checkout: %v", err)
	}
}

// R1: --no-persist prints the result and writes nothing anywhere; without it
// and outside a sealed checkout the records are written as before.
func TestAssessNoPersistWritesNothing(t *testing.T) {
	root := assessSealedRoot(t)
	before := treeDigest(t, root)
	for _, sub := range []string{"discover", "integrate", "tech-debt"} {
		code, out, errOut := runAssess(t, root, sub, "--no-persist", "--update-state")
		if code != 0 || !strings.Contains(errOut, "--no-persist") || out == "" {
			t.Fatalf("assess %s --no-persist: %d %s %s", sub, code, out, errOut)
		}
		if after := treeDigest(t, root); after != before {
			t.Fatalf("assess %s --no-persist wrote", sub)
		}
	}
	if code, _, errOut := runAssess(t, root, "tech-debt"); code != 0 {
		t.Fatal(errOut)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/state/technical-debt.json")); err != nil {
		t.Fatalf("a persisting run outside a sealed checkout wrote nothing: %v", err)
	}
}
