//go:build unix

package pose

import (
	"fmt"

	"golang.org/x/sys/unix"
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

// Review rounds 4 and 5 (N4, N7): a go-build directory replaced by a FIFO
// after it was listed and before it was first opened is not opened as one:
// the cleanup ends without blocking, and incomplete.
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
	case err := <-done:
		if err == nil {
			t.Fatal("cleanup was confirmed after the go-build directory was replaced before its first open")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("cleanup blocked on a FIFO that replaced the go-build directory before its first open")
	}
	if info, err := os.Stat(away); err != nil || !info.IsDir() {
		t.Fatalf("the renamed directory was lost: %v", err)
	}
}

// Review round 5 (N8): the descriptor that enumerates a go-build directory
// is close-on-exec, as every other descriptor of the cleanup.
func TestReviewGoBuildEnumerationDescriptorIsCloseOnExec(t *testing.T) {
	dir := t.TempDir()
	dfd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(dfd)
	seen := -1
	original := reviewGoBuildEnumerating
	reviewGoBuildEnumerating = func(fd int) {
		flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
		if err != nil {
			t.Error(err)
		}
		seen = flags & unix.FD_CLOEXEC
	}
	defer func() { reviewGoBuildEnumerating = original }()
	budget := reviewGoBuildBudget{entries: 10, bytes: 10}
	if err := removeReviewGoBuildContents(dfd, 0, &budget); err != nil {
		t.Fatal(err)
	}
	if seen != unix.FD_CLOEXEC {
		t.Fatalf("enumeration descriptor flags %d lack FD_CLOEXEC", seen)
	}
}

// Review round 5 (N6): a go-build directory with more entries than its
// budget is refused before all of them are read.
func TestReviewGoBuildEnumerationStopsAtTheBudget(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 600; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dfd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(dfd)
	read := 0
	original := reviewGoBuildRead
	reviewGoBuildRead = func(n int) { read += n }
	defer func() { reviewGoBuildRead = original }()
	budget := reviewGoBuildBudget{entries: 100, bytes: 1 << 20}
	if err := removeReviewGoBuildContents(dfd, 0, &budget); err == nil {
		t.Fatal("a directory over its entry budget was confirmed")
	}
	if read > 256 {
		t.Fatalf("enumeration read %d entries of 600 before refusing a budget of 100", read)
	}
}

// Review round 6 (N9): a nested directory of a go-build tree renamed away
// and replaced between the check of its name and its removal leaves the
// cleanup incomplete, and the renamed directory is not lost.
func TestAdapterCleanupRefusesANestedGoBuildDirectoryReplacedBeforeRemoval(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux descriptor and symlink qualification")
	}
	f := makeContainmentCleanupFixture(t)
	nested := filepath.Join(f.tmp, "go-build15", "b001")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	away := filepath.Join(t.TempDir(), "away")
	original := reviewGoBuildNestedBeforeRemove
	reviewGoBuildNestedBeforeRemove = func(string) {
		if err := os.Rename(nested, away); err != nil {
			t.Error(err)
		}
		if err := os.Mkdir(nested, 0o700); err != nil {
			t.Error(err)
		}
	}
	defer func() { reviewGoBuildNestedBeforeRemove = original }()
	if err := f.cleanup(); err == nil {
		t.Fatal("removing a nested replacement confirmed the go-build cleanup")
	}
	if info, err := os.Stat(away); err != nil || !info.IsDir() {
		t.Fatalf("the renamed directory was lost: %v", err)
	}
}

// Review round 6 (N10): go-build directories do not count against the
// runtime inventory they are removed before: a runtime with exactly as many
// other entries as that inventory admits, plus go-build directories, is
// cleaned up.
func TestGoBuildLeftoversDoNotCountAgainstTheRuntimeInventory(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < reviewRuntimeMaxEntries; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%04d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := os.MkdirAll(filepath.Join(dir, fmt.Sprintf("go-build%d", i), "b001"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := removeReviewGoBuildLeftovers(root); err != nil {
		t.Fatalf("go-build directories counted against the runtime inventory: %v", err)
	}
	if err := removeReviewOwnedContents(root, reviewRuntimeMaxEntries, 128<<20); err != nil {
		t.Fatalf("the runtime inventory refused what it admits: %v", err)
	}
	left, err := os.ReadDir(dir)
	if err != nil || len(left) != 0 {
		t.Fatalf("runtime not emptied: %d entries, %v", len(left), err)
	}
}
