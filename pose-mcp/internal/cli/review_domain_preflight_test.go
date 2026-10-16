package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/mcpserver"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-approval-domain-preflight: the CLI, a signed import and MCP
// consume the domain writer's refusal; none keeps an approval policy of its own.

func reviewArtifactFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{}
	for _, dir := range []string{"review-attestations", "review-bundles", "review-attribution-supplements", "reviews"} {
		matches, _ := filepath.Glob(filepath.Join(root, ".pose", dir, "*"))
		files = append(files, matches...)
	}
	return files
}

func preflightIssuerFixture(t *testing.T) (string, ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	root := reviewBundleCLIFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 13)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	keyHash := sha256.Sum256(public)
	policy["trusted_attestation_issuers"] = []string{fmt.Sprintf("fixture#sha256:%x", keyHash)}
	raw, _ = json.Marshal(policy)
	if err := os.WriteFile(policyPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, private, public
}

const riskFinding = "f1|critical|wont-fix|leave it unresolved|-"

// V1/R7: one invalid approval refused with the same blockers by the CLI
// preview, the CLI write, the CLI signed import, the Store and MCP, and none
// of them writes a review artifact.
func TestReviewDomainPreflightCLIImportAndMCPAgreeWithTheDomain(t *testing.T) {
	root, private, public := preflightIssuerFixture(t)
	bundle, evidence := sealCLIBundle(t, root)
	store := posemodel.Store{Root: root}
	args := attestArgs(bundle.BundleID, evidence, "agent:reviewer", "--criterion", "correctness|finding|f1", "--finding", riskFinding)

	now := time.Now()
	att := posemodel.ReviewAttestation{BundleID: bundle.BundleID, Reviewer: "agent:reviewer", Decision: "approved", EvidenceRefs: []string{evidence},
		Criteria: []posemodel.ReviewCriterion{{ID: "correctness", Disposition: "finding", Evidence: "f1"}},
		Findings: []posemodel.ReviewFinding{{ID: "f1", Severity: "critical", Disposition: "wont-fix", Action: "leave it unresolved", Evidence: "-"}}}
	for _, tool := range bundle.Payload.Plan.Tools {
		switch tool.ID {
		case "artifact-check":
			att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "passed", Evidence: "check:artifact"})
		case "validate":
			att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "passed", Evidence: "validation:module"})
		default:
			// What the CLI derives for the post-review gates it is not given.
			for _, precondition := range tool.Preconditions {
				if precondition == "review-complete" {
					att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "deferred", Rationale: "post-review gate"})
					break
				}
			}
		}
	}
	domain := store.PreflightReviewAttestation(att, now)
	if !containsText(domain, "unapproved or incomplete accepted risk") {
		t.Fatalf("fixture: the domain does not refuse the approval: %v", domain)
	}
	before := reviewArtifactFiles(t, root)

	blockersOf := func(output string) []string {
		lines := []string{}
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				lines = append(lines, strings.TrimPrefix(strings.TrimSpace(line), "- "))
			}
		}
		return lines
	}
	run := func(args []string) (int, string) {
		var out, errOut bytes.Buffer
		code := cmdReview(root, args, &out, &errOut)
		return code, out.String() + errOut.String()
	}

	// The CLI write: exit 1, the domain's blockers, nothing written.
	code, output := run(args)
	if code != 1 || !strings.Contains(output, "nothing was written") {
		t.Fatalf("CLI write code=%d: %s", code, output)
	}
	if got := blockersOf(output); !reflect.DeepEqual(got, domain) {
		t.Fatalf("CLI blockers diverged from the domain:\n%v\n%v", got, domain)
	}
	// The CLI preview reports the same refusal.
	code, output = run(args[:len(args)-1])
	if got := blockersOf(output); code != 1 || !reflect.DeepEqual(got, domain) {
		t.Fatalf("CLI preview code=%d diverged: %v", code, got)
	}

	// The signed import: a trusted signature is verified, not admitted.
	completed, canonical, err := store.SignableReviewAttestation(att, now)
	if err != nil {
		t.Fatal(err)
	}
	envelope := posemodel.ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical)), Attestation: completed}
	raw, _ := json.Marshal(envelope)
	if err := os.WriteFile(filepath.Join(root, "envelope.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, importArgs := range [][]string{{"attest", "--envelope", "envelope.json"}, {"attest", "--envelope", "envelope.json", "--apply"}} {
		code, output = run(importArgs)
		if got := blockersOf(output); code != 1 || !reflect.DeepEqual(got, domain) {
			t.Fatalf("signed import %v code=%d diverged: %s", importArgs, code, output)
		}
	}

	// MCP: the envelope preview names the blockers; apply is a typed refusal.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.preflight-test")
	ts := httptest.NewServer(mcpserver.New(store).Handler("", ""))
	defer ts.Close()
	call := func(apply bool) map[string]any {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_review_attest", "arguments": map[string]any{"envelope": envelope, "apply": apply}}})
		response, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var rpc struct {
			Result map[string]any `json:"result"`
		}
		if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
			t.Fatal(err)
		}
		return rpc.Result
	}
	stringsOf := func(value any) []string {
		out := []string{}
		items, _ := value.([]any)
		for _, item := range items {
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	preview := call(false)
	structured, _ := preview["structuredContent"].(map[string]any)
	if preview["isError"] != false || !reflect.DeepEqual(stringsOf(structured["verify_blockers"]), domain) {
		t.Fatalf("MCP preview diverged: %+v", preview)
	}
	applied := call(true)
	structured, _ = applied["structuredContent"].(map[string]any)
	if applied["isError"] != true || structured["error_code"] != "review_approval_refused" || structured["bundle_id"] != bundle.BundleID || !reflect.DeepEqual(stringsOf(structured["blockers"]), domain) {
		t.Fatalf("MCP apply diverged: %+v", applied)
	}

	if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("a refused approval wrote review artifacts:\n%v\n%v", before, after)
	}
}

