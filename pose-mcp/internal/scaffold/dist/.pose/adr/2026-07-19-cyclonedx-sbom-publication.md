# ADR: CycloneDX SBOM publication

## Status
Accepted (2026-07-19) — spec `pose-cyclonedx-sbom`

## Context

Releases carried no dependency inventory: vulnerability response and
enterprise adoption reviews had to reverse-engineer what each binary
contains. The spec requires a per-release CycloneDX inventory tied to the
artifact it describes, generated from exact build inputs.

Alternatives considered:

1. **SPDX format** — equally standard; CycloneDX chosen because the spec
   names it, scanners consume it broadly and syft emits it natively
   ([CycloneDX](https://cyclonedx.org/specification/overview/)).
2. **One source-tree SBOM per release** — describes the repository, not the
   shipped artifact; a consumer cannot tie it to their download.
3. **One binary-analysis SBOM per archive** — syft scans the exact packaged
   artifact; Go build info embedded in the binary yields module versions.

## Decision

Option 3, generated in GoReleaser:

- **Mapping (R2):** `<archive>.cdx.json` describes `<archive>`, produced by
  `syft scan <artifact> --output cyclonedx-json` from the exact release
  artifact (binary analysis; the analysis type is recorded by syft in the
  SBOM metadata). Filenames are stable public metadata.
- **Validation (R3):** `tests/release/verify.sh` fails the release when an
  SBOM is missing, when `bomFormat`/`specVersion`/`components` are invalid,
  or when a direct production dependency from `pose-mcp/go.mod` (parsed from
  the require list, indirects excluded) is absent from the inventory.
- **Content policy:** component versions, hashes and detected licenses are
  published; secrets, private paths and credentials never enter SBOM
  metadata (syft scans artifacts, not the CI environment). License fields
  are inventory data — incompleteness is reviewed against `NOTICE`, and no
  claim of license-risk absence is made.
- **Distribution:** SBOMs ship as release assets beside the archives and are
  themselves Sigstore-signed (ADR
  `2026-07-19-keyless-release-signing-identity`), so the inventory is tied
  to the release identity as well as the digest.

## Consequences

- Positive: scanners and enterprise intake processes consume a standard
  media type per artifact; vulnerability response can match advisories to
  exact shipped versions.
- Positive: generation is reproducible from the artifact and diff-reviewable
  between releases.
- Trade-off: binary analysis reflects the Go build info graph; tooling that
  wants source-level context can regenerate from the tagged source.
- Residual: generated license fields may be incomplete; the NOTICE review
  policy owns corrections, and the non-goal stands — an SBOM proves
  inventory, not safety.
