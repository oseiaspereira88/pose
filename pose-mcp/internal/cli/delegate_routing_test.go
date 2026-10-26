package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// resealAs seals spec:alpha again with a declared implementation, in the next
// second so the new bundle is the newer one.
func resealAs(t *testing.T, root, lead string) {
	t.Helper()
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	if code, out, errOut := runPoseSplit(t, root, "review", "bundle", "spec:alpha", "--seal", "--lead", lead); code != 0 || !strings.Contains(out, "review_bundle.implementation=") {
		t.Fatalf("seal --lead %s: %d %s %s", lead, code, out, errOut)
	}
}

// R3: --lead declares the identity a seal records, only with --seal, and only
// as <vendor>[/<model>].
func TestReviewBundleLeadIsDeclaredOnSealOnly(t *testing.T) {
	root := dispatchInstance(t)
	for _, args := range [][]string{
		{"review", "bundle", "spec:alpha", "--lead", "openai"},
		{"review", "bundle", "spec:alpha", "--seal", "--lead"},
		{"review", "bundle", "spec:alpha", "--seal", "--lead", "unknown"},
		{"review", "bundle", "spec:alpha", "--seal", "--lead", "a/b/c"},
	} {
		if code, _, _ := runPoseSplit(t, root, args...); code != 2 {
			t.Fatalf("%v: %d", args, code)
		}
	}
	resealAs(t, root, "anthropic/claude-opus-5-5")
	code, out, _ := runPoseSplit(t, root, "review", "bundle", "spec:alpha", "--seal")
	if code != 0 || !strings.Contains(out, "review_bundle.implementation=anthropic/claude-opus-5-5 (declared)") {
		t.Fatalf("a reseal without --lead did not keep the declared bundle: %d %s", code, out)
	}
}

const routedPolicy = `{"schema_version": 2, "agents": {
  "same": {"vendor": "anthropic", "model": "claude-opus-5-5", "cli": "sh", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo decision: approved"]}},
  "other": {"vendor": "openai", "model": "gpt-6.1-sol", "cli": "sh", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo decision: approved"]}}
}, "purposes": {"review": {"max_runs_per_bundle": 9}}}`

// R5, R8, R9: auto takes the vendor opposite to the sealed implementation and
// the run records the decision and its independence; a named agent of the
// implementation's vendor runs with a warning and records same-model.
func TestDelegateRoutesByTheSealedImplementation(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, routedPolicy)
	resealAs(t, root, "anthropic/claude-opus-5-5")
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--json")
	var run map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &run) != nil {
		t.Fatalf("auto: %d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Adapter != "other" || runs[0].IndependenceAchieved != posemodel.IndependenceDifferentVendor || runs[0].Routing == nil ||
		runs[0].Routing.Step != posemodel.DelegationStepPreferred || runs[0].Routing.ImplementationFrom != posemodel.ImplementationFromBundle || runs[0].Routing.Implementation.Vendor != "anthropic" {
		t.Fatalf("routed run: %+v", runs[0])
	}
	code, _, errOut = runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "same", "--reason", "same vendor on purpose")
	if code != 0 || !strings.Contains(errOut, "shares the implementation's vendor") {
		t.Fatalf("named same-vendor agent: %d %s", code, errOut)
	}
	runs, _ = posemodel.Store{Root: root}.ListReviewRuns("")
	last := runs[len(runs)-1]
	if last.Adapter != "same" || last.IndependenceAchieved != posemodel.IndependenceSameModel || last.Routing == nil || last.Routing.Step != posemodel.DelegationStepExplicit {
		t.Fatalf("named run: %+v", last)
	}
}

// R5, R6: with every candidate unavailable the run does not start, and the
// error document carries the routing decision.
func TestDelegateRefusesWhenNoAgentIsAvailable(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, strings.ReplaceAll(routedPolicy, `"cli": "sh"`, `"cli": "no-such-cli-for-pose-routing"`))
	resealAs(t, root, "anthropic/claude-opus-5-5")
	code, out, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--json")
	if code != 1 || !strings.Contains(out, delegateErrorSchema) || !strings.Contains(out, `"routing"`) || !strings.Contains(out, "not on the path") {
		t.Fatalf("no agent available: %d %s", code, out)
	}
	if runs, _ := (posemodel.Store{Root: root}).ListReviewRuns(""); len(runs) != 0 {
		t.Fatal("a run started without an available agent")
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "agents", "--json")
	if code != 0 || strings.Count(out, `"state": "missing"`) != 2 {
		t.Fatalf("agents: %s", out)
	}
}

