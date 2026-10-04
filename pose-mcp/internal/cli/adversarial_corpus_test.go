package cli

// Adversarial corpus against formal compliance without value (spec
// pose-mechanization-adversarial-corpus).
//
// Every anti-mechanization gate has a mechanized version of itself: a record
// that satisfies the schema while defeating the purpose. Each case below is
// that cheapest formal satisfaction, written as a behaviour the engine must
// refuse or disclose. `enforced` cases fail if the engine accepts the
// shortcut. `known-gap` cases assert the shortcut still works today, so the
// day someone closes the gap the test fails and the case must be flipped to
// `enforced` — a gap cannot close or reopen silently. The README lists every
// case; the test checks the two agree.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

type adversarialCase struct {
	name   string
	status string // enforced | known-gap
	spec   string // the spec that owns the invariant
	run    func(t *testing.T) (shortcutWorks bool, detail string)
}

var adversarialCorpus = []adversarialCase{
	{"human-label-without-confirmation", "enforced", "pose-review-attribution-roles", func(t *testing.T) (bool, string) {
		// The 6.3.0 shape: human identity, agent-written conclusions, no
		// attribution. Schema-valid; must not read as a human review.
		var att posemodel.ReviewAttestation
		_ = json.Unmarshal([]byte(`{"schema_version":2,"attestation_id":"rva-0000000000000000","bundle_id":"rvb-0000000000000000","bundle_digest":"sha256:b","reviewer":"human:oseias","decision":"approved","criteria":[{"id":"correctness","disposition":"passed","evidence":"unit:x"}],"findings":[],"attested_at":"2026-10-02T10:00:00Z"}`), &att)
		bundle := posemodel.ReviewBundle{Payload: posemodel.ReviewBundlePayload{Plan: posemodel.ReviewBundlePlan{Independence: "same-actor-separate-execution"}}}
		render := posemodel.RenderReviewAssurance(posemodel.Store{}.DescribeReviewAssurance(bundle, &att))
		readsHuman := strings.Contains(render, "verified human") || strings.Contains(render, "confirmed by human") || !strings.Contains(render, "legacy-undifferentiated")
		return readsHuman, render
	}},
	{"declared-independent-prefix", "enforced", "pose-review-assurance-disclosure", func(t *testing.T) (bool, string) {
		att := posemodel.ReviewAttestation{Reviewer: "agent:independent-anything"}
		bundle := posemodel.ReviewBundle{Payload: posemodel.ReviewBundlePayload{Plan: posemodel.ReviewBundlePlan{Independence: "different-actor"}}}
		a := posemodel.Store{}.DescribeReviewAssurance(bundle, &att)
		return a.SeparationVerified != "not-verified" || a.CognitiveIndependence != "not-observable", posemodel.RenderReviewAssurance(a)
	}},
	{"confirmation-reused-for-edited-content", "enforced", "pose-review-attribution-roles", func(t *testing.T) (bool, string) {
		att := posemodel.ReviewAttestation{BundleDigest: "sha256:b", Reviewer: "agent:a", Decision: "approved", Criteria: []posemodel.ReviewCriterion{{ID: "c", Disposition: "passed"}}}
		att.Attribution = &posemodel.ReviewAttribution{SchemaVersion: 1, PreparedBy: "agent:a", ConfirmedBy: "human:p", ConfirmationMode: posemodel.ReviewConfirmationAdoptedConclusions}
		att.Attribution.ConfirmationDigest = posemodel.ReviewConfirmationDigest(att)
		att.Decision = "approved-with-reservations"
		return posemodel.ValidateReviewAttribution(att) == nil, "a confirmation of other content"
	}},
	{"unknown-recorded-as-satisfied", "enforced", "pose-obligation-contract", func(t *testing.T) (bool, string) {
		o := corpusObligation()
		o.Knowledge, o.Satisfaction = posemodel.KnowledgeUnknown, posemodel.SatisfactionSatisfied
		return posemodel.ValidateObligation(o) == nil, "unknown + satisfied"
	}},
	{"legacy-blocker-gains-an-actor", "enforced", "pose-obligation-contract", func(t *testing.T) (bool, string) {
		o := corpusObligation()
		o.Observation.Coverage = posemodel.CoverageLegacyOpaque
		o.Targets = nil
		o.Recipient = posemodel.ObligationActor{Role: "maintainer"}
		return posemodel.ValidateObligation(o) == nil, "legacy-opaque with an invented role"
	}},
	{"blocked-counted-as-resolved", "enforced", "pose-blocked-semantics-alignment", func(t *testing.T) (bool, string) {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, ".pose/specs/2026-09-01-a.md"), "---\nslug: a\nstatus: done\n---\n\n# Spec: a\n")
		mustWrite(t, filepath.Join(root, ".pose/specs/2026-09-02-b.md"), "---\nslug: b\nstatus: blocked\n---\n\n# Spec: b\n")
		var out, errB bytes.Buffer
		cmdAdoptionMetrics(root, []string{"--json"}, &out, &errB)
		var report adoptionReport
		_ = json.Unmarshal(out.Bytes(), &report)
		return report.TaskSuccessRatioV2 == nil || *report.TaskSuccessRatioV2 != 1, out.String()
	}},
	{"flat-specs-share-one-journal", "enforced", "pose-flat-spec-amendments", func(t *testing.T) (bool, string) {
		a := posemodel.AmendmentsPath(filepath.Join("x", ".pose", "specs", "2026-10-04-a.md"))
		b := posemodel.AmendmentsPath(filepath.Join("x", ".pose", "specs", "2026-10-04-b.md"))
		return a == b, a + " / " + b
	}},
	{"local-metadata-read-as-published", "enforced", "pose-public-claims-publication-provenance", func(t *testing.T) (bool, string) {
		root := t.TempDir()
		writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"}`)
		writeSurface(t, root, "README.md", "POSE 1.7.10\n")
		p := resolvePublicVersionProvenance(root, "compatibility.json", "1.7.10")
		return p.PublishedVersion != "" || p.PublishedVersionState != "unproven", p.PublishedVersionState
	}},
	{"unavailable-producer-reads-as-empty", "enforced", "pose-obligation-projection", func(t *testing.T) (bool, string) {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), "{not json")
		mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-open.md"), "---\nslug: open\nstatus: in-progress\n---\n\n# Spec: open\n")
		r, err := posemodel.Store{Root: root}.ProjectObligations(posemodel.ObligationQuery{})
		if err != nil {
			return false, err.Error()
		}
		return r.Complete || len(r.Obligations) == 0 && !hasUnavailableCoverage(r), "complete=" + fmt.Sprint(r.Complete)
	}},
	{"trivial-change-raises-a-request", "enforced", "pose-action-requests", func(t *testing.T) (bool, string) {
		// No command opens a request implicitly: reading state, previewing a
		// request and running the lifecycle leave no journal behind.
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-tiny.md"), "---\nslug: tiny\nstatus: in-progress\n---\n\n# Spec: tiny\n\n## 2. Requirements\n\n- R1: Fix a typo.\n")
		for _, args := range [][]string{{"state", "--attention"}, {"start", "spec:tiny"}, {"close", "spec:tiny"}, {"action", "open", "--origin", "spec:tiny", "--kind", "approval", "--question", "q", "--requested-by", "agent:a", "--target", "requirement:R1", "--effect", "closeout:block"}} {
			var out, errB bytes.Buffer
			inDir(t, root, func() { Main(args, &out, &errB) })
		}
		_, err := os.Stat(filepath.Join(root, ".pose", "actions"))
		return err == nil, "a request journal exists"
	}},
	{"declined-approval-satisfies", "enforced", "pose-action-request-resolution", func(t *testing.T) (bool, string) {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-s.md"), "---\nslug: s\nstatus: in-progress\n---\n\n# Spec: s\n")
		mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:m"]}}`)
		store := posemodel.Store{Root: root}
		view, err := store.OpenActionRequest(posemodel.ActionRequest{Origin: "spec:s", Kind: posemodel.ActionApproval, Question: "Publish?", RequestedBy: posemodel.ActionPrincipal{Principal: "agent:a"},
			Recipient: posemodel.ObligationActor{Role: "maintainer"}, Targets: []posemodel.NodeRef{{Artifact: "self"}}, Effects: []posemodel.ObligationEffect{{Phase: posemodel.PhaseRelease, Mode: posemodel.EffectBlock}}}, time.Now())
		if err != nil {
			return false, err.Error()
		}
		after, err := store.ResolveActionRequest(posemodel.ActionResolution{RequestID: view.Request.ID, Type: posemodel.ActionEventAnswered, Actor: "human:m", Answer: "decline", RequestDigest: view.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: "k"}, time.Now())
		if err != nil {
			return false, err.Error()
		}
		cancel, _ := store.ResolveActionRequest(posemodel.ActionResolution{RequestID: view.Request.ID, Type: posemodel.ActionEventCancelled, Actor: "agent:other", Reason: "r", RequestDigest: view.Request.RequestDigest, ExpectedRevision: 2, IdempotencyKey: "c"}, time.Now())
		return after.Satisfaction == posemodel.SatisfactionSatisfied || cancel.State == posemodel.ActionStateCancelled, after.Satisfaction
	}},
	{"invented-trace-test-ref", "known-gap", "pose-mechanization-adversarial-corpus", func(t *testing.T) (bool, string) {
		// Follow-up 097 of pose-abm-design-basis: lint counts trace refs but
		// does not resolve `test:` names, so a trace citing a test that does
		// not exist passes --strict.
		root := t.TempDir()
		spec := strings.Replace(amendSpec, "- R1 [satisfied] check:test", "- R1 [satisfied] test:TestThatDoesNotExistAnywhere", 1)
		path := filepath.Join(root, ".pose", "specs", "2026-10-04-invented.md")
		mustWrite(t, path, strings.Replace(spec, "slug: amended", "slug: invented", 1))
		var out, errB bytes.Buffer
		rc := lintOneSpec(path, true, false, &out, &errB)
		return rc == 0, out.String()
	}},
}

