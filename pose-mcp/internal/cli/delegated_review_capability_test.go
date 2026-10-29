package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func delegationInitInput(t *testing.T, answer string, interactive, found bool) string {
	t.Helper()
	oldInput, oldLookup := setupInput, delegationLookupCLI
	t.Cleanup(func() { setupInput = oldInput; delegationLookupCLI = oldLookup })
	setupInput = func() (io.Reader, bool) { return strings.NewReader(answer), interactive }
	delegationLookupCLI = func(name string) (string, error) {
		if found && name == "codex" {
			return "/never-executed/codex", nil
		}
		return "", os.ErrNotExist
	}
	home := filepath.Join(t.TempDir(), "issuer-keys")
	t.Setenv("POSE_ISSUER_HOME", home)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	return home
}

func TestDelegationInitAcceptedOfferCreatesOnlyExecutorTrust(t *testing.T) {
	home := delegationInitInput(t, "\n", true, true)
	root := newGitRepo(t)
	var out, errOut bytes.Buffer
	if code := cmdInitCommand(root, []string{"--skip-mcp", "--project-id", "proj.init"}, &out, &errOut); code != 0 {
		t.Fatal(code, out.String(), errOut.String())
	}
	s := posemodel.Store{Root: root}
	p, err := s.GetReviewPolicy()
	if err != nil || !p.DelegatedReview || p.DelegatedReviewAdoptedAt != time.Now().UTC().Format(time.DateOnly) {
		t.Fatal(p, err)
	}
	policy, _, found, err := s.LoadDelegationPolicy()
	if err != nil || !found || len(policy.Agents) != 1 || policy.ConclusionsIssuer() == "" {
		t.Fatal(policy, err)
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateDelegatedReviewActivation(policy, docs); err != nil {
		t.Fatal(err)
	}
	if len(p.HumanAuthorityIssuers) != 0 {
		t.Fatal("init minted human authority")
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	config, _ := os.ReadFile(filepath.Join(root, ".pose/policy/delegation.json"))
	setupInput = func() (io.Reader, bool) { t.Fatal("existing init asked again"); return nil, false }
	if code := cmdInitCommand(root, nil, &out, &errOut); code != 0 {
		t.Fatal(code, errOut.String())
	}
	after, _ := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	again, _ := os.ReadFile(filepath.Join(root, ".pose/policy/delegation.json"))
	if !bytes.Equal(before, after) || !bytes.Equal(config, again) {
		t.Fatal("existing init rewrote adoption")
	}
}

func TestDelegationInitRefusalAndHeadlessCreateNoSigningArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name, answer       string
		interactive, found bool
	}{{"refused", "n\n", true, true}, {"headless", "", false, true}, {"no CLI", "\n", true, false}, {"EOF", "", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			home := delegationInitInput(t, tc.answer, tc.interactive, tc.found)
			root := newGitRepo(t)
			var out, errOut bytes.Buffer
			if code := cmdInitCommand(root, []string{"--skip-mcp"}, &out, &errOut); code != 0 {
				t.Fatal(code, errOut.String())
			}
			p, err := (posemodel.Store{Root: root}).GetReviewPolicy()
			if err != nil || p.DelegatedReview || !p.DelegatedReviewConfigured {
				t.Fatal(p, err)
			}
			if _, err := os.Stat(filepath.Join(root, ".pose/policy/delegation.json")); !os.IsNotExist(err) {
				t.Fatal("refusal wrote agent policy", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatal("refusal created an issuer", err)
			}
			if len(p.TrustedAttestationIssuers) != 0 {
				t.Fatal("refusal pinned an issuer")
			}
		})
	}
}

