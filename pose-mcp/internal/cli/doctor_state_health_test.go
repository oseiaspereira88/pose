package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func doctorStateFindings(t *testing.T, root string) []doctorFinding {
	t.Helper()
	var report struct {
		Errors   int             `json:"errors"`
		Findings []doctorFinding `json:"findings"`
	}
	inDir(t, root, func() {
		var stdout, stderr bytes.Buffer
		code := Main([]string{"doctor", "--json"}, &stdout, &stderr)
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("doctor JSON: %v, stderr=%s", err, stderr.String())
		}
		if (code == 0) != (report.Errors == 0) {
			t.Fatalf("doctor exit=%d errors=%d", code, report.Errors)
		}
	})
	return report.Findings
}

func TestDoctorStateHealth(t *testing.T) {
	for _, tc := range []struct {
		name, body, check, level string
	}{
		{"valid", "doc:docs/context.md", "state.pointers", "ok"},
		{"broken-doc", "doc:docs/missing.md", "state.pointers", "error"},
		{"wrong-origin", "spec:docs:docs/context.md", "state.pointers", "error"},
		{"ordinary-spec", "spec:alpha", "state.pointers", "ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := stateTestRoot(t)
			writeStateTestFile(t, root, "docs/context.md", "# Context\n")
			body := "- " + tc.body
			content := "---\nschema_version: 1\ngenerated_at: " + time.Now().UTC().Format(time.RFC3339) +
				"\nbaseline_commit: " + strings.TrimSpace(runStateGit(t, root, "rev-parse", "HEAD")) +
				"\n---\n\n## Follow-ups\n<!-- state:derived hash:" + pose.ContentHash12(body) + " -->\n" + body + "\n"
			writeStateTestFile(t, root, ".pose/state/project-state.md", content)
			f, ok := findDoctorFinding(doctorStateFindings(t, root), tc.check)
			if !ok || f.Level != tc.level {
				t.Fatalf("finding = %+v, expected %s", f, tc.level)
			}
			if tc.level == "error" && (f.RemediationClass != remediationDetectable || !strings.Contains(f.Hint, "pose state")) {
				t.Fatalf("error must provide explicit, non-automatic remediation: %+v", f)
			}
			after, err := os.ReadFile((pose.Store{Root: root}).StatePath())
			if err != nil || string(after) != content {
				t.Fatalf("doctor mutated state or made it unreadable: %v", err)
			}
			code, _, _ := runState(t, root)
			if (code == 0) != (tc.level == "ok") {
				t.Fatalf("state/doctor disagree: state exit %d", code)
			}
		})
	}
}

func TestDoctorStateAbsentAndMalformed(t *testing.T) {
	root := stateTestRoot(t)
	f, ok := findDoctorFinding(doctorStateFindings(t, root), "state.artifact")
	if !ok || f.Level != "ok" || !strings.Contains(f.Message, "not initialized") {
		t.Fatalf("absent optional state: %+v", f)
	}
	writeStateTestFile(t, root, ".pose/state/project-state.md", "malformed\n")
	f, ok = findDoctorFinding(doctorStateFindings(t, root), "state.artifact")
	if !ok || f.Level != "error" || f.RemediationClass != remediationDetectable {
		t.Fatalf("malformed state: %+v", f)
	}
}

func TestDoctorStateIntegrityAndFreshness(t *testing.T) {
	root := stateTestRoot(t)
	writeStateTestFile(t, root, ".pose/state/project-state.md", "---\nschema_version: 1\ngenerated_at: 2000-01-01T00:00:00Z\nbaseline_commit: abcdef0\nrefresh_pending: spec-closed\n---\n\n## Follow-ups\n<!-- state:derived hash:000000000000 -->\nchanged\n")
	findings := doctorStateFindings(t, root)
	for check, level := range map[string]string{"state.integrity": "error", "state.freshness": "warn", "state.pointers": "ok"} {
		f, ok := findDoctorFinding(findings, check)
		if !ok || f.Level != level {
			t.Errorf("%s: %+v, want %s", check, f, level)
		}
	}
}
