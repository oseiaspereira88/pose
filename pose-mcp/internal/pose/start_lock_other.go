//go:build !linux

package pose

// identifyStartLockProcess records only what every platform has: host and
// PID, set by the caller.
func identifyStartLockProcess(*StartLockOwner) {}

// probeStartLockOwner cannot prove an owner dead without the boot and
// process start identity Linux exposes, so a lock held by another call is
// always owner-unknown here and is removed only by a person (R7 declares
// the platform limit).
func probeStartLockOwner(owner StartLockOwner) (string, string) {
	return LockOwnerUnknown, "the liveness of a lock owner can be checked only on Linux; confirm no start is running before removing the lock"
}
