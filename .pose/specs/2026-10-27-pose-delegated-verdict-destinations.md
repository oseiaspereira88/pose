---
slug: pose-delegated-verdict-destinations
status: done
created_at: 2026-10-27
completed_at: 2026-10-28
supersedes:
depends_on: pose-delegated-review-conclusions
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegated-verdict-destinations
---

# Spec: An adjudication or smoke run records its verdict where its origin expects it

## 1. Intent

### Goal

Give the verdict of a delegated adjudication and of a delegated smoke run a governed destination, and let the engine record it there under the rules a review conclusion follows.

### Business value

R2 of [`pose-delegated-review-conclusions`](2026-10-27-pose-delegated-review-conclusions.md) asks for these verdicts to be recorded "where the origin expects them" — the adjudicated spec's decision record, the smoke target's evidence — and neither destination exists. The maintainer chose on 2026-10-27 to deliver review conclusions first and to define these destinations in a spec of their own.

### Constraints

The same checks as a review conclusion apply: an intact, signed run of a current bundle, a complete typed verdict, idempotence, refusals recorded with their reason. A verdict never becomes an approval of another scope.

### Non-goals

Review conclusions (delivered by `pose-delegated-review-conclusions`); turning the capability on.

## 2. Requirements

### Functional

- R1: An `adjudication` run that passes every check shall record its verdict as a decision record of the adjudicated spec, readable by the spec's lint and closeout, and the adjudicated questions shall carry the verdict's answer for each.
- R2: A `smoke` run that passes every check shall record, for each delivery target it exercised, an observation that the surface checks read as evidence of that target, with the commands it ran and what it observed.
- R3: A verdict that fails a check, is incomplete or does not parse, shall record no governed effect and keep its reason, as a review conclusion does.

## 3. Technical Plan

### Artifacts

- modified: .pose/specs/2026-10-27-pose-delegated-verdict-destinations.md
- created: .pose/starts/pose-delegated-verdict-destinations.json
- created: .pose/changelogs/unreleased/pose-delegated-verdict-destinations.md
- created: pose-mcp/internal/pose/delegated_verdicts.go
- created: pose-mcp/internal/pose/delegated_verdicts_test.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/surface_check_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- created: pose-mcp/internal/cli/delegated_verdicts_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-verdict-destinations module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_conclude.go

## 4. Tasks

- [x] Adjudication verdict as a decision record
- [x] Smoke observation as surface evidence
- [x] Refusals and idempotence
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: the engine writes the verdict into the adjudicated spec's Decisions or Validation section
- Selected option: an immutable adjudication record per run in the `adjudications` directory of `.pose`, naming the run and the proof the issuer signed, holding the verdict per question — each decision `D<n>` and requirement `R<n>` the spec records, which the brief of template `adjudication-brief/2` lists and the pose-draft block answers in `questions`; `lint-spec` reads the records of the spec, verifies each against its run, and fails on a question the current bundle's adjudication says does not hold. The maintainer chose this on 2026-10-27
- Rationale: the engine never edits a governed spec, so its change set stays the implementer's; a record verified against its run's proof and draft cannot say what the run did not
- Consequences: a spec's adjudications are read beside it, not in it; a question the current bundle's verified adjudication says does not hold fails `lint-spec` and refuses `close`; an adjudication of an earlier bundle is kept but no longer gates either. Every record is admitted on read under what its conclusion required: its proof signed by the issuer it names, the independence its plan requires, and a verdict answering exactly what was asked (review round 1)
- Falsifier: an adjudication record that verifies while stating a verdict the run's draft did not give

### Decision D2
- Basis: R2
- Minimal option: a record of its own that the surface check gains a reader for
- Selected option: validation results in the format `pose validate` writes, of the new evidence class `smoke`, one required check per delivery target the smoke run exercised, passing when it was observed, in the `results/smoke` directory of `.pose` beside the results path, which `surface-check` reads after verifying each document against its run; a delivery profile may then demand `smoke`. The maintainer chose this on 2026-10-27
- Rationale: the surface check already decides delivery from validation results by class, module and provenance; a smoke observation is one more result, not a new kind of evidence
- Consequences: `smoke` joins the closed vocabulary of evidence classes; no profile demands it until an instance adds it. A smoke result names the target and the spec it observed and is evidence of that target only; its head, time and provenance come from the run, its bundle and the current index, never from the document's bytes, so it is current only while the run's bundle governs its scope (review round 1)
- Falsifier: a smoke result the surface check counts that no verified smoke run observed

### Decision D3
- Basis: R3
- Minimal option: separate commands to record each verdict
- Selected option: `pose review conclude` and the automatic conclusion handle adjudication and smoke runs under the same checks, lock, refusals and idempotence as a review conclusion, plus one more check: the verdict answers exactly what the brief asked — every question, every delivery target, nothing else
- Rationale: one path keeps one bar; a verdict missing a question or naming an unasked target is not what the run was asked
- Consequences: runs briefed with `adjudication-brief/1` and `smoke-brief/1` carry no typed verdict and are refused with that reason, while still verifying with their template
- Falsifier: a recorded verdict that leaves a question or a target unanswered


