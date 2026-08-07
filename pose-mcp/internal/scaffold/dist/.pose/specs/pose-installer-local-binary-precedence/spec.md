---
slug: pose-installer-local-binary-precedence
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-public-install-contract
priority: 1
components: installer
delivers:
---

# Spec: Installer prefers the binary beside it and invokes the CLI it has

## 1. Intent

### Goal
Make `install.sh` install from the native binary shipped beside it before
reaching the network, and call `pose upgrade`/`pose check` with the arguments
those commands actually accept.

### Business value
The release workflow cannot publish: its installer E2E installs the *previously
published* release into the target and then runs `check --strict` with the *new*
binary. Since `pose-command-reference-parity` added the manual-parity gate, the
old POSE.md fails the new gate, so every release blocks on the release before
it. The v0.18.0 cut died on exactly this (run 31138240912) and the tag is
recorded `failed`.

### Constraints
- The `curl | bash` one-liner must keep working unchanged when no binary sits
  beside the script.
- A release bundle install must not depend on the network, on a source tree, or
  on Go being present.

### Non-goals
- Redesigning the download flow, checksum policy or `~/.local/bin` placement.
- Changing the manual-parity gate, which is behaving correctly.

---

## 2. Requirements

### Functional
- R1: When an executable `pose` sits beside `install.sh`, the installer shall
  use it and shall not query or download from the release provider.
- R2: When no binary sits beside `install.sh`, the installer shall keep its
  current provider-download behaviour unchanged.
- R3: The installer shall invoke `pose upgrade` and `pose check --strict` in a
  form those commands accept, and shall surface a failing final gate instead of
  discarding it.

### Non-functional
- The fix stays inside `install.sh` and its E2E; no Go source changes.

### Security
- Preferring a local binary must not widen the trust boundary: only an
  executable file literally beside the script qualifies, never `PATH` lookup or
  a caller-supplied path.

### Compatibility
- `pose install`, `pose upgrade` and `pose check` keep their current public
  signatures; only the caller is corrected.

---

## 3. Technical Plan

### Affected areas
- `install.sh` — bundle-first binary resolution and correct CLI invocation.
- `tests/install/run.sh` — regression proving the bundle path is hermetic.

### Artifacts
- modified: install.sh
- modified: tests/install/run.sh

### API/contract changes
- None. The installer is the only caller corrected.

### Technical risks
- A repository that legitimately keeps an unrelated `pose` file beside a copied
  `install.sh` would now be preferred over the download. Mitigated by requiring
  the file to be executable, which a stray artifact is not.

---

## 4. Tasks

### Planning
- [x] Reproduce the failing E2E locally and isolate the root cause
- [x] Confirm `upgrade` and `check` reject the positional directory

### Implementation
- [x] Prefer an executable `pose` beside the script before any network call
- [x] Run `upgrade` and `check --strict` from inside the target directory
- [x] Propagate a failing final gate instead of `|| true`

### Validation
- [x] Run the installer E2E with a network-free `PATH`
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Drive the real script end to end: build a release-shaped bundle (binary plus
`install.sh`), install from it with a `PATH` whose first entry is a `curl` stub
that fails loudly, and require `check --strict` to pass afterwards with the
freshly built binary. Poisoning `curl` rather than removing it keeps the rest of
the environment intact while making any provider access an outright failure.
That is exactly the composition the release workflow performs, so a green E2E
means the release workflow can publish.

### Deterministic checks

#### Test
- Command: `bash tests/install/run.sh`
- Scope: installer end to end, including the network-free bundle scenario
- Expected: `native installer scenarios: PASS`; with `install.sh` reverted the
  same command fails with `installer reached the network`

#### Security / Contract
- Command: `pose check --strict`
- Scope: POSE structural gate, including manual parity
- Expected: SUCCESS

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.18.0-dev.
- Notes: before the fix, the bundle scenario installed the published v0.17.0
  POSE.md (367 lines) and `check --strict` reported 25 commands missing from the
  manual; after it, the bundle installs the built binary's own manual.

### Results summary
- Successes: installer E2E, `pose check --strict`, `go test ./...`, `go vet ./...`
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] check:installer-e2e-offline-bundle test:tests/install/run.sh — the bundle scenario installs with a failing `curl` stub first on PATH; reverting install.sh makes it fail with "installer reached the network"
- R2 [satisfied] check:installer-e2e test:tests/install/run.sh — the download branch is untouched and still covered by the pre-existing scenarios
- R3 [satisfied] check:installer-e2e test:tests/install/run.sh — the final gate now fails the script instead of printing a usage banner

### Known gaps
- The download branch itself remains unexercised offline: no scenario asserts
  what happens when the provider is reachable but returns a bad asset. That gap
  predates this fix and belongs to `pose-public-install-contract`.

---

## 7. Final Report

### Delivered scope
`install.sh` resolves an executable `pose` beside itself before contacting the
provider, and runs `upgrade`/`check --strict` from inside the target with the
arguments those commands accept, failing the script when the gate fails. The
E2E gained a network-free assertion for the bundle path.

### Files and modules changed
- install.sh
- tests/install/run.sh

### Validation executed
- Command: `bash tests/install/run.sh`
- Result: PASS

### Residual risks
- The one-liner download path is still only covered indirectly; a provider-side
  regression would not be caught by this E2E.

### Follow-ups

- [open] Cover the provider-download branch of `install.sh` with a stubbed HTTP
  origin so a malformed or truncated asset is a test failure rather than a
  production discovery.
