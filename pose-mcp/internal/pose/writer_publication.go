package pose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Artifact classes of a publication (spec
// pose-writer-coordination-and-atomic-publication R5, ADR 2026-10-16
// "Writers publish through one exclusive, compare-and-swap publisher").
const (
	// PublishAuthoritative: a record the engine reads as the truth — a spec,
	// a start record, a policy. The staged file is synced before it is
	// published and the directory after, so a published name survives a
	// process crash and, where the filesystem honors fsync, a restart.
	PublishAuthoritative = "authoritative"
	// PublishDerived: a projection the engine can compute again — indexes,
	// state, reports. It is published whole but not synced: a crash may lose
	// the last publication, never present a torn one.
	PublishDerived = "derived"
)

// Preconditions of a publication.
const (
	// ExpectAny publishes whatever is there now: last writer wins. Only for
	// derived artifacts whose writer recomputes them from their inputs, or for
	// a mutable record whose writer holds that record's lock while it reads,
	// computes and publishes (the start record under the lifecycle lock).
	ExpectAny = iota
	// ExpectDigest publishes only if the file still holds the bytes with
	// Digest (SHA-256, hex): the read-modify-write of a mutable artifact.
	ExpectDigest
	// ExpectAbsent publishes only if no file exists: an immutable artifact.
	// A retry that finds the same bytes is idempotent; other bytes conflict.
	ExpectAbsent
)

// PublishExpectation is what a writer read before it computed its bytes.
type PublishExpectation struct {
	Mode   int
	Digest string
}

// ExpectBytes is the expectation for a file read as raw (nil: absent).
func ExpectBytes(raw []byte, exists bool) PublishExpectation {
	if !exists {
		return PublishExpectation{Mode: ExpectAbsent}
	}
	return PublishExpectation{Mode: ExpectDigest, Digest: digestHex(raw)}
}

// ErrPublishConflict marks a publication refused because the file is no
// longer what its writer read. Errors that carry it are *PublishConflictError.
var ErrPublishConflict = errors.New("publish-conflict")

// PublishConflictError says which file changed, what the writer expected and
// what it found. Nothing was published.
type PublishConflictError struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Found    string `json:"found"`
}

func (e *PublishConflictError) Error() string {
	return fmt.Sprintf("%s: %s changed since it was read (expected %s, found %s); nothing was published", ErrPublishConflict, e.Path, e.Expected, e.Found)
}

func (e *PublishConflictError) Unwrap() error { return ErrPublishConflict }

// ErrPublishUnsynced marks a publication that took effect but whose
// durability could not be confirmed: the rename or link happened and the
// directory sync failed. Errors that carry it are *PublishDurabilityError.
var ErrPublishUnsynced = errors.New("publish-unsynced")

// PublishDurabilityError is a file that is published — readers see the new
// bytes — whose directory entry was not confirmed durable. It is never a
// failure without effect (R6).
type PublishDurabilityError struct {
	Path string `json:"path"`
	Err  error  `json:"-"`
}

func (e *PublishDurabilityError) Error() string {
	return fmt.Sprintf("%s: %s is published but its durability is not confirmed: %v", ErrPublishUnsynced, e.Path, e.Err)
}

func (e *PublishDurabilityError) Unwrap() error { return ErrPublishUnsynced }

// PublishSyncHook, when set, is called before each sync the publisher makes
// ("file" for the staged file, "dir" for the directory) and its error is
// used as that sync's error. Only tests set it, to observe and fail syncs.
var PublishSyncHook func(kind, path string) error

// PublishFault, when set, is called with each path about to be published,
// after its bytes are staged; an error stops that publication there. Only
// tests set it, to inject a failure at the publication point.
var PublishFault func(path string) error

// PublishBeforeReplace, when set, is called after an ExpectDigest publication
// has checked the bytes it replaces and before it renames its own into
// place: the window that only the caller's coordination closes. Only tests
// set it (spec pose-delegation-marker-stop-joins-heartbeat).
var PublishBeforeReplace func(path string)

