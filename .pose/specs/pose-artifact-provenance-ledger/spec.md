---
slug: pose-artifact-provenance-ledger
status: done
created_at: 2026-08-02
completed_at: 2026-08-03
supersedes:
depends_on: pose-hierarchical-review-closeout
priority: 1
components: pose-mcp, cli, indexes, reports, mcp, scaffold
---

# Spec: Artifact provenance ledger

## 1. Intent

### Goal
Turn each spec's artifact claim into a structured, Git-verifiable delivery
ledger and the factual foundation of POSE delivery-integrity checks.

### Business value
POSE can currently prove that declared checks passed, but cannot prove which
repository paths a spec changed or answer which specs contributed to a path.
Issue [#8](https://github.com/oseiaspereira88/pose/issues/8) measured the
result in a real repository: 17% of artifact declarations were not
machine-resolvable and 34% of tracked code had no spec claim despite every
completed spec filling the narrative section. A falsifiable ledger turns that
prose into reviewable provenance and supplies the artifact side of the combined
delivery graph required by issue #7.

### Constraints
- Keep the engine language- and framework-agnostic; inspect repository-relative
  paths and Git metadata, not source-code semantics.
- Keep the spec as the reviewed source of artifact claims and generated indexes
  as disposable projections.
- Separate author declarations from observed Git change sets; never treat one
  as proof of the other.
- Preserve offline and deterministic operation for identical repository state,
  policy and revision selectors.
- Confine every path and Git revision to the selected project root and reject
  option injection, traversal, symlink escape and unbounded filesystem scans.
- Stage enforcement for legacy specs and repositories with existing unclaimed
  files.
- Keep release-package provenance, SBOM and SLSA contracts independent; this
  spec governs source-tree delivery artifacts.

### Non-goals
- Decide whether an artifact is useful, reachable or user-visible; spec
  `pose-delivery-surface-assurance` owns that guarantee.
- Infer a perfect historical attribution for commits that have no spec or
  revision-range evidence.
- Assign permanent code ownership; the ledger records provenance while module
  metadata remains the operational ownership source.
- Execute arbitrary shell commands from specs or policy files.

## 2. Requirements

### Functional
- R1: A spec shall declare source-tree artifacts in a machine-parseable
  `### Artifacts` subsection using the closed actions `created`, `modified`,
  `renamed`, `removed` or the mutually exclusive `none` declaration with a
  reason.
- R2: Artifact paths shall be canonical, project-relative, exact tracked paths;
  strict mode shall reject absolute paths, traversal, ambiguous basenames,
  directories and globs, while `renamed` shall carry explicit old and new paths.
- R3: A `done` spec under the activated contract shall contain at least one
  artifact action or one `none` reason; absence, an empty section, mixed
  `none` plus actions, and duplicate contradictory actions shall fail closeout.
- R4: `pose artifact-check` shall compare declared actions with an explicit,
  immutable Git change set and emit stable findings for `resolvability`,
  `existence`, `action-mismatch`, `undeclared` and `orphan`.
- R5: A change set shall be selected either from commits carrying repeatable
  `POSE-Spec` trailers or from explicit base/head revisions recorded by a
  spec-linked POSE report; POSE shall never guess attribution from timestamps,
  authors, branches or path proximity.
- R6: Spec-linked report history shall persist the resolved base, head, commit
  IDs, normalized changed paths and diff digest, and those fields shall
  participate in the record's stable hash.
- R7: `pose index` shall generate a schema-versioned
  `.pose/indexes/delivery-integrity.json` containing spec, artifact and change-set
  nodes, typed provenance edges, reverse traversal and findings; the index shall
  be reproducible from specs, policy, history and Git.
- R8: Artifact policy shall define governed roots, exact exclusions and finding
  severities so generated, vendored, fixture and scaffold paths do not create an
  unbounded orphan baseline.
- R9: `pose artifact-backfill --from-git` shall produce a deterministic dry-run
  proposal with confidence and conflicts, shall never edit a spec without an
  explicit confirmation flag, and shall preserve ambiguous history as an
  unresolved finding.
- R10: CLI JSON and a project-scoped, read-only MCP projection shall expose the
  same ledger schema, reverse lookup and findings without leaking file contents
  or unrestricted absolute paths.

### Non-functional
- Produce byte-stable JSON ordering and finding IDs for the same inputs.
- Bound Git subprocess output and runtime through existing validation
  guardrails.
- Keep parsing linear in spec text plus the selected Git diff and indexing
  proportional to governed paths.
- Make every warning and error include spec, path, change-set selector and
  remediation.

### Security
- Invoke Git with structured arguments and `--` path separation where
  applicable; reject revision strings that begin with options.
- Resolve canonical paths under the authorized project root before reading
  metadata.
- Persist names and digests only; never persist source contents, credentials or
  unredacted command output in the ledger.
- Keep the MCP tool read-only and reuse the existing project-scope authorization
  contract.

### Compatibility
- Parse legacy narrative `Files and modules changed` sections unchanged and
  emit an advisory migration finding instead of fabricating structured claims.
- Introduce the section, history fields, check command and index additively in
  observability mode.
- Enable strict closeout only through policy opt-in until a repository schema
  bump makes the contract the default for newly created specs.
- Keep existing report-history records readable when change-set fields are
  absent.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: artifact grammar, policy, ledger model and reverse
  traversal.
- `pose-mcp/internal/cli`: `artifact-check`, `artifact-backfill`, lint-spec,
  report/history persistence, index generation and check integration.
- `pose-mcp/internal/mcpserver`: project-scoped read projection and catalog
  contract.
- `.pose/templates`, locales, workflows, rules, POSE manual and embedded
  scaffold mirrors.

### Consulted context
- `knowledge:contract-baseline-handoff` for current index, report, MCP and
  release contracts.
- `adr:2026-08-02-delivery-integrity-graph-and-git-observed-provenance` for the
  shared graph, witness separation and selector precedence.

### Artifacts
- modified: .pose/rules/delivery-evidence.md
- modified: .pose/specs/pose-artifact-provenance-ledger/spec.md
- modified: .pose/templates/spec.md
- modified: .pose/workflows/feature.md
- modified: POSE.md
- modified: docs-site/docs/mcp.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
- created: .pose/policy/artifacts.json
- created: .pose/indexes/delivery-integrity.json
- created: pose-mcp/internal/cli/artifact_integrity.go
- created: pose-mcp/internal/cli/artifact_integrity_test.go
- created: pose-mcp/internal/mcpserver/delivery_integrity_tool_test.go
- created: pose-mcp/internal/pose/delivery_integrity.go
- created: pose-mcp/internal/pose/delivery_integrity_test.go

### API/contract changes
- Add a strict artifact bullet grammar:

  ```text
  - created: path/to/file
  - modified: path/to/file
  - renamed: old/path -> new/path
  - removed: path/to/file
  - none: analysis
  ```

- Treat `none` reasons as a closed initial vocabulary: `analysis`, `decision`,
  `documentation`, `governance`, `spike`; policy may add project-local values
  without weakening the requirement for a reason.
- Add `pose artifact-check` selectors `--spec`, `--from`, `--to`, `--strict`
  and `--json`; repeated `POSE-Spec` trailers remain an equivalent selector,
  not a hidden fallback.
- Extend `pose report` history with a versioned `change_set` object and include
  it in stable comparison.
- Add a read-only MCP tool only after its JSON schema and CLI projection share
  golden fixtures.

### Data/storage changes
- Add artifact policy under `.pose/` as reviewed source data; keep generated
  orphan allowlists out of the index.
- Generate one combined `delivery-integrity.json` rather than independent
  `artifacts.json` and `surfaces.json` files that can drift. This spec seeds the
  artifact portion; `pose-delivery-surface-assurance` adds delivery and evidence
  nodes without changing provenance semantics.
- Represent each observed change set by selector, resolved commits, base/head,
  normalized path actions and SHA-256 diff digest.
- Represent each claim and observation as separate typed edges so a mismatch is
  a finding rather than overwritten data.

### Technical risks
- Squash or rewritten history can invalidate commit selectors; explicit ranges
  and a visible stale-evidence finding are required recovery paths.
- A broad base/head range can include another spec's work; strict closeout must
  require explicit attribution or disposition for every path instead of silently
  assigning it.
- Orphan checks can be noisy in mature repositories; governed roots and exact
  exclusions must ship before enforcement.
- Mechanical provenance can be mistaken for value assurance; command output and
  docs must link the follow-on surface check when relevant.

### Rollout
1. Accept and index declarations, persist change-set evidence and report all
   findings without changing existing exit codes.
2. Enable `resolvability`, `existence` and `action-mismatch` as opt-in required
   findings for new specs; keep `undeclared` and `orphan` policy-controlled.
3. Provide dry-run backfill and baseline metrics before any schema bump.
4. Require `### Artifacts` for newly scaffolded specs under the new schema;
   preserve advisory behavior for legacy completed specs.

## 4. Tasks

### Planning
- [x] Record one shared ADR for the delivery-integrity graph, declaration versus
  observation boundary, Git selector precedence and staged migration before
  either provenance or surface implementation becomes `in-progress`.
- [x] Turn issue #8's measured cases into checked-in positive and negative
  fixtures, including ambiguous basename, rename-with-edit, removed path,
  mixed-spec range, excluded fixture and legacy narrative.
- [x] Freeze artifact grammar, policy and ledger JSON schemas with golden files.

### Implementation
- [x] Implement the artifact parser, canonical path validator and typed findings
  in `internal/pose`.
- [x] Extend report/history with immutable change-set evidence and stable hashing.
- [x] Implement Git selector resolution and `pose artifact-check` without shell
  evaluation.
- [x] Add closeout lint rules and tolerant/strict migration behavior.
- [x] Extend `pose index` with the artifact portion of
  `delivery-integrity.json` and deterministic reverse traversal.
- [x] Implement dry-run-first backfill with explicit conflict output.
- [x] Add the project-scoped MCP projection, CLI/MCP golden parity and docs.
- [x] Update English and pt-BR templates, workflow/rule guidance, POSE manual,
  changelog and embedded scaffold.

### Validation
- [x] Run focused parser, Git fixture, lint, report-history, index and MCP tests.
- [x] Run negative tests for traversal, unsafe revisions, symlink escape,
  conflicting claims, malformed history and sensitive output.
- [x] Run the full Go suite and embedded-distribution parity test.
- [x] Run strict POSE structure, spec lint and module validation gates.

## 5. Decisions

### Decision 1: Use one delivery-integrity graph
- Date: 2026-08-02
- Context: Issues #7 and #8 need a deterministic cross-query, not two indexes
  whose relationship exists only in prose.
- Options considered: independent `artifacts.json` and `surfaces.json`; one
  artifact index plus an ad hoc join; one schema-versioned delivery graph.
- Decision: Generate `delivery-integrity.json` with typed nodes and edges, seeded
  here and extended by the dependent surface-assurance spec.
- Rationale: A single projection makes cross-findings contractual and prevents
  independent regeneration or version drift.
- Consequences: Both specs must share a schema ADR and golden fixtures; commands
  retain focused views while reading the same graph.

### Decision 2: Keep claims and observations separate
- Date: 2026-08-02
- Context: A declaration is reviewable intent; Git is evidence of repository
  state. Either one can be incomplete or wrong.
- Options considered: trust spec prose; infer everything from Git; store both
  and reconcile them.
- Decision: Persist both as separate edges and report mismatches.
- Rationale: Falsifiability requires an independent witness without erasing the
  author's declared scope.
- Consequences: Closeout has an explicit reconciliation cost and can surface
  honest ambiguity instead of manufacturing attribution.

## 6. Validation

### Strategy
Use temporary real Git repositories to test every action and selector, then
prove that lint, report history, index, CLI JSON and MCP expose identical facts.
The acceptance fixture shall include a spec with perfectly valid artifact
claims whose code is unreachable; artifact-check must pass it so the dependent
surface spec proves the complementary failure.

### Deterministic checks
- Test: `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'Artifact|ChangeSet|DeliveryIntegrity' -count=1`.
- Full suite: `go -C pose-mcp test ./... -count=1`.
- Scaffold parity: `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1`.
- Structure: `pose check --strict`.
- Spec readiness: `pose lint-spec pose-artifact-provenance-ledger --ready-check`.
- Spec lint: `pose lint-spec pose-artifact-provenance-ledger --strict`.
- Delivery gate: `pose validate --strict --module pose-mcp --report`.

### Risk-based test plan

| Scenario | Command | Expected evidence |
|---|---|---|
| Unit — artifact grammar and canonical paths | `go -C pose-mcp test ./internal/pose -run 'Artifact|DeliveryIntegrity' -count=1` | Closed actions parse; traversal, globs, directories and contradictory claims fail. |
| Integration — real Git actions and selectors | `go -C pose-mcp test ./internal/cli -run 'Artifact|ChangeSet' -count=1` | Create/modify/rename/remove match a bounded diff and unsafe revisions fail before Git invocation. |
| Contract — index and report history | `go -C pose-mcp test ./internal/cli -run 'DeliveryIntegrity|ReportChangeSet' -count=1` | Byte-stable graph/reverse traversal and change-set fields participate in stable hashes. |
| Contract — MCP projection | `go -C pose-mcp test ./internal/mcpserver -run 'DeliveryIntegrity' -count=1` | Project-scoped read model matches CLI schema and leaks no contents or absolute roots. |
| Migration — legacy/backfill | `go -C pose-mcp test ./internal/cli -run 'ArtifactBackfill|LegacyArtifact' -count=1` | Dry-run is non-mutating and ambiguity remains an unresolved finding. |
| Required module gate | `pose validate --strict --module pose-mcp --report` | Full Go validation and evidence report pass. |

### Execution log
- 2026-08-02, planning: repository discovery completed with
  `pose assess discover --component .`; implementation checks remain pending.
- 2026-08-02, planning validation: readiness and strict lint passed; the full Go
  suite and embedded-scaffold parity passed after regenerating the mirror.
- 2026-08-03, implementation: added the closed artifact grammar, confined path
  validator, declaration/observation graph, immutable Git change sets, report
  hashing, index, CLI checks/backfill and read-only MCP projection.
- 2026-08-03, review remediation: made spec parsing independent of the process
  working directory, required the explicit `--from-git` selector, exposed
  per-spec backfill confidence and added real rename-with-edit plus malformed
  history regression tests.
- 2026-08-03, final validation: focused and full Go suites, strict structure,
  ready/strict spec lint, module validation, scaffold parity, artifact-to-Git
  reconciliation and `govulncheck` passed.

### Results summary
- Successes: all ten requirements are implemented and traced. The final
  `artifact-check` resolved the implementation range to immutable commits and a
  diff digest with zero error or critical findings. CLI and MCP share schema 1.
- Failures: the first full-suite review found assessment mirror drift; the
  scaffold was regenerated and the full suite then passed. A rename fixture
  initially fell below Git's similarity threshold; the fixture was corrected
  to exercise an actual rename-with-edit and now passes.
- Warnings: 192 historical governed files have no structured spec attribution.
  They remain visible `orphan` warnings under the staged policy and do not
  weaken action-mismatch, undeclared or path-confinement enforcement for this
  adopted spec.

### Requirement trace
- R1 [satisfied] test:TestArtifactClaimsClosedGrammarAndContradictions.
- R2 [satisfied] test:TestArtifactPathRejectsSymlinkEscapeAndDirectory.
- R3 [satisfied] check:artifact-aware strict spec lint and guarded closeout.
- R4 [satisfied] test:TestArtifactCheckFindsUndeclaredAndActionMismatch.
- R5 [satisfied] test:TestArtifactCheckMatchesExplicitGitChangeSetAndRejectsUnsafeRevision.
- R6 [satisfied] test:TestReportChangeSetPersistsImmutableGitEvidence.
- R7 [satisfied] test:TestIndexWritesDeliveryIntegrityGraphAndReverseLookup.
- R8 [satisfied] check:.pose/policy/artifacts.json.
- R9 [satisfied] test:TestArtifactBackfillDryRunDoesNotMutateSpecs.
- R10 [satisfied] test:TestDeliveryIntegrityToolReadsProjectScopedGraphAndReversePath.

### Known gaps
- Historical orphan attribution is intentionally not fabricated. Repositories
  can use the deterministic backfill proposal and review ambiguous ownership
  before raising orphan severity.
- Reachability and user-visible delivery remain intentionally outside this
  ledger and are owned by dependent spec `pose-delivery-surface-assurance`.

## 7. Final Report

### Delivered scope
Implemented the Git-verifiable artifact half of POSE's delivery-integrity
graph: structured claims, immutable observations, reconciliation findings,
policy, strict lifecycle gates, dry-run backfill, reverse index and MCP parity.

### Files and modules changed
- `pose-mcp/internal/pose`: grammar, policy, graph, findings and reverse query.
- `pose-mcp/internal/cli`: Git evidence, report history, check/backfill/index and lifecycle gates.
- `pose-mcp/internal/mcpserver`: project-scoped read projection and golden catalog.
- `.pose/`, `POSE.md`, locales and docs: policy, ADR, templates and operating contract.

### Validation executed
- Commands: focused artifact/change-set/report/MCP tests; `go test ./...
  -count=1`; scaffold parity; `pose artifact-check --spec
  pose-artifact-provenance-ledger --from 5403f56 --to HEAD --strict`; `pose
  check --strict`; ready/strict spec lint; `pose validate --strict --module
  pose-mcp --report`; integration and tech-debt assessments; `govulncheck ./...`.
- Result: all blocking gates passed; no called Go vulnerability was found and
  the implementation change set has no error or critical provenance finding.

### Residual risks
- Rewritten Git history can stale selectors. POSE retains resolved commit IDs
  and diff digest so the mismatch is explicit and requires new evidence.
- Mature repositories need reviewed orphan-policy migration; warnings are not
  interpreted as automatic ownership.

### Follow-ups
No deferred implementation follow-up. Surface and release completeness are
already represented by the dependent specs in the active roadmap.