// V4: a legitimate negative recorded through the CLI does not need its
// criteria to satisfy the approval bar.
func TestReviewDomainPreflightCLIRecordsANegative(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	bundle, evidence := sealCLIBundle(t, root)
	args := attestArgs(bundle.BundleID, evidence, "agent:reviewer", "--criterion", "correctness|finding|f1", "--finding", "f1|high|open|fix it|-")
	for i, value := range args {
		if value == "approved" {
			args[i] = "changes-requested"
		}
	}
	var out, errOut bytes.Buffer
	if code := cmdReview(root, args, &out, &errOut); code != 0 {
		t.Fatalf("a negative was refused: code=%d %s %s", code, out.String(), errOut.String())
	}
	if att := onlyAttestation(t, root); att.Decision != "changes-requested" {
		t.Fatalf("recorded %q", att.Decision)
	}
}

// V6: the approval of a bundle that a newer seal superseded is refused by the
// CLI write, and the existing artifacts stay.
func TestReviewDomainPreflightCLIRefusesASupersededBundle(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	old, evidence := sealCLIBundle(t, root)
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"pose-mcp":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	current, _ := sealCLIBundle(t, root)
	if current.BundleID == old.BundleID {
		t.Fatal("fixture: the inputs did not change the bundle")
	}
	before := reviewArtifactFiles(t, root)
	var out, errOut bytes.Buffer
	code := cmdReview(root, attestArgs(old.BundleID, evidence, "agent:reviewer"), &out, &errOut)
	if code != 1 || !strings.Contains(out.String()+errOut.String(), "superseded") {
		t.Fatalf("a superseded bundle was approved: code=%d %s %s", code, out.String(), errOut.String())
	}
	if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("the refusal changed review artifacts:\n%v\n%v", before, after)
	}
}

