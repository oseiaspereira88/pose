package pose

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const reviewRunProofDir = ".pose/review-run-proofs"
const ReviewRunMachineAttested = "machine-attested"

// ModelIdentity is explicitly declared by the executor's protected policy.
// Signing authenticates that declaration, never an independent vendor receipt.
type ModelIdentity struct {
	Principal string `json:"principal"`
	Execution string `json:"execution,omitempty"`
	Vendor    string `json:"vendor"`
	Model     string `json:"model"`
}
type ReviewRunProvenancePolicy struct {
	Trust          string        `json:"trust"`
	Implementation ModelIdentity `json:"implementation"`
}
type ReviewRunProvenance struct {
	ExecutionState        string        `json:"execution_state"`
	PlannedWireDigest     string        `json:"planned_wire_digest"`
	WireBytes             int64         `json:"wire_bytes"`
	WireComplete          bool          `json:"wire_complete"`
	SchemaVersion         int           `json:"schema_version"`
	Project               string        `json:"project"`
	Audience              string        `json:"audience"`
	WorktreeDigest        string        `json:"worktree_digest"`
	Trust                 string        `json:"trust"`
	Implementation        ModelIdentity `json:"implementation"`
	Reviewer              ModelIdentity `json:"reviewer"`
	ConfigDigest          string        `json:"config_digest"`
	CommandTemplateDigest string        `json:"command_template_digest"`
	ArgvDigest            string        `json:"argv_digest"`
	ExecutableDigest      string        `json:"executable_digest"`
	TemplateVersion       string        `json:"template_version"`
	WireDigest            string        `json:"wire_digest"`
	NotesDigest           string        `json:"notes_digest"`
	RawTranscriptDigest   string        `json:"raw_transcript_digest"`
	RawOutputDigest       string        `json:"raw_output_digest"`
	OutputSource          string        `json:"output_source"`
	OutputComplete        bool          `json:"output_complete"`
}
type ReviewRunReference struct {
	RunID   string `json:"run_id"`
	ProofID string `json:"proof_id"`
}
type ReviewRunProofClaim struct {
	SchemaVersion    int    `json:"schema_version"`
	Project          string `json:"project"`
	Audience         string `json:"audience"`
	RunID            string `json:"run_id"`
	RecordDigest     string `json:"record_digest"`
	BundleDigest     string `json:"bundle_digest"`
	Scope            string `json:"scope"`
	Kind             string `json:"kind"`
	AttemptID        string `json:"attempt_id"`
	Resumes          string `json:"resumes,omitempty"`
	TranscriptDigest string `json:"transcript_digest"`
	OutputDigest     string `json:"output_digest"`
	ProvenanceDigest string `json:"provenance_digest"`
	IssuedAt         string `json:"issued_at"`
	ExpiresAt        string `json:"expires_at,omitempty"`
}
type ReviewRunProof struct {
	SchemaVersion int                 `json:"schema_version"`
	ProofID       string              `json:"proof_id,omitempty"`
	Issuer        string              `json:"issuer"`
	Algorithm     string              `json:"algorithm"`
	PublicKey     string              `json:"public_key"`
	Signature     string              `json:"signature,omitempty"`
	Claim         ReviewRunProofClaim `json:"claim"`
}
type ReviewRunAssurance struct {
	DraftDecision    string        `json:"draft_decision"`
	ProofIssuedAt    string        `json:"proof_issued_at"`
	ProofExpiresAt   string        `json:"proof_expires_at,omitempty"`
	State            string        `json:"state"`
	CompleteReview   bool          `json:"complete_review"`
	Assurance        string        `json:"assurance"`
	RunID            string        `json:"run_id"`
	ProofID          string        `json:"proof_id"`
	Issuer           string        `json:"issuer"`
	Reviewer         ModelIdentity `json:"reviewer"`
	Implementation   ModelIdentity `json:"implementation"`
	IdentitySource   string        `json:"identity_source"`
	VendorDifference bool          `json:"vendor_difference"`
	ModelDifference  bool          `json:"model_difference"`
	OutputComplete   bool          `json:"output_complete"`
	Limitations      []string      `json:"limitations"`
}

