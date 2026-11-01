---
slug: pose-milestone-closes-when-reached
status: draft
created_at: 2026-11-01
supersedes:
depends_on: pose-closeout-judges-the-manifest
remediates:
priority: 1
components: pose-mcp
task_type: feature
delivers: capability:milestone-closes-when-reached
completed_at:
---

# Spec: A milestone closes when it is reached

## 1. Intent

### Goal

A milestone can be closed as soon as its own edges allow it, and the engine says so at that moment.

### Business value

The eleven milestones of `delegation` piled up for the end of the roadmap for two reasons. Nothing told the operator that a milestone had been reached, and the gate applied the `depends_on` of the roadmap to every milestone, so none could close before `delegated-review` did. The error met on the way, `source-artifact-revision-mismatch`, meant only that the closed roadmap had not been committed.

### Constraints

- A milestone still answers to its `after:` edges, its members and its own `consumes`.
- A roadmap still answers to its `depends_on` in full.
- Closed milestones keep their approvals: no reseal follows from this spec.
- The new signals add no cost to `pose check`.

### Non-goals

- Refusing `pose start` over an unclosed `after:`: it warns. Refusal is a policy for an instance to decide, and no instance has asked for it.
- The `consumes` of a roadmap, which still reaches its milestones: it records outcome consumption, not an order, and it held nothing up here.
- A separate lighter track for small fixes.

## 2. Requirements

### Functional

- R1: The federated acceptance of a milestone shall not carry the `depends_on` of its roadmap, and the acceptance of the roadmap shall.
- R2: A closed milestone sealed with an edge that the `depends_on` of its roadmap still names shall keep its approval, and any other edge missing from the current manifest shall stale it.
- R3: When `pose close spec:` leaves a milestone with every member terminal and without approval, it shall name `pose close milestone:` for that milestone, and when `pose close milestone:` leaves every milestone of a roadmap verified, it shall name `pose close roadmap:`.
- R4: `pose state --attention` shall list a milestone of a roadmap not yet done whose members are all terminal and which has no approval.
- R5: `pose start` shall report, without refusing, a milestone that the spec's own milestone names in `after:` and that has no approval.
- R6: A federated source whose working file differs from the file at its revision shall be reported as `source-artifact-uncommitted`, and the blocker shall say that the artifact has to be committed.
- R7: `pose lint-spec --strict` shall fail on an in-progress spec with a changed node that no amendment acknowledges.
- R8: The closeout skill, in both locales and in the scaffold, shall tell the operator to close a milestone when it is reached.

### Non-functional

- `pose check` reads no milestone state: the two signals run in `pose close`, `pose start` and `pose state --attention`, which already pay for closeout state.

### Security

- R1 loosens a gate that protects authority, so it comes with an ADR. A milestone gains no path to approval except over the one edge the ADR moves to the roadmap.

### Compatibility

- Bundles sealed with the wider manifest are read as they are, by R2. New seals carry the narrower one.
- An instance with an in-progress spec edited without an amendment sees `lint-spec --strict` fail earlier than `pose close` would. `--tolerant` tolerates it, as it does every failure.

## 3. Technical Plan

### Affected areas

