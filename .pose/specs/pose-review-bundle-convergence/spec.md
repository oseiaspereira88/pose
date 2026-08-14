---
slug: pose-review-bundle-convergence
status: done
created_at: 2026-08-13
completed_at: 2026-08-14
supersedes:
depends_on: pose-hierarchical-review-closeout, pose-component-aware-review-plans
priority: 0
components: pose-mcp, cli, reviews, mcp, scaffold, docs
delivers: surface:review-bundle-cli, contract:review-bundle-mcp, governance:convergent-review-closeout
---

# Spec: Convergent review bundles and attestations

## 1. Intent

### Goal
Make POSE review and closeout converge in one bounded cycle by reviewing an
immutable, semantically scoped `ReviewBundle` and recording approval as a
separate attestation that cannot invalidate the bundle it approves.

### Business value
The component-aware review mechanism detects real gaps, but its current
freshness model can turn ordinary closeout bookkeeping into repeated review.
The implementation of `pose-component-aware-review-plans` required successive
provenance, validation, state-refresh and rereview corrections because the
closeout process kept changing inputs used by its own digest. That behavior is
too mechanical to promote from opt-in/preview to the normal path.

This feature establishes a fixed point: prepare all governed inputs, seal one
review subject, review it, attach an attestation and close it without creating
new review inputs. It also keeps POSE useful offline while allowing Harne8's
Conductor to own durable assignment, retries, findings, remediation and
targeted rereview.

### Constraints
- Preserve immutable review history and schema-v1/v2 readability.
- Keep deterministic validation separate from technical review judgment.
- Keep the POSE core offline, provider-neutral and independently verifiable.
- Preserve existing authorization boundaries; bundle creation never executes
  recommended tools or authorizes external writes.
- Treat canonicalization and digest composition as public compatibility
  contracts with golden fixtures.
- Fail closed on ambiguous provenance, malformed attestations, digest drift and
  untrusted external issuers.
- Keep reviewer identity to an alias or execution identity and exclude prompt,
  productivity and personal telemetry.
- Retain component-aware review as opt-in/preview until every release-blocking
  convergence case in this spec passes.
- Materialize the validated source build at `/home/go/.local/bin/pose` only
  after implementation checks pass; never replace the home binary with an
  unvalidated build.

### Non-goals
- Make Harne8 or Conductor mandatory for local review or closeout.
- Make POSE schedule agents, retry workflows or manage provider credentials.
- Claim that a digest, checklist or attestation guarantees reviewer quality.
- Accept arbitrary shell commands, provider-defined executable hooks or
  unsigned remote assertions without an explicit local trust policy.
- Reopen historical closed scopes merely because a repository adopts the new
  bundle schema.
- Implement the Conductor review workflow in this repository; this spec ships
  the portable contract and POSE-side verifier it will consume.

## 2. Requirements

### Functional
- R1: When a typed review scope is prepared, POSE shall resolve a canonical
  `ReviewBundle` containing the scope projection, attributed change subject,
  effective review plan, required evidence manifest and child bundle digests
  applicable to that scope.
- R2: POSE shall expose bundle states `prepared`, `sealed`, `superseded` and
  `attested`; only a sealed bundle may receive an approval attestation.
- R3: Sealing shall require an immutable attributed change set with stable
  patch/tree identities, a current unblocked review plan and every required
  pre-review validation result; a working-tree-only or unresolved provider SHA
  shall remain previewable but unsealable.
- R4: The canonical bundle payload shall include only governed semantic inputs:
  normalized intent, requirements, technical plan and decisions; typed
  components and delivery claims; acknowledged amendments; the consumed
  policy/profile/rule/index projections; immutable implementation subject;
  required validation identities; and hierarchical child digests.
- R5: The bundle digest shall exclude lifecycle fields, review artifacts,
  reviewer identity, attestation content, timestamps, execution-log prose,
  rendered reports, state caches, generated assessments/indexes not consumed by
  plan resolution and the bookkeeping performed by closeout.
- R6: POSE shall classify every included and excluded input in `--explain`
  output with its source, normalized identity and reason; an unclassified path
  in the attributed change set shall block sealing rather than be silently
  excluded.
- R7: A sealed bundle shall be stored as immutable canonical JSON under
  `.pose/review-bundles/`; its ID shall derive from its SHA-256 payload digest,
  and an existing ID with different bytes shall fail.
- R8: A review attestation shall reference the exact bundle ID and digest,
  reviewer/execution alias, decision, criterion dispositions, tool
  dispositions, evidence references, findings, creation time and optional
  superseded attestation without embedding or mutating the bundle.
- R9: `review-check` and `closeout-check` shall validate the current attestation
  against the sealed bundle. Creating, importing or verifying the attestation
  shall not change the bundle digest.
- R10: After an approved attestation, `pose close` shall perform only the
  guarded lifecycle transition and derived refreshes. Those mutations shall not
  invalidate the approved bundle or require a second review attempt.
- R11: When governed semantic inputs, attributed implementation content,
  consumed review policy or required evidence change, POSE shall create a new
  bundle version, retain the prior bundle and attestation, and make the prior
  approval ineligible for the new subject.
- R12: When only excluded derived or lifecycle inputs change, POSE shall retain
  the current bundle and approval and report the excluded delta without
  creating false staleness.
- R13: A superseding bundle shall expose a deterministic delta containing
  changed components, criteria, evidence classes, findings and subject paths so
  an orchestrator can request targeted rereview.
- R14: POSE may reuse an earlier passed criterion only when its criterion
  contract digest, governed input slice, evidence identities and independence
  policy are unchanged; reuse shall be explicit in the new attestation and
  policy may disable it by criticality or profile.