func hasUnavailableCoverage(r posemodel.ObligationReport) bool {
	for _, c := range r.Coverage {
		if c.State == posemodel.CoverageStateUnavailable {
			return true
		}
	}
	return false
}

func corpusObligation() posemodel.Obligation {
	src := posemodel.NodeRef{Artifact: "xref:proj.x/spec:s", Kind: "requirement", ID: "R1"}
	o := posemodel.Obligation{SchemaVersion: 1, Project: "proj.x", Source: posemodel.ObligationSource{Producer: "readiness", Ref: src}, Category: posemodel.ObligationEvidence,
		ReasonCode: "evidence-missing", Condition: "evidence exists", Recipient: posemodel.ObligationActor{Unassigned: true}, Targets: []posemodel.NodeRef{src},
		Effects: []posemodel.ObligationEffect{{Phase: posemodel.PhaseCloseout, Mode: posemodel.EffectBlock}}, Satisfaction: posemodel.SatisfactionPending,
		Knowledge: posemodel.KnowledgeKnown, Waiting: posemodel.WaitingExecution, Observation: posemodel.ObligationObservation{Freshness: posemodel.FreshnessCurrent, Coverage: posemodel.CoverageComplete}, Message: "m"}
	o.ID = posemodel.ObligationID(o.Project, o.Source.Producer, o.Source.Ref, "r", "d")
	return o
}

