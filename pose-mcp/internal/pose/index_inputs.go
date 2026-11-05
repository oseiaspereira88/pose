package pose

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func validationResultsBindingIn(files Files, root string) (string, bool) {
	files = FilesOrNative(files)
	policy, err := LoadDeliveryPolicyIn(files, root)
	if err != nil {
		return "", false
	}
	results := []DeliveryValidationResult{}
	if policy.ResultsPath != "" {
		path := filepath.Join(root, filepath.FromSlash(policy.ResultsPath))
		raw, readErr := files.ReadFile(path)
		if readErr == nil {
			results, err = ParseDeliveryValidationRun(raw, policy.ResultsPath)
			if err != nil {
				return "", false
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return "", false
		}
	}
	return ReleaseDigest(results), true
}

func smokeResultsBindingIn(files Files, root string) (string, bool) {
	results, err := loadSmokeResultsInputIn(Store{Root: root, FS: files}, root)
	if err != nil {
		return "", false
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return ReleaseDigest(results), true
}

func releaseStatusBindingIn(files Files, root string) (string, bool) {
	status, err := (Store{Root: root, FS: files}).GetReleaseStatus("")
	if err != nil || status == nil {
		return "", false
	}
	return ReleaseDigest(status), true
}
