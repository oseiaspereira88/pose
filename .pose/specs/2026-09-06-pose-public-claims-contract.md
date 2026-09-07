---
slug: pose-public-claims-contract
status: draft
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-release-recovery-verification
priority: 1
components: cli, pose-mcp, ci
delivers: contract:public-claims
---

# Spec: Public claims contract

## 1. Intent

### Goal
Give POSE a machine-checkable contract for the claims its public surfaces
make — version, canonical docs route, download URL, license — so a surface
that contradicts a released fact fails a gate instead of aging quietly.

### Business value
The audit found the same stale version in eleven files across two
repositories: the landing hero, the `<title>`, the Schema.org
`softwareVersion`, the homepage pill, a shared footer repeated across ten
pages, the docs home, and a "What's new in v1.4.3" README section — while the
product shipped 1.7.10.

Fixing those strings once is an hour of work that decays immediately. The
durable fix is that no human has to remember five places. POSE exists to make
this class of drift detectable; not applying it to POSE's own public surfaces
is the most visible possible failure to dogfood.

### Constraints
- Must work offline. Resolving claims cannot require a network call, or it
  cannot run in the same gate that runs before a release exists.
- The parent `harne8` repository owns the site; POSE cannot reach into it.
  The checker consumes a declared surface manifest, so either repository can
  run it against its own files.

### Non-goals
- Rewriting site content. This spec supplies the gate; the narrative rewrite
  is `pose-canonical-positioning` and the landing spec in the site repository.
- Auto-editing surfaces. The gate reports; a human or an explicit command fixes.

---

## 2. Requirements

### Functional
- R1: A `.pose/public/claims.yaml` contract shall declare the canonical
  product name, category, license, repository URL, canonical docs route and
  release source — each as a single authoritative value.
- R2: `pose public-claims` shall verify every surface declared in the contract
  and report, per surface, which claim is contradicted and what the
  authoritative value is.
- R3: The check shall fail when a surface asserts a version that is not the
  latest released version, when it links to a non-canonical docs host, or when
  it publishes a download URL that does not resolve in the release contract.
- R4: A surface shall be able to declare that it deliberately carries no
  version claim, and the check shall then fail if a version string appears
  there — the evergreen case must be enforceable, not merely recommended.
- R5: `--json` output shall be emitted for CI consumption, and a non-zero exit
  under `--strict`.

### Non-functional
- No network access in the default path; the latest released version is read
  from the release contract already present in the repository.

### Compatibility
- Additive command. No existing behavior changes.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/` — new `public-claims` command
- `.pose/public/claims.yaml` — new contract
- `.github/workflows/ci.yml` — gate wiring

### Artifacts
- created: .pose/public/claims.yaml
- created: pose-mcp/internal/cli/publicclaims.go
- created: pose-mcp/internal/cli/publicclaims_test.go
- modified: .github/workflows/ci.yml

### Delivery targets
- contract:public-claims module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- A checker that is too eager produces false failures on prose that merely
  mentions a historical version (a changelog, an assessment snapshot). The
  contract must distinguish a *claim about the current product* from a
  *reference to a past release*, which is why surfaces are declared explicitly
  rather than discovered by scanning every file for version-shaped strings.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Contract schema and loader (R1)
- [ ] Increment 2: `public-claims` command and per-surface verification (R2, R3)
- [ ] Increment 3: Evergreen-surface assertion (R4)
- [ ] Increment 4: `--json` / `--strict` and CI wiring (R5)

---

## 5. Decisions

### Decision 1
- Date: 2026-09-06
- Context: how much of the version should appear on marketing surfaces at all.
- Options considered: (a) keep the exact version and regenerate it on release;
  (b) remove version claims from marketing surfaces and show the current
  version only next to the install command, resolved at view time.
- Decision: (b) for hero, title and metadata; (a) only next to install.
- Rationale: a marketing page should not age on every patch. Most of the
  observed drift is not a maintenance failure but a design failure — the
  version was placed where it had no reason to be. R4 makes the absence
  enforceable, so the claim cannot creep back in.
- Consequences: the Schema.org `softwareVersion` field is dropped rather than
  automated; `downloadUrl` already points at `releases/latest`, which is the
  evergreen equivalent.

---

## 6. Validation

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run PublicClaims -count=1`
- Expected: exit 0, covering a drifted surface, a compliant one, and an
  evergreen surface that wrongly gained a version string

#### Security / Contract
- Command: `pose public-claims --strict --json`
- Expected: exit 0 on a compliant tree; non-zero naming the offending surface
  and claim on a drifted one

### Requirement trace
<!-- Filled at closeout. -->

---

## 7. Final Report

### Follow-ups

- [open]