- R15: The verifier shall project a complete final disposition for every
  effective criterion even when some dispositions were reused; an incomplete,
  transitively stale or policy-forbidden partial rereview shall block approval.
- R16: Provider merge/squash SHAs shall be recorded as advisory provenance.
  Stable subject identity shall use canonical patch and tree/manifest digests
  so a non-fetchable synthetic SHA does not by itself block verification.
- R17: POSE shall support a local flow and an external-orchestrator flow over
  the same versioned bundle/attestation schemas. External orchestration may
  assign reviewers and manage retries, but POSE remains the closeout authority.
- R18: POSE shall export a sealed bundle and import an attestation without
  network access. Import shall be dry-run by default and append only after
  explicit `--apply`.
- R19: External attestations shall support an optional signed envelope with
  issuer, subject and signature metadata. When trust policy requires signing,
  issuer/subject mismatch, invalid signature or missing verification material
  shall block import.
- R20: Milestone and roadmap bundles shall bind ordered child bundle digests
  and their own exit/cut criteria; a changed child shall invalidate only the
  affected ancestor chain and shall identify the smallest stale child.
- R21: The CLI shall provide `pose review bundle <scope>` for prepare/explain,
  `--seal` for immutable persistence, `pose review attest` for local recording,
  and `pose review verify` for bundle/attestation verification. Equivalent
  read-only projections shall remain available through `review-plan`,
  `review-check` and `closeout-check`.
- R22: The MCP server shall expose read-only project-confined bundle resolution
  and verification. It shall not seal bundles or import attestations through an
  implicit write path.
- R23: Human and JSON state output shall distinguish `needs-validation`,
  `ready-to-seal`, `ready-for-review`, `changes-requested`, `ready-to-close`,
  `closed` and `superseded`, with the smallest next governed action.
- R24: Usage analytics shall count bundle preparation, sealing, verification,
  false-staleness avoided, supersession and targeted criterion reuse without
  storing reviewer productivity metrics or attestation bodies.

### Non-functional
- R25: Identical governed inputs shall produce byte-identical canonical payloads
  and bundle digests across repeated runs, CLI/MCP paths and supported operating
  systems.
- R26: The happy path `prepare -> validate -> seal -> review -> attest -> close`
  shall converge without a corrective post-approval mutation or rereview.
- R27: Bundle resolution and verification shall remain bounded to the
  authorized project root, attributed change set and explicitly consumed
  governed files; no recursive unbounded repository scan is permitted.
- R28: JSON schemas, canonical ordering, digest algorithms and state enums shall
  be versioned and documented; incompatible canonicalization changes require a
  new schema version and ADR.
- R29: Bundle operations shall be idempotent. Replaying an identical seal or
  attestation import shall return the existing identity; conflicting bytes for
  an existing identity shall fail visibly.
- R30: Error output shall identify the exact invalid field, source path,
  expected digest or trust rule without leaking file contents, prompts, command
  output or secrets.
- R31: The feature shall add no network dependency and no new third-party Go
  dependency unless separately justified, vulnerability-scanned and recorded.

### Security
- R32: All bundle, evidence, policy, signature and attestation paths shall be
  canonicalized and confined beneath the authorized project root, including
  traversal and symlink-escape rejection.
- R33: Bundle and attestation parsers shall reject unknown critical fields,
  duplicate IDs, duplicate JSON keys where detectable, unsupported algorithms,
  malformed digests, oversized documents and control-character injection.
- R34: Repository-controlled data shall never select an arbitrary executable.
  Tool dispositions shall continue to reference the closed native catalog and
  typed arguments.
- R35: A less trusted profile, imported attestation or external issuer shall not
  weaken reviewer-independence, evidence, signature or accepted-risk policy.
- R36: Secret scanning, `govulncheck`, parser/path negative tests and sensitive
  log review shall pass before the feature can leave preview.

### Compatibility
- R37: Existing schema-v1 and schema-v2 policies, plans and review attempts
  shall remain readable and verifiable under their original rules; migration
  shall be explicit and shall not rewrite immutable artifacts.
- R38: New repositories may adopt bundle schema v1 directly. Existing
  repositories shall receive a dry-run migration explanation and adoption date;
  legacy done scopes shall retain the configured exemption.
- R39: `pose review record` shall remain a supported compatibility entrypoint.
  Under bundle-enabled policy it shall delegate to the attestation path and
  require or resolve an exact sealed bundle; under legacy policy it shall retain
  current behavior.
- R40: CLI, MCP catalog, JSON schemas, English/pt-BR scaffolds, workflows,
  skills, POSE manual and docs-site shall ship in parity in the same release.
- R41: The feature shall remain opt-in/preview until R26 and every required case
  in the validation plan pass in both source-build and installed-binary smoke
  tests.
- R42: After all required checks pass, the exact validated source revision shall
  be built with reproducible flags, installed atomically at
  `/home/go/.local/bin/pose`, and verified with `pose version`, `pose doctor
  --json` and bundle CLI smoke tests before release preparation.
- R43: Adding or changing a delivery target, bundle or evidence record shall
  invalidate only open/current scopes that consume the changed input. Unrelated
  historical closed specs and their provenance-bound evidence shall remain
  valid instead of being re-evaluated against the new module-wide target set.
- R44: Human review diagnostics shall group repeated unmapped or derived-path
  warnings by stable cause code, show count and representative paths, and keep
  the full machine-readable detail available without emitting one checklist
  item per path.
