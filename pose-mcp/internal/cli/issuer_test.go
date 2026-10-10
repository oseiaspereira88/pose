package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-native-attestation-issuer: a project with POSE alone creates an
// issuer, pins it, adopts signed-attestations and closes a review with a
// natively signed attestation, while an unsigned one stops counting.
func TestNativeIssuerJourneyWithPOSEAlone(t *testing.T) {
	home := filepath.Join(t.TempDir(), "issuers")
	t.Setenv("POSE_ISSUER_HOME", home)
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\n")
	graph := posemodel.DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ChangeSets: []posemodel.ChangeSet{{
			ID: "cs-alpha", Spec: "alpha", Selector: "range:base..head", Base: "base", Head: "head", ResolvedBase: "base-resolved", ResolvedHead: "head-resolved",
			Paths:      []posemodel.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}},
			DiffDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		}},
		Deliveries:        []posemodel.DeliveryTarget{{Spec: "alpha", Ref: "contract:alpha-api", Kind: "contract", ID: "alpha-api", Module: "pose-mcp", Profile: "api-contract", Entrypoint: "pose-mcp/lib.go"}},
		ValidationResults: []posemodel.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}},
		Reverse:           map[string][]string{"pose-mcp/lib.go": {"alpha"}},
	}
	rawGraph, _ := json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(rawGraph))

	run := func(want int, f func([]string, *bytes.Buffer, *bytes.Buffer) int, args ...string) string {
		t.Helper()
		var out, errOut bytes.Buffer
		if code := f(args, &out, &errOut); code != want {
			t.Fatalf("%v: code=%d want %d\nout=%s\nerr=%s", args, code, want, out.String(), errOut.String())
		}
		return out.String() + errOut.String()
	}
	issuer := func(args []string, out, errOut *bytes.Buffer) int { return cmdIssuer(root, args, out, errOut) }
	adopt := func(args []string, out, errOut *bytes.Buffer) int { return cmdAdopt(root, args, out, errOut) }
	attest := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewAttest(root, args, out, errOut) }
	verify := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewVerify(root, args, out, errOut) }
	seal := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewBundle(root, args, out, errOut) }

	// Without a pinned issuer the capability names the native way to get one.
	if msg := run(1, adopt, "signed-attestations", "--apply"); !strings.Contains(msg, "pose issuer init") {
		t.Fatalf("the prerequisite does not name the native issuer: %s", msg)
	}
	initOut := run(0, issuer, "init", "maintainer")
	// The suggested pin is for attestations only (spec pose-signed-legacy-attestation-ledger, R6).
	if !strings.Contains(initOut, "issuer.next=pose issuer pin maintainer --attestations --apply") || !strings.Contains(initOut, "a person alone controls") {
		t.Fatalf("init suggests a human-authority pin for a shared key: %s", initOut)
	}
	raw, err := os.ReadFile(filepath.Join(home, "maintainer.key"))
	if err != nil {
		t.Fatal(err)
	}
	var keyFile struct{ Seed string }
	_ = json.Unmarshal(raw, &keyFile)
	if keyFile.Seed == "" || strings.Contains(initOut, keyFile.Seed) {
		t.Fatalf("init printed the private key or wrote none: %s", initOut)
	}
	if preview := run(0, issuer, "pin", "maintainer"); !strings.Contains(preview, "issuer.apply=false") {
		t.Fatalf("pin wrote without --apply: %s", preview)
	}
	run(0, issuer, "pin", "maintainer", "--apply")
	run(0, adopt, "signed-attestations", "--apply")
	run(0, seal, "spec:alpha", "--seal")

	base := []string{"spec:alpha", "--reviewer", "agent:reviewer", "--decision", "approved", "--evidence", "integration:val-alpha",
		"--criterion", "correctness|passed|integration:val-alpha|negative paths were exercised against the sealed subject"}
	// An unsigned attestation no longer closes the scope.
	if msg := run(1, attest, append(base, "--apply")...); !strings.Contains(msg, "sign") {
		t.Fatalf("an unsigned attestation was accepted under signed-attestations: %s", msg)
	}
	signed := run(0, attest, append(base, "--sign", "maintainer", "--apply")...)
	if !strings.Contains(signed, "review_attestation.envelope=verified") || strings.Contains(signed, keyFile.Seed) {
		t.Fatalf("signed attest output: %s", signed)
	}
	run(0, verify, "spec:alpha")
	if listed := run(0, issuer, "list"); !strings.Contains(listed, "maintainer#sha256:") || !strings.Contains(listed, "local key") {
		t.Fatalf("list does not show the pinned local issuer: %s", listed)
	}
}

