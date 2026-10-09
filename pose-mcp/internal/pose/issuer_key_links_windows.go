//go:build windows

package pose

import "os"

// extraHardLinks is not checked on Windows, where FileInfo carries no link
// count; the symlink and permission checks still apply.
func extraHardLinks(os.FileInfo) bool { return false }