- R45: Human tool guidance shall be phase-aware and actionable: deduplicate
  equivalent invocations, separate required from recommended tools and defer
  completion-only tools from the active execution list until their precondition
  is met. The canonical JSON plan shall retain every underlying criterion and
  provenance record.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: bundle model, semantic scope projection, subject
  manifest, canonicalization, delta, attestation verification and closeout.
- `pose-mcp/internal/cli`: bundle, attest and verify subcommands plus legacy
  `review record` compatibility.
- `pose-mcp/internal/mcpserver`: read-only bundle/state tools and catalog parity.
- `pose-mcp/schemas/v1`: public bundle, attestation and external-envelope
  schemas.
- `.pose/policy` and `.pose/review-profiles`: explicit bundle adoption, reuse,
  trust and independence policy.
- `.pose/workflows`, `.agents/skills`, `POSE.md`, locales and docs-site:
  prepare/validate/seal/review/attest/close operating flow.
- Embedded scaffold: byte-equivalent defaults and migration behavior.
- `.pose/indexes/validation-matrix.json`: permanent convergence/contract gate if
  the focused suite proves stable enough for every strict validation.

### Consulted context
- `knowledge:pr15-component-aware-review-provenance` records the independent
  review, repeated provenance corrections and required-tool enforcement.
- `knowledge:project-agnostic-assessment-evidence` supplies the confined-root,
  symlink and evidence-derivation invariants reused by the closeout remediation.
- `knowledge:adr-component-aware-review-plans-review` records production
  false-staleness as an explicit review trigger.
- `adr:2026-08-02-immutable-hierarchical-review-and-closeout-evidence` defines
  immutable attempts and typed hierarchical scopes.
- `adr:2026-08-12-component-aware-effective-review-plans` defines effective
  plan composition, closed native tools and schema-v2 adoption.
- `adr:2026-08-13-sealed-review-bundles-and-attestations` defines the fixed-point
  boundary and portable orchestration contract introduced here.
- `knowledge:adr-sealed-review-bundles-review` tracks the decision's production
  review triggers through the preview and first release.
- `spec:pose-hierarchical-review-closeout` and
  `spec:pose-component-aware-review-plans` are retained historical contracts
  extended, not rewritten, by this spec.

### Artifacts
- created: .pose/specs/pose-review-bundle-convergence/spec.md
- created: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- created: .pose/knowledge/2026-08-13-decision-log-adr-sealed-review-bundles-review.md
- created: .pose/changelogs/unreleased/pose-review-bundle-convergence.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- created: .pose/results/review-bundle-convergence.json
- created: .pose/reports/2026-08-13-standard-review-bundle-convergence.md
- created: .pose/reports/history/standard-review-bundle-convergence.jsonl
- created: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_bundle_test.go
- created: pose-mcp/schemas/v1/review-bundle.schema.json
- created: pose-mcp/schemas/v1/review-attestation.schema.json
- created: pose-mcp/schemas/v1/review-attestation-envelope.schema.json
- created: tests/e2e/review-bundle/run.sh
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/delivery_surface_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/usage.go
- modified: pose-mcp/internal/cli/usage_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/closeout_tool_test.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/schemas/README.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/workflows/feature.md
- modified: .pose/workflows/review.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: POSE.md
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/architecture.md
- modified: locales/pt-BR/POSE.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/review.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md

### Delivery targets
- surface:review-bundle-cli module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- contract:review-bundle-mcp module:pose-mcp profile:api-contract entrypoint:pose-mcp/internal/mcpserver/server.go
- governance:convergent-review-closeout module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### Contract model
Store an immutable envelope whose identity is derived only from the canonical
payload:

```json
{
  "schema_version": 1,
  "bundle_id": "rvb-<digest-prefix>",
  "bundle_digest": "sha256:<hex>",
  "sealed_at": "2026-08-13T00:00:00Z",
  "payload": {
    "scope": {},
    "subject": {
      "base": "<advisory-sha>",
      "head": "<advisory-sha>",
      "patch_digest": "sha256:<hex>",
      "tree_digest": "sha256:<hex>",
      "entries": []
    },
    "plan": {"plan_digest": "sha256:<hex>", "consumed_inputs": []},
    "evidence": [],
    "children": []
  },
  "excluded_inputs": []
}
```

`sealed_at`, the rendered exclusion explanation and storage path are envelope
metadata and do not participate in `bundle_digest`. Canonical payload JSON uses
UTF-8, lexicographically ordered object keys, schema-defined array ordering,
normalized newlines and no insignificant whitespace. Unknown critical fields
fail; additive non-critical envelope metadata may be ignored by an older
reader.

The attestation references, but never contains, the canonical payload:

```json
{
  "schema_version": 1,
  "attestation_id": "rva-<digest-prefix>",
  "bundle_id": "rvb-...",
  "bundle_digest": "sha256:<hex>",
  "reviewer": "agent:review-execution",
  "decision": "approved",
  "criteria": [],
  "tools": [],
  "findings": [],
  "reused_from": [],
  "supersedes": "rva-...",
  "attested_at": "2026-08-13T00:00:00Z"
}
```

The implementation shall refine field shapes in the schemas without weakening
R1-R42. Any departure from the digest boundary requires updating the ADR before
code changes.

### Semantic projection and exclusion rules
1. Parse the typed scope and authorize its project root.
2. Normalize only the spec/roadmap sections and frontmatter fields declared by
   R4; do not hash the entire Markdown document.
3. Resolve attributed source artifacts. Treat the spec file through its
   semantic projection rather than its raw Git bytes.
4. Classify source, governed configuration and derived artifacts through a
   closed registry. Block any attributed path without a class.
