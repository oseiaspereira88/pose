package cli

// Structured validation results (spec pose-structured-validation-results):
// one canonical, versioned result model emitted as deterministic JSON with
// documented JUnit and SARIF projections. Text output stays authoritative
// for humans; machine formats are additive. Captured output is bounded and
// redacted; inherited environment values never enter the result.

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// validationResultSchema versions the JSON result contract.
const validationResultSchema = 1

// checkOutcome vocabulary keeps POSE semantics distinguishable (R3):
// "fail" = the tool ran and failed; "error" = infrastructure failure (the
// tool could not run); "skipped" = deterministic selection with a reason.
type checkResult struct {
	ID              string            `json:"id"` // <module>/<stack>/<name> — stable
	Module          string            `json:"module"`
	Stack           string            `json:"stack"`
	Name            string            `json:"name"`
	Program         string            `json:"program"`
	Args            []string          `json:"args,omitempty"`
	Env             map[string]string `json:"env,omitempty"` // configured only, secret values redacted
	Severity        string            `json:"severity"`      // required | optional
	Outcome         string            `json:"outcome"`       // pass | fail | error | skipped
	SkipReason      string            `json:"skip_reason,omitempty"`
	ExitCode        *int              `json:"exit_code,omitempty"`
	DurationSeconds float64           `json:"duration_seconds"`
	Output          string            `json:"output,omitempty"` // bounded tail, redacted
	// Runtime guardrails (spec pose-validation-runtime-guardrails), additive:
	LimitState string `json:"limit_state,omitempty"` // timeout | output-limit
	Isolation  string `json:"isolation,omitempty"`   // required = delegated to Harness
}

// outputLimiter cancels the check when total output exceeds the limit —
// an explicit guardrail state, distinct from the bounded capture tail.
type outputLimiter struct {
	limit    int
	written  int
	exceeded bool
	cancel   func()
}

func (l *outputLimiter) Write(p []byte) (int, error) {
	l.written += len(p)
	if !l.exceeded && l.written > l.limit {
		l.exceeded = true
		l.cancel()
	}
	return len(p), nil
}

type validationRunResult struct {
	SchemaVersion int    `json:"schema_version"`
	GeneratedAt   string `json:"generated_at"`
	Mode          string `json:"mode"`
	StackFilter   string `json:"stack_filter,omitempty"`
	ModuleFilter  string `json:"module_filter,omitempty"`
	// Outcome semantics (R3): fail = required check failed; partial =
	// tolerated (optional or infra-only) failures; pass = everything green.
	Outcome string        `json:"outcome"`
	Counts  runCounts     `json:"counts"`
	Checks  []checkResult `json:"checks"`
}

type runCounts struct {
	Executed       int `json:"executed"`
	Passed         int `json:"passed"`
	Failed         int `json:"failed"`
	OptionalFailed int `json:"optional_failed"`
	Errored        int `json:"errored"`
	Skipped        int `json:"skipped"`
}

// secretEnvKeyRE marks configured env keys whose values are redacted from
// both metadata and captured output.
var secretEnvKeyRE = regexp.MustCompile(`(?i)(token|secret|password|passwd|credential|apikey|api_key|private)`)

const redactedValue = "«redacted»"

// redactedEnv returns the check's configured env with secret values masked.
// Inherited process environment is never included by design.
func redactedEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if secretEnvKeyRE.MatchString(k) {
			out[k] = redactedValue
		} else {
			out[k] = v
		}
	}
	return out
}

// redactSecrets removes configured secret values from captured output.
func redactSecrets(text string, env map[string]string) string {
	for k, v := range env {
		if secretEnvKeyRE.MatchString(k) && v != "" {
			text = strings.ReplaceAll(text, v, redactedValue)
		}
	}
	return text
}

