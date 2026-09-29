package version

// v6.0.0 was published from a commit whose CI failed on two gates the release
// workflow did not run. The release now calls the CI workflow at the commit it
// publishes and waits for it (spec release-runs-the-ci-gates). That wiring is
// three lines across two files, and removing any one of them restores the
// subset silently, so it is pinned here.

import (
	"os"
	"strings"
	"testing"
)

// jobBlocks returns each job under the top-level `jobs:` key, by name, as the
// lines indented below it. Workflows here indent jobs by two spaces.
func jobBlocks(workflow string) map[string][]string {
	jobs := map[string][]string{}
	inJobs, current := false, ""
	for _, line := range strings.Split(workflow, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		switch {
		case indent == 0:
			inJobs, current = trimmed == "jobs:", ""
		case inJobs && indent == 2 && strings.HasSuffix(trimmed, ":"):
			current = strings.TrimSuffix(trimmed, ":")
			jobs[current] = nil
		case inJobs && current != "":
			jobs[current] = append(jobs[current], trimmed)
		}
	}
	return jobs
}

// triggers returns the event names under the top-level `on:` key.
func triggers(workflow string) map[string]bool {
	out := map[string]bool{}
	inOn := false
	for _, line := range strings.Split(workflow, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			inOn = trimmed == "on:"
			continue
		}
		if inOn && indent == 2 {
			out[strings.TrimSuffix(strings.SplitN(trimmed, ":", 2)[0], ":")] = true
		}
	}
	return out
}

// releaseNeedsCIFindings reports what is missing for the release workflow to
// run the CI workflow and wait for it.
func releaseNeedsCIFindings(release, ci string) []string {
	findings := []string{}
	if !triggers(ci)["workflow_call"] {
		findings = append(findings, "ci.yml is not callable (no workflow_call trigger)")
	}
	jobs := jobBlocks(release)
	calls := false
	for _, line := range jobs["ci"] {
		if line == "uses: ./.github/workflows/ci.yml" {
			calls = true
		}
	}
	if !calls {
		findings = append(findings, "release.yml has no `ci` job calling ./.github/workflows/ci.yml")
	}
	waits := false
	for _, line := range jobs["release"] {
		if line == "needs: ci" || (strings.HasPrefix(line, "needs: [") && strings.Contains(line, "ci")) {
			waits = true
		}
	}
	if !waits {
		findings = append(findings, "release.yml's `release` job does not need `ci`")
	}
	return findings
}

func TestReleaseWorkflowWaitsForCI(t *testing.T) {
	release, err := os.ReadFile("../../../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	ci, err := os.ReadFile("../../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range releaseNeedsCIFindings(string(release), string(ci)) {
		t.Error(finding)
	}
}

func TestReleaseNeedsCIFindingsRejectEachMissingLink(t *testing.T) {
	ci := "name: CI\non:\n  push:\n  workflow_call:\njobs:\n  test:\n    runs-on: x\n"
	release := "name: Release\njobs:\n  ci:\n    uses: ./.github/workflows/ci.yml\n  release:\n    needs: ci\n    runs-on: x\n"
	if got := releaseNeedsCIFindings(release, ci); len(got) != 0 {
		t.Fatalf("wired workflows reported %v", got)
	}
	for name, tc := range map[string]struct{ release, ci string }{
		"ci not callable":       {release, strings.Replace(ci, "  workflow_call:\n", "", 1)},
		"no ci job":             {strings.Replace(release, "  ci:\n    uses: ./.github/workflows/ci.yml\n", "", 1), ci},
		"release does not wait": {strings.Replace(release, "    needs: ci\n", "", 1), ci},
		"comment is not wiring": {strings.Replace(release, "    needs: ci\n", "    # needs: ci\n", 1), ci},
	} {
		if got := releaseNeedsCIFindings(tc.release, tc.ci); len(got) != 1 {
			t.Errorf("%s: want one finding, got %v", name, got)
		}
	}
}
