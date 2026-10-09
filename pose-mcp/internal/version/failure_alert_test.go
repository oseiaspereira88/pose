// Red-signal contract (spec pose-red-signals-reach-a-person).
//
// Ten consecutive releases failed and CI stayed red on main for weeks because a
// failing workflow notified nobody. failure-alert.yml turns a red run on main,
// on a tag or on a schedule into an issue assigned to the owner. That only
// holds while it watches every workflow that runs there: a workflow added or
// renamed later would fail as silently as before. This test makes that drift a
// red build instead of a quiet gap.
package version_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// notAlerting names the workflows that run on main, a tag, a release or a
// schedule but are deliberately not watched, each with its reason.
var notAlerting = map[string]string{
	"Package channels":                   "dispatch-only optional round that gates nothing (pose-package-channels-deferred-native-verification)",
	"Repair Dependabot runtime evidence": "reacts to CI on Dependabot branches; a red CI it repairs is already alerted",
	"Failure alert":                      "the alert itself; watching it would loop",
}

var (
	workflowNameRe   = regexp.MustCompile(`(?m)^name:\s*(.+?)\s*$`)
	watchedListRe    = regexp.MustCompile(`(?m)^\s+workflows:\s*\[(.*)\]`)
	runsOutsidePRsRe = regexp.MustCompile(`(?m)^\s{2}(push|schedule|release|workflow_run):`)
)

func workflowName(t *testing.T, content, file string) string {
	t.Helper()
	m := workflowNameRe.FindStringSubmatch(content)
	if m == nil {
		t.Fatalf("%s: no top-level name", file)
	}
	return strings.Trim(m[1], `"'`)
}

// watchedWorkflows parses the workflow_run list of failure-alert.yml.
func watchedWorkflows(t *testing.T, content string) map[string]bool {
	t.Helper()
	m := watchedListRe.FindStringSubmatch(content)
	if m == nil {
		t.Fatal("failure-alert.yml: no workflow_run workflows list")
	}
	watched := map[string]bool{}
	for _, item := range strings.Split(m[1], ",") {
		if name := strings.Trim(strings.TrimSpace(item), `"'`); name != "" {
			watched[name] = true
		}
	}
	return watched
}

// unwatchedWorkflows returns the workflows that run outside pull requests and
// are neither watched nor exempt with a reason.
func unwatchedWorkflows(workflows map[string]string, watched map[string]bool) []string {
	missing := []string{}
	for name, content := range workflows {
		if !runsOutsidePRsRe.MatchString(content) || watched[name] || notAlerting[name] != "" {
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return missing
}

func TestFailureAlertWatchesEveryWorkflowOutsidePullRequests(t *testing.T) {
	files, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	workflows := map[string]string{}
	var alert string
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		name := workflowName(t, string(raw), filepath.Base(file))
		workflows[name] = string(raw)
		if filepath.Base(file) == "failure-alert.yml" {
			alert = string(raw)
		}
	}
	if alert == "" {
		t.Fatal("failure-alert.yml is missing: a red release or main reaches nobody")
	}
	watched := watchedWorkflows(t, alert)
	if missing := unwatchedWorkflows(workflows, watched); len(missing) > 0 {
		t.Errorf("workflows that run outside pull requests but are not watched by failure-alert.yml: %v", missing)
	}
	for name := range watched {
		if _, ok := workflows[name]; !ok {
			t.Errorf("failure-alert.yml watches %q, which no workflow is named", name)
		}
		// The shell allowlist must name exactly the watched set, or a watched
		// workflow's alert is refused as an unexpected name.
		if !strings.Contains(alert, `"`+name+`"|`) && !strings.Contains(alert, `|"`+name+`")`) {
			t.Errorf("failure-alert.yml watches %q but its name check does not accept it", name)
		}
	}
	for _, want := range []string{"permissions: { issues: write }", "--assignee \"$OWNER\"", "gh issue close"} {
		if !strings.Contains(alert, want) {
			t.Errorf("failure-alert.yml no longer contains %q", want)
		}
	}
}

// The check itself must fail on a workflow that nobody watches.
func TestFailureAlertContractRejectsAnUnwatchedWorkflow(t *testing.T) {
	workflows := map[string]string{
		"CI":      "name: CI\non:\n  push:\n    branches: [main]\n",
		"Nightly": "name: Nightly\non:\n  schedule:\n    - cron: \"0 0 * * *\"\n",
		"PR only": "name: PR only\non:\n  pull_request:\n",
	}
	got := unwatchedWorkflows(workflows, map[string]bool{"CI": true})
	if strings.Join(got, ",") != "Nightly" {
		t.Fatalf("unwatched = %v, want [Nightly]", got)
	}
}

// alertStep returns the shell of the alert step, as the runner executes it.
func alertStep(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../../.github/workflows/failure-alert.yml")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "run: |" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " ")) + 2
		body := []string{}
		for _, next := range lines[i+1:] {
			if strings.TrimSpace(next) != "" && len(next)-len(strings.TrimLeft(next, " ")) < indent {
				break
			}
			if len(next) >= indent {
				next = next[indent:]
			}
			body = append(body, next)
		}
		return strings.Join(body, "\n")
	}
	t.Fatal("failure-alert.yml has no run block")
	return ""
}

// runAlert executes the step with a recording gh whose main head is head and
// whose open alert issue, if any, is existing.
func runAlert(t *testing.T, workflow, conclusion, branch, sha, head, existing string) []string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "gh.log")
	gh := "#!/usr/bin/env bash\necho \"$*\" >> \"$GH_LOG\"\ncase \"$1 $2\" in\n  \"api repos/o/pose/commits/main\") echo \"$MAIN_HEAD\" ;;\n  \"issue list\") [ -n \"$EXISTING\" ] && echo \"$EXISTING\" ;;\nesac\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(gh), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "step.sh")
	if err := os.WriteFile(script, []byte(alertStep(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "GH_LOG="+log, "REPO=o/pose", "OWNER=o",
		"WORKFLOW="+workflow, "CONCLUSION="+conclusion, "BRANCH="+branch, "SHA="+sha, "MAIN_HEAD="+head, "EXISTING="+existing,
		"RUN_URL=https://github.com/o/pose/actions/runs/1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("alert step failed: %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(log)
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func ghCalled(calls []string, prefix string) bool {
	for _, call := range calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}
	return false
}

// A late success of an older commit must not clear the alert of a newer red
// run; a success at main's head does (spec pose-red-signal-clears-only-at-the-head).
func TestFailureAlertClearsOnlyAtMainsHead(t *testing.T) {
	old, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
	if calls := runAlert(t, "CI", "success", "main", old, head, "134"); ghCalled(calls, "issue close") {
		t.Fatalf("a success of an older commit cleared the alert: %v", calls)
	}
	if calls := runAlert(t, "CI", "success", "main", head, head, "134"); !ghCalled(calls, "issue close 134") {
		t.Fatalf("a success at main's head did not clear the alert: %v", calls)
	}
	if calls := runAlert(t, "Release", "success", "v7.1.0", old, head, "9"); !ghCalled(calls, "issue close 9") {
		t.Fatalf("a successful release on its tag did not clear the alert: %v", calls)
	}
	if calls := runAlert(t, "CI", "failure", "main", head, head, ""); !ghCalled(calls, "issue create") {
		t.Fatalf("a red run on main opened no alert: %v", calls)
	}
}