func TestDelegationInitDoesNotBootstrapOverExistingGovernance(t *testing.T) {
	home := delegationInitInput(t, "yes\n", true, true)
	root, _ := installedInstance(t)
	if err := os.Remove(filepath.Join(root, ".pose/schema-version")); err != nil {
		t.Fatal(err)
	}
	setupInput = func() (io.Reader, bool) { t.Fatal("missing stamp was treated as new governance"); return nil, false }
	var out, errOut bytes.Buffer
	if code := cmdInitCommand(root, []string{"--skip-mcp"}, &out, &errOut); code != 0 {
		t.Fatal(code, out.String(), errOut.String())
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("existing governance acquired a new issuer", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/policy/delegation.json")); !os.IsNotExist(err) {
		t.Fatal("existing governance acquired agent policy", err)
	}
}

func TestDelegationInitPolicyPublicationFailureRetainsRecoverableState(t *testing.T) {
	delegationInitInput(t, "yes\n", true, true)
	root, _ := installedInstance(t)
	var out, errOut bytes.Buffer
	old := posemodel.PublishFault
	t.Cleanup(func() { posemodel.PublishFault = old })
	posemodel.PublishFault = func(path string) error {
		if path == filepath.Join(root, ".pose/policy/review.json") {
			return errors.New("injected review publication failure")
		}
		return nil
	}
	if code := initDelegation(root, &out, &errOut); code == 0 {
		t.Fatal("failed activation reported success")
	}
	if !strings.Contains(errOut.String(), "activation failed") {
		t.Fatal(errOut.String())
	}
	p, err := (posemodel.Store{Root: root}).GetReviewPolicy()
	if err != nil || p.DelegatedReview {
		t.Fatal("failed activation enabled delegation", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/policy/delegation.json")); err != nil {
		t.Fatal("partial policy was deleted", err)
	}
	keys, err := posemodel.ListIssuerKeys(root)
	if err != nil || len(keys) != 1 {
		t.Fatal("partial issuer was lost", keys, err)
	}
}

func TestDelegationInitRejectsLinkedGovernance(t *testing.T) {
	home := delegationInitInput(t, "yes\n", true, true)
	root := newGitRepo(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".pose")); err != nil {
		t.Skip(err)
	}
	var out, errOut bytes.Buffer
	if code := cmdInitCommand(root, nil, &out, &errOut); code == 0 {
		t.Fatal("linked governance accepted")
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("unsafe init created key", err)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("unsafe init wrote outside project")
	}
}

func TestDelegatedReviewCapabilityAdoptPreviewAndOff(t *testing.T) {
	delegationInitInput(t, "yes\n", true, true)
	root, _ := installedInstance(t)
	var out, errOut bytes.Buffer
	if code := initDelegation(root, &out, &errOut); code != 0 {
		t.Fatal(code, errOut.String())
	}
	if code, got := runPose(t, root, "adopt", "delegated-review", "--off", "--apply"); code != 0 {
		t.Fatal(code, got)
	}
	before, _ := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	if code, got := runPose(t, root, "adopt", "delegated-review"); code != 0 || !strings.Contains(got, "adopt.apply=false") {
		t.Fatal(code, got)
	}
	after, _ := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("preview activated delegation")
	}
	if code, got := runPose(t, root, "adopt", "delegated-review", "--apply"); code != 0 {
		t.Fatal(code, got)
	}
}

func TestDelegatedReviewCapabilityUpdateOnlySuggests(t *testing.T) {
	root, _ := installedInstance(t)
	var out bytes.Buffer
	suggestDelegationUpdate(posemodel.NativeFiles(), root, &out)
	if strings.Count(out.String(), "pose adopt delegated-review") != 1 {
		t.Fatal(out.String())
	}
	p, err := (posemodel.Store{Root: root}).GetReviewPolicy()
	if err != nil || p.DelegatedReview {
		t.Fatal("suggestion enabled capability", err)
	}
}

