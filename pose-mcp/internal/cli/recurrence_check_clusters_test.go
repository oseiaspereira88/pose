package cli

// recurrence-check judges failure clusters, not attempts (spec
// pose-recurrence-check-resolved-clusters).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func rec(at time.Time, task, hash, outcome string) historyRecord {
	return historyRecord{GeneratedAt: at.UTC().Format(time.RFC3339), TaskSlug: task, ReportType: "standard", StableHash: hash, Outcome: outcome}
}

func TestRecurrenceGroupsResolveFailuresByALaterPassOfTheSameHash(t *testing.T) {
	now := time.Now()
	cutoff := now.AddDate(0, 0, -14)
	h := func(n int) time.Time { return now.Add(-time.Duration(n) * time.Hour) }
	tests := []struct {
		name         string
		records      []historyRecord
		wantUnres    int
		wantResolved int
	}{
		{"fail then pass does not flag", []historyRecord{rec(h(4), "t", "a", "fail"), rec(h(3), "t", "a", "fail"), rec(h(2), "t", "a", "fail"), rec(h(1), "t", "a", "pass")}, 0, 1},
		{"fail fail fail stays unresolved", []historyRecord{rec(h(3), "t", "a", "fail"), rec(h(2), "t", "a", "fail"), rec(h(1), "t", "a", "fail")}, 3, 0},
		{"failures after the latest pass are unresolved", []historyRecord{rec(h(4), "t", "a", "fail"), rec(h(3), "t", "a", "pass"), rec(h(2), "t", "a", "fail")}, 1, 0},
		{"a pass of another hash resolves nothing", []historyRecord{rec(h(3), "t", "a", "fail"), rec(h(2), "t", "a", "fail"), rec(h(1), "t", "b", "pass")}, 2, 0},
		{"a record without hash clusters with its task", []historyRecord{rec(h(3), "t", "", "fail"), rec(h(2), "t", "", "fail"), rec(h(1), "t", "", "pass")}, 0, 1},
		{"unresolved hashes add up per task", []historyRecord{rec(h(4), "t", "a", "fail"), rec(h(3), "t", "a", "fail"), rec(h(2), "t", "b", "fail")}, 3, 0},
		{"records out of order are judged by time", []historyRecord{rec(h(1), "t", "a", "pass"), rec(h(3), "t", "a", "fail")}, 0, 1},
		{"records before the window are ignored", []historyRecord{rec(now.AddDate(0, 0, -30), "t", "a", "fail"), rec(h(1), "t", "a", "pass")}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			groups, resolved := recurrenceGroups(tc.records, cutoff, false)
			got := 0
			for _, g := range groups {
				got += len(g.records)
			}
			if got != tc.wantUnres || len(resolved) != tc.wantResolved {
				t.Fatalf("unresolved=%d resolved=%d, want %d and %d", got, len(resolved), tc.wantUnres, tc.wantResolved)
			}
		})
	}
}

func TestRecurrenceGroupsIncludePassKeepsCountingEveryRecord(t *testing.T) {
	now := time.Now()
	records := []historyRecord{rec(now.Add(-3*time.Hour), "t", "a", "fail"), rec(now.Add(-2*time.Hour), "t", "b", "pass"), rec(now.Add(-time.Hour), "t", "a", "fail")}
	groups, resolved := recurrenceGroups(records, now.AddDate(0, 0, -14), true)
	if len(groups) != 1 || len(groups[0].records) != 3 || len(resolved) != 0 {
		t.Fatalf("groups=%+v resolved=%+v", groups, resolved)
	}
}

