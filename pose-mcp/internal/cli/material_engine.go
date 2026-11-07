package cli

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

func init() {
	posemodel.SetEngineVersion(version.Version)
	posemodel.SetCurrentMaterialDigests(func(root string, in posemodel.MaterialInputs) string {
		git := func(limit int, args ...string) ([]byte, error) { return gitOutputBounded(root, limit, args...) }
		return materialSetDigestAt(git, root, "HEAD", in)
	})
}