5. Resolve an immutable patch/tree or offline content manifest and retain
   provider SHAs only as human-usable provenance.
6. Resolve the component-aware plan and capture only the policy, profile, rule
   and index slices it actually consumed.
7. Verify and capture required validation result identities before sealing.
8. Bind child bundle digests for milestone and roadmap scopes.
9. Canonicalize the payload, compute the digest and derive the bundle ID.
10. Persist atomically and refuse mutation or ID collision.

Lifecycle status, `completed_at`, task checkbox state, execution logs, rendered
result summaries, follow-up bookkeeping, review/attestation files and derived
state/assessment/report/index refreshes are outside the semantic projection.
The corresponding gates still validate them at closeout; exclusion from the
review subject does not waive their own correctness requirements.

### CLI and MCP changes
- Add `pose review bundle <scope> [--json] [--explain] [--seal]`.
- Add `pose review attest <bundle-id|path> ... [--apply]`.
- Add `pose review verify <scope|bundle-id|path> [--json]`.
- Keep `pose review record` as the compatibility adapter specified by R39.
- Add read-only `pose_review_bundle` and extend `pose_closeout_state` with
  bundle, attestation, delta and next-action fields.
- Use existing project selection and authorization semantics for MCP.
- Do not expose a mutating MCP seal/import tool in this release.

### Migration and rollout
1. Land schemas, canonical fixtures and the ADR before runtime behavior.
2. Implement read-only prepare/explain and legacy projections.
3. Implement sealing, immutable storage and idempotency.
4. Implement local attest/verify and closeout integration.
5. Implement delta and policy-bounded criterion reuse.
6. Implement external envelope verification without making signatures
   mandatory for local/manual attestations.
7. Ship CLI/MCP/docs/scaffold parity and migration dry-run.
8. Run the installed-binary smoke on `/home/go/.local/bin/pose`.
9. Keep the feature preview/opt-in until all release blockers pass.
10. Promote it in the next release only after an independent review confirms
    the one-cycle convergence evidence.

### Data/storage changes
- Add append-only `.pose/review-bundles/<bundle-id>.json` artifacts.
- Retain `.pose/reviews/` as the immutable human-review history; bundle-enabled
  attempts become schema-v3-compatible attestations while prior files remain
  unchanged and readable.
- Add optional review-policy fields for bundle adoption date, classification,
  criterion reuse, signature requirements and trusted issuers.
- Do not persist mutable `current` pointers; derive current bundle and
  attestation from immutable ancestry and scope state.

### Technical risks
- An overly broad semantic projection recreates false staleness; an overly
  narrow one lets meaningful changes bypass rereview. Golden include/exclude
  fixtures and unknown-path fail-closed behavior mitigate both directions.
- Patch IDs can be unstable across rename or merge representation. Pair a
  canonical patch digest with a sorted content/tree manifest and test synthetic
  provider merges.
- Criterion reuse can hide a cross-cutting impact. Require exact sliced-input
  equality, explicit provenance and stricter policy overrides.
- External signatures can introduce provider lock-in or unavailable
  dependencies. Keep the base contract offline and make signed envelopes an
  optional trust-policy layer.
- Compatibility adapters can make two sources of truth. Resolve both legacy and
  new commands through one internal evaluator once bundle policy is active.
- Scope classification rules can drift from scaffold distribution. Contract
  tests must compare source, embedded English and pt-BR copies.

## 4. Tasks

### Planning and contracts
- [x] Run `pose state` and confirm the project-state artifact is fresh.
- [x] Run `pose assess discover --component pose-mcp` before modifying code.
- [x] Read feature, ADR and test-plan rules and the validation matrix.
- [x] Consume the two review knowledge artifacts and historical ADR/specs.
- [x] Define the fixed-point boundary and migration constraints in an ADR.
- [x] Validate this spec with `pose lint-spec pose-review-bundle-convergence --ready-check`.

### Increment 1 — canonical bundle
- [x] Add schemas and golden canonicalization fixtures.
- [x] Implement typed semantic projection and closed path classification.
- [x] Implement patch/tree subject identity and synthetic-SHA fallback.
- [x] Implement prepare/explain without persistent writes.

### Increment 2 — seal and attestation
- [x] Implement immutable, atomic and idempotent bundle sealing.
- [x] Implement local attestation record/import and exact bundle verification.
- [x] Bind review/closeout checks to bundle attestations.
- [x] Prove attestation and closeout mutations do not change the bundle.

### Increment 3 — remediation and orchestration contract
- [x] Implement bundle supersession and deterministic delta projection.
- [x] Implement policy-bounded criterion reuse with complete final projection.
- [x] Implement optional signed external envelopes and trust-policy negatives.
- [x] Document the Conductor contract without adding an online dependency.

### Increment 4 — distribution and migration
- [x] Add CLI and read-only MCP projections with catalog/schema parity.
- [x] Add explicit policy migration dry-run and legacy compatibility fixtures.
- [x] Update workflows, skills, docs, locales and embedded scaffold.
- [x] Add the stable convergence suite to the validation matrix.
- [x] Add the unreleased changelog fragment.

### Validation and release readiness
- [x] Run all focused unit, contract, negative and end-to-end cases in section 6.
- [x] Run `pose assess integrate` for CLI/MCP/schema contract changes.
- [x] Run `pose validate --strict --module pose-mcp --report --report-task review-bundle-convergence`.
- [x] Run security, artifact, surface, skill and knowledge checks.
- [x] Run staged-history and strict spec lifecycle checks.
- [x] Build the validated revision and atomically install it at
  `/home/go/.local/bin/pose`.
