package cli

// Stack-to-extension resolution (spec pose-adaptive-rule-delivery,
// github.com/oseiaspereira88/pose issues #21/#24). A curated, small
// mapping — not a catalog — matching what actually exists today: two
// content extensions produced by pose-domain-rule-extension-migration
// plus the pre-existing pose-rule-kubernetes. `node` deliberately does
// not map to a stack name alone: pose-rule-frontend-react is about React
// specifically, and a Node.js backend with no React dependency must never
// be recommended it (R3) — the exact "wrong rule for the stack" complaint
// issue #21/#24 raised about the old embedded-by-default machinery.

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ruleExtensionByStack maps a stack-detection domain (module-metadata.json's
// "domain" field, as written by seedModuleMetadataFromDiscovery) to the
// extension ID that applies unconditionally — every stack needing an extra
// content check (currently only "node") is resolved by resolveRuleExtension
// instead of appearing here.
var ruleExtensionByStack = map[string]string{
	"go": "pose-rule-backend-go",
}

// resolveRuleExtension resolves the rule extension matching a discovered
// module's stack, or ok=false when no extension applies — including when
// the stack is real but no extension has been authored for it yet (Rust,
// Python, etc — R2), and when the manifest-level signal is too coarse on
// its own (Node without a confirmed React dependency — R3).
func resolveRuleExtension(root, modulePath, stack string) (string, bool) {
	if id, ok := ruleExtensionByStack[stack]; ok {
		return id, true
	}
	if stack == "node" && hasReactDependency(root, modulePath) {
		return "pose-rule-frontend-react", true
	}
	return "", false
}

// hasReactDependency reports whether modulePath's package.json lists
// "react" under dependencies or devDependencies. Deliberately a content
// check, not a presence check: the whole point is distinguishing a React
// frontend from a plain Node.js backend, which package.json's mere
// existence cannot do.
func hasReactDependency(root, modulePath string) bool {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(modulePath), "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return false
	}
	if _, ok := pkg.Dependencies["react"]; ok {
		return true
	}
	_, ok := pkg.DevDependencies["react"]
	return ok
}

// ruleExtensionFile maps an extension ID to the rule file it installs
// under .pose/rules/, for the doctor advisory's "already installed?" check.
var ruleExtensionFile = map[string]string{
	"pose-rule-backend-go":     "backend-go.md",
	"pose-rule-frontend-react": "frontend-react.md",
}

// ruleExtensionInstalled reports whether the extension's rule file is
// already present at .pose/rules/<file>.md — either because the operator
// already installed it, or because it predates the migration to
// extensions and still lives on disk untouched (deliverMachinery never
// deletes a retired file). Either way, no advisory is needed.
func ruleExtensionInstalled(root, ruleFile string) bool {
	_, err := os.Stat(filepath.Join(root, ".pose", "rules", ruleFile))
	return err == nil
}
