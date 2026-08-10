// Package distpolicy declares what the embedded scaffold excludes.
//
// The generator and the drift guard both need this list, and both used to carry
// their own copy — the test's even said "Mirrors gen/main.go", which is an
// admission, not a mechanism. Adding an exclusion to one and not the other
// leaves the generator producing a tree the guard then rejects.
//
// It lives in its own package rather than in `scaffold` because `scaffold`
// embeds `dist/` with go:embed: the generator that *creates* that directory
// cannot depend on a package that requires it to already exist.
package distpolicy

import (
	"slices"
	"strings"
)

// IncludedTopLevel is the allowlist: only these repository-root entries enter
// the embedded scaffold.
//
// It was a denylist until a published product contract nearly shipped to every
// instance because nobody remembered to exclude it. Inclusion-by-default makes
// the failure silent — a new product file becomes distribution — while
// exclusion-by-default makes it loud: a new scaffold file is simply missing, and
// the drift guard says so.
//
// The list is what `pose install` and `pose upgrade` actually read. Everything
// else in this repository is the product's own material: its specs, ADRs,
// reviews, changelogs, release manifests, CI, tests and tooling. An instance
// runs none of it, and embedding it made the binary carry megabytes it never
// opened.
var IncludedTopLevel = []string{
	// Managed manuals, plus their translations.
	"AGENTS.md",
	"POSE.md",
	"locales",
	// Machinery the engine owns and delivers (machineryRoots).
	".agents",
	// MCP client configuration seeded into an instance.
	".mcp.json",
	// Read by the installer for redistribution.
	"install.sh",
	// Legal texts, vendored under .pose/ by the installer.
	"LICENSE",
	"NOTICE",
	".pose",
}

// IncludedPoseSubtrees narrows `.pose/` to the machinery and contract files an
// instance needs. The rest of `.pose/` is this project's own governance record.
var IncludedPoseSubtrees = []string{
	"workflows",
	"rules",
	"templates",
	"indexes",
	"policy",
	"review-profiles",
	"schema-version",
	"release-policy.json",
	"LICENSE",
	"NOTICE",
}

// IsIncluded reports whether a slash-separated repository-relative path belongs
// in the embedded scaffold.
func IsIncluded(rel string) bool {
	if rel == "" || rel == "." {
		return true
	}
	parts := strings.SplitN(rel, "/", 3)
	if !slices.Contains(IncludedTopLevel, parts[0]) {
		return false
	}
	if parts[0] != ".pose" || len(parts) == 1 {
		return true
	}
	return slices.Contains(IncludedPoseSubtrees, parts[1])
}
