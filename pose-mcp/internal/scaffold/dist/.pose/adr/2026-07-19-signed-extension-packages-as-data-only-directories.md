# ADR: Signed extension packages as data-only directories

## Status
Accepted (2026-07-19) — spec `pose-extension-catalog-lifecycle`

## Context

POSE could already be extended by hand-editing repository data (task-map
entries, rules, workflows, skills, matrix checks — the "Extension
boundaries" already documented in the architecture). What was missing was a
*lifecycle* around that: an operator adopting a third-party skill had no
manifest, no install/removal contract, no conflict detection, and no
signature check — they just copied files in. The spec's own non-goals are
strict: no unmoderated marketplace, and — critically — never execute
installer scripts. An extension mechanism that runs arbitrary code on
install would reopen exactly the untrusted-remote-execution risk the safe
validation orchestration spec (`pose-safe-validate-orchestration`) had just
closed for a different surface.

Alternatives considered:

1. **tar.gz/zip archives with extraction** — the OCI Distribution
   Specification's natural transport, but safe archive extraction (symlink
   rejection, zip-slip/path-traversal defense) is a real, separate security
   surface. This codebase already solved the equivalent problem for
   untrusted external content a different way (Spec Kit/OpenSpec import:
   operate on an already-materialized, symlink-rejected directory) — no
   value in re-deriving that from scratch.
2. **A plugin runtime that executes install scripts** — exactly the
   forbidden pattern; rejected outright.
3. **Directory-based packages (manifest + `files/` tree), content-digested,
   Sigstore-signed, installed via a transactional file-copy lifecycle.**

## Decision

Option 3:

- **Manifest (R1):** `extension.json` declares `id`, `version`, `kind`
  (`skill|workflow|rule|import-adapter`), `pose_schema_range`, `files`
  (repo-relative targets), `permissions` (prefixes the package may write —
  validated to be a subset of the global `extensionWhitelist`:
  `.agents/skills/`, `.pose/workflows/`, `.pose/rules/`,
  `.pose/templates/`), optional `conflicts_with` and `provenance`
  (source, commit, expected Sigstore signer identity/issuer). A
  `revoked: true` manifest — the catalog's revocation mechanism — is
  rejected unconditionally, with the reason surfaced to the operator.
- **Transactional lifecycle (R2):** `install`/`remove` are dry-runnable
  (`--dry-run`, prints the exact plan and stops), require explicit consent
  (`--yes`), and apply as a single transaction: every file about to be
  overwritten has its bytes captured first; any failure anywhere in the
  loop rolls back every change already made in that call — deleted if new,
  restored if overwritten. A repository is never left in a half-applied
  state. Ownership is tracked in `.pose/indexes/extensions.lock.json`
  (per-file content digest at install time); `remove` compares current
  content against that digest and refuses to touch a file the user has
  since modified without `--force` — user modifications are preserved by
  default, as required.
- **Conflict detection:** a target already owned by a *different*
  extension, or present on disk but untracked by any extension, blocks the
  operation (named explicitly) unless `--force`. `conflicts_with` in the
  manifest additionally blocks installing alongside a declared-incompatible
  extension.
- **Signature verification (R3, Security):** unsigned packages are
  rejected by default; verification shells to `cosign verify-blob` against
  the identity the *package's own manifest* declares (unlike release
  artifacts, an extension has no single pinned publisher — the manifest
  carries its own expected signer/issuer, and trusting a catalog at all is
  the operator's decision to point POSE at it). `--allow-unsigned` is an
  explicit, loudly-named opt-out for local development — never a silent
  default.
- **Deterministic digest** (non-functional): the package digest hashes
  every declared file's path and content in sorted order, independent of
  directory-walk or map-iteration order.
- **MCP exposure is read-only:** `pose_extension_list` projects the lock
  file; `install`/`remove` stay CLI-only, consistent with the architecture
  principle that POSE never exposes general-purpose write tools over MCP.
- **"Custom import adapters" (R3)** are supported as a first-class `kind`
  value — an extension can declare itself an import-adapter manifest that
  plugs into the existing Spec Kit/OpenSpec import mechanism as data, the
  same data-only boundary as every other kind; POSE does not gain a new
  code-execution surface to support them.

## Consequences

- Positive: extensions are installable, removable, inspectable and
  revocable without forks, without a hosted marketplace and without ever
  running third-party code — the non-goal is upheld by construction.
- Positive: transactional rollback and digest-tracked user-modification
  detection make the lifecycle safe to retry and safe to leave alone.
- Trade-off: directory-based packages (not a single-file archive) are less
  convenient to distribute as one artifact than a tar.gz — acceptable
  given the safety win of reusing an already-proven trust model instead of
  new extraction code.
- Residual: there is no live, hosted catalog-discovery service — an
  operator supplies a package (or catalog directory) explicitly, matching
  the "no unmoderated marketplace" non-goal; a future hosted catalog
  remains a distinct, separately-reviewed decision.
