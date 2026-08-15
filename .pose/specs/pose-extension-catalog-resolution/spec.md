---
slug: pose-extension-catalog-resolution
status: in-progress
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-extension-catalog-lifecycle
priority: 4
components: pose-mcp
delivers: capability:extension-catalog-resolution
---

# Spec: pose-extension-catalog-resolution

> Single POSE spec template. Fill the relevant sections; remove the ones that
> don't apply. Keep the order: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Lifecycle:** update `status` as you go (`draft` → `in-progress` → `done`).
> On completion, run the closeout flow (skill `pose-spec-closeout`): set
> `status: done`, fill `completed_at` and disposition every follow-up.

---

## 1. Intent

### Goal
Give `pose extension install` a way to resolve an extension ID
(`pose-rule-backend-go`, `pose-rule-frontend-react`, ...) to a real,
signature-verified package through a defined catalog source, instead of
requiring the operator to already have a local directory containing it.

### Business value
`pose-adaptive-rule-delivery` (v1.3.0) made `pose doctor` recommend the
right extension for a detected stack, but the recommendation stops at
"install `pose-rule-backend-go`" — the operator still has to source the
package by hand, because `cmdExtensionInstall`
(`pose-mcp/internal/cli/extension.go`) only accepts a local directory path.
This is the deferred scope `pose-adaptive-rule-delivery`'s Final Report
named explicitly: "needed before the doctor advisory can become a single
runnable command." It also closes a gap between promise and implementation:
`pose-extension-catalog-lifecycle` (2026-07-19, `status: done`) already
specifies "R3: A signed catalog shall support discovery" — read against
today's `cmdExtensionInstall`, discovery of *new* packages by ID was never
actually built; what exists is listing/verifying an already-installed
catalog. Completing R3 as originally intended is this spec's throughline,
not a new promise.

### Constraints
- Must not weaken the existing trust model: `pose extension install`
  already verifies cosign/Sigstore signatures on the resolved package
  before applying it — resolution adds a lookup step in front of that
  verification, it does not replace or bypass it.
- The extension whitelist restriction (installs confined to
  `.agents/skills/`, `.pose/workflows/`, `.pose/rules/`, `.pose/templates/`)
  is unchanged by this spec.
- Requires deciding a concrete catalog source (a versioned index published
  alongside POSE releases, a GitHub-hosted registry, or something else) —
  this is a real trust-boundary decision and belongs in this spec's
  Decisions section (and possibly its own ADR) before implementation, not
  assumed here.

### Non-goals
- A public, unmoderated extension marketplace —
  `pose-extension-catalog-lifecycle`'s own Non-goals already rule this out
  and this spec inherits that boundary.
- Executing installer scripts as part of resolution — resolution only
  locates and hands off a package; installation semantics (dry-run,
  transactional apply, conflict handling) are unchanged from what
  `cmdExtensionInstall` already does for a local directory today.

---

## 2. Requirements

> Definition of Ready (entry gate): before `status: in-progress`, functional
> requirements must have **acceptance criteria with stable IDs** (`- R<N>: ...`).
> Published IDs are never renumbered; a removed criterion is marked as
> withdrawn. Verify with `pose lint-spec <slug> --ready-check`.
>
> Optional EARS form: `- R1: When <trigger>, the <system> shall <behavior>.`
> Verify an opted-in spec with `pose lint-spec <slug> --ears`.

### Functional
- R1: `pose extension install <arg>` shall treat `<arg>` as a local package
  directory when one exists at that path, unchanged from today; otherwise
  it shall treat `<arg>` as an extension ID and resolve it through the
  catalog (Decision 1).
- R2: catalog resolution shall find, for a given ID, the matching
  `<id>.tar.gz` and `<id>.sigstore.json` assets in the latest published
  GitHub release of this project, download both, and safely extract the
  tarball into a fresh local directory.
- R3: the extracted package shall then go through the exact same
  `loadExtensionManifest` → `verifyExtensionSignature` → plan → conflicts
  → apply pipeline a local directory install already uses — no duplicated
  or parallel verification logic.
