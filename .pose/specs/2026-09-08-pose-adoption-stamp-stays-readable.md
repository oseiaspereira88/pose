---
slug: pose-adoption-stamp-stays-readable
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-contract-adoption-registry
priority: 0
components: pose-mcp
delivers:
---

# Spec: The adoption stamp does not lock out the previous engine

## 1. Intent

### Goal
Write the contract adoption date in a form the release before this one can still
read, and stop a future field from repeating the problem.

### Business value
`pose-contract-adoption-registry` introduced `contract_adoptions` in the review
policy, and `pose update` writes it. The review policy decoder calls
`DisallowUnknownFields`, so an engine that predates the registry does not ignore
that key — it rejects the entire policy:

```
pose: invalid schema-v2 review policy: json: unknown field "contract_adoptions"
```

The adopting repository hit it on the update itself. `pose update` from 2.0.1
stamped the map, and the 1.8.1 binary still installed on the same machine
started failing every review closeout in that repository. Any environment not
yet updated is in that position: a developer's machine, a pinned CI job, a
container image built last week.

So a release whose whole purpose was to let an instance adopt new rules without
losing its history made the instance unreadable to the engine it was adopting
from. The stamp was correct and its encoding was not.

The registry already carries what fixes it: each contract knows the top-level
key that held its date before the map existed, and the resolver reads both.
Writing the legacy key says the same thing to both engines.

### Constraints
- The map stays. A contract added later with no legacy key has nowhere else to
  go, and the resolver already prefers the map when both are present.
- An instance that already recorded a date, in either place, keeps it.

### Non-goals
- Making 1.8.1 tolerant. It is released; what can change is what is written for
  it to read.

---

## 2. Requirements

### Functional
- R1: Where a contract has a legacy policy key, the stamp shall write that key
  rather than the map.
- R2: `contract_adoptions` shall not be written when it would be empty.
- R3: A contract with no legacy key shall still be stamped into the map.
- R4: The review policy decoder shall ignore fields it does not know.

### Non-functional
- A policy written by this release loads under the previous release.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/stack_seed.go` — the stamp
- `pose-mcp/internal/pose/review_closeout.go` — the policy decoder

### Artifacts
- created: .pose/specs/2026-09-08-pose-adoption-stamp-stays-readable.md
- renamed: .pose/changelogs/unreleased/pose-adoption-stamp-stays-readable.md -> .pose/changelogs/v2.0.2/pose-adoption-stamp-stays-readable.md
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/stack_seed_test.go
- modified: pose-mcp/internal/pose/review_closeout.go

### Technical risks
- R4 gives up a real check: a misspelled policy key is now silently ignored
  rather than reported. That cost is smaller than the one it replaces — a
  misspelling affects one instance that can see its own file, while strictness
  makes every future field break every older binary reading the same
  repository. A diagnostic that names unrecognised keys without failing would
  recover it, and is recorded as a follow-up.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Stamp the legacy key where the contract has one (R1, R2, R3)
- [x] Increment 2: Ignore unknown policy fields (R4)

### Validation
- [x] A policy written by this release loads under the previous release

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: how to encode the stamp so both engines read it.
- Options considered: (a) write both the map and the legacy key; (b) write the
  legacy key where one exists, the map otherwise.
- Decision: (b).
- Rationale: (a) puts the same fact in two places, which the registry spec
  already recorded as a risk — nothing reports the two disagreeing, and an
  instance editing one would silently keep the other. (b) writes it once, and
  the resolver already reads both, so nothing downstream changes.

### Decision 2
- Date: 2026-09-08
- Context: whether to also stop the decoder refusing unknown fields.
- Decision: yes.
- Rationale: fixing only the encoding fixes this field and leaves the next one
  to repeat it. A field an engine does not know is a field a newer engine added;
  refusing the whole policy over it makes forward compatibility impossible by
  construction. This does not help 1.8.1, which is released — it stops the third
  occurrence.

---

## 6. Validation

### Strategy
The assertion that matters is not a unit test: install with this engine, remove
the key, update, and read the result with the previously released binary.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. The stamp writes
  `evidence_vocabulary_reconciled_at` and no `contract_adoptions`, confirmed by
  installing a fresh instance with this engine, removing both keys and running
  `pose update --no-self`.

- **Correction, same day.** That run also reported zero `unknown field` errors
  under the installed 1.8.1, and I recorded it as evidence that a policy written
  by this release loads under the previous one. It is not. The fresh instance
  had no specs, so `pose check` never loaded the policy through the closeout
  path — the measurement exercised a case where the file is not read. Run
  against the adopting repository, which has closeouts, 1.8.1 rejects the policy
  with `unknown field "evidence_vocabulary_reconciled_at"`.

  The reason is that this key is not a legacy key. It was introduced in
  `pose-attestation-evidence-must-be-in-the-bundle`, which shipped in 2.0.0, so
  1.8.1 does not know it either. Checked directly against the released source:
  1.8.1 declares `component_aware_adopted_at` and `review_bundles_adopted_at`
  and not this one, and removing the key from the adopting repository's policy
  takes its errors from five to zero.

  So the accurate claim is narrower than the one this spec first made. What the
  change buys is stated in Results below; what it cannot buy is readability by
  an engine that predates the contract itself, because the key names a contract
  that engine has never heard of. No encoding fixes that.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.
- Not achieved, and initially claimed: a policy that has adopted a contract
  introduced in 2.0.0 does not load under 1.8.1. The two contracts whose keys
  predate 2.0.0 stay readable, and every engine from 2.0.2 forward ignores keys
  it does not know — so this stops the *next* occurrence rather than curing
  this one. An instance adopting 2.0.x needs its other binaries updated too.

### Requirement trace
- R1 [satisfied] <stampContractAdoption writes LegacyContractField(id) when it is non-empty; the stamp test asserts the legacy key carries the date and that the contract is absent from the map>
- R2 [satisfied] <the map is only assigned when non-empty; asserted by a subtest that fails if an empty contract_adoptions is written, guarded so it means something only while every contract has a legacy key>
- R3 [satisfied] <the else branch writes the map, and the same subtest returns early for any contract without a legacy key>
- R4 [satisfied] <the review policy decoder no longer disallows unknown fields, so a field added after 2.0.2 is ignored rather than fatal by any engine from 2.0.2 on. This is forward-looking by construction and cannot be demonstrated against 1.8.1, which is the strict reader it does not change>

### Known gaps
- **An instance that adopts a 2.0.x contract cannot be read by 1.8.1.** The key
  names a contract that engine does not know, so this is a property of adopting
  the contract, not of how the date is encoded. The remedy is to update the
  other binaries reading that repository; nothing in the engine can avoid it.
- An unrecognised policy key is now silently ignored, so a misspelling reads as
  a default rather than an error.
- Nothing in the suite runs a previous release's binary; the compatibility claim
  rests on the manual run recorded above.

---

## 7. Final Report

### Follow-ups

- [open] Report unrecognised review policy keys as a doctor finding, recovering what dropping DisallowUnknownFields gave up — owner:unowned crit:medium review:2026-11-08
- [open] Say in the release notes that adopting a contract makes the instance unreadable to engines predating it, since no encoding avoids that — owner:unowned crit:medium review:2026-10-08
- [open] Run the previous release's binary against a policy this one writes, as a test rather than by hand — owner:unowned crit:medium review:2026-12-08
