package cli

// Ownership and service-level behavior of follow-ups (spec
// pose-followup-ownership-sla): metadata parsing, overdue projection,
// risk-based blocking and the legacy-unowned migration path.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOwnerFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(slug, followups string) {
		path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nslug: " + slug + "\nstatus: done\ncompleted_at: 2026-07-01\n---\n\n## 7. Final Report\n\n### Follow-ups\n" + followups
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("owned", "- [open] tune cache (owner:@core crit:high review:2026-01-01)\n- [open] fresh review (owner:@core crit:low review:2999-01-01)\n")
	write("legacy", "- [open] no metadata here\n- [done] finished item\n")
	return root
}

func TestFollowupOwnershipProjection(t *testing.T) {
	root := writeOwnerFixture(t)
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-07-19")
	var out, errB bytes.Buffer
	if code := cmdFollowups(root, []string{"--overdue"}, &out, &errB); code != 0 {
		t.Fatalf("followups --overdue exit=%d stderr=%s", code, errB.String())
	}
	s := out.String()
	if !strings.Contains(s, "overdue=1") || !strings.Contains(s, "unowned=1") {
		t.Errorf("header should count overdue=1 unowned=1, got: %s", s)
	}
	if !strings.Contains(s, "tune cache") || strings.Contains(s, "fresh review") {
		t.Errorf("overdue filter should keep only the expired review, got: %s", s)
	}
	if !strings.Contains(s, "OVERDUE") {
		t.Errorf("expired review should be marked OVERDUE, got: %s", s)
	}
}

func TestFollowupFailOverduePolicy(t *testing.T) {
	root := writeOwnerFixture(t)
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-07-19")
	var out, errB bytes.Buffer
	if code := cmdFollowups(root, []string{"--fail-overdue"}, &out, &errB); code != 1 {
		t.Fatalf("--fail-overdue with one expired review should exit 1, got %d", code)
	}
	t.Setenv("POSE_FOLLOWUP_TODAY", "2025-01-01")
	out.Reset()
	if code := cmdFollowups(root, []string{"--fail-overdue"}, &out, &errB); code != 0 {
		t.Fatalf("--fail-overdue with nothing expired should exit 0, got %d", code)
	}
}

func TestFollowupOwnerFilter(t *testing.T) {
	root := writeOwnerFixture(t)
	var out, errB bytes.Buffer
	if code := cmdFollowups(root, []string{"--owner", "@core"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if strings.Contains(out.String(), "no metadata here") {
		t.Errorf("owner filter should exclude unowned entries, got: %s", out.String())
	}
}

func TestFollowupMetaParsing(t *testing.T) {
	cases := []struct {
		text    string
		owner   string
		wantErr string
	}{
		{"do it (owner:@a crit:high review:2026-01-01)", "@a", ""},
		{"do it (owner:@a crit:high review:2026-01-01 by:@b)", "@a", ""},
		{"do it", "unowned", ""},
		{"do it (owner:@a crit:urgent review:2026-01-01)", "@a", "invalid crit"},
		{"do it (owner:@a crit:high review:soon)", "@a", "invalid review date"},
		{"do it (owner:@a)", "@a", "incomplete ownership group"},
		{"do it (owner:@a crit:high review:2026-01-01 color:red)", "@a", "unknown ownership field"},
	}
	for _, c := range cases {
		_, owner, _, _, _, metaErr := parseFollowupMeta(c.text)
		if owner != c.owner {
			t.Errorf("%q: owner = %q, want %q", c.text, owner, c.owner)
		}
		if c.wantErr == "" && metaErr != "" {
			t.Errorf("%q: unexpected error %q", c.text, metaErr)
		}
		if c.wantErr != "" && !strings.Contains(metaErr, c.wantErr) {
			t.Errorf("%q: error = %q, want contains %q", c.text, metaErr, c.wantErr)
		}
	}
}

func TestLintCloseoutOwnershipGate(t *testing.T) {
	base := `---
slug: fixture
status: done
created_at: 2026-07-01
completed_at: 2026-07-02
---

## 1. Intent
Content.
## 2. Requirements
- R1: behave.
## 3. Technical Plan
Content.
## 4. Tasks
- [x] done
## 6. Validation
### Requirement trace
- R1 [satisfied] check:test
## 7. Final Report
### Follow-ups
`
	malformed := base + "- [open] broken meta (owner:@a crit:urgent review:2026-01-01)\n"
	rc, output := lintFixture(t, malformed)
	if rc == 0 {
		t.Fatal("malformed ownership metadata on a done spec must fail")
	}
	if !strings.Contains(output, "invalid crit") {
		t.Errorf("expected crit diagnostic, got: %s", output)
	}
	legacy := base + "- [open] legacy unowned item\n"
	rc, output = lintFixture(t, legacy)
	if rc != 0 {
		t.Fatalf("legacy unowned open follow-up must warn, not fail: %s", output)
	}
	if !strings.Contains(output, "unowned") {
		t.Errorf("expected unowned warning, got: %s", output)
	}
}
