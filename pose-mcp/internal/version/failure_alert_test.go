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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// notAlerting names the workflows that run on main, a tag, a release or a
// schedule but are deliberately not watched, each with its reason.
var notAlerting = map[string]string{
	"Package channels":                    "dispatch-only optional round that gates nothing (pose-package-channels-deferred-native-verification)",
	"Repair Dependabot runtime evidence": "reacts to CI on Dependabot branches; a red CI it repairs is already alerted",
	"Failure alert":                       "the alert itself; watching it would loop",
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