var runIdentityComponent = regexp.MustCompile(`^[a-z0-9][a-z0-9._:/@+\-]{0,255}$`)

func normalizeModelIdentity(id ModelIdentity) (ModelIdentity, error) {
	id.Vendor = strings.ToLower(strings.TrimSpace(id.Vendor))
	id.Model = strings.ToLower(strings.TrimSpace(id.Model))
	if !validReviewPrincipal(id.Principal) || strings.TrimSpace(id.Principal) != id.Principal {
		return id, fmt.Errorf("provenance principal must be explicit")
	}
	for _, value := range []string{id.Vendor, id.Model} {
		if !runIdentityComponent.MatchString(value) || value == "unknown" || value == "unspecified" || value == "default" {
			return id, fmt.Errorf("provenance vendor/model baseline is unknown or malformed")
		}
	}
	if id.Execution != "" && (len(id.Execution) > 256 || strings.TrimSpace(id.Execution) != id.Execution || strings.ContainsAny(id.Execution, "\r\n\x00")) {
		return id, fmt.Errorf("invalid implementation execution")
	}
	return id, nil
}
func validateRunProvenancePolicy(policy ReviewersPolicy) error {
	if policy.Provenance == nil {
		return nil
	}
	if policy.Provenance.Trust != "executor-attested" {
		return fmt.Errorf("provenance requires explicit executor-attested trust")
	}
	id, e := normalizeModelIdentity(policy.Provenance.Implementation)
	if e != nil {
		return e
	}
	if id.Execution == "" {
		return fmt.Errorf("provenance needs implementation execution")
	}
	for name, adapter := range policy.Adapters {
		reviewer, e := normalizeModelIdentity(ModelIdentity{Principal: adapter.Executor, Vendor: adapter.Vendor, Model: adapter.Model})
		if e != nil || !strings.HasPrefix(reviewer.Principal, "agent:") {
			return fmt.Errorf("adapter %s needs an explicit agent executor and known vendor/model", name)
		}
	}
	return nil
}
func (s Store) prepareRunProvenance(plan DispatchPlan, policy ReviewersPolicy) (*ReviewRunProvenance, error) {
	if policy.Provenance == nil {
		return nil, nil
	}
	if e := validateRunProvenancePolicy(policy); e != nil {
		return nil, e
	}
	review, _, e := s.loadReviewPolicy()
	if e != nil {
		return nil, e
	}
	if err := s.runProvenanceProjectBinding(review.AuthorityProject); err != nil {
		return nil, err
	}
	if !validGovernanceProjectID(review.AuthorityProject) || strings.TrimSpace(review.AuthorityAudience) == "" {
		return nil, fmt.Errorf("run provenance needs protected authority_project and authority_audience")
	}
	implementation, e := normalizeModelIdentity(policy.Provenance.Implementation)
	if e != nil {
		return nil, e
	}
	reviewer, e := normalizeModelIdentity(ModelIdentity{Principal: plan.Config.Executor, Vendor: plan.Config.Vendor, Model: plan.Config.Model})
	if e != nil {
		return nil, e
	}
	raw, _ := json.Marshal(plan.Config)
	command := plan.Config.Command
	if plan.predecessor != nil {
		command = plan.Config.Resume.Command
	}
	argv, _ := json.Marshal(command)
	return &ReviewRunProvenance{ExecutionState: "not-started", SchemaVersion: 1, Project: review.AuthorityProject, Audience: review.AuthorityAudience, WorktreeDigest: digestBytes([]byte(plan.OriginWorktree)), Trust: policy.Provenance.Trust, Implementation: implementation, Reviewer: reviewer, ConfigDigest: digestBytes(raw), CommandTemplateDigest: digestBytes(argv), TemplateVersion: plan.Brief.TemplateVersion, PlannedWireDigest: digestBytes([]byte(plan.Brief.Text)), NotesDigest: digestBytes([]byte(plan.notes))}, nil
}
func reviewExecutableDigest(path string) (string, error) {
	// Resolve the observed target for hashing; execution retains the selected
	// path spelling so relative and shebang argv0 semantics stay unchanged.
	resolved, e := filepath.EvalSymlinks(path)
	if e != nil {
		return "", e
	}
	info, e := os.Lstat(resolved)
	if e != nil || !info.Mode().IsRegular() || info.Size() > reviewTranscriptMaxBytes {
		return "", fmt.Errorf("provenance executable is not a bounded regular file")
	}
	f, e := os.Open(resolved)
	if e != nil {
		return "", e
	}
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		f.Close()
		return "", fmt.Errorf("executable changed while opening")
	}
	hash := sha256.New()
	n, e := io.Copy(hash, io.LimitReader(f, reviewTranscriptMaxBytes+1))
	closeErr := f.Close()
	if e != nil || closeErr != nil || n > reviewTranscriptMaxBytes {
		return "", fmt.Errorf("executable digest could not be read within bound")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
func reviewRunCanonicalDigest(run ReviewRun) string {
	run.Path, run.SupersededBy = "", ""
	raw, _ := json.Marshal(run)
	return digestBytes(raw)
}
func (s Store) LoadProvenanceReviewRun(id string) (ReviewRun, error) {
	if !regexp.MustCompile(`^rrn-[0-9a-f]{16}$`).MatchString(id) {
		return ReviewRun{}, fmt.Errorf("invalid provenance run id")
	}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil {
		return ReviewRun{}, e
	}
	if len(lost) > 0 {
		return ReviewRun{}, fmt.Errorf("run provenance history is incomplete: %v", lost)
	}
	for _, run := range runs {
		if run.RunID == id {
			if err := s.provenanceAttemptLineage(run, runs); err != nil {
				return ReviewRun{}, err
			}
			return run, nil
		}
	}
	return ReviewRun{}, fmt.Errorf("provenance run not found")
}
func runProofSigningBytes(proof ReviewRunProof) []byte {
	proof.ProofID, proof.Signature = "", ""
	raw, _ := json.Marshal(proof)
	return raw
}
func runProofIdentity(proof ReviewRunProof) string {
	proof.ProofID = ""
	raw, _ := json.Marshal(proof)
	return "rrp-" + strings.TrimPrefix(digestBytes(raw), "sha256:")
}
func runProofPath(id, proof string) (string, error) {
	if !regexp.MustCompile(`^rrn-[0-9a-f]{16}$`).MatchString(id) || !regexp.MustCompile(`^rrp-[0-9a-f]{64}$`).MatchString(proof) {
		return "", fmt.Errorf("invalid run proof reference")
	}
	return filepath.ToSlash(filepath.Join(reviewRunProofDir, id, proof+".json")), nil
}

// SignReviewRunProof reuses a loaded native issuer; no key or run is rewritten.
// It signs a captured executor statement, never inventing evidence for legacy runs.
func (s Store) SignReviewRunProof(id string, key IssuerKey, ttl time.Duration, now time.Time, apply bool) (ReviewRunProof, error) {
	run, e := s.LoadProvenanceReviewRun(id)
	if e != nil {
		return ReviewRunProof{}, e
	}
	if run.Provenance == nil {
		return ReviewRunProof{}, fmt.Errorf("legacy run has no captured provenance; it cannot be reconstructed")
	}
	if key.private == nil {
		return ReviewRunProof{}, fmt.Errorf("issuer has no loaded native key")
	}
	policy, _, e := s.loadReviewPolicy()
	if e != nil {
		return ReviewRunProof{}, e
	}
	if !reviewIssuerHoldsGrant(policy.TrustedAttestationIssuers, key.Name, key.PublicKey()) {
		return ReviewRunProof{}, fmt.Errorf("run issuer is not pinned for attestations")
	}
	if ttl < 0 || ttl > 365*24*time.Hour {
		return ReviewRunProof{}, fmt.Errorf("run proof TTL must be nonnegative and at most 365 days")
	}
	if why := s.runProofContextConflict(run); why != "" {
		return ReviewRunProof{}, fmt.Errorf("run proof cannot govern: %s", why)
	}
	snapshot, _ := json.Marshal(run.Provenance)
	claim := ReviewRunProofClaim{SchemaVersion: 1, Project: run.Provenance.Project, Audience: run.Provenance.Audience, RunID: run.RunID, RecordDigest: reviewRunCanonicalDigest(run), BundleDigest: run.BundleDigest, Scope: run.Scope, Kind: run.Kind, AttemptID: run.AttemptID, Resumes: run.Resumes, TranscriptDigest: run.TranscriptDigest, OutputDigest: digestBytes([]byte(run.Output)), ProvenanceDigest: digestBytes(snapshot), IssuedAt: now.UTC().Format(time.RFC3339Nano)}
	if ttl > 0 {
		claim.ExpiresAt = now.Add(ttl).UTC().Format(time.RFC3339Nano)
	}
	proof := ReviewRunProof{SchemaVersion: 1, Issuer: key.Name, Algorithm: "ed25519", PublicKey: key.PublicKey(), Claim: claim}
	proof.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key.private, runProofSigningBytes(proof)))
	proof.ProofID = runProofIdentity(proof)
	if _, e = s.verifyRunProof(proof, run, now); e != nil {
		return ReviewRunProof{}, e
	}
	if apply {
		rel, e := runProofPath(run.RunID, proof.ProofID)
		if e != nil {
			return ReviewRunProof{}, e
		}
		if e = publishLedgerJSON(s.Root, rel, proof); e != nil {
			return ReviewRunProof{}, e
		}
	}
	return proof, nil
}

