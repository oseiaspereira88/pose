package pose

// Assessments in a sealed review checkout (spec
// pose-assess-leaves-sealed-checkouts-alone): a reviewer running a planned
// assessment tool must not change the checkout it reviews, or its own run
// fails. The CLI does not persist there; these writers refuse as the last
// line, so no other caller can publish an assessment into it either.

import "errors"

// ErrSealedAssessmentWrite is returned by an assessment writer whose root is
// the sealed checkout of a delegated review.
var ErrSealedAssessmentWrite = errors.New("pose: assessments are not persisted in a sealed review checkout; run the command with --no-persist")

func refuseSealedAssessmentWrite(root string) error {
	if IsSealedCheckout(root) {
		return ErrSealedAssessmentWrite
	}
	return nil
}