- [x] Run installed-binary doctor/version and fresh-repository smoke tests.
- [x] Perform an independent review against the sealed bundle.
- [x] Run `pose assess discover --update-state`.
- [x] Run closeout and final release gates.

## 5. Decisions

### Decision 1 — fixed-point review subject
- Date: 2026-08-13
- Context: The current full-body `scope_digest` changes when closeout appends
  operational evidence and lifecycle data.
- Options considered: keep adding exclusions to `scope_digest`; freeze the
  entire repository; create a sealed semantic bundle and separate attestation.
- Decision: Create a sealed semantic `ReviewBundle` and bind approvals through
  separate immutable attestations.
- Rationale: This gives review an immutable subject while allowing closeout to
  record its own result without changing that subject.
- Consequences: Canonicalization becomes a public schema contract and needs
  explicit classification plus compatibility fixtures.

### Decision 2 — core versus orchestration
- Date: 2026-08-13
- Context: Assignment, retries and targeted rereview need durable workflow
  state, but POSE must continue to work without Harne8.
- Options considered: move all review to Conductor; keep all orchestration in
  POSE; let POSE own portable verification and Conductor own optional workflow.
- Decision: POSE prepares/seals/verifies bundles and attestations; Conductor may
  orchestrate them through the same exported contracts.
- Rationale: It matches “POSE governs, Conductor orchestrates” without creating
  an online or vendor dependency.
- Consequences: The cross-service boundary is versioned JSON, not an internal
  Go API or remote execution hook.

### Decision 3 — provenance identity
- Date: 2026-08-13
- Context: Git providers can expose transient merge/squash SHAs that cannot be
  fetched later.
- Options considered: require a reachable SHA; trust provider metadata; bind to
  canonical patch plus tree/content identities and retain SHA as advisory.
- Decision: Use patch and tree/content digests as stable subject identity.
- Rationale: Verification remains reproducible after provider refs move.
- Consequences: Rename, merge and line-ending normalization need explicit
  golden fixtures.

### Decision 4 — targeted rereview
- Date: 2026-08-13
- Context: Restarting every criterion after a narrow remediation makes review
  unnecessarily repetitive.
- Options considered: always full rereview; trust an orchestrator's affected
  list; let POSE derive deltas and permit exact policy-bounded reuse.
- Decision: POSE derives the delta and verifies any reused disposition.
- Rationale: Review becomes proportional without delegating validity to an
  external scheduler.
- Consequences: High-criticality policy may still require full rereview.

## 6. Validation

### Strategy
Treat this as a high-risk governance and schema change. Require unit coverage
for canonicalization and parsing, contract coverage for CLI/MCP/schema parity,
negative coverage for path/trust/digest inputs and an end-to-end convergence
smoke using both the source build and the installed home binary. No test may
claim external Conductor behavior; the portable export/import contract is the
integration boundary verified here.

### Risk-based cases