func (s Store) VerifyReviewRunProof(id, proofID string) (ReviewRunAssurance, error) {
	s = s.reviewOperation()
	run, e := s.LoadProvenanceReviewRun(id)
	if e != nil {
		return ReviewRunAssurance{}, e
	}
	rel, e := runProofPath(id, proofID)
	if e != nil {
		return ReviewRunAssurance{}, e
	}
	if _, e = ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Dir(rel)), false); e != nil {
		return ReviewRunAssurance{}, e
	}
	var proof ReviewRunProof
	if e = readLedgerJSON(filepath.Join(s.Root, rel), &proof); e != nil {
		return ReviewRunAssurance{}, e
	}
	if proof.ProofID != proofID {
		return ReviewRunAssurance{}, fmt.Errorf("run proof filename identity differs")
	}
	return s.verifyRunProof(proof, run, s.reviewClock())
}
func (s Store) verifyRunProof(proof ReviewRunProof, run ReviewRun, now time.Time) (ReviewRunAssurance, error) {
	fail := func(message string) (ReviewRunAssurance, error) {
		return ReviewRunAssurance{}, fmt.Errorf("run provenance: %s", message)
	}
	if proof.SchemaVersion != 1 || proof.Claim.SchemaVersion != 1 || proof.Algorithm != "ed25519" || proof.ProofID != runProofIdentity(proof) {
		return fail("unsupported or rewritten proof")
	}
	policy, _, e := s.loadReviewPolicy()
	if e != nil {
		return ReviewRunAssurance{}, e
	}
	if !reviewIssuerHoldsGrant(policy.TrustedAttestationIssuers, proof.Issuer, proof.PublicKey) {
		return fail("issuer is not trusted under current pins")
	}
	public, e := base64.StdEncoding.DecodeString(proof.PublicKey)
	if e != nil || len(public) != ed25519.PublicKeySize {
		return fail("invalid issuer key")
	}
	signature, e := base64.StdEncoding.DecodeString(proof.Signature)
	if e != nil || !ed25519.Verify(public, runProofSigningBytes(proof), signature) {
		return fail("signature verification failed")
	}
	c, p := proof.Claim, run.Provenance
	if p == nil || p.SchemaVersion != 1 || run.LedgerVersion != 1 || p.Trust != "executor-attested" {
		return fail("captured executor provenance is absent")
	}
	if err := s.runProvenanceProjectBinding(policy.AuthorityProject); err != nil {
		return ReviewRunAssurance{}, err
	}
	if c.Project != policy.AuthorityProject || c.Project != p.Project || c.Audience == "" || c.Audience != policy.AuthorityAudience || c.Audience != p.Audience {
		return fail("project or audience differs from protected policy")
	}
	if c.RunID != run.RunID || c.RecordDigest != reviewRunCanonicalDigest(run) || c.BundleDigest != run.BundleDigest || c.Scope != run.Scope || c.Kind != run.Kind || c.AttemptID == "" || c.AttemptID != run.AttemptID || c.Resumes != run.Resumes || c.TranscriptDigest != run.TranscriptDigest || c.OutputDigest != digestBytes([]byte(run.Output)) {
		return fail("proof does not bind the exact run and output")
	}
	snapshot, _ := json.Marshal(p)
	if c.ProvenanceDigest != digestBytes(snapshot) {
		return fail("captured provenance digest differs")
	}
	times := &ReviewAuthorityClaim{IssuedAt: c.IssuedAt, ExpiresAt: c.ExpiresAt}
	if blocked := reviewClaimTimeBlockers(times, now, 0); len(blocked) > 0 {
		return fail(strings.Join(blocked, "; "))
	}
	end, e := time.Parse(time.RFC3339Nano, run.EndedAt)
	issued, e2 := time.Parse(time.RFC3339Nano, c.IssuedAt)
	if e != nil || e2 != nil || issued.Before(end) {
		return fail("proof predates the ended run")
	}
	if why := s.runProofContextConflict(run); why != "" {
		return fail(why)
	}
	config, found, e := s.LoadReviewersPolicy()
	if e != nil || !found || config.Provenance == nil {
		return fail("current execution policy is missing")
	}
	adapter, ok := config.Adapters[run.Adapter]
	if !ok {
		return fail("current adapter is missing")
	}
	raw, _ := json.Marshal(adapter)
	if p.ConfigDigest != digestBytes(raw) {
		return fail("adapter configuration differs")
	}
	implementation, e := normalizeModelIdentity(config.Provenance.Implementation)
	if e != nil || implementation != p.Implementation || config.Provenance.Trust != p.Trust {
		return fail("implementation baseline differs or is unknown")
	}
	reviewer, e := normalizeModelIdentity(ModelIdentity{Principal: adapter.Executor, Vendor: adapter.Vendor, Model: adapter.Model})
	recorded, recordErr := normalizeModelIdentity(ModelIdentity{Principal: p.Reviewer.Principal, Vendor: run.Vendor, Model: run.Model})
	if e != nil || recordErr != nil || recorded != reviewer || reviewer != p.Reviewer {
		return fail("reviewer identity differs or is unknown")
	}
	origin, e := dispatchWorktreeIdentity(s.Root)
	if e != nil || p.WorktreeDigest != digestBytes([]byte(origin)) {
		return fail("origin worktree identity differs")
	}
	brief, e := s.RenderReviewBrief(run.BundleID, run.Kind, run.BriefNotes)
	if e != nil || brief.Digest != run.BriefDigest || p.PlannedWireDigest != digestBytes([]byte(brief.Text)) || p.TemplateVersion != brief.TemplateVersion || p.NotesDigest != digestBytes([]byte(run.BriefNotes)) {
		return fail("brief/template/notes/transmitted bytes differ")
	}
	command := adapter.Command
	if run.Resumes != "" {
		if adapter.Resume == nil {
			return fail("resume configuration missing")
		}
		command = adapter.Resume.Command
	}
	raw, _ = json.Marshal(command)
	if p.CommandTemplateDigest != digestBytes(raw) {
		return fail("command template differs")
	}
	if p.WireBytes < 0 || p.WireBytes > int64(len(brief.Text)) {
		return fail("stdin byte count is outside the configured brief")
	}
	switch p.ExecutionState {
	case "started":
		if !validProofDigest(p.ArgvDigest) || !validProofDigest(p.ExecutableDigest) || !validProofDigest(p.RawTranscriptDigest) || !validProofDigest(p.RawOutputDigest) {
			return fail("execution or original-byte digests missing")
		}
		if p.WireDigest != digestBytes([]byte(brief.Text)[:int(p.WireBytes)]) {
			return fail("actual stdin digest differs from transmitted brief prefix")
		}
		if p.WireComplete && (p.WireBytes != int64(len(brief.Text)) || p.WireDigest != p.PlannedWireDigest) {
			return fail("complete stdin claim is inconsistent")
		}
	case "not-started":
		if p.WireBytes != 0 || p.WireComplete || p.WireDigest != "" || run.Status == ReviewRunCompleted || run.Status == ReviewRunInterrupted {
			return fail("unstarted audit claims an executed or transmitted review")
		}
	default:
		return fail("unknown execution observation")
	}
	if run.Status == ReviewRunCompleted && (!p.WireComplete || !p.OutputComplete || strings.TrimSpace(run.Output) == "") {
		return fail("truncated or incomplete draft cannot establish completed review")
	}
	decision := draftReviewDecision(run.Output)
	return ReviewRunAssurance{DraftDecision: decision, ProofIssuedAt: c.IssuedAt, ProofExpiresAt: c.ExpiresAt, State: run.Status, CompleteReview: run.Status == ReviewRunCompleted && p.WireComplete && p.OutputComplete && decision != "unknown", Assurance: ReviewRunMachineAttested, RunID: run.RunID, ProofID: proof.ProofID, Issuer: proof.Issuer, Reviewer: reviewer, Implementation: implementation, IdentitySource: "native-issuer-signed-executor-declaration", VendorDifference: reviewer.Vendor != implementation.Vendor, ModelDifference: reviewer.Model != implementation.Model, OutputComplete: p.OutputComplete, Limitations: []string{"executor-declaration-not-independent-vendor-receipt", "raw-bytes-hashed-by-executor-redacted-artifacts-retained", ReviewLimitationIssuerTruthfulness, ReviewLimitationCognitive}}, nil
}
func validProofDigest(value string) bool {
	return regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(value)
}
func (s Store) verifyRunReviewReference(bundle ReviewBundle, att ReviewAttestation) (ReviewRunAssurance, error) {
	if att.Run == nil {
		return ReviewRunAssurance{}, fmt.Errorf("signed run reference is missing")
	}
	assurance, e := s.VerifyReviewRunProof(att.Run.RunID, att.Run.ProofID)
	if e != nil {
		return assurance, e
	}
	run, e := s.LoadProvenanceReviewRun(att.Run.RunID)
	if e != nil {
		return assurance, e
	}
	if run.BundleID != bundle.BundleID || run.BundleDigest != bundle.BundleDigest || run.Scope != bundle.Payload.Scope.Ref || run.Kind != ReviewBriefKindReview || run.Status != ReviewRunCompleted || !assurance.CompleteReview || att.Reviewer != assurance.Reviewer.Principal {
		return assurance, fmt.Errorf("attestation and referenced run context differ")
	}
	attested, e1 := time.Parse(time.RFC3339Nano, att.AttestedAt)
	ended, e2 := time.Parse(time.RFC3339Nano, run.EndedAt)
	issued, e3 := time.Parse(time.RFC3339Nano, assurance.ProofIssuedAt)
	if e1 != nil || e2 != nil || e3 != nil || ended.After(attested.Add(reviewTimestampPrecision(att.AttestedAt))) || issued.After(attested.Add(reviewTimestampPrecision(att.AttestedAt))) {
		return assurance, fmt.Errorf("run-backed attestation predates its completed run or authenticated run proof")
	}
	draft := draftReviewDecision(run.Output)
	approval := att.Decision == "approved" || att.Decision == "approved-with-reservations"
	if approval && draft != "approved" {
		return assurance, fmt.Errorf("run draft does not support an approval: decision is %s", draft)
	}
	if !approval && (draft != "rejected" && draft != "changes-requested") {
		return assurance, fmt.Errorf("run draft does not support a rejecting attestation: decision is %s", draft)
	}
	if att.Authority == nil || att.Authority.Role != "agent" || att.Authority.ReviewExecution != run.RunID || att.Authority.ImplementationPrincipal != assurance.Implementation.Principal || att.Authority.ImplementationExecution != assurance.Implementation.Execution {
		return assurance, fmt.Errorf("attestation authority differs from run provenance")
	}
	if assurance.Implementation.Execution == run.RunID {
		return assurance, fmt.Errorf("implementation and review executions are the same")
	}
	if bundle.Payload.Plan.Independence == "different-actor" && !assurance.VendorDifference && !assurance.ModelDifference {
		return assurance, fmt.Errorf("reviewer vendor and model both match the implementation")
	}
	if bundle.Payload.Plan.Independence == "mandatory-human" {
		return assurance, fmt.Errorf("machine-attested run cannot satisfy mandatory-human")
	}
	return assurance, nil
}
func (s Store) runBackedReviewBlockers(bundle ReviewBundle, att ReviewAttestation) []string {
	blockers := s.verifiedAuthorityBlockers(bundle, att)
	if _, e := s.verifyRunReviewReference(bundle, att); e != nil {
		blockers = append(blockers, e.Error())
	}
	return uniqueSorted(blockers)
}

