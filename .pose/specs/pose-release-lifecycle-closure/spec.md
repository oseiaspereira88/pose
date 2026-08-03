---
slug: pose-release-lifecycle-closure
status: in-progress
created_at: 2026-08-02
completed_at:
supersedes:
depends_on: pose-hierarchical-review-closeout, pose-version-contract, pose-release-compatibility-matrix
priority: 3
components: pose-mcp, cli, changelogs, release, mcp, workflows, scaffold
delivers: surface:release-lifecycle-cli, contract:release-lifecycle-mcp, governance:release-integrity
---

# Spec: Release lifecycle closure

## 1. Intent

### Goal
Turn unreleased changelog fragments into immutable, versioned release snapshots
and track each release from preparation through publication and independent
verification with falsifiable evidence.

### Business value
POSE currently governs creation of one changelog fragment per delivered spec,
but does not govern release completion. `pose release-notes --version` reads the
mutable `unreleased` directory and uses the version only as a heading. The tag
workflow publishes those notes without moving the fragments or recording a
release transition. The repository has tags through `v0.15.0`, but only one
manually archived fragment directory (`v0.9.0`) and no release ledger for
`v0.10.0` through `v0.15.0`; the changelog reader additionally expects
consolidated `.md` files and ignores that real directory. The result is a system
that can publish binaries while continuing to describe its delivery state as
unreleased. A closed lifecycle makes release history, current pending changes
and publication confidence mechanically distinguishable.

### Constraints
- Preserve unreleased fragments as a queue of candidate user-facing changes,
  never as evidence that a release exists.
- Distinguish a prepared snapshot, a Git tag, an externally published release
  and an independently verified release.
- Keep preparation, status projection and offline gates provider-neutral; accept
  provider evidence through a typed adapter/import contract.
- Keep release manifests and state transitions append-only or immutable after
  publication.
- Derive notes from the immutable release snapshot, never from the live
  unreleased queue during tagged publication.
- Reuse the authoritative project-version, compatibility, signing, SBOM,
  provenance and independent-verification contracts already implemented.
- Keep Git mutation explicit and reviewable; never stage unrelated files,
  rewrite an existing tag or force-push as an implicit release step.
- Support projects whose language-specific version update remains outside the
  POSE engine through declarative version evidence rather than raw commands.
- Preserve offline determinism for all local gates.

### Non-goals
- Replace GoReleaser, package registries, GitHub Releases, signing, SBOM or SLSA
  provenance generation.
- Infer that a tag was published successfully.
- Choose or apply a SemVer bump without an explicit target and project policy.
- Fabricate historical release notes or fragment ownership during backfill.
- Push tags, branches or provider releases from the read-only MCP surface.
- Guarantee that every external package channel becomes available at the same
  time as the primary release.

## 2. Requirements

### Functional
- R1: POSE shall model release states `prepared`, `tagged`, `published`,
  `verified`, `failed` and `yanked`, with only evidence-backed forward
  transitions and explicit terminal or recovery semantics.
- R2: `pose release plan --version` shall inspect unreleased fragments,
  authoritative version evidence, previous releases and policy, then emit a
  deterministic target summary, SemVer recommendation and blockers without
  mutating the repository.
- R3: `pose release prepare --version` shall default to dry-run and, with an
  explicit apply flag, atomically create an immutable release manifest, archive
  exactly the selected fragments and generate canonical notes.
- R4: A release manifest shall record schema version, target/previous version,
  preparation time, selected specs, fragment categories and digests, breaking
  status, notes digest, release-input digest, policy digest and declared version
  evidence.
- R5: Archived fragments shall live under
  `.pose/changelogs/version/` and canonical consolidated notes under
  `.pose/changelogs/version.md`; generation shall be byte-stable and preserve
  fragment-to-spec provenance.
- R6: Preparation shall reject malformed, duplicated, empty or already released
  fragments, missing specs, invalid categories, incompatible target versions and
  an empty selection unless policy explicitly allows an empty release.
- R7: A fragment shall exist in exactly one lifecycle location: unreleased or one
  released version; strict checks shall reject duplication, disappearance from a
  prepared manifest and assignment to multiple versions.
- R8: `pose release-notes --version` shall read the immutable prepared snapshot
  for that version; unreleased preview shall require an explicit preview mode and
  shall never be accepted by a tagged publication gate.
- R9: Release preparation shall bind notes and fragments to a release-input
  digest; any post-preparation change shall make the candidate stale and require
  a new preparation rather than silently changing published notes.