// Spec pose-signed-legacy-attestation-ledger: in an instance with an unsigned
// attestation, adopting signed attestations seals its history in one step.
func TestAdoptingSignedAttestationsSealsHistory(t *testing.T) {
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(t.TempDir(), "issuers"))
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\n")
	graph := posemodel.DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ChangeSets: []posemodel.ChangeSet{{
			ID: "cs-alpha", Spec: "alpha", Selector: "range:base..head", Base: "base", Head: "head", ResolvedBase: "base-resolved", ResolvedHead: "head-resolved",
			Paths:      []posemodel.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}},
			DiffDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		}},
		Deliveries:        []posemodel.DeliveryTarget{{Spec: "alpha", Ref: "contract:alpha-api", Kind: "contract", ID: "alpha-api", Module: "pose-mcp", Profile: "api-contract", Entrypoint: "pose-mcp/lib.go"}},
		ValidationResults: []posemodel.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}},
		Reverse:           map[string][]string{"pose-mcp/lib.go": {"alpha"}},
	}
	rawGraph, _ := json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(rawGraph))
	run := func(want int, f func([]string, *bytes.Buffer, *bytes.Buffer) int, args ...string) string {
		t.Helper()
		var out, errOut bytes.Buffer
		if code := f(args, &out, &errOut); code != want {
			t.Fatalf("%v: code=%d want %d\nout=%s\nerr=%s", args, code, want, out.String(), errOut.String())
		}
		return out.String() + errOut.String()
	}
	issuer := func(args []string, out, errOut *bytes.Buffer) int { return cmdIssuer(root, args, out, errOut) }
	adopt := func(args []string, out, errOut *bytes.Buffer) int { return cmdAdopt(root, args, out, errOut) }
	attest := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewAttest(root, args, out, errOut) }
	verify := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewVerify(root, args, out, errOut) }
	seal := func(args []string, out, errOut *bytes.Buffer) int { return cmdReviewBundle(root, args, out, errOut) }

	run(0, seal, "spec:alpha", "--seal")
	run(0, attest, "spec:alpha", "--reviewer", "agent:reviewer", "--decision", "approved", "--evidence", "integration:val-alpha",
		"--criterion", "correctness|passed|integration:val-alpha|negative paths were exercised against the sealed subject", "--apply")
	run(0, issuer, "init", "maintainer")
	// Without a local issuer pinned for attestations, adopting names what to do.
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14","trusted_attestation_issuers":["conductor#sha256:`+strings.Repeat("a", 64)+`"]}`)
	if msg := run(1, adopt, "signed-attestations", "--apply"); !strings.Contains(msg, "pose issuer init") {
		t.Fatalf("adoption with history and no local issuer: %s", msg)
	}
	run(0, issuer, "pin", "maintainer", "--apply")
	// One contract: the preview shows the sealing, and apply does it.
	preview := run(0, adopt, "signed-attestations")
	if !strings.Contains(preview, "seal 1 attestation(s) recorded without a signature") || !strings.Contains(preview, "maintainer#sha256:") || !strings.Contains(preview, "adopt.apply=false") {
		t.Fatalf("adopt preview does not show the sealing: %s", preview)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose/review-ledgers")); len(entries) != 0 {
		t.Fatal("the preview sealed a ledger")
	}
	// A second local issuer makes the sealer a choice: --issuer resolves it.
	run(0, issuer, "init", "second")
	run(0, issuer, "pin", "second", "--apply")
	if msg := run(1, adopt, "signed-attestations", "--apply"); !strings.Contains(msg, "--issuer") {
		t.Fatalf("an ambiguous sealer was not named: %s", msg)
	}
	run(0, adopt, "signed-attestations", "--issuer", "maintainer", "--apply")
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose/review-ledgers")); len(entries) != 1 {
		t.Fatalf("adoption did not seal exactly one ledger: %d", len(entries))
	}
	// The scope is open, so the policy change supersedes its bundle; a closed
	// scope keeps the policy it was sealed with, and only signing is live —
	// that path is covered by TestLegacyLedgerKeepsSealedAttestationsValid.
	if msg := run(1, verify, "spec:alpha"); !strings.Contains(msg, "superseded") {
		t.Fatalf("verify on an open scope after a policy change: %s", msg)
	}
}
