package cli

// Concurrent checks (spec pose-validate-concurrent-steps): checks run up to
// --jobs at a time, results keep matrix order, an exclusive check runs alone,
// and a cancelled run stops its process groups.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func shellCheck(name, script string) map[string]any {
	return map[string]any{"name": name, "program": "sh", "args": []string{"-c", script}, "severity": "required"}
}

func checksFixture(t *testing.T, checks ...map[string]any) string {
	t.Helper()
	return guardrailFixture(t, map[string]any{
		"defaults": map[string]any{"mode": "strict"},
		"stacks":   map[string]any{"go": map[string]any{"checks": checks}},
	})
}

// rendezvous is a check that passes only if its peer is running at the same
// time: each announces itself and waits, briefly, for the other.
func rendezvous(dir, self, peer string) string {
	return fmt.Sprintf(`touch %[1]s/%[2]s; i=0; while [ ! -e %[1]s/%[3]s ]; do i=$((i+1)); [ $i -gt 150 ] && exit 1; sleep 0.02; done`, shellQuote(dir), self, peer)
}

// R1, R3: two checks that need each other pass at --jobs 2 and fail at
// --jobs 1, and the report records the concurrency.
func TestValidateRunsChecksConcurrentlyWithinJobs(t *testing.T) {
	for jobs, want := range map[int]int{2: 0, 1: 1} {
		dir := t.TempDir()
		root := checksFixture(t, shellCheck("a", rendezvous(dir, "a", "b")), shellCheck("b", rendezvous(dir, "b", "a")))
		code, out := runValidate(t, root, "--jobs", strconv.Itoa(jobs), "--json", "result.json")
		if code != want {
			t.Fatalf("jobs %d: exit %d, want %d: %s", jobs, code, want, out)
		}
		run := loadRun(t, root)
		if run.Jobs != jobs || run.WallSeconds <= 0 {
			t.Fatalf("jobs %d: the report says jobs=%d wall=%v", jobs, run.Jobs, run.WallSeconds)
		}
	}
}

// R1: results keep matrix order, whatever order the checks finish in; R5: a
// failure, a timeout and an optional failure read as in a serial run.
func TestValidateKeepsMatrixOrderAndOutcomes(t *testing.T) {
	checks := func() []map[string]any {
		timeout := map[string]any{"name": "times-out", "program": "sleep", "args": []string{"5"}, "severity": "required", "timeoutSeconds": 1}
		optional := shellCheck("optional", "exit 3")
		optional["severity"] = "optional"
		return []map[string]any{shellCheck("slow", "sleep 0.3"), shellCheck("fails", "echo boom; exit 7"), timeout, optional, shellCheck("fast", "true")}
	}
	var runs []validationRunResult
	var codes []int
	for _, jobs := range []string{"1", "4"} {
		root := checksFixture(t, checks()...)
		code, out := runValidate(t, root, "--jobs", jobs, "--json", "result.json")
		codes = append(codes, code)
		run := loadRun(t, root)
		runs = append(runs, run)
		if !strings.Contains(out, "exit=7") {
			t.Fatalf("jobs %s: the failure's detail is missing: %s", jobs, out)
		}
	}
	if codes[0] != 1 || codes[1] != codes[0] {
		t.Fatalf("exit codes: %v", codes)
	}
	if runs[0].Counts != runs[1].Counts {
		t.Fatalf("counts differ: %+v %+v", runs[0].Counts, runs[1].Counts)
	}
	for i := range runs[0].Checks {
		a, b := runs[0].Checks[i], runs[1].Checks[i]
		if a.Name != b.Name || a.Outcome != b.Outcome || a.LimitState != b.LimitState || (a.ExitCode == nil) != (b.ExitCode == nil) {
			t.Fatalf("check %d differs: %+v %+v", i, a, b)
		}
	}
	if runs[1].Checks[0].Name != "slow" || runs[1].Checks[4].Name != "fast" {
		t.Fatalf("order: %+v", runs[1].Checks)
	}
}

// R2: an exclusive check runs with no other check running.
func TestValidateExclusiveCheckRunsAlone(t *testing.T) {
	dir := t.TempDir()
	busy := func(name string) map[string]any {
		return shellCheck(name, fmt.Sprintf(`touch %[1]s/%[2]s; sleep 0.3; rm %[1]s/%[2]s`, shellQuote(dir), name))
	}
	alone := shellCheck("alone", fmt.Sprintf(`for f in %s/*; do [ -e "$f" ] && { echo "$f is running"; exit 1; }; done; sleep 0.2; for f in %s/*; do [ -e "$f" ] && exit 1; done; exit 0`, shellQuote(dir), shellQuote(dir)))
	alone["exclusive"] = true
	root := checksFixture(t, busy("a"), busy("b"), alone, busy("c"), busy("d"))
	if code, out := runValidate(t, root, "--jobs", "4", "--json", "result.json"); code != 0 {
		t.Fatalf("exclusive: %d %s", code, out)
	}
}

// R5: a cancelled run stops every running check's process group, and starts
// nothing more.
func TestValidateCancelStopsRunningChecks(t *testing.T) {
	dir := t.TempDir()
	hold := func(name string) map[string]any {
		return shellCheck(name, fmt.Sprintf(`echo $$ > %[1]s/%[2]s.pid; sleep 60`, shellQuote(dir), name))
	}
	never := shellCheck("never", fmt.Sprintf(`touch %s/never`, shellQuote(dir)))
	root := checksFixture(t, hold("a"), hold("b"), never)
	cmd := exec.Command(poseBinary(t), "validate", "--jobs", "2", "--json", "result.json")
	cmd.Dir = root
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pidFiles := []string{filepath.Join(dir, "a.pid"), filepath.Join(dir, "b.pid")}
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		_, errA := os.Stat(pidFiles[0])
		_, errB := os.Stat(pidFiles[1])
		if errA == nil && errB == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("the checks did not start")
		}
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("a cancelled validation exited 0")
	}
	for _, file := range pidFiles {
		waitForProcessGone(t, file)
	}
	if _, err := os.Stat(filepath.Join(dir, "never")); err == nil {
		t.Fatal("a check started after the run was cancelled")
	}
	raw, err := os.ReadFile(filepath.Join(root, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var run validationRunResult
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	for _, check := range run.Checks {
		if check.LimitState != "cancelled" {
			t.Fatalf("a check of a cancelled run: %+v", check)
		}
	}
}

// The default is bounded and at least one.
func TestDefaultValidationJobsIsBounded(t *testing.T) {
	if jobs := defaultValidationJobs(); jobs < 1 || jobs > 8 {
		t.Fatalf("default jobs: %d", jobs)
	}
}

// --jobs refuses what is not a positive count.
func TestValidateJobsFlagRefusesBadValues(t *testing.T) {
	root := checksFixture(t, shellCheck("a", "true"))
	for _, value := range []string{"0", "-1", "x"} {
		if code, _ := runValidate(t, root, "--jobs", value); code != 2 {
			t.Fatalf("--jobs %s: exit %d", value, code)
		}
	}
}
