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

// SelfReferentialPolicyFiles are `.pose/policy/` files whose live content in
// *this* repository is pose-dist's own governance configuration (dogfooding)
// — not a generic default. `delivery.json.roots` and `artifacts.json.
// governed_roots` name literal paths into pose-mcp's own source tree
// (pose-mcp/internal/cli, pose-mcp/internal/mcpserver, ...). Copying them
// byte-for-byte into the embedded scaffold made every `pose install`/`pose
// update --force` on a target project inherit roots that describe pose-mcp,
// not the target — silently emptying that project's delivery-integrity and
// artifact-contract graphs (issue #17).
//
// These two files are still shipped, but the generator (gen/main.go) writes
// an explicit neutral placeholder for them instead of copying pose-dist's
// live content; see distpolicy_test.go and scaffold_test.go for the
// contract this exclusion is paired with.
var SelfReferentialPolicyFiles = []string{
	"delivery.json",
	"artifacts.json",
}

// SelfReferentialIndexFiles are `.pose/indexes/` files whose live content in
// *this* repository describes pose-mcp's own dogfooded module graph — not a
// generic default. `module-metadata.json` names pose-mcp/mcp-enforce with
// owner @pose-maintainers; `validation-matrix.json`'s `moduleOverrides` names
// pose-mcp's own internal package paths and docs-site's Python override.
// Copying them byte-for-byte into the embedded scaffold made every fresh
// `pose install` on a target project inherit module entries and check
// overrides that describe pose-mcp's development repo, not the target
// (issue #22 — the same leak class `SelfReferentialPolicyFiles` closed for
// `.pose/policy/` under issue #17).
//
// Unlike the policy files, these two are not dropped from the sync entirely:
// `validation-matrix.json`'s `stacks` catalog (generic per-language checks)
// and `deliveryProfiles` (generic profile kinds) are legitimate, reusable
// defaults every instance should receive. Only the self-referential subtrees
// (`modules`, `defaults.owner`/`defaults.domain`, `moduleOverrides`) are
// neutralized — see NeutralIndexTemplates.
var SelfReferentialIndexFiles = []string{
	"module-metadata.json",
	"validation-matrix.json",
	// The remaining index files below are `pose index`'s own *computed*
	// output — repo-map.json, services.json and packages.json from
	// scanModules; spec-graph.json from this repository's own 100+ specs;
	// roadmaps.json from its own roadmaps; delivery-integrity.json from
	// buildCurrentDeliveryGraph; releases.json from this repository's own
	// release lifecycle (v1.4.0, in-progress evidence, ...). None of it
	// describes a fresh target project. Before this exclusion, `pose
	// install` never surfaced the leak in practice only because it always
	// calls `cmdIndex` again moments later, silently overwriting whatever
	// was just seeded — but a plain `pose update` (spec
	// pose-upgrade-path-audit-fixes, R2) seeds `.pose/indexes/*.json` too
	// and, unlike install, never called cmdIndex afterward, so the leak
	// reached a real instance for the first time (spec
	// pose-derived-index-self-referential-leak).
	"repo-map.json",
	"services.json",
	"packages.json",
	"spec-graph.json",
	"roadmaps.json",
	"delivery-integrity.json",
	"releases.json",
	// extensions.lock.json records pose-dist's own two locally-installed
	// rule extensions (pose-rule-backend-go, pose-rule-frontend-react) —
	// not something `cmdIndex` regenerates; the neutral empty registry is
	// this file's actual final state until the target's own operator runs
	// `pose extension install`, same as module-metadata.json's empty
	// `modules: {}` waits for discovery/manual curation.
	"extensions.lock.json",
}