- R4: the extracted manifest's declared `id` shall be checked against the
  requested ID; a mismatch shall fail before any file is written (defense
  in depth, not the trust boundary — R3's signature check is).
- R5: an extension ID with no matching assets in the latest release shall
  fail with a clear, actionable error, not a generic HTTP/parse error.
- R6: a local-directory install (the existing, pre-this-spec usage) shall
  make zero network calls — verified by pointing the catalog base URL at
  an address nothing listens on and confirming the install still succeeds.

### Non-functional
- Tarball extraction shall reject path-traversal entries and symlinks/
  hardlinks — the package format's "symlink-free directory" trust model,
  actively enforced here because this content crosses a network boundary
  instead of already sitting reviewed on disk (unlike a local package
  directory, which the operator chose to point at).

### Security
- Resolution does not change what `verifyExtensionSignature` verifies
  (`extension.json`, against the manifest's own declared
  signer/issuer) or which target paths a package may write
  (`extensionWhitelist`/`m.Permissions`) — R3.
- No installer script is ever executed; resolution only downloads,
  extracts and hands off data files (Non-goals, inherited from
  `pose-extension-catalog-lifecycle`).

### Compatibility
- `pose extension install <existing-local-directory>` behaves identically
  to before this spec — R1/R6.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/extension_catalog.go` (new): resolution, download,
  safe extraction.
- `pose-mcp/internal/cli/extension.go` (`cmdExtensionInstall`): ID-vs-
  directory disambiguation, manifest-ID consistency check.

### Artifacts
- created: pose-mcp/internal/cli/extension_catalog.go
- created: pose-mcp/internal/cli/extension_catalog_test.go
- modified: pose-mcp/internal/cli/extension.go

### Delivery targets
- capability:extension-catalog-resolution module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `pose extension install`'s first positional argument now accepts either
  a local directory (unchanged) or an extension ID — additive, existing
  directory-based invocations are unaffected (R1/R6).

### Technical risks
- Medium (the reason this spec was sequenced last in the roadmap):
  introduces the extension mechanism's first network-facing code path.
  Mitigated by: reusing the entire existing verification pipeline
  unchanged (R3); active path-traversal/symlink rejection on extraction
  (Non-functional); a real end-to-end run against the actual published
  `pose-rule-kubernetes` asset with genuine `cosign` signature
  verification during implementation, not only mocked tests (see
  Validation).
- Low: only the latest release is consulted (Decision 1's accepted scope
  limit) — an extension published in an older release and since removed
  from the latest one's assets becomes unresolvable by ID; a local
  directory install remains available as a fallback.

---

## 4. Tasks

### Planning
- [x] Investigated the release workflow and confirmed exactly one
      extension (`pose-rule-kubernetes`) is currently signed/published as
      a release asset — informed Decision 1's scope and Decision 1 in
      `pose-rule-extension-locale-parity`'s note that the other two
      extensions are not yet catalog-resolvable
- [x] Confirmed `verifyExtensionSignature` only ever verifies
      `extension.json` (not `files/` content) — resolution's job is purely
      to produce a local directory the existing pipeline already trusts
      correctly, no new verification logic needed (R3)

### Implementation
- [x] `extension_catalog.go`: `resolveCatalogAssets`, `downloadCatalogAsset`,
      `extractExtensionTarball` (path-traversal/symlink rejection),
      `fetchCatalogPackage` (R2, Non-functional)
- [x] `cmdExtensionInstall`: directory-vs-ID disambiguation via `isDir`,
      manifest-ID consistency check (R1, R4, Decision 2)
- [x] Verified the local-directory path makes zero network calls by
      pointing `catalogAPIBase` at an unreachable address during that test
      (R6)

### Validation
- [x] Unit tests: safe extraction (valid package, path traversal
      rejected, symlink rejected), asset resolution (found/not-found),
      full fetch (download + extract + sigstore sibling written) — all
      against a local `httptest` server, no real network dependency
- [x] Integration tests: full `pose extension install <id>` end to end
      against a fake catalog server; manifest-ID mismatch rejected before
      any write; local-directory install proven unaffected
- [x] **Real end-to-end run** (not mocked): built the dev binary, ran
      `pose extension install pose-rule-kubernetes --dry-run` against the
      actual live GitHub API and the real v1.3.0 release — resolved,
      planned correctly. Then ran the full install *without*
      `--allow-unsigned`, with the real `cosign` binary present: genuine
      Sigstore signature verification passed against the real published
      bundle, `pose extension list` reported `signed=true`. This is the
      strongest evidence available that the mechanism works against the
      actual trust chain, not just its test double.
- [x] `go test ./...`, `go vet ./...`, `gofmt -l .`: all clean

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision 1
- Date: 2026-08-15
- Context: needed a concrete catalog source before writing any code
  (Constraints). The candidates: (a) a new hosted registry/index service;
  (b) a versioned index file published alongside POSE releases, fetched
  and parsed separately from the extension itself; (c) this project's own
  GitHub Releases, which already publish exactly one extension
  (`pose-rule-kubernetes`) as a signed `<id>.tar.gz` +
  `<id>.sigstore.json` asset pair (`.github/workflows/release.yml`,
  `pose-extension-reference-publication`) — an already-existing,
  already-trusted distribution channel, just never resolved by ID.
- Options considered: (a) new registry service — real infrastructure to
  build, host and secure, with no operator demand established yet; (b) a
  separate index file — adds an extra artifact and a second thing that can
  drift from what a release actually shipped; (c) resolve directly against
  GitHub Releases' own asset list for the latest tag.
- Decision: (c).
- Rationale: (c) needs zero new infrastructure, zero new publishing steps,
  and cannot drift from what a release actually contains because the
  asset list *is* the release. It also directly completes
  `pose-extension-catalog-lifecycle`'s R3 ("a signed catalog shall support
  discovery") using the one real signed artifact this project has ever
  published, rather than inventing a second distribution mechanism nobody
  has exercised. (a) and (b) both remain available as later evolutions if
  a real multi-extension registry need emerges — this decision does not
  foreclose them, it just doesn't build them speculatively.
- Consequences: resolution only ever sees what the *latest* release
  published (R2's scope limit) — no history search across older releases.
  It also means an ID only resolves once that ID has actually been
  published this way; `pose-rule-backend-go`/`pose-rule-frontend-react`
  (not yet signed/published — see `pose-rule-extension-locale-parity`'s
  Decision 1) are not yet catalog-resolvable by this mechanism, only
  `pose-rule-kubernetes` is, today. The mechanism itself is general — the
  moment another extension is published the same way, it becomes
  resolvable with no further code change.

### Decision 2
- Date: 2026-08-15
- Context: R4 (manifest-ID consistency) is a defense-in-depth check, not a
  trust boundary — `verifyExtensionSignature` already is the trust
  boundary, unconditionally, regardless of R4.
- Options considered: (a) skip the ID-match check, rely on signature
  verification alone; (b) add the cheap check anyway.
- Decision: (b).
- Rationale: a mismatched or misconfigured published asset (wrong file
  under the wrong name, a build-system mistake) is a realistic failure
  mode signature verification alone would not catch early or explain
  clearly — R4 turns a confusing downstream symptom into an immediate,
  legible error, at near-zero implementation cost.
- Consequences: none — purely additive safety, no behavior trade-off.

---

## 6. Validation

### Strategy
`httptest`-backed unit and integration tests for every case that does not
need real infrastructure (the large majority — resolution, extraction
safety, mismatch rejection, no-network-for-local-installs), plus one
genuine, non-mocked end-to-end run against the real GitHub API and the
real published `pose-rule-kubernetes` asset with real `cosign` signature
verification, since a mechanism whose entire point is crossing a network
and trust boundary is not fully validated by mocks alone.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...`
- Scope: whole module
- Expected: PASS

#### Lint
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Scope: whole module
- Expected: clean

#### Build
- Command: `go -C pose-mcp build -trimpath -o ./pose ./cmd/pose`
- Scope: `cmd/pose`
- Expected: builds; used for the manual end-to-end run

#### Security / Contract
- Manual: `pose extension install pose-rule-kubernetes` (no
  `--allow-unsigned`) against the real v1.3.0 release, with a real
  `cosign` binary present.
- Expected: `Result: SUCCESS`, `pose extension list` reports
  `signed=true`.

### Execution log
- Date: 2026-08-15
- Environment: local (Linux, Go toolchain per `go.mod`, real `cosign`
  binary present, real network access to `api.github.com` and
  `github.com` release asset downloads confirmed available in this
  session).
- Notes: manual end-to-end run performed in a throwaway `/tmp` directory,
  removed after the run — no stray state left in this repository.

### Results summary
- Successes: `go test ./...`, `go vet ./...`, `gofmt -l .` all clean;
  eight new tests (extraction safety ×3, resolution ×1, fetch ×1,
  end-to-end ×2, no-network-regression ×1) all pass; real end-to-end dry-
  run and full signed install against the actual published extension both
  succeeded.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] test:TestExtensionInstallByIDResolvesFromCatalogEndToEnd test:TestExtensionInstallLocalDirectoryPathNeverTouchesCatalog
