//go:build unix

package pose

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// Review round 3 (N2): a go-build directory replaced by a FIFO after it was
// emptied leaves the cleanup incomplete without blocking: the verified
// directory is never reopened by its name.
func TestAdapterCleanupDoesNotBlockOnAFIFOReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux descriptor and symlink qualification")
	}
	f := makeContainmentCleanupFixture(t)
	build := filepath.Join(f.tmp, "go-build11")
	if err := os.MkdirAll(build, 0o700); err != nil {
		t.Fatal(err)
	}
	away := filepath.Join(t.TempDir(), "away")
	original := reviewGoBuildAfterContents
	reviewGoBuildAfterContents = func(string) {
		if err := os.Rename(build, away); err != nil {
			t.Error(err)
		}
		if err := syscall.Mkfifo(build, 0o600); err != nil {
			t.Error(err)
		}
	}
	defer func() { reviewGoBuildAfterContents = original }()
	done := make(chan error, 1)
	go func() { done <- f.cleanup() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cleanup was confirmed over a FIFO replacement")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("cleanup blocked on a FIFO that replaced the go-build directory")
	}
}

// Review round 4 (N4): a go-build directory replaced by a FIFO after it was
// listed and before it was first opened is not opened as one: the cleanup
// ends without blocking, and the FIFO stays for the runtime inventory.
func TestAdapterCleanupDoesNotBlockOnAFIFOBeforeTheFirstOpen(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux descriptor and symlink qualification")
	}
	f := makeContainmentCleanupFixture(t)
	build := filepath.Join(f.tmp, "go-build13")
	if err := os.MkdirAll(build, 0o700); err != nil {
		t.Fatal(err)
	}
	away := filepath.Join(t.TempDir(), "away")
	original := reviewGoBuildBeforeOpen
	reviewGoBuildBeforeOpen = func(string) {
		if err := os.Rename(build, away); err != nil {
			t.Error(err)
		}
		if err := syscall.Mkfifo(build, 0o600); err != nil {
			t.Error(err)
		}
	}
	defer func() { reviewGoBuildBeforeOpen = original }()
	done := make(chan error, 1)
	go func() { done <- f.cleanup() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("cleanup blocked on a FIFO that replaced the go-build directory before its first open")
	}
	if info, err := os.Stat(away); err != nil || !info.IsDir() {
		t.Fatalf("the renamed directory was lost: %v", err)
	}
}
