---
slug: pose-review-scope-trailer-check
status: in-progress
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-mcp, cli
delivers: capability:pose-mcp
---

# Spec: pose-review-scope-trailer-check

---

## 1. Intent

### Goal
Make `pose doctor` detect and warn when a spec that declares `delivers:` has
no recorded change set in `.pose/reports/history/*.jsonl`, instead of letting
the failure surface only much later as an opaque `no immutable attributed
change set exists for spec:<slug>` when someone tries to seal a review
bundle.

### Business value
Surfaced in `github.com/oseiaspereira88/pose#17` (closed, scope: root
contamination and stdio approval messaging). The issue's second and third
comments found a second, unrelated root cause for the review-bundle blocking
reported in its title, and attributed it to commits missing the
`POSE-Spec: <slug>` trailer. **That attribution turned out to be imprecise —
see Decision 2.** The trailer is real and exists in the code
(`commitsWithSpecTrailer`, `pose-mcp/internal/cli/artifact_integrity.go`),
but only feeds `resolveGitChangeSet`'s live auto-discovery fallback used by
`pose artifact-check`/`artifact-backfill --from-git` when called without
`--from`/`--to` — an ephemeral, in-memory computation that is never
persisted. `pose review bundle <scope> --seal` never calls that fallback: it
reads its subject exclusively from `.pose/indexes/delivery-integrity.json`,
which `pose index` builds solely from change sets `pose report
--change-from/--to` has persisted to `.pose/reports/history/*.jsonl`
(`loadRecordedChangeSets`). A trailer commit, on its own, changes nothing
about whether that file has an entry for the spec. The actual, verified
condition that blocks sealing is simpler: **no persisted change set exists
for the spec at all**, trailer or not.

### Constraints
- Detection only; do not attempt to auto-run `pose report` on the operator's
  behalf — picking the correct `--change-from`/`--change-to` range is a
  human decision about which commits actually delivered the spec.
- Must not require network access or any provider-specific integration —
  stays inside the existing offline `pose doctor`/local git model.
- Must not silence the warning based on trailer presence alone (see
  Decision 2) — only a recorded change set does.

### Non-goals
- Making `pose report` auto-persist a change set via trailer discovery when
  `--change-from`/`--change-to` are omitted — a real option, but a change to
  `report.go`'s contract, not just a doctor check; left for a future spec if
  wanted.
- A commit-msg hook or CI gate that enforces the trailer on every commit —
  the trailer's value is narrower than originally thought (see Decision 2),
  so this is even less of a priority than before.

---

## 2. Requirements

### Functional
- R1: When a spec has `delivers:` populated and `status: in-progress` or
  `status: done`, and no change set recorded in
  `.pose/reports/history/*.jsonl` names that spec, `pose doctor` shall warn
  that sealing a review bundle for this scope will fail, and shall point to
  `pose report --spec <slug> --change-from <rev> --change-to <rev>` as the
  remediation.
- R2: The warning message shall name the exact failure it prevents
  (`no immutable attributed change set exists for spec:<slug>`) so it is
  discoverable by someone who already hit that error and searches for it.

### Non-functional
- Cheap: reads `.pose/reports/history/*.jsonl` and spec frontmatter only, no
  git subprocess spawn per spec (unlike the trailer scan this replaces).

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go`

### Artifacts
- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/doctor_review_scope_trailer_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Planning
- [x] Capture the finding from `github.com/oseiaspereira88/pose#17`
      (comments 2-3) as this spec
- [x] Decide the exact `pose doctor` check shape and message wording
- [x] Trace `resolveGitChangeSet`/`loadRecordedChangeSets`/`pose index`/
      review-bundle sealing end to end to confirm the actual blocking
      condition before implementing (see Decision 2 — the original
      trailer-based premise did not survive this)

### Implementation
- [x] `review.scope-change-set` check in `runDoctorDiagnostics`
      (`doctor.go`): reuses `loadRecordedChangeSets` (already package-level
      in `cli`), classifies `remediationDetectable` by default (no registry
      entry) since the fix is a human decision, not an automated one

### Validation
- [x] `go -C pose-mcp test ./internal/cli/... -run TestDoctor` and full
      `go -C pose-mcp test ./...`, `go -C pose-mcp vet ./...`,
      `gofmt -l pose-mcp/internal/cli/doctor.go
      pose-mcp/internal/cli/doctor_review_scope_trailer_test.go`

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: What signal should silence the warning: trailer presence,
  recorded change set, or both?
- Options considered: (a) trailer only; (b) recorded change set only; (c)
  either one.
- Decision: (b) — recorded change set only.
- Rationale: see Decision 2; a trailer alone does not affect whether
  `review bundle --seal` succeeds, so treating it as sufficient would make
  the check silently wrong for a real case.
- Consequences: the check is simpler (no `commitsWithSpecTrailer` git
  subprocess call) and strictly accurate for the failure it targets.

