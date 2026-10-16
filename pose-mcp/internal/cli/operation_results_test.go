package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	usagepkg "github.com/harne8/pose-mcp/internal/usage"
)

// Spec pose-cli-mcp-operation-results: one result per call, strict parsers,
// waiting distinct from failure, effects confirmed by the domain.

type closeApplyJSON struct {
	posemodel.OperationResult
	Data struct {
		Plan          *posemodel.CloseoutPlan `json:"plan"`
		BundleID      string                  `json:"bundle_id"`
		EvidenceRefs  []string                `json:"evidence_refs"`
		AttestCommand string                  `json:"attest_command"`
	} `json:"data"`
}

// runPoseJSON runs pose with stdout and stderr apart and decodes stdout as
// exactly one JSON document: a second document or trailing text fails.
func runPoseJSON(t *testing.T, root string, into any, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := 0
	inDir(t, root, func() { code = Main(args, &out, &errB) })
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	if err := decoder.Decode(into); err != nil {
		t.Fatalf("%v: stdout is not a JSON document: %v\n%s", args, err, out.String())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil || strings.TrimSpace(out.String()[decoder.InputOffset():]) != "" {
		t.Fatalf("%v: stdout holds more than one JSON document:\n%s", args, out.String())
	}
	return code, errB.String()
}

func effectNames(result posemodel.OperationResult) []string {
	names := []string{}
	for _, effect := range result.Effects {
		names = append(names, effect.Effect)
	}
	return names
}

func TestCloseApplyJSONReportsWaitingWithPendingAndNextAction(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}]
}`)
	var result closeApplyJSON
	code, stderr := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer", "--json")
	if code != 3 || result.Outcome != posemodel.OutcomeWaiting || result.Operation != "close.apply" || result.Step == "" {
		t.Fatalf("waiting was not reported as waiting: code=%d %+v %s", code, result.OperationResult, stderr)
	}
	if len(result.Pending) != 1 || result.Pending[0].ID != "security" || !strings.Contains(result.NextAction, "--resume") || result.Error != nil || !result.Resumable {
		t.Fatalf("waiting lost its pending item or next action: %+v", result.OperationResult)
	}
	if result.Data.AttestCommand == "" || result.Data.BundleID == "" || result.Data.Plan == nil {
		t.Fatalf("the reviewer handoff is missing: %+v", result.Data)
	}
	if !strings.Contains(strings.Join(effectNames(result.OperationResult), ","), "bundle-sealed") {
		t.Fatalf("the seal this call made is not a confirmed effect: %+v", result.Effects)
	}
	// The human output reads the same result.
	human := reviewBundleCLIFixture(t)
	writeCloseoutCLIFile(t, human, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}]
}`)
	code, out := runPose(t, human, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer")
	if code != 3 || !strings.Contains(out, "closeout_plan.pending.security=") || !strings.Contains(out, "closeout_plan.attest=pose review attest") || !strings.Contains(out, "closeout_plan.step.seal=done") {
		t.Fatalf("human rendering diverged: %d %s", code, out)
	}
}