func TestAdversarialCorpus(t *testing.T) {
	for _, tc := range adversarialCorpus {
		t.Run(tc.name, func(t *testing.T) {
			works, detail := tc.run(t)
			switch tc.status {
			case "enforced":
				if works {
					t.Errorf("the shortcut works again (owner %s): %s", tc.spec, detail)
				}
			case "known-gap":
				if !works {
					t.Errorf("the gap is closed — flip %s to enforced in the corpus and the README", tc.name)
				}
			default:
				t.Fatalf("unknown status %q", tc.status)
			}
		})
	}
}

func TestAdversarialCorpusReadmeListsEveryCase(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "adversarial", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile("(?m)^\\| `([a-z0-9-]+)` \\| (enforced|known-gap|planned) \\|")
	listed := map[string]string{}
	for _, m := range row.FindAllStringSubmatch(string(raw), -1) {
		listed[m[1]] = m[2]
	}
	for _, tc := range adversarialCorpus {
		if listed[tc.name] != tc.status {
			t.Errorf("README lists %s as %q, the corpus as %q", tc.name, listed[tc.name], tc.status)
		}
		delete(listed, tc.name)
	}
	for name, status := range listed {
		if status != "planned" {
			t.Errorf("README lists %s (%s) with no corpus case", name, status)
		}
	}
}
