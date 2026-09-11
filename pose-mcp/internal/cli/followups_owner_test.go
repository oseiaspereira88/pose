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

// A follow-up may wrap across lines. Before this was handled, only the first
// line was read: the text was truncated and a wrapped "(owner:… review:…)"
// group vanished, leaving the item silently unowned with no diagnostic
// (spec pose-release-cycle-debt-closure, R4).
func TestFollowupMetadataSurvivesLineWrapping(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".pose", "specs", "wrapped", "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nslug: wrapped\nstatus: done\ncompleted_at: 2026-07-01\n---\n\n" +
		"## 7. Final Report\n\n### Follow-ups\n\n" +
		"- [open] a follow-up whose text runs on for a while and\n" +
		"  therefore wraps onto a second line before its metadata\n" +
		"  (owner:@core crit:high review:2999-01-01)\n" +
		"- [open] a single-line one (owner:@core crit:low review:2999-01-01)\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := collectFollowups(root)
	var wrapped *followup
	for i := range entries {
		if strings.HasPrefix(entries[i].Text, "a follow-up whose text runs on") {
			wrapped = &entries[i]
		}
	}
	if wrapped == nil {
		t.Fatalf("the wrapped follow-up was not collected: %+v", entries)
	}
	if wrapped.Owner != "@core" {
		t.Errorf("wrapped metadata must still be parsed, got owner=%q", wrapped.Owner)
	}
	if wrapped.Criticality != "high" || wrapped.Review != "2999-01-01" {
		t.Errorf("wrapped crit/review lost: crit=%q review=%q", wrapped.Criticality, wrapped.Review)
	}
	if !strings.Contains(wrapped.Text, "wraps onto a second line") {
		t.Errorf("continuation lines must join the text, got: %q", wrapped.Text)
	}
	if strings.Contains(wrapped.Text, "owner:") {
		t.Errorf("the metadata group must be stripped from the text, got: %q", wrapped.Text)
	}
}

func TestFollowupsCollectsFlatSpecFiles(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, ".pose", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nslug: flat-spec\nstatus: done\ncompleted_at: 2026-07-01\n---\n\n" +
		"## 7. Final Report\n\n### Follow-ups\n\n" +
		"- [open] flat follow-up (owner:@core crit:medium review:2999-01-01)\n"
	if err := os.WriteFile(filepath.Join(specsDir, "2026-08-23-flat-spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := collectFollowups(root)
	if len(entries) != 1 {
		t.Fatalf("expected 1 follow-up from flat spec, got %d: %+v", len(entries), entries)
	}
	if entries[0].Spec != "flat-spec" || entries[0].Owner != "@core" {
		t.Fatalf("unexpected collected follow-up: %+v", entries[0])
	}
}

// The lint and `pose followups` must read a follow-up the same way. The lint
// read only a bullet's first line, so a wrapped ownership group — which `pose
// followups` reads as owned — was reported as unowned at closeout (spec
// pose-one-follow-up-format).
func TestLintReadsAWrappedOwnershipGroupLikeFollowups(t *testing.T) {
	spec := ownershipLintBase("done") +
		"- [open] a follow-up whose text runs on for a while and\n" +
		"  wraps before its metadata\n" +
		"  (owner:@core crit:medium review:2999-01-01)\n"
	rc, output := lintFixture(t, spec)
	if rc != 0 {
		t.Fatalf("a wrapped, owned follow-up must lint clean: %s", output)
	}
	if strings.Contains(output, "unowned") {
		t.Errorf("the lint read a wrapped ownership group as unowned: %s", output)
	}
}

// Ownership written any way but the trailing group is ignored by the parser,
// so the item reads as unowned with no crit and no review date. The lint names
// the one format that works, in any status — before closeout an unowned item
// is not reported, but metadata nothing reads is.
func TestLintWarnsOwnershipOutsideTheTrailingGroup(t *testing.T) {
	dashed := "- [open] something left — owner:unowned crit:low review:2999-01-01\n"
	for _, status := range []string{"in-progress", "done"} {
		rc, output := lintFixture(t, ownershipLintBase(status)+dashed)
		if rc != 0 {
			t.Fatalf("%s: misplaced ownership must warn, not fail: %s", status, output)
		}
		if !strings.Contains(output, "outside the trailing group") || !strings.Contains(output, "(owner:@alias crit:low|medium|high review:YYYY-MM-DD)") {
			t.Errorf("%s: expected the misplaced-ownership warning naming the format, got: %s", status, output)
		}
	}
	canonical := "- [open] something left (owner:unowned crit:low review:2999-01-01)\n"
	if _, output := lintFixture(t, ownershipLintBase("in-progress")+canonical); strings.Contains(output, "outside the trailing group") {
		t.Errorf("the canonical group was reported as misplaced: %s", output)
	}
}

func TestFollowupMetaMisplaced(t *testing.T) {
	for text, want := range map[string]bool{
		"left over (owner:@a crit:low review:2026-01-01)":             false,
		"left over — owner:@a crit:low review:2026-01-01":             true,
		"left over (owner:@a crit:low review:2026-01-01), then prose": true,
		"no metadata at all": false,
	} {
		if got := followupMetaMisplaced(text); got != want {
			t.Errorf("followupMetaMisplaced(%q) = %v, want %v", text, got, want)
		}
	}
}

func ownershipLintBase(status string) string {
	completed := ""
	if status == "done" {
		completed = "2026-07-02"
	}
	return "---\nslug: fixture\nstatus: " + status + "\ncreated_at: 2026-07-01\ncompleted_at: " + completed + "\n---\n\n" +
		"## 1. Intent\nContent.\n## 2. Requirements\n- R1: behave.\n## 3. Technical Plan\nContent.\n## 4. Tasks\n- [x] done\n" +
		"## 6. Validation\n### Requirement trace\n- R1 [satisfied] check:test\n## 7. Final Report\n### Follow-ups\n"
}
