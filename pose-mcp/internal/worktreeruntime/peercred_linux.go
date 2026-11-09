//go:build linux

package worktreeruntime

import (
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// Supported reports whether this platform has a protected local endpoint.
func Supported() bool { return true }

// peerUID is the uid the operating system reports for the other end of the socket.
func peerUID(c *net.UnixConn) (uint32, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, err
	}
	var uid uint32
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		cred, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if e != nil {
			sockErr = e
			return
		}
		uid = cred.Uid
	}); err != nil {
		return 0, err
	}
	return uid, sockErr
}

func fileOwnerUID(info os.FileInfo) (uint32, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Uid, true
}

// lockEndpoint takes the exclusive lock of an endpoint for the life of the runtime.
func lockEndpoint(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, ErrAlreadyServed
	}
	return func() { file.Close() }, nil
}
