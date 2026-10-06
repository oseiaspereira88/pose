# Review: validation check materiality (2026-10-06)

Decision: approved subject to the fresh strict matrix and final review/closeout gates recorded by the bundle. This is a separate review execution of the delivered implementation.

## Rules applied during review

- Change type: bugfix.
- Workflow: .pose/workflows/review.md and .pose/workflows/bugfix.md.
- backend-go: bounded detector reads and conservative error paths.
- security: name collisions, unreadable JSON and non-additive metadata changes remain material.
- documentation-style and delivery-evidence: the requirement trace and all four manual copies describe parser v2.
- knowledge: multirepo-review-continuation and module-metadata-discovery-invalidates-review-provenance were consulted; neither authorizes fabricating attribution or evidence.
- Frontend, Workers and infrastructure rules are inapplicable to this detector-only scope.

## Findings and judgments

No outstanding finding in the reviewed implementation. R1-R4 have named tests in design_delta_matrix_test.go. The walker strips only newly named checks from existing arrays and compares the whole remaining document, so a removal, edit, mode change or new module does not escape materiality. It rejects duplicate new names and collisions with names already present anywhere. JSON failures keep the prior material metadata fact. The parser version changes while already sealed bundles preserve their original facts.

Compatibility: the refinement applies only to newly resolved plans; it changes no public command, policy schema or existing bundle. Documentation: POSE.md and the English and Portuguese distributed copies describe validation-check facts. Operability: additions remain visible with their module/stack location and name; every unproven change falls back to a material fact. Scope: the ten declared paths match the attributed 470abf0 change set. Security: no execution or dependency is introduced; an ambiguous evidence identity remains consequential.

## Tool disposition

artifact-check was run with strict mode; global legacy orphan warnings are outside this spec, with no error in its claims. assess tech-debt found zero uncovered markers. assess integrate's unobserved consumer inventory is not a failed composition test and introduces no detector-specific gap. assess discover was run for pose-mcp. suggest review selected backend/security/documentation rules. assess design was inspected through the canonical bundle. Recurrence was checked. Completion tools are deferred until explicit attestation; surface-check, review verify and closeout-check must pass before close.

Validation evidence and exact plan/bundle digests are retained in the immutable bundle and attestation. Full checks are generated at .pose/results/delivery-validation.json before sealing; historical results alone do not establish fresh delivery.
