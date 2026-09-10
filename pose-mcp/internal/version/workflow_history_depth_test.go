// Full-history contract for jobs that run the Go suite
// (spec pose-release-boundary-rehearsal).
//
// The cross-version test builds the previous release from its own tag. A
// default checkout is shallow and carries no tag objects, so the test cannot
// run — and a test that cannot run reads exactly like a test that passed.
//
// Adding `fetch-depth: 0` to the one job that was known to run the suite is the
// enumerate-by-hand shape this repository has been bitten by three times: the
// shellcheck file list, the docs-parity source list, the clean-tree assertions.
// It was wrong immediately here too — `validation-findings` runs `pose
// validate`, which runs the suite, in a different workflow, and it skipped.
//
// So the pairing is checked rather than remembered.
package version_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	jobKeyRe = regexp.MustCompile(`^  ([A-Za-z0-9_.-]+):\s*$`)
	// Commands that run, or transitively run, the whole Go test suite. `pose
	// validate` executes the module's registered checks, `go test ./...` among
	// them.
	fullSuiteRe  = regexp.MustCompile(`go\s+(-C\s+\S+\s+)?test\s+\./\.\.\.|cmd/pose\s+validate|pose\s+validate\b`)
	fetchDepthRe = regexp.MustCompile(`fetch-depth:\s*0\b`)
	checkoutRe   = regexp.MustCompile(`uses:\s*actions/checkout@`)
	// The variable the cross-version test reads to know the checkout carries the
	// release history. Declared by the workflow, not inherited from the provider.
	promiseRe = regexp.MustCompile(`POSE_RELEASE_HISTORY_AVAILABLE`)
)

// jobsRunningFullSuite returns, per workflow job, whether it runs the whole Go
// suite and whether its checkout asks for full history.
type jobDepth struct {
	workflow, job string
	runsFullSuite bool
	hasCheckout   bool
	fullHistory   bool
	promisesTags  bool
}

func scanJobDepths(t *testing.T, path string) []jobDepth {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var jobs []jobDepth
	var current *jobDepth
	inJobs := false
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	// Set while inside a checkout step, so `fetch-depth` is attributed to the
	// checkout rather than to any later step that happens to mention it.
	inCheckout := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "jobs:") {
			inJobs = true
			continue
		}
		if !inJobs {
			continue
		}
		if m := jobKeyRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				jobs = append(jobs, *current)
			}
			current = &jobDepth{workflow: filepath.Base(path), job: m[1]}
			inCheckout = false
			continue
		}
		if current == nil {
			continue
		}
		if checkoutRe.MatchString(line) {
			current.hasCheckout = true
			inCheckout = true
			continue
		}
		if inCheckout {
			if fetchDepthRe.MatchString(line) {
				current.fullHistory = true
			}
			// A new step ends the checkout step's own block.
			if strings.HasPrefix(strings.TrimLeft(line, " "), "- ") {
				inCheckout = false
			}
		}
		if fullSuiteRe.MatchString(line) {
			current.runsFullSuite = true
		}
		if promiseRe.MatchString(line) {
			current.promisesTags = true
		}
	}
	if current != nil {
		jobs = append(jobs, *current)
	}
	return jobs
}

func TestJobsRunningTheGoSuiteCheckOutFullHistory(t *testing.T) {
	workflows, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(workflows) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	examined := 0
	for _, wf := range workflows {
		for _, job := range scanJobDepths(t, wf) {
			if !job.runsFullSuite || !job.hasCheckout {
				continue
			}
			examined++
			if !job.fullHistory {
				t.Errorf("%s job %q runs the Go suite on a shallow checkout: the cross-version test needs tag objects to build the previous release, and without them it skips rather than fails — add `with: { fetch-depth: 0 }` to its actions/checkout",
					job.workflow, job.job)
			}
			// Full history is the promise; the variable is the job saying so
			// where the test can read it. Without the variable the test skips,
			// and a skip on this repository's own CI is the failure the guard
			// exists to prevent — which is how the first two versions of it went
			// wrong, once for every consumer and once for every provider.
			if !job.promisesTags {
				t.Errorf("%s job %q runs the Go suite with full history but does not declare POSE_RELEASE_HISTORY_AVAILABLE, so the cross-version test skips instead of failing when a tag is missing — add `env: { POSE_RELEASE_HISTORY_AVAILABLE: \"true\" }` to the step",
					job.workflow, job.job)
			}
		}
	}
	if examined == 0 {
		t.Error("no job was found running the Go suite — the command detection is broken, not the workflows")
	}
}