func writeRecurrenceHistory(t *testing.T, records []historyRecord) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, r := range records {
		fmt.Fprintf(&b, "{\"generated_at\":%q,\"task_slug\":%q,\"report_type\":\"standard\",\"stable_hash\":%q,\"outcome\":%q}\n", r.GeneratedAt, r.TaskSlug, r.StableHash, r.Outcome)
	}
	if err := os.WriteFile(filepath.Join(dir, "standard-t.jsonl"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRecurrenceCheckDisclosesResolvedClustersAndExitsClean(t *testing.T) {
	now := time.Now()
	root := writeRecurrenceHistory(t, []historyRecord{
		rec(now.Add(-4*time.Hour), "validate-native", "5b47855e60f6", "fail"),
		rec(now.Add(-3*time.Hour), "validate-native", "5b47855e60f6", "fail"),
		rec(now.Add(-2*time.Hour), "validate-native", "5b47855e60f6", "fail"),
		rec(now.Add(-time.Hour), "validate-native", "5b47855e60f6", "pass"),
	})
	var out, errb bytes.Buffer
	if code := cmdRecurrenceCheck(root, []string{"--strict"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out.String(), errb.String())
	}
	text := out.String()
	for _, want := range []string{"resolved", "validate-native (standard)", "stable_hash 5b47855e", "3 failed run(s)", "recurrence.flagged_keys=0", "recurrence.resolved_clusters=1"} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "recurrent") {
		t.Errorf("resolved cluster was flagged:\n%s", text)
	}
}

func TestRecurrenceCheckStillFlagsAnUnresolvedCluster(t *testing.T) {
	now := time.Now()
	root := writeRecurrenceHistory(t, []historyRecord{
		rec(now.Add(-4*time.Hour), "t", "a", "fail"),
		rec(now.Add(-3*time.Hour), "t", "a", "fail"),
		rec(now.Add(-2*time.Hour), "t", "a", "fail"),
		rec(now.Add(-time.Hour), "t", "b", "pass"),
	})
	var out, errb bytes.Buffer
	if code := cmdRecurrenceCheck(root, []string{"--strict"}, &out, &errb); code != 1 {
		t.Fatalf("exit %d, want 1\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "recurrent") || !strings.Contains(out.String(), "recurrence.flagged_keys=1") {
		t.Errorf("unresolved cluster not flagged:\n%s", out.String())
	}
}

// A task that keeps alternating is reported as flapping, never gated, even
// when every failure cluster resolved (spec pose-recurrence-flapping-signal).
func TestRecurrenceFlappingIsReportedNotGated(t *testing.T) {
	now := time.Now()
	outcomes := []string{"fail", "pass", "fail", "pass", "fail", "pass"}
	records := []historyRecord{}
	for i, o := range outcomes {
		records = append(records, rec(now.Add(time.Duration(i-len(outcomes))*time.Hour), "validate-native", "5b47855e60f6", o))
	}
	root := writeRecurrenceHistory(t, records)
	var out, errb bytes.Buffer
	if code := cmdRecurrenceCheck(root, []string{"--strict"}, &out, &errb); code != 0 {
		t.Fatalf("flapping gated the run: exit %d\n%s%s", code, out.String(), errb.String())
	}
	text := out.String()
	for _, want := range []string{"flapping", "5 transitions between failing and passing in 6 runs", "recurrence.flapping_keys=1"} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}

	// Below the threshold, and with a higher threshold, nothing is reported.
	out.Reset()
	cmdRecurrenceCheck(root, []string{"--strict", "--flap-threshold", "6"}, &out, &errb)
	if !strings.Contains(out.String(), "recurrence.flapping_keys=0") {
		t.Errorf("a raised threshold still reported flapping:\n%s", out.String())
	}
	steady := writeRecurrenceHistory(t, []historyRecord{
		rec(now.Add(-3*time.Hour), "validate-native", "h", "fail"),
		rec(now.Add(-2*time.Hour), "validate-native", "h", "pass"),
		rec(now.Add(-time.Hour), "validate-native", "h", "pass"),
	})
	out.Reset()
	cmdRecurrenceCheck(steady, []string{"--strict"}, &out, &errb)
	if !strings.Contains(out.String(), "recurrence.flapping_keys=0") {
		t.Errorf("a single recovery was reported as flapping:\n%s", out.String())
	}
}
