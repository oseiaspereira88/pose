# ADR: Cross-repo portfolio — reuse the MCP server's own project authorization, no new discovery mechanism

## Status
Accepted (2026-07-19) — spec `pose-cross-repo-portfolio`

## Context

The spec's Constraint is exact: "repositories remain authoritative;
central state is a reconciled projection." POSE already has a working,
tested authorization boundary for "which other repositories may this
process see" — the MCP server's multi-project support
(`pose.ScanProjectsDir`/`HARNE8_PROJECTS_DIR`, `pose.ParseRootsJSON`/
`POSE_PROJECT_ROOTS`), which never walks the filesystem beyond an
explicitly configured directory and never returns a directory lacking a
`.pose/` marker. A cross-repo feature building its own, second discovery
mechanism would either have to duplicate that boundary (two places to
keep in sync) or be looser than it (a real authorization gap).

Alternatives considered:

1. **A new, feature-specific project registry** (e.g. a
   `.pose/indexes/portfolio-projects.json` listing other repos by path).
   Adds a second source of truth for "which repos can this process read"
   alongside the one the MCP server already has — exactly the kind of
   duplication that drifts.
2. **An unrestricted scan of a configured parent directory for any
   `.pose/`-containing subdirectory**, no allowlist. Simpler, but
   violates the Security requirement ("enforce tenant/project
   authorization") outright — any directory dropped next to the current
   repo would silently join the portfolio.
3. **Reuse `pose.ScanProjectsDir`/`pose.ParseRootsJSON` verbatim** — the
   exact same authorization boundary the MCP server already enforces —
   plus the current repository itself (as `POSE_DEFAULT_PROJECT_ID`, or a
   `proj.<dirname>` default matching `pose install`'s own convention).

## Decision

Option 3.

- **`discoverAuthorizedProjects`** (`internal/cli/portfolio_projection.go`)
  calls `pose.ScanProjectsDir(projectsDir, "")` and
  `pose.ParseRootsJSON(os.Getenv("POSE_PROJECT_ROOTS"))` — the identical
  functions and env vars `internal/bootstrap.Run` already uses to build
  the MCP server's project registry. A repository not registered through
  either path is invisible to the projection, full stop; a cross-repo
  reference to it resolves as `unauthorized-project`, never silently
  ignored or (worse) silently read anyway.
- **Cross-repo identity is `xref:<project_id>/<spec-slug>`**
  (`depXrefRE` in `lintspec.go`), additive to the existing
  `other-spec`/`milestone:.../roadmap:...` forms in `depends_on` — a
  `pose lint-spec --ready-check` on an existing spec with a local-only
  `depends_on` is completely unaffected (Compatibility).
- **Staleness is an honest proxy, documented as such**: since there is no
  network fetch (a projection is a *local* reconciliation across
  directories, per the Constraint), staleness is measured as "how long
  since this project's newest spec file was modified" — a real, if
  approximate, freshness signal without inventing a fake "last synced"
  timestamp. `--max-staleness-days` (default 7) is explicit and
  overridable.
- **Blocked/stale/unauthorized/unknown are four distinct, explicit
  reasons** (R2), never collapsed into a single boolean: `Resolved`
  (found and readable), `Blocking` (target isn't `done`), and `Reason`
  (`unauthorized-project` | `unknown-spec` | `stale-source`) are
  orthogonal fields on `xrefResolution` so a reader never has to guess
  which failure mode they're looking at.
- **Ownership and criticality come from the target project's own
  `module-metadata.json` defaults** (`loadModuleMetadata`, already used
  by `pose index`) — no capacity, velocity or ETA field exists anywhere
  in `projectedSpec`; R3's "without fabricating capacity" is a structural
  absence, verified by a test that greps the whole JSON output for those
  words.
- **No filesystem path of any project ever appears in the output** —
  `projectedSpec`/`xrefResolution` carry only the logical `project_id`
  and `slug`; `TestPortfolioProjectionNeverLeaksFilesystemPaths` asserts
  this directly against both fixture roots' absolute paths.
- **Revisioned with tombstones** (Data/storage changes): each run reads
  the previous `.pose/reports/portfolio-projection.json` (if any) and
  carries forward a tombstone (with `removed_at`) for every
  `(project, slug)` pair present before and absent now, until it
  reappears — a disappearance is explicit, never a silent gap in the next
  read.

## Consequences

- Positive: the authorization boundary cannot drift between the MCP
  server and this CLI feature — they call the exact same functions, so a
  future change to project discovery/authorization automatically applies
  to both.
- Positive: zero new environment variables or config files — an operator
  who already configured `HARNE8_PROJECTS_DIR` for the MCP server gets
  cross-repo portfolio projection for free.
- Negative: staleness is filesystem-mtime-based, not a real "last
  successfully synced" signal — acceptable for a purely local
  reconciliation (no network fetch exists to timestamp), but a future
  spec that adds real remote fetching would need a better freshness
  signal than mtime.
- Neutral: the xref grammar (`xref:<project_id>/<slug>`) intentionally
  mirrors the existing `milestone:<roadmap>/<id>` shape rather than
  inventing a different separator or ordering — one grammar family to
  learn, not two.
