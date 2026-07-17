package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeImportFixture(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func prepareImportRepo(t *testing.T) string {
	t.Helper()
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestImportSpecKitDryRunThenWrite(t *testing.T) {
	repo := prepareImportRepo(t)
	source := filepath.Join(repo, "legacy")
	writeImportFixture(t, source, "specs/001-user-auth/spec.md", `# Feature Specification: User Authentication

**Created**: 2026-07-10
**Input**: User description: "Sign in securely"

## User Scenarios & Testing

### User Story 1 - Sign in
Users sign in with valid credentials.

## Requirements

- FR-001: System MUST authenticate valid credentials.
- FR-002: System MUST reject invalid credentials.

## Success Criteria

- SC-001: Sign-in completes within two seconds.
`)
	writeImportFixture(t, source, "specs/001-user-auth/plan.md", "# Implementation Plan: User Authentication\n\n## Summary\nUse the existing identity boundary.\n")
	writeImportFixture(t, source, "specs/001-user-auth/tasks.md", "# Tasks\n\n- [ ] T001 Add the authentication service.\n- [ ] T002 Add contract tests.\n")

	previousNow := importNow
	importNow = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { importNow = previousNow })

	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		code := Main([]string{"import", "spec-kit", source, "--dry-run"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("dry-run exit=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "specs=1 warnings=0 written=0 dry_run=true") {
			t.Fatalf("unexpected dry-run report: %s", out.String())
		}
		if _, err := os.Stat(filepath.Join(repo, ".pose", "specs", "user-auth")); !os.IsNotExist(err) {
			t.Fatalf("dry-run wrote destination: %v", err)
		}

		out.Reset()
		code = Main([]string{"import", "spec-kit", source}, &out, &errOut)
		if code != 0 {
			t.Fatalf("import exit=%d stderr=%s", code, errOut.String())
		}
		out.Reset()
		errOut.Reset()
		code = Main([]string{"lint-spec", "user-auth", "--ready-check"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("generated spec is not ready: exit=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
		}
	})

	destination := filepath.Join(repo, ".pose", "specs", "user-auth", "spec.md")
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"slug: user-auth", "created_at: 2026-07-17", "R1: FR-001", "R2: FR-002",
		"User Story 1 - Sign in", "SC-001: Sign-in completes", "Use the existing identity boundary", "T001 Add the authentication service",
		"Format: `spec-kit`", "specs/001-user-auth/spec.md",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("generated spec missing %q", want)
		}
	}
}

func TestImportOpenSpecChangePreservesDeltaAndScenario(t *testing.T) {
	repo := prepareImportRepo(t)
	change := filepath.Join(repo, "openspec", "changes", "add-2fa")
	writeImportFixture(t, change, "proposal.md", "# Add Two-Factor Authentication\n\n## Why\nReduce account takeover risk.\n")
	writeImportFixture(t, change, "design.md", "# Design\n\nUse time-based one-time passwords.\n")
	writeImportFixture(t, change, "tasks.md", "# Tasks\n\n- [ ] Add enrollment.\n- [ ] Add verification.\n")
	writeImportFixture(t, change, "specs/auth/spec.md", `# Authentication Specification

## ADDED Requirements

### Requirement: Two-factor challenge
The system SHALL request a second factor after password authentication.

#### Scenario: Valid one-time password
- **WHEN** the user submits a valid code
- **THEN** access is granted
`)

	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"import", "openspec", change}, &out, &errOut); code != 0 {
			t.Fatalf("import exit=%d stderr=%s", code, errOut.String())
		}
	})

	content, err := os.ReadFile(filepath.Join(repo, ".pose", "specs", "add-2fa-auth", "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[ADDED] Two-factor challenge", "Scenario: Valid one-time password",
		"Reduce account takeover risk", "Use time-based one-time passwords", "Add enrollment",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("generated OpenSpec import missing %q", want)
		}
	}
}

