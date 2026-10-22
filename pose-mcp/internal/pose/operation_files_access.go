//go:build linux || darwin

package pose

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// observedWriteAccess asks the kernel whether the effective user may write
// path now (and search it, for a directory): access(2) with AT_EACCESS sees
// read-only mounts and, where the kernel implements it, ACLs, which
// permission bits do not. Only a refusal is returned; an answer the platform
// cannot give is not one.
func observedWriteAccess(path string, dir bool) error {
	mode := uint32(unix.W_OK)
	if dir {
		mode |= unix.X_OK
	}
	err := unix.Faccessat(unix.AT_FDCWD, path, mode, unix.AT_EACCESS)
	if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EROFS) || errors.Is(err, unix.EPERM) {
		return err
	}
	return nil
}

// observedUmask reads the process umask where the kernel reports it, without
// setting it: umask(2) cannot be read without being changed, process-wide.
func observedUmask() os.FileMode {
	raw, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if value, ok := strings.CutPrefix(line, "Umask:"); ok {
				if mask, err := strconv.ParseUint(strings.TrimSpace(value), 8, 32); err == nil {
					return os.FileMode(mask)
				}
			}
		}
	}
	return 0o022
}