### Decision 2
- Date: 2026-08-15
- Context: This spec's own Intent originally warned on "no trailer commit
  AND no recorded range", mirroring the causal claim in issue #17's second
  comment. While implementing and dogfooding it against this repository's
  own `pose-review-scope-trailer-check` spec — committed with a
  `POSE-Spec: pose-review-scope-trailer-check` trailer but, deliberately, no
  `pose report --change-from/--to` run yet — `pose review bundle --seal`
  still failed with `no immutable attributed change set exists`, exactly
  the failure this check exists to predict. Tracing the code confirmed why:
  `reviewBundleSubject` reads only `graph.ChangeSets`
  (`.pose/indexes/delivery-integrity.json`), built by `pose index` from
  `loadRecordedChangeSets` (`.pose/reports/history/*.jsonl` entries with a
  `change_set` field) — and `pose report` only ever populates that field
  when `--change-from`/`--change-to` are passed explicitly (`report.go`
  line ~126). `resolveGitChangeSet`'s trailer-scan fallback
  (`commitsWithSpecTrailer`) is reachable only from `pose artifact-check`
  and `artifact-backfill --from-git` when called with no `--from`/`--to` —
  both ephemeral, in-memory computations that never write to
  `reports/history`. A trailer commit therefore has zero effect on whether
  review bundle sealing succeeds.
- Options considered: (a) keep the trailer-based logic as originally
  planned, since it matches the issue's own investigation; (b) correct the
  check to key off recorded change sets only, and document why the
  trailer's causal role was narrower than the issue comments concluded; (c)
  drop the check entirely as based on a false premise.
- Decision: (b).
- Rationale: (a) would ship a check that can go silently `ok` on a spec that
  still cannot seal, which is worse than not having the check — it manufactures
  false confidence. (c) throws away a still-real, still-useful warning; the
  underlying "no immutable attributed change set exists" failure is real and
  worth predicting, the fix is just simpler than first modeled. Empirical
  verification against this repository's own commits is stronger evidence
  than the original issue thread's correlation-based read.
- Consequences: the delivered check, its tests, and its commit/message
  wording all key off `loadRecordedChangeSets` only; `commitsWithSpecTrailer`
  is not called from `doctor.go`. The trailer convention remains real and
  useful for `artifact-check`/`artifact-backfill` ergonomics, just not for
  this specific failure — worth a note in `AGENTS.md` someday (see
  Follow-ups), not a `pose doctor` check.

---

## 6. Validation

### Strategy
Unit tests in `pose-mcp/internal/cli` covering: a spec with a recorded
change set (no warning), a spec with only a trailer commit and no recorded
change set (warning still fires — the regression this check exists to
prevent), a spec with neither (warning fires, naming the slug and pointing
at `pose report`), and draft/no-`delivers:` specs (never counted). Exercised
against `pose doctor`'s existing test fixtures rather than a new harness.

### Requirement trace
- R1 [satisfied] `TestDoctorWarnsWhenSpecHasNoRecordedChangeSet`,
  `TestDoctorStillWarnsWhenOnlyATrailerCommitExists`,
  `TestDoctorSilentWhenChangeSetIsRecorded`,
  `TestDoctorIgnoresDraftSpecsAndSpecsWithoutDelivers`.
- R2 [satisfied] `TestDoctorWarnsWhenSpecHasNoRecordedChangeSet` asserts the
  hint names the `pose report --change-from`/`--change-to` remediation and
  cites issue #17.

### Known gaps
- None identified.

---

## 7. Final Report

### Delivered scope
A new `review.scope-change-set` `pose doctor` check that warns, per
untraceable spec, when a spec with `delivers:` populated and
`status: in-progress`/`done` has no change set recorded in
`.pose/reports/history/*.jsonl` — the exact condition under which `pose
review bundle <scope> --seal` fails with `no immutable attributed change set
exists`. No auto-fix (`remediationDetectable`): recording the correct range
is a human decision. Corrects, with empirical verification, the trailer-based
root-cause attribution from issue #17's own investigation comments (Decision
2) — trailers remain real but are irrelevant to this specific failure mode.

### Files and modules changed
- `pose-mcp/internal/cli/doctor.go`: new check #9 in `runDoctorDiagnostics`,
  `sort` import added.
- `pose-mcp/internal/cli/doctor_review_scope_trailer_test.go`: 4 new tests.

### Validation executed
- `go -C pose-mcp test ./internal/cli/... -run TestDoctor`: SUCCESS.
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l`: clean.
- Dogfooded on `pose-dist` itself: flagged `pose-debt-marker-lexical-precision`
  as a pre-existing untraceable spec in this repository, and (before the
  Decision 2 correction) caught this very spec's own commit — trailer
  present, no recorded range yet — proving the corrected logic where the
  original design would have gone silently `ok`.

### Residual risks
- None beyond the documented non-goal (auto-persisting via trailer
  discovery is out of scope here).

### Follow-ups
- [open] Note in `AGENTS.md`/the feature workflow that `POSE-Spec: <slug>`
  trailers are useful for `pose artifact-check`/`artifact-backfill
  --from-git` ergonomics (live discovery without `--from`/`--to`), but do
  not by themselves satisfy review-bundle sealing — only `pose report
  --change-from/--to` does (owner:@pose-maintainers crit:low
  review:2026-11-15)
