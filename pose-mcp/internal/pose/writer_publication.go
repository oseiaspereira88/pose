package pose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	// derived artifacts whose writer recomputes them from their inputs.
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

// PublishFault, when set, is called with each path about to be published,
// after its bytes are staged; an error stops that publication there. Only
// tests set it, to inject a failure at the publication point.
var PublishFault func(path string) error

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
		err = tmp.Sync()
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
		if err := os.Link(name, path); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
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
		fallthrough
	default:
		if err := os.Rename(name, path); err != nil {
			return err
		}
		published = true
	}
	if class == PublishAuthoritative {
		return syncDir(dir)
	}
	return nil
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
