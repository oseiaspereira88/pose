//go:build !linux

package mcpserver

import "os"

// fileIdentity has no portable inode or status-change time off Linux; the
// size, mode and modification time remain the signature there.
func fileIdentity(info os.FileInfo) string { return "" }