// R6, R7, R14: an agent whose last run hit a limit is limited until its reset;
// when every candidate is limited within max_wait, the run waits for the
// first reset, without polling, and then starts.
func TestDelegateWaitsForTheFirstReset(t *testing.T) {
	root := dispatchInstance(t)
	marker := filepath.Join(t.TempDir(), "limited-once")
	resetFile := filepath.Join(t.TempDir(), "reset")
	writeDelegation(t, root, `{"schema_version": 2, "agents": {
  "other": {"vendor": "openai", "model": "gpt-6.1-sol", "cli": "sh",
    "commands": {"read": ["sh", "-c", "cat >/dev/null; if [ -e `+marker+` ]; then echo decision: approved; else touch `+marker+`; echo \"LIMIT reset: $(cat `+resetFile+`)\"; exit 1; fi"]},
    "limit": {"output_patterns": ["LIMIT"], "reset_pattern": "reset: (\\S+)"}}
}, "purposes": {"review": {"max_runs_per_bundle": 9}}}`)
	resealAs(t, root, "anthropic/claude-opus-5-5")
	reset := time.Now().Add(3 * time.Second).UTC().Truncate(time.Second)
	if err := os.WriteFile(resetFile, []byte(reset.Format(time.RFC3339)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha"); code != 1 || !strings.Contains(out, "interrupted") {
		t.Fatalf("first run: %d %s %s", code, out, errOut)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "agents"); code != 0 || !strings.Contains(out, "limited until "+reset.Format(time.RFC3339)) {
		t.Fatalf("agents: %s", out)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--reason", "after the reset", "--dry-run", "--json"); code != 0 || !strings.Contains(out, `"step": "wait"`) {
		t.Fatalf("dry run: %d %s", code, out)
	}
	code, _, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--reason", "after the reset")
	if code != 0 || !strings.Contains(errOut, "waiting until "+reset.Format(time.RFC3339)) || time.Now().Before(reset) {
		t.Fatalf("wait: %d %s", code, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	last := runs[len(runs)-1]
	if last.Status != posemodel.ReviewRunCompleted || last.Routing == nil || last.Routing.Step != posemodel.DelegationStepWait {
		t.Fatalf("waited run: %+v", last)
	}
}

// B05 and B06 of the first review: the deprecated alias records the agent it
// names and warns as delegate does; --lead takes the <vendor>[/<model>] the
// manuals document; the help no longer says auto is the only agent.
func TestAliasRecordsRoutingAndLeadTakesAModel(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, routedPolicy)
	resealAs(t, root, "anthropic/claude-opus-5-5")
	code, _, errOut := runPoseSplit(t, root, "review", "dispatch", "spec:alpha", "--via", "same", "--apply")
	if code != 0 || !strings.Contains(errOut, "shares the implementation's vendor") {
		t.Fatalf("alias: %d %s", code, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Routing == nil || runs[0].Routing.Step != posemodel.DelegationStepExplicit || runs[0].IndependenceAchieved != posemodel.IndependenceSameModel {
		t.Fatalf("alias run: %+v", runs)
	}
	code, out, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--lead", "Anthropic/Claude-Opus-5-5", "--reason", "again", "--dry-run", "--json")
	if code != 0 || !strings.Contains(out, `"lead": "anthropic/claude-opus-5-5"`) {
		t.Fatalf("--lead with a model: %d %s", code, out)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "agents", "--lead", "anthropic/claude-opus-5-5", "--json"); code != 0 {
		t.Fatalf("agents --lead with a model: %d", code)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--lead", "a/b/c", "--dry-run"); code != 2 {
		t.Fatalf("a malformed --lead was accepted: %d", code)
	}
	if _, out, _ := runPoseSplit(t, root, "help", "delegate"); strings.Contains(out, "the only agent configured for the purpose") {
		t.Fatal("the help still says auto is the only agent")
	}
}
