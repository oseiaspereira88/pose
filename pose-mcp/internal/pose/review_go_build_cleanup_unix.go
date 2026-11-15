//go:build unix

package pose

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

const (
	reviewGoBuildMaxEntries = 200000
	reviewGoBuildMaxBytes   = int64(8 << 30)
	reviewGoBuildMaxDepth   = 128
)

// reviewGoBuildEnumerating receives the descriptor that enumerates a
// go-build directory; tests use it to read its flags.
var reviewGoBuildEnumerating = func(int) {}

// reviewGoBuildRead receives the size of each batch of entries read; tests
// use it to see that enumeration stops at the budget.
var reviewGoBuildRead = func(int) {}

// reviewGoBuildNestedBeforeRemove runs between the last check of a nested
// directory's name and its removal; tests use it to change the name there.
var reviewGoBuildNestedBeforeRemove = func(string) {}

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
	// The identity seen before the open binds the name: a directory moved
	// away and replaced, or gone, before the open leaves the cleanup
	// incomplete instead of handing the replacement to the runtime
	// inventory (review round 5, N7). Anything else than a real directory
	// at the listing is left to that inventory.
	var listed unix.Stat_t
	if err := unix.Fstatat(pfd, name, &listed, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	if listed.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil
	}
	reviewGoBuildBeforeOpen(name)
	dfd, err := unix.Openat(pfd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return errReviewGoBuildChanged
	}
	defer unix.Close(dfd)
	var opened unix.Stat_t
	if err := unix.Fstat(dfd, &opened); err != nil || opened.Dev != listed.Dev || opened.Ino != listed.Ino {
		return errReviewGoBuildChanged
	}
	budget := reviewGoBuildBudget{entries: reviewGoBuildMaxEntries, bytes: reviewGoBuildMaxBytes}
	if err := removeReviewGoBuildContents(dfd, 0, &budget); err != nil {
		return err
	}
	reviewGoBuildAfterContents(name)
	return unlinkReviewGoBuildBound(pfd, name, dfd, &opened, reviewGoBuildBeforeRemove)
}

// unlinkReviewGoBuildBound removes the emptied directory open as dfd by its
// name in the directory open as pfd, and confirms the removal through dfd:
// the name must still bind it before the removal, and afterwards it must be
// unlinked. Every level of a go-build tree goes through it (review rounds 2
// and 6, N1 and N9).
func unlinkReviewGoBuildBound(pfd int, name string, dfd int, opened *unix.Stat_t, beforeRemove func(string)) error {
	var bound unix.Stat_t
	if err := unix.Fstatat(pfd, name, &bound, unix.AT_SYMLINK_NOFOLLOW); err != nil || bound.Dev != opened.Dev || bound.Ino != opened.Ino {
		return errReviewGoBuildChanged
	}
	beforeRemove(name)
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
	// The enumeration descriptor is duplicated close-on-exec atomically, so
	// a concurrent exec never inherits it (review round 5, N8), and entries
	// are read in batches charged to the budget as they are read (N6).
	dup, err := unix.FcntlInt(uintptr(dfd), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	dir := os.NewFile(uintptr(dup), "go-build")
	defer dir.Close()
	reviewGoBuildEnumerating(dup)
	for {
		names, err := dir.Readdirnames(256)
		reviewGoBuildRead(len(names))
		if len(names) == 0 {
			if err == nil || errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
		}
		if err := removeReviewGoBuildEntries(dfd, depth, names, budget); err != nil {
			return err
		}
	}
}

// removeReviewGoBuildEntries removes one batch of entries of the directory
// open as dfd.
func removeReviewGoBuildEntries(dfd, depth int, names []string, budget *reviewGoBuildBudget) error {
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
		if err == nil {
			err = unlinkReviewGoBuildBound(dfd, name, child, &opened, reviewGoBuildNestedBeforeRemove)
		}
		unix.Close(child)
		if err != nil {
			return err
		}
	}
	return nil
}
