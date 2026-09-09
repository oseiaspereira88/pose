---
slug: pose-release-boundary-rehearsal
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-self-update-handoff-path, pose-adoption-stamp-stays-readable
priority: 0
components: pose-mcp
delivers:
---

# Spec: Rehearse the two paths only a release has ever run

## 1. Intent

### Goal
Run, in tests, the two sequences whose first execution has always been a real
release: downloading a new binary and handing off to it, and reading a policy
this engine writes with the binary the previous release shipped.

### Business value
Both were left as follow-ups by the specs that found the corresponding defects,
and both defects have the same shape: an engine cannot observe the boundary it
sits on. It downloads its successor but never runs one; it writes a policy but
only ever reads it back with itself.

That blindness has cost two releases already. v2.0.0's handoff called
`os.Executable()` after renaming the running binary out of the way, and failed
with `fork/exec .../pose.old: no such file or directory` — on the release run,
because nothing else had ever reached that line. v2.0.0 also added
`contract_adoptions` to the review policy, which v1.8.1 refused outright: its
loader used `DisallowUnknownFields`, so one added key invalidated the file.

The second one also produced a false claim. It was reported as fixed in v2.0.2
on the strength of a run in a fresh instance that had no specs, where the policy
loader is never reached — the claim was retracted. A test is worth more than the
by-hand check precisely because the by-hand check was wrong.

### Constraints
- Neither test may need the network, or it is the same gap in a new place.
- The update source must not become runtime-configurable: a binary that can be
  told where to download its own replacement from is a supply-chain surface.

### Non-goals
- Testing every past release. The contract is with the release immediately
  below this one, which is what an instance being updated is running.
- Verifying published artifacts. `verify-release.yml` owns that.

---

## 2. Requirements

### Functional
- R1: A test shall run `pose update` end to end against a local release server:
  fetch the release metadata, download the archive, replace the binary on disk,
  and hand off to the binary that was written.
- R2: The release endpoints shall be overridable at build time only, so the test
  can point a binary at a local server without adding a runtime override.
- R3: A test shall build the previous release from its own tag and require it to
  load a review policy this engine writes.
- R4: R3 shall prove it reached the previous release's policy loader, rather
  than passing because nothing read the file.

### Non-functional
- Both tests run offline, given a warm module cache.
- Both skip, rather than fail, when their precondition is genuinely absent — a
  shallow clone with no tag objects.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/maintenance.go` — the release endpoints
- `pose-mcp/internal/cli/self_update_release_test.go` — R1
- `pose-mcp/internal/cli/release_compatibility_test.go` — R3, R4

### Artifacts
- created: .pose/specs/2026-09-09-pose-release-boundary-rehearsal.md
- created: pose-mcp/internal/cli/self_update_release_test.go
- created: pose-mcp/internal/cli/release_compatibility_test.go
- created: .pose/changelogs/unreleased/pose-release-boundary-rehearsal.md
- modified: pose-mcp/internal/cli/maintenance.go
- modified: .pose/specs/2026-09-08-pose-self-update-handoff-path.md
- modified: .pose/specs/2026-09-08-pose-adoption-stamp-stays-readable.md

### Technical risks
- Building the previous release from source costs a `go build` per run, and
  fails if that tag no longer builds under the current toolchain. That failure
  is worth seeing: a previous release that cannot be built is a previous release
  nobody can check against.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Build-time seam for the release endpoints (R2)
- [x] Increment 2: End-to-end download, replace and handoff (R1)
- [x] Increment 3: Previous release against this engine's policy (R3, R4)

### Validation
- [x] Each test shown to fail against the defect it covers

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: the download URL is built from a `const releaseRepo`, so no test can
  redirect it.
- Decision: make the two base URLs package variables, documented as settable
  only through `-ldflags -X`, and not read from the environment.
- Rationale: an environment variable would let anything in a user's shell decide
  where `pose update` fetches a replacement binary from. A build-time override
  reaches the test and nothing else.

### Decision 2
- Date: 2026-09-09
- Context: what to compare the previous release against — a recorded list of the
  keys it understands, or the binary itself.
- Decision: build the binary from its tag.
- Rationale: a recorded list is a second source of truth, and its only failure
  mode is being stale on precisely the release that needed it. `git archive`
  reads the tag without touching the working checkout.

### Decision 3
- Date: 2026-09-09
- Context: which command to probe the previous release with.
- Decision: `review-plan` against a scope that does not exist.
- Rationale: `check --strict` reads the policy through a four-field struct of
  its own and never reaches the loader that enforces the schema — against
  v1.8.1 it reports SUCCESS on a policy carrying an outright unknown key. That
  is the same wrong probe that produced the retracted v2.0.2 claim.
  `review-plan` loads the policy before resolving the scope, so it reaches the
  loader on an instance with no specs and then fails on the missing scope, with
  a message distinct from a policy failure.

---

## 6. Validation

### Strategy
Each test is required to fail against the defect it exists for, not only to
pass against the current code.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-09
- Environment: local, Go 1.26, offline module cache
- Notes: reintroducing `os.Executable()` in the handoff makes the new
  end-to-end test fail with `fork/exec .../install/pose.old: no such file or
  directory` — the v2.0.0 release failure, reproduced locally for the first
  time. Pointing the compatibility test at v1.8.1 instead of the resolved
  previous tag fails with `invalid schema-v2 review policy: json: unknown field
  "evidence_vocabulary_reconciled_at"`, which is the regression it exists to
  catch; against v2.0.2, the actual previous release, it passes.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestSelfUpdateDownloadsReplacesAndHandsOff serves release metadata and a tar.gz from an httptest server, runs a pose binary built at an older version against it, and asserts the fixture's own output carries `update --no-self`, that the binary on disk equals the downloaded one, and that no `.old` backup survives>
- R2 [satisfied] <releaseAPIBase and releaseDownloadBase are package variables set by the test's `go build -ldflags -X`; nothing reads them from the environment>
- R3 [satisfied] <TestPreviousReleaseReadsThisEnginesReviewPolicy resolves the highest tag below this version, extracts it with `git archive`, builds `./cmd/pose` from it, and runs it against an instance installed in-process by this engine>
- R4 [satisfied] <the same test first writes a policy whose `enabled` is a string and requires the previous release to reject it through the same command; a probe that never reaches the loader fails this control, as `check --strict` does>

### Known gaps
- The compatibility test compares against one release. An instance two or more
  releases behind is not covered, and the engine makes no promise about that.

---

## 7. Final Report

### Summary
The two release-boundary sequences now run in tests, and both were shown to fail
against the defects that motivated them.

### Follow-ups

- [open] Consider asserting the archive layout against what `.goreleaser.yaml` actually produces, rather than against the name `extractPoseBinary` expects — the fixture and the extractor agree by construction here — owner:unowned crit:low review:2026-12-09
