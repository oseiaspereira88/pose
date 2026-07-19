// Package version is the single authoritative source of the public POSE
// release version (spec pose-version-contract). Every public surface — CLI
// output, MCP serverInfo, telemetry and registry metadata contract tests —
// derives from Version; no other package may declare its own version literal.
package version

import "strings"

// devSuffix marks a binary that was not stamped by the release pipeline.
const devSuffix = "-dev"

// Version is the authoritative public version. Release builds stamp it from
// the git tag via -ldflags (see .goreleaser.yaml); development builds keep the
// devSuffix so they never impersonate a release.
var Version = "0.9.0" + devSuffix

// IsDevelopment reports whether this binary is an unstamped development build.
func IsDevelopment() bool { return strings.HasSuffix(Version, devSuffix) }

// ReleaseBase returns Version without the development suffix. This is the
// value public release metadata (server.json, archives) must carry for the
// next release; on stamped release builds it equals Version.
func ReleaseBase() string { return strings.TrimSuffix(Version, devSuffix) }
