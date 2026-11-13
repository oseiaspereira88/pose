package pose

// Closeout receipts (spec pose-closeout-review-policy-by-cost-benefit R4, R7).
// Under an adopted closeout review policy, the explicit `pose close` of a
// milestone or a roadmap publishes a receipt: the terminal marker of that
// close, with the evaluator's decision and the policy it was made under. A
// receipt is written once and never reinterpreted: a later policy change does
// not re-evaluate it, and a different or corrupt receipt is never overwritten.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CloseoutReceiptSchemaVersion = 1

type CloseoutReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	Scope         string `json:"scope"`
	ClosedAt      string `json:"closed_at"`
	// Review is the literal `required` or `not-required`.
	Review        string                 `json:"review"`
	Decision      CloseoutReviewDecision `json:"decision"`
	PolicyDigest  string                 `json:"policy_digest"`
	SubjectDigest string                 `json:"subject_digest"`
	// ExitGateDigest binds a milestone's exit gate as it was closed.
	ExitGateDigest string `json:"exit_gate_digest,omitempty"`
	// ReviewBundle and ReviewAttestation name the verified review the close
	// stood on: required, or made by choice when not required.
	ReviewBundle      string `json:"review_bundle,omitempty"`
	ReviewAttestation string `json:"review_attestation,omitempty"`
	ApprovalOrigin    string `json:"approval_origin,omitempty"`
}

// CloseoutReceiptPath is where the receipt of an aggregate scope lives.
func (s Store) CloseoutReceiptPath(ref string) (string, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return "", err
	}
	base := filepath.Join(s.Root, ".pose", "review-closeouts")
	switch scope.Kind {
	case "milestone":
		if ValidateSlug(scope.Roadmap) != nil || !filepath.IsLocal(scope.Milestone) || filepath.Base(scope.Milestone) != scope.Milestone {
			return "", fmt.Errorf("pose: invalid milestone scope %s", ref)
		}
		return filepath.Join(base, "milestone", scope.Roadmap, scope.Milestone+".json"), nil
	case "roadmap":
		if ValidateSlug(scope.Slug) != nil {
			return "", fmt.Errorf("pose: invalid roadmap scope %s", ref)
		}
		return filepath.Join(base, "roadmap", scope.Slug+".json"), nil
	}
	return "", fmt.Errorf("pose: %s has no closeout receipt; only milestones and roadmaps do", ref)
}

// ReadCloseoutReceipt returns the receipt of a scope, nil when there is none.
// A receipt that cannot be read or does not validate is an error, never an
// absence: it would otherwise reopen a closed scope or hide a corruption.
func (s Store) ReadCloseoutReceipt(ref string) (*CloseoutReceipt, error) {
	path, err := s.CloseoutReceiptPath(ref)
	if err != nil {
		return nil, err
	}
	info, err := s.fsys().Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pose: closeout receipt of %s: %w", ref, err)
	}
	if !info.Mode().IsRegular() || !artifactPathWithin(s.fsys(), s.Root, path) {
		return nil, fmt.Errorf("pose: closeout receipt of %s is not a regular file of this project", ref)
	}
	raw, err := s.fsys().ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: closeout receipt of %s: %w", ref, err)
	}
	receipt, err := parseCloseoutReceipt(raw)
	if err != nil {
		return nil, fmt.Errorf("pose: closeout receipt of %s: %w", ref, err)
	}
	if receipt.Scope != ref || receipt.Decision.Scope != ref {
		return nil, fmt.Errorf("pose: closeout receipt at %s names %s, not %s", path, receipt.Scope, ref)
	}
	return &receipt, nil
}

func parseCloseoutReceipt(raw []byte) (CloseoutReceipt, error) {
	var receipt CloseoutReceipt
	if err := rejectDuplicateJSONKeysAndControls(raw); err != nil {
		return receipt, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return receipt, errors.New("trailing content")
	}
	if receipt.SchemaVersion != CloseoutReceiptSchemaVersion {
		return receipt, fmt.Errorf("unsupported schema_version %d", receipt.SchemaVersion)
	}
	if _, valid := reviewInstant(receipt.ClosedAt); !valid {
		return receipt, fmt.Errorf("closed_at %q is not an instant", receipt.ClosedAt)
	}
	d := receipt.Decision
	if !d.Adopted || d.EvaluatorVersion != CloseoutReviewEvaluatorVersion || !validCloseoutReviewValue(d.PolicyValue) || d.Reason == "" {
		return receipt, errors.New("decision is not one of " + CloseoutReviewEvaluatorVersion)
	}
	if d.PolicyValue == CloseoutReviewNever && d.Required && d.Reason == "" || d.PolicyValue == CloseoutReviewAlways && !d.Required {
		return receipt, fmt.Errorf("a %s decision cannot be required=%v", d.PolicyValue, d.Required)
	}
	if receipt.Review != d.Review() {
		return receipt, fmt.Errorf("review %q contradicts the recorded decision (%s)", receipt.Review, d.Review())
	}
	if d.Required && receipt.ReviewBundle == "" && receipt.ApprovalOrigin == "" {
		return receipt, errors.New("a required review names no verified review")
	}
	if !d.Known {
		return receipt, errors.New("a close cannot stand on unknown members or associations")
	}
	if receipt.PolicyDigest == "" || receipt.SubjectDigest == "" {
		return receipt, errors.New("policy_digest and subject_digest are required")
	}
	return receipt, nil
}

