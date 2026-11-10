package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Spec pose-worktree-runtime-qos-and-measurement R8.
//
// The measurement of the optional runtime on a real repository. It runs only when
// POSE_RUNTIME_MEASURE names the output file, because it takes minutes and measures the
// repository it runs in. The samples, scenarios and thresholds below are fixed here and in the
// spec before any number is read; the test fails when a threshold is not met, and writes the
// figures either way.
const (
	measureColdSamples  = 3
	measureWarmSamples  = 5
	measureChurnSamples = 3
	measureShortSamples = 10
	// Thresholds (fixed before the first run).
	thresholdWarmOverCold    = 0.60 // median warm run <= 0.6 x median cold run
	thresholdChurnOverCold   = 0.80 // a restarted runtime inherits nothing: median >= 0.8 x cold
	thresholdShortUnderHeavy = 3.0  // p95 of a short query under heavy load <= 3 x idle p95 ...
	thresholdShortFloorMS    = 250  // ... or 250 ms, whichever is more
	thresholdRSSMiB          = 1536 // resident set of the runtime after the warm runs
	thresholdFallbackExtraMS = 500  // trying a runtime that is not there costs at most this
	thresholdTwoHeavyOverOne = 2.5  // T7: two simultaneous heavy checks on the one heavy slot: wall <= 2.5 x one warm run
)

type measureReport struct {
	Environment map[string]any     `json:"environment"`
	Scenarios   map[string][]int64 `json:"scenarios_ms"`
	Figures     map[string]float64 `json:"figures"`
	Metrics     map[string]int64   `json:"runtime_metrics"`
	Verdicts    map[string]string  `json:"verdicts"`
}

// runtimeProc reads the runtime process's cumulative CPU time (user + system, ms) and its
// resident set (MiB) from /proc; ok is false where there is none.
func runtimeProc(pid int) (cpuMS int64, rssMiB float64, ok bool) {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, 0, false
	}
	// The command name is in parentheses and may hold spaces: fields count from after it.
	text := string(stat)
	fields := strings.Fields(text[strings.LastIndex(text, ")")+2:])
	if len(fields) < 13 {
		return 0, 0, false
	}
	utime, _ := strconv.ParseInt(fields[11], 10, 64)
	stime, _ := strconv.ParseInt(fields[12], 10, 64)
	status, _ := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			kb, _ := strconv.ParseFloat(strings.Fields(line)[1], 64)
			rssMiB = kb / 1024
		}
	}
	const clockTicksPerSecond = 100
	return (utime + stime) * 1000 / clockTicksPerSecond, rssMiB, true
}

