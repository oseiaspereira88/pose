---
slug: pose-first-release-evidence-confirmation
status: in-progress
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

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, `gh` against the published v0.18.2 release; every
  artifact downloaded from the provider, none rebuilt locally.
- Notes: R1, R2, R5 and R7 ran against published assets. R4 read the OpenSSF
  API result for commit `01af76e`. R6 had to be dispatched by hand — see below.
  R3 could not be dispatched from this session.

### Results summary
- Successes: R1 (7/7 attestations), R2 (6 layers + bit-identical rebuild), R7
  (strict docs build clean)
- Failures: R6 — `package-channels.yml` is a dead gate that also does not run
- Warnings: R4 baseline is 4.2 with four checks at 0; R5 shows an SBOM that
  resolves almost no licenses
- Blocked: R3

### Requirement trace
- R1 [satisfied] check:attestation-verify report:.pose/reports/2026-08-07-standard-first-release-evidence-confirmation.md — `gh attestation verify` passes for all six archives and checksums.txt; SLSA v1 predicate, signer `release.yml@refs/tags/v0.18.2`, source digest f62c014
- R2 [satisfied] check:verify-release-run — run 31148431844: checksums, Sigstore+SBOM pinned identity, SLSA provenance (archive and checksums), version match, install→doctor→check, and a bit-identical rebuild (sha256 7b5e82da…)
- R3 [waived: dispatching the Release workflow was denied by this session's policy; it needs a maintainer to run it] — the workflow_dispatch path is already the snapshot rehearsal (`release --clean --snapshot --skip=publish`), so the rehearsal is one manual dispatch away
- R4 [satisfied] check:openssf-scorecard — baseline recorded: aggregate 4.2 on 2026-08-07. Zeroes triaged below; two are real hardening findings, two are artefacts of repository age and review model
- R5 [satisfied] check:sbom-inspection — syft resolves the replaced module path `github.com/harne8/mcp-enforce` but not its version (`UNKNOWN`, no license), and only 1 of 27 components carries a license at all. Reconciliation against NOTICE is therefore vacuous today
- R6 [withdrawn: the confirmation cannot be performed as written — the gate has never run and fails immediately when forced] — see the finding below; a new spec has to repair the workflow before this confirmation means anything
- R7 [satisfied] check:mkdocs-strict — run 31148183904 built the site with `--strict` and no warning; the `Monorepo recipes` nav entry resolves (mkdocs.yml:36)

### Findings

**F1 — `package-channels.yml` is a gate that has never executed (severity: high).**
It triggers on `release: published`, but a release created by the workflow's own
`GITHUB_TOKEN` does not emit an event that starts another workflow run. The
workflow shows zero runs across every release the project has cut. `Verify
release` escaped this only because it also carries a `workflow_run` trigger.

**F2 — and it fails on its first line when forced (severity: high).**
Dispatched by hand against v0.18.2 (run 31199572548), both the macOS and Windows
legs died in 26s: `go run ./pose-mcp/cmd/pose …` executes from the repository
root, which is not a Go module — the module lives in `pose-mcp/`. Every other
call site in the repository uses `go -C pose-mcp run ./cmd/pose`. So the clean-
host package verification has never worked, and no WinGet manifest was ever
generated to submit.

**F3 — untrusted code checkout in `verify-release.yml` (severity: medium).**
Scorecard's `Dangerous-Workflow` is 0 because the verify job checks out
`${{ github.event.workflow_run.head_branch || … }}` under a `workflow_run`
trigger. The blast radius is small — the job holds only `contents: read` and
`attestations: read` and no secrets — but the pattern lets any ref starting with
`v` execute in the base-repository context.

**F4 — `release.yml` grants `contents: write` at top level (severity: low).**
Only the publish step needs it; scoping it to that job would take
`Token-Permissions` off zero.

**F5 — the SBOM resolves almost no licenses (severity: medium).**
26 of 27 components carry no license field, and `mcp-enforce` has no version
either because it is a directory `replace`. The follow-up asked to reconcile
detected licenses against NOTICE; there is nothing to reconcile, and NOTICE
itself is 12 lines with no dependency section. The advertised SBOM is
structurally valid CycloneDX 1.6 and materially uninformative.

**Not findings.** `Maintained` is 0 because the repository is under 90 days old,
and `Code-Review` is 0 because a single maintainer pushes to `main` — both are
accurate descriptions, not defects. `Branch-Protection` is 0 because `main` is
unprotected, which is a maintainer's call, not a bug.

### Known gaps
- R6's WinGet submission never became possible: the manifest generator never
  ran. The item moves to the repair spec rather than staying here.
- R3 remains unproven until a maintainer dispatches the Release workflow.

---

## 7. Final Report

### Delivered scope
Five of the seven confirmations ran and passed or produced a recorded baseline.
One (R6) could not be performed as written because the thing it confirms has
never worked, and one (R3) needs a dispatch this session was not permitted to
make. The supply-chain claims that could be checked — Sigstore signatures, SLSA
provenance, checksums, reproducible rebuild — hold against the published
artifacts. The claims that could not be checked are now named rather than
assumed.

### Files and modules changed
- .pose/specs/pose-first-release-evidence-confirmation/spec.md
- the nine originating follow-ups, each dispositioned

### Validation executed
- Command: `gh attestation verify <asset> --repo oseiaspereira88/pose`
- Result: PASS for six archives and checksums.txt

### Residual risks
- The package-manager channel is advertised in the README and docs but has
  never been verified end to end on a clean host. Until F1 and F2 are fixed,
  that install path is undemonstrated.
- The published SBOMs satisfy the format but carry almost no license data, so
  any downstream license audit that trusts them will conclude nothing.

### Follow-ups

- [open] F1+F2: repair `package-channels.yml` — give it a trigger that actually fires (the `workflow_run` pattern `verify-release.yml` uses) and fix `go run ./pose-mcp/cmd/pose` to `go -C pose-mcp run ./cmd/pose` — then re-run this confirmation and submit the first WinGet manifest. (owner:@pose-maintainers crit:high review:2026-08-21)
- [open] F3: narrow `verify-release.yml`'s checkout so a `workflow_run` cannot execute an arbitrary `v*` ref in the base-repository context, or document why the read-only token makes it acceptable. (owner:@pose-maintainers crit:medium review:2026-09-04)
- [open] F4: scope `contents: write` in `release.yml` to the publishing job instead of the workflow top level. (owner:@pose-maintainers crit:low review:2026-10-02)
- [open] F5: decide whether the SBOM is meant to carry license data — if so, configure syft to resolve licenses and give NOTICE a dependency section; if not, stop implying a license inventory in the docs. (owner:@pose-maintainers crit:medium review:2026-09-18)
- [open] R3: dispatch the Release workflow manually to complete the signing rehearsal, then record sign+verify in this spec. (owner:@pose-maintainers crit:high review:2026-08-21)
- [open] If any confirmation fails, decide whether the advertised guarantee should be softened in the docs until it is fixed — an unverified claim in README or docs-site is worse than an absent one. (owner:@pose-maintainers crit:medium review:2026-09-18)
