---
slug: review-verify-retains-completed-scopes
status: in-progress
created_at: 2026-09-29
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
delivers: capability:completed-review-retention
---

# Spec: `review verify` keeps the approval a closed scope was closed with

## 1. Intent

### Goal

A scope closed with an approved sealed review stays approved under `pose review
verify` and under federated acceptance, as it already does under `pose
review-check`, until a newer attestation says otherwise.

### Business value

Moving Harne8's pin to the 6.0.1 revision failed its adoption gates: the
preflight reported `upstream roadmap review is not fresh and approved`, and every
spec depending on pose-dist through `xref:` reported
`source-review-not-approved`. Nothing in the reviewed work had changed. Under
`review verify`, 16 of the 17 closed pose-dist specs Harne8 consumes were
`superseded`, and so was the multirepo foundation roadmap.

Two routine events cause it. The closeout commit extends the scope's change set,
so the next validation run seals evidence under a new provenance digest; and each
check added to the validation matrix adds evidence the old bundle never saw. At
the 6.0.0 pin these scopes were `closed` because their bundles had just been
renewed. The renewal cascade after 6.0.0 resealed and reattested 28 scopes, and
this release would have needed about as many, for no change in what was reviewed.

`review-check`, which `close`, `check` and `closeout-check` use, has retained a
completed scope's approved bundle since review bundles exist. `review verify` and
`federatedSourceProof`, which read the scope through `VerifyReviewBundle`, never
did, so the engine gave two answers about the same scope and federation used the
stricter one.

### Constraints

A newer review of a closed scope must still win: a rejection or a request for
changes cannot be overridden by an older approval. Open scopes are unaffected:
their superseded bundles stay superseded. Federated acceptance keeps every other
proof it requires (terminal closeout, trust pin, evidence eligibility at the
pinned revision).

### Non-goals

Narrowing which part of the validation matrix a bundle seals. It would not help:
the closeout commit alone supersedes the bundle on the next validation.

## 2. Requirements

- R1: `VerifyReviewBundle` on a done scope whose newest attested bundle still
  stands returns `closed`, fresh and approved, with that bundle and attestation,
  a warning naming the retention, and the delta to the current preparation.
- R2: A done scope whose newest attestation rejects, requests changes or no longer
  validates is not approved, even when an older bundle was approved. This also
  corrects `review-check`, which fell back past such an attestation.
- R3: An open scope with a superseded bundle is still `superseded`.
- R4: Federated acceptance accepts a closed upstream scope through the retained
  approval.
- R5: Retention applies only while the sealed federated manifest is unchanged. Revoked
  trust or a moved or unauthorized source still stales a closed scope.

## 3. Technical Plan

`retainedCompletedReview` holds the rule once: take the newest bundle that has an
attestation; retain it when that attestation validates, otherwise retain nothing.
`ReviewCheck` calls it where it had the loop inline. `VerifyReviewBundle` calls it
when no sealed bundle matches the current preparation and the scope is done,
before declaring it superseded, and only when the retained bundle's federated
manifest equals the one prepared now.

### Artifacts

- created: .pose/specs/2026-09-29-review-verify-retains-completed-scopes.md
- created: .pose/changelogs/unreleased/review-verify-retains-completed-scopes.md
- created: pose-mcp/internal/pose/completed_review_retention_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:completed-review-retention module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Closed scopes that verify reported superseded become closed; nothing that was
approved becomes unapproved, except a closed scope whose newest attestation did
not approve it, which R2 corrects. Reverting restores the renewal cascade.

## 4. Tasks

- [x] Reproduce the stale federation at the new pin and compare with the old pin.
- [x] Retain the approval of a closed scope in `review verify` through one shared rule.
- [x] Stop the retention from falling back past a newer negative attestation.
- [x] Cover retention, the negative case and open scopes in tests.
- [x] Keep a changed federated manifest staling a closed scope.

## 5. Decisions

### Decision D1

- Status: active
- Align `review verify` with `review-check` instead of teaching federation a
  separate rule. The two commands must not disagree about one scope, and every
  consumer of `VerifyReviewBundle` benefits.

### Decision D2

- Status: active
- The newest attested bundle decides. The inline loop searched every bundle for
  any standing approval, which let an older approval hide a later rejection;
  skipping only unattested bundles keeps the intent and closes that gap.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Retention and its limits | `go test ./internal/pose -run CompletedReviewRetention -count=1` | closed retains; rejection wins; open stays superseded |
| Federation | `go test ./internal/pose -run Federated -count=1` | existing federated acceptance tests pass |

### Execution log

2026-09-29: with the Harne8 pin at `c5c029c` and the trust recomputed,
`closeout-check spec:harne8-abm-governed-execution` failed with
`source-review-not-approved` for three pose-dist specs. At `eca1e25` the same
`review verify` calls returned `closed`, fresh and approved; at `c5c029c` they
returned `superseded`, with `release-runs-the-ci-gates` superseded by evidence
alone and no changed path. With the fix, the ABM sources, the multirepo foundation
roadmap and the specs closed today verify `closed`, each with the bundle it was
closed with.

2026-09-29, review of the change. The first version retained any closed scope, and
`TestFederatedRoadmapMilestoneBundleSealsOwnManifestAndKeepsLegacySeals` failed:
revoking the source's authorization no longer staled a closed milestone. Retention
is now limited to an unchanged federated manifest, and that test passes with the
full suite. Each new test was proven against the defect: without the retention the
closed-scope case reports `superseded`; with the old fallback loop the rejection
case reports an approval. With the corrected binary, Harne8's
`closeout-check spec:harne8-abm-governed-execution` at pin `c5c029c` is terminal.

### Requirement trace

- R1 [pending] test:TestCompletedReviewRetention
- R2 [pending] test:TestCompletedReviewRetention
- R3 [pending] test:TestCompletedReviewRetention
- R4 [pending] harne8 closeout-check
- R5 [pending] test:TestFederatedRoadmapMilestoneBundleSealsOwnManifestAndKeepsLegacySeals

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

A closed scope's approval now survives later evidence changes under `verify` as it
already did under `review-check`. A semantic change to a closed spec is not caught
here; it is the job of the amendment gate, which reads requirements, assumptions
and decisions, not of review freshness.

### Follow-ups

- [open] One malformed draft spec is reported as a delivery-contract error on every other spec (148 in this repository) instead of once on itself (owner:@pose-maintainers crit:low review:2026-10-13)
