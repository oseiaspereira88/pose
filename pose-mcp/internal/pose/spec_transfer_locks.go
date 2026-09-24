package pose

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func lockSpecTransferRoots(stores map[string]Store) (func(), error) {
	type target struct{ root string }
	byRoot := map[string]target{}
	for _, store := range stores {
		abs, err := filepath.Abs(store.Root)
		if err != nil {
			return nil, specTransferError("lock-unavailable")
		}
		byRoot[filepath.Clean(abs)] = target{root: filepath.Clean(abs)}
	}
	roots := make([]string, 0, len(byRoot))
	for root := range byRoot {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	files := []*os.File{}
	unlockers := []func(){}
	cleanup := func() {
		for i := len(unlockers) - 1; i >= 0; i-- {
			unlockers[i]()
		}
		for i := len(files) - 1; i >= 0; i-- {
			_ = files[i].Close()
		}
	}
	for _, root := range roots {
		poseDir := filepath.Join(root, ".pose")
		info, err := os.Lstat(poseDir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, poseDir) {
			cleanup()
			return nil, specTransferError("path-escape")
		}
		transfersDir := filepath.Join(poseDir, "transfers")
		if info, err := os.Lstat(transfersDir); err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !artifactPathWithin(root, transfersDir)) {
			cleanup()
			return nil, specTransferError("path-escape")
		}
		if err := os.MkdirAll(transfersDir, 0o755); err != nil {
			cleanup()
			return nil, specTransferError("lock-unavailable")
		}
		lockPath := filepath.Join(transfersDir, ".authority-transfer.lock")
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			cleanup()
			return nil, specTransferError("lock-unavailable")
		}
		if info, err := os.Lstat(lockPath); err != nil || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, lockPath) {
			_ = file.Close()
			cleanup()
			return nil, specTransferError("path-escape")
		}
		files = append(files, file)
		unlock, err := lockSpecTransferFile(file)
		if err != nil {
			cleanup()
			return nil, specTransferError("concurrent-transfer")
		}
		unlockers = append(unlockers, unlock)
	}
	return cleanup, nil
}

func cleanTransferRelative(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
