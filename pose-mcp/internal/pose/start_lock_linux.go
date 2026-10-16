//go:build linux

package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// identifyStartLockProcess records the machine, boot, PID namespace and
// process start time of the current process.
func identifyStartLockProcess(owner *StartLockOwner) {
	owner.MachineID = startLockMachineID()
	owner.BootID = startLockBootID()
	owner.PIDNamespace, _ = os.Readlink("/proc/self/ns/pid")
	owner.ProcessStart, _ = processStartTime(owner.PID)
}

// probeStartLockOwner says whether a lock owner is running. Only an owner
// on this machine (same host and machine ID) can be judged: another boot of
// it means the owner is gone; in the same boot and PID namespace, a PID that
// is not running, or that runs a process started at another time, is not the
// owner. Anything else — another host, no machine ID, another namespace, a
// read that fails — is owner-unknown (R6).
func probeStartLockOwner(owner StartLockOwner) (string, string) {
	host, _ := os.Hostname()
	machine := startLockMachineID()
	if owner.Host != host || owner.MachineID == "" || owner.MachineID != machine {
		return LockOwnerUnknown, fmt.Sprintf("held by pid %d on %s, another host or machine; its liveness cannot be checked here", owner.PID, owner.Host)
	}
	boot := startLockBootID()
	if owner.BootID == "" || boot == "" {
		return LockOwnerUnknown, "the boot identity is unavailable"
	}
	if owner.BootID != boot {
		return LockOwnerDead, fmt.Sprintf("held by pid %d in an earlier boot of this machine", owner.PID)
	}
	namespace, _ := os.Readlink("/proc/self/ns/pid")
	if owner.PIDNamespace == "" || owner.PIDNamespace != namespace {
		return LockOwnerUnknown, fmt.Sprintf("held by pid %d in another PID namespace", owner.PID)
	}
	if owner.PID <= 0 || owner.ProcessStart == "" {
		return LockOwnerUnknown, "the lock does not record the owner's process start"
	}
	started, err := processStartTime(owner.PID)
	if os.IsNotExist(err) {
		return LockOwnerDead, fmt.Sprintf("pid %d is not running", owner.PID)
	}
	if err != nil {
		return LockOwnerUnknown, fmt.Sprintf("pid %d cannot be inspected: %v", owner.PID, err)
	}
	if started != owner.ProcessStart {
		return LockOwnerDead, fmt.Sprintf("pid %d now belongs to another process", owner.PID)
	}
	return LockOwnerActive, fmt.Sprintf("held by running pid %d (%s, since %s)", owner.PID, owner.Operation, owner.AcquiredAt)
}

// processStartTime is field 22 of /proc/<pid>/stat: the start time of the
// process in clock ticks since boot. It is read after the command name,
// which may itself contain spaces and parentheses.
func processStartTime(pid int) (string, error) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	text := string(raw)
	end := strings.LastIndexByte(text, ')')
	if end < 0 {
		return "", fmt.Errorf("unreadable stat")
	}
	fields := strings.Fields(text[end+1:])
	// fields[0] is field 3 (state); the start time is field 22.
	if len(fields) < 20 {
		return "", fmt.Errorf("unreadable stat")
	}
	return fields[19], nil
}

func readTrimmed(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// startLockMachineID and startLockBootID are application-specific digests of
// the machine and boot IDs: a lock file compares them without carrying the
// machine ID, which systemd asks applications not to expose.
func startLockMachineID() string { return appSpecificID("machine", readTrimmed("/etc/machine-id")) }
func startLockBootID() string {
	return appSpecificID("boot", readTrimmed("/proc/sys/kernel/random/boot_id"))
}

func appSpecificID(kind, raw string) string {
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("pose-start-lock/" + kind + "/" + raw))
	return hex.EncodeToString(sum[:16])
}
