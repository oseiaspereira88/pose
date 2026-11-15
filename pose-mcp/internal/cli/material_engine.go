package cli

import (
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

func init() {
	posemodel.SetEngineVersion(version.Version)
	posemodel.SetCurrentMaterialAt(func(root string) string {
		// A shallow checkout has no commit-bound material: it reads HEAD
		// uncached, and is unreadable (review round 1, F1).
		git := func(limit int, args ...string) ([]byte, error) { return gitOutputBounded(root, limit, args...) }
		if shallow, err := repositoryIsShallow(git, root); err != nil || shallow {
			return ""
		}
		out, err := gitOutputBounded(root, 1024, "rev-parse", "--verify", "HEAD")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}, func(root, head string, in posemodel.MaterialInputs) string {
		git := func(limit int, args ...string) ([]byte, error) { return gitOutputBounded(root, limit, args...) }
		return materialSetDigestAt(git, root, head, in)
	})
}
