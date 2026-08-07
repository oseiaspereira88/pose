---
slug: pose-release-cycle-debt-closure
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-compat-gate-candidate-integrity, pose-installer-local-binary-precedence
priority: 1
components: release, installer
delivers: governance:release-cycle-debt
---

# Spec: Close the debts the 0.18 release cycle exposed

## 1. Intent

### Goal
Make the release cycle stop paying the same tolls: an upgrade-pair gate with
nothing to exercise, a `release prepare` that breaks the structural gate of
every spec it consumes, an installer download path no test covers, and follow-up
metadata that disappears on a line break.

### Business value
Cutting 0.18.0 through 0.18.2 took three tagged releases, two of which never
published. Each of these three items contributed: the compatibility gate could
not catch a real defect because its pair machinery had no entry to run, the
structural gate broke after every cut and cost several reindex-and-review cycles
to restore, and the installer's provider-download branch — the one every public
`curl | bash` user hits — is still only covered indirectly. A fourth item joined
them from the same cycle: follow-up metadata is parsed only on an item's first
line, so five wrapped items silently lost their owner and review date with no
gate objecting. None is urgent on its own; together they are what makes a
routine cut expensive.

### Constraints
- `pose release prepare` archives fragments as an immutable operation. Rewriting
  the claims of the specs it consumes must not mutate an already-cut snapshot.
- The download-branch test must not reach the real provider, so the gate stays
  runnable offline and in a fork without credentials.

### Non-goals
- Revisiting the support window itself. Starting at 0.18.2 was decided in
  `pose-compat-gate-candidate-integrity` and stands.
- Changing what `MergeManagedDoc` treats as engine-owned.

---

## 2. Requirements

### Functional
- R1: `supported_upgrades` shall carry the pair from 0.18.2 to the release being
  cut, with the checksums.txt SHA-256 pin, and the compatibility gate shall
  report that pair as exercised rather than as an empty window.
- R2: `pose release prepare` shall keep the artifact claims of every spec it
  consumes resolvable after the cut, so `pose check --strict` stays green
  without a manual reindex-and-review pass.
- R3: The installer's provider-download branch shall be covered by a test that
  serves the archive from a local origin, asserting both the success path and
  rejection of a malformed or truncated asset.
- R4: Follow-up metadata shall be recognised regardless of where it sits inside
  the item, or a malformed item shall fail `pose lint-spec --strict` naming the
  spec and the text. Today `(owner:… crit:… review:…)` is only parsed on the
  first line: a wrapped item silently loses its owner and review date, and no
  gate says so.

### Non-functional
- The added coverage must not make `tests/install/run.sh` depend on the network.

### Security
- The stub origin binds to loopback only and serves from a temporary directory;
  no fixture is fetched from outside the repository.

### Compatibility
- No public CLI, file or MCP contract changes. `supported_upgrades` gaining an
  entry is the matrix working as designed.

---

## 3. Technical Plan

### Affected areas
- `compatibility.json` and `tests/release/compat.sh` — first real pair (R1).
- `pose-mcp/internal/cli/release_lifecycle.go` — claim rewriting on prepare (R2).
- `install.sh` and `tests/install/run.sh` — download-branch coverage (R3).
- `pose-mcp/internal/cli/followups.go` — follow-up metadata parsing (R4).

### Artifacts
- modified: .pose/specs/pose-release-cycle-debt-closure/spec.md
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: pose-mcp/internal/cli/followups.go
- modified: pose-mcp/internal/cli/followups_owner_test.go
- modified: tests/install/run.sh
- modified: .pose/specs/pose-debt-marker-lexical-precision/spec.md
- modified: .pose/specs/pose-project-agnostic-assessment-engines/spec.md

### Delivery targets
- governance:release-cycle-debt module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- To be determined for R2: rewriting claims may need `prepare` to know which
  specs a fragment belongs to, which the fragment frontmatter already carries.

### Technical risks
- R2 is the delicate one: the claim rewrite must be part of the same atomic
  prepare, or a partial failure leaves specs pointing at a half-archived path.
- R4 has a migration edge: existing wrapped items across the repository would
  start parsing, which changes `pose followups` counts in one step. Preferable
  to a silent loss, but worth announcing.
- R1 cannot be validated until a release after 0.18.2 exists, so its check runs
  for the first time during that cut.

---

## 4. Tasks

### Planning
- [x] Confirm whether `prepare` can resolve consuming specs from fragment
      frontmatter alone — it does not need the index: the fragment filename is
      the claim path, so a substring match over the specs directory is enough
- [x] Decide where the download stub lives — inside the installer E2E, as a
      `curl` stub rather than a configurable origin in the shipped installer

### Implementation
- [ ] R1: add the 0.18.2 pair with its checksum pin at the next cut
- [x] R2: rewrite consumed specs' claims inside the prepare transaction
- [x] R3: serve the archive from a local origin and cover the bad-asset case
- [x] R4: parse follow-up metadata anywhere in the item

