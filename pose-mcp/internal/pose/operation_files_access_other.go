//go:build !linux && !darwin

package pose

import "os"

// observedWriteAccess cannot ask this platform in advance; no refusal is
// predicted, and the write itself is what fails.
func observedWriteAccess(path string, dir bool) error { return nil }

func observedUmask() os.FileMode { return 0 }
