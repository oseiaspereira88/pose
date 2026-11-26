//go:build !unix

package pose

import "os"

// openIndexNoBlock opens path for reading; there are no FIFOs at a path here.
func openIndexNoBlock(path string) (*os.File, error) {
	return os.Open(path)
}
