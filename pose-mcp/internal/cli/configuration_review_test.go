package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// Spec pose-update-configuration-review.

func updateOnce(t *testing.T, repo string) string {
	t.Helper()
	var out, errB bytes.Buffer
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update: %s %s", out.String(), errB.String())
	}
	return out.String()
}

func reviewedInstance(t *testing.T) (string, []reviewRequest) {
	t.Helper()
	repo := olderInstance(t)
	mustWrite(t, filepath.Join(repo, ".pose/policy/actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:ada"]},"identity_assurance":"declared"}`)
	out := updateOnce(t, repo)
	if !strings.Contains(out, "configuration review: .pose/specs/") || !strings.Contains(out, "nothing is adopted until they are answered") {
		t.Fatalf("update does not announce the review:\n%s", out)
	}
	requests, err := configurationReviewRequests(repo)
	if err != nil {
		t.Fatal(err)
	}
	return repo, requests
}

func requestFor(t *testing.T, requests []reviewRequest, capability string) reviewRequest {
	t.Helper()
	for _, req := range requests {
		if req.Capability == capability {
			return req
		}
	}
	t.Fatalf("no review request for %s", capability)
	return reviewRequest{}
}

func answerReview(t *testing.T, repo string, req reviewRequest, answer, reason string) {
	t.Helper()
	args := []string{"action", "resolve", req.View.Request.ID, "--actor", "human:ada", "--answer", answer, "--request-digest", req.View.Request.RequestDigest,
		"--expected-revision", strconv.Itoa(req.View.Revision), "--idempotency-key", "k-" + req.Capability, "--apply"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	if code, out := runPose(t, repo, args...); code != 0 {
		t.Fatalf("resolve: %s", out)
	}
}

func TestConfigurationReviewAsksOncePerCapabilityAndChangesNothing(t *testing.T) {
	repo, requests := reviewedInstance(t)
	pending, _ := posemodel.CapabilitiesToReview(repo, version.ReleaseBase())
	if len(requests) != len(pending) || len(requests) == 0 {
		t.Fatalf("%d requests for %d pending capabilities", len(requests), len(pending))
	}
	slug := configurationReviewSlug(version.ReleaseBase())
	matches, _ := filepath.Glob(filepath.Join(repo, ".pose/specs/*-"+slug+".md"))
	if len(matches) != 1 {
		t.Fatalf("review spec: %v", matches)
	}
	raw, _ := os.ReadFile(matches[0])
	for _, req := range requests {
		if !strings.Contains(string(raw), "`"+req.Capability+"`") || req.View.Request.Recipient.Role != "maintainer" || len(req.View.Request.Options) != 3 {
			t.Fatalf("request %+v is not the spec's question to the maintainer", req.View.Request)
		}
		if req.Capability == "contract-nodes" && req.View.Request.Recommend != "adopt" {
			t.Fatalf("a default capability is not recommended: %+v", req.View.Request)
		}
	}
	if states, _ := posemodel.CapabilityStates(repo); stateOf(states, "contract-nodes") != posemodel.CapabilityOff {
		t.Fatal("the update adopted a capability nobody answered")
	}
	if code, out := runPose(t, repo, "check", "--strict"); code != 0 {
		t.Fatalf("the review breaks the strict check: %s", out)
	}
	if code, out := runPose(t, repo, "lint-spec", slug); code != 0 || strings.Contains(out, "✖") {
		t.Fatalf("the review spec does not lint: %s", out)
	}
	// A second update asks nothing new.
	if out := updateOnce(t, repo); strings.Contains(out, "configuration review: .pose/specs/") {
		t.Fatalf("a second update asked again:\n%s", out)
	}
	again, _ := configurationReviewRequests(repo)
	if len(again) != len(requests) {
		t.Fatalf("requests after a second update: %d, want %d", len(again), len(requests))
	}
	// The maintainer sees them in Attention.
	if _, attention := runPose(t, repo, "state", "--attention", "--actor", "maintainer"); !strings.Contains(attention, requests[0].View.Request.ID) {
		t.Fatalf("Attention does not show the review to the maintainer:\n%s", attention)
	}
}

func TestConfigurationReviewAppliesOnlyAnsweredRequests(t *testing.T) {
	repo, requests := reviewedInstance(t)
	adopt := requestFor(t, requests, "contract-nodes")
	decline := requestFor(t, requests, "criterion-reuse")
	if code, out := runPose(t, repo, "adopt", "--request", adopt.View.Request.ID, "--apply"); code == 0 || !strings.Contains(out, "only an answered request") {
		t.Fatalf("an open request was applied: %d %s", code, out)
	}
	answerReview(t, repo, adopt, "adopt", "")
	answerReview(t, repo, decline, "decline", "bundles are always reviewed in full")
	if code, out := runPose(t, repo, "adopt", "--request", adopt.View.Request.ID); code != 0 || !strings.Contains(out, "adopt.apply=false") {
		t.Fatalf("preview: %s", out)
	}
	if states, _ := posemodel.CapabilityStates(repo); stateOf(states, "contract-nodes") != posemodel.CapabilityOff {
		t.Fatal("a preview applied the answer")
	}
	// setup names the answered request as a step.
	if step := stepOf(setupJSON(t, repo), "review:"+adopt.View.Request.ID); step.State != "todo" || step.Command != "pose adopt --request "+adopt.View.Request.ID+" --apply" {
		t.Fatalf("setup does not offer the answered request: %+v", step)
	}
	if code, out := runPose(t, repo, "adopt", "--request", adopt.View.Request.ID, "--apply"); code != 0 {
		t.Fatalf("apply adopt: %s", out)
	}
	if code, out := runPose(t, repo, "adopt", "--request", decline.View.Request.ID, "--apply"); code != 0 {
		t.Fatalf("apply decline: %s", out)
	}
	states, _ := posemodel.CapabilityStates(repo)
	if stateOf(states, "contract-nodes") != posemodel.CapabilityOn || stateOf(states, "criterion-reuse") != posemodel.CapabilityDeclined {
		t.Fatalf("answers not applied: %v", states)
	}
	decisions, _ := posemodel.ReadAdoptionDecisions(repo)
	if d := decisions.Decisions["criterion-reuse"]; d.Request != decline.View.Request.ID || d.Reason != "bundles are always reviewed in full" {
		t.Fatalf("the decision does not record its request and reason: %+v", d)
	}
	if code, out := runPose(t, repo, "adopt", "--request", adopt.View.Request.ID, "--apply"); code != 0 || !strings.Contains(out, "already applied") {
		t.Fatalf("re-applying: %d %s", code, out)
	}
	// A request that is not a configuration review is refused.
	code, out := runPose(t, repo, append(append([]string{}, openArgsFor("spec:"+configurationReviewSlug(version.ReleaseBase()))...), "--apply", "--json")...)
	if code != 0 {
		t.Fatalf("open: %s", out)
	}
	var other posemodel.ActionRequestView
	if err := json.Unmarshal([]byte(out), &other); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if code, out := runPose(t, repo, "adopt", "--request", other.Request.ID, "--apply"); code == 0 || !strings.Contains(out, "not a configuration-review request") {
		t.Fatalf("a foreign request was applied: %d %s", code, out)
	}
}

func TestSetupAnswersAndAppliesAReviewRequestAtATerminal(t *testing.T) {
	repo, requests := reviewedInstance(t)
	// The person at the terminal is human:ada, who holds the maintainer role.
	if code, out := runPose(t, repo, "identity", "add", "human:ada", "--key", testAuthorizedKey(t), "--apply"); code != 0 {
		t.Fatalf("identity: %s", out)
	}
	mustRun(t, repo, "git", "config", "user.email", "ada@example.com")
	plan := setupJSON(t, repo)
	if plan.You.Suggested != "human:ada" {
		t.Fatalf("suggested %q", plan.You.Suggested)
	}
	answers := []string{}
	for _, step := range plan.Steps {
		switch {
		case step.ID == "hooks.pre-commit":
			answers = append(answers, "n")
		case step.ID == "capability:contract-nodes":
			answers = append(answers, "a")
		case strings.HasPrefix(step.ID, "capability:"):
			answers = append(answers, "s")
		}
	}
	withSetupAnswers(t, strings.Join(answers, "\n")+"\n")
	code, setupOut := runPose(t, repo, "setup")
	if code != 0 {
		t.Fatalf("setup: %s", setupOut)
	}
	req := requestFor(t, requests, "contract-nodes")
	view, _ := posemodel.Store{Root: repo}.LoadActionRequest(req.View.Request.ID)
	if view.State != posemodel.ActionStateAnswered || view.AnsweredBy != "human:ada" || view.Answer != "adopt" {
		t.Fatalf("setup did not answer the request as the person: state=%s\n%s", view.State, setupOut)
	}
	if states, _ := posemodel.CapabilityStates(repo); stateOf(states, "contract-nodes") != posemodel.CapabilityOn {
		t.Fatal("setup did not apply the answer")
	}
}

func stateOf(states []posemodel.CapabilityState, id string) string {
	for _, state := range states {
		if state.ID == id {
			return state.State
		}
	}
	return ""
}

func openArgsFor(origin string) []string {
	return []string{"action", "open", "--origin", origin, "--kind", "approval", "--question", "Ship it?", "--recipient-role", "maintainer",
		"--requested-by", "agent:impl", "--target", "self", "--effect", "closeout:advisory"}
}

// testAuthorizedKey returns a fresh ssh-ed25519 public key line.
func testAuthorizedKey(t *testing.T) string {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	field := func(b []byte) []byte {
		out := make([]byte, 4+len(b))
		binary.BigEndian.PutUint32(out, uint32(len(b)))
		copy(out[4:], b)
		return out
	}
	blob := append(field([]byte("ssh-ed25519")), field(public)...)
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob) + " test"
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v %s", name, args, err, out)
	}
}

// Spec pose-adopt-request-keeps-the-reason: a decline answered without a
// reason is not recorded until one is given, and adopt --list shows it.
func TestDeclinedRequestNeedsAReason(t *testing.T) {
	repo, requests := reviewedInstance(t)
	decline := requestFor(t, requests, "criterion-reuse")
	answerReview(t, repo, decline, "decline", "")
	if code, out := runPose(t, repo, "adopt", "--request", decline.View.Request.ID, "--apply"); code == 0 || !strings.Contains(out, "--reason") {
		t.Fatalf("a decline without a reason was recorded: %d %s", code, out)
	}
	if states, _ := posemodel.CapabilityStates(repo); stateOf(states, "criterion-reuse") == posemodel.CapabilityDeclined {
		t.Fatal("a refused apply recorded the decline")
	}
	if code, out := runPose(t, repo, "adopt", "--request", decline.View.Request.ID, "--reason", "every bundle is reviewed in full here", "--apply"); code != 0 {
		t.Fatalf("apply with --reason: %s", out)
	}
	if _, out := runPose(t, repo, "adopt", "--list"); !strings.Contains(out, "every bundle is reviewed in full here") {
		t.Fatalf("adopt --list does not show the reason: %s", out)
	}
}
