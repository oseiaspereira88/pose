package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestSpecFormatMigrate_SingleAndAll(t *testing.T) {
	root := t.TempDir()
	writeSpecsFixture(t, root, ".pose/specs/auth/spec.md", "---\nslug: auth\nstatus: done\ncreated_at: 2026-06-01\n---\n# Auth\n")
	writeSpecsFixture(t, root, ".pose/specs/billing.md", "---\nslug: billing\nstatus: in-progress\ncreated_at: 2026-07-15\n---\n# Billing\n")

	// 1. Single spec migration
	var stdout, stderr bytes.Buffer
	code := cmdSpecFormat(root, []string{"migrate", "auth", "--format", "folder"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("migrate auth failed: %d, stderr: %s", code, stderr.String())
	}

	// Verify auth moved to 2026-06-01-auth/spec.md
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "2026-06-01-auth", "spec.md")); err != nil {
		t.Fatalf("expected date-prefixed folder 2026-06-01-auth, got err: %v", err)
	}

	// Verify old folder removed
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "auth")); err == nil {
		t.Errorf("expected old folder .pose/specs/auth to be removed")
	}

	// 2. Batch migration --all
	stdout.Reset()
	code = cmdSpecFormat(root, []string{"migrate", "--all", "--format", "flat"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("migrate --all failed: %d, stderr: %s", code, stderr.String())
	}

	// Verify billing moved to 2026-07-15-billing.md
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "2026-07-15-billing.md")); err != nil {
		t.Fatalf("expected flat file 2026-07-15-billing.md, got err: %v", err)
	}

	// Verify engine can still resolve both
	store := pose.Store{Root: root}
	spAuth, err := store.GetSpec("auth")
	if err != nil || spAuth.Slug != "auth" {
		t.Fatalf("engine failed to resolve migrated auth: %v, %+v", err, spAuth)
	}
	spBilling, err := store.GetSpec("billing")
	if err != nil || spBilling.Slug != "billing" {
		t.Fatalf("engine failed to resolve migrated billing: %v, %+v", err, spBilling)
	}
}

func TestSpecFormatMigrate_AmendmentsPreserved(t *testing.T) {
	root := t.TempDir()
	writeSpecsFixture(t, root, ".pose/specs/feature-x/spec.md", "---\nslug: feature-x\nstatus: done\ncreated_at: 2026-05-10\n---\n# Feature X\n")
	writeSpecsFixture(t, root, ".pose/specs/feature-x/amendments.jsonl", `{"event":"created"}`+"\n")

	// Even when --format flat is requested, companion amendments.jsonl MUST force folder envelope
	var stdout, stderr bytes.Buffer
	code := cmdSpecFormat(root, []string{"migrate", "feature-x", "--format", "flat"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("migrate feature-x failed: %d, stderr: %s", code, stderr.String())
	}

	// Verify migrated to folder envelope
	targetDir := filepath.Join(root, ".pose", "specs", "2026-05-10-feature-x")
	if _, err := os.Stat(filepath.Join(targetDir, "spec.md")); err != nil {
		t.Fatalf("expected folder envelope with spec.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "amendments.jsonl")); err != nil {
		t.Fatalf("expected amendments.jsonl to be preserved in target folder: %v", err)
	}
}

func TestSpecFormatMigrate_DryRun(t *testing.T) {
	root := t.TempDir()
	writeSpecsFixture(t, root, ".pose/specs/preview-demo/spec.md", "---\nslug: preview-demo\nstatus: draft\ncreated_at: 2026-08-10\n---\n# Demo\n")

	var stdout, stderr bytes.Buffer
	code := cmdSpecFormat(root, []string{"migrate", "preview-demo", "--dry-run", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("dry-run failed: %d, stderr: %s", code, stderr.String())
	}

	var results []SpecMigrationItem
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if len(results) != 1 || results[0].Status != "dry-run" {
		t.Fatalf("unexpected dry-run results: %+v", results)
	}

	// Verify original file remained untouched
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "preview-demo", "spec.md")); err != nil {
		t.Fatalf("original file should remain on dry-run: %v", err)
	}
}

func TestSpecFormatMigrate_DateDerivation(t *testing.T) {
	root := t.TempDir()
	// Spec with only completed_at
	writeSpecsFixture(t, root, ".pose/specs/completed-only/spec.md", "---\nslug: completed-only\nstatus: done\ncompleted_at: 2026-04-12\n---\n# Completed\n")

	var stdout, stderr bytes.Buffer
	code := cmdSpecFormat(root, []string{"migrate", "completed-only"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("migrate failed: %d", code)
	}

	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "2026-04-12-completed-only", "spec.md")); err != nil {
		t.Fatalf("expected date from completed_at: %v", err)
	}
}

func TestSpecFormat_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"spec-format", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("spec-format --help exit=%d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "migrate") {
		t.Fatalf("expected migrate subcommand in help output: %s", stdout.String())
	}
}
