//go:build !linux

package pose

import "os/exec"

// reapSealedCheckoutProcesses has no portable way to read another process's
// environment; outside Linux the process-group kill is the only termination.
func reapSealedCheckoutProcesses(string, map[int]string) ([]string, int, error) { return nil, 0, nil }

func reviewOpaqueProcessBaseline() map[int]string { return nil }

// RunReviewShim is the Linux review run shim; elsewhere it is not used.
func RunReviewShim(args []string) int { return 125 }

func parseReviewShimReport([]byte) (reviewShimReport, error) { return reviewShimReport{}, nil }

func stopReviewShim(cmd *exec.Cmd) func() error { return cmd.Cancel }
