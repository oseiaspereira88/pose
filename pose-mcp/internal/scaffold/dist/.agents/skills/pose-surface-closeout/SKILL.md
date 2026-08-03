---
name: pose-surface-closeout
description: Use to close a POSE spec or roadmap that delivers a UI surface, contract, capability, infrastructure or governance target only after production composition, reachability, evidence freshness and roadmap criteria are proven. Trigger keywords - surface closeout, reachability, composed delivery, delivery target, surface-check, roadmap-check, unreachable UI, uncomposed capability.
when_to_use: A delivery-bearing spec has passed ordinary tests but must prove that its target is reachable or composed from a production entrypoint before closeout.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, validate, spec-write
---

# Skill: pose-surface-closeout

Close delivery claims only after the shared graph proves composition.

## Required reading

1. The target spec and its `delivers`, `### Artifacts`, `### Delivery targets` and requirement trace.
2. [UI surface workflow](../../../.pose/workflows/ui-surface.md).
3. [Delivery surface rule](../../../.pose/rules/delivery-surface.md).
4. The applicable validation matrix and delivery policy.

## Steps

1. Confirm artifact claims reconcile with the attributed immutable Git change set.
2. Confirm every target uses a registered profile and confined production entrypoint.
3. Run the registered validation checks into the policy result path; do not substitute raw commands or manual evidence.
4. Run `pose surface-check --spec <slug> --strict`; remediate missing, failing or stale evidence and repeat.
5. For a roadmap, run `pose roadmap-check <slug> --strict` and resolve member, criterion and graph blockers.
6. Record the normal independent review only after the composed path is green.
7. Apply guarded closeout and rerun strict structure/spec gates.

## Output requirements

- Explainable spec → artifact → delivery target → production entrypoint → current result path.
- Required evidence classes passed for the current provenance digest.
- Surface requirement trace uses `evidence:integration` or `evidence:e2e`.
- No required roadmap criterion or delivery-integrity finding remains.

## Anti-patterns

- Treating build, unit or artifact success as delivery.
- Inventing a delivery ref from source proximity.
- Accepting a stale result after artifact or entrypoint change.
- Embedding shell commands in roadmap criteria.
