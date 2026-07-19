package cli

// DORA and adoption-value metrics (spec pose-dora-adoption-metrics):
// explicit event ingestion with quality metadata (R1), the five DORA
// metrics with valid-only denominators and an explicit "unavailable"
// state (R2), adoption views derived from data POSE already owns (R3),
// retention/deletion housekeeping and a structural check that no event
// or report ever carries a per-identity field (Constraint: team/application
// outcomes only, never individual scores).

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecordDeploymentValidation(t *testing.T) {
	repo := newGitRepo(t)
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"missing-required", []string{"--status", "success", "--source", "ci"}, 2},
		{"invalid-status", []string{"--application", "a", "--environment", "prod", "--status", "maybe", "--source", "ci"}, 2},
		{"invalid-source", []string{"--application", "a", "--environment", "prod", "--status", "success", "--source", "vibes"}, 2},
		{"invalid-deployed-at", []string{"--application", "a", "--environment", "prod", "--status", "success", "--source", "ci", "--deployed-at", "not-a-date"}, 2},
		{"invalid-lead-time", []string{"--application", "a", "--environment", "prod", "--status", "success", "--source", "ci", "--lead-time-seconds", "-5"}, 2},
		{"valid", []string{"--application", "a", "--environment", "prod", "--status", "success", "--source", "ci", "--deployed-at", "2026-06-01T10:00:00Z", "--lead-time-seconds", "3600"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errB bytes.Buffer
			if code := cmdRecordDeployment(repo, c.args, &out, &errB); code != c.want {
				t.Fatalf("exit=%d want=%d out=%s err=%s", code, c.want, out.String(), errB.String())
			}
		})
	}
}

func TestRecordIncidentValidation(t *testing.T) {
	repo := newGitRepo(t)
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"missing-required", []string{"--severity", "major", "--source", "ci"}, 2},
		{"invalid-severity", []string{"--application", "a", "--started-at", "2026-06-01T10:00:00Z", "--severity", "yikes", "--source", "ci"}, 2},
		{"invalid-source", []string{"--application", "a", "--started-at", "2026-06-01T10:00:00Z", "--severity", "major", "--source", "vibes"}, 2},
		{"resolved-before-started", []string{"--application", "a", "--started-at", "2026-06-01T10:00:00Z", "--resolved-at", "2026-06-01T09:00:00Z", "--severity", "major", "--source", "ci"}, 2},
		{"valid", []string{"--application", "a", "--started-at", "2026-06-01T10:00:00Z", "--resolved-at", "2026-06-01T11:00:00Z", "--severity", "major", "--source", "ci"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errB bytes.Buffer
			if code := cmdRecordIncident(repo, c.args, &out, &errB); code != c.want {
				t.Fatalf("exit=%d want=%d out=%s err=%s", code, c.want, out.String(), errB.String())
			}
		})
	}
}

func TestEventsAreAppendOnlyMonthlyJSONL(t *testing.T) {
	repo := newGitRepo(t)
	record := func(deployedAt string) {
		var out, errB bytes.Buffer
		if code := cmdRecordDeployment(repo, []string{
			"--application", "a", "--environment", "prod", "--status", "success", "--source", "ci", "--deployed-at", deployedAt,
		}, &out, &errB); code != 0 {
			t.Fatalf("record exit=%d err=%s", code, errB.String())
		}
	}
	record("2026-06-01T10:00:00Z")
	record("2026-06-15T10:00:00Z") // same month, second line
	record("2026-07-01T10:00:00Z") // different month, new file

	juneFile := filepath.Join(repo, ".pose", "events", "deployments", "2026-06.jsonl")
	julyFile := filepath.Join(repo, ".pose", "events", "deployments", "2026-07.jsonl")
	juneContent, err := os.ReadFile(juneFile)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(juneContent)), "\n") + 1; lines != 2 {
		t.Errorf("expected 2 appended lines in June file, got %d:\n%s", lines, juneContent)
	}
	if _, err := os.Stat(julyFile); err != nil {
		t.Errorf("expected a separate July file: %v", err)
	}
}

