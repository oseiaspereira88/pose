//go:build !windows

package pose

import (
	"errors"
	"testing"
	"time"
)

var errProbeFixture = errors.New("probe fixture failure")

// R3: a successful process-group probe holds for its TTL; a failed one is
// not kept.
func TestProcessGroupProbeKeepsASuccessForItsTTL(t *testing.T) {
	original := processGroupVerify
	t.Cleanup(func() {
		processGroupVerify = original
		processGroupProbe.Lock()
		processGroupProbe.ok = time.Time{}
		processGroupProbe.Unlock()
	})
	processGroupProbe.Lock()
	processGroupProbe.ok = time.Time{}
	processGroupProbe.Unlock()
	probes, fail := 0, true
	processGroupVerify = func(int) error {
		probes++
		if fail {
			return errProbeFixture
		}
		return nil
	}
	if reviewProcessGroupAvailable() == nil || reviewProcessGroupAvailable() == nil || probes != 2 {
		t.Fatalf("a failed probe was kept: probes=%d", probes)
	}
	fail = false
	for i := 0; i < 3; i++ {
		if err := reviewProcessGroupAvailable(); err != nil {
			t.Fatal(err)
		}
	}
	if probes != 3 {
		t.Fatalf("a successful probe was not kept: probes=%d, want 3", probes)
	}
	processGroupProbe.Lock()
	processGroupProbe.ok = time.Now().Add(-2 * processGroupProbeTTL)
	processGroupProbe.Unlock()
	reviewProcessGroupAvailable()
	if probes != 4 {
		t.Fatalf("an expired probe was not repeated: probes=%d, want 4", probes)
	}
}
