---
slug: pose-first-release-evidence-confirmation
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on:
priority: 1
components: release, security, docs
delivers:
---

# Spec: Confirm on a real release what nine follow-ups were waiting for

## 1. Intent

### Goal
Execute the nine deferred confirmations that all shared one precondition — a
first genuinely published release — now that v0.18.2 is tagged, published and
verified.

### Business value
This is the highest-ratio work in the backlog: it closes four of the five `high`
items and costs verification, not implementation. More importantly, every one of
these covers a supply-chain claim POSE already advertises — Sigstore signatures,
SLSA provenance, CycloneDX SBOMs, reproducible rebuild, package channels — that
has never been checked against a real published artifact by anyone other than
the workflow that produced it. Until they run, the guarantees are asserted, not
demonstrated.

v0.18.2 already produced the raw material: 30 assets with digests, an attested
provenance, SBOMs per archive, and a `Verify release` run reporting a
bit-identical rebuild. What is missing is independent confirmation and the
recording of the result.

### Constraints
- Confirmation must run against the published artifacts, not a local rebuild:
  the point is to check what users actually download.
- Findings are recorded whether they pass or fail. A failed confirmation is a
  result to act on, never a reason to re-cut the release.

### Non-goals
- Fixing whatever these confirmations reveal. Each finding gets its own
  disposition; this spec establishes the baseline.
- Re-running the release. v0.18.2 stands.

---

## 2. Requirements

### Functional
- R1: `gh attestation verify` shall be run against every published archive and
  `checksums.txt` of v0.18.2, and its evidence recorded.
- R2: The `Verify release` run for v0.18.2 shall be reviewed layer by layer, and
  its reproducibility result recorded as MATCH or as an explained delta.
- R3: A `workflow_dispatch` snapshot rehearsal shall confirm that signing and
  verification both pass in the release environment.
- R4: The first `security.yml` and `scorecard.yml` runs shall be triaged and the
  baseline OpenSSF score recorded.
- R5: The published SBOMs shall be inspected to confirm syft resolves the
  replaced `mcp-enforce` module path, and detected licenses reconciled against
  `NOTICE`.
- R6: The `package-channels.yml` clean-host run shall be confirmed against
  v0.18.2, and the first WinGet manifest submitted with its observed
  publication lag recorded in `package-channels.md`.
- R7: The `docs.yml` `mkdocs build --strict` run shall be confirmed to pass,
  including the monorepo-recipes nav entry and its internal links.

### Non-functional
- Each confirmation records where it ran and against which digest, so a later
  reader can tell evidence from assertion.

### Security
- Verification uses only the published artifacts and their pinned identities. No
  producer credential or cache participates in a confirmation step.

### Compatibility
- No product change. This spec produces evidence and dispositions.

---

## 3. Technical Plan

### Affected areas
- `.pose/reports/` — recorded confirmation evidence.
- `docs-site/docs/package-channels.md` — observed WinGet publication lag (R6).
- The nine originating specs — each follow-up gets its disposition.

### Artifacts
<!-- Declared at closeout, once the evidence files exist. -->
- created: .pose/specs/pose-first-release-evidence-confirmation/spec.md

### API/contract changes
- None.

### Technical risks
- R3's rehearsal consumes release-environment credentials; it must run as
  `workflow_dispatch` on a snapshot, never against the v0.18.2 tag.
- R6 depends on WinGet review latency, which is outside the project's control.
  Its result may be "submitted, pending" rather than a closed loop.

---

## 4. Tasks

### Planning
- [ ] List each confirmation with the exact command and the artifact digest it
      applies to

### Implementation
- [ ] R1: attestation verify across the six archives and checksums.txt
- [ ] R2: layer-by-layer review of the Verify release run
- [ ] R3: snapshot rehearsal of sign + verify
- [ ] R4: triage security/scorecard runs, record the baseline
- [ ] R5: SBOM inspection and NOTICE reconciliation
- [ ] R6: package-channels clean-host run and WinGet submission
- [ ] R7: docs strict build confirmation

### Validation
- [ ] Every confirmation recorded with its command, digest and outcome
- [ ] Every originating follow-up dispositioned

---

## 6. Validation

### Strategy
Each requirement is a command run against published v0.18.2 artifacts, with its
output retained as evidence. The spec is done when all seven have an outcome and
each originating follow-up has moved off `open` — including outcomes that are
negative, which become their own follow-ups rather than blockers here.

### Deterministic checks

#### Security / Contract
- Command: `gh attestation verify <asset> --repo oseiaspereira88/pose`
- Scope: every published v0.18.2 archive plus checksums.txt
- Expected: verification passes against the expected signer workflow

### Requirement trace
<!-- Filled at closeout. -->

### Known gaps
- R6's WinGet half cannot be closed synchronously; expect it to land as a
  recorded submission with a pending publication, revisited later.

---

## 7. Final Report

<!-- Filled at closeout. -->

### Follow-ups

- [open] If any confirmation fails, decide whether the advertised guarantee should be softened in the docs until it is fixed — an unverified claim in README or docs-site is worse than an absent one. (owner:@pose-maintainers crit:medium review:2026-09-18)
