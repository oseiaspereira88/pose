//go:build windows

package pose

import "os"

// stateDirOwner is not checked on Windows, where ownership and mode bits are
// not the access model; the link refusal still applies (spec
// pose-delegation-task-detach-and-state, Security).
func stateDirOwner(os.FileInfo) (user, root bool) { return true, false }

func stateDirWritableByOthers(os.FileInfo, bool) bool { return false }
