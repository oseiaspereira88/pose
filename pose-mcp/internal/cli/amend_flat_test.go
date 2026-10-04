package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-flat-spec-amendments: a flat dated spec amends exactly like a
// folder spec, with its own journal, and folder journals are untouched.
func TestAmendWorksForFlatSpecsWithAJournalPerSpec(t *testing.T) {
	root := t.TempDir()
	specs := filepath.Join(root, ".pose", "specs")
	flatA := filepath.Join(specs, "2026-10-04-amended.md")
	flatB := filepath.Join(specs, "2026-10-04-sibling.md")
	mustWrite(t, flatA, amendSpec)
	mustWrite(t, flatB, strings.Replace(amendSpec, "slug: amended", "slug: sibling", 1))
	folder, folderPath := amendFixture(t)
	_ = folder

	if code, out := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatalf("flat baseline exit=%d: %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(specs, "2026-10-04-amended.amendments.jsonl")); err != nil {
		t.Fatalf("flat journal not written beside the spec: %v", err)
	}
	if _, err := os.Stat(filepath.Join(specs, "amendments.jsonl")); err == nil {
		t.Fatal("a shared journal was written for flat specs")
	}
	if code, out := runAmend(t, root, "sibling", "--list"); code != 0 || strings.Contains(out, "baseline") && strings.Contains(out, "@core") {
		t.Fatalf("the sibling flat spec sees the other spec's amendments: exit=%d %s", code, out)
	}

	// The same material change is acknowledged and gated as for a folder spec.
	mutated := strings.Replace(amendSpec, "- R2: keep behaving.", "- R2: do something else entirely.", 1)
	mustWrite(t, flatA, mutated)
	var o, e bytes.Buffer
	if rc := lintOneSpec(flatA, false, false, &o, &e); rc == 0 || !strings.Contains(o.String(), "R2 changed after its last acknowledged amendment") {
		t.Fatalf("unacknowledged change on a flat spec must fail lint: %s", o.String()+e.String())
	}
	if code, out := runAmend(t, root, "amended", "--ids", "R2", "--change", "semantic", "--rationale", "scope pivot", "--author", "@core"); code != 0 {
		t.Fatalf("flat amend exit=%d: %s", code, out)
	}
	o.Reset()
	e.Reset()
	if rc := lintOneSpec(flatA, false, false, &o, &e); rc != 0 {
		t.Fatalf("acknowledged flat change must pass: %s", o.String()+e.String())
	}

	// Folder layout unchanged: journal stays the sibling amendments.jsonl.
	folderRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(folderPath))))
	if code, out := runAmend(t, folderRoot, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatalf("folder baseline exit=%d: %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(folderPath), "amendments.jsonl")); err != nil {
		t.Fatalf("folder journal moved: %v", err)
	}
}

func TestSpecFormatMigrationCarriesAFlatJournal(t *testing.T) {
	root := t.TempDir()
	specs := filepath.Join(root, ".pose", "specs")
	mustWrite(t, filepath.Join(specs, "2026-10-04-amended.md"), amendSpec)
	if code, out := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatalf("baseline exit=%d: %s", code, out)
	}
	var out, errB bytes.Buffer
	inDir(t, root, func() {
		Main([]string{"spec-format", "migrate", "amended", "--format", "folder", "--json"}, &out, &errB)
	})
	var items []SpecMigrationItem
	if err := json.Unmarshal(out.Bytes(), &items); err != nil || len(items) != 1 || !items[0].HasCompanion {
		t.Fatalf("migration did not see the flat journal as a companion: %v %s %s", err, out.String(), errB.String())
	}
	if _, err := os.Stat(filepath.Join(specs, "2026-10-04-amended", "amendments.jsonl")); err != nil {
		t.Fatalf("journal did not travel with the spec: %v", err)
	}
	if code, list := runAmend(t, root, "amended", "--list"); code != 0 || !strings.Contains(list, "baseline") {
		t.Fatalf("history lost after migration: exit=%d %s", code, list)
	}
}