- R2 [satisfied] test:TestFetchCatalogPackageDownloadsExtractsAndWritesSigstoreSibling manual end-to-end run against the real v1.3.0 release, 2026-08-15
- R3 [satisfied] manual end-to-end run, 2026-08-15 (real cosign verification, signed=true)
- R4 [satisfied] test:TestExtensionInstallByIDReportsMismatchedManifestID
- R5 [satisfied] test:TestResolveCatalogAssetsFindsMatchingAssets
- R6 [satisfied] test:TestExtensionInstallLocalDirectoryPathNeverTouchesCatalog

### Known gaps
- Only the latest release is searched (Decision 1) — an extension
  published in an older release and dropped from the latest one's assets
  is not resolvable by ID until republished; a local directory install
  remains available.
- `pose-rule-backend-go`/`pose-rule-frontend-react` are not yet
  catalog-resolvable in practice (they are not signed/published as
  release assets today) — the mechanism is ready for them the moment they
  are, but this spec does not itself change the release workflow to
  publish them.

---

## 7. Final Report

### Delivered scope
`pose extension install <id>` now resolves an extension ID against the
latest published GitHub release's assets, downloads and safely extracts
the signed tarball, and feeds it through the exact same verified-install
pipeline a local directory always used — completing
`pose-extension-catalog-lifecycle`'s R3 as originally intended and closing
the gap `pose-adaptive-rule-delivery` named: the doctor's rule-extension
recommendation can now become a single runnable command, for any extension
actually published this way. Verified end to end against the real,
currently-published `pose-rule-kubernetes` extension with genuine `cosign`
signature verification, not only mocked tests. A new registry service and
a separate published index were both considered and deliberately not
built (Decision 1) in favor of resolving directly against this project's
own already-existing, already-signed release assets.

