package pose

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const startSpecBody = "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: Record the baseline.\n- R2: Move to in-progress once.\n\n## 5. Decisions\n\n### Decision D1\n- Basis: R1\n"

func startFixture(t *testing.T, adopted bool, spec string) (Store, string) {
	t.Helper()
	root := t.TempDir()
	policy := `{"schema_version":1,"enabled":false}`
	if adopted {
		policy = `{"schema_version":1,"enabled":false,"atomic_start_version":1}`
	}
	for path, body := range map[string]string{
		".pose/policy/review.json":       policy,
		".pose/specs/2026-09-27-work.md": spec,
		".pose/specs/2026-09-27-dep.md":  "---\nslug: dep\nstatus: in-progress\ncreated_at: 2026-09-27\n---\n\n# Spec: dep\n\n## 2. Requirements\n\n- R1: Dep.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "POSE start"}, {"config", "user.email", "start@example.invalid"}, {"add", "--all"}, {"commit", "-q", "-m", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return Store{Root: root}, filepath.Join(root, ".pose/specs/2026-09-27-work.md")
}

func TestABMAtomicStartPreviewIsReadOnlyAndDigestBound(t *testing.T) {
	store, specPath := startFixture(t, false, startSpecBody)
	before, _ := os.ReadFile(specPath)
	first, err := store.PreviewStart("work")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := store.PreviewStart("work")
	if first.Digest == "" || first.Digest != second.Digest || !first.Ready || first.FromStatus != "draft" || len(first.Baseline.Nodes) != 3 || first.Revision == "" {
		t.Fatalf("preview is not a stable, ready, digest-bound plan: %+v", first)
	}
	after, _ := os.ReadFile(specPath)
	if string(after) != string(before) {
		t.Fatal("preview wrote the spec")
	}
	if _, err := os.Stat(filepath.Join(store.Root, ".pose", "starts")); !os.IsNotExist(err) {
		t.Fatal("preview created start state")
	}
	if _, err := store.ApplyStart(first, first.Digest, nil); err == nil || !strings.Contains(err.Error(), "atomic-start-capability-not-adopted") {
		t.Fatalf("apply ran without the capability: %v", err)
	}
}

func TestABMAtomicStartApplyRecordsBaselineAndTransitionsOnce(t *testing.T) {
	store, specPath := startFixture(t, true, startSpecBody)
	plan, _ := store.PreviewStart("work")
	record, err := store.ApplyStart(plan, plan.Digest, nil)
	if err != nil || record.Phase != "started" || record.Baseline.Digest != plan.Baseline.Digest {
		t.Fatalf("apply = %+v err=%v", record, err)
	}
	spec, _ := store.GetSpec("work")
	if spec.Status != "in-progress" {
		t.Fatalf("spec status = %s", spec.Status)
	}
	again, err := store.ApplyStart(plan, plan.Digest, nil)
	if err != nil || again.Phase != "started" || again.RecordedAt != record.RecordedAt {
		t.Fatalf("repeating the same start was not idempotent: %+v err=%v", again, err)
	}
	status, _ := store.GetStartStatus("work")
	for _, node := range status.Nodes {
		if node.Origin != "recorded-before-managed-execution" {
			t.Fatalf("unchanged node not classified as recorded before execution: %+v", status.Nodes)
		}
	}
	raw, _ := os.ReadFile(specPath)
	edited := strings.Replace(strings.Replace(string(raw), "- R2: Move to in-progress once.", "- R2: Move to in-progress exactly once.\n- R3: Report origins.", 1), "- R1:", "- R1:", 1)
	if err := os.WriteFile(specPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	status, _ = store.GetStartStatus("work")
	origins := map[string]string{}
	for _, node := range status.Nodes {
		origins[node.ID] = node.Origin
	}
	if origins["R1"] != "recorded-before-managed-execution" || origins["R2"] != "introduced-during-execution" || origins["R3"] != "introduced-during-execution" {
		t.Fatalf("origins after edits = %v", origins)
	}
	if _, err := store.ApplyStart(plan, plan.Digest, nil); err != nil {
		t.Fatalf("a started spec's own plan must stay a no-op: %v", err)
	}
	fresh, _ := store.PreviewStart("work")
	if _, err := store.ApplyStart(fresh, fresh.Digest, nil); err == nil || !strings.Contains(err.Error(), "spec-already-started-with-another-plan") {
		t.Fatalf("a second, different start was accepted: %v", err)
	}
}

func TestABMAtomicStartResumesAfterInterruptionAndCancels(t *testing.T) {
	store, _ := startFixture(t, true, startSpecBody)
	plan, _ := store.PreviewStart("work")
	stop := errors.New("interrupted")
	if _, err := store.ApplyStart(plan, plan.Digest, func(phase string) error {
		if phase == "baseline-recorded" {
			return stop
		}
		return nil
	}); !errors.Is(err, stop) {
		t.Fatalf("interruption not reported: %v", err)
	}
	spec, _ := store.GetSpec("work")
	status, _ := store.GetStartStatus("work")
	if spec.Status != "draft" || status.Phase != "baseline-recorded" || !strings.Contains(strings.Join(status.Reconciliation, " "), "interrupted") {
		t.Fatalf("interrupted start left an unclear state: status=%s %+v", spec.Status, status)
	}
	if record, err := store.ApplyStart(plan, plan.Digest, nil); err != nil || record.Phase != "started" {
		t.Fatalf("retry did not resume: %+v err=%v", record, err)
	}
	if err := store.CancelStart("work"); err == nil {
		t.Fatal("a completed start was cancelled")
	}

	store2, _ := startFixture(t, true, startSpecBody)
	plan2, _ := store2.PreviewStart("work")
	_, _ = store2.ApplyStart(plan2, plan2.Digest, func(phase string) error { return stop })
	if err := store2.CancelStart("work"); err != nil {
		t.Fatalf("cancel of an untransitioned start failed: %v", err)
	}
	if status, _ := store2.GetStartStatus("work"); status.Phase != "not-started" {
		t.Fatalf("cancel left start state: %+v", status)
	}
}

func TestABMAtomicStartRefusesStaleAndUnreadyPlans(t *testing.T) {
	store, specPath := startFixture(t, true, startSpecBody)
	plan, _ := store.PreviewStart("work")
	raw, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, []byte(strings.Replace(string(raw), "Record the baseline.", "Record the baseline first.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyStart(plan, plan.Digest, nil); err == nil || !strings.Contains(err.Error(), "stale-plan") {
		t.Fatalf("a plan made before an edit was applied: %v", err)
	}
	if _, err := store.ApplyStart(plan, "0000", nil); err == nil || !strings.Contains(err.Error(), "plan-digest-mismatch") {
		t.Fatalf("a wrong digest was accepted: %v", err)
	}
	blocked, _ := startFixture(t, true, strings.Replace(startSpecBody, "created_at: 2026-09-27\n", "created_at: 2026-09-27\ndepends_on: dep\n", 1))
	waiting, _ := blocked.PreviewStart("work")
	if waiting.Ready {
		t.Fatalf("a spec waiting on an open dependency previewed ready: %+v", waiting)
	}
	if _, err := blocked.ApplyStart(waiting, waiting.Digest, nil); err == nil || !strings.Contains(err.Error(), "spec-not-ready") {
		t.Fatalf("an unready spec was started: %v", err)
	}
}

func TestABMAtomicStartConcurrentAppliesRecordOneBaseline(t *testing.T) {
	store, _ := startFixture(t, true, startSpecBody)
	plan, _ := store.PreviewStart("work")
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = store.ApplyStart(plan, plan.Digest, nil)
		}(i)
	}
	wg.Wait()
	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		} else if !strings.Contains(err.Error(), "start-in-progress") {
			t.Fatalf("concurrent apply failed for another reason: %v", err)
		}
	}
	raw, err := os.ReadFile(startRecordPath(store.Root, "work"))
	if err != nil || succeeded == 0 {
		t.Fatalf("no start recorded: %v", err)
	}
	var record StartRecord
	if json.Unmarshal(raw, &record) != nil || record.Phase != "started" || record.PlanDigest != plan.Digest {
		t.Fatalf("record after concurrent applies = %s", raw)
	}
}

