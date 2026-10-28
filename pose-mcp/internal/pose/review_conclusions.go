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
	// ReviewConclusionPending reserves a run while its conclusion is signed
	// and recorded.
	ReviewConclusionPending = "concluding"
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
	c, _, err := s.readReviewConclusionRaw(runID)
	return c, err
}

// readReviewConclusionRaw is the conclusion record of run and the bytes it
// was parsed from: one read, so what is judged is what a write expects
// (review round 2, B4).
func (s Store) readReviewConclusionRaw(runID string) (*ReviewConclusion, []byte, error) {
	raw, err := readBoundedRegular(reviewConclusionPath(s.Root, runID), 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var c ReviewConclusion
	if err := json.Unmarshal(raw, &c); err != nil || c.RunID != runID {
		return nil, nil, fmt.Errorf("pose: the conclusion record of %s cannot be read", runID)
	}
	return &c, raw, nil
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
		att.Criteria = append(att.Criteria, ReviewCriterion{ID: c.ID, Disposition: c.Disposition, Evidence: c.Evidence, Rationale: c.Rationale, Mappings: c.Mappings})
		if c.Evidence != "" {
			evidence[c.Evidence] = true
		}
	}
	var tools []ReviewToolDisposition
	for _, t := range draft.Tools {
		tools = append(tools, ReviewToolDisposition{ID: t.ID, Component: t.Component, Disposition: t.Disposition, Evidence: t.Evidence, Rationale: t.Rationale})
	}
	att.Tools = ReviewToolDefaults(bundle.Payload.Plan.Tools, tools)
	for _, f := range draft.Findings {
		finding := ReviewFinding{ID: f.ID, Severity: f.Severity, Evidence: findingEvidence(f), Action: f.Action, Disposition: "open"}
		if f.Action == "" {
			finding.Action = f.Summary
		}
		// A follow-up the reviewer gave an action, an owner and a review date
		// is a risk the reviewer accepted; anything else stays open and blocks
		// an approval. The engine never writes the action for the reviewer
		// (review round 1, B3).
		if f.Disposition == "follow-up" && f.Action != "" && f.Owner != "" && f.ReviewBy != "" {
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

// ReviewToolDefaults is given followed by the dispositions that state no
// conclusion, for the plan's tools given leaves out: a post-review gate
// deferred, a recommended tool not used. A required tool left out stays out,
// for the domain writer to name. `review attest` and a run's conclusion fill
// the same defaults, so a run's conclusion and its attestation agree on them.
func ReviewToolDefaults(plan []ReviewPlanTool, given []ReviewToolDisposition) []ReviewToolDisposition {
	out := append([]ReviewToolDisposition{}, given...)
	provided := map[string]bool{}
	for _, t := range given {
		provided[t.ID+"\x00"+t.Component] = true
	}
	for _, tool := range plan {
		if provided[tool.ID+"\x00"+tool.Component] {
			continue
		}
		if containsString(tool.Preconditions, "review-complete") {
			out = append(out, ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "deferred", Rationale: "post-review gate"})
		} else if tool.Requiredness == "recommended" {
			out = append(out, ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "not-used", Rationale: "not used during review"})
		}
	}
	return out
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

// runConclusionDiffers is why att does not state the typed conclusion of a
// run briefed to write one (template review-brief/3), or "": its criteria,
// tools and findings are the ones the run's draft concludes, whoever signs
// it. A run briefed with an earlier template carried no typed conclusion and
// binds only its decision, as before (review round 1, B1; decision D1).
func runConclusionDiffers(run ReviewRun, bundle ReviewBundle, att ReviewAttestation) string {
	if run.Provenance == nil || run.Provenance.TemplateVersion != reviewBriefTemplateConclusions {
		return ""
	}
	draft, err := ParseReviewDraft(run.Output)
	if err != nil {
		return "the run's draft cannot be read: " + err.Error()
	}
	if draft == nil {
		draft = &ReviewDraft{Decision: att.Decision}
	}
	want := reviewAttestationFromDraft(run, bundle, *draft, att.Reviewer)
	var differ []string
	differ = append(differ, differingItems("criterion", mappingsByDelta(att.Criteria), mappingsByDelta(want.Criteria))...)
	differ = append(differ, differingItems("tool", concludingTools(bundle.Payload.Plan.Tools, att.Tools), concludingTools(bundle.Payload.Plan.Tools, want.Tools))...)
	differ = append(differ, differingItems("finding", att.Findings, want.Findings)...)
	if len(differ) > 0 {
		return strings.Join(differ, "; ")
	}
	return ""
}

// mappingsByDelta is criteria with each one's mappings ordered by delta: the
// order a writer lists them in states nothing (review round 2, F1).
func mappingsByDelta(criteria []ReviewCriterion) []ReviewCriterion {
	out := make([]ReviewCriterion, 0, len(criteria))
	for _, c := range criteria {
		c.Mappings = append([]ReviewCriterionMapping(nil), c.Mappings...)
		sort.SliceStable(c.Mappings, func(i, j int) bool {
			a, _ := json.Marshal(c.Mappings[i])
			b, _ := json.Marshal(c.Mappings[j])
			return c.Mappings[i].Delta < c.Mappings[j].Delta || c.Mappings[i].Delta == c.Mappings[j].Delta && string(a) < string(b)
		})
		if len(c.Mappings) == 0 {
			c.Mappings = nil
		}
		out = append(out, c)
	}
	return out
}

// concludingTools are the dispositions of tools that state a conclusion:
// a post-review gate deferred and a recommended tool not used state none,
// whatever their rationale says, so a writer that fills them is not
// answering for the reviewer.
func concludingTools(plan []ReviewPlanTool, tools []ReviewToolDisposition) []ReviewToolDisposition {
	planned := map[string]ReviewPlanTool{}
	for _, tool := range plan {
		planned[tool.ID+"\x00"+tool.Component] = tool
	}
	out := []ReviewToolDisposition{}
	for _, t := range tools {
		tool, ok := planned[t.ID+"\x00"+t.Component]
		if ok && t.Evidence == "" && (t.Disposition == "deferred" && containsString(tool.Preconditions, "review-complete") || t.Disposition == "not-used" && tool.Requiredness == "recommended") {
			continue
		}
		out = append(out, t)
	}
	return out
}

// differingItems names the items of got and want, keyed by id and
// component, that one has and the other lacks or states differently.
func differingItems[T any](kind string, got, want []T) []string {
	index := func(items []T) (map[string]string, []string) {
		out, keys := map[string]string{}, []string{}
		for _, v := range items {
			raw, _ := json.Marshal(v)
			var m struct {
				ID        string `json:"id"`
				Component string `json:"component"`
			}
			_ = json.Unmarshal(raw, &m)
			key := m.ID
			if m.Component != "" {
				key += " (component " + m.Component + ")"
			}
			if _, seen := out[key]; !seen {
				keys = append(keys, key)
			}
			out[key] += string(raw)
		}
		return out, keys
	}
	g, gk := index(got)
	w, wk := index(want)
	var differ []string
	for _, k := range gk {
		switch wv, ok := w[k]; {
		case !ok:
			differ = append(differ, kind+" "+k+" is not in the draft")
		case wv != g[k]:
			differ = append(differ, kind+" "+k+" differs from the draft's")
		}
	}
	for _, k := range wk {
		if _, ok := g[k]; !ok {
			differ = append(differ, kind+" "+k+" of the draft is missing")
		}
	}
	return differ
}

// reviewConclusionLockWait bounds how long a conclusion waits for another
// conclusion of the same run to finish.
const reviewConclusionLockWait = 2 * time.Minute

// lockReviewConclusion holds run's conclusion lock: an exclusive lock on a
// private file that the system releases when the process ends, so a
// reservation nobody holds was interrupted, and two conclusions of one run
// never decide at once (review round 2, B1).
func (s Store) lockReviewConclusion(runID string) (func(), error) {
	dir, err := s.DelegationPrivateDir("conclusion-locks")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, runID+".lock")
	// The lock is this regular file and no other: a link planted in its
	// place would lock what it points at (review round 3, F1).
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("pose: the conclusion lock of %s is not a regular file", runID)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	named, lerr := os.Lstat(path)
	if err != nil || lerr != nil || !named.Mode().IsRegular() || !os.SameFile(opened, named) {
		file.Close()
		return nil, fmt.Errorf("pose: the conclusion lock of %s changed while it was opened", runID)
	}
	deadline := time.Now().Add(reviewConclusionLockWait)
	for {
		unlock, err := lockSpecTransferFile(file)
		if err == nil {
			return func() { unlock(); file.Close() }, nil
		}
		// A platform without locks fails at once with that cause (review
		// round 3, F2).
		if errors.Is(err, errFileLockUnsupported) {
			file.Close()
			return nil, fmt.Errorf("pose: a conclusion needs a file lock: %w", err)
		}
		if time.Now().After(deadline) {
			file.Close()
			return nil, fmt.Errorf("pose: another conclusion of %s holds its lock", runID)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ConcludeReviewRun records the conclusion of run id signed by issuer (R1),
// or the reasons it records none (R3); a run that concluded before returns
// its record (R4). The run's conclusion lock is held from the first read to
// the last write, so two conclusions of one run never both sign and record
// (review rounds 1 and 2, B2/B1). Without apply nothing is written: the
// result says what the checks found, and that the attestation preflight runs
// only on apply.
func (s Store) ConcludeReviewRun(id, issuer string, apply bool, now time.Time) (ReviewConclusion, error) {
	if !validAttemptFileName(id) {
		return ReviewConclusion{}, fmt.Errorf("pose: %q is not a run id", id)
	}
	if apply {
		unlock, err := s.lockReviewConclusion(id)
		if err != nil {
			return ReviewConclusion{}, err
		}
		defer unlock()
	}
	existing, existingRaw, err := s.readReviewConclusionRaw(id)
	if err != nil {
		return ReviewConclusion{}, err
	}
	var over []byte
	if existing != nil {
		if existing.State != ReviewConclusionPending {
			existing.Replayed = true
			return *existing, nil
		}
		// Nobody holds the lock, so this reservation was interrupted: an
		// attestation it recorded that verifies as this run's conclusion
		// completes it; otherwise the run is concluded again in its place.
		recovered, err := s.recoveredReviewConclusion(*existing)
		if err != nil {
			return ReviewConclusion{}, err
		}
		if recovered != nil {
			if !apply {
				recovered.Reasons = append(recovered.Reasons, "preview: on apply the interrupted conclusion is completed with this attestation")
				return *recovered, nil
			}
			out, err := s.publishReviewConclusion(*recovered, true, existingRaw)
			out.Replayed = true
			return out, err
		}
		over = existingRaw
	}
	return s.concludeReviewRun(id, issuer, apply, now, over)
}

// recoveredReviewConclusion is pending completed with the attestation an
// interrupted conclusion recorded, when one verifies as this run's
// conclusion: signed, run-backed by a proof that verifies, for the run's
// bundle, stating its draft (review round 2, B3). A record that cannot be
// read is an error, not an absence.
func (s Store) recoveredReviewConclusion(pending ReviewConclusion) (*ReviewConclusion, error) {
	if pending.BundleID == "" {
		return nil, nil
	}
	bundle, err := s.LoadReviewBundle(pending.BundleID)
	if err != nil {
		return nil, err
	}
	read, err := s.ReadReviewAttestations(pending.BundleID)
	if err != nil {
		return nil, err
	}
	// A history read with losses may hide the attestation that is current:
	// nothing is recovered from it, and the reservation stays (review round
	// 3, B1).
	if !read.Coverage.Complete() {
		return nil, fmt.Errorf("pose: the attestation history of %s is incomplete, so the interrupted conclusion of %s cannot be recovered: %v", pending.BundleID, pending.RunID, read.Coverage.Losses)
	}
	for _, att := range read.Attestations {
		if att.Run == nil || att.Run.RunID != pending.RunID || att.Decision != pending.Decision {
			continue
		}
		if blockers := s.runBackedReviewBlockers(bundle, att); len(blockers) > 0 {
			continue
		}
		done := pending
		done.State, done.Attestation, done.ProofID = ReviewConclusionRecorded, att.AttestationID, att.Run.ProofID
		done.Reasons = []string{"completed from the attestation an interrupted conclusion recorded"}
		return &done, nil
	}
	return nil, nil
}

func (s Store) concludeReviewRun(id, issuer string, apply bool, now time.Time, takeOver []byte) (ReviewConclusion, error) {
	result := ReviewConclusion{SchemaVersion: 1, RunID: id, State: ReviewConclusionRefused, Issuer: issuer, ConcludedAt: now.UTC().Format(time.RFC3339Nano)}
	run, err := s.LoadProvenanceReviewRun(id)
	if errors.Is(err, errReviewRunHistoryIncomplete) {
		// A record that cannot be read is a reason too, kept under the id
		// asked for (review round 1, B5).
		result.Reasons = []string{"the run's record cannot be read: " + err.Error()}
		return s.publishReviewConclusion(result, apply, takeOver)
	}
	if err != nil {
		return ReviewConclusion{}, err
	}
	result.BundleID, result.Scope = run.BundleID, run.Scope
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
		return s.publishReviewConclusion(result, apply, takeOver)
	}
	if !apply {
		result.State = ReviewConclusionRecorded
		result.Reasons = []string{"preview: the run passes the conclusion checks; on apply its proof and its attestation are signed by " + issuer + " and recorded if the attestation preflight admits them, which a preview does not run"}
		return result, nil
	}
	// The reservation records that a conclusion started, for a later one to
	// recover if this one is interrupted.
	pending := result
	pending.State, pending.Reasons = ReviewConclusionPending, nil
	reservation, err := s.writeReviewConclusion(pending, takeOver)
	if err != nil {
		return result, err
	}
	finish := func(final ReviewConclusion) (ReviewConclusion, error) {
		return s.publishReviewConclusion(final, true, reservation)
	}
	proof, err := s.SignReviewRunProof(run.RunID, key, 0, now, true)
	if err != nil {
		result.Reasons = []string{"the run's proof cannot be signed: " + strings.TrimPrefix(err.Error(), "pose: ")}
		return finish(result)
	}
	assurance, err := s.VerifyReviewRunProof(run.RunID, proof.ProofID)
	if err != nil || !assurance.CompleteReview {
		why := "the signed proof does not establish a complete review"
		if err != nil {
			why = "the signed proof does not verify: " + strings.TrimPrefix(err.Error(), "pose: ")
		}
		result.ProofID, result.Reasons = proof.ProofID, []string{why}
		return finish(result)
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
		return finish(result)
	}
	result.State, result.ProofID, result.Attestation = ReviewConclusionRecorded, proof.ProofID, recorded.AttestationID
	return finish(result)
}

// writeReviewConclusion writes result where nothing is, or in place of the
// bytes over when over is set, and returns what it wrote.
func (s Store) writeReviewConclusion(result ReviewConclusion, over []byte) ([]byte, error) {
	if _, err := ensureReviewArtifactDir(s.Root, ReviewConclusionsDir, true); err != nil {
		return nil, err
	}
	raw, _ := json.MarshalIndent(result, "", "  ")
	raw = append(raw, '\n')
	expect := PublishExpectation{Mode: ExpectAbsent}
	if over != nil {
		expect = ExpectBytes(over, true)
	}
	return raw, PublishFile(reviewConclusionPath(s.Root, result.RunID), raw, 0o644, PublishAuthoritative, expect)
}

// publishReviewConclusion writes result once, or in place of the
// reservation it holds; a concurrent writer's record wins and is returned
// (R4).
func (s Store) publishReviewConclusion(result ReviewConclusion, apply bool, over []byte) (ReviewConclusion, error) {
	if !apply {
		return result, nil
	}
	if _, err := s.writeReviewConclusion(result, over); err != nil {
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
