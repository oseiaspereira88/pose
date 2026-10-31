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
	"runtime"
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

// Review round 1, B6: the default is judged by the budget in force — cgroup
// memory headroom and CPU quota, v2 and v1, ancestors included — and stays
// bounded when nothing can be read.
func TestDefaultValidationJobsFollowsTheBudget(t *testing.T) {
	const gib = uint64(1) << 30
	meminfo := fmt.Sprintf("MemTotal: 1 kB\nMemAvailable: %d kB\n", 64*gib/1024)
	v2 := "30 20 0:26 / /sys/fs/cgroup rw - cgroup2 cgroup2 rw\n"
	v1 := "31 20 0:27 / /sys/fs/cgroup/memory rw - cgroup cgroup rw,memory\n32 20 0:28 / /sys/fs/cgroup/cpu,cpuacct rw - cgroup cgroup rw,cpu,cpuacct\n"
	cases := []struct {
		name  string
		files map[string]string
		cores int
		want  int
	}{
		{"unlimited host", map[string]string{"/proc/meminfo": meminfo}, 64, 8},
		{"half the cores", map[string]string{"/proc/meminfo": meminfo}, 6, 3},
		{"little memory", map[string]string{"/proc/meminfo": fmt.Sprintf("MemAvailable: %d kB\n", 4*gib/1024)}, 16, 2},
		{"nothing readable", map[string]string{}, 64, 4},
		{"small machine", map[string]string{"/proc/meminfo": meminfo}, 1, 1},
		{"no memory left is not unknown memory", map[string]string{"/proc/meminfo": "MemAvailable: 0 kB\n"}, 64, 1},
		{"cgroup v2 memory", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "0::/a/b\n", "/sys/fs/cgroup/a/b/memory.max": fmt.Sprint(5 * gib), "/sys/fs/cgroup/a/b/memory.current": fmt.Sprint(2 * gib)}, 64, 2},
		{"cgroup v2 ancestor", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "0::/a/b\n", "/sys/fs/cgroup/a/b/memory.max": "max", "/sys/fs/cgroup/a/memory.max": fmt.Sprint(2 * gib)}, 64, 1},
		{"cgroup v2 cpu", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "0::/a\n", "/sys/fs/cgroup/a/cpu.max": "200000 100000"}, 64, 1},
		{"cgroup v2 no cpu limit", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "0::/a\n", "/sys/fs/cgroup/a/cpu.max": "max 100000"}, 8, 4},
		{"cgroup full", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "0::/a\n", "/sys/fs/cgroup/a/memory.max": fmt.Sprint(gib), "/sys/fs/cgroup/a/memory.current": fmt.Sprint(gib)}, 64, 1},
		{"cgroup v2 subtree mount", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": "30 20 0:26 /tenant /sys/fs/cgroup rw - cgroup2 cgroup2 rw\n", "/proc/self/cgroup": "0::/tenant/worker\n", "/sys/fs/cgroup/worker/memory.max": fmt.Sprint(2 * gib), "/sys/fs/cgroup/worker/memory.current": fmt.Sprint(gib)}, 64, 1},
		{"cgroup v1", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v1, "/proc/self/cgroup": "4:memory:/a\n3:cpu,cpuacct:/a\n", "/sys/fs/cgroup/memory/a/memory.limit_in_bytes": fmt.Sprint(9 * gib), "/sys/fs/cgroup/memory/a/memory.usage_in_bytes": "0", "/sys/fs/cgroup/cpu,cpuacct/a/cpu.cfs_quota_us": "1000000", "/sys/fs/cgroup/cpu,cpuacct/a/cpu.cfs_period_us": "100000"}, 64, 5},
		{"cgroup v1 combined controller", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v1, "/proc/self/cgroup": "3:cpu,cpuacct:/a\n", "/sys/fs/cgroup/cpu,cpuacct/a/cpu.cfs_quota_us": "200000", "/sys/fs/cgroup/cpu,cpuacct/a/cpu.cfs_period_us": "100000"}, 64, 1},
		{"cgroup v1 unlimited", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v1, "/proc/self/cgroup": "4:memory:/\n", "/sys/fs/cgroup/memory/memory.limit_in_bytes": "9223372036854771712"}, 16, 8},
		{"malformed cgroup files", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": v2, "/proc/self/cgroup": "garbage\n0::/a\n", "/sys/fs/cgroup/a/memory.max": "x", "/sys/fs/cgroup/a/cpu.max": "y z w"}, 8, 4},
		{"a cgroup whose mount cannot be found", map[string]string{"/proc/meminfo": meminfo, "/proc/self/cgroup": "0::/a\n"}, 64, 4},
		{"a path outside the mounted subtree", map[string]string{"/proc/meminfo": meminfo, "/proc/self/mountinfo": "30 20 0:26 /tenant /sys/fs/cgroup rw - cgroup2 cgroup2 rw\n", "/proc/self/cgroup": "0::/other\n"}, 64, 4},
	}
	for _, c := range cases {
		read := func(path string) ([]byte, error) {
			if body, ok := c.files[path]; ok {
				return []byte(body), nil
			}
			return nil, os.ErrNotExist
		}
		if got := defaultValidationJobs(effectiveValidationResources(read, c.cores)); got != c.want {
			t.Errorf("%s: %d jobs, want %d", c.name, got, c.want)
		}
	}
	if jobs := defaultValidationJobs(effectiveValidationResources(os.ReadFile, runtime.NumCPU())); jobs < 1 || jobs > 8 {
		t.Fatalf("default jobs on this machine: %d", jobs)
	}
}

