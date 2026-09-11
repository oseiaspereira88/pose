package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

// A plain upgrade must deliver machinery — that is the whole point of the spec
// — while keeping a backup of whatever the instance had edited, and without
// resurrecting a file the instance deliberately deleted.
func TestDeliverMachineryRefreshesBacksUpAndRespectsDeletion(t *testing.T) {
	target := t.TempDir()
	dist := scaffold.Dist()
	var errB bytes.Buffer

	// First delivery: a fresh instance receives everything.
	if err := deliverMachinery(dist, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	editedPath := filepath.Join(target, ".pose", "rules", "security.md")
	if _, err := os.Stat(editedPath); err != nil {
		t.Fatalf("machinery was not delivered: %v", err)
	}
	manifest := loadMachineryManifest(target)
	if !manifest[".pose/rules/security.md"] {
		t.Error("delivered paths must be recorded in the manifest")
	}

	// Second delivery on an untouched instance changes nothing.
	before, _ := os.ReadFile(editedPath)
	if err := deliverMachinery(dist, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("second delivery: %v", err)
	}
	if _, err := os.Stat(editedPath + ".pose-backup"); err == nil {
		t.Error("an untouched instance must not produce a backup")
	}
	after, _ := os.ReadFile(editedPath)
	if string(before) != string(after) {
		t.Error("redelivery must be idempotent")
	}

	// An edited file is refreshed, and the edit survives as .pose-backup.
	if err := os.WriteFile(editedPath, []byte("# locally edited rule\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errB.Reset()
	if err := deliverMachinery(dist, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("delivery over an edit: %v", err)
	}
	backup, err := os.ReadFile(editedPath + ".pose-backup")
	if err != nil {
		t.Fatalf("an edited file must be backed up before refresh: %v", err)
	}
	if string(backup) != "# locally edited rule\n" {
		t.Errorf("backup lost the instance's content: %q", backup)
	}
	refreshed, _ := os.ReadFile(editedPath)
	if string(refreshed) == "# locally edited rule\n" {
		t.Error("engine content must refresh over a local edit")
	}
	if !strings.Contains(errB.String(), "backed up customized") {
		t.Errorf("the backup must be reported, got: %q", errB.String())
	}

	// A deleted file stays deleted: delivering it again would undo a
	// deliberate decision.
	deletedPath := filepath.Join(target, ".pose", "rules", "delivery-evidence.md")
	if err := os.Remove(deletedPath); err != nil {
		t.Fatal(err)
	}
	if err := deliverMachinery(dist, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("delivery after a deletion: %v", err)
	}
	if _, err := os.Stat(deletedPath); err == nil {
		t.Error("a deliberately removed machinery file must not be resurrected")
	}

	// --force is the escape hatch and restores it.
	if err := deliverMachinery(dist, target, "", true, false, &errB, nil); err != nil {
		t.Fatalf("forced delivery: %v", err)
	}
	if _, err := os.Stat(deletedPath); err != nil {
		t.Error("--force must restore a removed machinery file")
	}
}

// An instance installed in pt-BR must keep receiving pt-BR machinery: the
// locale is a property of the instance, not of the shell running the upgrade.
func TestDeliverMachineryHonoursTheInstanceLocale(t *testing.T) {
	target := t.TempDir()
	dist := scaffold.Dist()
	var errB bytes.Buffer

	if err := deliverMachinery(dist, target, "pt-BR", false, false, &errB, nil); err != nil {
		t.Fatalf("pt-BR delivery: %v", err)
	}
	delivered, err := os.ReadFile(filepath.Join(target, ".pose", "workflows", "review.md"))
	if err != nil {
		t.Fatalf("workflow not delivered: %v", err)
	}
	expected, err := fs.ReadFile(dist, "locales/pt-BR/.pose/workflows/review.md")
	if err != nil {
		t.Skip("distribution carries no pt-BR overlay for this workflow")
	}
	if string(delivered) != string(expected) {
		t.Error("a pt-BR instance must receive the pt-BR overlay, not the English file")
	}

	// Redelivering must not rewrite it back to English or churn a backup.
	if err := deliverMachinery(dist, target, "pt-BR", false, false, &errB, nil); err != nil {
		t.Fatalf("pt-BR redelivery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".pose", "workflows", "review.md.pose-backup")); err == nil {
		t.Error("a localized instance must not back itself up on every upgrade")
	}
}

// A release changes machinery the instance never touched. That file must be
// refreshed quietly: backing it up and calling it "customized" made every
// release read as if local edits were being discarded (spec
// pose-machinery-backs-up-only-local-edits). A file the instance did edit
// still gets its backup.
func TestDeliverMachineryBacksUpOnlyWhatTheInstanceEdited(t *testing.T) {
	target := t.TempDir()
	var errB bytes.Buffer
	older := fstest.MapFS{
		".pose/rules/untouched.md": {Data: []byte("release one\n")},
		".pose/rules/edited.md":    {Data: []byte("release one\n")},
	}
	if err := deliverMachinery(older, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if loadMachineryDigests(target)[".pose/rules/untouched.md"] == "" {
		t.Fatal("the manifest must record the digest of what was delivered")
	}
	edited := filepath.Join(target, ".pose", "rules", "edited.md")
	if err := os.WriteFile(edited, []byte("an instance edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	newer := fstest.MapFS{
		".pose/rules/untouched.md": {Data: []byte("release two\n")},
		".pose/rules/edited.md":    {Data: []byte("release two\n")},
	}
	errB.Reset()
	if err := deliverMachinery(newer, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("second delivery: %v", err)
	}
	untouched := filepath.Join(target, ".pose", "rules", "untouched.md")
	if got, _ := os.ReadFile(untouched); string(got) != "release two\n" {
		t.Errorf("an untouched file must take the new release, got %q", got)
	}
	if _, err := os.Stat(untouched + ".pose-backup"); err == nil {
		t.Error("an untouched file was backed up as if the instance had edited it")
	}
	if strings.Contains(errB.String(), "untouched.md") {
		t.Errorf("an untouched file was reported: %q", errB.String())
	}
	backup, err := os.ReadFile(edited + ".pose-backup")
	if err != nil || string(backup) != "an instance edit\n" {
		t.Fatalf("an edited file must keep its edit as a backup, got %q (%v)", backup, err)
	}
	if !strings.Contains(errB.String(), "backed up customized: .pose/rules/edited.md") {
		t.Errorf("the edited file must be reported as customized, got %q", errB.String())
	}
}

// A manifest written before digests existed cannot tell an edit from an older
// release, so the file is still backed up — and the report says that instead
// of calling it customized. The delivery then records a digest, so the next
// release needs no guess.
func TestDeliverMachineryWithoutADigestSaysWhyItBacksUp(t *testing.T) {
	target := t.TempDir()
	rule := filepath.Join(target, ".pose", "rules", "rule.md")
	if err := os.MkdirAll(filepath.Dir(rule), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rule, []byte("an older release\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveMachineryManifest(target, []string{".pose/rules/rule.md"}, nil); err != nil {
		t.Fatal(err)
	}
	var errB bytes.Buffer
	dist := fstest.MapFS{".pose/rules/rule.md": {Data: []byte("this release\n")}}
	if err := deliverMachinery(dist, target, "", false, false, &errB, nil); err != nil {
		t.Fatalf("delivery: %v", err)
	}
	if _, err := os.Stat(rule + ".pose-backup"); err != nil {
		t.Fatalf("with no digest a local edit cannot be ruled out, so a backup is kept: %v", err)
	}
	if !strings.Contains(errB.String(), "no record of what POSE delivered") || strings.Contains(errB.String(), "customized") {
		t.Errorf("the report must say why it backed up, not claim a customization: %q", errB.String())
	}
	if loadMachineryDigests(target)[".pose/rules/rule.md"] == "" {
		t.Error("the delivery must record a digest for the next release")
	}
}
