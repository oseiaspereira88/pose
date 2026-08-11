package usage

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func usageFixture(t *testing.T) (string, time.Time) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("POSE_USAGE_DIR", dir)
	t.Setenv("POSE_USAGE_DISABLED", "")
	return dir, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
}

func TestUsageFindingLifecycleAndFilters(t *testing.T) {
	_, now := usageFixture(t)
	record := func(offset time.Duration, findings []Finding, complete bool) {
		t.Helper()
		if err := Record("/ignored", Observation{
			At: now.Add(offset), Tool: "validate", Surface: "cli", DurationMS: 10,
			ExecutionOutcome: "completed", SemanticOutcome: map[bool]string{true: "fail", false: "pass"}[len(findings) > 0],
			Findings: findings, FindingSetComplete: complete, Scope: "module=pose-mcp", Version: "test",
		}); err != nil {
			t.Fatal(err)
		}
	}
	record(-4*time.Hour, []Finding{{ID: "check-a", Severity: "error"}}, true)
	record(-3*time.Hour, []Finding{{ID: "check-a", Severity: "error"}, {ID: "check-b", Severity: "warning"}}, true)
	record(-2*time.Hour, nil, true)
	record(-time.Hour, []Finding{{ID: "check-a", Severity: "error"}}, true)

	report, err := Aggregate("/ignored", Query{SinceDays: 0, Tool: "validate", Surface: "cli", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Available || report.RecordsMatched != 4 || len(report.Rows) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	row := report.Rows[0]
	if row.Calls != 4 || row.FindingsObserved != 4 || row.UniqueFindings != 2 || row.NewFindings != 2 || row.ResolvedFindings != 2 || row.ReopenedFindings != 1 {
		t.Fatalf("unexpected lifecycle: %+v", row)
	}
	if row.FindingsBySeverity["error"] != 3 || row.FindingsBySeverity["warning"] != 1 {
		t.Fatalf("unexpected severity counts: %+v", row.FindingsBySeverity)
	}
	if row.Pass != 1 || row.Fail != 3 || row.P50DurationMS != 10 || row.P95DurationMS != 10 {
		t.Fatalf("unexpected outcomes/duration: %+v", row)
	}

	none, err := Aggregate("/ignored", Query{Tool: "check", Surface: "cli", Now: now})
	if err != nil || none.Available || len(none.Rows) != 0 {
		t.Fatalf("filtered empty report = %+v err=%v", none, err)
	}
}

func TestUsageMalformedLineIsVisible(t *testing.T) {
	dir, now := usageFixture(t)
	if err := os.WriteFile(filepath.Join(dir, "2026-08.jsonl"), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Record("/ignored", Observation{At: now, Tool: "check", Surface: "cli", ExecutionOutcome: "completed", SemanticOutcome: "pass", Scope: "project"}); err != nil {
		t.Fatal(err)
	}
	report, err := Aggregate("/ignored", Query{Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if report.InvalidRecords != 1 || report.RecordsScanned != 1 || report.RecordsMatched != 1 {
		t.Fatalf("malformed accounting = %+v", report)
	}
}

func TestUsageConcurrentAppend(t *testing.T) {
	_, now := usageFixture(t)
	const writers = 64
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Record("/ignored", Observation{At: now, Tool: "stats", Surface: "cli", ExecutionOutcome: "completed", SemanticOutcome: "unknown", Scope: "project"})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	report, err := Aggregate("/ignored", Query{Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if report.InvalidRecords != 0 || report.RecordsMatched != writers {
		t.Fatalf("concurrent report = %+v", report)
	}
}

func TestUsageSchemaPrivacy(t *testing.T) {
	forbidden := []string{"path", "root", "repo", "project", "argument", "output", "principal", "user", "actor", "run_id", "task_id", "content"}
	typ := reflect.TypeOf(Event{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		for _, word := range forbidden {
			if strings.Contains(strings.ToLower(name), word) {
				t.Fatalf("persisted field %q contains forbidden identity/content shape %q", name, word)
			}
		}
	}
	dir, now := usageFixture(t)
	secretScope := "/home/acme/private-repo --token super-secret"
	secretFinding := "services/private/payment.go:42 customer@example.com"
	if err := Record("/ignored", Observation{At: now, Tool: "validate", Surface: "cli", ExecutionOutcome: "completed", SemanticOutcome: "fail", Scope: secretScope, Findings: []Finding{{ID: secretFinding, Severity: "error"}}, FindingSetComplete: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "2026-08.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-repo") || strings.Contains(string(raw), "payment.go") || strings.Contains(string(raw), "super-secret") || strings.Contains(string(raw), "customer@example.com") {
		t.Fatalf("raw identity leaked: %s", raw)
	}
	var event Event
	if err := json.Unmarshal(raw, &event); err != nil || len(event.ScopeHash) != 32 || len(event.FindingFingerprints) != 1 || len(event.FindingFingerprints[0]) != 32 {
		t.Fatalf("unexpected encoded event: %+v err=%v", event, err)
	}
}

func TestUsageUnavailableAndValidation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	t.Setenv("POSE_USAGE_DIR", dir)
	report, err := Aggregate("/ignored", Query{SinceDays: 30})
	if err != nil || report.Available || report.Reason == "" {
		t.Fatalf("unavailable report = %+v err=%v", report, err)
	}
	if _, err := Aggregate("/ignored", Query{SinceDays: -1}); err == nil {
		t.Fatal("negative window must fail")
	}
	if err := Record("/ignored", Observation{Tool: "bad tool", Surface: "cli", ExecutionOutcome: "completed", SemanticOutcome: "pass"}); err == nil {
		t.Fatal("unsafe tool name must fail")
	}
	t.Setenv("POSE_USAGE_DIR", filepath.Join(t.TempDir(), "project", "usage"))
	project := filepath.Dir(os.Getenv("POSE_USAGE_DIR"))
	if _, err := Aggregate(project, Query{}); err == nil {
		t.Fatal("usage override inside worktree must fail")
	}
}

func TestUsageStorageResolutionStaysOutsideWorktree(t *testing.T) {
	t.Setenv("POSE_USAGE_DIR", "")
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	dir, err := storageDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(repo, ".git", "pose", "usage")
	if dir != want {
		t.Fatalf("git storage = %q, want %q", dir, want)
	}
	if strings.HasPrefix(dir, filepath.Join(repo, ".pose")+string(filepath.Separator)) {
		t.Fatalf("usage journal entered governed worktree: %s", dir)
	}

	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	nonGit := t.TempDir()
	dir, err = storageDir(nonGit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dir, filepath.Join(cache, "pose", "usage")+string(filepath.Separator)) {
		t.Fatalf("fallback storage = %q, want under %q", dir, cache)
	}
}
