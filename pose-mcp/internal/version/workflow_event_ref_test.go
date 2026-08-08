// Event-supplied ref contract (spec pose-workflow-event-ref-contract).
//
// A workflow triggered by workflow_run or pull_request_target runs in the base
// repository's context with its secrets and token. The ref the event carries is
// attacker-influenceable, so checking it out — or interpolating it into a shell
// script — executes untrusted input with trusted authority. Scorecard reports
// this as a dangerous workflow.
//
// The pattern was fixed in verify-release.yml and reintroduced in
// package-channels.yml the same day, by a sibling spec, with the correct form
// already in the repository. Knowledge did not prevent the recurrence; this
// does.
//
// Validated forms stay allowed: `if:` guards a job rather than executing
// anything, and `env:` is the point where the raw value is bound so it can be
// checked before use.
package version_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	eventExprRe    = regexp.MustCompile(`\$\{\{[^}]*github\.event\.[^}]*\}\}`)
	eventTriggerRe = regexp.MustCompile(`(?m)^\s{2}(workflow_run|pull_request_target):`)
	refKeyRe       = regexp.MustCompile(`^(\s*)ref:\s*(.*)$`)
	runKeyRe       = regexp.MustCompile(`^(\s*)(?:-\s+)?run:\s*(.*)$`)
)

// usesEventTrigger reports whether the workflow is triggered by an event whose
// payload an outside contributor can influence.
func usesEventTrigger(content string) bool {
	return eventTriggerRe.MatchString(content)
}

// eventRefFindings returns one message per place where an event-supplied value
// reaches a checkout ref or a shell script. Empty means the workflow is clean.
func eventRefFindings(content string) []string {
	findings := []string{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	// Set while consuming a block scalar under `run:`; holds its indentation.
	inRunBlock, runBlockIndent, runBlockLine := false, 0, 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		indent := len(text) - len(strings.TrimLeft(text, " "))

		if inRunBlock {
			if strings.TrimSpace(text) != "" && indent <= runBlockIndent {
				inRunBlock = false
			} else {
				if eventExprRe.MatchString(text) {
					findings = append(findings, fmt.Sprintf(
						"line %d: the run: block at line %d interpolates an event value into a shell script", line, runBlockLine))
				}
				continue
			}
		}

		if m := refKeyRe.FindStringSubmatch(text); m != nil && eventExprRe.MatchString(m[2]) {
			findings = append(findings, fmt.Sprintf(
				"line %d: checkout ref: is an event-supplied value", line))
			continue
		}

		if m := runKeyRe.FindStringSubmatch(text); m != nil {
			rest := strings.TrimSpace(m[2])
			if rest == "|" || rest == ">" || strings.HasPrefix(rest, "|") || strings.HasPrefix(rest, ">") {
				inRunBlock, runBlockIndent, runBlockLine = true, len(m[1]), line
				continue
			}
			if eventExprRe.MatchString(rest) {
				findings = append(findings, fmt.Sprintf(
					"line %d: run: interpolates an event value into a shell script", line))
			}
		}
	}
	return findings
}

func TestWorkflowEventRefContract(t *testing.T) {
	workflows, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(workflows) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	checked := 0
	for _, wf := range workflows {
		raw, err := os.ReadFile(wf)
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw)
		if !usesEventTrigger(content) {
			continue
		}
		checked++
		for _, finding := range eventRefFindings(content) {
			t.Errorf("%s: %s — resolve the ref against the release-tag pattern and pass it on through the environment, as verify-release.yml does", filepath.Base(wf), finding)
		}
	}
	if checked == 0 {
		t.Error("no workflow_run or pull_request_target workflow was examined — the trigger detection is broken, not the workflows")
	}
}

// The contract is only worth its failure path: these are the two exact forms
// that shipped, and the two that must keep being allowed.
func TestWorkflowEventRefFindingsRejectsAndAllows(t *testing.T) {
	rejected := map[string]string{
		"checkout of an event-supplied ref": `
jobs:
  build:
    steps:
      - uses: actions/checkout@abc
        with:
          ref: ${{ github.event.workflow_run.head_branch || inputs.tag }}
`,
		"event value inlined into a script": `
jobs:
  build:
    steps:
      - run: bash verify.sh "${{ github.event.workflow_run.head_branch }}"
`,
		"event value inside a run block": `
jobs:
  build:
    steps:
      - name: Generate
        run: |
          set -euo pipefail
          TAG="${{ github.event.release.tag_name }}"
          echo "$TAG"
`,
	}
	for name, content := range rejected {
		if len(eventRefFindings(content)) == 0 {
			t.Errorf("%s: accepted, but it is the pattern this contract exists to reject", name)
		}
	}

	allowed := map[string]string{
		"if: guard": `
jobs:
  build:
    if: github.event_name != 'workflow_run' || startsWith(github.event.workflow_run.head_branch, 'v')
    steps:
      - uses: actions/checkout@abc
`,
		"env: binding for validation": `
jobs:
  build:
    steps:
      - name: Resolve the release tag
        env:
          RAW_REF: ${{ github.event.workflow_run.head_branch || inputs.tag }}
        run: |
          case "$RAW_REF" in
            v[0-9]*.[0-9]*.[0-9]*) ;;
            *) exit 1 ;;
          esac
          echo "RELEASE_TAG=$RAW_REF" >> "$GITHUB_ENV"
`,
	}
	for name, content := range allowed {
		if findings := eventRefFindings(content); len(findings) > 0 {
			t.Errorf("%s: rejected (%v), but it is the validated form the fix depends on", name, findings)
		}
	}
}

func TestWorkflowEventTriggerDetection(t *testing.T) {
	if !usesEventTrigger("on:\n  workflow_run:\n    workflows: [\"Release\"]\n") {
		t.Error("workflow_run trigger not detected")
	}
	if !usesEventTrigger("on:\n  pull_request_target:\n    types: [opened]\n") {
		t.Error("pull_request_target trigger not detected")
	}
	if usesEventTrigger("on:\n  push:\n    tags: [\"v*\"]\n") {
		t.Error("a push-only workflow must not be treated as event-triggered")
	}
}
