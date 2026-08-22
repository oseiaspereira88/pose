package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func writeSpecsFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSpecsCommand_RecentAndFilters(t *testing.T) {
	root := t.TempDir()
	writeSpecsFixture(t, root, ".pose/specs/2026-08-20-auth/spec.md", "---\nslug: auth\nstatus: done\ncreated_at: 2026-08-20\n---\n# Auth\n")
	writeSpecsFixture(t, root, ".pose/specs/2026-08-21-billing.md", "---\nslug: billing\nstatus: in-progress\ncreated_at: 2026-08-21\ncomponents: finance\n---\n# Billing\n")
	writeSpecsFixture(t, root, ".pose/specs/2026-08-01-legacy/spec.md", "---\nslug: legacy\nstatus: draft\ncreated_at: 2026-08-01\n---\n# Legacy\n")

	// 1. Test basic listing
	var stdout, stderr bytes.Buffer
	code := cmdSpecs(root, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdSpecs failed: %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "billing") || !strings.Contains(out, "auth") || !strings.Contains(out, "legacy") {
		t.Errorf("expected all specs in output, got: %s", out)
	}

	// 2. Test --recent 2
	stdout.Reset()
	code = cmdSpecs(root, []string{"--recent", "2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdSpecs --recent 2 failed: %d", code)
	}
	out = stdout.String()
	if !strings.Contains(out, "showing 2") {
		t.Errorf("expected showing 2, got: %s", out)
	}

	// 3. Test --json
	stdout.Reset()
	code = cmdSpecs(root, []string{"--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdSpecs --json failed: %d", code)
	}
	var parsed []pose.Spec
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(parsed) != 3 {
		t.Errorf("expected 3 specs in JSON, got %d", len(parsed))
	}
	// First should be newest (2026-08-21)
	if parsed[0].Slug != "billing" {
		t.Errorf("expected first spec to be billing, got %s", parsed[0].Slug)
	}

	// 4. Test --status in-progress
	stdout.Reset()
	code = cmdSpecs(root, []string{"--status", "in-progress", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdSpecs --status failed: %d", code)
	}
	parsed = nil
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Slug != "billing" {
		t.Errorf("expected 1 in-progress spec (billing), got %+v", parsed)
	}
}

func TestNewSpec_DatePrefixScaffold(t *testing.T) {
	root := t.TempDir()
	writeSpecsFixture(t, root, ".pose/templates/spec.md", "---\nslug: <feature-slug>\nstatus: draft\ncreated_at: <created_at>\n---\n# Spec: <feature-slug>\n")

	var stdout, stderr bytes.Buffer
	// Default: creates flat modern dated spec
	code := cmdNewSpec(root, []string{"checkout-flow"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdNewSpec failed: %d, stderr: %s", code, stderr.String())
	}

	today := time.Now().UTC().Format("2006-01-02")
	expectedFlat := filepath.Join(root, ".pose", "specs", today+"-checkout-flow.md")
	if _, err := os.Stat(expectedFlat); err != nil {
		t.Fatalf("expected default spec at %s: %v", expectedFlat, err)
	}

	store := pose.Store{Root: root}
	sp, err := store.GetSpec("checkout-flow")
	if err != nil {
		t.Fatalf("GetSpec failed to find created dated spec: %v", err)
	}
	if sp.Slug != "checkout-flow" || sp.Status != "draft" {
		t.Errorf("unexpected spec content: %+v", sp)
	}

	// Verify spec-format reports conforming
	stdout.Reset()
	stderr.Reset()
	code = cmdSpecFormatStatus(root, []string{"--json"}, &stdout, &stderr, localeEN)
	if code != 0 {
		t.Fatalf("spec-format status failed: %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"conforming": true`) {
		t.Fatalf("expected conforming: true for fresh spec, got: %s", stdout.String())
	}

	// Folder spec creation
	stdout.Reset()
	code = cmdNewSpec(root, []string{"folder-flow", "--folder"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdNewSpec --folder failed: %d, stderr: %s", code, stderr.String())
	}
	expectedFolder := filepath.Join(root, ".pose", "specs", today+"-folder-flow", "spec.md")
	if _, err := os.Stat(expectedFolder); err != nil {
		t.Fatalf("expected folder spec at %s: %v", expectedFolder, err)
	}

	// Legacy spec creation
	stdout.Reset()
	code = cmdNewSpec(root, []string{"legacy-flow", "--legacy"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdNewSpec --legacy failed: %d, stderr: %s", code, stderr.String())
	}
	expectedLegacy := filepath.Join(root, ".pose", "specs", "legacy-flow", "spec.md")
	if _, err := os.Stat(expectedLegacy); err != nil {
		t.Fatalf("expected legacy spec at %s: %v", expectedLegacy, err)
	}
}