`internal/pose/federated_acceptance.go`, `internal/pose/review_bundle.go`, `internal/pose/obligation_projection.go`, `internal/pose/roadmaps.go`, `internal/cli/review_closeout.go`, `internal/cli/start.go`, `internal/cli/lintspec.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-milestone-closes-when-reached.md
- created: .pose/starts/pose-milestone-closes-when-reached.json
- created: .pose/changelogs/unreleased/pose-milestone-closes-when-reached.md
- created: .pose/adr/2026-11-01-a-milestone-answers-to-its-own-edges.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/start.go
- modified: pose-mcp/internal/cli/lintspec.go
- created: pose-mcp/internal/pose/milestone_closes_when_reached_test.go
- created: pose-mcp/internal/cli/milestone_closes_when_reached_test.go

### Delivery targets

- capability:milestone-closes-when-reached module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

A milestone's manifest no longer lists the `depends_on` of its roadmap. `pose close` prints `close.next`, `pose start` prints `start.warning`, and `pose state --attention` gains the rule `milestone-reached`. The reason `source-artifact-revision-mismatch` becomes `source-artifact-uncommitted`. `lint-spec --strict` fails where it warned.

## 4. Tasks

- [ ] Narrow the milestone manifest and keep what was sealed wider
- [ ] Name the next close, list reached milestones, warn at start
- [ ] Say what the revision mismatch means
- [ ] Make strict lint fail on an unacknowledged change
- [ ] Update the closeout skill
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: document that a milestone waits for the dependencies of its roadmap
- Selected option: drop `depends_on` from the edges of a milestone, as recorded in `.pose/adr/2026-11-01-a-milestone-answers-to-its-own-edges.md`
- Rationale: the roadmap file declares the order of its milestones in `after:`, and `depends_on` says what the roadmap as a whole needs before it is done; applying it to each milestone made the whole order wait on one edge
- Consequences: a milestone can close while its roadmap's prerequisite is open, and the roadmap cannot
- Falsifier: a milestone whose outcome holds only once the prerequisite of its roadmap is done, with no `after:` saying so

### Decision D2
- Basis: R2
- Minimal option: reseal the eleven milestones of `delegation`
- Selected option: excuse a sealed edge the current manifest lacks, only for a milestone and only when the current `depends_on` of its roadmap still names it
- Rationale: the edge moved to the roadmap by rule and was not removed by anyone; a reseal would redo eleven reviews of unchanged work
- Consequences: one more case in `federatedManifestStands`, with a test for each edge it must not excuse
- Falsifier: an edge dropped by an edit of the roadmap that keeps the approval

### Decision D3
- Basis: R3, R4, R5
- Minimal option: a warning in `pose check`
- Selected option: signals in `pose close`, `pose start` and `pose state --attention`, none in `pose check`
- Rationale: `pose check` already takes 425 seconds here, and those three commands are the moments the operator acts on
- Consequences: a run that only reads `pose check` does not see a reached milestone
- Falsifier: reached milestones piling up on a roadmap while these signals are printed

### Decision D4
- Basis: R7
- Minimal option: keep the check in the operator's closeout script
- Selected option: the finding is an error on an in-progress spec as on a done one, so strict lint fails and tolerant lint tolerates it
- Rationale: the finding was already computed and only its severity let it pass, which cost a full revalidation, seal and attestation
- Consequences: an edit of a requirement needs its amendment before strict lint passes, not only before close
- Falsifier: a legitimate in-progress edit that cannot be amended

## 6. Validation

### Strategy

Integration tests over real roadmaps and commits, with each gate seen failing first, and a measure of every closed milestone of this repository with both readers: none may change.

### Requirement trace

- R1 [planned] capability:milestone-closes-when-reached evidence:integration check:milestone-closes-when-reached-integration test:TestMilestoneClosesBeforeTheDependencyOfItsRoadmap test:TestRoadmapStillWaitsForItsDependency
- R2 [planned] check:milestone-closes-when-reached-integration test:TestSealedRoadmapEdgeKeepsTheApproval test:TestOtherMissingEdgeStales
- R3 [planned] check:milestone-closes-when-reached-integration test:TestCloseNamesTheNextClose
- R4 [planned] check:milestone-closes-when-reached-integration test:TestAttentionListsReachedMilestones
- R5 [planned] check:milestone-closes-when-reached-integration test:TestStartWarnsOfAnUnclosedAfter
- R6 [planned] check:milestone-closes-when-reached-integration test:TestUncommittedSourceSaysSo
- R7 [planned] check:milestone-closes-when-reached-integration test:TestStrictLintFailsOnAnUnacknowledgedChange
- R8 [planned] check:milestone-closes-when-reached-integration test:TestCloseoutSkillNamesTheMilestoneClose

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
