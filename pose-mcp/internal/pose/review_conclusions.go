package pose

// Conclusions of delegated runs (spec pose-delegated-review-conclusions).
//
// A review run whose record is intact and signed, whose draft is complete
// and valid for its round, and whose bundle still governs its scope is
// turned by the engine into the attestation its draft concludes: the run's
// proof is signed, the attestation is built from the draft's typed fields and
// signed by the issuer the delegation policy names, and the domain writer
// records it under the same preflight as every attestation (R1). Anything
// short of that records no governed effect; the conclusion record of the run
// says why (R3). One run concludes once: replaying it returns its record (R4).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReviewConclusionsDir holds one conclusion record per run.
const ReviewConclusionsDir = ".pose/review-conclusions"

// Conclusion states.
const (
	ReviewConclusionRecorded = "recorded"
	ReviewConclusionRefused  = "refused"
)

// ReviewConclusion is what the engine did with a run's conclusion.
type ReviewConclusion struct {
	SchemaVersion int      `json:"schema_version"`
	RunID         string   `json:"run_id"`
	BundleID      string   `json:"bundle_id"`
	Scope         string   `json:"scope"`
	State         string   `json:"state"`
	Decision      string   `json:"decision,omitempty"`
	Attestation   string   `json:"attestation,omitempty"`
	ProofID       string   `json:"proof_id,omitempty"`
	Issuer        string   `json:"issuer,omitempty"`
	Reasons       []string `json:"reasons,omitempty"`
	ConcludedAt   string   `json:"concluded_at"`
	// Replayed is set on a result that returns an earlier record (R4).
	Replayed bool `json:"replayed,omitempty"`
}

func reviewConclusionPath(root, runID string) string {
	return filepath.Join(root, filepath.FromSlash(ReviewConclusionsDir), runID+".json")
}