// NewCloseoutReceipt builds the receipt of a close from the evaluator's
// decision and the verification it stood on. It writes nothing.
func (s Store) NewCloseoutReceipt(ref string, decision CloseoutReviewDecision, verification ReviewEvaluation, subjectDigest string, now time.Time) (CloseoutReceipt, error) {
	_, raw, err := s.loadReviewPolicy()
	if err != nil {
		return CloseoutReceipt{}, err
	}
	_, gate, _, err := s.closeoutSubject(ref)
	if err != nil {
		return CloseoutReceipt{}, err
	}
	receipt := CloseoutReceipt{
		ExitGateDigest: gate,
		SchemaVersion:  CloseoutReceiptSchemaVersion, Scope: ref, ClosedAt: now.UTC().Format(time.RFC3339),
		Review: decision.Review(), Decision: decision, PolicyDigest: "sha256:" + digestHex(raw), SubjectDigest: subjectDigest,
		ReviewBundle: verification.BundleID, ReviewAttestation: verification.AttestationID,
	}
	if receipt.ReviewBundle == "" && verification.Current != nil {
		receipt.ApprovalOrigin = "review:" + verification.Current.ReviewID
	}
	if receipt.ReviewBundle == "" && receipt.ApprovalOrigin == "" {
		receipt.ApprovalOrigin = verification.ApprovalOrigin
	}
	if _, err := parseCloseoutReceipt(CanonicalJSON(receipt)); err != nil {
		return CloseoutReceipt{}, fmt.Errorf("pose: the closeout receipt of %s would be refused: %w", ref, err)
	}
	return receipt, nil
}

// ErrCloseoutReceiptExists reports a receipt already published for the scope.
var ErrCloseoutReceiptExists = errors.New("pose: a closeout receipt already exists")

// PublishCloseoutReceipt publishes a receipt where none exists. It never
// replaces one: an equal receipt is the caller's to verify, a different one
// stays as it is.
func (s Store) PublishCloseoutReceipt(receipt CloseoutReceipt) error {
	path, err := s.CloseoutReceiptPath(receipt.Scope)
	if err != nil {
		return err
	}
	if existing, err := s.ReadCloseoutReceipt(receipt.Scope); err != nil {
		return err
	} else if existing != nil {
		return ErrCloseoutReceiptExists
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := s.fsys().MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	err = s.fsys().Publish(path, append(raw, '\n'), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAbsent})
	if errors.Is(err, ErrPublishConflict) {
		return ErrCloseoutReceiptExists
	}
	return err
}

// A roadmap's close publishes its status and then its receipt. Between the
// two an intent stands, so a close interrupted there reads as incomplete,
// never as a roadmap closed before adoption (spec
// pose-closeout-review-policy-by-cost-benefit R4, R7).
func (s Store) closeoutIntentPath(ref string) (string, error) {
	path, err := s.CloseoutReceiptPath(ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(path, ".json") + ".intent.json", nil
}

// closeoutIntentPending reports an intent on file. What cannot be read is an
// error, which the caller treats as pending: it never proves history.
func (s Store) closeoutIntentPending(ref string) (bool, error) {
	path, err := s.closeoutIntentPath(ref)
	if err != nil {
		return false, err
	}
	if _, err := s.fsys().Lstat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// PublishCloseoutIntent records that this engine is closing ref under the
// adopted policy, before the roadmap's status is published.
func (s Store) PublishCloseoutIntent(ref string, decision CloseoutReviewDecision, now time.Time) error {
	path, err := s.closeoutIntentPath(ref)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(map[string]any{"schema_version": 1, "scope": ref, "started_at": now.UTC().Format(time.RFC3339), "review": decision.Review()}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.fsys().MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return s.fsys().Publish(path, append(raw, '\n'), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny})
}

// ClearCloseoutIntent removes the intent once the receipt stands.
func (s Store) ClearCloseoutIntent(ref string) error {
	path, err := s.closeoutIntentPath(ref)
	if err != nil {
		return err
	}
	if err := s.fsys().Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