// tailBuffer keeps the last capacity bytes written (bounded capture).
type tailBuffer struct {
	capacity int
	data     []byte
	trimmed  bool
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if len(b.data) > b.capacity {
		b.data = b.data[len(b.data)-b.capacity:]
		b.trimmed = true
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	s := string(b.data)
	if b.trimmed {
		s = "…" + s
	}
	return s
}

// --- Harness execution plan -------------------------------------------------
// Remote plans bind project, spec, check plan, input digests and an approval
// slot (spec pose-validation-runtime-guardrails R3). The CLI only authors the
// envelope; approval identity is stamped by the control plane (Conductor)
// with an expiring execution identity before the Harness may run it. The
// local boundary never executes isolation-required checks.

type executionPlan struct {
	SchemaVersion int           `json:"schema_version"`
	GeneratedAt   string        `json:"generated_at"`
	ProjectID     string        `json:"project_id"`
	Spec          string        `json:"spec,omitempty"`
	GitHead       string        `json:"git_head,omitempty"`
	MatrixSHA256  string        `json:"matrix_sha256"`
	Checks        []checkResult `json:"checks"` // isolation-required only
	Approval      planApproval  `json:"approval"`
}

type planApproval struct {
	Required bool   `json:"required"`
	Identity string `json:"identity,omitempty"` // expiring execution identity (ADR-007), stamped externally
	Expires  string `json:"expires,omitempty"`
}

func writeExecutionPlan(path string, plan executionPlan) error {
	payload, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func writeValidationJSON(path string, run validationRunResult) error {
	payload, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

// --- JUnit projection -------------------------------------------------------
// Mapping: one <testsuite> per module, one <testcase> per check. POSE
// "error" maps to <error> (infrastructure), "fail" to <failure>, "skipped"
// to <skipped> with the deterministic reason. Optional-severity failures are
// still <failure> — JUnit has no tolerated level, so the POSE severity is
// preserved in the testcase classname suffix (documented lossy edge).

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitMessage `xml:"failure,omitempty"`
	Error     *junitMessage `xml:"error,omitempty"`
	Skipped   *junitMessage `xml:"skipped,omitempty"`
}

type junitMessage struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type junitSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitSuites struct {
	XMLName xml.Name     `xml:"testsuites"`
	Name    string       `xml:"name,attr"`
	Suites  []junitSuite `xml:"testsuite"`
}

func writeValidationJUnit(path string, run validationRunResult) error {
	byModule := map[string]*junitSuite{}
	var order []string
	for _, c := range run.Checks {
		suite, ok := byModule[c.Module]
		if !ok {
			suite = &junitSuite{Name: c.Module}
			byModule[c.Module] = suite
			order = append(order, c.Module)
		}
		tc := junitTestCase{
			Name:      c.Name,
			ClassName: "pose-validate." + c.Stack + "." + c.Severity,
			Time:      fmt.Sprintf("%.3f", c.DurationSeconds),
		}
		switch c.Outcome {
		case "fail":
			suite.Failures++
			tc.Failure = &junitMessage{Message: c.Program + " exited non-zero", Body: c.Output}
		case "error":
			suite.Errors++
			tc.Error = &junitMessage{Message: c.Program + " could not run", Body: c.Output}
		case "skipped":
			suite.Skipped++
			tc.Skipped = &junitMessage{Message: c.SkipReason}
		}
		suite.Tests++
		suite.Cases = append(suite.Cases, tc)
	}
	doc := junitSuites{Name: "pose-validate"}
	for _, m := range order {
		doc.Suites = append(doc.Suites, *byModule[m])
	}
	payload, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(xml.Header), append(payload, '\n')...), 0o644)
}

// --- SARIF projection -------------------------------------------------------
// Mapping (SARIF 2.1.0): one run, driver "pose-validate", one reportingRule
// per check ID; results only for fail/error (level: error for required,
// warning for optional, note for infra errors of optional checks). The full
// POSE outcome, severity and skip reason live in result properties —
// documented extensions so the projection is never silently lossy.

func writeValidationSARIF(path string, run validationRunResult) error {
	rules := []map[string]any{}
	results := []map[string]any{}
	for _, c := range run.Checks {
		rules = append(rules, map[string]any{
			"id":   c.ID,
			"name": c.Name,
			"properties": map[string]any{
				"pose/severity": c.Severity,
				"pose/stack":    c.Stack,
			},
		})
		if c.Outcome != "fail" && c.Outcome != "error" {
			continue
		}
		level := "error"
		if c.Severity == "optional" {
			level = "warning"
		}
		message := c.Program + " failed"
		if c.Outcome == "error" {
			message = c.Program + " could not run (infrastructure)"
		}
		results = append(results, map[string]any{
			"ruleId":  c.ID,
			"level":   level,
			"message": map[string]any{"text": message},
			"locations": []map[string]any{{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]any{"uri": c.Module},
				},
			}},
			"properties": map[string]any{
				"pose/outcome":  c.Outcome,
				"pose/severity": c.Severity,
			},
		})
	}
	doc := map[string]any{
		"$schema": "https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]any{{
			"tool": map[string]any{"driver": map[string]any{
				"name":           "pose-validate",
				"informationUri": "https://github.com/oseiaspereira88/pose",
				"version":        Version,
				"rules":          rules,
			}},
			"results": results,
			"properties": map[string]any{
				"pose/schema_version": run.SchemaVersion,
				"pose/mode":           run.Mode,
				"pose/outcome":        run.Outcome,
			},
		}},
	}
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}
