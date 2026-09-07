---
slug: pose-release-recovery-verification
status: draft
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-release-security-gate-integrity
priority: 0
components: ci, release
delivers: governance:release-recovery-verification
---

# Spec: Release recovery verification

## 1. Intent

### Goal
Prove, from a clean environment and without maintainer knowledge, that the
published install command produces a working POSE — and keep proving it on
every subsequent release.

### Business value
`pose-release-security-gate-integrity` removes the two defects that blocked
the pipeline. It does not establish that publication works: the steps after
the security gate — cosign signing, goreleaser publish, `verify.sh`,
package-manager manifests — have not executed successfully since **v1.6.0**.
They are unproven, not proven-good.

The launch funnel's narrowest point is a stranger copying one command. That
path has to be exercised the way a stranger exercises it, not the way a
maintainer with a warm toolchain does.

### Constraints
- The verification must run on a machine with no Go toolchain, no repository
  checkout and no cached modules. Verifying on a developer machine proves
  nothing about the visitor's experience.
- Tags v1.7.1 through v1.7.10 are left as published. They stay assetless.

### Non-goals
- Retroactively attaching artifacts to the failed tags.
- Changing what the release publishes.

---

## 2. Requirements

### Functional
- R1: A `workflow_dispatch` snapshot run shall complete every step after the
  security gate without publishing, before any recovery tag is pushed.
- R2: A recovery release shall publish the complete artifact set — archives
  for each supported OS/arch pair, `checksums.txt`, per-archive CycloneDX
  SBOMs, signatures, `compatibility.json` and `install.sh`.
- R3: `curl -fsSL .../releases/latest/download/install.sh` shall return HTTP
  200, and the script it returns shall install a binary whose `pose version`
  matches the released tag.
- R4: The installed binary shall pass `pose doctor` in a directory that is not
  a POSE project and in a freshly initialized one.
- R5: A scheduled job shall re-run R3 against `releases/latest` on a recurring
  basis and fail loudly, so a future assetless release is detected within a
  day rather than by an audit ten releases later.

### Non-functional
- The clean-environment check runs in a container image with no Go toolchain
  and no repository checkout.

### Security
- Installation is verified against the published checksum and signature, not
  merely by "the binary ran".

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/release.yml`
- `.github/workflows/verify-release.yml`
- `tests/install/`

### Artifacts
- modified: .github/workflows/verify-release.yml
- modified: tests/install/run.sh
- created: tests/install/clean-environment.sh

### Delivery targets
- governance:release-recovery-verification module:. profile:release-governance entrypoint:.github/workflows/verify-release.yml

### Technical risks
- A latent failure in the never-exercised publish steps. This is precisely
  what the snapshot rehearsal in R1 exists to surface, before a tag is burned
  on discovering it.

---

## 4. Tasks

### Planning
- [ ] Confirm the recovery version number and whether it is a patch or minor

### Implementation
- [ ] Increment 1: Run the snapshot rehearsal and fix whatever it surfaces (R1)
- [ ] Increment 2: Push the recovery tag and confirm the artifact set (R2)
- [ ] Increment 3: Clean-container install verification (R3, R4)
- [ ] Increment 4: Scheduled latest-release liveness check (R5)

### Validation
- [ ] Verify from a container with no toolchain and no checkout

---

## 6. Validation

### Strategy
Rehearse without publishing, then publish, then verify as an outsider. Each
step is allowed to fail cheaply before the next one becomes expensive.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/verify.sh dist-release`
- Scope: signatures, artifact identity, SBOM completeness
- Expected: exit 0

#### Test
- Command: `bash tests/install/clean-environment.sh`
- Scope: container with no Go toolchain and no repository checkout
- Expected: installs from the published URL; `pose version` matches the tag;
  `pose doctor` exits 0

### Requirement trace
<!-- Filled at closeout. -->

---

## 7. Final Report

### Follow-ups

- [open]
