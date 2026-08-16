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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// seedAbsentInstanceConfig seeds every engine-owned config file a fresh
// instance needs that is currently absent: `.pose/indexes/*.json`,
// module-metadata's discovered modules, and `.pose/policy`/`.pose/
// review-profiles`. Every step is additive-only — an existing file is never
// touched — so it is safe to call unconditionally on both `pose install`
// (always) and `pose update` (with or without --force): skipping it on a
// plain update was the root cause of an instance whose refreshed
// AGENTS.md/POSE.md already reference these subsystems while nothing had
// ever seeded them, so it reported "Result: SUCCESS" and then failed its own
// very next `pose check --strict` with broken references that `pose doctor`
// did not catch (spec pose-update-instance-config-completeness).
func seedAbsentInstanceConfig(dist fs.FS, target string, log func(english, portuguese string, a ...any)) {
	idxEntries, _ := fs.ReadDir(dist, ".pose/indexes")
	_ = os.MkdirAll(filepath.Join(target, ".pose", "indexes"), 0o755)
	for _, e := range idxEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		dst := filepath.Join(target, ".pose", "indexes", e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := copyFile(dist, ".pose/indexes/"+e.Name(), dst, 0o644); err == nil {
			log("index (seed): %s", "índice (semente): %s", e.Name())
		}
	}

	// Discover this repository's actual modules and merge them into
	// module-metadata.json (spec pose-stack-detection-consolidation) —
	// additive only, never overwrites an existing entry.
	seedModuleMetadataFromDiscovery(target, log)

	// Seed governed configuration contracts for a fresh repository. These
	// are user-owned after installation, so reruns never overwrite them;
	// engine defaults still need to exist for direct adoption of review
	// bundles and the validation/delivery policies they consume.
	for _, dir := range []string{".pose/policy", ".pose/review-profiles"} {
		entries, _ := fs.ReadDir(dist, dir)
		_ = os.MkdirAll(filepath.Join(target, filepath.FromSlash(dir)), 0o755)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			dst := filepath.Join(target, filepath.FromSlash(dir), e.Name())
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			if err := copyFile(dist, dir+"/"+e.Name(), dst, 0o644); err == nil {
				log("config (seed): %s/%s", "configuração (semente): %s/%s", dir, e.Name())
			}
		}
	}
}

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
