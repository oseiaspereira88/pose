# Kit: OpenSpec import

`fixture/` is a small pre-existing repository already using
[OpenSpec](https://github.com/Fission-AI/OpenSpec) for one in-flight
change (`openspec/changes/add-notifications/`), with `proposal.md`,
`tasks.md` and a capability delta (`specs/notifications/spec.md`) present
but — realistically — no `design.md` yet.

Verified end to end by `TestBrownfieldOpenSpecImportKit`.

## Stage 0 — adopt POSE first

```bash
pose install . --skip-mcp
```

## Stage 1 — visibility

```bash
pose import openspec openspec/changes/add-notifications --dry-run
```

## Stage 2 — import / reconciliation

```bash
pose import openspec openspec/changes/add-notifications
```

Writes `.pose/specs/add-notifications-notifications/spec.md`, reconciling
the `ADDED Requirements` delta (with its scenarios) into POSE's
Requirements section. `README.md` and `src/auth.py` are untouched.
Because this fixture's `design.md` is missing, the import surfaces a
curation warning instead of silently guessing at a technical plan.

## Stage 3 — readiness

```bash
pose lint-spec add-notifications-notifications --ready-check   # spec.ready=true
```

## Rollback

Nothing pre-existing was modified. `git clean -fdx` (or `rm -rf
.pose/specs/add-notifications-notifications`) fully reverts the import.
