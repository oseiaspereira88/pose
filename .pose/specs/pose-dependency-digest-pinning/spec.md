---
slug: pose-dependency-digest-pinning
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-actions-node24-bump
priority: 1
components: release
delivers: governance:dependency-digest-pinning
---

# Spec: Every dependency is pinned to something that cannot move

## 1. Intent

### Goal
Pin every GitHub Action, container base image and docs-build Python package to
an immutable digest or hash, so no build input can change under a reference
this repository already trusts.

### Business value
Pinned-Dependencies was the largest open block in the OpenSSF baseline: 45
alerts on the run of 2026-08-07T21:29Z, across all eight workflows and both
Dockerfiles. The first-party actions were tag-pinned under an owned exception,
and the container bases carried no digest at all.

The exception's reasoning was that `actions/` and `github/` are maintained by
the platform owner and receive security fixes through the tracked tag. That is
true and beside the point: a mutable tag means the code executed by a release
build is whatever that tag pointed at that morning. The release pipeline signs
artifacts and publishes provenance about how they were built; a moving tag
makes that provenance describe a build the repository cannot reproduce.

The decision was scheduled for "before the repository goes public" and the
repository is about to go public.

Verified rather than assumed: pinning surfaced a defect the tag form was
hiding. `actions/dependency-review-action@v5`, set by the Node 24 bump earlier
today, referenced a tag that does not exist — that action publishes no major
tags above v3. The workflow never failed because its job is gated on
`github.event_name == 'pull_request'` and every run since had been a push. A
digest cannot be nonexistent, so the class of defect goes away with the form.

### Constraints
- Every digest must be resolved from the registry or the Git ref API at pin
  time and recorded with the human-readable version beside it. A digest with no
  version comment is unreviewable.
- The container digests are multi-arch index digests, not per-platform image
  digests, so `docker build` keeps resolving the right architecture.

### Non-goals
- Automating the refresh. A digest pin trades silent drift for explicit
  staleness, and the refresh mechanism (Dependabot or equivalent) is its own
  decision with its own review burden.
- Pinning Go module dependencies, which `go.sum` already covers by digest.

---

## 2. Requirements

### Functional
- R1: Every `uses:` reference in `.github/workflows/` shall be pinned to a full
  40-character commit SHA, with the version it corresponds to in a trailing
  comment.
- R2: Every `FROM` in `pose-mcp/Dockerfile` and `mcp-enforce/Dockerfile` shall
  carry an image digest alongside the human-readable tag.
- R3: The workflow security contract shall accept a digest pin from any owner
  without requiring an exception, and shall keep requiring the exception for a
  first-party tag pin.
- R4: The docs build shall install its Python dependencies from a lock file in
  which every package, direct or transitive, is pinned by hash, and pip shall
  refuse anything that is not.

### Non-functional
- Readability is preserved by the version comment; a reviewer never has to
  resolve a SHA to know what is pinned.

### Security
- This is the whole point: an immutable reference cannot be repointed at
  attacker-controlled code by whoever can move a tag.
- The `first-party-actions-tag-pinning` exception is now unused. It is left in
  place until its 2026-10-19 expiry rather than deleted, so that an action
  added back by tag is still caught by the existing guard instead of silently
  accepted.

### Compatibility
- No behaviour change. Each digest is the exact commit the previously
  referenced tag resolved to at pin time, except `dependency-review-action`,
  where the previous reference did not resolve at all and the digest is v5.0.0.

---

## 3. Technical Plan

### Affected areas
- All eight workflows under `.github/workflows/`.
- `pose-mcp/Dockerfile`, `mcp-enforce/Dockerfile`.
- `pose-mcp/internal/version/workflow_security_test.go` — the contract.
- `docs-site/requirements.in` / `requirements.txt` — the docs build's lock.

### Artifacts
- created: .pose/specs/pose-dependency-digest-pinning/spec.md
- modified: .github/workflows/ci.yml
- modified: .github/workflows/security.yml
- modified: .github/workflows/scorecard.yml
- modified: .github/workflows/release.yml
- modified: .github/workflows/verify-release.yml
- modified: .github/workflows/package-channels.yml
- modified: .github/workflows/governance-audit.yml
- modified: .github/workflows/docs.yml
- modified: pose-mcp/Dockerfile
- modified: mcp-enforce/Dockerfile
- modified: pose-mcp/internal/version/workflow_security_test.go
- created: docs-site/requirements.in
- created: docs-site/requirements.txt

### Delivery targets
- governance:dependency-digest-pinning module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None externally. The workflow contract's internal rule changes as R3
  describes.

### Technical risks
- **Pins go stale silently.** This is the cost of the trade, not a defect: a
  digest never moves, so a security fix published upstream does not arrive
  until someone refreshes. Nothing in this change notices that, and the
  follow-up says so rather than implying the problem is handled.
