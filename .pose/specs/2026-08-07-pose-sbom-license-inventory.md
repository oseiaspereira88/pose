---
slug: pose-sbom-license-inventory
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-cyclonedx-sbom
priority: 1
components: release
delivers: governance:sbom-license-inventory
---

# Spec: The SBOM carries the licenses it is advertised to carry

## 1. Intent

### Goal
Make the published CycloneDX SBOMs resolve component licenses, so the artifact
matches what `pose-cyclonedx-sbom` R1 requires and what the documentation
advertises.

### Business value
`pose-cyclonedx-sbom` R1 asks for an SBOM "with versions, hashes and known
licenses". The published SBOMs carry versions and hashes and, until now, a
license for exactly one component out of 27. When that spec's trace was
retrofitted, R1 had to be recorded as waived for that reason.

Meanwhile `docs-site/docs/architecture.md` and `capability-assessment.md` sell
"signed, SBOM'd, provenance-attested releases". A license inventory that
inventories nothing is the kind of claim a downstream audit checks first.

The cause is mechanical: syft scans the packaged binary, which carries module
names and versions in the Go buildinfo but no license texts. Those live in the
module sources, which the release build has already downloaded.

### Constraints
- The SBOM must keep describing the packaged artifact, not the source tree.

### Non-goals
- Hand-maintaining a license list, or adding a dependency-license policy gate.

---

## 2. Requirements

### Functional
- R1: The published SBOMs shall resolve licenses for the third-party components
  they list.
- R2: Components with no resolvable license shall be limited to the project's
  own modules and the binary itself, all covered by the repository LICENSE.

### Non-functional
- No new tool: the same syft invocation, reading the module cache the build
  already populated.

### Security
- Reading the local module cache introduces no network fetch and no new trust.

### Compatibility
- The SBOM gains license fields; its format and identity are unchanged.

---

## 3. Technical Plan

### Affected areas
- `.goreleaser.yaml` — the SBOM generation environment.

### Artifacts
- created: .pose/specs/pose-sbom-license-inventory/spec.md
- modified: .goreleaser.yaml

### Delivery targets
- governance:sbom-license-inventory module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- License resolution depends on the module cache being populated when syft
  runs. In the release job the build precedes SBOM generation, so it is; a
  future reordering would silently return to an empty inventory.

---

## 4. Tasks

### Planning
- [x] Establish why licenses were absent before changing anything

### Implementation
- [x] R1: enable local module-cache license search
- [x] R2: confirm which components remain unresolved and why

### Validation
- [x] Measure license coverage before and after against a real binary
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Measure rather than assume: scan a locally built binary with syft, count
components carrying a license, enable the setting, and count again. Then name
every component still unresolved and confirm each is covered by the repository
LICENSE.

### Deterministic checks

#### Security / Contract
- Command: `syft scan <binary> --output cyclonedx-json` with and without `SYFT_GOLANG_SEARCH_LOCAL_MOD_CACHE_LICENSES`
- Scope: license coverage across SBOM components
- Expected: coverage rises from 1/27 to 24/27

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5; syft via the anchore/syft image, since it
  is not installed on this host.
- Notes: without the setting, 1 of 27 components carried a license (`stdlib`).
  With it, 24 of 27. The three remaining are `github.com/harne8/mcp-enforce`,
  `github.com/harne8/pose-mcp` and the binary itself — the project's own code,
  covered by the repository LICENSE.

### Results summary
- Successes: license coverage measured before and after on a real binary
- Failures: none
- Warnings: the published SBOM only gains licenses from the next cut onward

### Requirement trace
- R1 [satisfied] governance:sbom-license-inventory evidence:integration check:delivery-integration check:sbom-license-coverage — enabling local module-cache license search raises coverage from 1/27 to 24/27 on a real scan of the built binary
- R2 [satisfied] check:sbom-license-coverage — the three unresolved components are the two project modules and the binary, all covered by the repository LICENSE rather than by a third-party license

### Known gaps
- The measurement was made locally against the same binary the release builds,
  not against a published SBOM. The first published SBOM with licenses is the
  next cut.
- Nothing asserts license coverage in CI, so a regression to an empty inventory
  would be silent.

---

## 7. Final Report

### Delivered scope
SBOM generation resolves licenses from the module cache the build populates,
taking published SBOMs from one licensed component to twenty-four of
twenty-seven, with the remainder being the project's own code.

### Files and modules changed
- .goreleaser.yaml

### Validation executed
- Command: syft scan with and without the setting, counting licensed components
- Result: 1/27 → 24/27

### Residual risks
- If SBOM generation ever runs before the build, the inventory silently empties
  again and nothing would catch it.

### Follow-ups

- [covered: pose-sbom-license-coverage-gate] Assert a minimum license coverage in the artifact-identity gate, so an SBOM that silently stops resolving licenses fails the release instead of shipping empty. (owner:@pose-maintainers crit:medium review:2026-10-02)
