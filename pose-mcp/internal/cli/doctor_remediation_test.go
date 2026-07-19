package cli

// Doctor-guided remediation (spec pose-doctor-guided-remediation): every
// finding carries a stable code/severity/evidence/remediation (R1), machine
// output distinguishes fixable/detectable/blocked (R2), and safe fixes
// support preview, confirmation, idempotency and recheck (R3) — all under
// a strict default-to-advice-or-dry-run posture.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

func TestClassifyFinding(t *testing.T) {
	cases := []struct {
		check, level, wantClass, wantFixCode string
	}{
		{"binary", "ok", remediationNA, ""},
		{"deps.git", "ok", remediationNA, ""},
		{"deps.git", "error", remediationBlocked, ""},
		{"deps.go", "warn", remediationBlocked, ""},
		{"hooks.pre-commit", "warn", remediationFixable, "hooks.pre-commit"},
		{"mcp.config", "warn", remediationFixable, "mcp.config"},
		{"skills.symlinks", "error", remediationFixable, "skills.symlinks"},
		{"schema.version", "warn", remediationDetectable, ""},
		{"schema.version", "error", remediationDetectable, ""},
		{"instance.pose-dir", "error", remediationDetectable, ""},
	}
	for _, c := range cases {
		gotClass, gotFixCode := classifyFinding(c.check, c.level)
		if gotClass != c.wantClass || gotFixCode != c.wantFixCode {
			t.Errorf("classifyFinding(%q, %q) = (%q, %q), want (%q, %q)", c.check, c.level, gotClass, gotFixCode, c.wantClass, c.wantFixCode)
		}
	}
}

func TestRedactSecretShapedContent(t *testing.T) {
	fakeAWSKeyShapedFixture := "AKIA" + "ABCDEFGHIJKLMNOP"
	redacted := redactSecretShapedContent("evidence: " + fakeAWSKeyShapedFixture)
	if strings.Contains(redacted, fakeAWSKeyShapedFixture) {
		t.Errorf("secret-shaped content leaked through redaction: %q", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Errorf("expected a redaction marker: %q", redacted)
	}
}

// doctorFixture installs a native instance with --skip-mcp, which leaves
// exactly two fixable findings: no pre-commit hook, no .mcp.json.
func doctorFixture(t *testing.T) string {
	t.Helper()
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install fixture exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	return repo
}

type doctorJSON struct {
	DoctorSchemaVersion int              `json:"doctor_schema_version"`
	Findings            []doctorFinding  `json:"findings"`
	Errors              int              `json:"errors"`
	Warnings            int              `json:"warnings"`
	Fix                 *json.RawMessage `json:"fix,omitempty"`
}

func TestDoctorFindingsHaveEvidenceAndRemediationClass(t *testing.T) {
	repo := doctorFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("doctor --json exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		var report doctorJSON
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
		if report.DoctorSchemaVersion != doctorSchemaVersion {
			t.Errorf("doctor_schema_version = %d, want %d", report.DoctorSchemaVersion, doctorSchemaVersion)
		}
		if len(report.Findings) == 0 {
			t.Fatal("expected at least one finding")
		}
		for _, f := range report.Findings {
			if f.Check == "" {
				t.Errorf("finding missing stable code: %+v", f)
			}
			if f.Level != "ok" && f.Level != "warn" && f.Level != "error" {
				t.Errorf("finding has invalid severity: %+v", f)
			}
			if f.Evidence == "" {
				t.Errorf("finding missing evidence: %+v", f)
			}
			switch f.RemediationClass {
			case remediationNA, remediationFixable, remediationDetectable, remediationBlocked:
			default:
				t.Errorf("finding has invalid remediation_class: %+v", f)
			}
			if f.Level == "ok" && f.RemediationClass != remediationNA {
				t.Errorf("ok finding must classify as n/a: %+v", f)
			}
			if f.Level != "ok" && f.RemediationClass == remediationNA {
				t.Errorf("non-ok finding must not classify as n/a: %+v", f)
			}
			if f.RemediationClass == remediationFixable && f.FixCode == "" {
				t.Errorf("fixable finding missing fix_code: %+v", f)
			}
		}
	})
}