- The contract had a latent defect that this work exposed: it errored on any
  first-party action whenever the exception was missing, including one pinned
  by SHA, because the branch never checked whether the pin was actually a tag.
  It would have blocked exactly this change. Fixed as part of R3.

---

## 4. Tasks

### Planning
- [x] Resolve every action's tag to its commit and confirm the semantic version
- [x] Resolve both container bases to their index digests

### Implementation
- [x] R1: digest-pin every action with a version comment
- [x] R2: digest-pin both Dockerfiles' bases
- [x] R3: accept digest pins from any owner in the contract
- [x] R4: lock the docs build's Python closure by hash

### Validation
- [x] Workflow security contract
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Two properties, checked where each is actually decidable. That the pins are
well-formed is a repository property and belongs in the contract test. That
each digest is the *right* one is a claim about the outside world, settled at
pin time by resolving the tag through the API and recording the version beside
the SHA — and re-settled by the workflows continuing to run, since a wrong
digest fails immediately and loudly.

### Deterministic checks

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/version/... -count=1`
- Scope: pinning form and permissions across every workflow
- Expected: pass

#### Build
- Command: `pip install --require-hashes -r docs-site/requirements.txt` then `mkdocs build --strict -f docs-site/mkdocs.yml`
- Scope: the docs build's locked Python closure
- Expected: install and build both succeed

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64. Action digests resolved via the GitHub Git ref API,
  dereferencing annotated tags to their commits; container digests read from
  the registry manifest's `Docker-Content-Digest` for the multi-arch index.
- Notes: eleven distinct action references, two base images and 29 Python
  packages pinned. `actions/dependency-review-action@v5` did not resolve — no
  major tag above v3 exists — and is pinned to v5.0.0's commit. The Python lock
  was installed into a clean 3.12 environment under `--require-hashes` and the
  strict docs build ran against it before the workflow was changed.

### Results summary
- Successes: every action, base image and docs Python package pinned; contract
  defect fixed
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:dependency-digest-pinning evidence:integration check:delivery-integration report:.github/workflows/ci.yml — every `uses:` across the eight workflows carries a 40-character SHA and a version comment
- R2 [satisfied] check:workflow-security-contract report:pose-mcp/Dockerfile — both Dockerfiles pin `golang:1.26.4-alpine` and `alpine:3.21` by index digest
- R3 [satisfied] test:pose-mcp/internal/version/workflow_security_test.go — a SHA pin short-circuits before the ownership branch, so no exception is consulted; a first-party tag pin still requires the live exception
- R4 [satisfied] check:docs-build report:.github/workflows/docs.yml — `docs-site/requirements.txt` locks all 29 packages of the mkdocs-material closure by hash and the step runs `pip install --require-hashes`; proven locally by installing into a clean Python 3.12 environment under that flag and running `mkdocs build --strict`, which succeeded

### Known gaps
- Nothing refreshes a stale digest. The pins are correct as of 2026-08-07 and
  will drift from upstream security fixes until someone updates them.
- The version comments are unverified prose: nothing checks that the comment
  beside a SHA names the version that SHA actually is.

---

## 7. Final Report

### Delivered scope
Every action across the eight workflows, both container base images and the
docs build's full Python closure are pinned to immutable digests or hashes.
The workflow security contract accepts a digest pin from any owner and keeps
the exception guard for tag pins.

### Files and modules changed
- .github/workflows/*.yml (eight files)
- pose-mcp/Dockerfile, mcp-enforce/Dockerfile
- pose-mcp/internal/version/workflow_security_test.go
- docs-site/requirements.in, docs-site/requirements.txt

### Validation executed
- Command: `go -C pose-mcp test ./internal/version/... -count=1`
- Result: pass

### Residual risks
- Staleness is now the failure mode, and it is silent. That is a deliberate
  trade of a loud, immediate risk for a quiet, slow one, but it is a trade.

### Follow-ups

- [open] Digest pins do not refresh themselves: upstream security fixes now arrive only when someone updates a SHA by hand, and nothing reports that a pin has fallen behind. Decide between Dependabot on the actions and Docker ecosystems, or a check that compares each pin against its tag's current target and warns. This is the cost this spec deliberately accepted and it should not stay unmanaged. (owner:@pose-maintainers crit:medium review:2026-10-02)
- [open] Nothing verifies that the version comment beside a SHA names the version that SHA actually is, so a wrong comment misleads every future reviewer while the pin itself stays valid. The same resolution the `pose-action-runtime-currency-gate` draft needs would settle this too — consider folding the two checks together rather than building both. (owner:@pose-maintainers crit:low review:2026-11-06)
