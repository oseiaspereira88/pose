# Monorepo validation recipes

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

POSE does not implement a monorepo orchestrator. It composes with whatever
build graph a repository already has — real Bazel, Nx, npm/yarn workspaces,
or nothing at all — through two versioned inputs: `.pose/indexes/module-
metadata.json` (dependency edges and criticality) and `.pose/indexes/
validation-matrix.json` (per-stack checks). `pose validate --changed-from`
turns those into a minimum safe selection (see
[technical architecture](architecture.md)); every recipe below is a docs-as-
test — the exact fixture layout and commands are executed in CI
(`internal/cli/monorepo_recipes_test.go`), so this page cannot drift from
what the engine actually does.

Non-guarantee: path- and metadata-based selection cannot see semantic
coupling POSE was never told about. Declare `dependsOn` edges for anything
that must widen together; an undeclared edge is a gap in your metadata, not
a POSE defect.

## Recipe 1 — JavaScript/npm workspace

```text
package.json                 # "workspaces": ["packages/*"]
packages/core/package.json
packages/app/package.json    # depends on core
.pose/indexes/module-metadata.json
```

```json title=".pose/indexes/module-metadata.json"
{
  "schemaVersion": 1,
  "modules": {
    "packages/app": { "dependsOn": ["packages/core"] }
  }
}
```

A change inside `packages/core` selects `core` directly and widens to `app`
through the declared edge:

```bash
pose validate --changed-from HEAD --explain
```

```text
[changed-scope] HEAD..worktree: 1 changed file(s), 2/2 module(s) selected
  + packages/app: depends on selected module packages/core (contains changed file)
  + packages/core: contains changed file: packages/core/index.js
```

A change to the root `package.json` (the workspace manifest itself, outside
every module) runs the whole workspace — root-level changes always prefer
safe execution:

```text
  + packages/app: root-level change outside any module: package.json
  + packages/core: root-level change outside any module: package.json
```

## Recipe 2 — declared dependency graph (Bazel-style fine-grained modules)

POSE does not read `BUILD` files. When a repository already runs Bazel (or
any other build-graph tool), declare the same edges in `module-metadata.json`
so `pose validate` can select without invoking that tool, or front the real
`bazel test //affected/...` command as a single structured check when you
want POSE to delegate execution entirely.

```text
base/go.mod
mid/go.mod                   # depends on base
leaf/go.mod                  # depends on mid
.pose/indexes/module-metadata.json
```

```json title=".pose/indexes/module-metadata.json"
{
  "schemaVersion": 1,
  "modules": {
    "mid": { "dependsOn": ["base"] },
    "leaf": { "dependsOn": ["mid"] }
  }
}
```

A change in `base` widens transitively through the whole chain — dependency
resolution is a fixed point, not a single hop:

```bash
pose validate --changed-from HEAD --explain
```

```text
  + base: contains changed file: base/changed.go
  + mid: depends on selected module base (contains changed file)
  + leaf: depends on selected module mid (depends on selected module base)
```

## Recipe 3 — mixed-language monorepo with a shared dependency

```text
services/api/go.mod
services/web/package.json
services/worker/requirements.txt
shared/go.mod                # criticality: high
.pose/indexes/module-metadata.json
```

```json title=".pose/indexes/module-metadata.json"
{
  "schemaVersion": 1,
  "modules": {
    "shared": { "criticality": "high" }
  }
}
```

`pose stacks --path <module>` identifies each module's stack independently —
go, node and python coexist in one repository with no cross-contamination:

```bash
pose stacks --path services/api     # -> # go
pose stacks --path services/web     # -> # node
pose stacks --path services/worker  # -> # python
```

A change to `services/web` selects only `web` — plus `shared`, which always
runs because it is declared `criticality: high` (policy widening, independent
of any `dependsOn` edge — this is how a repository marks a module every
change must validate, such as a shared schema or proto directory):

```bash
pose validate --changed-from HEAD --explain --json result.json
```

```text
  + services/web: contains changed file: services/web/index.js
  + shared: policy: criticality high always runs
  - services/api: not affected by HEAD..worktree
  - services/worker: not affected by HEAD..worktree
```

Severity composes across stacks in the one structured result: `shared`'s Go
check is `required`, `web`'s Node check is `optional` — a required failure
in a Python or Go module blocks the run the same way it would in a
single-language repository, and an optional failure elsewhere never does.
`services/api` and `services/worker` checks are recorded `skipped` in
`result.json` with the changed-scope reason, never silently dropped.

## What these recipes prove

- **Metadata** (`dependsOn`, `criticality`) is the only monorepo-specific
  input POSE needs — no new configuration surface.
- **Changed scope** composes with metadata: direct match → transitive
  widening → policy widening → safe-execution fallback, in that order.
- **Severity** is per-check, per-stack, and unaffected by module count or
  language mix.
- **Shared dependencies** are declared, not inferred — `criticality: high`
  is the explicit "always validate this" signal.

## Non-goals

- POSE does not replace Bazel, Nx, Turborepo or any native build graph — it
  reads declared metadata and delegates execution to structured checks.
- Perfect semantic impact analysis from paths alone is not promised; an
  undeclared dependency is a metadata gap the safe-execution fallback and
  `--explain` output make visible, not a silent risk.
