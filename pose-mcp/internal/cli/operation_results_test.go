package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
		{"close", "spec:bundle", "--json"},
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
		{"specs", "--since", "999999999999999999d", "--json"},
		{"specs", "--since", "", "--json"},
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

// publicationFailureFixture is an in-progress spec whose closeout evidence
// step runs validate into a results path that cannot be written (a
// directory sits there), from the independent review's F1 reproduction.
func publicationFailureFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".pose/project.json":                       `{"schema_version":1,"project_id":"proj.publication"}`,
		".pose/policy/review.json":                 `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-13","profiles":{"spec":"spec-closeout@2"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-13","component_aware":true,"component_aware_adopted_at":"2026-08-13"}`,
		".pose/review-profiles/spec-closeout.json": `{"schema_version":2,"id":"spec-closeout","version":2,"scope":"spec","criteria":[{"id":"correctness","description":"tested","evidence_classes":["unit"]}]}`,
		".pose/indexes/validation-matrix.json":     `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"test","program":"true","args":[],"severity":"required","evidenceClass":"unit"}]}}}`,
		".pose/policy/delivery.json":               `{"schema_version":1,"results_path":".pose/results/blocked.json"}`,
		"svc/go.mod":                               "module example.invalid/svc\n\ngo 1.22\n",
		".pose/specs/alpha.md":                     "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\ncomponents: \n---\n\n# Spec: alpha\n\n## 1. Intent\nFix one behavior.\n\n## 2. Requirements\n- R1: Preserve the behavior.\n\n## 6. Validation\n### Requirement trace\n- R1 [waived: synthetic fixture] explicit fixture exception\n",
	}
	for path, body := range files {
		writeCloseoutCLIFile(t, root, path, body)
	}
	if err := os.MkdirAll(filepath.Join(root, ".pose/results/blocked.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=Probe", "-c", "user.email=probe@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	return root
}

func TestEvidenceThatValidateCouldNotPublishIsNotAConfirmedEffect(t *testing.T) {
	root := publicationFailureFixture(t)
	var plan posemodel.CloseoutPlan
	if code, errOut := runPoseJSON(t, root, &plan, "close", "spec:alpha", "--plan", "--json"); code != 0 {
		t.Fatalf("plan: %d %s", code, errOut)
	}
	if next, ok := plan.NextCloseoutStep(); !ok || next.ID != posemodel.CloseoutStepEvidence {
		t.Fatalf("the fixture does not start at the evidence step: %+v", plan.Steps)
	}
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:alpha", "--apply", "--reviewer", "agent:reviewer", "--json")
	if code != 1 || result.Outcome != posemodel.OutcomeFailed || result.Step != posemodel.CloseoutStepEvidence || result.Error == nil || result.Error.Cause != "results-not-published" {
		t.Fatalf("an unpublished result was not the failure: code=%d %+v", code, result.OperationResult)
	}
	if strings.Contains(strings.Join(effectNames(result.OperationResult), ","), "results-written") {
		t.Fatalf("results-written was confirmed for a directory: %+v", result.Effects)
	}
	if len(result.Unconfirmed) == 0 || result.Unconfirmed[0].Step != posemodel.CloseoutStepEvidence {
		t.Fatalf("the evidence step is not left unconfirmed: %+v", result.Unconfirmed)
	}
}

func TestJSONRoutesReportOperationalFailuresAsOneDocument(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	start := startCLIFixture(t, true)
	for _, c := range []struct {
		root      string
		args      []string
		operation string
	}{
		{start, []string{"start", "spec:missing", "--json"}, "start.preview"},
		{start, []string{"start", "spec:missing", "--status", "--json"}, "start.status"},
		{start, []string{"start", "spec:missing", "--apply", "--digest", "bad", "--json"}, "start.apply"},
		{root, []string{"close", "spec:missing", "--plan", "--json"}, "close.plan"},
		{root, []string{"specs", "facts", "missing", "--json"}, "specs.facts"},
	} {
		var result posemodel.OperationResult
		code, _ := runPoseJSON(t, c.root, &result, c.args...)
		if code != 1 || result.Operation != c.operation || result.Outcome != posemodel.OutcomeFailed || result.Error == nil || result.Error.Cause == "" {
			t.Fatalf("%v: failure is not one typed document: %d %+v", c.args, code, result)
		}
	}
	// A specs directory that cannot be read fails the listing as JSON too.
	broken := t.TempDir()
	writeCloseoutCLIFile(t, broken, ".pose/specs", "not a directory")
	var result posemodel.OperationResult
	if code, _ := runPoseJSON(t, broken, &result, "specs", "--json"); code != 1 || result.Operation != "specs.list" || result.Error == nil {
		t.Fatalf("listing failure: %d %+v", code, result)
	}
}

func TestSpecFactsRefusesUnknownFlagsWhereverJSONIs(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	for _, args := range [][]string{
		{"specs", "facts", "bundle", "--bogus", "--json"},
		{"specs", "facts", "bundle", "--json", "--bogus"},
		{"specs", "facts", "bundle", "extra"},
		{"specs", "facts"},
	} {
		var out, errB bytes.Buffer
		code := 0
		inDir(t, root, func() { code = Main(args, &out, &errB) })
		if code != 2 || out.Len() != 0 {
			t.Fatalf("%v was not refused: %d %s", args, code, out.String())
		}
	}
	var before, after posemodel.SpecFacts
	codeA, _ := runPoseJSON(t, root, &before, "specs", "facts", "--json", "bundle")
	codeB, _ := runPoseJSON(t, root, &after, "specs", "facts", "bundle", "--json")
	if codeA != 0 || codeB != 0 {
		t.Fatalf("--json position changed the channel: %d %d", codeA, codeB)
	}
}

func TestARecentResultThatThisRunCouldNotRewriteIsNotConfirmed(t *testing.T) {
	root := publicationFailureFixture(t)
	target := filepath.Join(root, ".pose/results/blocked.json")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	// A valid, recent document from an earlier run that this run cannot
	// overwrite (second-round F1).
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"earlier":true}`+"\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o644) })
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:alpha", "--apply", "--reviewer", "agent:reviewer", "--json")
	if code != 1 || result.Error == nil || result.Error.Cause != "results-not-published" || result.Step != posemodel.CloseoutStepEvidence {
		t.Fatalf("an earlier document was taken as this run's results: code=%d %+v", code, result.OperationResult)
	}
	if strings.Contains(strings.Join(effectNames(result.OperationResult), ","), "results-written") {
		t.Fatalf("results-written was confirmed: %+v", result.Effects)
	}
}

func TestResultsThisRunPublishedAreConfirmed(t *testing.T) {
	root := publicationFailureFixture(t)
	if err := os.Remove(filepath.Join(root, ".pose/results/blocked.json")); err != nil {
		t.Fatal(err)
	}
	var result closeApplyJSON
	runPoseJSON(t, root, &result, "close", "spec:alpha", "--apply", "--reviewer", "agent:reviewer", "--json")
	if len(result.StepsRun) == 0 || result.StepsRun[0].ID != posemodel.CloseoutStepEvidence || result.StepsRun[0].Result != "done" || !strings.Contains(strings.Join(effectNames(result.OperationResult), ","), "results-written") {
		t.Fatalf("a published result was not confirmed: %+v", result.OperationResult)
	}
}

func TestAStartThatFailsAfterRecordingItsBaselineSaysSoAndResumes(t *testing.T) {
	root := startCLIFixture(t, true)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	// The baseline can be written; the spec transition cannot (second-round F5).
	specs := filepath.Join(root, ".pose", "specs")
	if err := os.Chmod(specs, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(specs, 0o755) })
	var result posemodel.OperationResult
	code, _ := runPoseJSON(t, root, &result, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json")
	if code != 1 || result.Outcome != posemodel.OutcomeFailed || !result.Resumable || !strings.Contains(result.NextAction, plan.Digest) {
		t.Fatalf("a partial start is not resumable: code=%d %+v", code, result)
	}
	if len(result.Effects) != 1 || result.Effects[0].Effect != "baseline-recorded" || len(result.Unconfirmed) != 1 || result.Unconfirmed[0].Step != "transition" {
		t.Fatalf("the published baseline or the unconfirmed transition is missing: %+v", result)
	}
	if err := os.Chmod(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	var record posemodel.StartRecord
	if code, errOut := runPoseJSON(t, root, &record, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json"); code != 0 || record.Phase != "started" {
		t.Fatalf("the advertised retry did not complete the start: %d %+v %s", code, record, errOut)
	}
}

func TestACheckThatPrintsTheWriteReceiptDoesNotConfirmIt(t *testing.T) {
	root := publicationFailureFixture(t)
	target := filepath.Join(root, ".pose/results/blocked.json")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"generated_at":"2026-10-16T00:00:00Z"}`+"\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o644) })
	// A failing check whose output is exactly the line validate prints after
	// its own write (third-round F1): text is not a receipt.
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"test","program":"sh","args":["-c","printf 'json: %s\\n' \"$1\"; exit 1","probe","`+target+`"],"severity":"required","evidenceClass":"unit"}]}}}`)
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=Probe", "-c", "user.email=probe@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "printing check"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	before, _ := os.ReadFile(target)
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:alpha", "--apply", "--reviewer", "agent:reviewer", "--json")
	after, _ := os.ReadFile(target)
	if !bytes.Equal(before, after) {
		t.Fatal("the fixture did not keep the earlier bytes")
	}
	if code != 1 || result.Error == nil || result.Error.Cause != "results-not-published" || strings.Contains(strings.Join(effectNames(result.OperationResult), ","), "results-written") {
		t.Fatalf("a printed receipt confirmed an unpublished result: code=%d %+v", code, result.OperationResult)
	}
}

func TestAStartThatFailsAfterMovingTheSpecResumesWithTheAdvertisedCommand(t *testing.T) {
	root := startCLIFixture(t, true)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	// The spec moves; the final record write fails (third-round F8).
	blocker := filepath.Join(root, ".pose", "starts", "work.json.tmp")
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	var result posemodel.OperationResult
	code, _ := runPoseJSON(t, root, &result, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json")
	spec, _ := os.ReadFile(filepath.Join(root, ".pose", "specs", "2026-09-27-work.md"))
	if code != 1 || !result.Resumable || !strings.Contains(string(spec), "status: in-progress") {
		t.Fatalf("the fixture did not stop after moving the spec: code=%d %+v\n%s", code, result, spec)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	var record posemodel.StartRecord
	if code, errOut := runPoseJSON(t, root, &record, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json"); code != 0 || record.Phase != "started" {
		t.Fatalf("the advertised command did not resume: %d %+v %s", code, record, errOut)
	}
	if code, out := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 0 || !strings.Contains(out, "start.result=already-started") {
		t.Fatalf("a finished start is not a no-op: %d %s", code, out)
	}
}

func TestARetriedStartDoesNotOutliveARequestThatNowRestrictsIt(t *testing.T) {
	root := startCLIFixture(t, true)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.start-retry")
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), `{"schema_version":1,"enabled":false,"atomic_start_version":1,"agency_readiness_version":1}`)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]}}`)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil || !plan.Ready {
		t.Fatalf("preview: %v %s", err, out)
	}
	// The baseline is recorded and the transition fails before publishing.
	specs := filepath.Join(root, ".pose", "specs")
	if err := os.Chmod(specs, 0o555); err != nil {
		t.Fatal(err)
	}
	if code, _ := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 1 {
		t.Fatal("the transition did not fail")
	}
	if err := os.Chmod(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	// A request now restricts start (fourth-round F9).
	open := []string{"action", "open", "--origin", "spec:work", "--kind", "decision", "--question", "Start now?",
		"--option", "go=Start", "--option", "wait=Hold", "--recommend", "wait",
		"--recipient-role", "maintainer", "--requested-by", "agent:impl", "--target", "requirement:R1",
		"--effect", "start:block", "--apply"}
	if code, out := runPose(t, root, open...); code != 0 {
		t.Fatalf("open: %s", out)
	}
	code, out := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest)
	spec, _ := os.ReadFile(filepath.Join(specs, "2026-09-27-work.md"))
	if code == 0 || !strings.Contains(string(spec), "status: draft") {
		t.Fatalf("the persisted baseline started a restricted spec: %d %s\n%s", code, out, spec)
	}
}

