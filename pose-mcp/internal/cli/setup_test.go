package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// Spec pose-setup-command.

func setupJSON(t *testing.T, repo string) setupPlan {
	t.Helper()
	var plan setupPlan
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"setup", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("setup exit=%d err=%s", code, errB.String())
		}
		if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
			t.Fatalf("%v\n%s", err, out.String())
		}
	})
	return plan
}

func stepOf(plan setupPlan, id string) setupStep {
	for _, step := range plan.Steps {
		if step.ID == id {
			return step
		}
	}
	return setupStep{}
}

func withSetupAnswers(t *testing.T, answers string) {
	t.Helper()
	previous := setupInput
	setupInput = func() (io.Reader, bool) { return strings.NewReader(answers), true }
	t.Cleanup(func() { setupInput = previous })
}

// olderInstance makes a fresh install look like one installed before the
// configuration review existed, with three capabilities off.
func olderInstance(t *testing.T) string {
	t.Helper()
	isolateHome(t)
	repo := freshInstall(t)
	_ = os.Remove(filepath.Join(repo, ".pose", "policy", "adoption-decisions.json"))
	for _, id := range []string{"contract-nodes", "criterion-reuse"} {
		if code, out := runPose(t, repo, "adopt", id, "--off", "--apply"); code != 0 {
			t.Fatalf("adopt %s --off: %s", id, out)
		}
	}
	_ = os.Remove(filepath.Join(repo, ".pose", "policy", "adoption-decisions.json"))
	return repo
}

// isolateHome keeps the machine's ~/.ssh keys out of the suggestions.
func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestSetupOnAFreshInstallNamesTheNextStep(t *testing.T) {
	isolateHome(t)
	repo := freshInstall(t)
	plan := setupJSON(t, repo)
	if plan.ReviewedVersion != version.ReleaseBase() || len(plan.New) != 0 {
		t.Fatalf("a fresh install has reviewed its own catalog: reviewed=%q new=%v", plan.ReviewedVersion, plan.New)
	}
	if stepOf(plan, "identity.project").State != "done" || stepOf(plan, "hooks.pre-commit").State != "todo" || stepOf(plan, "commit").State != "todo" {
		t.Fatalf("steps: %+v", plan.Steps)
	}
	maintainer := stepOf(plan, "identity.maintainer")
	if maintainer.State != "todo" || !strings.Contains(maintainer.Command, "pose identity add") || !strings.Contains(maintainer.Command, "--role maintainer") {
		t.Fatalf("an empty role map under agency readiness is not the first step: %+v", maintainer)
	}
	// The onboarding spec tracks the steps, so starting it comes first (spec
	// pose-onboarding-spec).
	if plan.Next == nil || plan.Next.ID != "onboarding" || plan.Next.Command != "pose start spec:pose-onboarding" || plan.Onboarding == "" {
		t.Fatalf("next = %+v", plan.Next)
	}
	if len(plan.InForce) == 0 || len(plan.Available) == 0 {
		t.Fatalf("capabilities in force and available are not listed: %+v %+v", plan.InForce, plan.Available)
	}
	// Without a terminal it only reports.
	withSetupAnswers(t, "")
	setupInput = func() (io.Reader, bool) { return strings.NewReader("y\n"), false }
	code, out := runPose(t, repo, "setup")
	if code != 0 || !strings.Contains(out, "setup.identity.maintainer.command=pose identity add human:<you> --key <file.pub> --role maintainer --apply") || !strings.Contains(out, "ssh-keygen -t ed25519-sk") {
		t.Fatalf("report: %d %s", code, out)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err == nil {
		t.Fatal("setup changed something without a terminal")
	}
}

func TestSetupFindsCapabilitiesAnOlderInstanceHasNotDecided(t *testing.T) {
	repo := olderInstance(t)
	plan := setupJSON(t, repo)
	ids := []string{}
	for _, state := range plan.New {
		ids = append(ids, state.ID)
	}
	joined := strings.Join(ids, ",")
	if !strings.Contains(joined, "contract-nodes") || !strings.Contains(joined, "criterion-reuse") || plan.ReviewedVersion != "" {
		t.Fatalf("new capabilities of a never-reviewed instance: %v (reviewed %q)", ids, plan.ReviewedVersion)
	}
	if step := stepOf(plan, "capability:contract-nodes"); step.State != "todo" || !strings.Contains(step.Summary, "recommended: adopt") {
		t.Fatalf("a default capability is not recommended: %+v", step)
	}
	f, ok := findDoctorFinding(runDoctorJSON(t, repo), "setup.capabilities")
	if !ok || f.Level != "next" || !strings.Contains(f.Hint, "pose setup") {
		t.Fatalf("doctor does not name pending decisions: %+v", f)
	}
	var out, errB bytes.Buffer
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update: %s", errB.String())
	}
	if !strings.Contains(out.String(), "capability decision(s) pending") || !strings.Contains(out.String(), "pose setup") || strings.Contains(out.String(), "Nothing to do") {
		t.Fatalf("update does not name what it brings:\n%s", out.String())
	}
}