| Class | Scenario | Required command | Expected evidence |
|---|---|---|---|
| Unit | Identical inputs are byte stable | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleCanonical|TestReviewBundleDigestStable' -count=1` | Repeated payload bytes, IDs and SHA-256 digests are identical. |
| Unit | Semantic versus derived projection | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleSemanticProjection|TestReviewBundleDerivedChangesDoNotStale' -count=1` | Governed edits supersede; lifecycle/state/report/attestation edits do not. |
| Unit | Unknown input fails closed | `go -C pose-mcp test ./internal/pose -run TestReviewBundleRejectsUnclassifiedSubjectPath -count=1` | Sealing reports the exact unclassified path. |
| Unit | Immutable seal and replay | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleSealIsAtomic|TestReviewBundleSealIsIdempotent|TestReviewBundleRejectsIdentityCollision' -count=1` | Replay reuses identical bytes; collision and partial write fail. |
| Unit | Synthetic provider merge | `go -C pose-mcp test ./internal/pose -run TestReviewBundleVerifiesSyntheticMergeByPatchAndTree -count=1` | Non-fetchable advisory SHA verifies from stable subject digests. |
| Unit | Attestation separation | `go -C pose-mcp test ./internal/pose -run TestReviewAttestationDoesNotMutateBundle -count=1` | Attest and close retain the exact bundle bytes/digest. |
| Unit | One-cycle convergence | `go -C pose-mcp test ./internal/pose -run TestReviewBundleCloseoutConvergesWithoutRereview -count=1` | Flow reaches closed with one sealed bundle and one approved attestation. |
| Unit | Targeted rereview | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleDelta|TestReviewAttestationCriterionReuse' -count=1` | Only exact unchanged slices reuse; complete projection remains current. |
| Unit | Cross-scope isolation | `go -C pose-mcp test ./internal/pose -run TestReviewBundleDeliveryChangeDoesNotStaleUnrelatedClosedScopes -count=1` | New targets invalidate only consuming open/current scopes. |
| Unit | Actionable plan ergonomics | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'TestReviewPlanGroupsRepeatedWarnings|TestReviewPlanActionableToolPhases' -count=1` | Warnings are cause-grouped and tool guidance is phase-aware without losing canonical coverage. |
| Negative | Traversal and symlink escape | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleRejectsTraversal|TestReviewBundleRejectsSymlinkEscape' -count=1` | Every escaped path is denied before reading content. |
| Negative | Malformed/oversized JSON | `go -C pose-mcp test ./internal/pose -run 'TestReviewBundleRejectsMalformedInput|TestReviewAttestationRejectsMalformedInput' -count=1` | Duplicate/unknown critical fields, bad digests and size overflow fail clearly. |
| Negative | External trust | `go -C pose-mcp test ./internal/pose -run TestReviewAttestationEnvelopeTrustPolicy -count=1` | Invalid/missing signature and issuer/subject mismatch block when required. |
| Compatibility | Legacy schema v1/v2 | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'TestReviewBundleLegacy|TestReviewRecordCompatibility' -count=1` | Old attempts stay readable; `review record` follows policy mode. |
| Contract | CLI/MCP/schema parity | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run 'ReviewBundle|ToolCatalog' -count=1` | Same project-scoped JSON contract and exact golden catalog. |
| Distribution | Scaffold/locales parity | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'ReviewBundle.*Scaffold|ReviewBundle.*Locale' -count=1` | Source, embedded English and pt-BR contracts match. |
| Integration | POSE assessment | `pose assess integrate` | CLI, MCP and JSON schema providers/consumers have no uncovered change. |
| Integration | Strict module matrix | `pose validate --strict --module pose-mcp --report --report-task review-bundle-convergence` | Every required matrix check passes with a structured report. |
| E2E | Source-build fresh repo | `tests/e2e/review-bundle/run.sh` | Prepare, validate, seal, attest, close and recheck finish without stale review. |
| E2E | Installed home binary | `/home/go/.local/bin/pose version && /home/go/.local/bin/pose doctor --json && POSE_BIN=/home/go/.local/bin/pose tests/e2e/review-bundle/run.sh` | Installed revision matches the validated build and passes the same flow. |
| Security | Vulnerability scan | `/home/go/go/bin/govulncheck ./pose-mcp/...` | No reachable known vulnerability is introduced. |
| Security | Secret scan | `gitleaks git --no-banner --redact --log-opts='--all'` | No secret is detected in the changed scope or history. |
| Delivery | Artifact provenance | `pose artifact-check --spec pose-review-bundle-convergence --strict` | Declared and Git-observed artifacts match an immutable change set. |
| Delivery | Surface and contract | `pose surface-check --spec pose-review-bundle-convergence --strict` | CLI reachability and CLI/MCP/governance integration evidence are current. |
| Governance | History/skills/spec | `pose history-check --strict && pose skills-check --strict && pose lint-spec pose-review-bundle-convergence --strict` | Append-only history, distributed skills and spec lifecycle all pass. |

### Release blockers
- R26 one-cycle convergence passes from a fresh repository.
- Attestation creation and closeout leave bundle bytes unchanged.
- Derived-only state, assessment, report and index refreshes do not stale review.
- Every semantic code/spec/policy/evidence mutation does stale or supersede.
- Synthetic merge/squash provenance verifies without a persistent provider SHA.
- Targeted reuse fails closed for a changed or transitively affected criterion.
- Adding the new delivery target does not stale evidence for unrelated closed
  specs or expand their historical provenance subject.
- Human output groups repeated mapping warnings and does not present deferred or
  duplicate tools as active work.
- Offline flow passes with all external integrations disabled.
- Installed `/home/go/.local/bin/pose` matches and passes the validated source
  build before release preparation.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...`
- Scope: all Go engine, CLI, MCP and scaffold packages.
- Expected: all unit, contract and regression tests pass.

#### Lint / build
- Command: `go -C pose-mcp vet ./... && go -C pose-mcp build -trimpath ./cmd/pose`
- Scope: Go static analysis and production entrypoint.
- Expected: no diagnostics and a successful reproducible build.

#### Security / contract
- Command: `/home/go/go/bin/govulncheck ./pose-mcp/...`
- Scope: reachable Go dependency graph.
- Expected: no reachable known vulnerability.
- Command: `pose assess integrate`
- Scope: CLI, MCP and JSON schema changes.
- Expected: all changed contracts and consumers are visible and covered.

#### POSE gates
- Command: `pose validate --strict --module pose-mcp --report --report-task review-bundle-convergence`
- Scope: canonical module validation matrix.
- Expected: every required check passes and structured evidence is recorded.
- Command: `pose artifact-check --spec pose-review-bundle-convergence --strict`
- Scope: declared versus immutable Git-observed artifacts.
- Expected: no missing, extra or unattributed source artifact.
- Command: `pose surface-check --spec pose-review-bundle-convergence --strict`
- Scope: three declared delivery targets.
- Expected: reachability and integration evidence are fresh.

### Execution log
- 2026-08-13 — discovery and routing: `pose state`,
  `pose assess discover --component pose-mcp` and
  `pose suggest feature --path pose-mcp` passed before implementation.
- 2026-08-13 — implementation landed in `89b100a`; independent review then
  found mutable-worktree inclusion, over-broad hierarchy subjects, managed
  directory symlink traversal, incomplete deltas and regressing overlapping
  ranges. Corrections landed in `e6fe84e` and `60171f8` with focused regression
  coverage; `dbd41e3` makes synthetic-provider verification explicit.
- 2026-08-13 — `go -C pose-mcp test ./...`, `go vet ./...`, the focused race
  suite and all five registered strict matrix checks passed at `dbd41e3`.
  Listener-dependent tests ran outside the restricted sandbox.
- 2026-08-13 — source and installed-binary E2E both passed. The exact
  `-trimpath` build installed atomically at `/home/go/.local/bin/pose` has
  SHA-256 `035ef05f066986eebf705eb71517b6bf1c6058534ed507fd0941404212fc1095`,
  reports `pose 1.0.0-dev`, and `pose doctor --json` reports zero errors.
- 2026-08-13 — `govulncheck ./...` found no vulnerabilities; gitleaks v8.21.2
  scanned 382 commits and found no secrets.
