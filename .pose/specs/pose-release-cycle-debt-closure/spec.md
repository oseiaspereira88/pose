---
slug: pose-release-cycle-debt-closure
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on: pose-compat-gate-candidate-integrity, pose-installer-local-binary-precedence
priority: 1
components: release, installer
delivers:
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
<!-- Declare exact paths at closeout, once the change set is known. -->
- created: .pose/specs/pose-release-cycle-debt-closure/spec.md

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
- [ ] Confirm whether `prepare` can resolve consuming specs from fragment
      frontmatter alone, or needs the spec index
- [ ] Decide where the download stub lives so both the installer E2E and the
      compatibility gate can reuse it

### Implementation
- [ ] R1: add the 0.18.2 pair with its checksum pin at the next cut
- [ ] R2: rewrite consumed specs' claims inside the prepare transaction
- [ ] R3: serve the archive from a loopback origin and cover the bad-asset case
- [ ] R4: parse follow-up metadata anywhere in the item, or fail the lint loudly

### Validation
- [ ] Run the compatibility gate and confirm the pair is exercised, not skipped
- [ ] Cut a release and confirm `pose check --strict` needs no manual repair
- [ ] Run the mandatory checks

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

### Requirement trace
<!-- Filled at closeout. -->

### Known gaps
- R1 cannot be validated in isolation: its first real execution is the next
  release cut, which is also when a regression in it would first bite.

---

## 7. Final Report

<!-- Filled at closeout. -->

### Follow-ups

- [open] Reassess whether these items belonged together once they are done: if R2 turns out to be substantially larger than R1, R3 and R4, split it out rather than letting this spec stay open across several cycles. (owner:@pose-maintainers crit:low review:2026-10-02)