// ExtensionOnlyRuleFiles are `.pose/rules/` files that ship exclusively
// through `pose extension install`, never through core machinery — the
// pattern `pose-rule-kubernetes` established. pose-dist installs some of
// these locally into its own `.pose/rules/` for its own dogfooded review
// needs (e.g. `pose-rule-backend-go`, since pose-mcp is itself a Go
// backend): without this exclusion, that install would make the wholesale
// `.pose/rules/` sync re-embed the file into every fresh instance, silently
// undoing the extension migration (spec
// pose-domain-rule-extension-migration). The file still exists on disk here
// and still satisfies `pose review`'s rule-presence check
// (review_closeout.go's validateReviewContractRefs) for pose-dist's own
// reviews — it just never leaves this repository through the embedded dist.
var ExtensionOnlyRuleFiles = []string{
	"backend-go.md",
	"frontend-react.md",
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
	if !slices.Contains(IncludedPoseSubtrees, parts[1]) {
		return false
	}
	if parts[1] == "policy" && len(parts) == 3 && slices.Contains(SelfReferentialPolicyFiles, parts[2]) {
		return false
	}
	if parts[1] == "indexes" && len(parts) == 3 && slices.Contains(SelfReferentialIndexFiles, parts[2]) {
		return false
	}
	if parts[1] == "rules" && len(parts) == 3 && slices.Contains(ExtensionOnlyRuleFiles, parts[2]) {
		return false
	}
	return true
}

// NeutralPolicyTemplates returns the embedded content for the
// SelfReferentialPolicyFiles IsIncluded excludes from the byte-for-byte
// sync. pose-dist's own delivery.json/artifacts.json describe THIS
// repository's roots (dogfooding); shipping that content verbatim made every
// installed instance inherit roots pointing at pose-mcp's own source tree
// instead of the target project's (issue #17). Each placeholder here is
// schema-valid and disabled with empty roots — the loaders in
// internal/pose (delivery_surface.go, delivery_integrity.go) tolerate empty
// roots/governed_roots as a no-op — until the target project configures its
// own. Both gen/main.go (writes these into the embedded scaffold) and
// scaffold_test.go (asserts the embedded copy still matches) call this, so
// the two can never drift from each other the way the generator and the
// drift guard once did before this package existed.
func NeutralPolicyTemplates() map[string][]byte {
	return map[string][]byte{
		".pose/policy/delivery.json": []byte(`{
  "_comment": "Delivery-integrity policy (spec pose-delivery-surface-assurance). Disabled by default: roots/entrypoints must name this project's own source paths, not pose-mcp's. Set enabled=true and populate roots once configured for this project.",
  "schema_version": 1,
  "enabled": false,
  "adopted_at": "",
  "results_path": ".pose/results/delivery-validation.json",
  "roots": [],
  "severities": {
    "unconnected-artifact": "error",
    "unconsumed-capability": "error",
    "surface-without-provenance": "error",
    "surface-without-reachability": "error",
    "undeclared-delivery": "error",
    "stale-evidence": "error",
    "roadmap-criterion": "error"
  }
}
`),
		".pose/policy/artifacts.json": []byte(`{
  "_comment": "Artifact-contract policy (spec pose-artifact-provenance-ledger). Disabled by default: governed_roots must name this project's own source directories, not pose-mcp's. Set enabled=true and populate governed_roots once configured for this project.",
  "schema_version": 1,
  "enabled": false,
  "adopted_at": "",
  "governed_roots": [],
  "exclusions": [],
  "none_reasons": ["release", "migration"],
  "severities": {
    "resolvability": "error",
    "existence": "error",
    "action-mismatch": "error",
    "undeclared": "error",
    "orphan": "warning",
    "legacy-narrative": "info"
  }
}
`),
	}
}