func measureMedian(v []int64) int64 {
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

func measureP95(v []int64) int64 {
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := (len(s)*95 + 99) / 100
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}

func TestMeasureTheRuntimeOnTheRealRepository(t *testing.T) {
	out := os.Getenv("POSE_RUNTIME_MEASURE")
	if out == "" {
		t.Skip("set POSE_RUNTIME_MEASURE=<output file> to measure the runtime on this repository")
	}
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	spec := os.Getenv("POSE_RUNTIME_MEASURE_SPEC")
	if spec == "" {
		spec = "spec:pose-governance-audit-skips-pure-git-reads"
	}
	bin := filepath.Join(t.TempDir(), "pose")
	if b, err := exec.Command("go", "build", "-o", bin, "../../cmd/pose").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, b)
	}
	rtDir, _ := os.MkdirTemp(os.Getenv("TMPDIR"), "rtm")
	defer os.RemoveAll(rtDir)
	env := append(os.Environ(), "POSE_RUNTIME_DIR="+rtDir+"/r")
	for _, drop := range []string{"POSE_PROJECT_ROOT", "POSE_PROJECT_ROOTS", "POSE_DEFAULT_PROJECT_ID", "CLAUDECODE"} {
		env = append(env, drop+"=")
	}
	run := func(args ...string) (string, int64, int) {
		cmd := exec.Command(bin, args...)
		cmd.Dir, cmd.Env = repo, env
		var o, e bytes.Buffer
		cmd.Stdout, cmd.Stderr = &o, &e
		start := time.Now()
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return o.String() + "\x00" + e.String(), time.Since(start).Milliseconds(), code
	}
	runtimeRun := func(args ...string) (stdout string, ms int64, via string) {
		o, ms, _ := run(append([]string{"runtime", "run", "--envelope"}, args...)...)
		var doc struct {
			Via, Stdout, Stderr string
		}
		doc.Via = ""
		var raw map[string]any
		if err := json.Unmarshal([]byte(strings.SplitN(o, "\x00", 2)[0]), &raw); err != nil {
			t.Fatalf("not an envelope: %q", o)
		}
		via, _ = raw["via"].(string)
		so, _ := raw["stdout"].(string)
		se, _ := raw["stderr"].(string)
		return so + "\x00" + se, ms, via
	}
	rep := measureReport{Scenarios: map[string][]int64{}, Figures: map[string]float64{}, Verdicts: map[string]string{}}
	head, _ := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	commits, _ := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	gitv, _ := exec.Command("git", "--version").Output()
	kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	rep.Environment = map[string]any{"cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"go": runtime.Version(), "git": strings.TrimSpace(string(gitv)), "kernel": strings.TrimSpace(string(kernel)), "repo_head": strings.TrimSpace(string(head)),
		"repo_commits": strings.TrimSpace(string(commits)), "check": "review-check " + spec, "short_query": "surface-check " + spec,
		"samples": map[string]int{"cold": measureColdSamples, "warm": measureWarmSamples, "churn": measureChurnSamples, "short": measureShortSamples}}

	fail := func(name, format string, a ...any) {
		rep.Verdicts[name] = "FAIL: " + fmt.Sprintf(format, a...)
		t.Errorf("%s: "+format, append([]any{name}, a...)...)
	}
	pass := func(name, format string, a ...any) { rep.Verdicts[name] = "pass: " + fmt.Sprintf(format, a...) }

	// S1: cold, standalone, a new process each time.
	var coldOut string
	for i := 0; i < measureColdSamples; i++ {
		o, ms, _ := run("review-check", spec)
		if i == 0 {
			coldOut = o
		}
		rep.Scenarios["cold_standalone"] = append(rep.Scenarios["cold_standalone"], ms)
		if o != coldOut {
			t.Fatalf("standalone output changed between samples %d and 0", i)
		}
	}
	cold := measureMedian(rep.Scenarios["cold_standalone"])

	// S2: asking for a runtime that is not there.
	for i := 0; i < measureColdSamples; i++ {
		o, ms, via := runtimeRun("review-check", spec)
		rep.Scenarios["fallback_no_runtime"] = append(rep.Scenarios["fallback_no_runtime"], ms)
		if via != "standalone" || o != coldOut {
			t.Fatalf("a run with no runtime: via %q, same output %v", via, o == coldOut)
		}
	}
	extra := measureMedian(rep.Scenarios["fallback_no_runtime"]) - cold
	rep.Figures["fallback_extra_ms"] = float64(extra)
	if extra <= thresholdFallbackExtraMS {
		pass("T6 fallback overhead", "%d ms <= %d ms", extra, thresholdFallbackExtraMS)
	} else {
		fail("T6 fallback overhead", "%d ms > %d ms", extra, thresholdFallbackExtraMS)
	}

	// S3: resident, warm. The first run is the cold one inside the runtime; it is recorded apart.
	if o, _, _ := run("runtime", "start"); !strings.Contains(o, "started") {
		t.Fatalf("start: %s", o)
	}
	defer run("runtime", "stop")
	pidOf := func() int {
		o, _, _ := run("runtime", "status", "--json")
		var d struct {
			PID int `json:"pid"`
		}
		_ = json.Unmarshal([]byte(strings.SplitN(o, "\x00", 2)[0]), &d)
		return d.PID
	}
	pid := pidOf()
	// Per scenario: the CPU time the runtime process spent and its resident set afterwards.
	mark := func(scenario string, since int64) int64 {
		cpu, rss, ok := runtimeProc(pid)
		if ok {
			rep.Figures["cpu_ms_"+scenario] = float64(cpu - since)
			rep.Figures["rss_mib_after_"+scenario] = rss
		}
		return cpu
	}
	cpuNow := func() int64 { cpu, _, _ := runtimeProc(pid); return cpu }
	cpuBefore := cpuNow()
	for i := 0; i < measureWarmSamples+1; i++ {
		o, ms, via := runtimeRun("review-check", spec)
		if via != "runtime" || o != coldOut {
			t.Fatalf("a hosted run: via %q, same output as standalone %v", via, o == coldOut)
		}
		if i == 0 {
			rep.Scenarios["resident_first"] = []int64{ms}
		} else {
			rep.Scenarios["resident_warm"] = append(rep.Scenarios["resident_warm"], ms)
		}
	}
	mark("resident_first_and_warm", cpuBefore)
	warm := measureMedian(rep.Scenarios["resident_warm"])
	rep.Figures["warm_over_cold"] = float64(warm) / float64(cold)
	if float64(warm) <= thresholdWarmOverCold*float64(cold) {
		pass("T1 warm over cold", "%d ms / %d ms = %.2f <= %.2f", warm, cold, float64(warm)/float64(cold), thresholdWarmOverCold)
	} else {
		fail("T1 warm over cold", "%d ms / %d ms = %.2f > %.2f", warm, cold, float64(warm)/float64(cold), thresholdWarmOverCold)
	}
	// S5: short queries, idle then under a heavy check.
	shortArgs := []string{"surface-check", spec}
	shortStandalone, _, _ := run(shortArgs...)
	for i := 0; i < measureShortSamples; i++ {
		so, ms, via := runtimeRun(shortArgs...)
		if via != "runtime" || so != shortStandalone {
			t.Fatalf("a short query went %s, same output as standalone %v", via, so == shortStandalone)
		}
		rep.Scenarios["short_idle"] = append(rep.Scenarios["short_idle"], ms)
	}
	cpuIdleShortEnd := cpuNow()
	_ = cpuIdleShortEnd
	cpuHeavyStart := cpuNow()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); runtimeRun("review-check", spec) }()
	time.Sleep(1500 * time.Millisecond)
	for i := 0; i < measureShortSamples; i++ {
		so, ms, via := runtimeRun(shortArgs...)
		if via != "runtime" || so != shortStandalone {
			t.Fatalf("a short query under load went %s, same output as standalone %v", via, so == shortStandalone)
		}
		rep.Scenarios["short_under_heavy"] = append(rep.Scenarios["short_under_heavy"], ms)
	}
	wg.Wait()
	mark("heavy_with_short_queries", cpuHeavyStart)
	// S6: two heavy checks at once on the one heavy slot: the second waits its turn.
	twoStart := time.Now()
	cpuTwo := cpuNow()
	var two [2]string
	var twoVia [2]string
	var wg2 sync.WaitGroup
	for k := 0; k < 2; k++ {
		wg2.Add(1)
		go func(k int) { defer wg2.Done(); two[k], _, twoVia[k] = runtimeRun("review-check", spec) }(k)
	}
	wg2.Wait()
	twoWall := time.Since(twoStart).Milliseconds()
	mark("two_simultaneous_heavy", cpuTwo)
	rep.Scenarios["two_heavy_wall"] = []int64{twoWall}
	rep.Figures["two_heavy_over_one_warm"] = float64(twoWall) / float64(warm)
	if two[0] != coldOut || two[1] != coldOut || twoVia[0] != "runtime" || twoVia[1] != "runtime" {
		fail("T7 two simultaneous heavy checks", "via %v, outputs equal to standalone %v %v", twoVia, two[0] == coldOut, two[1] == coldOut)
	} else if float64(twoWall) <= thresholdTwoHeavyOverOne*float64(warm) {
		pass("T7 two simultaneous heavy checks", "both through the runtime with the standalone output; wall %d ms <= %.1f x %d ms", twoWall, thresholdTwoHeavyOverOne, warm)
	} else {
		fail("T7 two simultaneous heavy checks", "wall %d ms > %.1f x %d ms", twoWall, thresholdTwoHeavyOverOne, warm)
	}
	idle, loaded := measureP95(rep.Scenarios["short_idle"]), measureP95(rep.Scenarios["short_under_heavy"])
	bound := int64(thresholdShortUnderHeavy * float64(idle))
	if bound < thresholdShortFloorMS {
		bound = thresholdShortFloorMS
	}
	rep.Figures["short_p95_idle_ms"], rep.Figures["short_p95_under_heavy_ms"] = float64(idle), float64(loaded)
	if loaded <= bound {
		pass("T3 short under heavy", "p95 %d ms <= %d ms (idle p95 %d ms)", loaded, bound, idle)
	} else {
		fail("T3 short under heavy", "p95 %d ms > %d ms (idle p95 %d ms)", loaded, bound, idle)
	}
	// Memory of the runtime after the warm runs and the concurrent phase.
	status, _, _ := run("runtime", "status", "--json")
	var st struct {
		PID int `json:"pid"`
	}
	_ = json.Unmarshal([]byte(strings.SplitN(status, "\x00", 2)[0]), &st)
	if raw, err := os.ReadFile("/proc/" + strconv.Itoa(st.PID) + "/status"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "VmHWM:") || strings.HasPrefix(line, "VmRSS:") {
				f := strings.Fields(line)
				kb, _ := strconv.ParseFloat(f[1], 64)
				rep.Figures[strings.TrimSuffix(f[0], ":")+"_mib"] = kb / 1024
			}
		}
		if mib := rep.Figures["VmHWM_mib"]; mib <= thresholdRSSMiB {
			pass("T4 runtime memory", "peak resident %.0f MiB <= %d MiB", mib, thresholdRSSMiB)
		} else {
			fail("T4 runtime memory", "peak resident %.0f MiB > %d MiB", mib, thresholdRSSMiB)
		}
	} else {
		rep.Verdicts["T4 runtime memory"] = "not measured: no /proc on this platform"
	}

	st2, _, _ := run("runtime", "status", "--json")
	var full struct {
		Metrics map[string]int64 `json:"metrics"`
	}
	_ = json.Unmarshal([]byte(strings.SplitN(st2, "\x00", 2)[0]), &full)
	rep.Metrics = full.Metrics
	if len(rep.Metrics) == 0 {
		t.Fatalf("the runtime status carries no lane metrics: %s", st2)
	}

	// S4: a restart inherits nothing.
	for i := 0; i < measureChurnSamples; i++ {
		run("runtime", "stop")
		run("runtime", "start")
		o, ms, via := runtimeRun("review-check", spec)
		if via != "runtime" || o != coldOut {
			t.Fatalf("a run after a restart: via %q, same output %v", via, o == coldOut)
		}
		rep.Scenarios["restart_then_run"] = append(rep.Scenarios["restart_then_run"], ms)
	}
	churn := measureMedian(rep.Scenarios["restart_then_run"])
	rep.Figures["churn_over_cold"] = float64(churn) / float64(cold)
	if float64(churn) >= thresholdChurnOverCold*float64(cold) {
		pass("T2 restart inherits nothing", "%d ms / %d ms = %.2f >= %.2f", churn, cold, float64(churn)/float64(cold), thresholdChurnOverCold)
	} else {
		fail("T2 restart inherits nothing", "%d ms / %d ms = %.2f < %.2f", churn, cold, float64(churn)/float64(cold), thresholdChurnOverCold)
	}
	pass("T5 output", "every hosted run, the review-check and the short query idle and under load, printed exactly what the standalone run printed (%d and %d bytes)", len(coldOut), len(shortStandalone))

	raw, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
