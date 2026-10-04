package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-blocked-semantics-alignment.

func blockedFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range map[string]string{
		".pose/specs/folder-done/spec.md":        "---\nslug: folder-done\nstatus: done\ncreated_at: 2026-08-01\n---\n\n# Spec: folder-done\n",
		".pose/specs/folder-blocked/spec.md":     "---\nslug: folder-blocked\nstatus: blocked\ncreated_at: 2026-08-01\n---\n\n# Spec: folder-blocked\n",
		".pose/specs/2026-09-01-flat-done.md":    "---\nslug: flat-done\nstatus: done\ncreated_at: 2026-09-01\n---\n\n# Spec: flat-done\n",
		".pose/specs/2026-09-02-flat-blocked.md": "---\nslug: flat-blocked\nstatus: blocked\ncreated_at: 2026-09-02\n---\n\n# Spec: flat-blocked\n",
		".pose/specs/2026-09-03-flat-dropped.md": "---\nslug: flat-dropped\nstatus: abandoned\ncreated_at: 2026-09-03\n---\n\n# Spec: flat-dropped\n",
	} {
		mustWrite(t, filepath.Join(root, rel), body)
	}
	return root
}

func TestAdoptionMetricsKeepsV1AndAddsAV2WithoutBlockedAsResolved(t *testing.T) {
	root := blockedFixture(t)
	var out, errB bytes.Buffer
	if code := cmdAdoptionMetrics(root, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit %d: %s", code, errB.String())
	}
	var report adoptionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	// v1 is unchanged: folder-layout specs only, blocked counted as resolved.
	if report.TaskSuccessRatio == nil || *report.TaskSuccessRatio != 0.5 || report.SpecsBlocked != 1 || report.SpecsDone != 1 {
		t.Fatalf("v1 changed meaning: %+v", report)
	}
	// v2 reads every spec and keeps blocked out of resolved outcomes:
	// done=2, abandoned=1 -> 2/3.
	if report.TaskSuccessRatioV2 == nil || *report.TaskSuccessRatioV2 < 0.666 || *report.TaskSuccessRatioV2 > 0.667 {
		t.Fatalf("v2 ratio: %+v", report.TaskSuccessRatioV2)
	}
	if v2 := report.SpecsV2; v2 == nil || v2.SpecsListed != 5 || v2.Blocked != 2 || v2.Done != 2 || v2.Abandoned != 1 {
		t.Fatalf("v2 counts: %+v", report.SpecsV2)
	}
	if report.MetricNotes["v1"] == "" || report.MetricNotes["v2"] == "" {
		t.Fatalf("both series must be labelled: %+v", report.MetricNotes)
	}
	out.Reset()
	cmdAdoptionMetrics(root, nil, &out, &errB)
	if !strings.Contains(out.String(), "task_success_ratio_v2=0.6667") || !strings.Contains(out.String(), "blocked(operational)=2") {
		t.Fatalf("human output lacks v2:\n%s", out.String())
	}
}

func TestLintWarnsOnBlockedWithoutACauseAndNeverFails(t *testing.T) {
	root := blockedFixture(t)
	var out, errB bytes.Buffer
	inDir(t, root, func() { Main([]string{"lint-spec", "flat-blocked"}, &out, &errB) })
	if !strings.Contains(out.String()+errB.String(), "status: blocked records no cause") {
		t.Fatalf("no warning for an unexplained block:\n%s%s", out.String(), errB.String())
	}
}

func TestNoCommandRewritesALegacyBlockedSpec(t *testing.T) {
	root := blockedFixture(t)
	path := filepath.Join(root, ".pose/specs/2026-09-02-flat-blocked.md")
	before, _ := os.ReadFile(path)
	for _, args := range [][]string{{"start", "spec:flat-blocked"}, {"close", "spec:flat-blocked"}, {"state"}} {
		var out, errB bytes.Buffer
		inDir(t, root, func() { Main(args, &out, &errB) })
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatalf("a command rewrote a legacy blocked spec:\n%s", after)
	}
}

func TestStateListsBlockedSpecsWithTheirCause(t *testing.T) {
	root := blockedFixture(t)
	content := provideSpecsRoadmaps(posemodel.Store{Root: root})
	if !strings.Contains(content, "- blocked (operational, non-terminal): spec:flat-blocked (cause: unknown), spec:folder-blocked (cause: unknown)") {
		t.Fatalf("state does not list blocked specs as operational:\n%s", content)
	}
}