func TestSetupAtATerminalPerformsOnlyConfirmedSteps(t *testing.T) {
	repo := olderInstance(t)
	plan := setupJSON(t, repo)
	// Scripted answers, in step order: skip the identity, install the hook,
	// then one answer (and a reason when needed) per capability to review.
	answers := []string{"-", "y"}
	for _, state := range plan.New {
		switch state.ID {
		case "contract-nodes":
			answers = append(answers, "a")
		case "criterion-reuse":
			answers = append(answers, "d", "bundles are always reviewed in full here")
		default:
			answers = append(answers, "l", "revisit after the pilot")
		}
	}
	withSetupAnswers(t, strings.Join(answers, "\n")+"\n")
	code, out := runPose(t, repo, "setup")
	if code != 0 {
		t.Fatalf("setup: %s", out)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("a confirmed hook installation did not happen:\n%s", out)
	}
	states, _ := posemodel.CapabilityStates(repo)
	byID := map[string]posemodel.CapabilityState{}
	for _, state := range states {
		byID[state.ID] = state
	}
	if byID["contract-nodes"].State != posemodel.CapabilityOn || byID["criterion-reuse"].State != posemodel.CapabilityDeclined {
		t.Fatalf("decisions not applied: contract-nodes=%s criterion-reuse=%s", byID["contract-nodes"].State, byID["criterion-reuse"].State)
	}
	decisions, _ := posemodel.ReadAdoptionDecisions(repo)
	if decisions.ReviewedVersion != version.ReleaseBase() || decisions.Decisions["criterion-reuse"].Version != version.ReleaseBase() {
		t.Fatalf("the review was not recorded as done for this engine: %+v", decisions)
	}
	if after := setupJSON(t, repo); len(after.New) != 0 {
		t.Fatalf("decided capabilities are still new: %+v", after.New)
	}

	// A refused prompt changes nothing.
	repo2 := olderInstance(t)
	withSetupAnswers(t, strings.Repeat("\n", 20))
	if code, out := runPose(t, repo2, "setup"); code != 0 {
		t.Fatalf("setup: %s", out)
	}
	if _, err := os.Lstat(filepath.Join(repo2, ".git", "hooks", "pre-commit")); err == nil {
		t.Fatal("an unanswered prompt installed the hook")
	}
	if pending, _ := posemodel.CapabilitiesToReview(repo2, version.ReleaseBase()); len(pending) == 0 {
		t.Fatal("unanswered prompts decided capabilities")
	}
}

func TestSetupDeferralIsAskedAgainUnderANewerEngine(t *testing.T) {
	repo := olderInstance(t)
	if err := posemodel.RecordAdoptionDecisionFor(repo, "contract-nodes", posemodel.AdoptionDecision{Decision: posemodel.AdoptionDeferred, Reason: "later", Date: "2026-01-01", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if pending, _ := posemodel.CapabilitiesToReview(repo, version.ReleaseBase()); !strings.Contains(stateIDs(pending), "contract-nodes") {
		t.Fatalf("a deferral from an older engine is not asked again: %v", stateIDs(pending))
	}
	if err := posemodel.RecordAdoptionDecisionFor(repo, "contract-nodes", posemodel.AdoptionDecision{Decision: posemodel.AdoptionDeferred, Reason: "later", Date: "2026-01-01", Version: version.ReleaseBase()}); err != nil {
		t.Fatal(err)
	}
	if pending, _ := posemodel.CapabilitiesToReview(repo, version.ReleaseBase()); strings.Contains(stateIDs(pending), "contract-nodes") {
		t.Fatal("a deferral taken under this engine is asked again at once")
	}
}

func stateIDs(states []posemodel.CapabilityState) string {
	ids := []string{}
	for _, state := range states {
		ids = append(ids, state.ID)
	}
	return strings.Join(ids, ",")
}
