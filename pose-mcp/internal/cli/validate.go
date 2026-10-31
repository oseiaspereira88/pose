package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/harne8/pose-mcp/internal/cli/cliout"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

type validationWhen struct {
	FileExists    string `json:"fileExists"`
	FileNotExists string `json:"fileNotExists"`
	// FileNotExistsAny (spec pose-stack-catalog-expansion): skip when ANY of
	// these files exist — expresses manager-priority exclusion (e.g. a pip
	// check yields to poetry/pipenv when their lockfiles are present)
	// without needing repeated single-field checks.
	FileNotExistsAny []string `json:"fileNotExistsAny"`
}

type validationCheck struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Program       string            `json:"program"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	Severity      string            `json:"severity"`
	EvidenceClass string            `json:"evidenceClass,omitempty"`
	When          validationWhen    `json:"when"`
	// Runtime guardrails (spec pose-validation-runtime-guardrails).
	TimeoutSeconds int    `json:"timeoutSeconds"` // 0 = defaults.timeoutSeconds (600)
	Isolation      string `json:"isolation"`      // "" | "required" — required never runs locally
	// Exclusive runs the check with no other check running (spec
	// pose-validate-concurrent-steps R2).
	Exclusive bool `json:"exclusive,omitempty"`
}

// validationEntry is one line of a validation run, in matrix order: a module
// note, a skipped check or a check to execute.
type validationEntry struct {
	note      string
	check     validationCheck
	moduleDir string
	severity  string
	// skip is why the check does not run. silent marks a changed-scope skip,
	// recorded in order without a step line.
	skip     string
	silent   bool
	isolated bool
	result   checkResult
	done     chan struct{}
}

// resolveWhen evaluates the check's condition at the moment it would start,
// so an earlier check, finished by then, can decide it (review round 1, B1).
func (e *validationEntry) resolveWhen() {
	if e.skip == "" {
		e.skip = validationSkipReason(e.moduleDir, e.check.When)
	}
}

// execute runs the check and records its outcome in the entry's result. It
// touches nothing else, so entries can run concurrently; the caller counts.
func (e *validationEntry) execute(parent context.Context, matrix validationMatrix, verbose bool, stdout, stderr io.Writer, beforeStream func()) {
	check, result := e.check, &e.result
	timeout := check.TimeoutSeconds
	if timeout <= 0 {
		timeout = matrix.Defaults.TimeoutSeconds
	}
	if timeout <= 0 {
		timeout = 600 // documented safe default
	}
	maxOutput := matrix.Defaults.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = 1 << 20 // 1 MiB documented safe default
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	capture := &tailBuffer{capacity: 4096}
	limiter := &outputLimiter{limit: maxOutput, cancel: cancel}
	packages := &goTestPackages{}
	execArgs := sanitizeGoCheckArgs(ctx, e.moduleDir, check.Program, check.Args)
	if parent.Err() != nil {
		cancel()
		e.markCancelled()
		return
	}
	cmd := exec.CommandContext(ctx, check.Program, execArgs...)
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcessGroup(cmd) }
	cmd.Dir = e.moduleDir
	// The renderer owns the terminal while a step is active: a child
	// writing to it would fight the status line, and streaming every
	// check's output is what buried the run's own verdict. The output is
	// captured and shown when the check fails; --verbose streams it
	// (spec pose-cli-output-rendering-system R7).
	if verbose {
		beforeStream()
		cmd.Stdout = io.MultiWriter(stdout, capture, limiter, packages)
		cmd.Stderr = io.MultiWriter(stderr, capture, limiter)
	} else {
		cmd.Stdout = io.MultiWriter(capture, limiter, packages)
		cmd.Stderr = io.MultiWriter(capture, limiter)
	}
	cmd.Env = os.Environ()
	for key, value := range check.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	started := time.Now()
	err := cmd.Run()
	deadline := ctx.Err()
	cancel()
	result.DurationSeconds = time.Since(started).Seconds()
	result.Output = redactSecrets(capture.String(), check.Env)
	var exitErr *exec.ExitError
	switch {
	case limiter.exceeded:
		// Explicit guardrail state: the check flooded output and was
		// cancelled (never conflated with a normal check failure).
		result.Outcome, result.LimitState = "error", "output-limit"
		result.Output = fmt.Sprintf("output limit exceeded (%d bytes) — process group terminated", maxOutput)
	case deadline == context.DeadlineExceeded:
		result.Outcome, result.LimitState = "error", "timeout"
		result.Output = fmt.Sprintf("timeout after %ds — process group terminated", timeout)
	case parent.Err() != nil:
		e.markCancelled()
	case err == nil:
		result.Outcome = "pass"
		// go test exits 0 when a pattern selects nothing: a check that ran
		// no test has no evidence (spec pose-validate-refuses-empty-go-checks).
		if pattern := goTestRunPattern(check); pattern != "" && packages.ranNone() {
			result.Outcome, result.LimitState = "fail", "no-tests"
			result.Output = fmt.Sprintf("no test ran: -run %s selected nothing in the check's packages\n%s", pattern, result.Output)
		}
	case errors.As(err, &exitErr):
		code := exitErr.ExitCode()
		result.Outcome, result.ExitCode = "fail", &code
	default:
		// Infrastructure failure: the tool never ran (R3 keeps this
		// distinguishable from a real check failure).
		result.Outcome = "error"
		result.Output = redactSecrets(err.Error(), check.Env)
	}
}

// goTestRunPattern is the -run pattern of a go test check, "" for any other
// check.
func goTestRunPattern(check validationCheck) string {
	if check.Program != "go" || len(check.Args) == 0 || check.Args[0] != "test" {
		return ""
	}
	pattern := ""
	for i, arg := range check.Args {
		name, value, inline := strings.Cut(arg, "=")
		if name != "-run" && name != "-test.run" && name != "--run" {
			continue
		}
		if inline {
			pattern = value
		} else if i+1 < len(check.Args) {
			pattern = check.Args[i+1]
		}
	}
	return pattern
}

// goTestPackages reads go test's per-package result lines from a check's
// standard output, a line at a time and in bounded memory.
type goTestPackages struct {
	partial   []byte
	overflow  bool
	withTests int
	empty     int
}

// goTestLineLimit bounds a line kept across writes; a package result line is
// far shorter, and a longer line is test output.
const goTestLineLimit = 4096

func (p *goTestPackages) Write(data []byte) (int, error) {
	for rest := data; len(rest) > 0; {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			if len(p.partial)+len(rest) > goTestLineLimit {
				p.overflow = true
				p.partial = p.partial[:0]
			} else {
				p.partial = append(p.partial, rest...)
			}
			break
		}
		if !p.overflow && len(p.partial)+i <= goTestLineLimit {
			p.partial = append(p.partial, rest[:i]...)
			p.line(string(p.partial))
		}
		p.partial, p.overflow = p.partial[:0], false
		rest = rest[i+1:]
	}
	return len(data), nil
}

func (p *goTestPackages) line(line string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}
	switch {
	case fields[0] == "ok" && strings.HasSuffix(line, "[no tests to run]"):
		p.empty++
	case fields[0] == "ok":
		p.withTests++
	case fields[0] == "?" && strings.HasSuffix(line, "[no test files]"):
		p.empty++
	}
}

// ranNone reports that the output named packages and none of them ran a test.
func (p *goTestPackages) ranNone() bool {
	return p.empty > 0 && p.withTests == 0
}

// markCancelled records a check the run's cancellation stopped or kept from
// starting.
func (e *validationEntry) markCancelled() {
	e.result.Outcome, e.result.LimitState = "error", "cancelled"
	e.result.Output = "validation cancelled — process group terminated"
}

// dispatchValidationEntries starts the entries in matrix order, up to jobs at
// a time; an exclusive check waits for the running ones and runs alone (spec
// pose-validate-concurrent-steps R1–R2). A cancelled run starts nothing more.
func dispatchValidationEntries(ctx context.Context, entries []*validationEntry, jobs int, matrix validationMatrix) {
	slots := make(chan struct{}, jobs)
	var running sync.WaitGroup
	cancelled := func(entry *validationEntry) bool {
		if ctx.Err() == nil {
			return false
		}
		entry.markCancelled()
		close(entry.done)
		return true
	}
	for _, entry := range entries {
		if entry.note != "" || entry.silent || entry.isolated {
			continue
		}
		if entry.check.Exclusive {
			running.Wait()
			if cancelled(entry) {
				continue
			}
			if entry.resolveWhen(); entry.skip == "" {
				entry.execute(ctx, matrix, false, nil, nil, nil)
			}
			close(entry.done)
			continue
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			cancelled(entry)
			continue
		}
		if cancelled(entry) {
			<-slots
			continue
		}
		running.Add(1)
		go func(entry *validationEntry) {
			defer running.Done()
			defer func() { <-slots }()
			if entry.resolveWhen(); entry.skip == "" {
				entry.execute(ctx, matrix, false, nil, nil, nil)
			}
			close(entry.done)
		}(entry)
	}
}

// validationResources is what a run may use. memoryKnown separates an empty
// reading from none; unresolved says a cgroup applies whose limits could not
// be located.
type validationResources struct {
	cpus        float64
	memory      uint64
	memoryKnown bool
	unresolved  bool
}

func (r *validationResources) limitMemory(available uint64) {
	if !r.memoryKnown || available < r.memory {
		r.memory, r.memoryKnown = available, true
	}
}

func (r *validationResources) limitCPU(quota, period float64) {
	if quota > 0 && period > 0 && quota/period < r.cpus {
		r.cpus = quota / period
	}
}

// defaultValidationJobs is half the cores, bounded by the memory available at
// roughly 1.5 GiB per check — a Go test link is what filled a small machine —
// and by 8, never below 1; while the budget is not established it stays at 4
// or below.
func defaultValidationJobs(available validationResources) int {
	jobs := int(available.cpus / 2)
	if available.memoryKnown {
		if byMemory := int(available.memory / (3 << 29)); byMemory < jobs {
			jobs = byMemory
		}
	}
	if (!available.memoryKnown || available.unresolved) && jobs > 4 {
		jobs = 4
	}
	if jobs > 8 {
		jobs = 8
	}
	if jobs < 1 {
		jobs = 1
	}
	return jobs
}

// cgroupMount is one mounted cgroup hierarchy: the subtree root it exposes,
// where, and, for v1, its controllers.
type cgroupMount struct {
	root, point string
	controllers []string
	unified     bool
}

// decodeMountField undoes mountinfo's octal escapes of whitespace and
// backslashes (review round 3, B6a).
func decodeMountField(field string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(field)
}

// parseCgroupMounts reads the cgroup hierarchies of /proc/self/mountinfo.
func parseCgroupMounts(raw string) []cgroupMount {
	var mounts []cgroupMount
	for _, line := range strings.Split(raw, "\n") {
		before, after, ok := strings.Cut(line, " - ")
		fields, tail := strings.Fields(before), strings.Fields(after)
		if !ok || len(fields) < 5 || len(tail) < 3 {
			continue
		}
		mount := cgroupMount{root: decodeMountField(fields[3]), point: decodeMountField(fields[4])}
		switch tail[0] {
		case "cgroup2":
			mount.unified = true
		case "cgroup":
			mount.controllers = strings.Split(tail[2], ",")
		default:
			continue
		}
		mounts = append(mounts, mount)
	}
	return mounts
}

// visibleCgroupDirs translates a cgroup path into the directories of mount,
// innermost first, up to the mount's own root; false when the path is not in
// that subtree.
func (m cgroupMount) visibleCgroupDirs(path string) ([]string, bool) {
	root := strings.TrimRight(m.root, "/")
	if path != root && !strings.HasPrefix(path, root+"/") {
		return nil, false
	}
	relative := strings.TrimPrefix(path, root)
	var dirs []string
	for {
		dirs = append(dirs, strings.TrimRight(m.point, "/")+relative)
		if relative == "" {
			return dirs, true
		}
		relative = relative[:strings.LastIndex(relative, "/")]
	}
}

// effectiveValidationResources reads the machine's cores and available memory
// and narrows them to the limits of the process's cgroup and its visible
// ancestors, v2 and v1, located through the process's own mounts, so a
// container is judged by its budget (review rounds 1–2, B6). read is
// os.ReadFile outside tests.
func effectiveValidationResources(read func(string) ([]byte, error), cores int) validationResources {
	found := validationResources{cpus: float64(cores)}
	if raw, err := read("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if rest, ok := strings.CutPrefix(line, "MemAvailable:"); ok {
				if fields := strings.Fields(rest); len(fields) > 0 {
					if kib, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
						found.limitMemory(kib * 1024)
					}
				}
			}
		}
	}
	membership, err := read("/proc/self/cgroup")
	if err != nil {
		return found
	}
	var mounts []cgroupMount
	if raw, err := read("/proc/self/mountinfo"); err == nil {
		mounts = parseCgroupMounts(string(raw))
	}
	number := func(path string) (uint64, bool) {
		raw, err := read(path)
		if err != nil {
			return 0, false
		}
		n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
		return n, err == nil
	}
	memoryLimit := func(limitFile, usedFile string) {
		if limit, ok := number(limitFile); ok && limit < 1<<62 {
			used, _ := number(usedFile)
			if used > limit {
				used = limit
			}
			found.limitMemory(limit - used)
		}
	}
	// A hierarchy that applies but is not mounted where this process can see
	// it leaves the budget unknown, not unlimited.
	// A hierarchy can be mounted more than once, a narrower view hiding the
	// ancestors a fuller one shows: every view counts (review round 3, B6b).
	resolve := func(path string, match func(cgroupMount) bool) []string {
		var all []string
		for _, mount := range mounts {
			if match(mount) {
				if dirs, ok := mount.visibleCgroupDirs(path); ok {
					all = append(all, dirs...)
				}
			}
		}
		if len(all) == 0 {
			found.unresolved = true
		}
		return all
	}
	hasController := func(name string) func(cgroupMount) bool {
		return func(m cgroupMount) bool {
			for _, controller := range m.controllers {
				if controller == name {
					return true
				}
			}
			return false
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(string(membership)), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		path := strings.TrimRight(parts[2], "/")
		if parts[1] == "" {
			for _, dir := range resolve(path, func(m cgroupMount) bool { return m.unified }) {
				memoryLimit(dir+"/memory.max", dir+"/memory.current")
				if cpu, err := read(dir + "/cpu.max"); err == nil {
					if fields := strings.Fields(string(cpu)); len(fields) == 2 {
						quota, errQuota := strconv.ParseFloat(fields[0], 64)
						period, errPeriod := strconv.ParseFloat(fields[1], 64)
						if errQuota == nil && errPeriod == nil {
							found.limitCPU(quota, period)
						}
					}
				}
			}
			continue
		}
		controllers := "," + parts[1] + ","
		if strings.Contains(controllers, ",memory,") {
			for _, dir := range resolve(path, hasController("memory")) {
				memoryLimit(dir+"/memory.limit_in_bytes", dir+"/memory.usage_in_bytes")
			}
		}
		if strings.Contains(controllers, ",cpu,") {
			for _, dir := range resolve(path, hasController("cpu")) {
				quota, okQuota := number(dir + "/cpu.cfs_quota_us")
				period, okPeriod := number(dir + "/cpu.cfs_period_us")
				if okQuota && okPeriod {
					found.limitCPU(float64(quota), float64(period))
				}
			}
		}
	}
	return found
}

// validationTimingNotes says where the run's time went: its wall time beside
// the sum of its checks, and the slowest checks (spec
// pose-validate-concurrent-steps R4).
func validationTimingNotes(run validationRunResult) []string {
	var total float64
	ran := make([]checkResult, 0, len(run.Checks))
	for _, check := range run.Checks {
		if check.DurationSeconds > 0 {
			total += check.DurationSeconds
			ran = append(ran, check)
		}
	}
	if len(ran) == 0 {
		return nil
	}
	sort.SliceStable(ran, func(i, j int) bool { return ran[i].DurationSeconds > ran[j].DurationSeconds })
	lines := []string{fmt.Sprintf("[time] wall %.1fs, checks %.1fs, jobs %d", run.WallSeconds, total, run.Jobs)}
	for i, check := range ran {
		if i == 5 {
			break
		}
		lines = append(lines, fmt.Sprintf("[time] %.1fs %s/%s", check.DurationSeconds, check.Module, check.Name))
	}
	return lines
}

type validationStack struct {
	Checks []validationCheck `json:"checks"`
}

type validationOverride struct {
	Stack                string            `json:"stack"`
	Mode                 string            `json:"mode"`
	Checks               []validationCheck `json:"checks"`
	ReplaceDefaultChecks bool              `json:"replaceDefaultChecks"`
}

type validationMatrix struct {
	Defaults struct {
		Mode           string `json:"mode"`
		TimeoutSeconds int    `json:"timeoutSeconds"` // safe default 600 when 0
		MaxOutputBytes int    `json:"maxOutputBytes"` // safe default 1 MiB when 0
	} `json:"defaults"`
	Stacks           map[string]validationStack           `json:"stacks"`
	ModuleOverrides  map[string]validationOverride        `json:"moduleOverrides"`
	DeliveryProfiles map[string]posemodel.DeliveryProfile `json:"deliveryProfiles,omitempty"`
}

func parseValidationMatrix(raw []byte) (validationMatrix, error) {
	var matrix validationMatrix
	err := json.Unmarshal(raw, &matrix)
	return matrix, err
}

func parseStructuredChecks(raw []byte) ([]validationCheck, error) {
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		return nil, err
	}
	var checks []validationCheck
	for _, stack := range matrix.Stacks {
		for _, check := range stack.Checks {
			if check.Program != "" {
				checks = append(checks, check)
			}
		}
	}
	return checks, nil
}

type validationModule struct {
	Rel, Abs, Stack string
}

func confinedRelativePath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return true
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func discoverValidationModulesIn(files posemodel.Files, root string) ([]validationModule, error) {
	// testdata/fixture(s): see index.go's scanModules for why these are
	// excluded (spec pose-upgrade-path-audit-fixes).
	ignored := map[string]bool{".git": true, ".qwen": true, "node_modules": true, "vendor": true, ".venv": true, ".pnpm-store": true, "target": true, "dist": true, "build": true, ".next": true, "coverage": true, ".pose": true, "testdata": true, "fixture": true, "fixtures": true}
	// A gitignored subtree (a locally-checked-out project the repository
	// deliberately excludes, e.g. a vendored tool) is never a governed
	// deliverable module — see posemodel.GitIgnoredPaths (spec
	// pose-discovery-gitignore-and-root-alias-fix).
	gitIgnored := posemodel.GitIgnoredPathsIn(files, root)
	byPath := map[string]string{}
	err := posemodel.WalkDirIn(files, root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && ignored[entry.Name()] {
				return filepath.SkipDir
			}
			if path != root {
				if rel, relErr := filepath.Rel(root, path); relErr == nil && (gitIgnored[filepath.ToSlash(rel)+"/"] || gitIgnored[filepath.ToSlash(rel)]) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		// spec pose-stack-catalog-expansion: the Python manager (poetry/
		// pipenv/pip/setuptools/pep517) is resolved by the matrix `when`
		// predicates, not at discovery time — every marker maps to the
		// "python" stack. stackForManifestFile is shared with scanModules
		// (spec pose-validation-scanner-consolidation) so a stack-detection
		// fix here applies there too.
		stack := stackForManifestFile(entry.Name())
		if stack != "" {
			// Existing overwrite-by-walk-order semantics preserved
			// unchanged for compatibility; `pose stacks` surfaces the full
			// per-directory catalog resolution (including any conflicting
			// markers) independently of what this single-stack map picks.
			byPath[filepath.Dir(path)] = stack
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	modules := make([]validationModule, 0, len(byPath))
	for abs, stack := range byPath {
		if stack == "java" && isAndroidModuleIn(files, abs) {
			// A Gradle module carrying an AndroidManifest.xml is an Android
			// app/library, not a generic JVM backend module — conflating the
			// two under "java" made every Android module's domain
			// misleading (spec pose-upgrade-path-audit-fixes). Kotlin-only
			// non-Android Gradle modules are unaffected: they have no
			// AndroidManifest.xml and keep classifying as "java".
			stack = "android"
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, err
		}
		modules = append(modules, validationModule{filepath.ToSlash(rel), abs, stack})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Rel < modules[j].Rel })
	return modules, nil
}

// discoverValidationModules reads and writes the filesystem directly; discoverValidationModulesIn takes the
// operation's files.
func discoverValidationModules(root string) ([]validationModule, error) {
	return discoverValidationModulesIn(posemodel.NativeFiles(), root)
}

// isAndroidModule reports whether a Gradle module directory is an Android
// app/library rather than a generic JVM module, by the one file every
// Android module has and no other Gradle module does: AndroidManifest.xml,
// at its conventional root-relative location or directly in the module
// (some legacy layouts skip the src/main nesting).
func isAndroidModuleIn(files posemodel.Files, moduleDir string) bool {
	for _, candidate := range []string{
		filepath.Join(moduleDir, "src", "main", "AndroidManifest.xml"),
		filepath.Join(moduleDir, "AndroidManifest.xml"),
	} {
		if _, err := files.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

// isAndroidModule reads and writes the filesystem directly; isAndroidModuleIn takes the
// operation's files.
func isAndroidModule(moduleDir string) bool {
	return isAndroidModuleIn(posemodel.NativeFiles(), moduleDir)
}

// validationSkipReason returns the deterministic selection reason when a
// check must be skipped ("" = run it). Every skip is recorded with its
// reason in the structured result (spec pose-structured-validation-results).
func validationSkipReason(module string, when validationWhen) string {
	if when.FileExists != "" {
		if !confinedRelativePath(when.FileExists) {
			return "when.fileExists escapes the module: " + when.FileExists
		}
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileExists))); err != nil {
			return "when.fileExists not met: " + when.FileExists
		}
	}
	if when.FileNotExists != "" {
		if !confinedRelativePath(when.FileNotExists) {
			return "when.fileNotExists escapes the module: " + when.FileNotExists
		}
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileNotExists))); err == nil {
			return "when.fileNotExists violated: " + when.FileNotExists + " exists"
		}
	}
	for _, excl := range when.FileNotExistsAny {
		if !confinedRelativePath(excl) {
			return "when.fileNotExistsAny escapes the module: " + excl
		}
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(excl))); err == nil {
			return "when.fileNotExistsAny violated: higher-priority manager marker present: " + excl
		}
	}
	return ""
}

func validateStructuredMatrixPaths(matrix validationMatrix) error {
	validate := func(scope string, checks []validationCheck) error {
		for index, check := range checks {
			for field, path := range map[string]string{"fileExists": check.When.FileExists, "fileNotExists": check.When.FileNotExists} {
				if !confinedRelativePath(path) {
					return fmt.Errorf("%s.checks[%d].when.%s must remain inside its module", scope, index, field)
				}
			}
			for i, path := range check.When.FileNotExistsAny {
				if !confinedRelativePath(path) {
					return fmt.Errorf("%s.checks[%d].when.fileNotExistsAny[%d] must remain inside its module", scope, index, i)
				}
			}
		}
		return nil
	}
	for name, stack := range matrix.Stacks {
		if err := validate("stacks."+name, stack.Checks); err != nil {
			return err
		}
	}
	for name, override := range matrix.ModuleOverrides {
		if err := validate("moduleOverrides."+name, override.Checks); err != nil {
			return err
		}
	}
	return nil
}

func validateDeliveryMatrixContract(matrix validationMatrix) error {
	for scope, checks := range matrix.Stacks {
		for _, check := range checks.Checks {
			if check.EvidenceClass != "" && !posemodel.ValidEvidenceClasses[check.EvidenceClass] {
				return fmt.Errorf("stacks.%s check %s has unknown evidenceClass %q", scope, check.Name, check.EvidenceClass)
			}
		}
	}
	for scope, override := range matrix.ModuleOverrides {
		for _, check := range override.Checks {
			if check.EvidenceClass != "" && !posemodel.ValidEvidenceClasses[check.EvidenceClass] {
				return fmt.Errorf("moduleOverrides.%s check %s has unknown evidenceClass %q", scope, check.Name, check.EvidenceClass)
			}
		}
	}
	for name, profile := range matrix.DeliveryProfiles {
		if _, _, err := posemodel.ParseDeliveryRef(profile.Kind + ":profile"); err != nil {
			return fmt.Errorf("deliveryProfiles.%s has invalid kind %q", name, profile.Kind)
		}
		for _, class := range append(append([]string{}, profile.RequiredEvidenceClasses...), profile.AnyEvidenceClasses...) {
			if !posemodel.ValidEvidenceClasses[class] {
				return fmt.Errorf("deliveryProfiles.%s has unknown evidence class %q", name, class)
			}
		}
		if profile.Kind == "surface" {
			requiresReachability := false
			for _, class := range profile.RequiredEvidenceClasses {
				if class == "reachability" {
					requiresReachability = true
				}
			}
			hasIntegrationLevel := false
			for _, class := range append(append([]string{}, profile.RequiredEvidenceClasses...), profile.AnyEvidenceClasses...) {
				if class == "integration" || class == "e2e" {
					hasIntegrationLevel = true
				}
			}
			if !requiresReachability || !hasIntegrationLevel {
				return fmt.Errorf("deliveryProfiles.%s surface profile requires reachability and integration/e2e evidence", name)
			}
		}
		if profile.Kind == "capability" {
			hasIntegration := false
			for _, class := range append(append([]string{}, profile.RequiredEvidenceClasses...), profile.AnyEvidenceClasses...) {
				if class == "integration" {
					hasIntegration = true
				}
			}
			if !hasIntegration {
				return fmt.Errorf("deliveryProfiles.%s capability profile requires integration evidence", name)
			}
		}
	}
	return nil
}

func matrixHasLegacyChecks(matrix validationMatrix) bool {
	for _, stack := range matrix.Stacks {
		for _, check := range stack.Checks {
			if check.Command != "" {
				return true
			}
		}
	}
	for _, override := range matrix.ModuleOverrides {
		for _, check := range override.Checks {
			if check.Command != "" {
				return true
			}
		}
	}
	return false
}

func cmdValidate(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode, stackFilter, moduleFilter, reportTask := "", "", "", ""
	jsonOut, junitOut, sarifOut, planOut := "", "", "", ""
	changedFrom, changedTo := "", ""
	explain := false
	verbose := false
	autoReport := false
	rootOnly := false
	workspaceFilter := ""
	jobs := 0
	var isolationChecks []checkResult
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--report":
			autoReport = true
		case "--explain":
			explain = true
		case "--verbose":
			// A check's output is captured by default; --verbose streams it
			// (spec pose-cli-output-rendering-system R7).
			verbose = true
		case "--root-only":
			// Documented alias of --module . (spec
			// pose-monorepo-validation-advisory, R2) — sugar over the
			// existing selector, no new selection logic.
			rootOnly = true
		case "--stack", "--module", "--workspace", "--report-task", "--json", "--json-out", "--junit", "--sarif", "--emit-plan", "--changed-from", "--changed-to", "--jobs":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: %s requires a value.\n", "Erro: %s exige um valor.\n"), args[i])
				return 2
			}
			i++
			switch args[i-1] {
			case "--jobs":
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					render(stdout, stderr).Failure(cliText(locale, "--jobs must be a positive integer", "--jobs deve ser um inteiro positivo"))
					return 2
				}
				jobs = n
			case "--stack":
				stackFilter = args[i]
			case "--module":
				moduleFilter = filepath.ToSlash(filepath.Clean(args[i]))
			case "--workspace":
				// Documented alias of --module <path> (R3) that resolves a
				// package/crate name to its module path, for the case
				// where the operator knows the workspace member's name but
				// not its relative directory.
				workspaceFilter = args[i]
			case "--report-task":
				reportTask = args[i]
			case "--json-out":
				jsonOut = args[i]
			case "--json":
				// Everywhere else --json prints to stdout; here it named a file.
				// It keeps working, announced as deprecated (spec
				// pose-cli-output-machine-channel, R2).
				jsonOut = args[i]
				render(stdout, stderr).Hint(cliText(locale, "validate --json <path> is deprecated; use --json-out <path>.", "validate --json <path> está depreciado; use --json-out <path>."))
			case "--junit":
				junitOut = args[i]
			case "--sarif":
				sarifOut = args[i]
			case "--emit-plan":
				planOut = args[i]
			case "--changed-from":
				changedFrom = args[i]
			case "--changed-to":
				changedTo = args[i]
			}
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if rootOnly && moduleFilter != "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --root-only and --module are mutually exclusive.", "Erro: --root-only e --module são mutuamente exclusivos."))
		return 2
	}
	if workspaceFilter != "" && (moduleFilter != "" || rootOnly) {
		fmt.Fprintln(stderr, cliText(locale, "Error: --workspace, --module and --root-only are mutually exclusive.", "Erro: --workspace, --module e --root-only são mutuamente exclusivos."))
		return 2
	}
	if rootOnly {
		moduleFilter = "."
	}
	for _, out := range []string{jsonOut, junitOut, sarifOut, planOut} {
		if out != "" && !confinedRelativePath(out) {
			fmt.Fprintln(stderr, cliText(locale, "Error: result output paths must remain inside the project.", "Erro: paths de saída de resultado devem permanecer dentro do projeto."))
			return 2
		}
	}
	// The report --report writes records this run, so it reads this run's own
	// output. It used to look for a pose-validate.latest.log that native
	// validation never writes, and every report said "Fill manually" and "No
	// validation output detected" under an outcome it could not support (spec
	// pose-validate-report-carries-its-run).
	// The report records this run from the run's own structured result, not
	// from its printed output (spec pose-cli-output-rendering-system R5).
	// Capturing stdout to re-parse it is what made every printed line a
	// contract.
	if stackFilter != "" && !map[string]bool{"node": true, "go": true, "rust": true, "java": true, "python": true, "dotnet": true, "contract": true}[stackFilter] {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid --stack: %s\n", "Erro: --stack inválido: %s\n"), stackFilter)
		return 2
	}
	if moduleFilter == ".." || strings.HasPrefix(moduleFilter, "../") || filepath.IsAbs(moduleFilter) {
		fmt.Fprintln(stderr, cliText(locale, "Error: --module must remain inside the project.", "Erro: --module deve permanecer dentro do projeto."))
		return 2
	}
	matrixPath := filepath.Join(root, ".pose", "indexes", "validation-matrix.json")
	raw, err := os.ReadFile(matrixPath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: validation matrix not found at %s\n", "Erro: matriz de validação não encontrada em %s\n"), matrixPath)
		return 2
	}
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid validation matrix: %v\n", "Erro: matriz de validação inválida: %v\n"), err)
		return 2
	}
	if err := validateStructuredMatrixPaths(matrix); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid validation matrix path: %v\n", "Erro: path inválido na matriz de validação: %v\n"), err)
		return 2
	}
	if err := validateDeliveryMatrixContract(matrix); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid delivery validation contract: %v\n", "Erro: contrato de validação de delivery inválido: %v\n"), err)
		return 2
	}
	if matrixHasLegacyChecks(matrix) {
		fmt.Fprintln(stderr, cliText(locale, "Error: legacy shell 'command' checks are unsupported; migrate each check to program + args + env.", "Erro: checks shell legados em 'command' não são suportados; migre cada check para program + args + env."))
		return 2
	}
	if mode == "" {
		mode = matrix.Defaults.Mode
		if mode == "" {
			mode = "strict"
		}
	}
	modules, err := discoverValidationModules(root)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: discovering modules: %v\n", "Erro: descobrir módulos: %v\n"), err)
		return 1
	}
	if workspaceFilter != "" {
		resolved, ok := resolveWorkspaceModulePath(root, modules, workspaceFilter)
		if !ok {
			fmt.Fprintf(stderr, cliText(locale, "Error: no module's manifest name matches --workspace %q.\n", "Erro: nenhum module tem name de manifesto igual a --workspace %q.\n"), workspaceFilter)
			return 2
		}
		moduleFilter = resolved
	}
	knownModules := map[string]bool{}
	for _, module := range modules {
		knownModules[module.Rel] = true
	}
	for rel, override := range matrix.ModuleOverrides {
		clean := filepath.ToSlash(filepath.Clean(rel))
		if clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(rel) {
			fmt.Fprintf(stderr, cliText(locale, "Error: moduleOverrides contains a path outside the project: %s\n", "Erro: moduleOverrides contém path fora do projeto: %s\n"), rel)
			return 2
		}
		if knownModules[clean] {
			if override.Stack != "" {
				for idx := range modules {
					if modules[idx].Rel == clean {
						modules[idx].Stack = override.Stack
					}
				}
			}
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(clean))
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			stack := override.Stack
			if stack == "" {
				stack = "contract"
			}
			modules = append(modules, validationModule{Rel: clean, Abs: abs, Stack: stack})
			knownModules[clean] = true
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Rel < modules[j].Rel })
	var scope *scopeSelection
	if changedTo != "" && changedFrom == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --changed-to requires --changed-from.", "Erro: --changed-to exige --changed-from."))
		return 2
	}
	if changedFrom != "" {
		sel, err := computeChangedScope(root, changedFrom, changedTo, modules)
		if err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: changed-scope selection: %v\n", "Erro: seleção por escopo alterado: %v\n"), err)
			return 2
		}
		scope = &sel
		if explain {
			fmt.Fprintf(stdout, "[changed-scope] %s: %d changed file(s), %d/%d module(s) selected\n", sel.rangeLabel(), len(sel.Changed), len(sel.Selected), len(modules))
			for _, m := range modules {
				if reason, ok := sel.Selected[m.Rel]; ok {
					fmt.Fprintf(stdout, "  + %s: %s\n", m.Rel, reason)
				} else {
					fmt.Fprintf(stdout, "  - %s: not affected by %s\n", m.Rel, sel.rangeLabel())
				}
			}
		}
	}
	run := validationRunResult{
		SchemaVersion: validationResultSchema,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Mode:          mode,
		StackFilter:   stackFilter,
		ModuleFilter:  moduleFilter,
		Checks:        []checkResult{},
	}
	matrixDigest := sha256.Sum256(raw)
	run.MatrixSHA256 = "sha256:" + hex.EncodeToString(matrixDigest[:])
	if head, err := gitOutputBounded(root, 1024, "rev-parse", "HEAD"); err == nil {
		run.GitHead = strings.TrimSpace(string(head))
	}
	if graph, err := buildCurrentDeliveryGraph(root); err == nil {
		run.ProvenanceDigest = graph.ProvenanceDigest
		run.ScopeProvenance = map[string]string{}
		seenSpecs := map[string]bool{}
		for _, set := range graph.ChangeSets {
			if set.Spec != "" {
				seenSpecs[set.Spec] = true
			}
		}
		for spec := range seenSpecs {
			run.ScopeProvenance[spec] = posemodel.ScopedDeliveryProvenanceDigest(graph, spec)
		}
	}
	failures := 0
	optionalFailures := 0
	executed := 0
	r := render(stdout, stderr)
	r.SetVerbose(verbose)
	// The counter needs the whole plan up front, so the checks a module runs are
	// resolved by one helper called twice: counting here, executing below. A
	// second copy of the filter logic would drift from this one.
	planned := 0
	for _, module := range modules {
		_, checks, selected := moduleValidationChecks(module, matrix, stackFilter, moduleFilter)
		if !selected {
			continue
		}
		if scope != nil {
			if _, inScope := scope.Selected[module.Rel]; !inScope {
				continue
			}
		}
		for _, check := range checks {
			if check.Program != "" {
				planned++
			}
		}
	}
	steps := r.Steps(planned)
	var entries []*validationEntry
	for _, module := range modules {
		override := matrix.ModuleOverrides[module.Rel]
		stack := module.Stack
		if override.Stack != "" {
			stack = override.Stack
		}
		if stackFilter != "" && stack != stackFilter || moduleFilter != "" && module.Rel != moduleFilter {
			continue
		}
		checks := append([]validationCheck(nil), matrix.Stacks[stack].Checks...)
		if override.ReplaceDefaultChecks {
			checks = nil
		}
		checks = append(checks, override.Checks...)
		if scope != nil {
			if _, selected := scope.Selected[module.Rel]; !selected {
				// R3: every unselected check keeps a machine-readable reason.
				reason := "changed-scope: module not affected by " + scope.rangeLabel()
				for _, check := range checks {
					if check.Program == "" {
						continue
					}
					severity := check.Severity
					if severity == "" {
						severity = "required"
					}
					entries = append(entries, &validationEntry{check: check, severity: severity, skip: reason, silent: true, result: checkResult{
						ID: module.Rel + "/" + stack + "/" + check.Name, Module: module.Rel,
						Stack: stack, Name: check.Name, Program: check.Program,
						Args: check.Args, Env: redactedEnv(check.Env), Severity: severity, EvidenceClass: check.EvidenceClass,
					}})
				}
				continue
			}
		}
		moduleMode := mode
		if override.Mode != "" {
			moduleMode = override.Mode
		}
		entries = append(entries, &validationEntry{note: fmt.Sprintf("[module] %s (%s, mode=%s)", module.Rel, stack, moduleMode)})
		for _, check := range checks {
			if check.Program == "" {
				continue
			}
			severity := check.Severity
			if severity == "" {
				severity = "required"
			}
			entry := &validationEntry{check: check, moduleDir: module.Abs, severity: severity, done: make(chan struct{}), result: checkResult{
				ID: module.Rel + "/" + stack + "/" + check.Name, Module: module.Rel,
				Stack: stack, Name: check.Name, Program: check.Program,
				Args: check.Args, Env: redactedEnv(check.Env), Severity: severity, EvidenceClass: check.EvidenceClass,
			}}
			if check.Isolation == "required" {
				// The local CLI never weakens its boundary: isolated
				// execution is delegated to the Harness via --emit-plan.
				entry.skip = "requires isolated execution (harness) — include via --emit-plan"
				entry.isolated = true
			}
			entries = append(entries, entry)
		}
	}
	// Checks run up to jobs at a time; results are shown and counted here,
	// in matrix order, whatever order they finish in (spec
	// pose-validate-concurrent-steps R1). --verbose streams a check's output,
	// so it runs them one at a time.
	if jobs == 0 {
		jobs = defaultValidationJobs(effectiveValidationResources(os.ReadFile, runtime.NumCPU()))
	}
	if verbose {
		jobs = 1
	}
	run.Jobs = jobs
	runCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	// The first signal cancels the run; a second one acts as it always did,
	// even while the run shuts down (review round 1, B4).
	go func() {
		<-runCtx.Done()
		stopSignals()
	}()
	wallStart := time.Now()
	if jobs > 1 {
		go dispatchValidationEntries(runCtx, entries, jobs, matrix)
	}
	for _, entry := range entries {
		if entry.note != "" {
			steps.Note(entry.note)
			continue
		}
		if !entry.silent && !entry.isolated {
			if jobs > 1 {
				<-entry.done
			} else if runCtx.Err() != nil {
				entry.markCancelled()
			} else {
				entry.resolveWhen()
			}
		}
		if entry.skip != "" {
			entry.result.Outcome, entry.result.SkipReason = "skipped", entry.skip
			run.Counts.Skipped++
			if entry.isolated {
				entry.result.Isolation = "required"
				isolationChecks = append(isolationChecks, entry.result)
			}
			run.Checks = append(run.Checks, entry.result)
			// A skipped check used to print nothing at all, so a reader
			// could not tell it from one that never existed (R7).
			if !entry.silent {
				steps.Start(entry.check.Name, "").Resolve(cliout.StateSkipped, entry.skip)
			}
			continue
		}
		executed++
		run.Counts.Executed++
		step := steps.Start(entry.check.Name, strings.TrimSpace(entry.check.Program+" "+strings.Join(entry.check.Args, " ")))
		if jobs == 1 && entry.result.Outcome == "" {
			entry.execute(runCtx, matrix, verbose, stdout, stderr, steps.Release)
		}
		result := &entry.result
		severity := entry.severity
		switch result.Outcome {
		case "pass":
			run.Counts.Passed++
		case "fail":
			if severity == "required" {
				failures++
				run.Counts.Failed++
			} else {
				optionalFailures++
				run.Counts.OptionalFailed++
			}
		default:
			run.Counts.Errored++
			if severity == "required" {
				failures++
			} else {
				optionalFailures++
			}
		}
		note := ""
		if result.LimitState != "" {
			note = result.LimitState
		} else if result.ExitCode != nil {
			note = fmt.Sprintf("exit=%d", *result.ExitCode)
		}
		step.Resolve(cliout.StateFromKey(result.Outcome), note)
		if !verbose && (result.Outcome == "fail" || result.Outcome == "error") {
			step.Detail(tailLines(result.Output, 10), jsonOut)
		}
		run.Checks = append(run.Checks, *result)
	}
	run.WallSeconds = time.Since(wallStart).Seconds()
	// A cancelled run never reads as a success, whatever the severity of the
	// checks it stopped (review round 1, B5).
	cancelled := false
	for _, check := range run.Checks {
		cancelled = cancelled || check.LimitState == "cancelled"
	}
	for _, line := range validationTimingNotes(run) {
		steps.Note(line)
	}
	steps.Summary()
	if executed == 0 {
		fmt.Fprintln(stdout, "No modules/checks matched the matrix and filters.")
	}
	result := "pass"
	if cancelled {
		failures++
		result = "fail"
		r.Verdict(cliout.Verdict{State: cliout.StateFail, Text: cliText(locale, "validation cancelled", "validação cancelada")})
	} else if failures > 0 {
		result = "fail"
		r.Verdict(cliout.Verdict{State: cliout.StateFail, Text: cliText(locale, "required check failed", "check obrigatório falhou")})
	} else {
		if optionalFailures > 0 {
			result = "partial"
			r.Finding(cliout.Finding{State: cliout.StateWarning, Code: "optional-check",
				Message: fmt.Sprintf(cliText(locale, "%d optional check(s) failed", "%d check(s) opcional(is) falharam"), optionalFailures)})
		}
		r.Verdict(cliout.Verdict{State: cliout.StatePass})
	}
	run.Outcome = result
	usageFindings := make([]usageFinding, 0, run.Counts.Failed+run.Counts.OptionalFailed+run.Counts.Errored)
	for _, check := range run.Checks {
		if check.Outcome != "fail" && check.Outcome != "error" {
			continue
		}
		severity := check.Severity
		if check.Outcome == "error" {
			severity = "error"
		}
		usageFindings = append(usageFindings, usageFinding{ID: check.ID, Severity: severity})
	}
	noteUsageFindings(stdout, result, usageFindings, true)
	for name, writer := range map[string]struct {
		path  string
		write func(string, validationRunResult) error
	}{
		"json":  {jsonOut, writeValidationJSON},
		"junit": {junitOut, writeValidationJUnit},
		"sarif": {sarifOut, writeValidationSARIF},
	} {
		if writer.path == "" {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(writer.path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: creating %s result directory: %v\n", "Erro: criando diretório do resultado %s: %v\n"), name, err)
			return 1
		}
		if err := writer.write(target, run); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: writing %s result: %v\n", "Erro: escrevendo resultado %s: %v\n"), name, err)
			return 1
		}
		fmt.Fprintf(stdout, "%s: %s\n", name, target)
	}
	if planOut != "" {
		digest := sha256.Sum256(raw)
		head := ""
		if out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
			head = strings.TrimSpace(string(out))
		}
		plan := executionPlan{
			SchemaVersion: validationResultSchema,
			GeneratedAt:   run.GeneratedAt,
			ProjectID:     filepath.Base(root),
			Spec:          reportTask,
			GitHead:       head,
			MatrixSHA256:  hex.EncodeToString(digest[:]),
			Checks:        isolationChecks,
			Approval:      planApproval{Required: true},
		}
		if plan.Checks == nil {
			plan.Checks = []checkResult{}
		}
		target := filepath.Join(root, filepath.FromSlash(planOut))
		if err := writeExecutionPlan(target, plan); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: writing execution plan: %v\n", "Erro: escrevendo plano de execução: %v\n"), err)
			return 1
		}
		fmt.Fprintf(stdout, "plan: %s (%d isolated check(s); approval required before Harness execution)\n", target, len(plan.Checks))
	}
	if autoReport {
		if reportTask == "" {
			reportTask = "validate-native"
		}
		reportArgs := []string{"--task", reportTask, "--context", "auto-validate", "--validation-profile", mode}
		// The outcome travels inside the summary, so it stays derived rather
		// than being recorded as a manual assertion.
		_ = runReportWith(root, reportArgs, nil, validationSummaryOf(run), io.Discard, stderr)
	}
	if failures > 0 {
		return 1
	}
	return 0
}

// validationSummaryOf projects a finished run for `pose report`: the commands
// it executed and one line per check, with the outcome the run reached. It is
// the structured replacement for parsing the printed output (spec
// pose-cli-output-rendering-system R5).
// moduleValidationChecks resolves the stack and checks one module runs after
// overrides, and whether the stack/module filters select it at all. The counting
// pass and the executing pass share it so a filter can never mean two things
// (spec pose-cli-output-rendering-system R6).
func moduleValidationChecks(module validationModule, matrix validationMatrix, stackFilter, moduleFilter string) (string, []validationCheck, bool) {
	override := matrix.ModuleOverrides[module.Rel]
	stack := module.Stack
	if override.Stack != "" {
		stack = override.Stack
	}
	if stackFilter != "" && stack != stackFilter || moduleFilter != "" && module.Rel != moduleFilter {
		return stack, nil, false
	}
	checks := append([]validationCheck(nil), matrix.Stacks[stack].Checks...)
	if override.ReplaceDefaultChecks {
		checks = nil
	}
	return stack, append(checks, override.Checks...), true
}

// tailLines keeps the last n lines of a check's captured output: enough to see
// why it failed, with the full output one path away.
func tailLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func validationSummaryOf(run validationRunResult) *validationSummary {
	summary := &validationSummary{Outcome: run.Outcome}
	for _, check := range run.Checks {
		command := strings.TrimSpace(check.Program + " " + strings.Join(check.Args, " "))
		if check.Outcome != "skipped" && command != "" {
			summary.Commands = append(summary.Commands, command)
		}
		line := "- [" + check.Outcome + "] " + check.ID
		switch {
		case check.SkipReason != "":
			line += " (" + check.SkipReason + ")"
		case check.DurationSeconds > 0:
			line += fmt.Sprintf(" (%.1fs)", check.DurationSeconds)
		}
		if check.ExitCode != nil {
			line += fmt.Sprintf(" exit=%d", *check.ExitCode)
		}
		summary.Results = append(summary.Results, line)
	}
	// The report records the run's verdict too, built from the outcome rather
	// than copied from the printed line, so the recorded evidence keeps its
	// shape while the terminal is free to change.
	if run.Counts.OptionalFailed > 0 {
		summary.Results = append(summary.Results, fmt.Sprintf("Warning: %d optional check(s) failed.", run.Counts.OptionalFailed))
	}
	switch run.Outcome {
	case "fail":
		summary.Results = append(summary.Results, "Result: FAILURE (required check failed)")
	default:
		summary.Results = append(summary.Results, "Result: SUCCESS")
	}
	return summary
}

func sanitizeGoCheckArgs(ctx context.Context, dir, program string, args []string) []string {
	if program != "go" || len(args) == 0 {
		return args
	}

	hasWildcard := false
	for _, a := range args {
		if a == "./..." || strings.HasSuffix(a, "/...") {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		return args
	}

	hasNodeModules := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && d.Name() == "node_modules" {
			hasNodeModules = true
			return filepath.SkipDir
		}
		return nil
	})
	if !hasNodeModules {
		return args
	}

	// Cancelling the run stops this too (review round 1, B4).
	cmd := exec.CommandContext(ctx, "go", "list", "./...")
	cmd.Dir = dir
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcessGroup(cmd) }
	out, err := cmd.Output()
	if err != nil {
		return args
	}

	var filteredPkgs []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, "/node_modules/") {
			filteredPkgs = append(filteredPkgs, trimmed)
		}
	}

	if len(filteredPkgs) == 0 {
		return args
	}

	var newArgs []string
	for _, a := range args {
		if a == "./..." || strings.HasSuffix(a, "/...") {
			newArgs = append(newArgs, filteredPkgs...)
		} else {
			newArgs = append(newArgs, a)
		}
	}
	return newArgs
}
