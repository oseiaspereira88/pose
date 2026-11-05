package pose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// MaxGovernanceWriteRoots bounds the root lock set of one operation.
	MaxGovernanceWriteRoots    = 16
	maxGovernanceFilesWrappers = 8
)

var (
	ErrGovernanceWriteGuardBusy         = errors.New("governance-write-guard-busy")
	ErrGovernanceWriteGuardUnsupported  = errors.New("governance-write-guard-unsupported")
	ErrGovernanceWriteGuardUnavailable  = errors.New("governance-write-guard-unavailable")
	ErrGovernanceWriteGuardInvalid      = errors.New("governance-write-guard-invalid")
	ErrGovernanceWriteGuardTooManyRoots = errors.New("governance-write-guard-too-many-roots")
	ErrGovernanceWriteGuardRootBinding  = errors.New("governance-write-guard-root-binding")
	ErrGovernanceWriteGuardReleased     = errors.New("governance-write-guard-released")
	ErrGovernanceWriteGuardReentrant    = errors.New("governance-write-guard-reentrant")
	ErrGovernanceWriteGuardMixedPreview = errors.New("governance-write-guard-mixed-preview")
)

// GovernanceWriteGuardError is a bounded, non-waiting refusal. Root names are
// project IDs when available and otherwise the final path element; absolute
// host paths are never included in the error.
type GovernanceWriteGuardError struct {
	kind      error
	operation string
	root      string
	cause     error
}

func (e *GovernanceWriteGuardError) Error() string {
	message := "pose: " + e.kind.Error()
	if e.operation != "" {
		message += " during " + e.operation
	}
	if e.root != "" {
		message += " for " + e.root
	}
	if e.cause != nil {
		message += ": " + e.cause.Error()
	}
	return message
}

func (e *GovernanceWriteGuardError) Unwrap() error { return e.cause }
func (e *GovernanceWriteGuardError) Is(target error) bool {
	return target == e.kind
}

func governanceGuardError(kind error, operation, root string, cause error) error {
	return &GovernanceWriteGuardError{kind: kind, operation: operation, root: root, cause: cause}
}

type governanceGuardRoot struct {
	path      string // canonical physical root; internal only
	key       string
	projectID string
	label     string
}

// GovernanceWriteLease is the explicit lease carried by guarded Store/Files
// copies. It only coordinates writers that acquire the same per-root lock.
// The lease is not an atomic filesystem snapshot and grants no authorization
// after Release.
type GovernanceWriteLease struct {
	operation string
	roots     []governanceGuardRoot // canonical acquisition order
	byKey     map[string]governanceGuardRoot
	preview   bool
	active    atomic.Bool
	once      sync.Once
	release   func()
}

// Release releases roots in reverse canonical order. Repeated calls are safe.
func (l *GovernanceWriteLease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		l.active.Store(false)
		if l.release != nil {
			l.release()
		}
	})
}

// Active reports whether this lease is still in its guarded callback.
func (l *GovernanceWriteLease) Active() bool {
	return l != nil && l.active.Load()
}

// Preview reports a non-locking lease used only to stage a preview. It never
// claims to exclude a concurrent writer.
func (l *GovernanceWriteLease) Preview() bool {
	return l != nil && l.preview
}

// Covers reports whether this lease is bound to the Store's physical root and
// project selector. It returns false after the lease is released.
func (l *GovernanceWriteLease) Covers(store Store) bool {
	if l == nil || !l.active.Load() {
		return false
	}
	root, err := canonicalGovernanceRoot(store.Root)
	if err != nil {
		return false
	}
	binding, ok := l.byKey[governanceRootKey(root)]
	return ok && binding.projectID == store.FederatedProjectID
}

