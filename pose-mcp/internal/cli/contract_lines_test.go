package cli

// The printed contract (spec pose-cli-output-rendering-system R5).
//
// The CLI's output is read by machines: `pose report` parsed `pose validate`'s
// printed lines, agents read `name.field=value` diagnostics, and operators grep
// verdicts. Enumerating those lines is what makes the rest prose the renderer
// may restyle — and what makes a change to one of them visible in review
// instead of silent in a consumer.

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
)

type contractLine struct {
	ID        string   `json:"id"`
	Shape     string   `json:"shape"`
	Example   string   `json:"example"`
	Consumers []string `json:"consumers"`
}

func loadContractLines(t *testing.T) map[string]*regexp.Regexp {
	t.Helper()
	raw, err := os.ReadFile("testdata/contract-lines.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Lines []contractLine `json:"lines"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Lines) == 0 {
		t.Fatal("the contract cannot be empty: machines read this output")
	}
	shapes := map[string]*regexp.Regexp{}
	for _, line := range doc.Lines {
		re, err := regexp.Compile(line.Shape)
		if err != nil {
			t.Fatalf("%s: %v", line.ID, err)
		}
		if !re.MatchString(line.Example) {
			t.Errorf("%s: the recorded example does not match its own shape: %q", line.ID, line.Example)
		}
		if len(line.Consumers) == 0 {
			t.Errorf("%s: a contract line without a named consumer is not a contract", line.ID)
		}
		shapes[line.ID] = re
	}
	return shapes
}

// What the renderer emits must satisfy the contract it inherits, in both
// languages: these are the lines a restyle may not move.
func TestRendererKeepsTheContractLines(t *testing.T) {
	shapes := loadContractLines(t)
	for locale, id := range map[string]string{cliout.LocaleEN: "verdict.en", cliout.LocalePtBR: "verdict.pt-BR"} {
		out := &bytes.Buffer{}
		profile := cliout.Plain()
		profile.Locale = locale
		cliout.New(out, &bytes.Buffer{}, profile, profile).Verdict(cliout.Verdict{State: cliout.StatePass, Text: "detail."})
		line := strings.TrimRight(out.String(), "\n")
		if !shapes[id].MatchString(line) {
			t.Errorf("%s: %q does not match its pinned shape", id, line)
		}
	}
	out := &bytes.Buffer{}
	cliout.NewPlain(out, &bytes.Buffer{}).Field("artifact.claims", "14")
	if line := strings.TrimRight(out.String(), "\n"); !shapes["field"].MatchString(line) {
		t.Errorf("field: %q does not match its pinned shape", line)
	}
}

// A log written by an older engine is still read line-wise, which is why the
// legacy shapes stay in the contract even after `pose report` stopped needing
// them.
func TestLegacyValidationLogsStillParse(t *testing.T) {
	shapes := loadContractLines(t)
	log := "  -> go test -count=1 ./...\nResult: SUCCESS\n"
	for _, line := range strings.Split(strings.TrimRight(log, "\n"), "\n") {
		if !shapes["validate.command"].MatchString(line) && !shapes["verdict.en"].MatchString(line) {
			t.Fatalf("the legacy log shape drifted: %q", line)
		}
	}
	commands, results, outcome := parseValidationLines([]byte(log))
	if len(commands) != 1 || commands[0] != "go test -count=1 ./..." || outcome != "pass" || len(results) != 1 {
		t.Fatalf("legacy parse: commands=%v results=%v outcome=%q", commands, results, outcome)
	}
}

// The report records a run from the run itself. Nothing here parses printed
// output, so restyling the terminal cannot change recorded evidence.
func TestReportReadsTheRunNotItsProse(t *testing.T) {
	shapes := loadContractLines(t)
	exit := 1
	run := validationRunResult{
		Outcome: "partial",
		Counts:  runCounts{Executed: 3, Passed: 1, OptionalFailed: 1, Skipped: 1},
		Checks: []checkResult{
			{ID: "pose-mcp/go/test", Program: "go", Args: []string{"test", "./..."}, Outcome: "pass", DurationSeconds: 18.42},
			{ID: "pose-mcp/go/lint", Program: "gofmt", Args: []string{"-l", "."}, Outcome: "fail", DurationSeconds: 2.1, ExitCode: &exit},
			{ID: "pose-mcp/go/e2e", Program: "npm", Args: []string{"test"}, Outcome: "skipped", SkipReason: "requires isolated execution (harness)"},
		},
	}
	summary := validationSummaryOf(run)
	if summary.Outcome != "partial" {
		t.Fatalf("the outcome must come from the run: %q", summary.Outcome)
	}
	if len(summary.Commands) != 2 {
		t.Fatalf("a skipped check ran nothing, so it contributes no command: %v", summary.Commands)
	}
	wantResults := []string{
		"- [pass] pose-mcp/go/test (18.4s)",
		"- [fail] pose-mcp/go/lint (2.1s) exit=1",
		"- [skipped] pose-mcp/go/e2e (requires isolated execution (harness))",
		"Warning: 1 optional check(s) failed.",
		"Result: SUCCESS",
	}
	if strings.Join(summary.Results, "\n") != strings.Join(wantResults, "\n") {
		t.Fatalf("results:\n%s\nwant:\n%s", strings.Join(summary.Results, "\n"), strings.Join(wantResults, "\n"))
	}
	for _, line := range summary.Results[:3] {
		if !shapes["report.result"].MatchString(line) {
			t.Errorf("result line drifted from its pinned shape: %q", line)
		}
	}
}