func (s Store) runProofContextConflict(run ReviewRun) string {
	if run.Status == ReviewRunCompleted {
		return s.ReviewRunStillGoverns(run)
	}
	if why := s.reviewRunTranscriptIntact(run); why != "" {
		return why
	}
	bundle, e := s.LoadReviewBundle(run.BundleID)
	if e != nil {
		return e.Error()
	}
	if bundle.BundleDigest != run.BundleDigest || bundle.Payload.Scope.Ref != run.Scope || bundle.Payload.Subject.Head != run.SealedCommit {
		return "run names conflicting sealed inputs"
	}
	return s.dispatchNoLongerGoverns(DispatchPlan{Bundle: bundle, Brief: ReviewBrief{Scope: run.Scope}, SealedCommit: run.SealedCommit, PolicyDigest: run.PolicyDigest})
}

func (s Store) provenanceAttemptLineage(run ReviewRun, runs []ReviewRun) error {
	if run.AttemptID == "" {
		return fmt.Errorf("run provenance has no reserved attempt")
	}
	attempts, e := s.readDelegatedReviewAttempts(run.BundleID)
	if e != nil {
		return e
	}
	var rootReceipt *reviewAttemptReceipt
	budget := int64(32 << 20)
	for _, attempt := range attempts {
		receipt, e := s.readAttemptReceipt(attempt, &budget)
		if e != nil {
			return fmt.Errorf("run provenance attempt history incomplete: %w", e)
		}
		if attempt.AttemptID == run.AttemptID {
			rootReceipt = &receipt
		}
	}
	if rootReceipt == nil {
		return fmt.Errorf("run provenance attempt reservation missing")
	}
	byID := map[string]ReviewRun{}
	for _, r := range runs {
		byID[r.RunID] = r
	}
	current := run
	seen := map[string]bool{}
	for current.Resumes != "" {
		if seen[current.RunID] {
			return fmt.Errorf("cyclic provenance continuation")
		}
		seen[current.RunID] = true
		parent, ok := byID[current.Resumes]
		if !ok || parent.AttemptID != run.AttemptID {
			return fmt.Errorf("provenance predecessor missing or foreign")
		}
		var receipt reviewAttemptReceipt
		if e := readLedgerJSONBudget(filepath.Join(s.Root, continuationPath(parent.RunID, true)), &receipt, &budget); e != nil {
			return e
		}
		if receipt.RunID != current.RunID || receipt.AttemptID != run.AttemptID || receipt.RunDigest != reviewRunCanonicalDigest(current) {
			return fmt.Errorf("provenance continuation receipt does not bind its endpoint")
		}
		current = parent
	}
	if rootReceipt.RunID != current.RunID || rootReceipt.RunDigest != reviewRunCanonicalDigest(current) {
		return fmt.Errorf("provenance attempt receipt does not bind the original endpoint")
	}
	return nil
}