## 6. Validation

### Strategy

Fixtures for each kind: a passing run records its verdict at its destination; each failed check, an incomplete verdict and a replay record nothing.

### Requirement trace

- R1 [satisfied] Immutable per-run adjudication answers the exact decisions and requirements and is verified on read; current negative answers gate lint and serialized close: test:TestAdjudicationVerdictIsRecordedOnTheSpec test:TestAdjudicationVerdictFailsLintSpec test:TestAdjudicationBlocksTheCloseout test:TestCloseAndAdjudicationAreSerialized check:pose-mcp/go/test commit:7008d6d183f1a68f663980cb3e1d554218383ebf
- R2 [satisfied] Verified smoke produces derived validation results for exactly the observed target and spec; CLI surface-check consumes them in both evidence lists and rejects unverified or differently targeted observations: test:TestSmokeVerdictIsRecordedAsSurfaceEvidence test:TestSmokeResultsAreDerivedAndTargeted test:TestSurfaceCheckReadsSmokeObservationsPerTarget check:pose-mcp/go/test
- R3 [satisfied] Exact typed coverage, proof/issuer/kind, independence and supported schema are required; refusal publishes no effect and busy lifecycle reservations remain recoverable: test:TestSmokeVerdictRefusesAnUnaskedTarget test:TestVerdictsMeetThePlansIndependence test:TestVerdictRecordsVerifyLikeTheConclusion test:TestSmokeResultsRefuseAnUnsupportedSchema test:TestAdjudicationSurvivesADeadLifecycleLock check:pose-mcp/go/delegated-run-provenance-integration

## 7. Final Report

### Delivered scope

Delivered R1–R3 through implementation commit `7008d6d183f1a68f663980cb3e1d554218383ebf`: immutable per-run adjudications, exact decision/requirement coverage read by lint and closeout, and smoke validation results derived from verified run data for the observed target and spec only. Shared conclusion admission preserves proof, typed-content binding, independence, idempotence, serialized publication/close and recovery after a dead lifecycle-lock owner. Legacy /1 briefs remain byte-compatible and their runs verify; they do not gain typed verdict effects. Four manuals, help and changelog describe the distinct destinations. Capability adoption is not claimed.

### Independent review and evidence

Reviewer `agent:independent-gpt-6.1-sol-review`, maintainer-authorized Codex/OpenAI execution, reviewed the Claude/Anthropic implementation, Claude Opus 5.5 session `f2a4d9f2-d557-4064-ae96-6ed52bb47cd2`. The engine-required `agent:independent-` prefix is a declaration; local attestations are unsigned and synthetic fixture signatures do not authenticate vendor or cognitive independence. The bundle was sealed with explicit lead `anthropic/claude-opus-5-5`; the warning about the absent Anthropic session reference in this Codex environment is expected.

Bundle `rvb-57cfad060c87c98c`, digest `sha256:57cfad060c87c98c276dccc710cad452a7e4d3b0e02a663971d2c17c2eef8f01`, plan `sha256:ada579c8ed38d9afb2242c9df934c7929e415080e315a1e5edac8d4660ad8b1c`; approved attestation `rva-89954037e256fc74`. Native verification and review-check confirmed a fresh approved review before the guarded close. Independent full Go suites passed in C without session markers and in pt_BR.UTF-8; vet, Windows/Darwin builds, focused race, gitleaks and govulncheck passed. Current global strict validation passed 158/158 checks; module validation passed 155/155 on the implemented commit.

Independent adversarial probes and mutation counterexamples reconciled the prior independence, provenance/target, exact coverage, closeout, concurrency, schema, issuer and real-child-process crash-recovery findings. F04 is resolved by the native CLI regression: removing its consumer or either target-selection filter compiles but fails its assertions. That test substitutes the loader boundary; actual smoke-document verification remains covered in the pose package. Local review evidence, including real process exit after lock acquisition and native recovery followed by one verified same-run verdict, remains in `.review-work/round5/` as requested. The canonical committed validation results and sealed review records carry the native evidence.

### Validation observations

The first round-5 global validation passed 157/158 checks but failed the install/upgrade journey with `contract-nodes is not on after the answer was applied`. Its full output was preserved. The simultaneous module validation passed that check, the maintainer reported eleven successful focused runs, and the authorized fresh global validation passed all 158 checks. This is recorded as an intermittent check observation, not a finding attributed to this diff. No production or test fix was made by this reviewer.

### Residual risks

No unresolved review finding remains for this spec. Evidence establishes admission, persistence, provenance and CLI composition over fixtures; it does not prove an external service was physically exercised. Windows and Darwin were compiled rather than executed. Integration's pre-existing observational consumer gaps and tech-debt's incomplete scan of two pre-existing invalid specs are disclosed without claiming complete source coverage or new regressions. Declared reviewer identity and unsigned attestation are existing assurance limits.

### Follow-ups

- [done] F04 (medium in rounds 1–4): versioned CLI smoke regression now covers absent/present, another target/spec and unverified observations in both required and any evidence lists; three independent consumer/filter mutations fail its native assertions.
