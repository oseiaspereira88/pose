# Workflow: UI surface delivery

## Objective

Prove that a human-facing surface is reachable from the production entrypoint,
not merely implemented and unit-tested.

## Steps

1. Declare `surface:<id>` in spec frontmatter and `### Delivery targets` with a confined module, profile and production entrypoint.
2. Reconcile exact artifacts with `pose artifact-check`.
3. Register structured validation checks with `evidenceClass: reachability` and `integration` or `e2e`; add a11y, design-system, contrast or visual-regression when the selected profile requires them.
4. Run `pose validate --json <policy-results-path>` and retain its provenance digest.
5. Run `pose surface-check --spec <slug> --strict` and inspect the full artifact-to-entrypoint-to-result path.
6. Trace the surface from a satisfied requirement with `evidence:integration` or `evidence:e2e`.
7. Use the normal independent review and guarded closeout flow.

## Exit gate

No required or stale surface finding remains, and the production reachability
check is current for the same provenance digest.