// cliBuiltAttestation is the attestation `pose review attest` builds from args:
// the same decoders, so a test can hand the Store exactly what the command
// would have written.
func cliBuiltAttestation(t *testing.T, bundle posemodel.ReviewBundle, decision string, evidence, criteria, tools, findings []string) posemodel.ReviewAttestation {
	t.Helper()
	plan := posemodel.ReviewPlan{Criteria: bundle.Payload.Plan.Criteria, Tools: bundle.Payload.Plan.Tools, Structure: bundle.Payload.Plan.Structure}
	decodedTools, err := decodeReviewToolDispositions(plan, tools)
	if err != nil {
		t.Fatal(err)
	}
	decodedFindings, err := parseCLIReviewFindings(findings)
	if err != nil {
		t.Fatal(err)
	}
	decodedCriteria, err := reviewCriterionDispositions(plan, evidence, criteria)
	if err != nil {
		t.Fatal(err)
	}
	return posemodel.ReviewAttestation{BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: "agent:reviewer", Decision: decision,
		Criteria: decodedCriteria, Tools: decodedTools, EvidenceRefs: evidence, Findings: decodedFindings}
}

func cliAttestArgs(bundleID, decision string, evidence, criteria, tools, findings []string, apply bool) []string {
	args := []string{"attest", bundleID, "--reviewer", "agent:reviewer", "--decision", decision}
	for _, value := range evidence {
		args = append(args, "--evidence", value)
	}
	for _, value := range criteria {
		args = append(args, "--criterion", value)
	}
	for _, value := range tools {
		args = append(args, "--tool", value)
	}
	for _, value := range findings {
		args = append(args, "--finding", value)
	}
	if apply {
		args = append(args, "--apply")
	}
	return args
}

func refusalBlockers(output string) []string {
	lines := []string{}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			lines = append(lines, strings.TrimPrefix(strings.TrimSpace(line), "- "))
		}
	}
	return lines
}

// F1 of the first independent review, R7: every semantic defect the command
// used to refuse in its own flag parser — with its own message, exit 2, one
// defect at a time — now reaches the domain writer, and the command reports
// the domain's typed refusal: the same blockers the Store returns for the
// attestation the command builds, all of them at once, nothing written.
func TestReviewDomainPreflightCLISemanticRefusalsComeFromTheDomain(t *testing.T) {
	validTools := []string{"artifact-check|-|passed|check:artifact|", "validate|pose-mcp|passed|validation:module|"}
	for _, tc := range []struct {
		name            string
		criteria, tools []string
		findings        []string
		want            []string
	}{
		{name: "finding naming one that is not recorded", criteria: []string{"correctness|finding|f-nope|"}, tools: validTools,
			want: []string{"criterion correctness names finding f-nope, which this attestation does not record"}},
		{name: "finding naming none", criteria: []string{"correctness|finding||"}, tools: validTools,
			want: []string{"criterion correctness is disposed as a finding and names none"}},
		{name: "not-applicable without a reason", criteria: []string{"correctness|not-applicable||"}, tools: validTools,
			want: []string{"criterion correctness lacks not-applicable rationale"}},
		{name: "unknown disposition", criteria: []string{"correctness|probably-fine||"}, tools: validTools,
			want: []string{"criterion correctness has invalid disposition"}},
		{name: "criterion not in the plan", criteria: []string{"invented|passed|unit:x|"}, tools: validTools,
			want: []string{"unknown criterion invented"}},
		{name: "duplicate criterion", criteria: []string{"correctness|passed|unit:x|", "correctness|passed|unit:x|"}, tools: validTools,
			want: []string{"duplicate criterion correctness"}},
		{name: "tool not in the plan", tools: append([]string{"invented-tool|-|passed|check:x|"}, validTools...),
			want: []string{"unknown review tool disposition invented-tool"}},
		{name: "required tool omitted", tools: validTools[:1],
			want: []string{"missing required review tool disposition validate (component pose-mcp)"}},
		{name: "several defects at once", criteria: []string{"correctness|finding|f-nope|", "invented|passed|unit:x|"}, tools: validTools[:1],
			want: []string{"names finding f-nope", "unknown criterion invented", "missing required review tool disposition validate"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := reviewBundleCLIFixture(t)
			bundle, evidence := sealCLIBundle(t, root)
			att := cliBuiltAttestation(t, bundle, "approved", []string{evidence}, tc.criteria, tc.tools, tc.findings)
			before := reviewArtifactFiles(t, root)

			_, err := (posemodel.Store{Root: root}).RecordReviewAttestation(att, time.Now())
			var refusal *posemodel.ReviewApprovalRefusal
			if !errors.As(err, &refusal) {
				t.Fatalf("Store: want a typed refusal, got %v", err)
			}
			for _, want := range tc.want {
				if !containsText(refusal.Blockers, want) {
					t.Fatalf("Store blockers %v lack %q", refusal.Blockers, want)
				}
			}
			for _, apply := range []bool{false, true} {
				var out, errOut bytes.Buffer
				code := cmdReview(root, cliAttestArgs(bundle.BundleID, "approved", []string{evidence}, tc.criteria, tc.tools, tc.findings, apply), &out, &errOut)
				output := out.String() + errOut.String()
				if code != 1 || !strings.Contains(output, "nothing was written") {
					t.Fatalf("CLI apply=%t: code=%d, want the domain refusal (exit 1): %s", apply, code, output)
				}
				if got := refusalBlockers(output); !reflect.DeepEqual(got, refusal.Blockers) {
					t.Fatalf("CLI apply=%t blockers diverged from the Store:\n%v\n%v", apply, got, refusal.Blockers)
				}
			}
			if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
				t.Fatalf("a refused approval wrote review artifacts:\n%v\n%v", before, after)
			}
		})
	}
}