// runProvenanceCommand hashes only bytes accepted by the child's stdin pipe.
// The configured brief is a separate digest; partial writes never claim full
// transmission, and no receipt claims that the vendor actually consumed it.
func runProvenanceCommand(cmd *exec.Cmd, text string) (digest string, count int64, complete, started bool, runErr error) {
	cmd.Stdin = nil
	stdin, e := cmd.StdinPipe()
	if e != nil {
		return "", 0, false, false, e
	}
	if e = cmd.Start(); e != nil {
		stdin.Close()
		return "", 0, false, false, e
	}
	type result struct {
		digest string
		count  int64
		err    error
	}
	done := make(chan result, 1)
	go func() {
		hash := sha256.New()
		total := int64(0)
		writer := provenanceStdinWriter{write: stdin, record: func(p []byte) { hash.Write(p); total += int64(len(p)) }}
		_, copyErr := io.Copy(writer, strings.NewReader(text))
		closeErr := stdin.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		done <- result{"sha256:" + hex.EncodeToString(hash.Sum(nil)), total, copyErr}
	}()
	runErr = cmd.Wait()
	input := <-done
	if runErr == nil && input.err != nil {
		runErr = fmt.Errorf("review brief delivery incomplete: %w", input.err)
	}
	return input.digest, input.count, input.err == nil && input.count == int64(len(text)), true, runErr
}

