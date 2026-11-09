//go:build !linux && !darwin

package worktreeruntime

import (
	"errors"
	"net"
	"os"
)

// Supported is false where the repository has no protected local endpoint (Windows has no
// named-pipe implementation here): every command runs standalone and the runtime says so.
func Supported() bool { return false }

func peerUID(*net.UnixConn) (uint32, error) {
	return 0, errors.New("no peer credentials on this platform")
}

func fileOwnerUID(os.FileInfo) (uint32, bool) { return 0, false }

func lockEndpoint(string) (func(), error) { return func() {}, ErrUnsupported }