### Files and modules changed
- `pose-mcp/internal/cli/extension_catalog.go` (new): resolution,
  download, safe tar.gz extraction.
- `pose-mcp/internal/cli/extension_catalog_test.go` (new): tests.
- `pose-mcp/internal/cli/extension.go`: ID-vs-directory disambiguation in
  `cmdExtensionInstall`, manifest-ID consistency check.

### Validation executed
- Command: `go -C pose-mcp test ./...`
- Result: SUCCESS
- Manual: `pose extension install pose-rule-kubernetes --dry-run` then
  full install (no `--allow-unsigned`) against the real v1.3.0 release
- Result: real signature verification passed (`signed=true`)

### Residual risks
- None beyond the accepted Known gaps (latest-release-only scope;
  backend-go/frontend-react not yet published this way).

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [open] extend the release workflow to sign and publish
  `pose-rule-backend-go`/`pose-rule-frontend-react` the same way
  `pose-rule-kubernetes` already is, so they become genuinely
  catalog-resolvable rather than only local-directory-installable — this
  is a `pose-extension-reference-publication`-shaped change, not part of
  this spec's own scope.
- [open] search across release history (not only the latest tag) once an
  extension actually needs to stay resolvable after falling out of the
  latest release's asset list (Decision 1's accepted scope limit) —
  premature without a real case demanding it.
