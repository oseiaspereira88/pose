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
	"go":                 "pose-rule-backend-go",
	"python":             "pose-rule-backend-python",
	"rust":               "pose-rule-backend-rust",
	"java":               "pose-rule-backend-java",
	"dotnet":             "pose-rule-backend-dotnet",
	"cloudflare-workers": "pose-rule-serverless-cloudflare",
}

// resolveRuleExtension resolves the rule extension matching a discovered
// module's stack, or ok=false when no extension applies — including when
// the stack is real but no extension has been authored for it yet (R2),
// and when the manifest-level signal is too coarse on its own (Node without
// a confirmed React/Vue/Svelte dependency — R3).
func resolveRuleExtension(root, modulePath, stack string) (string, bool) {
	if id, ok := ruleExtensionByStack[stack]; ok {
		return id, true
	}
	if stack == "node" {
		if hasReactDependency(root, modulePath) {
			return "pose-rule-frontend-react", true
		}
		if hasVueDependency(root, modulePath) {
			return "pose-rule-frontend-vue", true
		}
		if hasSvelteDependency(root, modulePath) {
			return "pose-rule-frontend-svelte", true
		}
	}
	return "", false
}

// hasReactDependency reports whether modulePath's package.json lists
// "react" under dependencies or devDependencies.
func hasReactDependency(root, modulePath string) bool {
	return hasPackageDependency(root, modulePath, "react")
}

// hasVueDependency reports whether modulePath's package.json lists
// "vue" or "nuxt" under dependencies or devDependencies.
func hasVueDependency(root, modulePath string) bool {
	return hasPackageDependency(root, modulePath, "vue", "nuxt", "@vue/runtime-core")
}

// hasSvelteDependency reports whether modulePath's package.json lists
// "svelte" or "@sveltejs/kit" under dependencies or devDependencies.
func hasSvelteDependency(root, modulePath string) bool {
	return hasPackageDependency(root, modulePath, "svelte", "@sveltejs/kit")
}

func hasPackageDependency(root, modulePath string, names ...string) bool {
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
	for _, name := range names {
		if _, ok := pkg.Dependencies[name]; ok {
			return true
		}
		if _, ok := pkg.DevDependencies[name]; ok {
			return true
		}
	}
	return false
}

// ruleExtensionFile maps an extension ID to the rule file it installs
// under .pose/rules/, for the doctor advisory's "already installed?" check.
var ruleExtensionFile = map[string]string{
	"pose-rule-backend-go":          "backend-go.md",
	"pose-rule-backend-python":      "backend-python.md",
	"pose-rule-backend-rust":        "backend-rust.md",
	"pose-rule-backend-java":        "backend-java.md",
	"pose-rule-backend-dotnet":      "backend-dotnet.md",
	"pose-rule-serverless-cloudflare": "serverless-cloudflare.md",
	"pose-rule-infra-terraform":     "infra-terraform.md",
	"pose-rule-frontend-react":      "frontend-react.md",
	"pose-rule-frontend-vue":        "frontend-vue.md",
	"pose-rule-frontend-svelte":     "frontend-svelte.md",
	"pose-rule-infra-docker":        "infra-docker.md",
	"pose-rule-cicd-github-actions": "cicd-github-actions.md",
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
