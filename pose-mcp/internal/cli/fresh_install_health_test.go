package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold/distpolicy"
)

// Spec pose-fresh-install-doctor-is-clean.

func freshInstall(t *testing.T) string {
	t.Helper()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--project-id", "proj.fresh"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	return repo
}

func freshAttention(t *testing.T, repo string) map[string]any {
	t.Helper()
	var doc struct {
		Attention map[string]any `json:"attention"`
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"state", "--attention", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("state --attention exit=%d err=%s", code, errB.String())
		}
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
	})
	return doc.Attention
}

func TestFreshInstallHealthDoctorReportsNoWarning(t *testing.T) {
	repo := freshInstall(t)
	findings := runDoctorJSON(t, repo)
	for _, f := range findings {
		if f.Level == "warn" || f.Level == "error" {
			t.Errorf("a fresh install reports %s %s: %s", f.Level, f.Check, f.Message)
		}
	}
	for _, check := range []string{"hooks.pre-commit", "actions.roles"} {
		f, ok := findDoctorFinding(findings, check)
		if !ok || f.Level != "next" || f.Hint == "" {
			t.Errorf("%s is not reported as a next step with its command: found=%v %+v", check, ok, f)
		}
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("doctor exit=%d", code)
		}
		var report struct {
			Version int `json:"doctor_schema_version"`
			Next    int `json:"next_steps"`
		}
		_ = json.Unmarshal(out.Bytes(), &report)
		if report.Version != 2 || report.Next < 2 {
			t.Fatalf("doctor JSON does not carry the next-step count: %+v", report)
		}
		out.Reset()
		Main([]string{"doctor"}, &out, &errB)
		if !strings.Contains(out.String(), "next step(s)") || !strings.Contains(out.String(), "[→] hooks.pre-commit") {
			t.Fatalf("the text report does not list next steps apart:\n%s", out.String())
		}
	})
}

func TestFreshInstallHealthAttentionCoverageIsComplete(t *testing.T) {
	repo := freshInstall(t)
	// Before the first commit there is no revision to bind the answer to, and
	// Attention says so; the install committed is the instance's first state.
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", "install POSE", "--no-verify"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	attention := freshAttention(t, repo)
	if attention["incomplete"] != false {
		t.Fatalf("a fresh install reads incomplete coverage: %v", attention)
	}
	// Every source is projected now (spec pose-attention-projects-every-source),
	// so none is reported as unused or unprojected.
	if notUsed, _ := attention["not_used"].([]any); len(notUsed) != 0 {
		t.Fatalf("a projected source is still reported as unused: %v", notUsed)
	}

	// A project that uses a source keeps complete coverage, because the source
	// is read; an open docs review mark becomes an obligation.
	if err := os.WriteFile(filepath.Join(repo, ".pose", "docs.json"), []byte(`{"schema_version":1,"docs":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".pose", "docs-review.jsonl"), []byte(`{"at":"2026-10-09T00:00:00Z","doc":"docs/guide.md","kind":"marked","trigger":"spec:demo"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	attention = freshAttention(t, repo)
	if attention["incomplete"] != false {
		t.Fatalf("a used, projected source made coverage incomplete: %v", attention)
	}
	if gates, _ := attention["gates"].([]any); len(gates) != 1 {
		raw, _ := json.Marshal(attention)
		t.Fatalf("the open docs review mark is not in Attention: %s", raw)
	}
}

func TestFreshInstallHealthRoadmapProfilesNeedARoadmap(t *testing.T) {
	repo := freshInstall(t)
	if f, ok := findDoctorFinding(runDoctorJSON(t, repo), "validate.class-producers"); !ok || f.Level != "ok" {
		t.Fatalf("a project without a roadmap is held to milestone criteria: %+v", f)
	}
	if err := os.WriteFile(filepath.Join(repo, ".pose", "roadmaps", "r.md"), []byte("# r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, _ := findDoctorFinding(runDoctorJSON(t, repo), "validate.class-producers")
	if f.Level != "warn" || !strings.Contains(f.Message, "milestone-integration") {
		t.Fatalf("a project with a roadmap is not held to milestone criteria: %+v", f)
	}
}

func TestShippedStackChecksDeclareTheirEvidenceClass(t *testing.T) {
	raw, ok := distpolicy.NeutralIndexTemplates()[".pose/indexes/validation-matrix.json"]
	if !ok {
		t.Fatal("the shipped validation matrix template is missing")
	}
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		t.Fatal(err)
	}
	for name, stack := range matrix.Stacks {
		for _, check := range stack.Checks {
			if (check.Severity == "" || check.Severity == "required") && check.EvidenceClass == "" {
				t.Errorf("stack:%s/%s is required and declares no evidence class", name, check.Name)
			}
		}
	}
}
