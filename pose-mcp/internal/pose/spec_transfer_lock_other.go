//go:build !unix && !windows

package pose

import (
	"errors"
	"os"
)

func lockSpecTransferFile(_ *os.File) (func(), error) {
	return nil, errors.New("transfer locking unsupported on this platform")
}
