package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestCmdDocsInit_ScaffoldsManifest(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsInit(root, []string{"--profile", "cli"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	store := pose.Store{Root: root}
	if !store.HasDocsManifest() {
		t.Fatal("expected manifest to be created")
	}
	manifest, err := store.LoadDocsManifest()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Profile != "cli" || len(manifest.Roots) == 0 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func TestCmdDocsInit_RejectsUnknownProfile(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsInit(root, []string{"--profile", "bogus"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestCmdDocsInit_RefusesToOverwriteExisting(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsInit(root, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("first init: expected exit 0, got %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdDocsInit(root, nil, &stdout, &stderr); code != 1 {
		t.Fatalf("expected exit 1 on re-init, got %d", code)
	}
}

func TestCmdDocsCheck_NoManifestErrors(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsCheck(root, nil, &stdout, &stderr); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestCmdDocsCheck_ExplainKnownRule(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsCheck(root, []string{"--explain", "undeclared"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("expected explanation text")
	}
}

func TestCmdDocsCheck_ExplainUnknownRuleErrors(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cmdDocsCheck(root, []string{"--explain", "bogus"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func writeCLIDocsFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCmdDocsCheck_ReportsUndeclaredAndExitsZeroOnWarningsOnly(t *testing.T) {
	root := t.TempDir()
	writeCLIDocsFixture(t, root, "docs/orphan.md", "---\ntitle: Orphan\ndoc_type: reference\n---\nbody\n")
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))

	var stdout, stderr bytes.Buffer
	code := cmdDocsCheck(root, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 (warnings only), got %d, stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("undeclared")) {
		t.Fatalf("expected undeclared mentioned in output, got %s", stdout.String())
	}
}

func TestCmdDocsCheck_ExitsOneOnErrors(t *testing.T) {
	root := t.TempDir()
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{
		{Path: "docs/missing.md", DocType: "reference"},
	}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))

	var stdout, stderr bytes.Buffer
	if code := cmdDocsCheck(root, nil, &stdout, &stderr); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestCmdDocsCheck_JSONOutputParses(t *testing.T) {
	root := t.TempDir()
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))

	var stdout, stderr bytes.Buffer
	if code := cmdDocsCheck(root, []string{"--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	var result pose.DocsCheckResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, stdout.String())
	}
	if result.SchemaVersion != pose.DocsManifestSchema {
		t.Fatalf("unexpected schema_version: %+v", result)
	}
}

func TestCheckStrict_IncorporatesDocsCheckWhenManifestPresent(t *testing.T) {
	root := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("cmdInstall failed: %d, stderr=%s", code, installErr.String())
	}
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{
		{Path: "docs/missing.md", DocType: "reference"},
	}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))

	var stdout, stderr bytes.Buffer
	code := cmdCheck(root, []string{"--strict"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected pose check --strict to fail on a broken docs manifest, got exit=%d stdout=%s", code, stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("docs:")) {
		t.Fatalf("expected docs issue mentioned in check output, got %s", stdout.String())
	}
}

func TestCheckStrict_NoDocsManifestIsNoOp(t *testing.T) {
	root := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("cmdInstall failed: %d, stderr=%s", code, installErr.String())
	}
	var stdout, stderr bytes.Buffer
	if code := cmdCheck(root, []string{"--strict"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit 0 without a docs manifest, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}