// WithGovernanceWriteGuard acquires one non-waiting root guard for every
// already-authorized Store, in canonical root order, then invokes callback
// with copies that carry the lease through Files. It never discovers or
// authorizes roots. The guard uses the stable B14 transfer lock path and must
// be acquired before resource locks. A preview stages through PreviewFiles
// without creating lock directories or files.
func WithGovernanceWriteGuard(operation string, roots []Store, callback func([]Store) error) error {
	if callback == nil || strings.TrimSpace(operation) == "" || len(operation) > 96 || strings.ContainsAny(operation, "\r\n\x00") {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("operation and callback are required"))
	}
	if len(roots) == 0 {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("no authorized roots were supplied"))
	}
	freshRoots := make([]Store, len(roots))
	for i, store := range roots {
		if lease, present := store.governanceWriteGuardInFiles(); present {
			if lease == nil || !lease.Active() {
				return governanceGuardError(ErrGovernanceWriteGuardReleased, operation, "", nil)
			}
			return governanceGuardError(ErrGovernanceWriteGuardReentrant, operation, "", errors.New("pass the existing lease through Store/Files instead of reacquiring it"))
		}
		freshRoots[i] = store.withoutGovernanceReadView()
	}
	roots = freshRoots

	bindings, storeKeys, preview, err := prepareGovernanceGuardRoots(roots, operation)
	if err != nil {
		return err
	}
	if len(bindings) > MaxGovernanceWriteRoots {
		return governanceGuardError(ErrGovernanceWriteGuardTooManyRoots, operation, "", fmt.Errorf("got %d unique roots; maximum is %d", len(bindings), MaxGovernanceWriteRoots))
	}

	lease := &GovernanceWriteLease{operation: operation, roots: bindings, byKey: make(map[string]governanceGuardRoot, len(bindings)), preview: preview}
	for _, binding := range bindings {
		lease.byKey[binding.key] = binding
	}
	if !preview {
		release, lockErr := acquireGovernanceRootLocks(bindings, operation)
		if lockErr != nil {
			return lockErr
		}
		lease.release = release
	}
	lease.active.Store(true)
	defer lease.Release()

	guarded := make([]Store, len(roots))
	for i, store := range roots {
		binding, ok := lease.byKey[storeKeys[i]]
		if !ok || binding.projectID != store.FederatedProjectID {
			return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, "", errors.New("root binding changed during guard preparation"))
		}
		// Bind both Files path checks and Store path construction to the same
		// physical root. Keeping a symlink spelling in Store.Root would make
		// legitimate writes appear outside the canonical root.
		store.Root = binding.path
		store.FS = &governanceGuardFiles{Files: store.fsys(), lease: lease, root: binding.path, projectID: binding.projectID}
		guarded[i] = store
	}
	return callback(guarded)
}

// GovernanceWriteLease retrieves the explicit lease from a Store's Files
// chain. Optional UnderlyingFiles wrappers are followed with a depth and
// cycle bound, so a read-only view can carry but cannot manufacture a lease.
func (s Store) GovernanceWriteLease() (*GovernanceWriteLease, bool) {
	lease, present := s.governanceWriteGuardInFiles()
	if !present || !lease.Active() || !lease.Covers(s) {
		return nil, false
	}
	return lease, true
}

// withGovernanceWriteGuard runs an effectful operation under the explicit
// per-root lease. Callers may compose guarded operations by carrying the Store
// received in their callback; a released or misbound lease is a refusal.
func (s Store) withGovernanceWriteGuard(operation string, callback func(Store) error) error {
	if callback == nil {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("callback is required"))
	}
	if lease, present := s.governanceWriteGuardInFiles(); present {
		if lease == nil || !lease.Active() {
			return governanceGuardError(ErrGovernanceWriteGuardReleased, operation, "", nil)
		}
		if !lease.Covers(s) {
			return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, "", nil)
		}
		// Reuse the explicit lease, but never let an earlier read-only
		// operation view authorize a fresh effect evaluation. Unwrap that
		// observation to the guarded source and clear Store-level memoized
		// inputs before evaluating under the lease.
		return callback(s.withoutGovernanceReadView())
	}
	return WithGovernanceWriteGuard(operation, []Store{s}, func(stores []Store) error {
		return callback(stores[0])
	})
}

func (s Store) governanceWriteGuardInFiles() (*GovernanceWriteLease, bool) {
	files := s.fsys()
	seen := make([]Files, 0, maxGovernanceFilesWrappers)
	for depth := 0; depth < maxGovernanceFilesWrappers; depth++ {
		for _, prior := range seen {
			if sameGovernanceFiles(prior, files) {
				return nil, false
			}
		}
		seen = append(seen, files)
		if carrier, ok := files.(interface{ GovernanceWriteLease() *GovernanceWriteLease }); ok {
			lease := carrier.GovernanceWriteLease()
			return lease, lease != nil
		}
		unwrapper, ok := files.(interface{ UnderlyingFiles() Files })
		if !ok {
			return nil, false
		}
		next := unwrapper.UnderlyingFiles()
		if next == nil {
			return nil, false
		}
		files = next
	}
	return nil, false
}

