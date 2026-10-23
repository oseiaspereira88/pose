//go:build !linux && !darwin

package pose

import "os"

// observedWriteAccess cannot ask this platform in advance; no refusal is
// predicted, and the write itself is what fails.
func observedWriteAccess(path string, dir bool) error { return nil }

func observedUmask() (os.FileMode, bool)                                  { return 0, false }
func physicalLinkCount(info os.FileInfo) (uint64, bool)                   { return 0, false }
func stagedOwnerAccessKnown() bool                                        { return false }
func observedReadWriteAccess(path string, need os.FileMode) (bool, error) { return false, nil }