- 2026-08-13 — `pose assess integrate` evaluated 53 contracts. Its 52
  unobserved-consumer warnings are repository-wide provider inventory, not an
  uncovered changed contract. `pose assess tech-debt` found one accepted
  construction-invariant panic and no uncovered debt.
- 2026-08-13 — `pose assess discover --update-state` refreshed two components:
  `pose-mcp` at 31,140 production / 18,914 test LOC and `mcp-enforce` at
  870 production / 1,029 test LOC, with zero TODO/FIXME markers.
- 2026-08-13 — strict artifact provenance matched 64 claims to 64 observed
  paths with no errors; the 231 warnings are preexisting repository-wide
  orphan inventory. Strict surface assurance passed all three delivery targets
  with five current results and zero findings. Skills and knowledge gates also
  passed without errors or warnings.

### Results summary
- Successes: immutable semantic bundles, separate append-only attestations,
  deterministic supersession deltas, policy-bounded criterion reuse,
  CLI/MCP/schema/scaffold parity, one-cycle closeout and scoped delivery
  provenance are implemented and covered by source and installed-binary tests.
- Independent review corrections: working-tree-only subjects now fail closed;
  hierarchy resolution stays within declared children and order; managed
  artifact directories reject symlinks; deltas project components, evidence
  classes and findings; overlapping ranges collapse to the broadest immutable
  subject; human explain output identifies every included subject.
- Warnings: the optional pre-commit hook is not installed, external MCP/HTTP
  consumers are intentionally unobserved in this portable repository, and the
  repository retains its historical orphan-provenance warning inventory. None
  is a delivery-target or changed-contract error for this spec.

### Requirement trace
- R1 [satisfied] `PrepareReviewBundle`, schema fixtures and
  `TestReviewBundleCanonicalAndDigestStable` commit:89b100a.
- R2 [satisfied] bundle/verification state projection and
  `TestReviewBundleCLISealsAttestsAndVerifies` commit:89b100a.
- R3 [satisfied] required-evidence gating plus
  `TestReviewBundleRejectsWorkingTreeOnlySubjectContent` commit:e6fe84e.
- R4 [satisfied] semantic projection and
  `TestReviewBundleSemanticProjectionAndDerivedChangesDoNotStale` commit:89b100a.
- R5 [satisfied] derived/lifecycle exclusions in
  `TestReviewBundleDerivedOnlyChangeSetDoesNotStale` commit:89b100a.
- R6 [satisfied] classified human/JSON explain output and
  `TestReviewBundleRejectsUnclassifiedSubjectPath` commit:e6fe84e.
- R7 [satisfied] immutable atomic storage and collision coverage in
  `TestReviewBundleSealIsAtomicIdempotentAndDetectsSubjectChange` commit:89b100a.
- R8 [satisfied] the separate attestation schema and
  `TestReviewAttestationDoesNotMutateBundleAndConverges` commit:89b100a.
- R9 [satisfied] bundle-aware verification and closeout checks exercised by
  `TestReviewBundleCLISealsAttestsAndVerifies` commit:89b100a.
- R10 [satisfied] one-cycle lifecycle convergence in
  `TestReviewAttestationDoesNotMutateBundleAndConverges` commit:89b100a
  governance:convergent-review-closeout evidence:integration
  check:review-bundle-convergence.
- R11 [satisfied] semantic supersession retention in
  `TestReviewBundleSealIsAtomicIdempotentAndDetectsSubjectChange` commit:89b100a.
- R12 [satisfied] `TestReviewBundleDerivedOnlyChangeSetDoesNotStale`
  commit:89b100a.
- R13 [satisfied] `TestReviewBundleDeltaIncludesChangedComponentsAndEvidenceClasses`
  and changed-finding projection commit:e6fe84e.
- R14 [satisfied] `TestReviewAttestationCriterionReuseRequiresExactUnchangedContract`
  commit:89b100a.
- R15 [satisfied] `TestReviewAttestationCriterionReuseRejectsChangedSubjectSlice`
  and complete attestation validation commit:89b100a.
- R16 [satisfied] `TestReviewBundleVerifiesSyntheticMergeByPatchAndTree`
  against a deliberately non-fetchable advisory ref commit:dbd41e3.
- R17 [satisfied] shared local/export/import schemas and the documented
  provider-neutral orchestration boundary commit:89b100a.
- R18 [satisfied] dry-run/apply CLI behavior in
  `TestReviewBundleCLISealsAttestsAndVerifies` commit:89b100a.
- R19 [satisfied] `TestReviewAttestationEnvelopeTrustPolicy` commit:89b100a.
- R20 [satisfied] `TestReviewBundleMilestoneSubjectIsConfinedAndChildOrderIsDeclared`
  commit:e6fe84e.
- R21 [satisfied] surface:review-bundle-cli evidence:e2e
  check:delivery-reachability; CLI command and compatibility coverage in
  `TestReviewBundleCLISealsAttestsAndVerifies` commit:89b100a.
- R22 [satisfied] `TestReviewBundleToolIsReadOnlyAndProjectScoped`
  commit:89b100a contract:review-bundle-mcp evidence:integration
  check:review-bundle-convergence.
- R23 [satisfied] human/JSON state and next-action assertions across the CLI
  and verification suites commit:89b100a.
- R24 [satisfied] `TestReviewBundleUsageDistinguishesOperationsAndSignals`
  commit:89b100a.
- R25 [satisfied] `TestReviewBundleCanonicalAndDigestStable` and CLI/MCP
  contract parity commit:89b100a.