func TestAResumeThatFailsAgainKeepsItsReceiptForPeople(t *testing.T) {
	root := startCLIFixture(t, true)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(root, ".pose", "starts", "work.json.tmp")
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _ := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 1 {
		t.Fatal("the first apply did not fail")
	}
	// The obstacle stays: the resume fails again (fourth-round F10).
	code, human := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest)
	if code != 1 || !strings.Contains(human, "start.baseline-recorded=") || !strings.Contains(human, "start.unconfirmed.record=") || !strings.Contains(human, "start.next_action=") {
		t.Fatalf("the human resume failure lost its receipt: %d %s", code, human)
	}
}

// staleBaselineFixture records a baseline whose transition failed before
// publishing, then lets change move the repository on (fifth-round F11).
func staleBaselineFixture(t *testing.T, change func(root string)) (string, string) {
	t.Helper()
	root := startCLIFixture(t, true)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posemodel.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	specs := filepath.Join(root, ".pose", "specs")
	if err := os.Chmod(specs, 0o555); err != nil {
		t.Fatal(err)
	}
	if code, _ := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 1 {
		t.Fatal("the transition did not fail")
	}
	if err := os.Chmod(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	change(root)
	return root, plan.Digest
}

func TestAStaleStartBaselineIsNotAdvertisedAsResumable(t *testing.T) {
	for name, c := range map[string]struct {
		change func(root string)
		cause  string
	}{
		"head moved": {func(root string) {
			if out, err := exec.Command("git", "-C", root, "-c", "user.name=Probe", "-c", "user.email=probe@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "advance").CombinedOutput(); err != nil {
				t.Fatalf("%v %s", err, out)
			}
		}, "stale-start-baseline"},
		"spec changed": {func(root string) {
			path := filepath.Join(root, ".pose", "specs", "2026-09-27-work.md")
			raw, _ := os.ReadFile(path)
			mustWrite(t, path, strings.Replace(string(raw), "Record the baseline.", "Record the baseline twice.", 1))
		}, "compare-and-swap-conflict"},
	} {
		t.Run(name, func(t *testing.T) {
			root, digest := staleBaselineFixture(t, c.change)
			var result posemodel.OperationResult
			code, _ := runPoseJSON(t, root, &result, "start", "spec:work", "--apply", "--digest", digest, "--json")
			if code != 1 || result.Outcome != posemodel.OutcomeConflict || result.Error == nil || result.Error.Cause != c.cause || result.Resumable {
				t.Fatalf("a stale baseline was advertised as resumable: %d %+v", code, result)
			}
			if strings.Contains(result.NextAction, "--digest "+digest) || !strings.Contains(result.NextAction, "--cancel") {
				t.Fatalf("the next action repeats a digest that cannot apply: %q", result.NextAction)
			}
			_, human := runPose(t, root, "start", "spec:work", "--apply", "--digest", digest)
			if !strings.Contains(human, "start.next_action=") || !strings.Contains(human, "--cancel") || strings.Contains(human, "repeat `pose start") {
				t.Fatalf("the human receipt diverged: %s", human)
			}
			spec, _ := os.ReadFile(filepath.Join(root, ".pose", "specs", "2026-09-27-work.md"))
			if !strings.Contains(string(spec), "status: draft") {
				t.Fatalf("a stale baseline started the spec:\n%s", spec)
			}
		})
	}
}

func stepsOf(result posemodel.OperationResult) string {
	var steps []string
	for _, step := range result.StepsRun {
		steps = append(steps, step.ID+"="+step.Result)
	}
	return strings.Join(steps, ",")
}

// publishedStartFixture previews a start and blocks only the final record
// write, so the first apply publishes the baseline and the spec transition
// and then fails (sixth-round F12).
func publishedStartFixture(t *testing.T) (string, string) {
	t.Helper()
	root := startCLIFixture(t, true)
	var plan posemodel.StartPlan
	if code, _ := runPoseJSON(t, root, &plan, "start", "spec:work", "--json"); code != 0 {
		t.Fatal("preview")
	}
	if err := os.MkdirAll(filepath.Join(root, ".pose", "starts", "work.json.tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, plan.Digest
}

func TestAStartReceiptNamesThePublishedTransitionAndOnlyItsOwnSteps(t *testing.T) {
	const specRef = ".pose/specs/2026-09-27-work.md"
	root, digest := publishedStartFixture(t)
	var first posemodel.OperationResult
	code, _ := runPoseJSON(t, root, &first, "start", "spec:work", "--apply", "--digest", digest, "--json")
	spec, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(specRef)))
	if code != 1 || first.Step != "record" || !strings.Contains(string(spec), "status: in-progress") {
		t.Fatalf("bad fixture: %d %+v", code, first)
	}
	if got := strings.Join(effectNames(first), ","); got != "baseline-recorded,spec-in-progress" || first.Effects[1].Ref != specRef {
		t.Fatalf("the published transition is not a confirmed effect: %+v", first.Effects)
	}
	if got := stepsOf(first); got != "baseline=done,transition=done,record=failed" {
		t.Fatalf("the first apply's steps: %q", got)
	}
	// The retry finds both published and runs only the record.
	var retry posemodel.OperationResult
	if code, _ := runPoseJSON(t, root, &retry, "start", "spec:work", "--apply", "--digest", digest, "--json"); code != 1 {
		t.Fatal("the retry did not fail")
	}
	if got := stepsOf(retry); got != "record=failed" || strings.Join(effectNames(retry), ",") != "baseline-recorded,spec-in-progress" {
		t.Fatalf("the retry mixed inherited effects into its steps: %q %+v", got, retry.Effects)
	}
	code, human := runPose(t, root, "start", "spec:work", "--apply", "--digest", digest)
	if code != 1 || !strings.Contains(human, "start.step.record=failed") || strings.Contains(human, "start.step.baseline") || !strings.Contains(human, "start.spec-in-progress="+specRef) {
		t.Fatalf("the human retry receipt: %d %s", code, human)
	}

	root, digest = publishedStartFixture(t)
	code, human = runPose(t, root, "start", "spec:work", "--apply", "--digest", digest)
	if code != 1 || !strings.Contains(human, "start.step.baseline=done") || !strings.Contains(human, "start.step.transition=done") || !strings.Contains(human, "start.step.record=failed") || !strings.Contains(human, "start.spec-in-progress="+specRef) {
		t.Fatalf("the human first-apply receipt: %d %s", code, human)
	}
}

func TestAStartWhoseTransitionFailsClaimsNoSpecEffect(t *testing.T) {
	root := startCLIFixture(t, true)
	var plan posemodel.StartPlan
	if code, _ := runPoseJSON(t, root, &plan, "start", "spec:work", "--json"); code != 0 {
		t.Fatal("preview")
	}
	specs := filepath.Join(root, ".pose", "specs")
	if err := os.Chmod(specs, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(specs, 0o755)
	var result posemodel.OperationResult
	if code, _ := runPoseJSON(t, root, &result, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json"); code != 1 {
		t.Fatal("the transition did not fail")
	}
	if got := stepsOf(result); got != "baseline=done,transition=failed" || strings.Join(effectNames(result), ",") != "baseline-recorded" {
		t.Fatalf("a failed transition: %q %+v", got, result.Effects)
	}
}
