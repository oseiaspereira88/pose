package pose

// Draft check (spec pose-review-readiness-before-seal R3): the reasons the
// conclusion step and the attestation preflight would refuse a reviewer's
// draft, computed before the run ends and without writing anything. It
// reuses the same parsers and content checks; only the run's own proof,
// signature and identity, which do not exist yet, are left out.
//
// A sealed checkout holds the subject, not the bundle sealed over it nor the
// run ledger, so the dispatch writes both to the run's private directory and
// names it in ReviewDraftContextEnv.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReviewDraftContextEnv names the file holding the bundle and round a draft
// is checked against inside a delegated review.
const ReviewDraftContextEnv = "POSE_REVIEW_DRAFT_CONTEXT"

// ReviewCLIEnv names the executable of the engine that dispatched a review,
// so the reviewer checks its draft with it and not with another installed
// `pose` (review round 1).
const ReviewCLIEnv = "POSE_REVIEW_CLI"

const reviewDraftContextSchema = "pose.review-draft-context/v1"

// ReviewDraftContext is what a draft is judged against: the sealed bundle and
// the findings its round must account for.
type ReviewDraftContext struct {
	Schema string              `json:"schema"`
	Bundle ReviewBundle        `json:"bundle"`
	Round  int                 `json:"round"`
	Open   []ReviewOpenFinding `json:"open,omitempty"`
}

// ReviewDraftContextFor builds the context of bundle from the run ledger.
func (s Store) ReviewDraftContextFor(bundle ReviewBundle) (ReviewDraftContext, error) {
	round, err := s.ReviewRoundOf(bundle)
	if err != nil {
		return ReviewDraftContext{}, err
	}
	return ReviewDraftContext{Schema: reviewDraftContextSchema, Bundle: bundle, Round: round.Number, Open: round.Open}, nil
}

// writeReviewDraftContext writes ctx under dir and returns its path.
func writeReviewDraftContext(dir string, ctx ReviewDraftContext) (string, error) {
	raw, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "pose-review-draft-context.json")
	return path, os.WriteFile(path, raw, 0o600)
}

// ReadReviewDraftContext reads a context file a dispatch wrote.
func ReadReviewDraftContext(path string) (ReviewDraftContext, error) {
	var ctx ReviewDraftContext
	raw, err := os.ReadFile(path)
	if err != nil {
		return ctx, err
	}
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return ctx, fmt.Errorf("pose: reading draft context %s: %w", path, err)
	}
	if ctx.Schema != reviewDraftContextSchema || ctx.Bundle.BundleID == "" {
		return ctx, fmt.Errorf("pose: %s is not a review draft context", path)
	}
	return ctx, nil
}

// CheckReviewDraft returns every reason a draft for ref would be refused at
// conclusion, or nil when none applies. ref is a bundle id, or a run id whose
// bundle is used.
func (s Store) CheckReviewDraft(ref, output string, now time.Time) ([]string, error) {
	bundleID := ref
	if strings.HasPrefix(ref, "rrn-") {
		run, err := s.LoadProvenanceReviewRun(ref)
		if err != nil {
			return nil, err
		}
		bundleID = run.BundleID
	}
	bundle, err := s.LoadReviewBundle(bundleID)
	if err != nil {
		return nil, err
	}
	ctx, err := s.ReviewDraftContextFor(bundle)
	if err != nil {
		return nil, err
	}
	return s.CheckReviewDraftIn(ctx, output, now), nil
}

// CheckReviewDraftIn judges output against ctx.
func (s Store) CheckReviewDraftIn(ctx ReviewDraftContext, output string, now time.Time) []string {
	var reasons []string
	draft, err := ParseReviewDraft(output)
	switch {
	case err != nil:
		reasons = append(reasons, "the conclusion cannot be read: "+err.Error())
	case draft == nil:
		reasons = append(reasons, "the final message carries no pose-draft block")
	}
	if err == nil {
		round := ReviewRound{Number: ctx.Round, Open: ctx.Open}
		if problems := ValidateRoundDraft(round, output); len(problems) > 0 {
			reasons = append(reasons, "the draft does not account for its round: "+strings.Join(problems, "; "))
		}
	}
	if draft != nil && reviewDecisionApproves(draft.Decision) {
		bundle := ctx.Bundle
		run := ReviewRun{RunID: "draft-check", BundleID: bundle.BundleID, Kind: ReviewBriefKindReview}
		att := reviewAttestationFromDraft(run, bundle, *draft, "agent:draft-check/unknown")
		completed, err := completeReviewAttestation(bundle, att, now)
		if err != nil {
			reasons = append(reasons, "the attestation preflight refuses it: "+strings.TrimPrefix(err.Error(), "pose: "))
		} else if blockers := s.attestationContentBlockers(bundle, completed, false); len(blockers) > 0 {
			reasons = append(reasons, "the attestation preflight refuses it: "+strings.Join(uniqueSorted(blockers), "; "))
		}
	}
	return reasons
}
