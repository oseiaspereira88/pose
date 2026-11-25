//go:build windows

package pose

// Ending the reviewer's process tree on Windows (spec
// pose-review-run-reaps-lingering-reviewer-processes R1, review round 13,
// N1): the adapter starts suspended, is assigned to a job object that kills
// every process in it when the job is terminated or its last handle closes,
// and only then resumes, so no descendant can start outside the job. The
// dispatch terminates the job after the run, on a timeout, and, through the
// kill-on-close limit, when the dispatching process ends.

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var reviewJobs sync.Map // *exec.Cmd -> windows.Handle

// jobBasicAccounting is JOBOBJECT_BASIC_ACCOUNTING_INFORMATION.
type jobBasicAccounting struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// prepareReviewProcessTree makes cmd start suspended, so it can be placed in
// its job before it runs.
func prepareReviewProcessTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	cmd.Cancel = func() error { return terminateReviewJob(cmd) }
}

// attachReviewProcessTree places the started, suspended process in a new
// kill-on-close job and resumes it. A process that cannot be placed is
// terminated: it never runs outside the job.
func attachReviewProcessTree(cmd *exec.Cmd) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("the reviewer's job object could not be created: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		_ = cmd.Process.Kill()
		return fmt.Errorf("the reviewer's job object could not be limited: %w", err)
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		_ = cmd.Process.Kill()
		return fmt.Errorf("the reviewer's process could not be opened: %w", err)
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = windows.CloseHandle(job)
		_ = cmd.Process.Kill()
		return fmt.Errorf("the reviewer's process could not join its job: %w", err)
	}
	reviewJobs.Store(cmd, job)
	if err := resumeProcessThreads(uint32(cmd.Process.Pid)); err != nil {
		_ = terminateReviewJob(cmd)
		return fmt.Errorf("the reviewer's process could not be resumed: %w", err)
	}
	return nil
}

// resumeProcessThreads resumes every thread of pid; a suspended process has
// only its main thread.
func resumeProcessThreads(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	resumed := 0
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		_, rerr := windows.ResumeThread(thread)
		_ = windows.CloseHandle(thread)
		if rerr != nil {
			return rerr
		}
		resumed++
	}
	if resumed == 0 {
		return fmt.Errorf("no thread of process %d was found", pid)
	}
	return nil
}

// terminateReviewJob ends every process of cmd's job and waits until the job
// is empty.
func terminateReviewJob(cmd *exec.Cmd) error {
	value, ok := reviewJobs.Load(cmd)
	if !ok {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	job := value.(windows.Handle)
	if err := windows.TerminateJobObject(job, 1); err != nil {
		return fmt.Errorf("the reviewer's job could not be terminated: %w", err)
	}
	deadline := time.Now().Add(reviewReapWait)
	for {
		var acct jobBasicAccounting
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&acct)), uint32(unsafe.Sizeof(acct)), nil); err != nil {
			return fmt.Errorf("the reviewer's job could not be inspected: %w", err)
		}
		if acct.ActiveProcesses == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%d process(es) of the reviewer's job could not be ended", acct.ActiveProcesses)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// releaseReviewProcessTree terminates the job and closes its handle.
func releaseReviewProcessTree(cmd *exec.Cmd) error {
	err := terminateReviewJob(cmd)
	if value, ok := reviewJobs.LoadAndDelete(cmd); ok {
		_ = windows.CloseHandle(value.(windows.Handle))
	}
	return err
}
