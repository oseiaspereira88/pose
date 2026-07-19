# ADR: Authoritative release version source

## Status
Accepted (2026-07-19) — spec `pose-version-contract`

## Context

Public version surfaces had drifted: the CLI reported `0.9.0-dev`
(`internal/cli.Version`, stamped by GoReleaser), the MCP server hard-coded
`serverInfo.version = "0.1.0"` and the registry metadata (`server.json`)
declared `0.9.0`. Divergent versions are a P0 credibility defect: consumers
cannot correlate a binary, its MCP identity and its release metadata, which
blocks trustworthy packaging, compatibility and supply-chain automation.

Alternatives considered:

1. **Keep per-surface constants and reconcile manually at release** — status
   quo; already failed (the MCP constant was forgotten for eight minor
   versions).
2. **Generate every artifact (including `server.json`) at build time** — no
   drift by construction, but the repository stops carrying reviewable release
   metadata and registry publication becomes coupled to the build toolchain.
3. **Single authoritative Go symbol + contract tests over checked-in
   artifacts** — one source of truth in code, drift on any surface fails
   `go test` instead of waiting for a human audit.

## Decision

Option 3. The contract, aligned with [Semantic Versioning](https://semver.org/)
and the [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle):

- `pose-mcp/internal/version.Version` is the **only** version authority. No
  other package may declare a public version literal.
- **Release builds**: the git tag is injected by GoReleaser via
  `-X github.com/harne8/pose-mcp/internal/version.Version={{ .Version }}`.
  Injection is build-time only and requires no network access.
- **Development builds** report `<next-release>-dev` (e.g. `0.9.0-dev`). The
  `-dev` suffix is the explicit development identifier required to never
  impersonate a release (`version.IsDevelopment()`).
- **Derived surfaces**: `internal/cli.Version` and the MCP
  `serverInfo.version` (stdio and Streamable HTTP) read the authority at
  build/init time.
- **Checked-in release metadata** (`server.json` `version_detail.version` and
  every `packages[].version`) must equal `version.ReleaseBase()` — the
  authority without the `-dev` suffix. Bumping the planned release version is
  therefore a single reviewable commit touching `internal/version` and
  `server.json` together.
- **Contract tests** (`internal/version/contract_test.go`, plus initialize
  assertions in `internal/mcpserver`) enumerate the surfaces and fail on any
  divergence, including a GoReleaser ldflags line that stamps a
  non-authoritative symbol.
- **Schema independence**: `.pose/schema-version` (instance/engine migration
  contract) remains an integer sequence independent of SemVer; compatibility
  is never inferred from the release version alone.

## Consequences

- Positive: version drift becomes a failing test at development time; release
  automation (packages, compatibility matrix, signing) can trust one value;
  `pose version`, MCP identity and registry metadata agree for the same build.
- Positive: no workflow identity, credentials or build-environment data enter
  the public version string.
- Trade-off: locally built binaries (`go build` without ldflags) always show
  `-dev`; distributing an unstamped binary as a release is now visibly wrong,
  which is intended.
- Follow-up owner: the release-compatibility milestone
  (`pose-release-compatibility-matrix`) extends this contract to schema,
  scaffold and docs compatibility per release.