- R26 [satisfied] source and installed `tests/e2e/review-bundle/run.sh` plus
  `TestReviewAttestationDoesNotMutateBundleAndConverges` test:review-bundle-e2e.
- R27 [satisfied] confined change-set resolution and hierarchy regression
  coverage commit:e6fe84e.
- R28 [satisfied] the three v1 schemas, schemas README and ADR
  `2026-08-13-sealed-review-bundles-and-attestations` commit:89b100a.
- R29 [satisfied] bundle and attestation idempotency/collision tests
  commit:89b100a.
- R30 [satisfied] malformed input, digest and path negative suites
  commit:89b100a commit:e6fe84e.
- R31 [satisfied] offline source/installed E2E, unchanged Go dependencies and
  successful `govulncheck ./...`; check:review-bundle-convergence.
- R32 [satisfied] traversal, file-symlink and managed-directory-symlink
  rejection tests commit:89b100a commit:e6fe84e.
- R33 [satisfied] `TestReviewBundleRejectsMalformedInput` and
  `TestReviewAttestationRejectsMalformedInput` commit:89b100a.
- R34 [satisfied] the closed native tool catalog and
  `TestReviewPlanRecommendationsAreClosedAndNonExecutable` commit:89b100a.
- R35 [satisfied] trust/independence preservation tests in review plan and
  signed-envelope suites commit:89b100a.
- R36 [satisfied] negative parser/path tests, race tests, govulncheck and the
  382-commit gitleaks scan; check:review-bundle-convergence.
- R37 [satisfied] `TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly`
  and closed-scope compatibility coverage commit:89b100a.
- R38 [satisfied] policy migration/exemption tests including
  `TestReviewPolicyExemptsLegacyDoneScopesUnlessOptedIn` commit:89b100a.
- R39 [satisfied] `TestReviewRecordDelegatesToBundleAttestationWhenAdopted`
  and legacy record tests commit:89b100a.
- R40 [satisfied] strict catalog, locale and embedded-scaffold parity tests
  in the full Go suite commit:89b100a.
- R41 [satisfied] keeping bundle adoption opt-in/preview while source and
  installed-binary convergence cases pass; test:review-bundle-e2e.
- R42 [satisfied] the atomically installed `-trimpath` binary with SHA-256
  `035ef05f...`, zero doctor errors and installed E2E pass commit:5646f90.
- R43 [satisfied] `TestReviewBundleDeliveryChangeDoesNotStaleUnrelatedClosedScopes`
  and current strict surface assurance commit:89b100a.
- R44 [satisfied] `TestReviewPlanGroupsRepeatedWarningsAndPresentsActionableToolPhases`
  while canonical JSON retains full warning detail commit:89b100a.
- R45 [satisfied] the same phase-aware tool-plan regression plus human CLI
  explain assertions commit:89b100a commit:e6fe84e.

### Known gaps
- The feature intentionally remains opt-in/preview; promotion is a later
  release decision after production adoption evidence, not part of this spec.
- Conductor-side assignment/retry orchestration remains owned by Harne8. This
  repository delivers and verifies only the offline portable contract.
- Repository-wide assessment still lists 52 unobserved external consumers and
  artifact assurance lists 231 historical orphan warnings. Current changed
  contracts, declared artifacts and all three delivery targets have no error.

## 7. Final Report

### Delivered scope
Delivered the preview/opt-in convergent review flow end to end: canonical
semantic bundle preparation, immutable sealing, separate local/imported
attestations, deterministic verification and deltas, bounded criterion reuse,
legacy compatibility, CLI/MCP/schema parity, closeout integration and portable
offline orchestration contracts.

### Files and modules changed
- Engine: `pose-mcp/internal/pose/review_bundle.go`, review closeout and delivery
  surface logic with regression suites.
- Interfaces: native CLI commands, read-only MCP projections, schemas v1,
  usage signals and catalog goldens.
- Distribution: POSE manual, feature/review workflows, skills, docs-site,
  English/pt-BR locales and embedded scaffold copies.
- Governance/evidence: this spec, ADR, decision log, changelog, validation
  matrix, assessment/state refreshes, reports and structured results.
- E2E: `tests/e2e/review-bundle/run.sh` for source and installed binaries.

### Validation executed
- `go -C pose-mcp test ./...`, `go vet ./...`, focused race tests and
  `go build -trimpath`: SUCCESS.
- `pose validate --strict --module pose-mcp` persisted to both canonical result
  files: five checks passed, zero failed/skipped/errored.
- Source and installed `tests/e2e/review-bundle/run.sh`: SUCCESS.
- `govulncheck ./...` and gitleaks history scan: SUCCESS.
- `pose artifact-check --strict`: 64/64 current artifacts, zero errors.
- `pose surface-check --strict`: three targets, five current results, zero
  findings.
- `pose assess integrate`, `pose assess tech-debt` and
  `pose assess discover --update-state`: completed with only the documented
  repository-wide inventory warnings.
- `pose skills-check --strict` and `pose knowledge-check --strict`: SUCCESS.

### Residual risks
- Production adoption may reveal additional semantic projection classes; the
  closed classifier intentionally fails sealing until each is governed.
- External signature providers remain optional. Any new algorithm or dependency
  requires a schema/ADR revision and fresh security validation.
- The preexisting scaffold embed parse panic remains an accepted construction
  invariant, outside ordinary request processing.

### Follow-ups
- [wont-do: Harne8 owns Conductor service delivery outside this repository]
  Implement Conductor assignment/retry orchestration against the portable
  bundle/attestation contract.
- [wont-do: compile-time scaffold embed parsing is a construction invariant]
  Replace the preexisting scaffold initialization panic as runtime debt.
