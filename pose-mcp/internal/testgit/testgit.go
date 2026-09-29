// Package testgit isolates the git processes a test binary starts (spec
// test-git-repos-run-no-background-maintenance).
//
// After a commit, git starts `maintenance run --auto` detached. It keeps
// repacking objects under .git/ after the command that triggered it has
// returned, so a test's TempDir cleanup can race it and fail with
// "unlinkat .../.git/objects: directory not empty" — observed on CI and locally
// under the full suite's load, in tests that commit many times. Pointing
// GIT_CONFIG_GLOBAL at a file that disables automatic maintenance removes the
// background writer. It also keeps the developer's ~/.gitconfig (hooks paths,
// signing, aliases) out of tests, which CI runners already lack.
package testgit

import (
	"os"
	"path/filepath"
)

const config = `[maintenance]
	auto = false
[gc]
	auto = 0
	autoDetach = false
`

// Isolate points GIT_CONFIG_GLOBAL at a config without automatic maintenance
// and returns a function that removes it. Call it from TestMain before m.Run.
func Isolate() (func(), error) {
	dir, err := os.MkdirTemp("", "pose-testgit-")
	if err != nil {
		return func() {}, err
	}
	path := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return func() {}, err
	}
	if err := os.Setenv("GIT_CONFIG_GLOBAL", path); err != nil {
		_ = os.RemoveAll(dir)
		return func() {}, err
	}
	return func() { _ = os.RemoveAll(dir) }, nil
}