// NeutralIndexTemplates returns the embedded content for the
// SelfReferentialIndexFiles IsIncluded excludes from the byte-for-byte sync.
// Unlike NeutralPolicyTemplates, these placeholders are not fully empty
// shells: validation-matrix.json's `stacks` catalog and `deliveryProfiles`
// are generic, reusable content every instance should receive as-is — only
// `modules`, `defaults.owner`/`defaults.domain` (module-metadata.json) and
// `moduleOverrides` (validation-matrix.json) are neutralized, since those
// are the subtrees that named pose-mcp's own module graph (issue #22).
func NeutralIndexTemplates() map[string][]byte {
	return map[string][]byte{
		".pose/indexes/module-metadata.json": []byte(`{
  "schemaVersion": 1,
  "defaults": {
    "owner": "",
    "criticality": "medium",
    "domain": "",
    "validationProfile": "baseline"
  },
  "modules": {}
}
`),
		".pose/indexes/validation-matrix.json": []byte(`{
  "defaults": {
    "mode": "strict"
  },
  "deliveryProfiles": {
    "cli-surface": {
      "kind": "surface",
      "requiredEvidenceClasses": ["reachability"],
      "anyEvidenceClasses": ["integration", "e2e"]
    },
    "composed-capability": {
      "kind": "capability",
      "requiredEvidenceClasses": ["integration"]
    },
    "api-contract": {
      "kind": "contract",
      "requiredEvidenceClasses": ["integration"]
    },
    "release-governance": {
      "kind": "governance",
      "requiredEvidenceClasses": ["integration"]
    },
    "backend-go": {
      "kind": "governance",
      "requiredEvidenceClasses": ["integration"]
    }
  },
  "stacks": {
    "node": {
      "checks": [
        {"name": "lint", "program": "npm", "args": ["run", "lint", "--if-present"], "severity": "optional"},
        {"name": "test", "program": "npm", "args": ["test", "--if-present"], "severity": "required"},
        {"name": "build", "program": "npm", "args": ["run", "build", "--if-present"], "severity": "required"},
        {"name": "typecheck", "program": "npm", "args": ["run", "typecheck", "--if-present"], "severity": "optional"}
      ]
    },
    "go": {
      "checks": [
        {"name": "test", "program": "go", "args": ["test", "./..."], "severity": "required", "evidenceClass": "unit"},
        {"name": "vet", "program": "go", "args": ["vet", "./..."], "severity": "optional", "evidenceClass": "build"}
      ]
    },
    "rust": {
      "checks": [
        {"name": "test", "program": "cargo", "args": ["test"], "severity": "required"}
      ]
    },
    "java": {
      "checks": [
        {"name": "maven-test", "program": "mvn", "args": ["-B", "test"], "severity": "required", "when": {"fileExists": "pom.xml"}},
        {"name": "gradle-test", "program": "./gradlew", "args": ["test"], "severity": "required", "when": {"fileExists": "gradlew"}},
        {"name": "gradle-test-wrapper", "program": "gradle", "args": ["test"], "severity": "required", "when": {"fileExists": "build.gradle", "fileNotExists": "gradlew"}},
        {"name": "gradle-test-wrapper-kts", "program": "gradle", "args": ["test"], "severity": "required", "when": {"fileExists": "build.gradle.kts", "fileNotExists": "gradlew"}}
      ]
    },
    "python": {
      "checks": [
        {"name": "poetry-test", "program": "poetry", "args": ["run", "pytest", "-q"], "severity": "required", "when": {"fileExists": "poetry.lock"}},
        {"name": "pipenv-test", "program": "pipenv", "args": ["run", "pytest", "-q"], "severity": "required", "when": {"fileExists": "Pipfile", "fileNotExistsAny": ["poetry.lock"]}},
        {"name": "pip-test", "program": "pytest", "args": ["-q"], "severity": "required", "when": {"fileExists": "requirements.txt", "fileNotExistsAny": ["poetry.lock", "Pipfile"]}},
        {"name": "setuptools-test", "program": "pytest", "args": ["-q"], "severity": "required", "when": {"fileExists": "setup.py", "fileNotExistsAny": ["poetry.lock", "Pipfile", "requirements.txt"]}},
        {"name": "pep517-test", "program": "pytest", "args": ["-q"], "severity": "optional", "when": {"fileExists": "pyproject.toml", "fileNotExistsAny": ["poetry.lock", "Pipfile", "requirements.txt", "setup.py"]}}
      ]
    },
    "dotnet": {
      "checks": [
        {"name": "dotnet-test", "program": "dotnet", "args": ["test"], "severity": "required"}
      ]
    },
    "cloudflare-workers": {
      "checks": [
        {"name": "wrangler-version", "program": "wrangler", "args": ["--version"], "severity": "optional"}
      ]
    }
  },
  "moduleOverrides": {}
}
`),
		// The eight placeholders below (spec
		// pose-derived-index-self-referential-leak) are byte-for-byte what
		// `pose index` itself computes for a target with zero specs/modules/
		// roadmaps/releases (verified empirically — a fresh `pose install`
		// on an empty repository produces exactly this, digests included;
		// the delivery-integrity digests are sha256 over an empty claims/
		// change-sets set, deterministic, not random/timestamped). They
		// exist only as the state between seeding and the `cmdIndex` call
		// that immediately follows it (seedAbsentInstanceConfig) — a
		// consumer that reads them in that narrow window sees a valid,
		// honestly-empty target, never pose-dist's own graph.
		".pose/indexes/repo-map.json": []byte(`{
  "root": ".",
  "apps": [],
  "services": [],
  "packages": [],
  "manifests": [],
  "dockerfiles": [],
  "helmCharts": [],
  "readmes": [],
  "moduleMetadata": {
    "schemaVersion": 1,
    "source": ".pose/indexes/module-metadata.json"
  }
}
`),
		".pose/indexes/services.json": []byte("[]\n"),
		".pose/indexes/packages.json": []byte("[]\n"),
		".pose/indexes/spec-graph.json": []byte(`{
  "schemaVersion": 1,
  "specs": {},
  "edges": []
}
`),
		".pose/indexes/roadmaps.json": []byte(`{
  "schemaVersion": 1,
  "roadmaps": {}
}
`),
		".pose/indexes/delivery-integrity.json": []byte(`{
  "schema_version": 1,
  "input_digest": "sha256:9cdc22ecf5b519b1ed2e91fed2b7ebe9eb29abe3417191f2a500f0cedc1158ea",
  "provenance_digest": "sha256:aade89ae2026afe39ab153cd6a6b5a3c34e87a69a4c677daf467b942da699414",
  "nodes": [],
  "edges": [],
  "claims": [],
  "change_sets": [],
  "reverse": {},
  "findings": []
}
`),
		".pose/indexes/releases.json": []byte(`{
  "pending": [],
  "releases": []
}
`),
		// extensions.lock.json is not computed by cmdIndex — this empty
		// registry is its actual final state until the target's own
		// operator runs `pose extension install` (see the comment on
		// SelfReferentialIndexFiles).
		".pose/indexes/extensions.lock.json": []byte(`{
  "schema_version": 1,
  "extensions": {}
}
`),
	}
}