func TestDelegatedReviewCapabilityUnavailableAgentEscalatesWithoutSpawn(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t, func(root, _ string) {
		writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
		var p posemodel.ReviewersPolicy
		raw, _ := os.ReadFile(filepath.Join(root, ".pose/policy/reviewers.json"))
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		a := p.Adapters["fake"]
		a.Command = []string{"missing-review-agent-command"}
		p.Adapters["fake"] = a
		raw, _ = json.Marshal(p)
		writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", string(raw))
	})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--dry-run", "--json"); code == 0 {
		t.Fatal("preview claimed an available agent", out)
	}
	if requests, err := (posemodel.Store{Root: root}).ListActionRequests(); err != nil || len(requests) != 0 {
		t.Fatal("preview opened an escalation", requests, err)
	}
	for i := 0; i < 2; i++ {
		code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--json")
		var doc struct {
			Escalation string `json:"escalation"`
		}
		if code == 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Escalation == "" {
			t.Fatal(code, out)
		}
	}
	s := posemodel.Store{Root: root}
	requests, err := s.ListActionRequests()
	if err != nil || len(requests) != 1 {
		t.Fatal(requests, err)
	}
	runs, losses, err := s.ReadReviewRuns("")
	if err != nil || len(runs) != 0 || len(losses) != 0 {
		t.Fatal("unavailable agent consumed execution", runs, losses, err)
	}
}

func TestDelegatedReviewCapabilityPersonAndSameModelEscalate(t *testing.T) {
	for _, person := range []bool{false, true} {
		t.Run(map[bool]string{false: "same model", true: "human required"}[person], func(t *testing.T) {
			root, _, _ := cliProvenanceInstance(t, func(root, _ string) {
				writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
				path := filepath.Join(root, ".pose/policy/review.json")
				raw, _ := os.ReadFile(path)
				if person {
					writeCloseoutCLIFile(t, root, ".pose/policy/review.json", strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1))
				} else {
					path = filepath.Join(root, ".pose/policy/reviewers.json")
					raw, _ = os.ReadFile(path)
					writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", strings.Replace(string(raw), `"model":"review-model"`, `"model":"implementation-model"`, 1))
				}
			})
			code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--json")
			var doc struct {
				Escalation string `json:"escalation"`
			}
			if code == 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Escalation == "" {
				t.Fatal(code, out)
			}
			s := posemodel.Store{Root: root}
			a, err := s.LoadActionRequest(doc.Escalation)
			if err != nil || a.Request.Kind != posemodel.ActionDecision || a.Request.Recipient.Role != "maintainer" || len(a.Request.Options) == 0 {
				t.Fatal(a, err)
			}
			runs, losses, err := s.ReadReviewRuns("")
			if err != nil || len(runs) != 0 || len(losses) != 0 {
				t.Fatal("non-independent reviewer started", runs, losses, err)
			}
		})
	}
}

