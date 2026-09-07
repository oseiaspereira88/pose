---
slug: pose-release-security-gate-integrity
status: in-progress
created_at: 2026-09-06
completed_at:
supersedes:
depends_on:
priority: 0
components: ci, release
delivers: governance:release-security-gate-integrity
---

# Spec: Release security gate integrity

## 1. Intent

### Goal
Restore the project's automated gates to a state where they report reality:
a tag push publishes its full artifact set, and CI stops failing for a
governance defect that does not exist. Three independent defects, none of
which weakens a detection rule to fix.

### Business value
`https://github.com/oseiaspereira88/pose/releases/latest/download/install.sh`
is the install command published on the landing page and in the README. It
returns **HTTP 404**. The latest release, v1.7.10, carries zero assets; so do
v1.7.1 through v1.7.9. The last release with artifacts is **v1.6.0**.

Every public acquisition surface currently points at a dead download. No
amount of positioning work matters while the first command a visitor copies
fails, so this spec blocks the whole `community-launch` roadmap.

### Constraints
- No detection rule may be disabled or globally relaxed. A secret-scanning
  gate that is loosened to make a release pass stops being evidence.
- Exceptions must be owned, documented and as narrow as the finding.
- The historical git objects are not rewritten. History rewriting on a public
  repository breaks every existing clone, fork and release tag reference, and
  the finding is a false positive that does not justify that cost.

### Non-goals
- Changing what the release publishes, or the signing/SBOM/provenance chain.
- Re-releasing v1.7.1–v1.7.10 retroactively. Those tags stay as they are; the
  recovery path is a new release, covered by
  `pose-release-recovery-verification`.

---

## 2. Requirements

### Functional
- R1: The gitleaks gate shall report no findings on the full repository
  history, with `[extend] useDefault = true` preserved and no default rule
  removed or globally weakened.
- R2: The exception covering the `.harne8-agent-sync.json` false positive
  shall be conjunctive — scoped simultaneously to the `generic-api-key` rule,
  to that single file path, and to the literal `"<path>":"<64-hex>"` digest
  shape — so that a differently-shaped finding in the same file still fails
  the gate, and the digest shape alone is not exempt elsewhere.
- R3: `.harne8-agent-sync.json` shall be untracked and ignored, so no future
  commit can reintroduce the finding or leave the worktree dirty in CI.
- R4: `.github/action-runtimes.json` shall record the currently pinned ref for
  every referenced action, with each `using` value read from that action's own
  `action.yml` at the pinned ref rather than carried over from the prior pin.
- R5: `go test ./...` in `pose-mcp` shall pass, which is the release
  pipeline's first gate.
- R6: The `governance` CI job shall check out enough history for the delivery
  contract to attribute change sets, so `pose check --strict` reports the same
  result in CI as it does locally.

### Security
- The two exceptions are documented inline with the reason, the originating
  commit and the owning spec. Both narrow a match, neither disables a rule.

### Compatibility
- No change to release contents, tag naming, or the installer contract.

---

## 3. Technical Plan

### Affected areas
- `.gitleaks.toml`
- `.gitignore`
- `.github/action-runtimes.json`

### Artifacts
- modified: .gitleaks.toml
- modified: .gitignore
- modified: .github/action-runtimes.json
- modified: .github/workflows/ci.yml
- removed: .harne8-agent-sync.json

### Delivery targets
- governance:release-security-gate-integrity module:. profile:release-governance entrypoint:.github/workflows/release.yml

### API/contract changes
- None.

### Technical risks
- A conjunctive allowlist that is written too broadly would silently exempt a
  real credential. Mitigated by requiring all three conditions to hold at once
  and by asserting the gate still fails on a differently-shaped finding.

---

## 4. Tasks

### Planning
- [x] Reproduce the release failure and identify the failing step
- [x] Establish the causal chain from commit to failing gate

