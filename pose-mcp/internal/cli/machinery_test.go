package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