func TestABMAtomicStartReconciliationNeverFabricatesHistory(t *testing.T) {
	legacy, _ := startFixture(t, true, strings.Replace(startSpecBody, "status: draft", "status: in-progress", 1))
	status, _ := legacy.GetStartStatus("work")
	if status.Phase != "not-started" || !strings.Contains(strings.Join(status.Reconciliation, " "), "without a recorded start") {
		t.Fatalf("an in-progress spec without a start was not flagged: %+v", status)
	}
	for _, node := range status.Nodes {
		if node.Origin != "legacy-unbaselined" {
			t.Fatalf("a legacy node was given an origin: %+v", status.Nodes)
		}
	}
	unadopted, _ := startFixture(t, false, strings.Replace(startSpecBody, "status: draft", "status: in-progress", 1))
	if status, _ := unadopted.GetStartStatus("work"); len(status.Reconciliation) != 0 {
		t.Fatalf("reconciliation was demanded without the capability: %+v", status)
	}

	store, specPath := startFixture(t, true, startSpecBody)
	plan, _ := store.PreviewStart("work")
	if _, err := store.ApplyStart(plan, plan.Digest, nil); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, []byte(strings.Replace(string(raw), "status: in-progress", "status: draft", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	recordPath := startRecordPath(store.Root, "work")
	recordRaw, _ := os.ReadFile(recordPath)
	var record StartRecord
	_ = json.Unmarshal(recordRaw, &record)
	record.Baseline.Nodes[0].Hash = "forged"
	forged, _ := json.Marshal(record)
	if err := os.WriteFile(recordPath, forged, 0o644); err != nil {
		t.Fatal(err)
	}
	status, _ = store.GetStartStatus("work")
	joined := strings.Join(status.Reconciliation, " ")
	if !strings.Contains(joined, "moved back to draft") || !strings.Contains(joined, "baseline was edited") {
		t.Fatalf("manual edits were not flagged: %+v", status)
	}
}
