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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// computedIndexFiles are the `.pose/indexes/*.json` files `cmdIndex` itself
// (re)computes from the target's own current state on every run. The
// embedded scaffold ships a neutral empty-shell placeholder for each
// (distpolicy.NeutralIndexTemplates) — the seed step below writes that
// placeholder when absent, purely so a consumer never sees a missing file;
// it is never the final content. See seedAbsentInstanceConfig.
var computedIndexFiles = map[string]bool{
	"repo-map.json":           true,
	"services.json":           true,
	"packages.json":           true,
	"spec-graph.json":         true,
	"roadmaps.json":           true,
	"delivery-integrity.json": true,
	"releases.json":           true,
}

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
	seededComputedIndex := false
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
			if computedIndexFiles[e.Name()] {
				seededComputedIndex = true
			}
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

	// Migrate existing review policy / profiles from v1 to v2 if present
	migrateInstanceReviewPolicy(dist, target, log)
	stampContractAdoption(target, time.Now().UTC(), log)
	stampChangelogAdoption(target, time.Now().UTC(), log)

	// The neutral placeholders just seeded above for repo-map.json,
	// spec-graph.json, delivery-integrity.json etc. are honestly empty, not
	// this target's real state — cmdIndex is the one thing that computes
	// them correctly, and policy/review-profiles (just seeded) are what it
	// needs to do that. `pose install` already called cmdIndex again a few
	// steps after this either way; this closes the same gap for a plain
	// `pose update`, which never did (spec
	// pose-derived-index-self-referential-leak).
	if seededComputedIndex {
		_ = cmdIndex(target, nil, io.Discard, io.Discard)
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
	// A pre-existing entry keyed by the project directory's own name (e.g.
	// a repository at .../acme installed with a curated "acme" entry),
	// when that key does not resolve to a real subdirectory, is the
	// common hand-curation
	// convention for aliasing the project root — discovering "." fresh
	// would otherwise add a second entry for the exact same physical
	// directory under a different key. This heuristic is deliberately
	// narrow (an exact name-of-the-root match, not "any orphaned key") to
	// avoid mistaking genuinely stale entries for root aliases (spec
	// pose-discovery-gitignore-and-root-alias-fix).
	rootAlreadyAliased := false
	if rootAlias := filepath.Base(target); doc.Modules[rootAlias] != nil {
		if _, err := os.Stat(filepath.Join(target, rootAlias)); err != nil {
			rootAlreadyAliased = true
		}
	}

	added := []string{}
	for _, m := range discovered {
		if _, exists := doc.Modules[m.Rel]; exists {
			continue
		}
		if m.Rel == "." && rootAlreadyAliased {
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

// stampChangelogAdoption records the date this instance received the changelog
// contract, in its own policy (spec pose-changelog-adoption-is-the-instances).
//
// The shipped policy used to carry this repository's own adoption date, and
// every instance inherited it. For a project starting today that reads as "gate
// everything", which is harmless by accident; for a project migrating in with a
// history of specs, it silently exempts every spec completed before a date that
// belongs to someone else.
//
// Stamping is what the review contracts already do, and it is the only answer
// that is true: the engine cannot know when a project adopted the contract, but
// it can know when it first put the policy there.
func stampChangelogAdoption(target string, now time.Time, log func(english, portuguese string, a ...any)) {
	path := filepath.Join(target, ".pose", "policy", "changelog.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Through the raw document, so a key the engine does not model survives the
	// write — the same reason stampContractAdoption does.
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	if adopted, ok := doc["adopted_at"].(string); !ok || strings.TrimSpace(adopted) != "" {
		return
	}
	doc["adopted_at"] = now.Format(time.DateOnly)
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	if writeAtomic(path, append(updated, '\n'), 0o644) == nil && log != nil {
		log("policy (changelog adoption): adopted_at=%s — specs completed before today keep their exemption",
			"política (adoção do changelog): adopted_at=%s — specs concluídas antes de hoje mantêm sua isenção",
			now.Format(time.DateOnly))
	}
}

// stampContractAdoption records, in an instance's own review policy, the date
// it received each governance contract that judges work retroactively.
//
// `.pose/policy/` is not machinery, so an update delivers the engine but never
// the policy: an instance receives a stricter rule and no statement of when it
// arrived, and every closeout recorded under the previous contract fails. In
// POSE's own repository that was 106 of them, and the only reason it did not
// happen here is that the date was added by hand in the same change.
//
// The contracts come from posemodel.ReviewContracts, so a future one is stamped
// by adding it there rather than by teaching this function about it.
//
// Today is the honest date, not `adopted_at`: work reviewed between adopting
// POSE and receiving a contract was reviewed under the previous one. The stamp
// is additive — a date the instance already recorded, in the map or in the
// legacy field, is never touched, and neither is one deliberately cleared.
func stampContractAdoption(target string, now time.Time, log func(english, portuguese string, a ...any)) {
	policyPath := filepath.Join(target, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		return
	}
	var policy posemodel.ReviewPolicy
	if json.Unmarshal(raw, &policy) != nil {
		return // don't fight a hand-edited or malformed file
	}
	// Round-tripping through the typed struct would drop any key the engine
	// does not model, so the write goes through the raw document.
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	adoptions, _ := doc["contract_adoptions"].(map[string]any)
	if adoptions == nil {
		adoptions = map[string]any{}
	}
	stamped := []string{}
	for _, contract := range posemodel.ReviewContracts() {
		if policy.ContractAdoptionRecorded(contract.ID) {
			continue
		}
		// An explicitly empty value — in the map or in the legacy field — is a
		// decision, not an absence: the instance is saying to judge its whole
		// history by the current contract. The typed policy renders both as "",
		// so the raw document is the only place that distinction survives, and
		// stamping over it would reverse the choice on every update.
		if _, present := adoptions[contract.ID]; present {
			continue
		}
		if legacy := posemodel.LegacyContractField(contract.ID); legacy != "" {
			if _, present := doc[legacy]; present {
				continue
			}
		}
		// Prefer the legacy key where the contract has one. An engine older
		// than the registry rejects a review policy carrying `contract_adoptions`
		// outright — the policy decoder disallows unknown fields — so stamping
		// the map would make this repository unreadable to any binary still on
		// the previous release: a developer who has not updated, a pinned CI.
		// The legacy field says the same thing to both.
		if legacy := posemodel.LegacyContractField(contract.ID); legacy != "" {
			doc[legacy] = now.Format(time.DateOnly)
		} else {
			adoptions[contract.ID] = now.Format(time.DateOnly)
		}
		stamped = append(stamped, contract.ID)
	}
	if len(stamped) == 0 {
		return
	}
	sort.Strings(stamped)
	if len(adoptions) > 0 {
		doc["contract_adoptions"] = adoptions
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	if writeAtomic(policyPath, append(updated, '\n'), 0o644) == nil && log != nil {
		for _, id := range stamped {
			log("policy (contract adoption): %s=%s — closeouts reviewed before today keep their approval",
				"política (adoção de contrato): %s=%s — closeouts revisados antes de hoje mantêm sua aprovação",
				id, now.Format(time.DateOnly))
		}
	}
}

// shippedProfileIsCurrentSchema reports whether the profile the distribution
// carries is already at the schema an instance is being migrated to. When it is
// not, replacing the instance's file with it migrates nothing, and the honest
// path is the explicit rewrite below rather than a log line.
func shippedProfileIsCurrentSchema(raw []byte) bool {
	var profile struct {
		SchemaVersion int `json:"schema_version"`
	}
	return json.Unmarshal(raw, &profile) == nil && profile.SchemaVersion == posemodel.ReviewPolicySchemaVersion
}

// migrateInstanceReviewPolicy upgrades existing schema-v1 review policies and
// review profiles to schema-v2 in an idempotent manner.
func migrateInstanceReviewPolicy(dist fs.FS, target string, log func(english, portuguese string, a ...any)) {
	policyPath := filepath.Join(target, ".pose", "policy", "review.json")
	if raw, err := os.ReadFile(policyPath); err == nil {
		var p map[string]any
		if err := json.Unmarshal(raw, &p); err == nil {
			v, _ := p["schema_version"].(float64)
			if int(v) == 1 {
				p["schema_version"] = 2
				if _, ok := p["component_aware"]; !ok {
					p["component_aware"] = true
				}
				if caa, _ := p["component_aware_adopted_at"].(string); caa == "" {
					if adopted, _ := p["adopted_at"].(string); adopted != "" {
						p["component_aware_adopted_at"] = adopted
					} else {
						p["component_aware_adopted_at"] = "2026-08-13"
					}
				}
				if ucb, _ := p["unmapped_component_behavior"].(string); ucb == "" {
					p["unmapped_component_behavior"] = "warning"
				}
				if _, ok := p["review_bundles"]; !ok {
					p["review_bundles"] = true
				}
				if rba, _ := p["review_bundles_adopted_at"].(string); rba == "" {
					if adopted, _ := p["adopted_at"].(string); adopted != "" {
						p["review_bundles_adopted_at"] = adopted
					} else {
						p["review_bundles_adopted_at"] = "2026-08-14"
					}
				}
				if _, ok := p["allow_criterion_reuse"]; !ok {
					p["allow_criterion_reuse"] = true
				}
				if profiles, ok := p["profiles"].(map[string]any); ok {
					if _, hasMilestone := profiles["milestone"]; !hasMilestone {
						profiles["milestone"] = "milestone-integration@1"
					}
					if _, hasRoadmap := profiles["roadmap"]; !hasRoadmap {
						profiles["roadmap"] = "roadmap-outcome@1"
					}
				}
				if _, ok := p["reviewer_independence"]; !ok {
					p["reviewer_independence"] = map[string]any{
						"spec":      "same-actor-separate-execution",
						"milestone": "same-actor-separate-execution",
						"roadmap":   "same-actor-separate-execution",
					}
				}
				if _, ok := p["overlay_profiles"]; !ok {
					p["overlay_profiles"] = []any{"backend-review@1", "frontend-review@1"}
				}

				if updatedRaw, err := json.MarshalIndent(p, "", "  "); err == nil {
					_ = writeAtomic(policyPath, append(updatedRaw, '\n'), 0o644)
					if log != nil {
						log("policy (migrated): .pose/policy/review.json (v1 -> v2)", "política (migrada): .pose/policy/review.json (v1 -> v2)")
					}
				}
			}
		}
	}

	// Upgrade review-profiles in target if they are schema v1
	profEntries, _ := os.ReadDir(filepath.Join(target, ".pose", "review-profiles"))
	for _, e := range profEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		profPath := filepath.Join(target, ".pose", "review-profiles", e.Name())
		if raw, err := os.ReadFile(profPath); err == nil {
			var prof map[string]any
			if err := json.Unmarshal(raw, &prof); err == nil {
				sv, _ := prof["schema_version"].(float64)
				if int(sv) == 1 {
					distProfPath := ".pose/review-profiles/" + e.Name()
					// Only claim the migration if the file that replaces it is
					// actually v2. Two shipped profiles were themselves v1, so
					// this branch copied a v1 file over a v1 file and logged a
					// migration that never happened — on every `pose update`,
					// forever, with the operator told it had succeeded (spec
					// pose-shipped-review-profiles-are-schema-v2).
					if distRaw, err := fs.ReadFile(dist, distProfPath); err == nil && shippedProfileIsCurrentSchema(distRaw) {
						_ = writeAtomic(profPath, distRaw, 0o644)
						if log != nil {
							log("review-profile (migrated): %s (v1 -> v2)", "perfil de review (migrado): %s (v1 -> v2)", e.Name())
						}
					} else {
						prof["schema_version"] = 2
						if updatedRaw, err := json.MarshalIndent(prof, "", "  "); err == nil {
							_ = writeAtomic(profPath, append(updatedRaw, '\n'), 0o644)
							if log != nil {
								log("review-profile (migrated): %s (v1 -> v2)", "perfil de review (migrado): %s (v1 -> v2)", e.Name())
							}
						}
					}
				}
			}
		}
	}
}
