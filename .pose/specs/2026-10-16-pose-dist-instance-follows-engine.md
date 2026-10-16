---
slug: pose-dist-instance-follows-engine
status: done
created_at: 2026-10-16
completed_at: 2026-10-16
depends_on: pose-review-approval-domain-preflight
priority: 0
components: docs
task_type: refactor
changelog: none
---

# Spec: Keep pose-dist's own instance on the engine it ships

## 1. Intent

### Goal
Record the engine at main (B02 merged, `268486b2`) as the engine this repository's own instance runs, and keep it there after each consolidation batch merges.

### Constraints
Run the native update without force and without self-update, with a binary built at the merged revision. The distributed POSE.md is this repository's scaffold template and is left as it is. No policy, capability or release changes.

## 2. Requirements

- R1: The native update without force records `engine_version` 7.4.1-dev, built at `268486b2bc473efee740cb513d4578755e607e9b`.
- R2: A repeated update produces no further change, and the instance's POSE.md, policies, ledgers and content are preserved.
- R3: `pose check --strict` passes, without the warning that the instance was last updated by an older engine.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-10-16-pose-dist-instance-follows-engine.md
- created: .pose/starts/pose-dist-instance-follows-engine.json
- modified: .pose/state/machinery-manifest.json

The instance was last stamped by 7.1.0 (`7eeca305`); the published 7.2.0 to 7.4.1 were not adopted here, so `pose check` warns that the instance is older than the engine. Rehearsed in a throwaway clone at `268486b2`: `pose update --no-self` changed only the manifest stamp (7.1.0 to 7.4.1-dev); the managed workflows, rules, templates and skills already matched; POSE.md was skipped (`unresolved scaffold placeholder`), which keeps the hand-maintained root manual, the source of the scaffold copies, out of the update's reach. Both the published 7.4.1 binary and the merged-engine binary read the stamped instance without warnings.

## 4. Tasks

- [x] Rehearse the update in a throwaway clone and inspect the real diff.
- [x] Apply the native update twice and verify idempotence.
- [x] Run strict checks, obtain review and close the adoption.

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: adopt only published releases, as the 6.x adoptions did
- Selected option: stamp the instance with the engine merged at main after each consolidation batch
- Rationale: during the consolidation program the batches change the managed contract between releases; an instance stamped by 7.1.0 in the repository that ships 7.4.1-dev makes every check warn and hides a real version drift
- Consequences: the stamp carries `-dev` between releases; a release adoption still stamps the published version
- Falsifier: an update that rewrites the root POSE.md or any instance-owned content here

## 6. Validation

### Strategy
Rehearse, apply twice, then run the strict check with both binaries.

### Execution log

2026-10-16: rehearsal in a throwaway clone at `268486b2` with the binary `vcs.revision=268486b2`, `vcs.modified=false`: dry-run and apply both reported one file (`.pose/state/machinery-manifest.json`, `engine_version` 7.1.0 to 7.4.1-dev) and skipped POSE.md. `pose check --strict` went from 30 warnings, including `instance: last updated by POSE 7.1.0, older than this engine 7.4.1`, to 29 without it; the published 7.4.1 binary (`vcs.revision=7ef3be2a`) reads the stamped instance with the same 29.

### Requirement trace

- R1 [satisfied] report:.pose/state/machinery-manifest.json — engine_version 7.4.1-dev recorded by the native update without force
- R2 [satisfied] check:test — a second update produced no change; POSE.md skipped, policies, ledgers and content preserved
- R3 [satisfied] check:test — `pose check --strict` passes without the stale-instance warning

## 7. Final Report

### Delivered scope

The native update records 7.4.1-dev for this repository's own instance using
the clean engine built at `268486b2bc473efee740cb513d4578755e607e9b`.
Independent replay from the main manifest announced exactly one changed file;
apply twice restored the PR state and preserved POSE.md, policies, ledgers and
managed content. Both the merged engine and published 7.4.1 passed
`pose check --strict` with the same 29 pre-existing warnings and no stale-instance
warning. D1 is accepted for manual adoption after consolidation batches; it does
not automate future adoption or represent a published release.

### Independent review and closeout evidence

On 2026-10-16, `agent:independent-gpt-6.1-sol-review` (Codex/OpenAI) reviewed the
full diff independently of implementer `agent:claude-opus-5-5`
(Claude/Anthropic), session `296d1fcb-e989-40ae-9362-cf3514db8b26`, with no
findings. The maintainer authorized this reviewer id. The `agent:independent-`
prefix is an engine-required declaration until
`pose-delegated-review-run-provenance` replaces it; identity is declared, without
a signature or human confirmation. Attestations in pose-dist are unsigned.

Strict validation in `.pose/results/delivery-validation.json`, at implementation
HEAD `e08138c7e71872463e363dcf064ad555872fb6b9`, passed 133/133 checks, with zero
failed, errored or skipped checks. Bundle `rvb-b93df07e17076c01` seals that evidence
and plan `sha256:afeff4bc88ec299051313a2fe23c70abbce9877be4c07561bb2a634402eff7be`.
Attestation `rva-7664d439cad0fc60` records seven explicit criterion dispositions,
five review judgments and all six tool dispositions. Strict artifact
reconciliation matched 3 claims to 3 observed paths; design assessment observed
only the manifest implementation entry and no structural deltas. Tech-debt found
zero markers; recurrence over 14 days flagged no active keys. Completion tools
were deferred until attestation: review-check then approved, and closeout-check
identified only the pending lifecycle transition. Native `pose close` completed,
strict spec lint passed, and the subsequent strict structural check passed with
29 existing warnings.

### Rules applied during review

Change type: documentation update and operational refactor. The review workflow,
documentation-style, delivery-evidence and security rules governed scope,
validation claims and diff inspection. Knowledge-governance guided consultation
of prior context; no knowledge artifact or new residual-risk finding was needed.
Language-specific rules and inter-module integration assessment do not apply:
there is no executable source, dependency, API, MCP, schema or policy change.

### Residual risks
The version stamp does not identify the exact binary revision or detect drift
between two batches sharing the same release base. This review independently
verified the current binary provenance; future batches need a fresh dry-run and
review of any managed-file changes. The 29 existing warnings and unmapped docs
component warnings remain visible; neither was hidden or fixed by this adoption.

### Follow-ups

No follow-ups were created by this review. Future batch and published-release
adoptions remain separate executions of the native update and validation flow.