- R10: `pose release check --version --strict` shall reconcile manifest,
  archived fragments, consolidated notes, version evidence, compatibility
  matrix, Git tag/commit when present and lifecycle events, explaining every
  mismatch.
- R11: The presence of a matching Git tag shall advance only the derived `tagged`
  fact; POSE shall require typed provider evidence before projecting
  `published`.
- R12: Publication evidence shall identify provider, repository/package,
  version/tag, immutable source commit, publication time, canonical URL,
  workflow/run identity when applicable and released asset names plus digests.
- R13: `pose release record` shall validate and append `tagged`, `published`,
  `verified`, `failed` or `yanked` events idempotently from confined evidence;
  conflicting transitions shall fail instead of overwriting history.
- R14: Verification evidence shall bind to the publication evidence and asset
  digests it verified; a newer or different publication shall invalidate the
  previous verified projection.
- R15: Provider adapters may create publication evidence, but the core shall
  consume one provider-neutral schema and remain fully usable offline after the
  evidence is imported.
- R16: `pose release status` shall distinguish unreleased fragments, prepared
  candidates, tagged-but-unpublished releases, published-unverified releases,
  verified releases, failures and yanked releases in human and JSON output.
- R17: The read-only MCP changelog/release projection shall expose pending
  fragments, immutable release history, lifecycle confidence, evidence refs and
  gaps from the same model; directory-based releases shall no longer be
  invisible.
- R18: `pose check --strict` shall enforce the adopted release policy: every done
  spec has `changelog: none` or one resolvable fragment, every governed tag has a
  release manifest, every published event resolves to prepared content, and no
  release claims a stronger state than its evidence.
- R19: Missing or invalid changelog/release policy shall produce an actionable
  finding after adoption; the gate shall not silently return and disable
  coverage enforcement.
- R20: The tag workflow shall validate a prepared candidate and generate notes
  from its version snapshot before publishing; it shall emit publication
  evidence as a retained release asset for later local reconciliation.
- R21: The local release script shall refuse dirty or unprepared candidates,
  stage no unrelated files, refuse existing tags, avoid force-push and stop with
  an actionable recovery state when provider publication fails.
- R22: After verified publication, `pose release open-next` shall create a
  non-mutating plan for the next development cycle and validate the chosen next
  version against project policy; project-specific version-file changes remain
  explicit reviewed work.
- R23: `pose release backfill --from-git` shall default to dry-run, inventory
  historical tags/changelog paths/provider evidence, assign confidence and
  report missing notes or ambiguous fragment attribution without fabricating
  released state.
- R24: The backfill flow shall detect the current repository mismatch: archived
  `v0.9.0` fragments, later tags without local manifests and any publication
  whose provider evidence has not been imported.
- R25: Re-running plan, prepare, check, record, status or backfill with identical
  inputs shall be idempotent and produce stable ordering and digests.

### Non-functional
- Keep release preparation linear in selected fragments and release checking
  bounded by governed release records and configured Git tags.
- Produce byte-stable notes, manifests, events and JSON projections for identical
  inputs.
- Show the smallest remediation for every mismatch, including stale snapshot,
  missing policy, tag/version drift and missing publication evidence.
- Preserve a complete audit path from spec to fragment to release manifest to
  tag to published assets to verification evidence.
- Keep provider/network retrieval exploratory; deterministic gates shall consume
  versioned or explicitly supplied evidence.

### Security
- Validate SemVer/tag values and confine every manifest, fragment, notes and
  evidence path to the project root.
- Invoke Git with structured arguments, reject revision option injection and
  never execute commands embedded in release metadata.
- Reject mutable branch refs as publication identity; bind evidence to immutable
  commit and asset digests.
- Verify provider evidence identity and integrity before accepting publication
  or verification transitions.
- Prohibit credentials, tokens and provider response bodies in release records;
  store only minimized identity and evidence fields.
- Remove force-tagging, force-pushing and broad `git add .` behavior from the
  release path.

### Compatibility
- Preserve existing `pose release-notes` and `pose_get_changelog` as additive
  compatibility aliases/views while routing them through the new model.
- Recognize both historical `.pose/changelogs/version.md` files and the existing
  `.pose/changelogs/version/` fragment directories during migration.
- Warn rather than fail for pre-adoption tags until backfill is reviewed.
- Keep existing signing, compatibility, package-manifest and verification
  workflows as downstream gates consuming the prepared release identity.