func TestDoctorFixPreviewIsNonMutating(t *testing.T) {
	repo := doctorFixture(t)
	before := snapshotTree(t, repo)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--fix"}, &out, &errB); code != 0 {
			t.Fatalf("doctor --fix (preview) exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "DRY-RUN") {
			t.Errorf("expected a DRY-RUN marker: %s", out.String())
		}
		for _, want := range []string{"hooks.pre-commit", "mcp.config"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("expected preview to list %q: %s", want, out.String())
			}
		}
	})
	after := snapshotTree(t, repo)
	added, removed, changed := diffSnapshots(before, after)
	if len(added) != 0 || len(removed) != 0 || len(changed) != 0 {
		t.Errorf("preview must not mutate the tree: added=%v removed=%v changed=%v", added, removed, changed)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err == nil {
		t.Error("preview must not install the pre-commit hook")
	}
	if _, err := os.Stat(filepath.Join(repo, ".mcp.json")); err == nil {
		t.Error("preview must not write .mcp.json")
	}
}

func TestDoctorFixApplyAppliesAndRechecksAndIsIdempotent(t *testing.T) {
	repo := doctorFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--fix", "--yes"}, &out, &errB); code != 0 {
			t.Fatalf("doctor --fix --yes exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "[APPLIED]") || !strings.Contains(out.String(), "Result: SUCCESS") {
			t.Errorf("expected an APPLIED/SUCCESS recheck report: %s", out.String())
		}
	})
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("pre-commit hook was not installed: %v", err)
	}
	mcpJSON, err := os.ReadFile(filepath.Join(repo, ".mcp.json"))
	if err != nil || !strings.Contains(string(mcpJSON), `"command": "pose"`) {
		t.Errorf(".mcp.json was not seeded correctly: content=%q err=%v", mcpJSON, err)
	}

	// Idempotency: nothing left to fix on a second run.
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--fix", "--yes"}, &out, &errB); code != 0 {
			t.Fatalf("second doctor --fix --yes exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "nothing fixable") {
			t.Errorf("expected a no-op report on reapply: %s", out.String())
		}
	})
}

func TestDoctorFixOnlyScopesToOneCheck(t *testing.T) {
	repo := doctorFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--fix", "--yes", "--only", "hooks.pre-commit"}, &out, &errB); code != 0 {
			t.Fatalf("scoped fix exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
	})
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("scoped fix did not install the pre-commit hook: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".mcp.json")); err == nil {
		t.Error("--only hooks.pre-commit must not also fix mcp.config")
	}
}

func TestDoctorFixRejectsInvalidOnlyCode(t *testing.T) {
	repo := doctorFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		code := Main([]string{"doctor", "--fix", "--yes", "--only", "not-a-real-code"}, &out, &errB)
		if code != 2 {
			t.Fatalf("expected usage error, exit=%d", code)
		}
		if !strings.Contains(errB.String(), "not a fixable check code") {
			t.Errorf("expected an invalid-code diagnostic: %s", errB.String())
		}
	})
}

func TestDoctorYesRequiresFix(t *testing.T) {
	repo := doctorFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		code := Main([]string{"doctor", "--yes"}, &out, &errB)
		if code != 2 {
			t.Fatalf("expected usage error, exit=%d", code)
		}
		if !strings.Contains(errB.String(), "--yes requires --fix") {
			t.Errorf("expected a --yes/--fix diagnostic: %s", errB.String())
		}
	})
}

// A stale-but-existing-target symlink is the real-world case
// recreateClaudeSkillSymlinks fixes: the link points somewhere wrong while
// the actual .agents/skills content is intact, so relinking is a genuine,
// safe repair (not a no-op masking a still-missing target).
func TestDoctorFixSkillsSymlinks(t *testing.T) {
	repo := doctorFixture(t)
	var staleName string
	for name := range scaffold.ClaudeSkillLinks {
		staleName = name
		break
	}
	link := filepath.Join(repo, ".claude", "skills", staleName)
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../nonexistent-decoy-target", link); err != nil {
		t.Fatal(err)
	}

	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 1 {
			t.Fatalf("expected the dangling symlink to fail doctor, exit=%d out=%s", code, out.String())
		}
		var report doctorJSON
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
		found := false
		for _, f := range report.Findings {
			if f.Check == "skills.symlinks" {
				found = true
				if f.RemediationClass != remediationFixable || f.FixCode != "skills.symlinks" {
					t.Errorf("skills.symlinks should be fixable: %+v", f)
				}
			}
		}
		if !found {
			t.Fatal("expected a skills.symlinks finding")
		}

		out.Reset()
		errB.Reset()
		if code := Main([]string{"doctor", "--fix", "--yes", "--only", "skills.symlinks"}, &out, &errB); code != 0 {
			t.Fatalf("skills.symlinks fix exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "Result: SUCCESS") {
			t.Errorf("expected the recheck to confirm success: %s", out.String())
		}
	})

	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != scaffold.ClaudeSkillLinks[staleName] {
		t.Errorf("symlink target = %q, want %q", target, scaffold.ClaudeSkillLinks[staleName])
	}
}
