//go:build !windows

package pose

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A special file named like a key is never opened: listing skips it and
// loading refuses it, instead of blocking on a FIFO (found in review by
// agent:gpt-6.1-sol).
func TestIssuerSpecialFileIsNeverOpened(t *testing.T) {
	dir := useIssuerHome(t)
	if _, err := CreateIssuerKey("", "maintainer", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "foreign.key"), 0o600); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	done := make(chan struct{})
	var listed []IssuerKeySummary
	var loadErr error
	go func() {
		listed, _ = ListIssuerKeys("")
		_, loadErr = LoadIssuerKey("", "foreign")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("a FIFO named like a key blocked the issuer store")
	}
	if len(listed) != 1 || listed[0].Issuer != "maintainer" {
		t.Fatalf("list with a FIFO present: %+v", listed)
	}
	if loadErr == nil || !strings.Contains(loadErr.Error(), "not a regular file") {
		t.Fatalf("a FIFO was loaded as a key: %v", loadErr)
	}
}
