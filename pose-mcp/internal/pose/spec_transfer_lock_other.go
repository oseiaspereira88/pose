//go:build !unix && !windows

package pose

import "os"

func lockSpecTransferFile(_ *os.File) (func(), error) {
	return nil, errFileLockUnsupported
}