### Implementation
- [x] Increment 1: Add the rule-scoped conjunctive allowlist (R1, R2)
- [x] Increment 2: Untrack and ignore the agent-sync cache (R3)
- [x] Increment 3: Refresh the action runtime record from resolved `action.yml` (R4)

### Validation
- [x] Run the gitleaks gate exactly as CI invokes it
- [x] Run the full `pose-mcp` test suite
- [ ] Rehearse the pipeline end to end via `workflow_dispatch` snapshot
- [ ] Confirm a tagged release publishes its artifact set

---

## 5. Decisions

### Decision 1
- Date: 2026-09-06
- Context: `gitleaks` flagged one finding — rule `generic-api-key`, file
  `.harne8-agent-sync.json`, commit `acff2e5`. The matched value is a
  SHA-256 content digest in a `path → digest` manifest; the rule fires because
  the adjacent key ends in `auth.py`. It is a false positive.
- Options considered:
  1. Add `acff2e5` to the existing commit allowlist. Simple, and consistent
     with the existing `b6d4a72` precedent — but exempts an entire large
     migration commit, including anything else it touched.
  2. Add a global content regex. Would exempt the `"key":"<64-hex>"` shape
     across the whole repository, since the global allowlist in gitleaks
     v8.21.2 supports no `condition` field and its criteria are OR-ed.
  3. Extend the `generic-api-key` rule with a `[[rules.allowlists]]` entry
     using `condition = "and"`, combining a path scope and a content regex.
- Decision: option 3.
- Rationale: it is the only option that is narrow in all three dimensions at
  once — rule, path, and match shape. Options 1 and 2 each buy a passing gate
  by exempting more than the actual finding.
- Consequences: depends on `[[rules.allowlists]]` with `condition`, which is
  rule-scoped and present in gitleaks v8.21.2 (the pinned version). A
  downgrade below that version would silently drop the allowlist and fail the
  gate loudly, which is the safe direction.

### Decision 3
- Date: 2026-09-07
- Context: opening the pull request ran CI for the first time on this work, and
  the `governance` job failed `pose check --strict` on specs this branch never
  touched — `pose-cli-ergonomics-and-stack-expansion` and others — all with
  "no Git change sets are attributed".
- Investigation: CI on `main` has been failing continuously since at least
  2026-08-22 with the identical error, so it is pre-existing. `ci.yml` checks
  out with the `actions/checkout` default (`fetch-depth: 1`) while
  `release.yml` uses `fetch-depth: 0`. The delivery contract attributes
  declared artifacts to change sets found by the `POSE-Spec:` trailer, and the
  trailer for that spec sits 62 commits below `main`. Reproduced by cloning
  this repository at depth 1: **256 errors**, against 0 on a full clone.
- Decision: `fetch-depth: 0` on the governance job.
- Rationale: the gate was correct and its input was truncated. Any fix that
  relaxed the gate would have been treating a reporting defect as a governance
  defect.
- Consequences: the governance job clones full history, which costs seconds on
  a 669-commit repository. More importantly, CI can go green on `main` for the
  first time in weeks — and a red CI that everyone has learned to ignore is
  the same failure mode as the ten silently failed releases.

### Decision 2
- Date: 2026-09-06
- Context: `TestActionRuntimeCurrency` fails on `main` — four actions bumped by
  dependabot after v1.7.10 without refreshing their runtime record. This is a
  second, independent blocker that would fail the release pipeline one step
  *earlier* than the gitleaks finding.
- Decision: resolve `runs.using` from each action's own `action.yml` at the
  newly pinned ref, then record it — rather than copying the previous `using`
  value alongside the new SHA.
- Rationale: the record's entire value is that its runtime claim was verified
  at the pin. Carrying the old value forward preserves the format while
  destroying the guarantee. All four resolved to `node24`, unchanged — but
  that is now a checked fact rather than an assumption.
- Consequences: dependabot bumps will keep failing this test until the record
  is refreshed. That is the gate working as designed; see follow-ups.