// PublishFile publishes data at path (R1/R2/R5). The bytes are staged in a
// temporary file only this call creates, next to path, and published by a
// rename (or, for ExpectAbsent, a hard link that fails if path exists); on
// any failure only that temporary file is removed. The expectation is checked
// immediately before publishing. It compares bytes, so it detects a writer
// that does not take the engine's locks, but between the check and the
// rename a non-cooperating writer can still win: callers that coordinate hold
// the resource's lock around read, compute and publish.
func PublishFile(path string, data []byte, perm os.FileMode, class string, expect PublishExpectation) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	published := false
	defer func() {
		if !published {
			os.Remove(name)
		}
	}()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(perm)
	}
	if err == nil && class == PublishAuthoritative {
		if PublishSyncHook != nil {
			err = PublishSyncHook("file", path)
		}
		if err == nil {
			err = tmp.Sync()
		}
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if PublishFault != nil {
		if err := PublishFault(path); err != nil {
			return err
		}
	}
	switch expect.Mode {
	case ExpectAbsent:
		if err := linkFile(name, path); err != nil {
			if !errors.Is(err, os.ErrExist) {
				// A filesystem without hard links: create exclusively and
				// write. A reader may then see a record whose write was cut
				// short, which the readers of immutable records refuse.
				return publishExclusive(path, data, perm, class)
			}
			current, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if digestHex(current) != digestHex(data) {
				return &PublishConflictError{Path: path, Expected: "absent", Found: digestHex(current)}
			}
			// The same immutable bytes are already published: a retry.
			return nil
		}
		os.Remove(name)
		published = true
	case ExpectDigest:
		current, err := os.ReadFile(path)
		found := "absent"
		if err == nil {
			found = digestHex(current)
		} else if !os.IsNotExist(err) {
			return err
		}
		if found != expect.Digest {
			return &PublishConflictError{Path: path, Expected: expect.Digest, Found: found}
		}
		if PublishBeforeReplace != nil {
			PublishBeforeReplace(path)
		}
		fallthrough
	default:
		if err := os.Rename(name, path); err != nil {
			return err
		}
		published = true
	}
	if class == PublishAuthoritative {
		err := error(nil)
		if PublishSyncHook != nil {
			err = PublishSyncHook("dir", path)
		}
		if err == nil {
			err = syncDir(dir)
		}
		if err != nil {
			return &PublishDurabilityError{Path: path, Err: err}
		}
	}
	return nil
}

// publishExclusive creates path only if it does not exist and writes data in
// place, for filesystems that cannot hard-link the staged file.
func publishExclusive(path string, data []byte, perm os.FileMode, class string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			current, readErr := os.ReadFile(path)
			if readErr == nil && digestHex(current) == digestHex(data) {
				return nil
			}
			return &PublishConflictError{Path: path, Expected: "absent", Found: digestHex(current)}
		}
		return err
	}
	_, err = f.Write(data)
	if err == nil && class == PublishAuthoritative {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// syncDir makes a rename in dir durable where the platform supports syncing
// a directory; Windows does not, and its rename is already metadata-journaled
// on NTFS.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// IsPublisherStaging reports whether path names a staging file of
// PublishFile (".<name>.<digits>.tmp"), which a crash can leave behind.
func IsPublisherStaging(path string) bool {
	base := filepath.Base(path)
	if !strings.HasPrefix(base, ".") || !strings.HasSuffix(base, ".tmp") {
		return false
	}
	middle := strings.TrimSuffix(base, ".tmp")
	dot := strings.LastIndexByte(middle, '.')
	if dot <= 0 || dot == len(middle)-1 {
		return false
	}
	for _, r := range middle[dot+1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// linkFile is os.Link; tests replace it to stand for a filesystem without
// hard links.
var linkFile = os.Link