### Validation
- [x] Run the compatibility gate and confirm the pair is exercised, not skipped
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
R2 is the only one that can be proven before the next cut: a test that prepares
a release from a fixture spec and asserts the spec's claims still resolve
afterwards. R1 is proven by the next real cut — the gate must print the pair as
exercised. R3 is proven offline by the stub origin, including the negative case.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run "Release|Prepare|Followup" -count=1`
- Scope: prepare keeps consumed specs' claims resolvable; wrapped follow-up
  metadata is parsed or rejected
- Expected: ok

#### Security / Contract
- Command: `bash tests/release/compat.sh v<next>`
- Scope: compatibility gate with a populated matrix
- Expected: the 0.18.2 pair reported as PASS, not "none declared"

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.18.2-dev.
- Notes: R1 was attempted for real, not assumed impossible. Adding
  `from: 0.18.2` and running the gate exercised `check_upgrade_pair` end to end
  against the published v0.18.2 artifact — it reported
  `PASS: 0.18.2 → 0.18.2 (verified artifact; … upgrade → strict gate →
  idempotent reapply → preservation verified)`. The machinery works. The entry
  itself cannot stay: `TestCompatibilityMatrixContract` rejects listing the
  candidate as its own prior release, which is correct. So the pair is proven
  and the permanent entry belongs to the next cut.

### Results summary
- Successes: R2, R3, R4, each with a regression that fails without the fix
- Failures: none
- Deferred: R1 (blocked by version arithmetic, machinery proven)

### Requirement trace
- R1 [waived: the candidate cannot be its own prior release] — running the gate with the pair added proved `check_upgrade_pair` executes and passes against the real published artifact, so it is no longer dead code. `TestCompatibilityMatrixContract` then rejects the entry because `from` equals `engine_version`, correctly. The pin is recorded in the follow-up so the next cut is a one-line change
- R2 [satisfied] governance:release-cycle-debt evidence:integration check:delivery-integration test:TestReleasePrepareRepointsConsumedSpecArtifactClaims — `prepare` repoints any spec claim naming a fragment it archives, inside the same transaction and undone by the same rollback; the two claims already broken in this repository were repointed too
- R3 [satisfied] check:installer-e2e test:tests/install/run.sh — the provider-download branch now installs from a local origin served by a `curl` stub, and a truncated asset fails the install instead of leaving a broken binary on PATH
- R4 [satisfied] test:TestFollowupMetadataSurvivesLineWrapping — a wrapped follow-up keeps its owner, criticality and review date, and its continuation lines join the text; verified against the real repository by wrapping an existing item and watching `unowned` stay at 0

### Findings

**F1 — the repository already had two broken claims (severity: medium, fixed here).**
`pose-debt-marker-lexical-precision` and `pose-project-agnostic-assessment-engines`
both claimed `.pose/changelogs/unreleased/<slug>.md`, archived long ago by the
v0.16.2 and v0.16.3 cuts. Both were repointed to the paths that exist.

**F2 — R1 is blocked by arithmetic, not by engineering (severity: low).**
`supported_upgrades` lists releases *prior* to the candidate, so an entry can
only name 0.18.2 once the candidate is 0.18.3 or later. Attempting it now is
what proved the machinery runs, which was the actual worry behind R1.

### Known gaps
- R1's permanent entry lands at the next cut. Its pin is
  `3584471824dd27a1d48c204de099ba1c119d6966094bf80dc51ebe4cf44ba824`, already
  verified against the published `checksums.txt`.
- R3's stub covers the script's logic, not real network behaviour: a provider
  returning a redirect, a rate limit or a partial body is still unexercised.

---

## 7. Final Report

### Delivered scope
Three of four debts closed. `pose release prepare` no longer breaks the
structural gate of the specs it consumes. The installer's provider-download
branch — the one every public `curl | bash` user takes — has coverage for the
first time, including a corrupted asset. Follow-up metadata survives line
wrapping, which is what had silently unowned five items in this repository.
R1 is deferred to the next cut, with its machinery proven and its pin recorded.

### Files and modules changed
- pose-mcp/internal/cli/release_lifecycle.go and its test
- pose-mcp/internal/cli/followups.go, followups_owner_test.go
- tests/install/run.sh
- .pose/specs/pose-debt-marker-lexical-precision/spec.md
- .pose/specs/pose-project-agnostic-assessment-engines/spec.md

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1` and `bash tests/install/run.sh`
- Result: PASS

### Residual risks
- `repointFragmentClaims` rewrites by substring over every spec file. A spec
  mentioning another spec's fragment path in prose would be rewritten too. Both
  are `.pose/changelogs/...` paths, so the rewrite would still be correct, but
  it is broader than strictly needed.

### Follow-ups

- [open] R1: at the next cut, add `{"from": "0.18.2", "checksums_sha256": "3584471824dd27a1d48c204de099ba1c119d6966094bf80dc51ebe4cf44ba824"}` to supported_upgrades and confirm the gate reports the pair as exercised. The pin is already verified against the published checksums.txt. (owner:@pose-maintainers crit:high review:2026-09-04)
- [done] Reassess whether these items belonged together: R2 was the largest but not disproportionately so, and all four shared the release cycle as their cause, so keeping them in one spec held up. (owner:@pose-maintainers crit:low review:2026-10-02)
- [open] R3's stub proves the script's logic, not the network: a provider redirect, rate limit or partial body is still unexercised. Decide whether that is worth a real local HTTP origin. (owner:@pose-maintainers crit:low review:2026-11-20)
