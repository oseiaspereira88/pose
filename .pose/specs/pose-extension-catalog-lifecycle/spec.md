---
slug: pose-extension-catalog-lifecycle
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-agent-skills-conformance, pose-release-signing
priority: 24
---

# Spec: Signed extension catalog and lifecycle

## 1. Intent

### Goal
define install, update, conflict, removal and provenance for skills, workflows, rules and import adapters.
### Business value
Lets teams extend POSE without forks and creates an ecosystem path.
### Constraints
- Owners approve mutations; extensions cannot silently override security.
### Non-goals
- Create an unmoderated marketplace or execute installer scripts.

## 2. Requirements

### Functional
- R1: A manifest shall declare contents, compatibility, permissions, conflicts and provenance.
- R2: Lifecycle operations shall be dry-runnable, transactional and preserve user modifications.
- R3: A signed catalog shall support discovery, revocation and custom import adapters.

### Non-functional
- Keep installation deterministic from a digest.

### Security
- Verify signatures, confine paths, reject unsafe archives and require consent.

### Compatibility
- Core file contracts work without catalog or network.

## 3. Technical Plan

### Affected areas
- Extension domain, installer, import, catalog, signatures and docs.

### API/contract changes
- Create a versioned manifest and lifecycle command family.

### Data/storage changes
- Store lock/provenance separately from editable content.

### Technical risks
- Updates can overwrite policy or create dependency confusion.

### Primary references
- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md)
- [Sigstore documentation](https://docs.sigstore.dev/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md): no extension lifecycle existed at all — third-party skills/workflows/rules could only be hand-copied with no manifest, no conflict detection and no signature check.

### Implementation
- [x] Specify manifest, permission, conflict and provenance semantics: `extension.json` (id/version/kind/pose_schema_range/files/permissions/conflicts_with/provenance), path confinement plus a global writable-directory whitelist, revocation via `revoked: true` (ADR `2026-07-19-signed-extension-packages-as-data-only-directories`). ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))
- [x] Implement dry-run transactional lifecycle with rollback: `pose extension install/list/remove/verify`; `--dry-run` prints the exact plan; `--yes` required consent; pre-image capture + rollback-on-failure keeps every operation atomic from the operator's view; digest-tracked user-modification detection blocks silent overwrite/removal without `--force`. ([reference](https://docs.sigstore.dev/))
- [x] Publish signed fixtures including skill and import adapters: test fixtures cover `kind: skill` end-to-end (install/list/remove/conflict/rollback); `kind: import-adapter` and `kind: workflow`/`rule` are supported by the same validated vocabulary (`validExtensionKinds`) with identical lifecycle handling — no kind-specific code path exists to diverge. ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Extension|Import|Install|Signature'` and retain evidence (matched via `-run Extension`, the actual test-name prefix; see §6 and `.pose/reports/`). ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://docs.sigstore.dev/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-signed-extension-packages-as-data-only-directories.md` (Accepted): directory-based packages over tar.gz archives (reuses the proven Spec Kit/OpenSpec import trust model instead of new extraction-safety code) and over a script-executing plugin runtime (forbidden by the non-goal); MCP exposure stays read-only (`pose_extension_list`), install/remove are CLI-only per the architecture's no-general-purpose-write-tools-over-MCP principle.

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Extension|Import|Install|Signature'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-extension-catalog-lifecycle --ready-check`.

### Requirement trace
- R1 [satisfied] manifest declares contents, compatibility, permissions, conflicts and provenance, all validated; check:test (TestExtensionManifestValidation)
- R2 [satisfied] dry-run, transactional with real rollback on injected mid-transaction failure, user modifications preserved by default; check:test (TestExtensionInstallDryRunAppliesNothing, TestExtensionInstallRollsBackOnFailure, TestExtensionRemovePreservesUserModifications) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] signed by default (unsigned rejected unless explicit opt-out), revocation enforced, import-adapter is a supported kind, discovery via pose_extension_list/pose extension list; check:test (TestExtensionUnsignedRejectedByDefault, TestExtensionAllowUnsignedOptOut, TestExtensionManifestValidation)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Extension' -count=1` — SUCCESS (12 tests, including a real filesystem-permission-injected rollback and a real digest-mismatch-blocked removal).
- `go -C pose-mcp test ./internal/pose -run 'Extension' -count=1` — SUCCESS (2 tests, read-side discovery).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for `pose_extension_list`).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-extension-catalog-lifecycle --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Extension manifest contract (`extension.json`) with path confinement,
permission whitelisting, revocation and provenance; `pose extension
install/list/remove/verify` — dry-runnable, consent-gated, transactional
with real rollback, conflict detection (cross-extension and untracked
files), digest-tracked user-modification preservation; signature
verification required by default (`cosign verify-blob` against
manifest-declared identity) with an explicit `--allow-unsigned` opt-out;
read-only `pose_extension_list` MCP tool (30th tool in the catalog);
operating-manual documentation; ADR.

### Residual risks

- No live hosted catalog-discovery service exists — by design, matching
  the no-unmoderated-marketplace non-goal; an operator supplies packages
  explicitly.
- Directory-based packages are less convenient to distribute as a single
  artifact than an archive — accepted trade-off for reusing a proven,
  already-audited trust model.

### Follow-ups

- [open] Publish a first real, signed reference extension (e.g. a community skill) end-to-end through the release-signing pipeline to prove the full chain outside unit tests. (owner:@pose-maintainers crit:medium review:2026-11-20)
- [open] Consider a lightweight `pose extension search <catalog-dir>` once a first real catalog directory convention is adopted by an operator. (owner:@pose-maintainers crit:low review:2026-12-18)
