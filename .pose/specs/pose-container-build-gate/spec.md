---
slug: pose-container-build-gate
status: done
created_at: 2026-08-09
completed_at: 2026-08-09
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
- `tests/release/container-build.sh` — the gate.
- `.github/workflows/ci.yml` — the step that runs it.

### Artifacts
- created: .pose/specs/pose-container-build-gate/spec.md
- created: tests/release/container-build.sh
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
- [x] Confirm both build contexts against the production compose file
- [x] Decide where the smoke test lives: inline steps or a script

### Implementation
- [x] R1: build both images with their correct contexts
- [x] R2: exercise both entrypoints
- [x] R3: attribute failures to a named image
- [x] R4: run on every push

### Validation
- [x] Prove the gate fails on a deliberately broken Dockerfile
- [x] Prove it fails on an image that builds but cannot start
- [x] Run the mandatory checks

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
- Date: 2026-08-09
- Environment: linux/amd64, Docker via /usr/local/bin/docker.
- Notes: both images build and start on the current tree. Each failure path was
  injected into the real Dockerfile rather than a fixture:
  1. **Does not build** — building `./cmd/does-not-exist` reports
     `FAIL: pose-mcp does not build from .` and re-runs the build without `-q`
     so the compiler error reaches the log.
  2. **Builds and does not start** — an entrypoint of `pose
     definitely-not-a-command` reports `FAIL: pose-mcp exited instead of
     starting`, while `mcp-enforce` still reports OK in the same run, which is
     R3's attribution working.
  3. The wrong-context error is real and confirmed: building
     `mcp-enforce/Dockerfile` from the repository root produces
     `"/go.mod": not found`.
- One fixture of mine was invalid and is worth recording: the first attempt at
  case 2 passed `--bogus-flag-that-aborts` to `serve-mcp`, which the binary
  accepts, so the server started and the gate correctly said nothing. The gate
  was right and the test was wrong.

### Results summary
- Successes: both images built and exercised; both failure paths demonstrated
  with attribution
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:container-build-gate evidence:integration check:delivery-integration test:tests/release/container-build.sh — `pose-mcp` builds from the repository root and `mcp-enforce` from its own directory, the contexts `docker-compose.prod.yaml` uses
- R2 [satisfied] check:container-build — each container is started and required to log `listening addr=`; the poll reports a crash-loop immediately instead of waiting out a fixed timeout
- R3 [satisfied] check:container-build — every failure names the image and which of build or start failed; in the injected start failure `mcp-enforce` still reported OK in the same run
- R4 [satisfied] report:.github/workflows/ci.yml — the step runs in the ordinary `ci` job on every push, not gated on Dockerfile changes

### Known gaps
- The build contexts are written down independently here and in the monorepo's
  compose file, and nothing keeps them in agreement — a context change on either
  side is invisible to the other.
- The gate proves each image starts alone. Whether the two talk to each other,
  and whether `pose-mcp` serves a mounted project correctly, remains the
  composition's integration concern.
- Startup is asserted by a log line. A container that logs `listening` and then
  fails to serve would pass.

---

## 7. Final Report

### Delivered scope
Both delivery images are built in CI with the contexts production uses, and each
entrypoint is started and required to announce itself. A failure names the image
and whether it was the build or the start.

### Files and modules changed
- tests/release/container-build.sh
- .github/workflows/ci.yml

### Validation executed
- Command: `bash tests/release/container-build.sh`, plus two injected failures
- Result: pass on the current tree; both failure paths named their cause

### Residual risks
- The contexts are duplicated across two repositories with nothing reconciling
  them, which is the follow-up this spec opens.

### Follow-ups

- [open] The images are built here and composed in `harne8`, with the build contexts written down independently in both places and nothing keeping them in agreement. A context change in the compose file would not fail anything in this repository, and vice versa. Consider whether the composition should consume a contract this repository publishes — the same problem `pose-machinery-distribution-contract` solved for the scaffold — rather than each side describing the other from memory. (owner:@pose-maintainers crit:medium review:2026-10-09)
