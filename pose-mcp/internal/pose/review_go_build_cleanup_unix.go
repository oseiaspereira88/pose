//go:build unix

package pose

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	reviewGoBuildMaxEntries = 200000
	reviewGoBuildMaxBytes   = int64(8 << 30)
	reviewGoBuildMaxDepth   = 128
)

var errReviewGoBuildChanged = errors.New("runtime path changed during cleanup; replacement retained")

// removeReviewGoBuildDir removes one go-build directory of the runtime root
// through descriptors only: it is opened relative to the runtime root's
// descriptor as a directory, without following a link and without blocking,
// so a link or a FIFO in its place fails at once and is left to the runtime
// inventory; its contents are removed relative to it under a bounded
// inventory; and the cleanup is confirmed only when the name still binds it
// and, after the removal, it is unlinked (review rounds 1 to 4).
func removeReviewGoBuildDir(root *os.Root, name string) error {
	parent, err := root.OpenFile(".", os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	defer parent.Close()
	pfd := int(parent.Fd())
	reviewGoBuildBeforeOpen(name)
	dfd, err := unix.Openat(pfd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOTDIR) || errors.Is(err, unix.ELOOP) || errors.Is(err, unix.ENOENT) {
		return nil // not a directory now: the runtime inventory judges it
	}
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	defer unix.Close(dfd)
	var opened unix.Stat_t
	if err := unix.Fstat(dfd, &opened); err != nil || opened.Mode&unix.S_IFMT != unix.S_IFDIR {
		return errReviewGoBuildChanged
	}
	budget := reviewGoBuildBudget{entries: reviewGoBuildMaxEntries, bytes: reviewGoBuildMaxBytes}
	if err := removeReviewGoBuildContents(dfd, 0, &budget); err != nil {
		return err
	}
	reviewGoBuildAfterContents(name)
	var bound unix.Stat_t
	if err := unix.Fstatat(pfd, name, &bound, unix.AT_SYMLINK_NOFOLLOW); err != nil || bound.Dev != opened.Dev || bound.Ino != opened.Ino {
		return errReviewGoBuildChanged
	}
	reviewGoBuildBeforeRemove(name)
	if err := unix.Unlinkat(pfd, name, unix.AT_REMOVEDIR); err != nil {
		return fmt.Errorf("runtime cleanup incomplete; resources retained")
	}
	var after unix.Stat_t
	if err := unix.Fstat(dfd, &after); err != nil || after.Dev != opened.Dev || after.Ino != opened.Ino || after.Nlink != 0 {
		return errReviewGoBuildChanged
	}
	return nil
}

type reviewGoBuildBudget struct {
	entries int
	bytes   int64
}

// removeReviewGoBuildContents empties the directory open as dfd: every entry
// is examined without following links, subdirectories are opened relative to
// dfd as directories without blocking, and every removal is relative to the
// descriptor that holds the entry.
func removeReviewGoBuildContents(dfd, depth int, budget *reviewGoBuildBudget) error {
	if depth > reviewGoBuildMaxDepth {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	dup, err := unix.Dup(dfd)
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	dir := os.NewFile(uintptr(dup), "go-build")
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	for _, name := range names {
		budget.entries--
		if budget.entries < 0 {
			return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
		}
		var st unix.Stat_t
		if err := unix.Fstatat(dfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
		}
		if st.Mode&unix.S_IFMT != unix.S_IFDIR {
			if st.Mode&unix.S_IFMT == unix.S_IFREG {
				budget.bytes -= st.Size
				if budget.bytes < 0 {
					return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
				}
			}
			if err := unix.Unlinkat(dfd, name, 0); err != nil {
				return fmt.Errorf("runtime cleanup incomplete; resources retained")
			}
			continue
		}
		child, err := unix.Openat(dfd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if err != nil {
			return errReviewGoBuildChanged
		}
		var opened unix.Stat_t
		if err := unix.Fstat(child, &opened); err != nil || opened.Dev != st.Dev || opened.Ino != st.Ino {
			unix.Close(child)
			return errReviewGoBuildChanged
		}
		err = removeReviewGoBuildContents(child, depth+1, budget)
		unix.Close(child)
		if err != nil {
			return err
		}
		if err := unix.Unlinkat(dfd, name, unix.AT_REMOVEDIR); err != nil {
			return fmt.Errorf("runtime cleanup incomplete; resources retained")
		}
	}
	return nil
}
