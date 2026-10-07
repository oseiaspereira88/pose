---
slug: pose-recurrence-check-resolved-clusters
status: done
created_at: 2026-10-07
completed_at: 2026-10-07
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:recurrence-cluster-resolution
---

# Spec: recurrence-check judges resolved failure clusters, not attempts

## 1. Intent

### Goal

Make `pose recurrence-check` flag a failure cluster only while its latest outcome in the window is still a failure, grouping records by `task_slug`, `report_type` and `stable_hash`, and disclose the clusters that a later `pass` resolved instead of counting them.

### Business value

The checker counts every `fail` record of a task and ignores `pass` records, so a development attempt that failed and was then fixed counts as an unresolved incident. In harne8 the strict gate has flagged `validate-native` (standard) with 24 failures between 2026-09-26 and 2026-09-28, 20 of them from the strict-profile hash `5b47855e`, whose latest record is a `pass` (2026-09-30); the other 4 belong to the tolerant-profile hash `5698000d`, which has not run since 2026-09-26. The recurrence workflow already says that a failed attempt followed by a pass is evidence but not an uncovered incident; the checker contradicts it. The decision log `escalation-validate-native` deferred this fix on 2026-08-21, its TTL lapsed on 2026-09-20 and was renewed on 2026-10-07 with a deadline of 2026-11-06. Meanwhile the gate turns CI red for a signal that ages out only after the window, which trains people to silence it by hand.

### Constraints

The history stays append-only and unchanged. No flag, allow-list or window change may hide an unresolved failure. A pass of one `stable_hash` never resolves failures of another. Resolved clusters stay visible in the output.

### Non-goals

Deciding why the attempts failed. Detecting a flaky task that alternates fail and pass (see the follow-up). Changing `recurrence-effect`, `stats` or the history schema.

---

## 2. Requirements

### Functional
- R1: When every failure of a cluster (`task_slug`, `report_type`, `stable_hash`) inside the window is followed by a `pass` of the same cluster, `pose recurrence-check` shall not flag that cluster.
- R2: When a cluster has `--threshold` or more failures after its latest `pass` in the window, or has no `pass` in the window, `pose recurrence-check` shall flag it, as today.
- R3: When a `pass` of one `stable_hash` exists, the checker shall not use it to resolve failures of a different `stable_hash` of the same task.
- R4: The checker shall list the clusters it did not flag because a later `pass` resolved them, with their failure count and the time of the resolving `pass`, in the human report and in the `--json` document (findings with code `resolved`), and shall keep `recurrence.flagged_keys` counting only flagged clusters.
- R5: A record without `stable_hash` shall form one cluster with the other records of the same task and report type, so existing histories keep their meaning.

### Non-functional
- Exit codes are unchanged: strict exits 1 only when a cluster is flagged.

### Security
- The checker only reads history files already read today.

### Compatibility
- The JSON document gains findings and a field and loses none; `--include-pass` keeps its meaning.

---

## 3. Technical Plan

### Affected areas
- `cmdRecurrenceCheck` and `historyRecord` in `pose-mcp/internal/cli/insights.go`, and the `recurrence-check` entry of the manual.

### Artifacts
- created: .pose/specs/2026-10-07-pose-recurrence-check-resolved-clusters.md
- modified: pose-mcp/internal/cli/insights.go
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-recurrence-check-resolved-clusters.md
- created: pose-mcp/internal/cli/recurrence_check_clusters_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:recurrence-cluster-resolution module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- The output of `pose recurrence-check` lists resolved clusters as findings with code `resolved` (severity `info`), and `recurrence.resolved_clusters` counts them.

### Data/storage changes
- None.

### Technical risks
- A pass can mask a cause that survives under the same `stable_hash`. The disclosure in R4 keeps the masked failures visible, and a task that alternates fail and pass stays unflagged by design until the follow-up.

---

## 6. Validation

### Strategy

Table tests over history fixtures, plus a run against a copy of the harne8 `standard-validate-native.jsonl`, which must list the strict hash as resolved and flag only the 4 failures of the tolerant hash.

### Deterministic checks

#### Test
- Command: `go test ./internal/cli -run Recurrence`
- Scope: pose-mcp
- Expected: `fail then pass` does not flag; `fail, fail, fail` flags; distinct hashes do not resolve each other; a record without hash clusters with its task; resolved clusters are disclosed.

### Requirement trace

- R1 [satisfied] test:TestRecurrenceGroupsResolveFailuresByALaterPassOfTheSameHash test:TestRecurrenceCheckDisclosesResolvedClustersAndExitsClean check:recurrence-cluster-integration
- R2 [satisfied] test:TestRecurrenceGroupsResolveFailuresByALaterPassOfTheSameHash test:TestRecurrenceCheckStillFlagsAnUnresolvedCluster check:recurrence-cluster-integration
- R3 [satisfied] test:TestRecurrenceGroupsResolveFailuresByALaterPassOfTheSameHash test:TestRecurrenceCheckStillFlagsAnUnresolvedCluster check:recurrence-cluster-integration
- R4 [satisfied] test:TestRecurrenceCheckDisclosesResolvedClustersAndExitsClean check:recurrence-cluster-integration
- R5 [satisfied] test:TestRecurrenceGroupsResolveFailuresByALaterPassOfTheSameHash check:recurrence-cluster-integration

### Known gaps
- A flaky task that alternates fail and pass is not flagged.

---

## 7. Final Report

### Delivered scope

### Residual risks
- Nothing yet; the spec is a draft.

### Follow-ups

- [open] Detect a task that alternates fail and pass many times in the window, as a separate non-blocking signal (owner:@pose-maintainers crit:medium review:2026-11-06)
