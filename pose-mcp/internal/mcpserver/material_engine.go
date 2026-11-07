package mcpserver

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

func init() { posemodel.SetEngineVersion(version.Version) }
