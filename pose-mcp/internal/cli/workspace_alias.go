package cli

// --workspace <name> resolution (spec pose-monorepo-validation-advisory,
// R3): documented sugar over --module <path> for the case where the
// operator knows a workspace member's package/crate name but not its
// relative directory. Resolves by reading each candidate module's own
// manifest — package.json's "name" for node, Cargo.toml's [package] name
// for rust — never by inference across modules.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

// resolveWorkspaceModulePath finds the module among modules whose manifest
// declares name, returning its Rel path. Only node and rust are supported:
// those are the two ecosystems whose own tooling uses "workspace"
// terminology (npm/pnpm/Yarn workspaces, Cargo workspaces) — the case R3
// and the motivating issue are both about.
func resolveWorkspaceModulePath(root string, modules []validationModule, name string) (string, bool) {
	for _, m := range modules {
		switch m.Stack {
		case "node":
			if packageJSONName(root, m.Rel) == name {
				return m.Rel, true
			}
		case "rust":
			if cargoTomlName(root, m.Rel) == name {
				return m.Rel, true
			}
		}
	}
	return "", false
}

func packageJSONName(root, modRel string) string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(modRel), "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return ""
	}
	return pkg.Name
}

var cargoPackageNameRe = regexp.MustCompile(`(?m)^\s*name\s*=\s*"([^"]+)"\s*$`)

// cargoTomlName extracts [package].name via a targeted line scan rather
// than a full TOML parser — Cargo.toml's structure for this one field is
// simple and stable enough that pulling in a TOML dependency for a single
// string field would be disproportionate.
func cargoTomlName(root, modRel string) string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(modRel), "Cargo.toml"))
	if err != nil {
		return ""
	}
	m := cargoPackageNameRe.FindSubmatch(raw)
	if m == nil {
		return ""
	}
	return string(m[1])
}
