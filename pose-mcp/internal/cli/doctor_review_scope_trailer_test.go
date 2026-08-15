package cli

// Review-scope recorded change set coverage (spec
// pose-review-scope-trailer-check, github.com/oseiaspereira88/pose issue #17
// comments 2-3): `pose review bundle <scope> --seal` reads its subject from
// the indexed graph, which is built solely from change sets persisted via
// `pose report --change-from/--to` (loadRecordedChangeSets). A commit
// carrying the POSE-Spec: <slug> trailer alone does NOT persist one — that
// trailer only feeds a separate, ephemeral live-discovery fallback used by
// `pose artifact-check`/`artifact-backfill --from-git` when called without
// --from/--to. `pose doctor` should warn before someone hits "no immutable
// attributed change set exists" at seal time.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func doctorTrailerGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	return artifactGit(t, root, args...)
}

func doctorTrailerFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	doctorTrailerGit(t, root, "init", "-q")
	doctorTrailerGit(t, root, "config", "user.email", "pose@example.invalid")
	doctorTrailerGit(t, root, "config", "user.name", "POSE Tests")
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeDoctorTrailerSpec(t *testing.T, root, slug, status string) {
	t.Helper()
	writeArtifactTestFile(t, root, filepath.Join(".pose", "specs", slug, "spec.md"),
		"---\nslug: "+slug+"\nstatus: "+status+"\ncreated_at: 2026-08-15\ndelivers: capability:"+slug+"\n---\n\n# Spec: "+slug+"\n\nwork\n")
}

func runDoctorJSON(t *testing.T, root string) []doctorFinding {
	t.Helper()
	var findings []doctorFinding
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("doctor exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		var report struct {
			Findings []doctorFinding `json:"findings"`
		}
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
		}
		findings = report.Findings
	})
	return findings
}

func findDoctorFinding(findings []doctorFinding, check string) (doctorFinding, bool) {
	for _, f := range findings {
		if f.Check == check {
			return f, true
		}
	}
	return doctorFinding{}, false
}

func TestDoctorWarnsWhenSpecHasNoRecordedChangeSet(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "alpha", "in-progress")
	doctorTrailerGit(t, root, "add", "--", ".")
	doctorTrailerGit(t, root, "commit", "-q", "-m", "implement alpha")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.scope-change-set")
	if !ok {
		t.Fatal("expected a review.scope-change-set finding")
	}
	if f.Level != "warn" || f.RemediationClass != remediationDetectable {
		t.Errorf("review.scope-change-set: level=%q class=%q, want warn/detectable", f.Level, f.RemediationClass)
	}
	if !strings.Contains(f.Message, "alpha") {
		t.Errorf("review.scope-change-set message does not name the untraceable spec: %q", f.Message)
	}
	if !strings.Contains(f.Hint, "issue #17") || !strings.Contains(f.Hint, "pose report") {
		t.Errorf("review.scope-change-set hint missing remediation pointer: %q", f.Hint)
	}
}

// TestDoctorStillWarnsWhenOnlyATrailerCommitExists pins the correction made
// while implementing this check (Decision 2 in the spec): a POSE-Spec:
// <slug> trailer commit, on its own, does NOT persist a change set that
// review bundle sealing can find — only `pose report --change-from/--to`
// does. A doctor check that went silent here would be actively misleading.
func TestDoctorStillWarnsWhenOnlyATrailerCommitExists(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "alpha", "in-progress")
	doctorTrailerGit(t, root, "add", "--", ".")
	doctorTrailerGit(t, root, "commit", "-q", "-m", "implement alpha", "-m", "POSE-Spec: alpha")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.scope-change-set")
	if !ok {
		t.Fatal("expected a review.scope-change-set finding")
	}
	if f.Level != "warn" {
		t.Errorf("review.scope-change-set level=%q, want warn — a trailer commit alone does not record a change set", f.Level)
	}
}

func TestDoctorSilentWhenChangeSetIsRecorded(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "alpha", "in-progress")
	doctorTrailerGit(t, root, "add", "--", ".")
	doctorTrailerGit(t, root, "commit", "-q", "-m", "implement alpha")
	head := doctorTrailerGit(t, root, "rev-parse", "HEAD")

	writeArtifactTestFile(t, root, filepath.Join(".pose", "reports", "history", "standard.jsonl"),
		`{"generated_at":"2026-08-15T00:00:00Z","sequence":1,"task":"closeout alpha","spec":"alpha","change_set":{"id":"cs-test","spec":"alpha","selector":"range:`+head+`^..`+head+`","base":"`+head+`^","head":"`+head+`","resolved_base":"`+head+`","resolved_head":"`+head+`","paths":[]}}`+"\n")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.scope-change-set")
	if !ok {
		t.Fatal("expected a review.scope-change-set finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("review.scope-change-set level=%q, want ok when a change set was already recorded", f.Level)
	}
}

func TestDoctorIgnoresDraftSpecsAndSpecsWithoutDelivers(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "draft-spec", "draft")
	writeArtifactTestFile(t, root, filepath.Join(".pose", "specs", "no-delivers", "spec.md"),
		"---\nslug: no-delivers\nstatus: in-progress\ncreated_at: 2026-08-15\n---\n\n# Spec: no-delivers\n\nwork\n")
	doctorTrailerGit(t, root, "add", "--", ".")
	doctorTrailerGit(t, root, "commit", "-q", "-m", "specs without delivers or in-progress/done status")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.scope-change-set")
	if !ok {
		t.Fatal("expected a review.scope-change-set finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("review.scope-change-set level=%q, want ok — draft status and missing delivers: must not count", f.Level)
	}
}