// ReadReviewConclusion is the conclusion record of run, if any.
func (s Store) ReadReviewConclusion(runID string) (*ReviewConclusion, error) {
	if !validAttemptFileName(runID) {
		return nil, fmt.Errorf("pose: %q is not a run id", runID)
	}
	raw, err := readBoundedRegular(reviewConclusionPath(s.Root, runID), 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c ReviewConclusion
	if err := json.Unmarshal(raw, &c); err != nil || c.RunID != runID {
		return nil, fmt.Errorf("pose: the conclusion record of %s cannot be read", runID)
	}
	return &c, nil
}

// reviewConclusionChecks are the conditions a run must meet before its
// conclusion is recorded, each a reason when it does not (R1, R3, R4).
func (s Store) reviewConclusionChecks(run ReviewRun) ([]string, *ReviewBundle, *ReviewDraft) {
	var reasons []string
	if run.Kind != ReviewBriefKindReview || run.Purpose != "" && run.Purpose != DelegationPurposeReview {
		reasons = append(reasons, "the run is not a review: an adjudication or a smoke run records its verdict elsewhere")
	}
	if run.Status != ReviewRunCompleted {
		reasons = append(reasons, "the run did not complete: it is "+run.Status)
	}
	if run.SupersededBy != "" {
		reasons = append(reasons, "the run is superseded by "+run.SupersededBy)
	}
	if run.Provenance == nil {
		reasons = append(reasons, "the run captured no provenance, so it cannot be signed")
	}
	if run.Safety == nil || run.Safety.Operation.Outcome != OutcomeCompleted {
		reasons = append(reasons, "the run's cleanup is not confirmed complete")
	}
	if why := s.reviewRunTranscriptIntact(run); why != "" {
		reasons = append(reasons, "the run's record is not intact: "+why)
	}
	draft, err := ParseReviewDraft(run.Output)
	switch {
	case err != nil:
		reasons = append(reasons, "the conclusion cannot be read: "+err.Error())
	case draft == nil:
		reasons = append(reasons, "the run's final message carries no pose-draft block")
	}
	if len(run.DraftProblems) > 0 {
		reasons = append(reasons, "the draft does not account for its round: "+strings.Join(run.DraftProblems, "; "))
	}
	bundle, err := s.LoadReviewBundle(run.BundleID)
	if err != nil {
		return append(reasons, "the run's bundle cannot be read: "+err.Error()), nil, draft
	}
	current, err := s.CurrentReviewBundle(bundle.Payload.Scope.Ref)
	if err != nil || current == nil || current.BundleID != bundle.BundleID {
		reasons = append(reasons, "the run's bundle no longer governs "+bundle.Payload.Scope.Ref+": the subject, the plan or the policy changed since it was sealed")
	}
	if why := s.ReviewRunStillGoverns(run); why != "" {
		reasons = append(reasons, "the run no longer governs: "+why)
	}
	return reasons, &bundle, draft
}

// reviewAttestationFromDraft builds the attestation a draft concludes, for
// the run that wrote it. Evidence is the run itself and each evidence the
// draft names; a criterion or a tool the draft does not answer is left for
// the preflight to refuse, never answered for it.
func reviewAttestationFromDraft(run ReviewRun, bundle ReviewBundle, draft ReviewDraft, reviewer string) ReviewAttestation {
	att := ReviewAttestation{BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: reviewer, Decision: draft.Decision}
	evidence := map[string]bool{"delegated-run:" + run.RunID: true}
	for _, c := range draft.Criteria {
		att.Criteria = append(att.Criteria, ReviewCriterion{ID: c.ID, Disposition: c.Disposition, Evidence: c.Evidence, Rationale: c.Rationale})
		if c.Evidence != "" {
			evidence[c.Evidence] = true
		}
	}
	for _, t := range draft.Tools {
		att.Tools = append(att.Tools, ReviewToolDisposition{ID: t.ID, Component: t.Component, Disposition: t.Disposition, Evidence: t.Evidence, Rationale: t.Rationale})
	}
	for _, f := range draft.Findings {
		finding := ReviewFinding{ID: f.ID, Severity: f.Severity, Evidence: findingEvidence(f), Action: f.Action, Disposition: "open"}
		if f.Action == "" {
			finding.Action = f.Summary
		}
		// A follow-up with an owner and a review date is a risk the reviewer
		// accepted; anything else stays open and blocks an approval.
		if f.Disposition == "follow-up" && f.Owner != "" && f.ReviewBy != "" {
			finding.Disposition, finding.Owner, finding.ReviewBy, finding.Rationale = "accepted-risk", f.Owner, f.ReviewBy, f.Summary
		}
		att.Findings = append(att.Findings, finding)
	}
	for ref := range evidence {
		att.EvidenceRefs = append(att.EvidenceRefs, ref)
	}
	sort.Strings(att.EvidenceRefs)
	att.Run = &ReviewRunReference{RunID: run.RunID}
	return att
}

func findingEvidence(f ReviewDraftFinding) string {
	if f.File == "" {
		return ""
	}
	if f.Line > 0 {
		return fmt.Sprintf("file:%s:%d", f.File, f.Line)
	}
	return "file:" + f.File
}

// ConcludeReviewRun records the conclusion of run id signed by issuer (R1),
// or the reasons it records none (R3); a run that concluded before returns
// its record (R4). Without apply nothing is signed or written: the result
// says what would be.
func (s Store) ConcludeReviewRun(id, issuer string, apply bool, now time.Time) (ReviewConclusion, error) {
	if existing, err := s.ReadReviewConclusion(id); err != nil {
		return ReviewConclusion{}, err
	} else if existing != nil {
		existing.Replayed = true
		return *existing, nil
	}
	run, err := s.LoadProvenanceReviewRun(id)
	if err != nil {
		return ReviewConclusion{}, err
	}
	result := ReviewConclusion{SchemaVersion: 1, RunID: run.RunID, BundleID: run.BundleID, Scope: run.Scope, State: ReviewConclusionRefused, Issuer: issuer, ConcludedAt: now.UTC().Format(time.RFC3339Nano)}
	reasons, bundle, draft := s.reviewConclusionChecks(run)
	if draft != nil {
		result.Decision = draft.Decision
	}
	var key IssuerKey
	if len(reasons) == 0 {
		key, err = LoadIssuerKey(s.Root, issuer)
		if err != nil {
			reasons = append(reasons, "the conclusions issuer cannot sign: "+strings.TrimPrefix(err.Error(), "pose: "))
		}
	}
	if len(reasons) > 0 {
		result.Reasons = reasons
		return s.publishReviewConclusion(result, apply)
	}
	if !apply {
		result.State = ReviewConclusionRecorded
		result.Reasons = []string{"preview: the run's proof and its attestation would be signed by " + issuer + " and recorded"}
		return result, nil
	}
	proof, err := s.SignReviewRunProof(run.RunID, key, 0, now, true)
	if err != nil {
		result.Reasons = []string{"the run's proof cannot be signed: " + strings.TrimPrefix(err.Error(), "pose: ")}
		return s.publishReviewConclusion(result, true)
	}
	assurance, err := s.VerifyReviewRunProof(run.RunID, proof.ProofID)
	if err != nil || !assurance.CompleteReview {
		why := "the signed proof does not establish a complete review"
		if err != nil {
			why = "the signed proof does not verify: " + strings.TrimPrefix(err.Error(), "pose: ")
		}
		result.ProofID, result.Reasons = proof.ProofID, []string{why}
		return s.publishReviewConclusion(result, true)
	}
	att := reviewAttestationFromDraft(run, *bundle, *draft, assurance.Reviewer.Principal)
	att.Run.ProofID = proof.ProofID
	authority := &NativeAuthority{ReviewExecution: run.RunID, ImplementationPrincipal: assurance.Implementation.Principal, ImplementationExecution: assurance.Implementation.Execution}
	envelope, err := s.SignReviewAttestation(att, key, authority, now)
	if err == nil {
		var verified ReviewAttestation
		if verified, err = s.RecordReviewAttestationEnvelope(envelope, false); err == nil {
			if blockers := s.PreflightReviewAttestation(verified, now); len(blockers) > 0 {
				err = fmt.Errorf("the attestation preflight refuses it: %s", strings.Join(blockers, "; "))
			}
		}
	}
	var recorded ReviewAttestation
	if err == nil {
		recorded, err = s.RecordReviewAttestationEnvelope(envelope, true)
	}
	if err != nil {
		var refusal *ReviewApprovalRefusal
		if errors.As(err, &refusal) {
			err = fmt.Errorf("the attestation preflight refuses it: %s", strings.Join(refusal.Blockers, "; "))
		}
		result.ProofID, result.Reasons = proof.ProofID, []string{strings.TrimPrefix(err.Error(), "pose: ")}
		return s.publishReviewConclusion(result, true)
	}
	result.State, result.ProofID, result.Attestation = ReviewConclusionRecorded, proof.ProofID, recorded.AttestationID
	return s.publishReviewConclusion(result, true)
}

// publishReviewConclusion writes result once; a concurrent writer's record
// wins and is returned (R4).
func (s Store) publishReviewConclusion(result ReviewConclusion, apply bool) (ReviewConclusion, error) {
	if !apply {
		return result, nil
	}
	if _, err := ensureReviewArtifactDir(s.Root, ReviewConclusionsDir, true); err != nil {
		return result, err
	}
	raw, _ := json.MarshalIndent(result, "", "  ")
	err := PublishFile(reviewConclusionPath(s.Root, result.RunID), append(raw, '\n'), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	if err != nil {
		if existing, rerr := s.ReadReviewConclusion(result.RunID); rerr == nil && existing != nil {
			existing.Replayed = true
			return *existing, nil
		}
		return result, err
	}
	return result, nil
}

// AutoConcludeReviewRun records the conclusion of a review run the moment it
// completes, when the instance has turned the capability on: review policy
// delegated_review and an issuer in the delegation policy. Otherwise it does
// nothing and returns nil (spec compatibility: until the capability is
// adopted, attestations stay as today).
func (s Store) AutoConcludeReviewRun(run ReviewRun, now time.Time) (*ReviewConclusion, error) {
	if run.Kind != ReviewBriefKindReview || run.Status != ReviewRunCompleted || run.RunID == "" {
		return nil, nil
	}
	review, err := s.GetReviewPolicy()
	if err != nil || !review.DelegatedReview {
		return nil, nil
	}
	policy, _, found, err := s.LoadDelegationPolicy()
	if err != nil || !found || policy.ConclusionsIssuer() == "" {
		return nil, err
	}
	conclusion, err := s.ConcludeReviewRun(run.RunID, policy.ConclusionsIssuer(), true, now)
	if err != nil {
		return nil, err
	}
	return &conclusion, nil
}