- Preserve project-specific version authority contracts through declarative
  evidence and named validation checks.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: release manifest/event model, changelog reader,
  lifecycle projection, backfill findings and MCP-facing types.
- `pose-mcp/internal/cli`: `pose release` command family, release notes,
  changelog/release structural checks and safe Git reconciliation.
- `pose-mcp/internal/mcpserver`: project-scoped read projection and catalog
  golden contract.
- `.github/workflows/release.yml`, `verify-release.yml`, `scripts/release.sh`,
  release tests and current version/compatibility gates.
- `.pose/templates`, policy, workflow/rule/skill guidance, POSE manual, docs,
  locales and embedded scaffold.

### Review findings driving the design
- High: no command or state transition consumes unreleased fragments at release
  cut; the publication workflow reads a mutable queue.
- High: Git tags and external releases are not represented as distinct lifecycle
  facts, so POSE cannot say whether a candidate was published or verified.
- High: `scripts/release.sh` stages the entire worktree and force-updates the tag
  and remote tag, risking unintended inclusion and history replacement.
- Medium: the changelog reader recognizes consolidated root `.md` releases,
  while the only real archive is a version directory; even that cut is invisible
  through the current read model.
- Medium: absent `.pose/policy/changelog.json` makes the current changelog check
  return silently, disabling post-adoption coverage enforcement.
- Medium: release comments reference nonexistent `pose-release-pipeline` and
  `conductor-release-cut` contracts, leaving ownership implied rather than
  executable.

### API/contract changes
- Add a cohesive CLI namespace while preserving compatibility aliases:

  ```text
  pose release plan --version vX.Y.Z
  pose release prepare --version vX.Y.Z --apply
  pose release check --version vX.Y.Z --strict
  pose release notes --version vX.Y.Z
  pose release record --version vX.Y.Z --event published --evidence path
  pose release status [--version vX.Y.Z] [--json]
  pose release open-next --version vX.Y.Z
  pose release backfill --from-git [--apply]
  ```

- Add versioned `.pose/release-policy.json` with adoption date, version scheme,
  tag pattern, empty-release policy, required publication/verification states,
  provider mode and legacy cutoff; allow no raw commands.
- Keep publication itself in provider workflows; the POSE core prepares,
  validates and records evidence.

### Data/storage changes
- Store immutable candidate data in
  `.pose/releases/version/manifest.json` and canonical notes in
  `.pose/changelogs/version.md`.
- Move selected source fragments to `.pose/changelogs/version/spec.md` so
  user-facing provenance remains inspectable after consolidation.
- Store append-only transition records in
  `.pose/releases/version/events.jsonl`; derive current state by replay rather
  than mutating manifest status.
- Generate a release index with pending counts, version/state/evidence summary
  and gaps; the later delivery-integrity graph may ingest release nodes without
  replacing this canonical ledger.

### Artifacts
- modified: .agents/skills/README.md
- created: .agents/skills/pose-release-closeout/SKILL.md
- modified: .github/workflows/release.yml
- created: .pose/adr/2026-08-03-immutable-release-ledger.md
- modified: .pose/indexes/task-map.json
- modified: .pose/indexes/validation-matrix.json
- created: .pose/release-policy.json
- created: .pose/policy/changelog.json
- created: .pose/rules/release-integrity.md
- modified: .pose/specs/pose-release-lifecycle-closure/spec.md
- created: .pose/workflows/release.md
- modified: POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- created: locales/pt-BR/.agents/skills/pose-release-closeout/SKILL.md
- created: locales/pt-BR/.pose/rules/release-integrity.md
- created: locales/pt-BR/.pose/workflows/release.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/native_only_test.go
- created: pose-mcp/internal/cli/release_lifecycle.go
- created: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: pose-mcp/internal/cli/skills_check_test.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- created: pose-mcp/internal/mcpserver/release_status_tool_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/changelogs.go
- created: pose-mcp/internal/pose/release_lifecycle.go
- created: pose-mcp/internal/pose/release_lifecycle_test.go
- modified: pose-mcp/internal/scaffold/scaffold.go
- modified: scripts/release.sh
- created: .pose/indexes/releases.json

