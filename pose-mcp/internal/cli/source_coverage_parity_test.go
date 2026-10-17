package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity: the follow-up backlog, the project state and the
// debt report say what they could not read.

func p01Fixture(t *testing.T) string {
	t.Helper()
	root := newGitRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"proj.coverage"}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/2026-10-01-late.md", "---\nslug: late\nstatus: done\ncreated_at: 2026-10-01\n---\n\n# Spec: late\n\n## 7. Final Report\n\n### Follow-ups\n\n- Missing mandatory correction (owner:@core crit:high review:2020-01-01)\n- [open] Confirmed regression (owner:@core crit:high review:2020-01-01)\n")
	return root
}

func followupsJSON(t *testing.T, root string, args ...string) map[string]any {
	t.Helper()
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-10-12")
	var out, errOut bytes.Buffer
	if code := cmdFollowups(root, append(args, "--json"), &out, &errOut); code != 0 {
		t.Fatalf("followups %v: code=%d err=%s", args, code, errOut.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("followups output: %v\n%s", err, out.String())
	}
	return doc
}

// V1 / P01: both unresolved follow-ups are open for the backlog, and the project state agrees.
func TestTheBacklogAndTheProjectStateCountTheSameUnresolvedFollowups(t *testing.T) {
	root := p01Fixture(t)
	doc := followupsJSON(t, root)
	if doc["open"].(float64) != 2 || doc["overdue"].(float64) != 2 || doc["coverage_complete"] != true {
		t.Fatalf("backlog: open=%v overdue=%v complete=%v", doc["open"], doc["overdue"], doc["coverage_complete"])
	}
	state := provideFollowups(root)
	if !strings.Contains(state, "abertos: 2") {
		t.Fatalf("the project state counts differently:\n%s", state)
	}
	candidates := followupsJSON(t, root, "--candidates")
	if list := candidates["candidates"].([]any); len(list) != 2 {
		t.Fatalf("candidates hold %d of the 2 unresolved follow-ups", len(list))
	}
}

// D01: an unreadable spec is reported, with the counts of what was read.
func TestFollowupsReportsASpecItCouldNotRead(t *testing.T) {
	root := p01Fixture(t)
	if err := os.Symlink("absent-target.md", filepath.Join(root, ".pose", "specs", "2026-10-11-unreadable.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	doc := followupsJSON(t, root)
	if doc["coverage_complete"] != false || doc["open"].(float64) != 2 {
		t.Fatalf("complete=%v open=%v", doc["coverage_complete"], doc["open"])
	}
	losses := doc["coverage"].(map[string]any)["losses"].([]any)
	if len(losses) != 1 || losses[0].(map[string]any)["artifact"] != ".pose/specs/2026-10-11-unreadable.md" || losses[0].(map[string]any)["scope"] != "unreadable" {
		t.Fatalf("losses = %v", losses)
	}
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-10-12")
	var out, errOut bytes.Buffer
	if code := cmdFollowups(root, nil, &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	for _, want := range []string{"coverage=INCOMPLETE", "unread read-error .pose/specs/2026-10-11-unreadable.md", "open=2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("the human output lacks %q:\n%s", want, out.String())
		}
	}
	if !strings.Contains(provideFollowups(root), "cobertura: INCOMPLETA") {
		t.Fatalf("the project state hides the loss:\n%s", provideFollowups(root))
	}
	candidates := followupsJSON(t, root, "--candidates")
	if candidates["coverage_complete"] != false {
		t.Fatalf("candidates claim complete coverage: %v", candidates["coverage_complete"])
	}
}

// --fail-overdue cannot certify "none overdue" over a read that lost a spec.
func TestFailOverdueRefusesToCertifyAnIncompleteRead(t *testing.T) {
	root := newGitRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/specs/2026-10-01-fine.md", "---\nslug: fine\nstatus: done\ncreated_at: 2026-10-01\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Later (owner:@core crit:low review:2099-01-01)\n")
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-10-12")
	var out, errOut bytes.Buffer
	if code := cmdFollowups(root, []string{"--fail-overdue"}, &out, &errOut); code != 0 {
		t.Fatalf("a complete read with nothing overdue failed: %d\n%s", code, out.String())
	}
	if err := os.Symlink("absent-target.md", filepath.Join(root, ".pose", "specs", "2026-10-11-unreadable.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	out.Reset()
	if code := cmdFollowups(root, []string{"--fail-overdue"}, &out, &errOut); code != 1 || !strings.Contains(out.String(), "cobertura incompleta") {
		t.Fatalf("code=%d\n%s", code, out.String())
	}
	out.Reset()
	if code := cmdFollowups(root, []string{"--fail-overdue", "--json"}, &out, &errOut); code != 1 {
		t.Fatalf("json code=%d", code)
	}
}

// V5 / P04: the CLI states the scope of the scan.
func TestAssessTechDebtStatesItsScopeAndFindsAMarkerAfterALongLine(t *testing.T) {
	root := newGitRepo(t)
	writeCloseoutCLIFile(t, root, "pkg/sample.go", "package pkg\n// "+strings.Repeat("x", 70000)+"\n// TODO: after a long line\n")
	writeCloseoutCLIFile(t, root, "pkg/generated.go", "// Code generated by tool. DO NOT EDIT.\npackage pkg\n")
	var out, errOut bytes.Buffer
	if code := cmdAssess(root, []string{"tech-debt"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	for _, want := range []string{"Total Debt Markers: 1", "techdebt.scan", "none unreadable", "generated-header"} {
		if !strings.Contains(out.String()+errOut.String(), want) {
			t.Fatalf("the report lacks %q:\n%s\n%s", want, out.String(), errOut.String())
		}
	}
	out.Reset()
	if code := cmdAssess(root, []string{"tech-debt", "--json"}, &out, &errOut); code != 0 {
		t.Fatal(code)
	}
	var report struct {
		Complete     bool `json:"complete"`
		ScannedFiles int  `json:"scanned_files"`
		Summary      struct {
			TODOs int `json:"todos"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || !report.Complete || report.ScannedFiles != 1 || report.Summary.TODOs != 1 {
		t.Fatalf("report = %+v err=%v\n%s", report, err, out.String())
	}
	if os.Geteuid() != 0 {
		locked := filepath.Join(root, "pkg", "locked.go")
		writeCloseoutCLIFile(t, root, "pkg/locked.go", "package pkg\n// TODO: unreadable\n")
		if err := os.Chmod(locked, 0o000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(locked, 0o644)
		out.Reset()
		errOut.Reset()
		if code := cmdAssess(root, []string{"tech-debt"}, &out, &errOut); code != 0 {
			t.Fatalf("code=%d", code)
		}
		if !strings.Contains(out.String()+errOut.String(), "INCOMPLETE") {
			t.Fatalf("an unreadable file is not visible:\n%s\n%s", out.String(), errOut.String())
		}
	}
}

// R4/V3 and the closeout consumers: the diagnostics are in the JSON and the text.
func TestReviewCheckCarriesTheHistoryDiagnosticInJSONAndText(t *testing.T) {
	root := newGitRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"continuous_closeout":true}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"requirements","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	if err := os.MkdirAll(filepath.Join(root, ".pose", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("absent-target.md", filepath.Join(root, ".pose", "reviews", "rvw-"+time.Now().UTC().Format("20060102T150405Z")+"-aaaa1111.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var out, errOut bytes.Buffer
	code := cmdReviewCheck(root, []string{"spec:alpha"}, &out, &errOut)
	combined := out.String() + errOut.String()
	if code == 0 || !strings.Contains(combined, "no readable review attempt exists for spec:alpha") {
		t.Fatalf("code=%d\n%s", code, combined)
	}
}
