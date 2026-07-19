---
slug: pose-reproducible-release-verification
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-slsa-provenance, pose-release-compatibility-matrix
priority: 9
---

# Spec: Independent release verification

## 1. Intent

### Goal
verify release artifacts from a clean environment and quantify reproducibility limits.
### Business value
Catches packaging, provenance and compatibility failures the producer can miss.
### Constraints
- Report nondeterministic inputs explicitly.
### Non-goals
- Guarantee identical binaries across unsupported environments.

## 2. Requirements

### Functional
- R1: A separate job shall download, authenticate and inspect artifacts without producer state.
- R2: The verifier shall compare files, versions, checksums, signatures, SBOM and provenance.
- R3: Rebuild comparisons shall document deterministic matches and explained differences.

### Non-functional
- Keep verification isolated from producer credentials and caches.

### Security
- Execute artifacts only after identity and digest verification.

### Compatibility
- Exercise every supported target natively or with documented emulation.

## 3. Technical Plan

### Affected areas
- Independent workflow, container/VM fixtures and release report.

### API/contract changes
- Publish a consumer-verification procedure and result schema.

### Data/storage changes
- Retain verification logs and reproducibility deltas.

### Technical risks
- Sharing producer workflow or credentials creates circular evidence.

### Primary references
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)
- [The Update Framework](https://theupdateframework.io/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [SLSA 1.2](https://slsa.dev/spec/v1.2/): all gates ran inside the producer workflow (shared checkout/caches/credentials — circular evidence); Go rebuild determinism bounded by toolchain revision and buildid.

### Implementation
- [x] Create an independent verifier environment and trust policy: `.github/workflows/verify-release.yml` triggers on `release: published` and per-tag dispatch; `contents: read` only, `cache: false`, no producer secrets or build state; consumes only public release data (ADR `2026-07-19-independent-release-verification`). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Verify archive contents and all linked metadata before execution: `tests/release/independent-verify.sh` authenticates in layers — checksums, Sigstore bundles + SBOMs with the tag-pinned identity, SLSA provenance (`gh attestation verify --repo --signer-workflow`) — and only then extracts, compares `pose version` to the tag and runs install → doctor → strict gate on a fresh repository. ([reference](https://theupdateframework.io/))
- [x] Attempt controlled rebuilds and classify nondeterministic differences: the verifier rebuilds the platform binary from the tag source with the pipeline flags; GoReleaser builds now pin `-trimpath` + commit `mod_timestamp`; a digest match is reported as MATCH, a delta is reported with its explained inputs (toolchain revision, buildid) and classified as reproducibility delta, never authenticity failure. ([reference](https://slsa.dev/spec/v1.2/))

### Validation
- [x] Run `pose check --strict` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://theupdateframework.io/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-independent-release-verification.md` (Accepted): separate consumer-shaped workflow over producer-side checks (circular evidence) and over promised bit-reproducibility (honest measurement with explained deltas); execute only after full authentication; native-target execution with other targets covered by digest/signature/provenance verification.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-reproducible-release-verification --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version -run 'AttestedRelease' -count=1` — SUCCESS (verifier isolation and reproducibility inputs pinned by contract test).
- `bash -n tests/release/independent-verify.sh` — syntax clean; workflow YAML parsed cleanly.
- `pose check --strict` — SUCCESS; `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- The verifier's first real execution happens on the next published release (it needs live release assets and the attestation store); no verification result is claimed here.

## 7. Final Report

### Delivered scope

Independent `Verify release` workflow (clean environment, read-only, no
producer caches/credentials) running layered authentication — checksum,
signature, SBOM, provenance — before any execution; post-verification
functional inspection (version match, install → doctor → strict gate);
controlled rebuild with honest reproducibility reporting; reproducible-build
inputs pinned in GoReleaser; 400-day retained verification report; published
consumer procedure; ADR.

### Residual risks

- Producer and verifier live in the same repository — a repository-level
  compromise could alter both; mitigated partially by the public, stable
  procedure any third party can run.
- Only linux/amd64 executes natively in the verifier; remaining targets are
  covered by digest/signature/provenance verification.

### Follow-ups

- [open] Review the first Verify release run on the next published release: confirm all layers pass and record the reproducibility result (MATCH or explained delta).
- [covered: pose-upgrade-compatibility-lab] Broaden executed-target coverage (emulation/VMs) beyond the verifier's native platform.
