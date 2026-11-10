package pose

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Spec pose-index-input-binding-reads-each-transfer-once.

type countingPlanFiles struct {
	Files
	plans *atomic.Int64
}

func (c countingPlanFiles) ReadFile(name string) ([]byte, error) {
	if strings.HasSuffix(name, "plan.json") {
		c.plans.Add(1)
	}
	return c.Files.ReadFile(name)
}

// R1: one operation reads the transfer plans of a project once, however many artifacts it
// resolves; without the operation's snapshots every resolution reads them again.
func TestAnOperationReadsTheTransferPlansOfAProjectOnce(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 4; i++ {
		qualifiedFile(t, root, "specs/n"+string(rune('a'+i))+".md", "---\nslug: n"+string(rune('a'+i))+"\nstatus: done\n---\n")
	}
	for _, id := range []string{"stf-000000000000000000000001", "stf-000000000000000000000002", "stf-000000000000000000000003"} {
		dir := filepath.Join(root, ".pose", "transfers", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	refs := []string{"xref:p/spec:na", "xref:p/spec:nb", "xref:p/spec:nc", "xref:p/spec:nd", "xref:p/spec:na"}
	count := func(snapshots bool) (int64, []string) {
		var plans atomic.Int64
		r := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"p": root}}), FS: countingPlanFiles{Files: NativeFiles(), plans: &plans}}
		if snapshots {
			r = r.WithSpecSnapshots()
		}
		var states []string
		for _, ref := range refs {
			states = append(states, r.Resolve("p", ref).State)
		}
		return plans.Load(), states
	}
	once, withSnapshots := count(true)
	many, without := count(false)
	if once != 3 {
		t.Fatalf("one operation read %d plan files for 3 plans and %d resolutions, want 3", once, len(refs))
	}
	if many != 3*int64(len(refs)) && many <= once {
		t.Fatalf("without snapshots %d reads: the comparison does not show the saving", many)
	}
	// What a resolution answers does not depend on whether the listing was shared.
	if strings.Join(withSnapshots, ",") != strings.Join(without, ",") {
		t.Fatalf("the answers differ: %v vs %v", withSnapshots, without)
	}
}
