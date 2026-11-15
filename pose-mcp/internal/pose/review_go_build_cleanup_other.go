//go:build !unix

package pose

import "os"

// removeReviewGoBuildDir has no descriptor-relative removal on this
// platform: the go-build directory is left to the runtime inventory, which
// keeps the cleanup incomplete when it exceeds its bounds (spec
// pose-review-cleanup-removes-go-build-leftovers).
func removeReviewGoBuildDir(*os.Root, string) error { return nil }

var (
	_ = reviewGoBuildBeforeOpen
	_ = reviewGoBuildAfterContents
	_ = reviewGoBuildBeforeRemove
)