func TestImportPreflightCollisionLeavesBatchUntouched(t *testing.T) {
	repo := prepareImportRepo(t)
	source := filepath.Join(repo, "legacy", "specs")
	for _, slug := range []string{"001-alpha", "002-beta"} {
		writeImportFixture(t, source, slug+"/spec.md", "# Feature Specification: "+slug+"\n\n## Requirements\n\n- FR-001: System MUST work.\n")
	}
	existing := writeImportFixture(t, repo, ".pose/specs/beta/spec.md", "existing\n")

	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		code := Main([]string{"import", "spec-kit", source}, &out, &errOut)
		if code != 1 || !strings.Contains(errOut.String(), "destination already exists") {
			t.Fatalf("collision exit=%d stderr=%s", code, errOut.String())
		}
	})
	if _, err := os.Stat(filepath.Join(repo, ".pose", "specs", "alpha")); !os.IsNotExist(err) {
		t.Fatalf("preflight failure left partial alpha spec: %v", err)
	}
	if content, err := os.ReadFile(existing); err != nil || string(content) != "existing\n" {
		t.Fatalf("existing destination changed: %q err=%v", content, err)
	}
}

func TestImportRejectsSymlinkAndMalformedOpenSpec(t *testing.T) {
	repo := prepareImportRepo(t)
	target := writeImportFixture(t, repo, "target/spec.md", "# Feature Specification: Target\n\n- FR-001: System MUST work.\n")
	symlink := filepath.Join(repo, "linked-spec.md")
	if err := os.Symlink(target, symlink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"import", "spec-kit", symlink}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "symlink") {
			t.Fatalf("symlink exit=%d stderr=%s", code, errOut.String())
		}
	})

	malformed := writeImportFixture(t, repo, "openspec/specs/auth/spec.md", "# Auth Specification\n\n## Purpose\nAuthentication.\n")
	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"import", "openspec", malformed}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "no '### Requirement:'") {
			t.Fatalf("malformed exit=%d stderr=%s", code, errOut.String())
		}
	})
}

func TestImportUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{
		{"import"},
		{"import", "unknown", "spec.md"},
		{"import", "spec-kit", "spec.md", "--force"},
	} {
		var out, errOut bytes.Buffer
		if code := Main(args, &out, &errOut); code != 2 {
			t.Errorf("Main(%v) exit=%d, want 2 (stderr=%s)", args, code, errOut.String())
		}
	}
}

func TestImportEnforcesByteLimit(t *testing.T) {
	repo := prepareImportRepo(t)
	source := filepath.Join(repo, "large", "spec.md")
	content := "# Feature Specification: Large\n\n## Requirements\n\n- FR-001: System MUST work.\n" + strings.Repeat("x", importMaxBytes)
	writeImportFixture(t, repo, "large/spec.md", content)
	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"import", "spec-kit", source}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "byte limit") {
			t.Fatalf("limit exit=%d stderr=%s", code, errOut.String())
		}
	})
}

func TestImportRollsBackFilesCreatedByCurrentRun(t *testing.T) {
	repo := prepareImportRepo(t)
	units := []importUnit{{slug: "alpha"}, {slug: "beta"}}
	rendered := []string{"alpha", "beta"}
	previousWrite := importWriteFile
	writes := 0
	importWriteFile = func(path string, data []byte, mode os.FileMode) error {
		writes++
		if writes == 2 {
			return errors.New("injected write failure")
		}
		return os.WriteFile(path, data, mode)
	}
	t.Cleanup(func() { importWriteFile = previousWrite })

	if err := writeImportedSpecs(repo, units, rendered); err == nil || !strings.Contains(err.Error(), "injected write failure") {
		t.Fatalf("writeImportedSpecs error=%v", err)
	}
	for _, slug := range []string{"alpha", "beta"} {
		if _, err := os.Stat(filepath.Join(repo, ".pose", "specs", slug)); !os.IsNotExist(err) {
			t.Errorf("rollback left %s behind: %v", slug, err)
		}
	}
}