func TestCloseApplyJSONSeparatesCompletionFromAlreadySatisfied(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	var result closeApplyJSON
	code, stderr := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer", "--json")
	if code != 0 || result.Outcome != posemodel.OutcomeCompleted || stderr != "" {
		t.Fatalf("apply did not complete: code=%d %+v %s", code, result.OperationResult, stderr)
	}
	effects := strings.Join(effectNames(result.OperationResult), ",")
	for _, want := range []string{"bundle-sealed", "attestation-recorded", "spec-closed"} {
		if !strings.Contains(effects, want) {
			t.Fatalf("effect %s is not confirmed: %s", want, effects)
		}
	}
	if len(result.StepsRun) == 0 || len(result.Unconfirmed) != 0 {
		t.Fatalf("steps run or unconfirmed effects are wrong: %+v", result.OperationResult)
	}
	// A done spec: nothing runs, and that is not a fresh verification (P03).
	var again closeApplyJSON
	code, _ = runPoseJSON(t, root, &again, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer", "--json")
	if code != 0 || again.Outcome != posemodel.OutcomeAlreadySatisfied || again.Operation != "close.resume" || len(again.StepsRun) != 0 || len(again.Effects) != 0 {
		t.Fatalf("a no-op was reported as a closeout: code=%d %+v", code, again.OperationResult)
	}
	if !strings.Contains(again.NextAction, "pose closeout-check spec:bundle") {
		t.Fatalf("the no-op does not point to the current verification: %q", again.NextAction)
	}
	code, out := runPose(t, root, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer")
	if code != 0 || !strings.Contains(out, "closeout_plan.result=already-satisfied spec:bundle") || strings.Contains(out, "result=closed") {
		t.Fatalf("human no-op still says closed: %d %s", code, out)
	}
	if countFiles(t, filepath.Join(root, ".pose/review-attestations")) != 1 {
		t.Fatal("the no-op wrote an attestation")
	}
}

func TestCloseApplyJSONFailureKeepsConfirmedEffectsAndOneDocument(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	var result closeApplyJSON
	code, stderr := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--json")
	if code != 1 || result.Outcome != posemodel.OutcomeFailed || result.Step != posemodel.CloseoutStepMechanical || result.Error == nil || result.Error.Cause != "reviewer-required" {
		t.Fatalf("the failure is not typed: code=%d %+v %s", code, result.OperationResult, stderr)
	}
	if !result.Resumable || !strings.Contains(result.NextAction, "--resume") {
		t.Fatalf("a resumable failure does not say how to continue: %+v", result.OperationResult)
	}
	// The seal ran before the failing step and is confirmed; the failing step
	// wrote nothing, so nothing is left unconfirmed.
	if effects := strings.Join(effectNames(result.OperationResult), ","); !strings.Contains(effects, "bundle-sealed") || len(result.Unconfirmed) != 0 {
		t.Fatalf("effects after a failure are wrong: %s %+v", effects, result.Unconfirmed)
	}
	var conflict closeApplyJSON
	code, _ = runPoseJSON(t, root, &conflict, "close", "spec:bundle", "--apply", "--digest", "sha256:old", "--reviewer", "agent:r", "--json")
	if code != 1 || conflict.Outcome != posemodel.OutcomeConflict || conflict.Error.Cause != "plan-digest-mismatch" || len(conflict.Effects) != 0 {
		t.Fatalf("a stale digest is not a conflict: %d %+v", code, conflict.OperationResult)
	}
}

func TestCloseAndStartRefuseAmbiguousModesWithoutWriting(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	for _, args := range [][]string{
		{"close", "spec:bundle", "--apply", "--plan", "--json"},
		{"close", "spec:bundle", "--plan", "--apply", "--json"},
		{"close", "spec:bundle", "--resume", "--plan"},
		{"close", "spec:bundle", "--plan", "--reviewer", "agent:r"},
		{"close", "spec:bundle", "--plan", "--digest", "sha256:x"},
		{"close", "spec:bundle", "--apply", "--digest", "--json"},
	} {
		if code, out := runPose(t, root, args...); code != 2 {
			t.Fatalf("%v was not refused: %d %s", args, code, out)
		}
	}
	if countFiles(t, filepath.Join(root, ".pose/review-bundles")) != 0 {
		t.Fatal("a refused close sealed a bundle")
	}
	start := startCLIFixture(t, true)
	out, _, _ := runCLI(t, start, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"start", "spec:work", "--apply", "--status", "--digest", plan.Digest},
		{"start", "spec:work", "--status", "--apply", "--digest", plan.Digest},
		{"start", "spec:work", "--status", "--digest", plan.Digest},
		{"start", "spec:work", "--cancel", "--json"},
		{"start", "spec:work", "--apply", "--cancel", "--digest", plan.Digest},
	} {
		if _, errOut, code := runCLI(t, start, args...); code != 2 {
			t.Fatalf("%v was not refused: %d %s", args, code, errOut)
		}
	}
	if _, err := os.Stat(filepath.Join(start, ".pose", "starts")); err == nil {
		t.Fatal("a refused start wrote a record")
	}
	// The compatible modes keep working, and a repeated apply with --json is
	// one document, not text.
	if _, errOut, code := runCLI(t, start, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json"); code != 0 {
		t.Fatalf("apply: %s", errOut)
	}
	var record posemodel.StartRecord
	if code, errOut := runPoseJSON(t, start, &record, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json"); code != 0 || record.Phase != "started" {
		t.Fatalf("repeated apply: %d %+v %s", code, record, errOut)
	}
	if out, _, code := runCLI(t, start, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 0 || !strings.Contains(out, "start.result=already-started") {
		t.Fatalf("repeated human apply does not say it was a no-op: %d %s", code, out)
	}
}

func TestSpecsRefusesInvalidFiltersAndKeepsAliases(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	for _, args := range [][]string{
		{"specs", "--sttaus", "done", "--json"},
		{"specs", "--since", "nonsense", "--json"},
		{"specs", "--since", "-3d", "--json"},
		{"specs", "--recent", "nonsense", "--json"},
		{"specs", "--recent", "0", "--json"},
		{"specs", "--status", "--json"},
		{"specs", "stray", "--json"},
	} {
		var out, errB bytes.Buffer
		code := 0
		inDir(t, root, func() { code = Main(args, &out, &errB) })
		if code != 2 || out.Len() != 0 {
			t.Fatalf("%v was not refused before listing: %d %s", args, code, out.String())
		}
	}
	for _, args := range [][]string{
		{"specs", "--component", "x", "--json"},
		{"specs", "--recent", "--json"},
		{"specs", "--recent", "3", "--json"},
		{"specs", "--since", "2026-01-01", "--json"},
		{"specs", "--since", "30d", "--json"},
	} {
		var specs []posemodel.Spec
		if code, errOut := runPoseJSON(t, root, &specs, args...); code != 0 {
			t.Fatalf("%v: documented form refused: %d %s", args, code, errOut)
		}
	}
}

func TestContinuousCloseoutStatusJSONSeparatesQueryFromReadiness(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdContinuousCloseout(root, []string{"status", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("inactive status: %d %s", code, errOut.String())
	}
	var inactive posemodel.OperationResult
	if err := json.Unmarshal(out.Bytes(), &inactive); err != nil || inactive.Outcome != posemodel.OutcomeCompleted || inactive.Data.(map[string]any)["active"] != false {
		t.Fatalf("inactive status JSON: %v %s", err, out.String())
	}
	if code := cmdContinuousCloseout(root, []string{"start", "spec:alpha", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("start: %s", errOut.String())
	}
	out.Reset()
	code := cmdContinuousCloseout(root, []string{"status", "--json"}, &out, &errOut)
	var active posemodel.OperationResult
	if err := json.Unmarshal(out.Bytes(), &active); err != nil {
		t.Fatalf("active status is not one JSON document: %v %s", err, out.String())
	}
	data := active.Data.(map[string]any)
	// The query completed; the scope is not terminal, which keeps exit 1.
	if code != 1 || active.Outcome != posemodel.OutcomeCompleted || data["active"] != true || data["terminal"] != false || active.Scope != "spec:alpha" || active.NextAction == "" {
		t.Fatalf("active status: code=%d %+v", code, active)
	}
	errOut.Reset()
	if code := cmdContinuousCloseout(root, []string{"status", "--verbose"}, &out, &errOut); code != 2 {
		t.Fatalf("an unsupported flag was ignored: %d", code)
	}
}

func TestUsageCountsAWaitingCloseAsWaitingNotFailed(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}]
}`)
	t.Setenv("POSE_USAGE_DIR", t.TempDir())
	t.Setenv("POSE_USAGE_DISABLED", "")
	if code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 3 {
		t.Fatalf("close did not wait: %d %s", code, out)
	}
	var report usagepkg.Report
	if code, errOut := runPoseJSON(t, root, &report, "usage", "--since-days", "0", "--tool", "close", "--json"); code != 0 {
		t.Fatalf("usage: %s", errOut)
	}
	if len(report.Rows) != 1 || report.Rows[0].Waiting != 1 || report.Rows[0].Failed != 0 {
		t.Fatalf("a waiting stop was not counted as waiting: %+v", report.Rows)
	}
}
