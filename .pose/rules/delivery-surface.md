# Rule: Delivery surface assurance

## Apply when

A spec changes a configured product-surface or composition root, declares a
typed delivery target, or participates in a roadmap cut criterion.

## Required

- Keep `delivers` and `### Delivery targets` ref sets identical.
- Use only registered validation-matrix checks and closed evidence classes.
- Require `reachability` plus `integration` or `e2e` for surfaces.
- Require `integration` for composed capabilities and contracts.
- Bind passed results to the current provenance digest; rerun when stale.
- Keep roadmap criteria declarative: typed refs, check names or confined manual reports only.

## Block

- Raw commands in specs or roadmap criteria.
- Build/unit success presented as proof of composition.
- Manual evidence used to satisfy mandatory reachability.
- A changed delivery root with no typed declaration.