// StripDynamicContributorSection removes local instance contributor-mode sections
// from AGENTS.md or POSE.md when syncing or testing base scaffold manuals.
func StripDynamicContributorSection(content string) string {
	marker := "<!-- pose:contributor-mode -->"
	if !strings.Contains(content, marker) {
		return content
	}
	type docSection struct {
		Heading string
		Body    []string
	}
	var preamble []string
	var sections []docSection
	var current docSection
	var buf []string
	flush := func() {
		if current.Heading == "" {
			preamble = buf
			return
		}
		current.Body = buf
		sections = append(sections, current)
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = docSection{Heading: line}
			buf = nil
			continue
		}
		buf = append(buf, line)
	}
	flush()

	var kept []docSection
	for _, sec := range sections {
		isContrib := false
		for _, line := range sec.Body {
			if strings.Contains(line, marker) {
				isContrib = true
				break
			}
		}
		if !isContrib {
			kept = append(kept, sec)
		}
	}

	out := append([]string{}, preamble...)
	for _, sec := range kept {
		out = append(out, sec.Heading)
		out = append(out, sec.Body...)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

// StripDynamicContributorSectionBytes is the byte slice variant of StripDynamicContributorSection.
func StripDynamicContributorSectionBytes(b []byte) []byte {
	return []byte(StripDynamicContributorSection(string(b)))
}