func TestDORAMetricsUnavailableWithNoData(t *testing.T) {
	report := computeDORA(nil, nil, "", 30)
	if len(report.Metrics) != 5 {
		t.Fatalf("expected 5 DORA metrics, got %d", len(report.Metrics))
	}
	for _, m := range report.Metrics {
		if m.Available {
			t.Errorf("%s: expected unavailable with zero events, got value=%v", m.Name, *m.Value)
		}
		if m.Reason == "" {
			t.Errorf("%s: unavailable metric must carry a reason", m.Name)
		}
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestDORAMetricsComputesFromSyntheticHistory(t *testing.T) {
	lead1, lead2 := 3600.0, 7200.0
	deployments := []deploymentEvent{
		{Application: "checkout", DeployedAt: "2026-06-02T10:00:00Z", Status: "success", LeadTimeSeconds: &lead1, Source: "ci"},
		{Application: "checkout", DeployedAt: "2026-06-05T10:00:00Z", Status: "success", LeadTimeSeconds: &lead2, Source: "ci"},
		{Application: "checkout", DeployedAt: "2026-06-10T10:00:00Z", Status: "failure", Source: "ci"},
		{Application: "checkout", DeployedAt: "2026-06-20T10:00:00Z", Status: "success", Source: "manual"}, // no lead time
	}
	incidents := []incidentEvent{
		{Application: "checkout", StartedAt: "2026-06-10T10:05:00Z", ResolvedAt: "2026-06-10T11:05:00Z", Severity: "major", CausedByDeployment: true, Source: "manual"},
		{Application: "checkout", StartedAt: "2026-06-25T00:00:00Z", Severity: "minor", Source: "manual"}, // unresolved, minor: excluded from reliability
	}

	// Fixed "now" via a 30-day window ending 2026-07-01 by construction:
	// computeDORA uses time.Now(), so anchor the window generously (90
	// days) to guarantee every fixture event falls inside it regardless
	// of when the test runs, and assert ratios/medians rather than
	// day-bucketed reliability (which does depend on "now").
	report := computeDORA(deployments, incidents, "checkout", 3650) // ~10 years: everything in-window

	byName := map[string]doraMetric{}
	for _, m := range report.Metrics {
		byName[m.Name] = m
	}

	freq := byName["deployment_frequency"]
	if !freq.Available || freq.SampleSize != 4 {
		t.Fatalf("deployment_frequency = %+v, want available with sample_size=4", freq)
	}
	wantFreq := 3.0 / 3650.0 // 3 successful / window days
	if *freq.Value < wantFreq-1e-9 || *freq.Value > wantFreq+1e-9 {
		t.Errorf("deployment_frequency = %v, want %v", *freq.Value, wantFreq)
	}

	lt := byName["lead_time_for_changes"]
	if !lt.Available || lt.SampleSize != 2 {
		t.Fatalf("lead_time_for_changes = %+v, want available with sample_size=2", lt)
	}
	if wantMedian := (lead1 + lead2) / 2; *lt.Value != wantMedian {
		t.Errorf("lead_time_for_changes = %v, want %v", *lt.Value, wantMedian)
	}

	cfr := byName["change_failure_rate"]
	if !cfr.Available || cfr.SampleSize != 4 {
		t.Fatalf("change_failure_rate = %+v, want available with sample_size=4", cfr)
	}
	if *cfr.Value != 0.25 {
		t.Errorf("change_failure_rate = %v, want 0.25 (1 of 4)", *cfr.Value)
	}

	mttr := byName["failed_deployment_recovery_time"]
	if !mttr.Available || mttr.SampleSize != 1 {
		t.Fatalf("failed_deployment_recovery_time = %+v, want available with sample_size=1", mttr)
	}
	if *mttr.Value != 3600 {
		t.Errorf("failed_deployment_recovery_time = %v, want 3600 (1h resolved incident)", *mttr.Value)
	}

	rel := byName["reliability"]
	if !rel.Available {
		t.Fatalf("reliability = %+v, want available (deployment activity exists)", rel)
	}
	if *rel.Value <= 0 || *rel.Value >= 100 {
		t.Errorf("reliability = %v, want strictly between 0 and 100 (one major incident day exists in a large window)", *rel.Value)
	}
}

func TestDORAMetricsApplicationFilterIsolatesData(t *testing.T) {
	deployments := []deploymentEvent{
		{Application: "checkout", DeployedAt: "2026-06-01T10:00:00Z", Status: "success", Source: "ci"},
		{Application: "search", DeployedAt: "2026-06-01T10:00:00Z", Status: "failure", Source: "ci"},
	}
	report := computeDORA(deployments, nil, "checkout", 3650)
	cfr := report.Metrics[2] // change_failure_rate
	if cfr.Name != "change_failure_rate" || !cfr.Available || *cfr.Value != 0 {
		t.Fatalf("checkout-scoped change_failure_rate = %+v, want 0 (only its own successful deploy counted)", cfr)
	}
}

// blockedIdentityFields must never appear as a JSON field on any DORA or
// adoption type — structural proof, not policy, that individual ranking
// is impossible from this data (Constraint).
var blockedIdentityFields = []string{"author", "user", "user_id", "principal", "email", "engineer", "developer_id", "committer"}

func assertNoIdentityFields(t *testing.T, typ reflect.Type) {
	t.Helper()
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		lower := strings.ToLower(name)
		for _, blocked := range blockedIdentityFields {
			if strings.Contains(lower, blocked) {
				t.Errorf("%s.%s: field name %q resembles an individual-identity field, forbidden by the team/application-only constraint", typ.Name(), typ.Field(i).Name, name)
			}
		}
	}
}

func TestNoDORAOrAdoptionTypeExposesIndividualIdentity(t *testing.T) {
	for _, v := range []any{deploymentEvent{}, incidentEvent{}, doraReport{}, doraMetric{}, adoptionReport{}} {
		assertNoIdentityFields(t, reflect.TypeOf(v))
	}
}

func TestAdoptionMetricsUnavailableBeforeActivation(t *testing.T) {
	specs := []specStatusCount{{status: "draft", created: mustTime(t, "2026-06-01T00:00:00Z"), hasDate: true}}
	report := computeAdoption("", specs, nil, mustTime(t, "2026-07-01T00:00:00Z"))
	if report.Activated {
		t.Fatal("expected not activated with only a draft spec and no history")
	}
	if report.RetentionRatio != nil {
		t.Errorf("retention must be unavailable before activation, got %v", *report.RetentionRatio)
	}
	if report.RetentionReason == "" {
		t.Error("expected a retention unavailability reason")
	}
	if report.TaskSuccessRatio != nil {
		t.Errorf("task success must be unavailable with zero resolved specs, got %v", *report.TaskSuccessRatio)
	}
}

func TestAdoptionMetricsComputesActivationTimeToGateRetentionTaskSuccess(t *testing.T) {
	specs := []specStatusCount{
		{status: "draft", created: mustTime(t, "2026-06-01T00:00:00Z"), hasDate: true}, // earliest artifact
		{status: "done", created: mustTime(t, "2026-06-08T00:00:00Z"), hasDate: true},  // activation candidate
		{status: "abandoned", created: mustTime(t, "2026-06-10T00:00:00Z"), hasDate: true},
		{status: "blocked", created: mustTime(t, "2026-06-12T00:00:00Z"), hasDate: true},
		{status: "in-progress", created: mustTime(t, "2026-06-14T00:00:00Z"), hasDate: true},
	}
	history := []historyRecord{
		{GeneratedAt: "2026-06-08T00:00:00Z", Outcome: "pass"}, // activation, week 0
		{GeneratedAt: "2026-06-09T00:00:00Z", Outcome: "fail"}, // not counted (not pass)
		{GeneratedAt: "2026-06-22T00:00:00Z", Outcome: "pass"}, // week 2
	}
	now := mustTime(t, "2026-06-29T00:00:00Z") // 3 weeks after activation
	report := computeAdoption("", specs, history, now)

	if !report.Activated {
		t.Fatal("expected activation from the done spec / passing history")
	}
	wantActivatedAt := mustTime(t, "2026-06-08T00:00:00Z")
	if report.ActivatedAt != wantActivatedAt.Format(time.RFC3339) {
		t.Errorf("ActivatedAt = %s, want %s", report.ActivatedAt, wantActivatedAt.Format(time.RFC3339))
	}
	if report.TimeToFirstGateDays == nil || *report.TimeToFirstGateDays != 7 {
		t.Fatalf("TimeToFirstGateDays = %v, want 7 (2026-06-01 -> 2026-06-08)", report.TimeToFirstGateDays)
	}
	if report.RetentionRatio == nil {
		t.Fatal("expected a retention ratio once activated")
	}
	// Weeks since activation (2026-06-08 -> 2026-06-29) = 3 weeks span;
	// active weeks with a passing record: week of 06-08 and week of 06-22 = 2.
	if *report.RetentionRatio <= 0 || *report.RetentionRatio > 1 {
		t.Errorf("RetentionRatio = %v, want a value in (0, 1]", *report.RetentionRatio)
	}
	if report.SpecsDone != 1 || report.SpecsAbandoned != 1 || report.SpecsBlocked != 1 || report.SpecsPending != 2 {
		t.Errorf("spec counts = done=%d abandoned=%d blocked=%d pending=%d, want 1/1/1/2 (draft + in-progress)",
			report.SpecsDone, report.SpecsAbandoned, report.SpecsBlocked, report.SpecsPending)
	}
	if report.TaskSuccessRatio == nil || *report.TaskSuccessRatio != 1.0/3.0 {
		t.Fatalf("TaskSuccessRatio = %v, want 1/3 (1 done of 3 resolved)", report.TaskSuccessRatio)
	}
}

func TestEventsHousekeepingListAndPurge(t *testing.T) {
	repo := newGitRepo(t)
	oldFile := filepath.Join(repo, ".pose", "events", "deployments", "2020-01.jsonl")
	recentFile := filepath.Join(repo, ".pose", "events", "deployments", time.Now().UTC().Format("2006-01")+".jsonl")
	for _, p := range []string{oldFile, recentFile} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(`{"application":"a"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var out, errB bytes.Buffer
	if code := cmdEventsHousekeeping(repo, []string{"list-expired", "--older-than-days", "400"}, &out, &errB); code != 0 {
		t.Fatalf("list-expired exit=%d err=%s", code, errB.String())
	}
	if !strings.Contains(out.String(), "2020-01.jsonl") || strings.Contains(out.String(), filepath.Base(recentFile)) {
		t.Errorf("list-expired output wrong set: %s", out.String())
	}

	out.Reset()
	if code := cmdEventsHousekeeping(repo, []string{"purge", "--older-than-days", "400"}, &out, &errB); code != 0 {
		t.Fatalf("purge (dry-run) exit=%d err=%s", code, errB.String())
	}
	if _, err := os.Stat(oldFile); err != nil {
		t.Fatal("dry-run purge must not delete anything")
	}

	out.Reset()
	if code := cmdEventsHousekeeping(repo, []string{"purge", "--older-than-days", "400", "--apply"}, &out, &errB); code != 0 {
		t.Fatalf("purge --apply exit=%d err=%s", code, errB.String())
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("expected the expired file to be removed")
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Error("recent file must be preserved")
	}
}

func TestDORAAndAdoptionMetricsCLIEndToEnd(t *testing.T) {
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"record-deployment", "--application", "checkout", "--environment", "prod", "--status", "success", "--source", "ci", "--lead-time-seconds", "120"}, &out, &errB); code != 0 {
			t.Fatalf("record-deployment exit=%d err=%s", code, errB.String())
		}
		out.Reset()
		if code := Main([]string{"dora-metrics", "--application", "checkout", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("dora-metrics exit=%d err=%s", code, errB.String())
		}
		var report doraReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
		if report.Application != "checkout" {
			t.Errorf("expected application filter to round-trip: %+v", report)
		}

		out.Reset()
		if code := Main([]string{"adoption-metrics", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("adoption-metrics exit=%d err=%s", code, errB.String())
		}
		var adoption adoptionReport
		if err := json.Unmarshal(out.Bytes(), &adoption); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
	})
}
