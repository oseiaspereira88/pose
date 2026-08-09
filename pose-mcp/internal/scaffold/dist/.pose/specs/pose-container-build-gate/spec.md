---
slug: pose-container-build-gate
status: draft
created_at: 2026-08-09
completed_at:
supersedes:
depends_on: pose-dependency-pin-refresh
priority: 1
components: pose-mcp
delivers: governance:container-build-gate
---

# Spec: The images this repository defines are built by this repository

## 1. Intent

### Goal
Build both container images in CI and exercise the binaries they produce, so a
broken Dockerfile fails here instead of when the composition stack comes up.

### Business value
`pose-mcp/Dockerfile` and `mcp-enforce/Dockerfile` are delivery artifacts, not
samples. The `harne8` monorepo composes both in production from this very
repository:

```
mcp-enforce-sidecar:
  build: {context: ./pose-dist/mcp-enforce, dockerfile: Dockerfile}
pose-mcp:
  # build context is the submodule root so `replace => ../mcp-enforce` resolves
```

Nothing in this repository builds either one. Not CI, not the release workflow,
not a gate — verified by grepping every workflow, `tests/`, `scripts/` and
`.goreleaser.yaml`. A Dockerfile defect is therefore discovered when someone
brings up the production stack, in a different repository, at the least
convenient moment.

That is not hypothetical. Dependabot PR #10 bumped `alpine` from 3.21 to **3.24**
— three minor versions — and arrived with every check green: `test`, `codeql`,
`governance`, `govulncheck`, `secrets`, `dependency-review`,
`workflow-contract`. Every one of them runs Go tooling and scanners; none
touches an image. The bump was merged only because it was built by hand first.
The next one will not be.

This is the same shape as the defects this project has been closing all cycle: a
delivery target whose reachability is asserted rather than demonstrated. POSE
has a name for it — `surface-check` exists precisely because "it builds" and "it
is composed" are different claims — and the container images are the one
delivery surface currently outside that discipline.

### Constraints
- The two build contexts differ and are not obvious. `pose-mcp` must be built
  from the repository root, because it consumes the sibling `mcp-enforce`
  module through `replace => ../mcp-enforce`; `mcp-enforce` must be built from
  its own directory, because it copies `go.mod` from the context root. Getting
  this wrong produces `"/go.mod": not found`, which reads like a broken
  Dockerfile rather than a wrong invocation — it happened on the first attempt
  while validating PR #10.
- The gate must exercise the produced binary, not only the build. An image that
  builds and cannot start is still broken, and the entrypoints differ: one
  serves HTTP, the other is a reverse proxy.

### Non-goals
- Publishing images to a registry. Nothing publishes them today; the composition
  builds from source. Adding a registry is a distribution decision with its own
  provenance and signing consequences.
- Reproducing the composition. Whether the two containers talk to each other is
  the monorepo's integration concern, not this repository's build gate.

---

## 2. Requirements

### Functional
- R1: CI shall build `pose-mcp/Dockerfile` from the repository root and
  `mcp-enforce/Dockerfile` from its own directory, matching the contexts the
  production compose file uses.
- R2: It shall exercise each produced image: the `pose-mcp` entrypoint must
  start and report its listening address, and the sidecar entrypoint must start.
- R3: A build or startup failure shall fail the job, naming which image and
  which of the two it was.
- R4: The gate shall run on every push, not only when a Dockerfile changes — a
  Go change can break the build without touching the Dockerfile.

### Non-functional
- Both images are small static Go builds on Alpine; the two builds together
  should stay within a couple of minutes with layer caching.

### Security
- Build only, no push and no registry credentials.
- The images are already digest-pinned by `pose-dependency-digest-pinning`, so
  the gate builds from immutable bases.

### Compatibility
- No change to either Dockerfile's contents or to how the monorepo composes
  them. The gate observes; it does not restructure.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/ci.yml` — the build-and-smoke step.
- possibly `tests/release/` — a script, if the smoke test is more than two
  `docker run` invocations.

### Artifacts
- created: .pose/specs/pose-container-build-gate/spec.md
- modified: .github/workflows/ci.yml

### Delivery targets
- governance:container-build-gate module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **The build contexts are duplicated knowledge.** The compose file in the
  monorepo and the gate here must agree on how each image is built, and nothing
  connects them; a change to one is invisible to the other. Encoding the
  contexts in a script with the reason written down narrows this, but the
  authoritative copy stays in another repository.
- The smoke test for `pose-mcp` starts a server, so it needs a timeout and a
  deliberate shutdown rather than a plain `docker run` that would hang the job.
  A naive implementation turns a fast gate into a five-minute one.
- Building on every push adds time to the most frequently run workflow. The
  alternative — building only when Dockerfiles change — would have missed
  exactly the class R4 exists for, so the cost is accepted rather than
  optimised away.

---

## 4. Tasks

### Planning
- [ ] Confirm both build contexts against the production compose file
- [ ] Decide where the smoke test lives: inline steps or a script

### Implementation
- [ ] R1: build both images with their correct contexts
- [ ] R2: exercise both entrypoints
- [ ] R3: attribute failures to a named image
- [ ] R4: run on every push

### Validation
- [ ] Prove the gate fails on a deliberately broken Dockerfile
- [ ] Prove it fails on an image that builds but cannot start
- [ ] Run the mandatory checks

---

## 6. Validation

### Strategy
A build gate is worth what its failure path demonstrates, and there are two
distinct failures to separate: an image that does not build, and an image that
builds and does not run. The second is the one a naive gate misses, and it is
the one that would reach production — a compose stack that pulls a green image
and then crash-loops.

Both are provable locally with the same tooling CI uses, so neither is deferred
to a real run.

### Deterministic checks

#### Build
- Command: `docker build -f pose-mcp/Dockerfile .` and `docker build -f mcp-enforce/Dockerfile mcp-enforce/`
- Scope: both images, with the contexts production uses
- Expected: both build

#### Test
- Command: the smoke invocations for both entrypoints
- Scope: the produced binaries start
- Expected: `pose-mcp` reports its listening address; the sidecar starts

### Execution log
<!-- Filled at implementation. The prior manual validation is evidence that the
     images build today, not that the gate works. -->

### Results summary
<!-- Filled at implementation. -->

### Requirement trace
<!-- Filled at implementation. -->

### Known gaps
<!-- Filled at implementation. Expected to record that the build contexts are
     duplicated between this gate and the monorepo's compose file, with no
     mechanism keeping them in agreement. -->

---

## 7. Final Report

### Delivered scope
<!-- Filled at closeout. -->

### Files and modules changed
<!-- Filled at closeout. -->

### Validation executed
<!-- Filled at closeout. -->

### Residual risks
<!-- Filled at closeout. -->

### Follow-ups

- [open] The images are built here and composed in `harne8`, with the build contexts written down independently in both places and nothing keeping them in agreement. A context change in the compose file would not fail anything in this repository, and vice versa. Consider whether the composition should consume a contract this repository publishes — the same problem `pose-machinery-distribution-contract` solved for the scaffold — rather than each side describing the other from memory. (owner:@pose-maintainers crit:medium review:2026-10-09)
