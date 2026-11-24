//go:build !linux

package pose

// reapSealedCheckoutProcesses has no portable way to read another process's
// environment; outside Linux the process-group kill is the only termination.
func reapSealedCheckoutProcesses(string) ([]string, error) { return nil, nil }