func sameGovernanceFiles(a, b Files) bool {
	if a == nil || b == nil || reflect.TypeOf(a) != reflect.TypeOf(b) {
		return a == nil && b == nil
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Comparable() {
		return va.Interface() == vb.Interface()
	}
	if va.Kind() == reflect.Ptr {
		return va.Pointer() == vb.Pointer()
	}
	return false // the wrapper-depth bound still stops an uncomparable cycle
}

type governanceGuardFiles struct {
	Files
	lease     *GovernanceWriteLease
	root      string
	projectID string
}

// UnderlyingFiles lets another operation-scoped Files wrapper preserve the
// explicit lease without making Files itself grow a mandatory method.
func (f *governanceGuardFiles) UnderlyingFiles() Files { return f.Files }

func (f *governanceGuardFiles) GovernanceWriteLease() *GovernanceWriteLease { return f.lease }

// Preserve operation-specific durability hooks carried by a Files wrapper.
// The guard remains the mutation boundary; the underlying hook still observes
// the same append receipt after the journal lock has been acquired.
func (f *governanceGuardFiles) syncActionJournal(file *os.File) error {
	if syncer, ok := f.Files.(interface{ syncActionJournal(*os.File) error }); ok {
		return syncer.syncActionJournal(file)
	}
	return file.Sync()
}

func (f *governanceGuardFiles) writable() error {
	if f == nil || f.lease == nil || !f.lease.Active() {
		return ErrGovernanceWriteGuardReleased
	}
	if !f.lease.preview && !f.lease.coversBinding(f.root, f.projectID) {
		return ErrGovernanceWriteGuardRootBinding
	}
	return nil
}

// Writes through a retained guarded Store stop when its callback releases the
// lease. Preview writes remain staged by PreviewFiles and never touch disk.
func (f *governanceGuardFiles) WriteFile(name string, data []byte, perm os.FileMode) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.WriteFile(name, data, perm)
}
func (f *governanceGuardFiles) CreateExclusive(name string, data []byte, perm os.FileMode, syncFile bool) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.CreateExclusive(name, data, perm, syncFile)
}
func (f *governanceGuardFiles) Publish(name string, data []byte, perm os.FileMode, class string, expect PublishExpectation) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.Publish(name, data, perm, class, expect)
}
func (f *governanceGuardFiles) MkdirAll(name string, perm os.FileMode) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.MkdirAll(name, perm)
}
func (f *governanceGuardFiles) Mkdir(name string, perm os.FileMode) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.Mkdir(name, perm)
}
func (f *governanceGuardFiles) Remove(name string) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.Remove(name)
}
func (f *governanceGuardFiles) RemoveAll(name string) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.RemoveAll(name)
}
func (f *governanceGuardFiles) Symlink(oldname, newname string) error {
	if err := f.writable(); err != nil {
		return err
	}
	return f.Files.Symlink(oldname, newname)
}

func (l *GovernanceWriteLease) coversBinding(root, projectID string) bool {
	if l == nil || !l.active.Load() {
		return false
	}
	binding, ok := l.byKey[governanceRootKey(root)]
	return ok && binding.path == root && binding.projectID == projectID
}

func prepareGovernanceGuardRoots(stores []Store, operation string) ([]governanceGuardRoot, []string, bool, error) {
	previewCount := 0
	for _, store := range stores {
		if store.fsys().Preview() != nil {
			previewCount++
		}
	}
	if previewCount != 0 && previewCount != len(stores) {
		return nil, nil, false, governanceGuardError(ErrGovernanceWriteGuardMixedPreview, operation, "", errors.New("all roots must use preview Files or all roots must use native Files"))
	}
	preview := previewCount == len(stores)

	byKey := make(map[string]governanceGuardRoot, len(stores))
	storeKeys := make([]string, len(stores))
	for i, store := range stores {
		root, err := canonicalGovernanceRoot(store.Root)
		if err != nil {
			return nil, nil, false, governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, store.FederatedProjectID, err)
		}
		key := governanceRootKey(root)
		binding := governanceGuardRoot{path: root, key: key, projectID: store.FederatedProjectID, label: governanceRootLabel(root, store.FederatedProjectID)}
		if prior, exists := byKey[key]; exists {
			if prior.projectID != binding.projectID {
				return nil, nil, false, governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, binding.label, errors.New("one physical root was selected with different project bindings"))
			}
		} else {
			if err := validateGovernanceLockPaths(binding); err != nil {
				return nil, nil, false, governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, binding.label, err)
			}
			byKey[key] = binding
		}
		storeKeys[i] = key
	}
	bindings := make([]governanceGuardRoot, 0, len(byKey))
	for _, binding := range byKey {
		bindings = append(bindings, binding)
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].key < bindings[j].key })
	return bindings, storeKeys, preview, nil
}

func canonicalGovernanceRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	physical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	physical = filepath.Clean(physical)
	info, err := os.Stat(physical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("root is not a directory")
	}
	return physical, nil
}

func governanceRootKey(root string) string {
	key := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}

func governanceRootLabel(root, projectID string) string {
	if projectID != "" {
		return projectID
	}
	return filepath.Base(root)
}

func validateGovernanceLockPaths(root governanceGuardRoot) error {
	poseDir := filepath.Join(root.path, ".pose")
	info, err := os.Lstat(poseDir)
	if errors.Is(err, os.ErrNotExist) {
		// A first effectful write may be the operation that creates .pose.
		// The canonical root is already validated and the lock path below is
		// created with non-recursive mkdirs that reject a link substituted in
		// this gap.
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(NativeFiles(), root.path, poseDir) {
		if err != nil {
			return fmt.Errorf(".pose directory is unavailable: %w", err)
		}
		return errors.New(".pose directory is linked, not a directory, or escapes the authorized root")
	}
	transfersDir := filepath.Join(poseDir, "transfers")
	if info, err := os.Lstat(transfersDir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(NativeFiles(), root.path, transfersDir) {
			return errors.New("transfer lock directory is linked, not a directory, or escapes the authorized root")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("transfer lock directory cannot be inspected: %w", err)
	}
	lockPath := filepath.Join(transfersDir, ".authority-transfer.lock")
	if info, err := os.Lstat(lockPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(NativeFiles(), root.path, lockPath) {
			return errors.New("authority transfer lock is linked, not a regular file, or escapes the authorized root")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("authority transfer lock cannot be inspected: %w", err)
	}
	return nil
}

func acquireGovernanceRootLocks(roots []governanceGuardRoot, operation string) (func(), error) {
	files := make([]*os.File, 0, len(roots))
	unlockers := make([]func(), 0, len(roots))
	cleanup := func() {
		for i := len(unlockers) - 1; i >= 0; i-- {
			unlockers[i]()
		}
		for i := len(files) - 1; i >= 0; i-- {
			_ = files[i].Close()
		}
	}
	for _, root := range roots {
		poseDir := filepath.Join(root.path, ".pose")
		if err := ensureGovernanceLockDirectory(root, poseDir); err != nil {
			cleanup()
			return nil, governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, root.label, err)
		}
		transfersDir := filepath.Join(root.path, ".pose", "transfers")
		if err := ensureGovernanceLockDirectory(root, transfersDir); err != nil {
			cleanup()
			return nil, governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, root.label, err)
		}
		if err := validateGovernanceLockPaths(root); err != nil {
			cleanup()
			return nil, governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, root.label, err)
		}
		path := filepath.Join(transfersDir, ".authority-transfer.lock")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			cleanup()
			return nil, governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, root.label, err)
		}
		files = append(files, file)
		opened, statErr := file.Stat()
		current, lstatErr := os.Lstat(path)
		if statErr != nil || lstatErr != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() || !os.SameFile(opened, current) || !artifactPathWithin(NativeFiles(), root.path, path) {
			cleanup()
			cause := errors.New("authority transfer lock changed or escaped during open")
			if statErr != nil {
				cause = statErr
			} else if lstatErr != nil {
				cause = lstatErr
			}
			return nil, governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, root.label, cause)
		}
		unlock, lockErr := lockSpecTransferFile(file)
		if lockErr != nil {
			cleanup()
			if errors.Is(lockErr, errFileLockUnsupported) {
				return nil, governanceGuardError(ErrGovernanceWriteGuardUnsupported, operation, root.label, lockErr)
			}
			return nil, governanceGuardError(ErrGovernanceWriteGuardBusy, operation, root.label, lockErr)
		}
		unlockers = append(unlockers, unlock)
	}
	return cleanup, nil
}

func ensureGovernanceLockDirectory(root governanceGuardRoot, path string) error {
	if info, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	} else if err != nil {
		return err
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(NativeFiles(), root.path, path) {
		return errors.New("guard directory is linked, not a directory, or escapes the authorized root")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(NativeFiles(), root.path, path) {
		return errors.New("guard directory changed to a link, non-directory, or escaped the authorized root")
	}
	return nil
}