// Review round 1, B1: when is read as the check is about to start, so an
// earlier check decides it — serially, with --verbose, and past an exclusive
// barrier.
func TestValidateWhenIsReadBeforeTheCheckStarts(t *testing.T) {
	when := func(check map[string]any, key string) map[string]any {
		check["when"] = map[string]any{key: "generated"}
		return check
	}
	for _, args := range [][]string{{"--jobs", "1"}, {"--verbose"}} {
		root := checksFixture(t, shellCheck("generate", "touch generated"), when(shellCheck("consume", "test -f generated"), "fileExists"))
		if code, out := runValidate(t, root, append(args, "--json", "result.json")...); code != 0 {
			t.Fatalf("%v: %d %s", args, code, out)
		}
		if run := loadRun(t, root); run.Checks[1].Outcome != "pass" {
			t.Fatalf("%v: a check enabled by an earlier one: %+v", args, run.Checks[1])
		}
		// The marker is there when the run is planned; an earlier check
		// removes it.
		root = checksFixture(t, shellCheck("setup", "true"), shellCheck("clean", "rm generated"), when(shellCheck("skipped", "exit 1"), "fileExists"))
		if err := os.WriteFile(filepath.Join(root, "mod", "generated"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if code, out := runValidate(t, root, append(args, "--json", "result.json")...); code != 0 {
			t.Fatalf("%v: %d %s", args, code, out)
		}
		if run := loadRun(t, root); run.Checks[2].Outcome != "skipped" {
			t.Fatalf("%v: a check disabled by an earlier one: %+v", args, run.Checks[2])
		}
	}
	consume := when(shellCheck("consume", "test -f generated"), "fileExists")
	consume["exclusive"] = true
	root := checksFixture(t, shellCheck("generate", "sleep 0.2; touch generated"), consume)
	if code, out := runValidate(t, root, "--jobs", "4", "--json", "result.json"); code != 0 {
		t.Fatalf("exclusive: %d %s", code, out)
	}
	if run := loadRun(t, root); run.Checks[1].Outcome != "pass" {
		t.Fatalf("past an exclusive barrier: %+v", run.Checks[1])
	}
}

// Review round 1, B2: changed-scope skips keep matrix order among the checks
// that ran.
func TestChangedScopeSkipsKeepMatrixOrder(t *testing.T) {
	for _, changed := range []string{"a", "b"} {
		for _, jobs := range []string{"1", "4"} {
			root := scopeFixture(t, false)
			if err := os.WriteFile(filepath.Join(root, changed, "changed.go"), []byte("package x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if code, out := runValidate(t, root, "--changed-from", "HEAD", "--jobs", jobs, "--json", "result.json"); code != 0 {
				t.Fatalf("%d %s", code, out)
			}
			run := loadRun(t, root)
			if len(run.Checks) != 2 || run.Checks[0].Module != "a" || run.Checks[1].Module != "b" {
				t.Fatalf("changed %s, jobs %s: %+v", changed, jobs, run.Checks)
			}
			if run.Counts.Skipped != 1 || run.Counts.Executed != 1 {
				t.Fatalf("changed %s, jobs %s: %+v", changed, jobs, run.Counts)
			}
		}
	}
}

// Review round 1, B3: exclusive is a key of the matrix check accepts, as a
// boolean.
func TestCheckAcceptsExclusiveAsABoolean(t *testing.T) {
	exclusive := shellCheck("alone", "true")
	exclusive["exclusive"] = true
	_, out := runPose(t, checksFixture(t, exclusive), "check", "--strict")
	if strings.Contains(out, "unknown key 'exclusive'") || strings.Contains(out, "exclusive must be") {
		t.Fatalf("a boolean exclusive was refused: %s", out)
	}
	exclusive["exclusive"] = "yes"
	if _, out := runPose(t, checksFixture(t, exclusive), "check", "--strict"); !strings.Contains(out, "exclusive must be a boolean") {
		t.Fatalf("a non-boolean exclusive was accepted: %s", out)
	}
}

// Review round 1, B4: the preparatory go list stops with the run.
func TestValidateCancelStopsThePreparatoryGoList(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	listing := filepath.Join(dir, "listing")
	fake := "#!/bin/sh\necho $$ > " + shellQuote(listing) + "\nsleep 60\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	root := checksFixture(t, map[string]any{"name": "t", "program": "go", "args": []string{"test", "./..."}, "severity": "required"})
	if err := os.MkdirAll(filepath.Join(root, "mod", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := poseBinary(t)
	cmd := exec.Command(binary, "validate", "--jobs", "1", "--json", "result.json")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(listing); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("go list did not start")
		}
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a cancelled validation exited 0")
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the run hung on its preparatory go list")
	}
	waitForProcessGone(t, listing)
}

// Review round 1, B5: a cancelled run is never a success, whatever the
// severity of what it stopped.
func TestACancelledRunIsNeverASuccess(t *testing.T) {
	for _, jobs := range []string{"1", "2"} {
		dir := t.TempDir()
		optional := shellCheck("optional", fmt.Sprintf(`echo $$ > %s/pid; sleep 60`, shellQuote(dir)))
		optional["severity"] = "optional"
		root := checksFixture(t, shellCheck("required", "true"), optional)
		cmd := exec.Command(poseBinary(t), "validate", "--jobs", jobs, "--json", "result.json")
		cmd.Dir = root
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		pid := filepath.Join(dir, "pid")
		for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(20 * time.Millisecond) {
			if _, err := os.Stat(pid); err == nil {
				break
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				t.Fatal("the check did not start")
			}
		}
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Fatal(err)
		}
		err := cmd.Wait()
		exit, _ := err.(*exec.ExitError)
		if exit == nil || exit.ExitCode() != 1 {
			t.Fatalf("jobs %s: a cancelled run: %v", jobs, err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "result.json"))
		if err != nil {
			t.Fatal(err)
		}
		var run validationRunResult
		if err := json.Unmarshal(raw, &run); err != nil {
			t.Fatal(err)
		}
		if run.Outcome != "fail" {
			t.Fatalf("jobs %s: outcome %s", jobs, run.Outcome)
		}
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