type provenanceStdinWriter struct {
	write  io.Writer
	record func([]byte)
}

func (w provenanceStdinWriter) Write(p []byte) (int, error) {
	n, e := w.write.Write(p)
	if n > 0 {
		w.record(p[:n])
	}
	return n, e
}

func (s Store) runProvenanceProjectBinding(project string) error {
	if _, err := os.Lstat(filepath.Join(s.Root, ".pose/project.json")); !os.IsNotExist(err) {
		if _, err := dispatchRegularInputPath(s.Root, ".pose/project.json"); err != nil {
			return err
		}
	}
	declared, found, err := ReadProjectFileIn(s.fsys(), s.Root)
	if err != nil {
		return err
	}
	if found && declared != project {
		return fmt.Errorf("run provenance protected authority_project differs from declared project identity")
	}
	return nil
}

func resolveProvenanceExecutable(command, worktree string) (string, error) {
	_, hashed, e := provenanceExecutableSelection(command, worktree)
	return hashed, e
}

// Keep the exec path's original relative/symlink spelling: kernels use that
// spelling as a shebang script's $0. Resolve only the hash observation.
func provenanceExecutableSelection(command, worktree string) (selected, hashed string, err error) {
	selected = command
	observed := command
	if !filepath.IsAbs(command) {
		if strings.ContainsAny(command, "/\\") {
			observed = filepath.Join(worktree, command)
		} else {
			selected, err = exec.LookPath(command)
			if err != nil {
				return "", "", err
			}
			observed = selected
		}
	}
	hashed, err = filepath.EvalSymlinks(observed)
	if err != nil {
		return "", "", err
	}
	hashed, err = filepath.Abs(hashed)
	return selected, hashed, err
}
