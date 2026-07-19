# Kit: Spec Kit import

`fixture/` is a small pre-existing repository already using [GitHub Spec
Kit](https://github.com/github/spec-kit) for one in-flight feature
(`.specify/specs/001-user-notifications/`), with `spec.md` and `tasks.md`
present but — realistically — no `plan.md` yet.

Verified end to end by `TestBrownfieldSpecKitImportKit`.

## Stage 0 — adopt POSE first

```bash
pose install . --skip-mcp
```

## Stage 1 — visibility

```bash
pose import spec-kit .specify/specs --dry-run
```

Reports what would be imported (`dry_run=true`); nothing is written.

## Stage 2 — import

```bash
pose import spec-kit .specify/specs
```

Writes `.pose/specs/user-notifications/spec.md`. `README.md` and
`src/notify.py` are untouched. Because this fixture's `plan.md` is
missing, the import prints a curation warning
(`import.curation slug=user-notifications warning="plan.md not found; ..."`)
instead of silently guessing — the generated Technical Plan section
becomes an explicit "curate this" placeholder, and the source path plus
every consumed artifact are recorded under `## 8. Import Provenance`.

## Stage 3 — readiness and follow-up

```bash
pose lint-spec user-notifications --ready-check   # spec.ready=true
pose followups --open                             # lists the curation follow-up
```

Readiness passes structurally — Intent/Requirements/Technical Plan are all
non-empty and R1–R3 have stable IDs — but the open follow-up (auto-created
by the importer) is exactly where the missing `plan.md` gap is tracked.
Curate it, close the follow-up, then promote the spec past `draft`.

## Rollback

Nothing pre-existing was modified. `git clean -fdx` (or `rm -rf
.pose/specs/user-notifications`) fully reverts the import.