### Delivery targets
- surface:release-lifecycle-cli module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- contract:release-lifecycle-mcp module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go
- governance:release-integrity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### Release state machine
1. `plan`: inspect target, fragments, policy and prior release without mutation.
2. `prepare`: freeze fragments, notes and release-input digest atomically.
3. `tagged`: reconcile an immutable Git tag with the prepared candidate.
4. `published`: import provider evidence bound to tag, commit and asset digests.
5. `verified`: import independent verification bound to published assets.
6. `failed` or `yanked`: append explicit evidence and remediation; never delete
   the prior history.
7. `open-next`: plan the next development version only after policy's terminal
   publication state is satisfied.

### Technical risks
- Publication occurs after the tagged commit and cannot mutate that snapshot;
  provider evidence must be retained externally and reconciled into main without
  pretending the tag already contained it.
- Backfill across existing tags can overstate history; confidence and missing
  evidence must remain explicit.
- Moving fragments creates large rename diffs; preparation must be atomic and
  dry-run-first.
- Multiple products in a monorepo may share tags selectively; policy and
  fragment filters need a stable release identity rather than basename guesses.
- Generated notes can drift from provider-edited notes; provider evidence should
  record the published notes digest and surface any manual divergence.

### Rollout
1. Add read-only status, dual-format changelog discovery and backfill dry-run;
  report current historical gaps without changing release gates.
2. Add plan/prepare and require version snapshots for new release candidates.
3. Make tagged publication consume prepared notes and emit evidence.
4. Require publication and verification evidence according to project policy.
5. Backfill legacy tags with reviewed confidence, then enable strict no-tag-
  without-manifest enforcement after the configured cutoff.

## 4. Tasks

### Planning
- [ ] Record an ADR for immutable release snapshots, append-only lifecycle
  events, tag-versus-publication semantics and provider-evidence reconciliation.
- [ ] Capture the repository's tags, `v0.9.0` fragment directory, missing later
  manifests and current unreleased fragment as a migration fixture.
- [ ] Freeze release policy, manifest, event, provider evidence and status JSON
  schemas with golden files.
- [ ] Define how publication evidence returns to the default branch without a
  privileged direct push from the tagged workflow.

### Implementation
- [ ] Implement dual-format changelog discovery and fix released-version MCP
  projection.
- [ ] Implement release policy parsing and non-silent adoption findings.
- [ ] Implement deterministic plan, prepare, notes and input-digest generation.
- [ ] Implement atomic fragment archival with rollback and idempotency.
- [ ] Implement append-only release events and legal transition validation.
- [ ] Implement Git tag reconciliation without force operations.
- [ ] Implement provider-neutral publication/verification evidence import.
- [ ] Implement status, check, open-next and dry-run backfill commands.
- [ ] Integrate release lifecycle into `pose check --strict` and hierarchical
  closeout/review evidence.
- [ ] Update the release workflow to consume prepared notes and retain evidence.
- [ ] Replace broad staging/force-tag behavior in `scripts/release.sh` with
  fail-closed preparation validation.
- [ ] Add project-scoped MCP release state, CLI/MCP golden parity and docs.
- [ ] Update templates, release workflow/skill, POSE manual, docs, locales,
  changelog and embedded scaffold.

### Validation
- [ ] Run focused changelog, manifest, lifecycle, Git fixture, backfill and MCP
  tests.
- [ ] Run negative tests for traversal, option injection, duplicate fragments,
  stale candidates, conflicting events, tag reuse and forged provider evidence.
- [ ] Run a full fixture from unreleased specs through prepared, tagged,
  published and verified, then prove the next release starts with an empty
  pending queue and prior immutable history.
- [ ] Run failure fixtures for provider publication failure and yanked release.
- [ ] Run the full Go suite, release shell syntax/security contract and embedded
  distribution parity.
- [ ] Run strict POSE structure, spec lint and module validation gates.

## 5. Decisions

### Decision 1: A tag is not a published release
- Date: 2026-08-02
- Context: The current tag-triggered workflow can fail after the tag exists, but
  POSE has no state capable of representing that difference.
- Options considered: treat tags as released; query the provider live; store
  separate prepared/tagged/published/verified evidence.
- Decision: Model each state separately and require imported provider evidence
  for publication.
- Rationale: External success becomes falsifiable without making network access
  a deterministic gate.
- Consequences: Local state may honestly report `tagged` until publication
  evidence is reconciled.

### Decision 2: Freeze release inputs before tagging
- Date: 2026-08-02
- Context: Rendering tagged notes from `unreleased` leaves the release content
  mutable and never consumes the pending queue.
- Options considered: continue rendering live; copy fragments after publication;
  create an immutable prepared snapshot before tagging.
