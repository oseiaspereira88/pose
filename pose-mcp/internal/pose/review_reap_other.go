//go:build !linux

package pose

// reapSealedCheckoutProcesses has no portable way to read another process's
// environment; outside Linux the process-group kill is the only termination.
func reapSealedCheckoutProcesses(string, map[int]string) ([]string, int, error) { return nil, 0, nil }

func reviewOpaqueProcessBaseline() map[int]string { return nil }

func enableReviewSubreaper() error                                      { return nil }
func reviewClockTicks() (uint64, error)                                 { return 0, nil }
func reapOrphanedEscapees(entry []byte, since uint64) ([]string, error) { return nil, nil }
