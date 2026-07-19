# ADR: Bounded release compatibility policy

## Status
Accepted (2026-07-19) — spec `pose-release-compatibility-matrix`

## Context

Nothing proved that a POSE release's parts — binary, repository schema,
embedded scaffold, MCP metadata and public documentation — were mutually
compatible, nor that an instance installed by a prior release upgrades cleanly
to the candidate. Two historical tags (`v0.1.0`, `v0.1.1`) predate the current
version, catalog and install contracts, so claiming upgrade support from them
would be an untested promise. An unbounded compatibility matrix would also
make release latency grow with every version
([SemVer](https://semver.org/) alone cannot express repository schema
compatibility, and [TUF](https://theupdateframework.io/) motivates
authenticating any prior artifact before executing it).

Alternatives considered:

1. **Test compatibility ad hoc in release notes** — unverifiable prose;
   exactly the drift this roadmap eliminates.
2. **Support every historical release** — unbounded matrix, and pre-0.9.0
   tags were built before the authoritative version contract existed.
3. **Versioned matrix with a bounded window, gated in release CI** — a
   machine-readable artifact declares exactly what is supported; the gate
   proves it per candidate.

## Decision

Option 3. The compatibility contract:

- **`compatibility.json`** (repository root, versioned) declares
  `engine_version`, `schema_version`, the support policy and
  `supported_upgrades` — prior releases whose upgrade path is tested. It is
  published as a release asset together with the generated
  `compatibility-report.md`.
- **Support window starts at 0.9.0.** Pre-contract tags (`v0.1.x`) are not
  supported upgrade sources. From the next release onward, each prior
  supported release is added to `supported_upgrades` with the SHA-256 pin of
  its `checksums.txt`; entries are pruned when a release leaves the window.
- **Two independent axes:** binary SemVer (authoritative version contract)
  and repository schema (`.pose/schema-version`, integer sequence, `pose
  upgrade`, downgrade always an error). `TestCompatibilityMatrixContract`
  pins the matrix to both; upgrade and downgrade fixtures test the schema
  axis (`TestCompatibilityUpgradeFromLegacyInstance`,
  `TestCompatibilityDowngradeRejected`).
- **Release gate** (`tests/release/compat.sh`, wired into `release.yml`):
  builds the candidate stamped exactly like GoReleaser, refuses a tag that
  diverges from the matrix, re-runs the version/catalog/install/scaffold
  contract gates against the same candidate tree, runs the installer E2E and
  exercises every `supported_upgrades` entry — downloading the prior
  release, verifying the pinned checksum before executing anything, then
  install → candidate `pose upgrade` → `pose check --strict`. The report is
  retained as a CI artifact (400 days) and attached to the release.
- **Unsupported pairs fail with actionable diagnostics** (tag mismatch names
  both values; downgrade names both schema versions and the remedy).

## Consequences

- Positive: "release ready" now has a machine-checkable definition — the
  roadmap's exit gate ("binary, schema, scaffold, MCP metadata and docs are
  mutually compatible") is a generated report, not an assertion.
- Positive: the matrix bounds release latency explicitly; adding an upgrade
  path is a reviewed one-line change with a checksum pin.
- Trade-off: the first releases carry an empty `supported_upgrades` list —
  honest, since no in-window prior release exists yet; the list grows with
  the release history.
- Residual: cryptographic signatures for the matrix and report belong to the
  supply-chain-trust roadmap (`pose-release-signing`,
  `pose-slsa-provenance`).