- Decision: Prepare and commit the release snapshot before tag creation; tagged
  CI consumes only that snapshot.
- Rationale: Notes, fragments and version identity become one reviewable
  candidate.
- Consequences: Release preparation is an explicit local step and stale
  candidates must be regenerated.

### Decision 3: Keep fragment provenance and consolidated notes
- Date: 2026-08-02
- Context: A single notes file is easy to publish but loses per-spec source;
  directories preserve provenance but the current reader ignores them.
- Options considered: consolidated file only; fragment directory only; canonical
  notes plus archived source fragments.
- Decision: Store both, tied by manifest digests.
- Rationale: Humans get readable notes and auditors retain spec-level origin.
- Consequences: Strict checks must prevent divergence between the two forms.

## 6. Validation

### Strategy
Use a real temporary Git repository and a fake provider-evidence fixture to
exercise every legal transition and failure mode. The main acceptance case must
prove that releasing consumes only selected fragments, tagged notes remain
immutable after new work enters `unreleased`, provider failure does not become
`published`, verification binds exact assets and the next cycle starts without
relabeling prior work as pending.

### Deterministic checks
- Test: `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'Changelog|ReleaseLifecycle|ReleasePrepare|ReleaseEvidence|ReleaseBackfill' -count=1`.
- Full suite: `go -C pose-mcp test ./... -count=1`.
- Release scripts: `bash -n scripts/release.sh tests/release/verify.sh tests/release/independent-verify.sh`.
- Scaffold parity: `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1`.
- Structure: `pose check --strict`.
- Spec readiness: `pose lint-spec pose-release-lifecycle-closure --ready-check`.
- Spec lint: `pose lint-spec pose-release-lifecycle-closure --strict`.
- Delivery gate: `pose validate --strict --module pose-mcp --report`.

### Execution log
- 2026-08-02, planning: changelog reader/checks, release-notes command, tag
  workflow, local release script, version/compatibility contracts, tags and
  archived fragments inspected.
- 2026-08-02, readiness: `pose lint-spec
  pose-release-lifecycle-closure --ready-check` and `--strict` passed; strict
  lint also passed for the other three delivery-integrity specs.
- 2026-08-02, repository validation: `pose validate --strict --module pose-mcp`,
  the embedded-scaffold parity test and release-script shell syntax passed.
- 2026-08-02, structure: `pose check --strict` reached the repository-wide
  pre-existing broken `.pose/feedback` reference in `POSE.md`; this planning
  change neither introduces nor resolves that unrelated baseline error.

### Results summary
- Successes: The review identifies the missing lifecycle boundary, defines a
  provider-neutral, evidence-backed release closure plan, passes spec readiness
  and strict lint, passes strict module validation and preserves scaffold
  parity.
- Failures: The repository-wide structure gate remains red on the pre-existing
  `POSE.md` reference to absent `.pose/feedback`; no new spec or scaffold
  failure was observed.
- Warnings: Current project state is stale by age and reports a hand-edited
  Architecture section; refresh is outside this planning-only change.

### Requirement trace
Requirement trace will be populated only after implementation evidence exists;
this draft does not claim any requirement as satisfied.

### Known gaps
- The evidence-return path from GitHub publication to the default branch needs an
  ADR decision and least-privilege implementation.
- Historical provider publication evidence for existing tags has not been
  imported; backfill must not infer it from tag presence.

## 7. Final Report

### Delivered scope
Planning artifact only: this draft defines complete release lifecycle closure as
a future POSE mechanism. No current changelog, tag or published release state is
rewritten or claimed.

### Files and modules changed
- `.pose/specs/pose-release-lifecycle-closure/spec.md`.
- `.pose/roadmaps/delivery-integrity.md` membership, ordering and cut criteria.

### Validation executed
- Passed: release-contract inspection; `pose assess discover --component
  pose-mcp`; spec readiness and strict lint for all four roadmap specs; `pose
  validate --strict --module pose-mcp`; release-script shell syntax; embedded
  scaffold parity; and `git diff --check`.
- Known repository baseline failure: `pose check --strict` reports the tracked
  `POSE.md` reference to absent `.pose/feedback`.

### Residual risks
- Release truth spans Git and an external provider; completeness depends on a
  secure, reviewable evidence reconciliation path rather than local inference.

### Follow-ups
No follow-up is opened by this planning revision; implementation tasks remain
inside this draft spec.
