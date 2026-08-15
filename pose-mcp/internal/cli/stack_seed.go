package cli

// Module-metadata seeding from discovered stacks (spec
// pose-stack-detection-consolidation, github.com/oseiaspereira88/pose
// issue #21). A fresh `pose install`/`pose init` previously left
// `.pose/indexes/module-metadata.json` at its neutral, empty template
// (spec pose-scaffold-index-template-neutralization) with no path forward
// to the target repository's actual modules — `scanModules` already
// discovers modules recursively into `repo-map.json` on every `pose
// index`, but nothing ever connected that discovery to module-metadata.json,
// which stayed static until hand-edited.
//
// This reuses discoverValidationModules (already recursive, already covers
// the node/go/rust/java/python/dotnet manifest union, already the source
// `pose init --wizard` uses for validation-matrix.json) rather than adding
// a fifth scanner — consolidating onto the widest-coverage existing walk
// instead of hand-rolling a new one.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// seedModuleMetadataFromDiscovery merges newly-discovered module paths into
// target's module-metadata.json, leaving every existing entry (whether
// hand-authored or seeded by an earlier install) untouched. Safe to call on
// every install/update: additive only, per-entry, never destructive.
func seedModuleMetadataFromDiscovery(target string, log func(english, portuguese string, a ...any)) {
	path := filepath.Join(target, ".pose", "indexes", "module-metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return // no module-metadata.json to merge into (e.g. --skip-mcp-only fixtures)
	}
	var doc struct {
		SchemaVersion int                          `json:"schemaVersion"`
		Defaults      map[string]string            `json:"defaults"`
		Modules       map[string]map[string]string `json:"modules"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return // don't fight a hand-edited or malformed file
	}
	if doc.Modules == nil {
		doc.Modules = map[string]map[string]string{}
	}
	discovered, err := discoverValidationModules(target)
	if err != nil {
		return
	}
	added := []string{}
	for _, m := range discovered {
		if _, exists := doc.Modules[m.Rel]; exists {
			continue
		}
		doc.Modules[m.Rel] = map[string]string{
			"criticality":       "medium",
			"domain":            m.Stack,
			"validationProfile": "baseline",
		}
		added = append(added, m.Rel)
	}
	if len(added) == 0 {
		return
	}
	sort.Strings(added)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	out = append(out, '\n')
	if writeAtomic(path, out, 0o644) == nil && log != nil {
		for _, rel := range added {
			log("module-metadata (discovered): %s", "module-metadata (descoberto): %s", rel)
		}
	}
}