func TestDelegatedReviewCapabilityRoutesPastASameModelAgent(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t, func(root, _ string) {
		s := posemodel.Store{Root: root}
		p, _, _, err := s.LoadDelegationPolicy()
		if err != nil {
			t.Fatal(err)
		}
		a := p.Agents["fake"]
		b := a
		a.Model = "implementation-model"
		p.Agents = map[string]posemodel.DelegationAgent{"aaa-same": a, "zzz-independent": b}
		p.Routing = &posemodel.DelegationRouting{Prefer: "any"}
		raw, err := posemodel.CanonicalDelegationPolicy(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, ".pose/policy/reviewers.json")); err != nil {
			t.Fatal(err)
		}
		writeCloseoutCLIFile(t, root, ".pose/policy/delegation.json", string(raw))
	})
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--dry-run", "--json")
	var doc struct {
		Agent   string                            `json:"agent"`
		Routing posemodel.DelegationRoutingRecord `json:"routing"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Agent != "zzz-independent" || len(doc.Routing.Candidates) != 2 || doc.Routing.Candidates[0].Skipped == "" {
		t.Fatal(code, out, errOut)
	}
}

func TestDelegatedReviewCapabilityEscalatesThroughMCP(t *testing.T) {
	for _, person := range []bool{false, true} {
		t.Run(map[bool]string{false: "unavailable", true: "person required"}[person], func(t *testing.T) {
			root, _, _ := cliProvenanceInstance(t, func(root, _ string) {
				writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
				path := filepath.Join(root, ".pose/policy/review.json")
				if person {
					raw, _ := os.ReadFile(path)
					writeCloseoutCLIFile(t, root, ".pose/policy/review.json", strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1))
				} else {
					path = filepath.Join(root, ".pose/policy/reviewers.json")
					raw, _ := os.ReadFile(path)
					var p posemodel.ReviewersPolicy
					if err := json.Unmarshal(raw, &p); err != nil {
						t.Fatal(err)
					}
					a := p.Adapters["fake"]
					a.Command = []string{"missing-review-agent-command"}
					p.Adapters["fake"] = a
					raw, _ = json.Marshal(p)
					writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", string(raw))
				}
			})
			call := delegateMCP(t, root)
			args := map[string]any{"purpose": "review", "target": "spec:alpha"}
			preview := call("pose_delegate_start", args)
			if preview["escalation_required"] != true || preview["escalation"] != nil {
				t.Fatal(preview)
			}
			s := posemodel.Store{Root: root}
			if requests, err := s.ListActionRequests(); err != nil || len(requests) != 0 {
				t.Fatal("preview mutated", requests, err)
			}
			args["apply"] = true
			first := call("pose_delegate_start", args)
			id, ok := first["escalation"].(string)
			if !ok || id == "" {
				t.Fatal(first)
			}
			if retry := call("pose_delegate_start", args); retry["escalation"] != id {
				t.Fatal(retry)
			}
			if requests, err := s.ListActionRequests(); err != nil || len(requests) != 1 {
				t.Fatal(requests, err)
			}
			if runs, losses, err := s.ReadReviewRuns(""); err != nil || len(runs) != 0 || len(losses) != 0 {
				t.Fatal("MCP spawned an unqualified run", runs, losses, err)
			}
		})
	}
}

// Trust boundary, not a claim that signatures observe review cognition: a
// trusted local command may return a canned complete draft without reading
// the code. Its proof must remain an executor declaration in every disclosure.
func TestDelegatedReviewCapabilityCannedApprovalDisclosesItsLimits(t *testing.T) {
	root, draft, bundle := cliProvenanceInstance(t, func(root, _ string) {
		s := posemodel.Store{Root: root}
		p, _, _, err := s.LoadDelegationPolicy()
		if err != nil {
			t.Fatal(err)
		}
		p.Conclusions = &posemodel.DelegationConclusions{Issuer: "fixture-executor"}
		raw, err := posemodel.CanonicalDelegationPolicy(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, ".pose/policy/reviewers.json")); err != nil {
			t.Fatal(err)
		}
		writeCloseoutCLIFile(t, root, ".pose/policy/delegation.json", string(raw))
	})
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundle, draft)
	if code, out := runPose(t, root, "delegate", "review", bundle); code != 0 {
		t.Fatal(code, out)
	}
	v, err := s.VerifyReviewBundle("spec:alpha")
	if err != nil || !v.Approved || v.Assurance == nil || v.Assurance.Run == nil {
		t.Fatal(v, err)
	}
	if v.Assurance.CognitiveIndependence != "not-observable" || !strings.Contains(strings.Join(v.Assurance.Limitations, ","), "executor-declaration-not-independent-vendor-receipt") {
		t.Fatal("canned output became proof of substantive review", v.Assurance)
	}
}

func TestDelegatedReviewCapabilityOffClosesWithDisclosure(t *testing.T) {
	root := dispatchInstanceWith(t, func(root string, git func(...string) string) {
		path := filepath.Join(root, ".pose/policy/review.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		p["delegated_review"] = false
		p["reviewer_independence"] = map[string]string{"spec": "different-actor"}
		raw, _ = json.Marshal(p)
		writeCloseoutCLIFile(t, root, ".pose/policy/review.json", string(raw))
		writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\nfunc Twice(n int) int { return n * 2 }\n")
		writeCloseoutCLIFile(t, root, "pose-mcp/lib_test.go", "package posemcp\nimport \"testing\"\nfunc TestFixture(t *testing.T) { if Twice(3) != 6 { t.Fatal(\"Twice must double its input\") } }\n")
		git("add", "-A")
	})
	s := posemodel.Store{Root: root}
	b, err := s.CurrentReviewBundle("spec:alpha")
	if err != nil || b == nil {
		t.Fatal(b, err)
	}
	args := []string{"review", "attest", b.BundleID, "--reviewer", "agent:solo", "--decision", "approved", "--criterion", "correctness|passed|integration:val-alpha|the fixture implementation satisfies its requirement", "--evidence", "integration:val-alpha", "--apply"}
	args = appendRunProvenanceTools(t, s, b.BundleID, "agent:solo", args)
	if code, out := runPose(t, root, args...); code != 0 {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "review", "verify", "spec:alpha", "--json"); code != 0 || !strings.Contains(out, posemodel.ReviewLimitationDelegationDisabled) {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "close", "spec:alpha"); code != 0 {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "closeout-check", "spec:alpha", "--json"); code != 0 || !strings.Contains(out, posemodel.ReviewLimitationDelegationDisabled) {
		t.Fatal(code, out)
	}
	state, err := s.GetCloseoutState("spec:alpha")
	if err != nil || !state.Terminal || state.Review.Assurance == nil || !strings.Contains(strings.Join(state.Review.Assurance.Limitations, ","), posemodel.ReviewLimitationDelegationDisabled) {
		t.Fatal(state, err)
	}
}

// R4 composes B19's automatic conclusion with the next round and the native
// close. The fake agent decides from the sealed source, not from a test-side
// switch, so changing only the draft cannot make the seeded defect disappear.
func TestDelegatedReviewCapabilityRejectFixApproveAndClose(t *testing.T) {
	rejected := filepath.Join(t.TempDir(), "rejected.txt")
	root, approved, first := cliProvenanceInstance(t, func(root, draft string) {
		writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\nfunc Twice(n int) int { return n } // seeded-defect\n")
		writeCloseoutCLIFile(t, root, "pose-mcp/lib_test.go", `package posemcp
import "testing"
func TestFixture(t *testing.T) {
 if Twice(3) != 6 { t.Fatal("Twice must double its input") }
}
`)
		writeCloseoutCLIFile(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"proj.fixture","name":"fixture"}`)
		s := posemodel.Store{Root: root}
		p, _, _, err := s.LoadDelegationPolicy()
		if err != nil {
			t.Fatal(err)
		}
		p.Conclusions = &posemodel.DelegationConclusions{Issuer: "fixture-executor"}
		a := p.Agents["fake"]
		a.Commands[posemodel.DelegationAccessRead] = []string{"sh", "-c", `cat >/dev/null; if grep -q seeded-defect pose-mcp/lib.go; then cat "$1"; else cat "$2"; fi`, "sh", rejected, draft}
		p.Agents["fake"] = a
		raw, err := posemodel.CanonicalDelegationPolicy(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, ".pose/policy/reviewers.json")); err != nil {
			t.Fatal(err)
		}
		writeCloseoutCLIFile(t, root, ".pose/policy/delegation.json", string(raw))
	})
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, first, approved)
	raw, err := os.ReadFile(approved)
	if err != nil {
		t.Fatal(err)
	}
	begin, end := strings.Index(string(raw), "{\"schema\""), strings.LastIndex(string(raw), "}")
	if begin < 0 || end < begin {
		t.Fatal(string(raw))
	}
	var draft posemodel.ReviewDraft
	if err := json.Unmarshal(raw[begin:end+1], &draft); err != nil {
		t.Fatal(err)
	}
	draft.Decision = "changes-requested"
	draft.Findings = []posemodel.ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "pose-mcp/lib.go", Line: 2, Summary: "seeded defect remains"}}
	reject, _ := json.Marshal(draft)
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(rejected, append(append([]byte("decision: changes-requested\n```pose-draft\n"), reject...), []byte("\n```\n")...))
	code, runOut, runErr := runPoseSplit(t, root, "delegate", "review", first)
	if code != 0 {
		t.Fatal(code, runOut, runErr)
	}
	v, err := s.VerifyReviewBundle("spec:alpha")
	if err != nil || v.Approved {
		t.Fatal("seeded defect approved", v, err)
	}
	attestations, err := s.ListReviewAttestations("")
	if err != nil || len(attestations) != 1 || attestations[0].Decision != "changes-requested" {
		t.Fatal(attestations, err, runOut, runErr)
	}
	if code, out := runPose(t, root, "close", "spec:alpha"); code == 0 {
		t.Fatal("rejected scope closed", out)
	}
	// Remediate and publish a genuinely new subject with current fixture
	// validation evidence, as a real round must do.
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\nfunc Twice(n int) int { return n * 2 } // defect fixed\n")
	cmd := exec.Command("git", "-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-am", "fix seeded defect")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	graphPath := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	graphRaw, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatal(err)
	}
	var graph posemodel.DeliveryIntegrityGraph
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		t.Fatal(err)
	}
	for i := range graph.ChangeSets {
		graph.ChangeSets[i].Head = strings.TrimSpace(string(head))
		graph.ChangeSets[i].ResolvedHead = graph.ChangeSets[i].Head
	}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].GitHead = strings.TrimSpace(string(head))
	}
	graphRaw, _ = json.Marshal(graph)
	write(graphPath, graphRaw)
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal", "--lead", "openai/implementation-model"); code != 0 {
		t.Fatal(code, out)
	}
	b, err := s.CurrentReviewBundle("spec:alpha")
	if err != nil || b == nil || b.BundleID == first {
		t.Fatal(b, err)
	}
	writeRunConclusion(t, s, b.BundleID, approved)
	raw, _ = os.ReadFile(approved)
	begin, end = strings.Index(string(raw), "{\"schema\""), strings.LastIndex(string(raw), "}")
	if err := json.Unmarshal(raw[begin:end+1], &draft); err != nil {
		t.Fatal(err)
	}
	draft.Sweep = "Inspected the full source and fixed seeded defect"
	round, err := s.ReviewRoundOf(*b)
	if err != nil || len(round.Open) != 1 {
		t.Fatal(round, err)
	}
	draft.PriorFindings = []posemodel.ReviewDraftPrior{{ID: round.Open[0].ID, Status: "fixed", Evidence: "pose-mcp/lib.go:2 now has the fix"}}
	approve, _ := json.Marshal(draft)
	write(approved, append(append([]byte("decision: approved\n```pose-draft\n"), approve...), []byte("\n```\n")...))
	code, runOut, runErr = runPoseSplit(t, root, "delegate", "review", b.BundleID)
	if code != 0 {
		t.Fatal(code, runOut, runErr)
	}
	v, err = s.VerifyReviewBundle("spec:alpha")
	if err != nil || !v.Approved {
		t.Fatal(v.Blockers, err, runOut, runErr)
	}
	if code, out := runPose(t, root, "close", "spec:alpha"); code != 0 {
		t.Fatal(code, out)
	}
	state, err := s.GetCloseoutState("spec:alpha")
	if err != nil || !state.Terminal {
		t.Fatal(state, err)
	}
	attestations, err = s.ListReviewAttestations("")
	if err != nil || len(attestations) != 2 {
		t.Fatal("history was replaced", attestations, err)
	}
}
