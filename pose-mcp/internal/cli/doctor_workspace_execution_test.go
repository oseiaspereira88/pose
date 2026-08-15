package cli

// Redundant root-plus-child monorepo execution (spec
// pose-monorepo-validation-advisory, github.com/oseiaspereira88/pose
// issue #23). moduleOverrides.<path>.replaceDefaultChecks already fixes
// this with zero code change — this check exists purely for
// discoverability.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorWarnsOnRedundantWorkspaceExecution(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, "package.json"), `{"name":"root","scripts":{"test":"npm test --workspaces"},"workspaces":["packages/*"]}`)
	mustWrite(t, filepath.Join(root, "packages", "foo", "package.json"), `{"name":"foo"}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"stacks":{"node":{"checks":[{"name":"test","program":"npm","args":["test","--workspaces"],"severity":"required"}]}},"moduleOverrides":{}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.redundant-workspace-execution")
	if !ok {
		t.Fatal("expected a validate.redundant-workspace-execution finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn", f.Level)
	}
	if !strings.Contains(f.Message, "packages/foo") {
		t.Errorf("message does not name the redundant child: %q", f.Message)
	}
	if !strings.Contains(f.Hint, "replaceDefaultChecks") {
		t.Errorf("hint does not name the fix: %q", f.Hint)
	}
}

func TestDoctorSilentWhenModuleOverrideAlreadyDeclared(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, "package.json"), `{"name":"root","scripts":{"test":"npm test --workspaces"},"workspaces":["packages/*"]}`)
	mustWrite(t, filepath.Join(root, "packages", "foo", "package.json"), `{"name":"foo"}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"stacks":{"node":{"checks":[{"name":"test","program":"npm","args":["test","--workspaces"],"severity":"required"}]}},"moduleOverrides":{"packages/foo":{"replaceDefaultChecks":true,"checks":[]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.redundant-workspace-execution")
	if !ok {
		t.Fatal("expected a validate.redundant-workspace-execution finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok — the fix is already declared", f.Level)
	}
}

func TestDoctorSilentWhenRootDoesNotDelegate(t *testing.T) {
	root := doctorTrailerFixture(t)
	// A root package.json with a workspaces field whose OWN script does
	// NOT delegate — mere presence of "workspaces" is not the signal
	// (confirmed empirically during issue #23's investigation).
	mustWrite(t, filepath.Join(root, "package.json"), `{"name":"root","scripts":{"test":"echo root-only"},"workspaces":["packages/*"]}`)
	mustWrite(t, filepath.Join(root, "packages", "foo", "package.json"), `{"name":"foo"}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"stacks":{"node":{"checks":[{"name":"test","program":"npm","args":["run","test"],"severity":"required"}]}},"moduleOverrides":{}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.redundant-workspace-execution")
	if !ok {
		t.Fatal("expected a validate.redundant-workspace-execution finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok — root script does not delegate", f.Level)
	}
}
