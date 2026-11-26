//go:build unix

package pose

import (
	"os"
	"syscall"
)

// openIndexNoBlock opens path for reading without blocking on a FIFO.
func openIndexNoBlock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