---

## 6. Validation

### Strategy
Run each failing gate exactly as `release.yml` invokes it, and confirm the
transition from failing to passing is caused by these changes.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./... -count=1`
- Scope: the release pipeline's first gate
- Expected: exit 0

#### Security / Contract
- Command: `go -C pose-mcp run github.com/zricethezav/gitleaks/v8@v8.21.2 git --no-banner --redact --config ../.gitleaks.toml ..`
- Scope: full history, 669 commits
- Expected: `no leaks found`, exit 0

### Execution log
- Date: 2026-09-06
- Environment: local Linux, Go 1.26.5, gitleaks v8.21.2 pinned as in CI
- Notes: before the change, the gitleaks gate reported `leaks found: 1` and
  `TestActionRuntimeCurrency` reported four drifted refs. After the change,
  the gitleaks scan reports `no leaks found` (exit 0) and the full test suite
  exits 0.

### Results summary
- Successes: R1, R2, R3, R4, R5 verified locally.
- Failures: none.
- Warnings: the end-to-end publication path is not verifiable locally — it
  needs cosign, syft and a tag push. Deferred to
  `pose-release-recovery-verification`.

### Requirement trace
- R1 [satisfied] check:gitleaks-history — `no leaks found`, exit 0 over 669 commits
- R2 [satisfied] check:gitleaks-config — allowlist is rule-scoped with `condition = "and"` over path + match shape
- R3 [satisfied] check:git-ls-files — `.harne8-agent-sync.json` untracked and ignored
- R4 [satisfied] test:TestActionRuntimeCurrency — four refs refreshed, each `using` re-resolved from the pinned `action.yml`
- R5 [satisfied] check:go-test — `go -C pose-mcp test ./... -count=1` exits 0
- R6 [satisfied] check:shallow-clone-repro — a depth-1 clone produces 256 `pose check --strict` errors against 0 on a full clone; `fetch-depth: 0` added to the governance job

### Known gaps
- The steps after the security gate (cosign signing, goreleaser publish,
  `verify.sh`, package-manager manifests) have not executed since v1.6.0.
  They are unproven against the current tree, not known-broken. The snapshot
  rehearsal in `pose-release-recovery-verification` exists to surface any
  latent failure there before a real tag is pushed.

---

## 7. Final Report

### Delivered scope
Both blockers removed at the source. Publication itself is deliberately left
to the recovery spec, so that the fix and the proof that releases work again
are separate, independently verifiable steps.

### Files and modules changed
- `.gitleaks.toml`, `.gitignore`, `.github/action-runtimes.json`,
  and the untracking of `.harne8-agent-sync.json`.

### Residual risks
- A latent failure may exist in the never-exercised post-security steps.
- The gate that was supposed to protect the release contract did protect it —
  what failed is that ten consecutive failures produced no signal anyone
  acted on. That is an alerting gap, not a pipeline gap.

### Follow-ups

- [spawned: pose-release-recovery-verification] Prove end to end that a tagged
  release publishes its artifacts and installs on a clean machine.
- [open] CI has been red on `main` since at least 2026-08-22 and nobody acted
  on it, exactly as with the ten failed releases. Two independent automated
  signals were screaming and neither reached a human. Whatever notification
  gap causes that is the real defect; the two fixes here only remove today's
  noise.
- [open] The release workflow fails silently: ten consecutive failed releases
  produced no notification, and the drift was found only by an audit. A failed
  release on a tag should page the maintainer.
- [open] Dependabot bumps action SHAs without refreshing
  `.github/action-runtimes.json`, so every actions bump breaks `main` until
  fixed by hand. Either teach the bump to refresh the record, or gate the
  dependabot PR on the same test so it never merges red.
- [open] `.harne8-agent-sync.json` is still tracked in the parent `harne8`
  repository. The same false positive will surface there if it ever runs an
  equivalent history scan.
