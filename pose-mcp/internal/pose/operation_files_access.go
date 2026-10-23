//go:build linux || darwin

package pose

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

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
func observedUmask() (os.FileMode, bool) {
	if runtime.GOOS != "linux" {
		return 0, false
	}
	raw, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if value, ok := strings.CutPrefix(line, "Umask:"); ok {
				if mask, err := strconv.ParseUint(strings.TrimSpace(value), 8, 32); err == nil {
					return os.FileMode(mask), true
				}
			}
		}
	}
	return 0, false
}

// physicalLinkCount reads inode metadata without enumerating aliases.
func physicalLinkCount(info os.FileInfo) (uint64, bool) {
	if info == nil {
		return 0, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(stat.Nlink), true
}

func stagedOwnerAccessKnown() bool { return true }

// IsRegular also includes procfs attributes backed by kernel handlers.
// Only observed ordinary storage providers support staging ordinary bytes.
func observedOrdinaryFileData(path string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	var info unix.Statfs_t
	if unix.Statfs(path, &info) != nil {
		return false
	}
	// Linux magic values: ext, xfs, btrfs, tmpfs, overlay, nfs, f2fs, ramfs.
	// Unknown and virtual providers remain unsupported.
	switch uint64(info.Type) {
	case 0xef53, 0x58465342, 0x9123683e, 0x01021994, 0x794c7630, 0x6969, 0xf2f52010, 0x858458f6:
		return true
	}
	return false
}

func observedCreationMode(path string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	n, err := unix.Getxattr(path, "system.posix_acl_default", nil)
	return (err == nil && n == 0) || errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOTSUP)
}

func observedReadWriteAccess(path string, need os.FileMode) (bool, error) {
	mode := uint32(0)
	if need&0400 != 0 {
		mode |= unix.R_OK
	}
	if need&0200 != 0 {
		mode |= unix.W_OK
	}
	if need&0100 != 0 {
		mode |= unix.X_OK
	}
	err := unix.Faccessat(unix.AT_FDCWD, path, mode, unix.AT_EACCESS)
	if err == nil || errors.Is(err, unix.EACCES) || errors.Is(err, unix.EROFS) || errors.Is(err, unix.EPERM) {
		return true, err
	}
	return false, err
}