// F1, signed import: the attestation with several defects, signed by a
// trusted issuer, is refused by the import with the very blockers the CLI and
// the Store report.
func TestReviewDomainPreflightSignedImportAgreesOnSeveralDefects(t *testing.T) {
	root, private, public := preflightIssuerFixture(t)
	bundle, evidence := sealCLIBundle(t, root)
	criteria := []string{"correctness|finding|f-nope|", "invented|passed|unit:x|"}
	tools := []string{"artifact-check|-|passed|check:artifact|"}
	store := posemodel.Store{Root: root}
	now := time.Now()
	att := cliBuiltAttestation(t, bundle, "approved", []string{evidence}, criteria, tools, nil)
	domain := store.PreflightReviewAttestation(att, now)
	for _, want := range []string{"names finding f-nope", "unknown criterion invented", "missing required review tool disposition validate"} {
		if !containsText(domain, want) {
			t.Fatalf("fixture: domain blockers %v lack %q", domain, want)
		}
	}
	completed, canonical, err := store.SignableReviewAttestation(att, now)
	if err != nil {
		t.Fatal(err)
	}
	envelope := posemodel.ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical)), Attestation: completed}
	raw, _ := json.Marshal(envelope)
	if err := os.WriteFile(filepath.Join(root, "envelope.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	before := reviewArtifactFiles(t, root)
	var out, errOut bytes.Buffer
	code := cmdReview(root, cliAttestArgs(bundle.BundleID, "approved", []string{evidence}, criteria, tools, nil, true), &out, &errOut)
	if got := refusalBlockers(out.String() + errOut.String()); code != 1 || !reflect.DeepEqual(got, domain) {
		t.Fatalf("CLI code=%d diverged from the domain:\n%v\n%v", code, got, domain)
	}
	for _, importArgs := range [][]string{{"attest", "--envelope", "envelope.json"}, {"attest", "--envelope", "envelope.json", "--apply"}} {
		out.Reset()
		errOut.Reset()
		code := cmdReview(root, importArgs, &out, &errOut)
		if got := refusalBlockers(out.String() + errOut.String()); code != 1 || !reflect.DeepEqual(got, domain) {
			t.Fatalf("signed import %v code=%d diverged from the domain:\n%v\n%v", importArgs, code, got, domain)
		}
	}
	if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("a refused approval wrote review artifacts:\n%v\n%v", before, after)
	}
}

