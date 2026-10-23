package pose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProvenAbsent says whether the path is not there, and proves it: every directory above it is
// looked at without being followed blindly. It returns (true, nil) only when an entry on the way
// (or the path itself) does not exist; (false, nil) when the path is there; and an error when
// something on the way cannot be looked into: a link whose target was lost, a loop, a permission.
// An empty result for a path that could not be looked at is a loss, not an absence.
func ProvenAbsent(path string) (bool, error) {
	return ProvenAbsentIn(NativeFiles(), path)
}

func ProvenAbsentIn(files Files, path string) (bool, error) {
	clean, absErr := filepath.Abs(path)
	if absErr != nil {
		// A relative path is searched from the working directory it was written for; when that
		// cannot be known, nothing is proven.
		return false, fmt.Errorf("pose: %s could not be made absolute: %w", path, absErr)
	}
	volume := filepath.VolumeName(clean)
	rest := strings.TrimPrefix(clean[len(volume):], string(filepath.Separator))
	current := volume + string(filepath.Separator)
	if rest == "" {
		return false, nil
	}
	parts := strings.Split(rest, string(filepath.Separator))
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := files.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("pose: %s could not be inspected: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if _, serr := files.Stat(current); serr != nil {
				return false, fmt.Errorf("pose: %s is a link whose target cannot be read: %w", current, serr)
			}
		}
		if i < len(parts)-1 && !info.IsDir() {
			// A file where a directory belongs: what is below it is not there, and not provably so.
			if resolved, serr := files.Stat(current); serr == nil && !resolved.IsDir() {
				return false, fmt.Errorf("pose: %s is not a directory", current)
			}
		}
	}
	return false, nil
}
