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

// ExcludedTopLevel are repository-root entries that never enter the embedded
// scaffold. In the standalone repository the dist root is also the product's
// own source tree, so the product's code, CI, docs and published contracts are
// all excluded — an instance that installed POSE runs none of it.
var ExcludedTopLevel = []string{
	".git", ".github", ".gitignore", ".docs-site-build", ".idea",
	"pose-mcp", "mcp-enforce", "pose-action",
	"docs-site", "tests", "examples",
	".goreleaser.yaml", ".gitleaks.toml", "dist-release",
	// Contracts this repository publishes about itself, for consumers of the
	// product rather than for an instance of it.
	"compatibility.json", "compatibility-report.md", "composition-contract.json",
}

// ExcludedPrefixes are paths whose whole subtree is instance state rather than
// scaffold: embedding them would make an ordinary run drift the embed it was
// tested against.
var ExcludedPrefixes = []string{
	".pose/reports",
	".pose/capabilities",
	".pose/state",
}

// IsExcludedTop reports whether a repository-root entry is excluded.
func IsExcludedTop(top string) bool {
	return slices.Contains(ExcludedTopLevel, top)
}

// IsExcludedPath reports whether a slash-separated relative path falls inside
// an excluded subtree.
func IsExcludedPath(rel string) bool {
	for _, prefix := range ExcludedPrefixes {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}