// F1, R4: a legitimate negative that omits the approval's judgment criteria
// and required tools is recorded by the CLI, the Store and a signed import
// alike; the approval bar is not the negative's. The same omissions on an
// approval are refused by the domain, through the command.
func TestReviewDomainPreflightNegativesOweNoApprovalAnswers(t *testing.T) {
	t.Run("judgment criterion omitted", func(t *testing.T) {
		root := judgmentReviewCLIFixture(t)
		var out, errOut bytes.Buffer
		code := cmdReview(root, []string{"attest", "spec:alpha", "--reviewer", "agent:reviewer", "--decision", "approved", "--evidence", "integration:val-alpha", "--apply"}, &out, &errOut)
		if output := out.String() + errOut.String(); code != 1 || !containsText(refusalBlockers(output), "missing criterion correctness") {
			t.Fatalf("an approval without its judgment was not refused by the domain: code=%d %s", code, output)
		}
		for _, decision := range []string{"changes-requested", "rejected"} {
			out.Reset()
			errOut.Reset()
			if code := cmdReview(root, []string{"attest", "spec:alpha", "--reviewer", "agent:reviewer", "--decision", decision, "--evidence", "integration:val-alpha", "--apply"}, &out, &errOut); code != 0 {
				t.Fatalf("%s without the approval judgment was refused: code=%d %s %s", decision, code, out.String(), errOut.String())
			}
		}
		atts, err := (posemodel.Store{Root: root}).ListReviewAttestations("")
		if err != nil || len(atts) != 2 {
			t.Fatalf("want the two negatives recorded: %d %v", len(atts), err)
		}
	})
	t.Run("required tools omitted", func(t *testing.T) {
		// The CLI.
		root := reviewBundleCLIFixture(t)
		bundle, evidence := sealCLIBundle(t, root)
		var out, errOut bytes.Buffer
		if code := cmdReview(root, cliAttestArgs(bundle.BundleID, "changes-requested", []string{evidence}, []string{"correctness|finding|f1|"}, nil, []string{"f1|high|open|fix it|-"}, true), &out, &errOut); code != 0 {
			t.Fatalf("CLI refused a negative without tools: code=%d %s %s", code, out.String(), errOut.String())
		}
		// The Store, with what the command builds.
		root = reviewBundleCLIFixture(t)
		bundle, evidence = sealCLIBundle(t, root)
		att := cliBuiltAttestation(t, bundle, "rejected", []string{evidence}, []string{"correctness|finding|f1|"}, nil, []string{"f1|high|open|fix it|-"})
		if _, err := (posemodel.Store{Root: root}).RecordReviewAttestation(att, time.Now()); err != nil {
			t.Fatalf("Store refused a negative without tools: %v", err)
		}
		// A signed import of the same negative.
		root, private, public := preflightIssuerFixture(t)
		bundle, evidence = sealCLIBundle(t, root)
		store := posemodel.Store{Root: root}
		att = cliBuiltAttestation(t, bundle, "changes-requested", []string{evidence}, []string{"correctness|finding|f1|"}, nil, []string{"f1|high|open|fix it|-"})
		completed, canonical, err := store.SignableReviewAttestation(att, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		envelope := posemodel.ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519",
			PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical)), Attestation: completed}
		raw, _ := json.Marshal(envelope)
		if err := os.WriteFile(filepath.Join(root, "envelope.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		errOut.Reset()
		if code := cmdReview(root, []string{"attest", "--envelope", "envelope.json", "--apply"}, &out, &errOut); code != 0 {
			t.Fatalf("signed import refused a negative without tools: code=%d %s %s", code, out.String(), errOut.String())
		}
		if got := onlyAttestation(t, root); got.Decision != "changes-requested" {
			t.Fatalf("recorded %q", got.Decision)
		}
	})
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
