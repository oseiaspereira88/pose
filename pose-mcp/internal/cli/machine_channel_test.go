package cli

// The machine channel for the gates that had none (spec
// pose-cli-output-machine-channel). An agent must learn what a gate decided
// without parsing prose: under --json, stdout is one document and nothing
// else, and the document carries the findings the human output shows.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
)

const machineChannelGoodSpec = `---
slug: good
status: in-progress
created_at: 2026-08-01
---
# Spec: Good
## 1. Intent
Content.
## 2. Requirements
- R1: It works.
## 3. Technical Plan
Content.
## 4. Tasks
- [ ] Task.
## 6. Validation
Pending.
## 7. Final Report
Pending.
`

// Done, with no Tasks and no Validation: two required sections missing.
const machineChannelBrokenSpec = `---
slug: broken
status: done
created_at: 2026-08-01
completed_at: 2026-08-02
---
# Spec: Broken
## 1. Intent
Content.
## 2. Requirements
- R1: It works.
## 3. Technical Plan
Content.
## 7. Final Report
Done.
`

func machineChannelFixture(t *testing.T, broken bool) string {
	t.Helper()
	root := t.TempDir()
	specs := filepath.Join(root, ".pose", "specs")
	if err := os.MkdirAll(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specs, "2026-08-01-good.md"), []byte(machineChannelGoodSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	if broken {
		if err := os.WriteFile(filepath.Join(specs, "2026-08-01-broken.md"), []byte(machineChannelBrokenSpec), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func decodeRecord(t *testing.T, stdout string) cliout.Record {
	t.Helper()
	var record cliout.Record
	dec := json.NewDecoder(strings.NewReader(stdout))
	if err := dec.Decode(&record); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if dec.More() {
		t.Fatalf("stdout carries more than one document:\n%s", stdout)
	}
	return record
}

func TestLintSpecJSONIsOneDocumentWithTheFindings(t *testing.T) {
	root := machineChannelFixture(t, true)
	var out, errB bytes.Buffer
	code := cmdLintSpecInRoot(root, []string{"--all", "--json"}, &out, &errB)
	if code != 1 {
		t.Fatalf("a broken spec must fail the gate under --json too: exit=%d", code)
	}
	record := decodeRecord(t, out.String())
	if record.Command != "lint-spec" || record.Outcome != cliout.StateFail.Key() {
		t.Fatalf("command/outcome = %q/%q", record.Command, record.Outcome)
	}
	if record.Counts["specs_checked"] != 2 || record.Counts["specs_failed"] != 1 {
		t.Errorf("counts = %v, want 2 checked and 1 failed", record.Counts)
	}
	sections := 0
	for _, f := range record.Findings {
		if f.Path == "good" && f.Severity == cliout.StateError.Key() {
			t.Errorf("a passing spec produced an error finding: %+v", f)
		}
		if f.Path == "broken" && f.Code == "section" && f.Severity == cliout.StateError.Key() {
			sections++
		}
	}
	if sections != 2 {
		t.Errorf("want the two missing required sections of broken as findings, got %d: %+v", sections, record.Findings)
	}
	// --all keeps per-spec fields out of the one document: they would collide.
	if _, ok := record.Fields["spec.path"]; ok {
		t.Errorf("--all recorded a per-spec field: %v", record.Fields)
	}
}

func TestLintSpecJSONForOneSpecKeepsItsFields(t *testing.T) {
	root := machineChannelFixture(t, false)
	var out, errB bytes.Buffer
	if code := cmdLintSpecInRoot(root, []string{"good", "--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
	record := decodeRecord(t, out.String())
	if record.Outcome != cliout.StatePass.Key() {
		t.Errorf("outcome = %q", record.Outcome)
	}
	if record.Fields["spec.status"] != "in-progress" || record.Fields["spec.required.missing"] != "0" {
		t.Errorf("single-spec fields missing: %v", record.Fields)
	}
}

func TestLintSpecQuietPrintsTheVerdictAlone(t *testing.T) {
	root := machineChannelFixture(t, true)
	var out, errB bytes.Buffer
	if code := cmdLintSpecInRoot(root, []string{"--all", "--quiet"}, &out, &errB); code != 1 {
		t.Fatalf("exit=%d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "Resultado: FALHA") {
		t.Fatalf("--quiet must print the verdict line alone, got:\n%s", out.String())
	}
}

func TestLintSpecHumanOutputKeepsItsContractLines(t *testing.T) {
	root := machineChannelFixture(t, true)
	var out, errB bytes.Buffer
	cmdLintSpecInRoot(root, []string{"--all"}, &out, &errB)
	for _, want := range []string{"spec.path=", "lint.specs.checked=2", "lint.specs.failed=1", "Resultado: FALHA (1 spec(s)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("human output lost %q:\n%s", want, out.String())
		}
	}
}

func TestIndexJSONReportsWhatItIndexed(t *testing.T) {
	t.Setenv("POSE_PROJECT_ROOT", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	root := machineChannelFixture(t, false)
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	if code := cmdIndex(root, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
	record := decodeRecord(t, out.String())
	if record.Command != "index" || record.Outcome != cliout.StatePass.Key() {
		t.Fatalf("command/outcome = %q/%q", record.Command, record.Outcome)
	}
	if record.Counts["specs"] != 1 || record.Fields["indexes_dir"] == "" {
		t.Errorf("document does not say what was indexed: counts=%v fields=%v", record.Counts, record.Fields)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "indexes", "spec-graph.json")); err != nil {
		t.Errorf("--json must still write the indexes: %v", err)
	}

	out.Reset()
	errB.Reset()
	if code := cmdIndex(root, nil, &out, &errB); code != 0 || !strings.HasPrefix(out.String(), "POSE indexes updated at ") {
		t.Errorf("human line changed: exit=%d %q", code, out.String())
	}
	if code := cmdIndex(root, []string{"--bogus"}, &out, &errB); code != 2 {
		t.Errorf("an unknown flag must be a usage error, got %d", code)
	}
}

// Every gate that joined the channel answers --json with one document naming
// itself, with an outcome the exit code agrees with, and --quiet with at most
// its verdict line. Run over this repository, read-only.
func TestEveryGateOnTheMachineChannelPrintsOneDocument(t *testing.T) {
	root, err := repoRootForTest()
	if err != nil {
		t.Skip(err)
	}
	t.Setenv("POSE_PROJECT_ROOT", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Chdir(root)
	gates := map[string]func(args []string, stdout, stderr *bytes.Buffer) int{
		"skills-check":     func(a []string, o, e *bytes.Buffer) int { return cmdSkillsCheck(root, a, o, e) },
		"knowledge-check":  func(a []string, o, e *bytes.Buffer) int { return cmdKnowledgeCheck(root, a, o, e) },
		"recurrence-check": func(a []string, o, e *bytes.Buffer) int { return cmdRecurrenceCheck(root, append([]string{"--tolerant"}, a...), o, e) },
		"history-check":    func(a []string, o, e *bytes.Buffer) int { return cmdHistoryCheck(a, o, e) },
		"state":            func(a []string, o, e *bytes.Buffer) int { return cmdState(root, a, o, e) },
	}
	valid := map[string]bool{}
	for _, s := range []cliout.State{cliout.StatePass, cliout.StateWarning, cliout.StateFail, cliout.StateError} {
		valid[s.Key()] = true
	}
	for name, run := range gates {
		var out, errB bytes.Buffer
		code := run([]string{"--json"}, &out, &errB)
		record := decodeRecord(t, out.String())
		if record.Command != name {
			t.Errorf("%s: document names %q", name, record.Command)
		}
		if !valid[record.Outcome] || record.Verdict == "" || record.Findings == nil {
			t.Errorf("%s: incomplete document: %+v", name, record)
		}
		if failed := record.Outcome == cliout.StateFail.Key() || record.Outcome == cliout.StateError.Key(); failed != (code != 0) {
			t.Errorf("%s: outcome %q disagrees with exit %d", name, record.Outcome, code)
		}
		out.Reset()
		errB.Reset()
		run([]string{"--quiet"}, &out, &errB)
		if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(lines) > 1 {
			t.Errorf("%s --quiet printed more than its verdict:\n%s", name, out.String())
		}
	}
}
