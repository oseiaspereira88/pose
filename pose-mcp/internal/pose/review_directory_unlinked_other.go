//go:build !unix

package pose

import "os"

// reviewDirectoryUnlinked cannot prove an unlink without a link count, so a
// platform without one keeps the cleanup incomplete (spec
// pose-review-cleanup-removes-go-build-leftovers).
func reviewDirectoryUnlinked(os.FileInfo) bool { return false }
