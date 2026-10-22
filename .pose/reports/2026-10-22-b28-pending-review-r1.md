# B28 separate source review R1

Session 01a16230-1d86-7b82-84e2-bd4744309c62. Separate Codex fallback after Claude weekly limit; task-designated gpt-6.1-sol, same vendor, no human/cryptographic/cross-vendor independence.

**Decision: code approved for source review at `a0d40eaf7e707359110f04132a75d826c389ee06`.** No remaining medium, high, or critical source finding identified. This does **not** approve the final seal, evidence collection, merge readiness, or lifecycle closeout.

Reviewed the full 42-file `origin/main...HEAD` diff, spec `pose-distributed-instructions-parity`, all six committed B28 review reports and their evidence summaries, and remediation after the latest changes-requested review, b28r6. No B28-specific historical approved attestation was found. The historical approval referenced by earlier reports concerns another spec and supplies no approval for this change.

**Rules applied during review**

- `pose-review` skill and `.pose/workflows/review.md`: scope, historical findings, consumer integration, effective criteria and tool dispositions.
- `backend-go.md`: input validation, error propagation and CLI/MCP consumers.
- `security.md`: identifiers, paths, availability claims and evidence integrity.
- `documentation-style.md`: distributed commands, messages and mirrors.
- `delivery-evidence.md`: separation of implementation, executed verification and pending collection.
- `knowledge-governance.md` and `_base-recurrence.md`: historical decisions, follow-ups and recurring validation gaps.

Change type: mixed bugfix, distributed documentation and regression coverage. Frontend and infrastructure runtime rules do not apply to this implementation.

**Source assessment**

The b28r6 blocker is addressed: `taskFieldSpellingError` rejects noncanonical case-folded spellings of operational and derived fields, and both selected-task `suggest` and `checkTaskMap` call it. The regression exercises both consumers. Source inspection supports rejection regardless of conflicting-field order; `strings.EqualFold` also covers relevant Unicode folding.

Earlier corrections remain present: root-module fallback and specificity, conflicting canonical module paths, duplicate JSON keys, explicit null/type checks, reserved output fields, validated rule identifiers, readable regular files, broken rule links, and confinement of the resolved `go.mod` file.

Production integration is intact: MCP dispatch calls `Store.Suggest`, which invokes CLI JSON rather than implementing a separate availability resolver. Existing conservative MCP restrictions remain explicit refusals. Availability discovery installs no extension or policy. Freshness instructions reuse the existing mechanism; migrate refuses apply without promising a future implementation.

The historical evidence shortcuts are materially constrained: instruction walkers propagate failures, and the mutation runner uses argv/cwd and requires a targeted Go test failure rather than treating any nonzero shell exit as success. These protections are principally local regressions; I did not execute their adversarial fixtures in this read-only session.

**Nonblocking findings and retained follow-ups**

- **Low — newest regression omitted from the dedicated integration filter.**
  Source: [validation-matrix.json](/tmp/pose-pending-b28/.pose/indexes/validation-matrix.json:2170), [regression test](/tmp/pose-pending-b28/pose-mcp/internal/cli/suggest_review_regression_test.go:347).
  Reproducer: compare that test name with the registered `-run` expression; it does not match. Minimal fix: broaden the expression. Until then, final collection must include the full Go suite.

- **Low — whitespace check remains red.**
  Source: [historical report](/tmp/pose-pending-b28/.pose/reports/2026-10-18-pose-b28-d7ebc300-independent-review.md:173).
  Executed reproducer: `git diff --check origin/main...HEAD` reports a new blank line at EOF. Minimal fix: remove the extra blank line.

The other documented low-risk follow-ups remain relevant, including human alias explanation, conservative MCP alias rejection, `unreadable` documentation, broken-reference diagnostics and residual “6.x” inventory wording. They do not recreate the successful omission of obligations that blocked b28r6.

**Executed verification**

- Canonical context resolved local authority without ambiguity or transfer.
- Prepared bundle and effective plan resolved: 15 criteria, 12 tools; component `pose-mcp` is `declared+observed`, with missing-owner warning.
- `skills-check --strict`: 22 skills, zero errors or warnings.
- `recurrence-check`: 403 records, zero flagged or flapping keys.
- CLI probes confirmed Go alias normalization, Kubernetes `extension-needed`, explicit unknown-domain status, traversal refusal and apply refusal.
- Read-only instruction scan: 136 Markdown visits, zero unconditional discovery commands.
- Four changed workflow/rule mirror pairs and the Portuguese manual matched their sources. The English manual difference is the instance contributor-mode section.
- Limited added-line secret-pattern scan: zero candidates.
- Working tree remained clean.

**Pending collection and final adjudication**

`index --verify` fails because generation inputs changed in specs and the validation matrix. This matches the disclosed post-merge collection gap; it must pass before final seal approval.

Current plan digest: `sha256:1d0f5f016d3bf73221502e29af8f9276419b0bb77890e2f9dac79ba4d40b8a4a`. Prepared bundle: `rvb-d8324ccb59d34df2`. Its attributed subject ends at `c64632d`; I additionally reviewed the merged working HEAD. These identifiers are provisional for final adjudication.

I did not execute Go tests, vet/build, mutation runs, install/update journeys or strict validation. Integration and tech-debt assessment commands persist artifacts, so they were deferred under the explicit read-only instruction, alongside artifact reconciliation and final surface evidence. Historical green runs are historical evidence. Seal, attestation, verification and closeout remain deferred.

Reviewer identity: OpenAI Codex, task-designated `gpt-6.1-sol`, separate execution and same vendor as the lead. Identity and execution separation are declared; no human, cryptographic, cross-vendor or proven cognitive independence is claimed. No source, Git, policy, attestation or lifecycle changes were made, and no GitHub messages or upstream drafts were sent.

# Final sealed adjudication — 2026-10-22

Actual separate session 01a16349-af4e-7c10-a93f-2ea7553bbb05; log /home/go/.local/state/secondary-agent/20261022T080535-codex-1239458. Original R1 decision above is unchanged.

**APPROVED** for the exact sealed subject [rvb-8356317c26979162](/tmp/pose-pending-b28/.pose/review-bundles/rvb-8356317c26979162.json), source HEAD `1ba6de34996cc1697be04dde999e8b1725181934`.

This is a semantic source and bundle review. Native completion gates remain pending parent attribution.

Verified identities:

- Bundle: `sha256:8356317c269791627a2cf7e835858b75a904f99a4baf413c4fc4c692b7c49a85`
- Plan: `sha256:17360b38b9526fdaf638b6aa4a5a02851630585b28885268659591be3f8a7db8`
- Implementation: `sha256:4b0d06d443c91dcc56ed3236b1c157c9a6bd7bd99f342950fe43bd8b1532db41`
- Scoped provenance: `sha256:19a6af3fe48a23899033067ddb610a977127afab9c905d68e1e353360080cbf1`

I independently reproduced the bundle, implementation, patch and tree digests. All 27 subject paths match their sealed current-byte digests. Fresh native bundle resolution produces the identical payload and digest, without blockers. Review followed attributed path segments; the contaminated commit range was not treated as exclusively B28.

The bundle contains **140 evidence entries: 139 current collected results plus one structural observation**. Canonical strict validation records **143/143 passing**, matching HEAD and scoped provenance.

**Criterion judgments**

| Criterion ID | Judgment and rationale |
|---|---|
| `assumption-integrity` | **Passed.** Filesystem availability is observed through regular-file/readability checks; missing, unreadable and unknown remain distinguishable. Integration discovery and incomplete debt coverage are not treated as proof of absence. |
| `backend-concurrency` | **Passed.** The delta adds synchronous parsing and filesystem inspection without shared mutable caches, goroutines or lifecycle changes. Current full-suite evidence supports regression safety. |
| `backend-contracts` | **Passed.** Existing valid task-map inputs remain supported; ambiguous spellings, duplicate keys and invalid types receive explicit refusals. Availability fields describe declared versus readable resources. |
| `backend-input-errors` | **Passed.** Nulls, malformed collections, conflicting canonical module paths, reserved fields and invalid rule identifiers cannot silently remove obligations. Both checker and selected-task suggest enforce exact operational field spelling. |
| `backend-integration-impact` | **Passed.** MCP dispatch reaches `Store.Suggest`, which invokes CLI JSON. Availability resolution therefore has one implementation. Existing conservative MCP argument refusals remain explicit. |
| `backend-observability` | **Passed.** Returned source/status/detail distinguish unavailable indices, invalid configuration, unmatched domains and unreadable references. Remediation is actionable without introducing sensitive logging. |
| `compatibility` | **Passed.** Valid canonical inputs retain support. Stricter rejection intentionally addresses ambiguous or malformed configuration; added status fields are additive. Apply remains unavailable, with corrected messaging. |
| `correctness` | **Passed.** Source handles root fallback, module specificity, aliases, availability and refusal paths coherently. Current full tests and dedicated integration evidence cover the latest spelling regression. |
| `design-causality` | **Passed.** Shared taxonomy and availability inspection address divergent discovery; freshness instructions reuse the delivered engine; migrate states present capability. Structural mappings appear below. |
| `documentation` | **Passed.** EN/pt-BR sources and scaffold instructions match runtime freshness and apply behavior. Historical documentation refinements remain explicitly owned follow-ups. |
| `operability` | **Passed.** Missing extensions receive installation commands; unreadable rules receive diagnostic context rather than installation advice. Unknown domains remain visible. |
| `requirements` | **Passed.** R1–R8 have relevant source behavior and current validation coverage, detailed below. |
| `scope` | **Passed.** Changes address distributed instructions, suggest availability/taxonomy, migrate messaging and regression coverage. No incidental policy adoption or unrelated runtime expansion. |
| `security` | **Passed.** Identifier validation, duplicate-key refusal, exact-field checks and resolved `go.mod` confinement protect obligation integrity. No new command interpolation, authorization bypass or secret exposure identified. |
| `solution-proportionality` | **Passed.** Text-only correction would leave runtime misreporting; independent consumer resolvers would duplicate behavior. Existing freshness, extension mappings and CLI adapters are sufficient. No new scheduler, installer or speculative plugin framework is introduced. Reconsider if consumers stop delegating or taxonomy ownership changes. |

Required `design-causality` mappings:

| Material fact | Basis and rationale |
|---|---|
| `SD-813aeaa2` — validation matrix | **R8.** Registers executable integration coverage and includes the operational-field regression, making current consumer behavior part of deterministic validation. |
| `SD-bc5a864b` — feature workflow | **D1 → R1/R2.** Uses existing `--if-stale` semantics at the prescribed execution point, eliminating unconditional rescanning without replacing freshness logic. |
| `SD-f906fe94` — delivery-evidence rule | **R1/R7.** Preserves closeout state refresh while permitting reuse when assessment inputs are unchanged. |

The cheapest formal shortcut would be adding `--if-stale` text while leaving distributed copies or runtime behavior unverified. Recursive source/installed inventories, mirror comparisons and current freshness/integration tests constrain it. The adversarial corpus includes `stale-assessment-reused`; B28’s unreadable-inventory regressions additionally reject silently incomplete scans.

**R1–R8**

| Requirement | Judgment |
|---|---|
| R1 | **Satisfied:** prescribed instructions use conditional discovery; current freshness integration verifies reuse/invalidation behavior. |
| R2 | **Satisfied:** distributed-tree inventories, scaffold comparisons, installed inventory and current install/upgrade journey checks cover delivery. |
| R3 | **Satisfied:** shared aliases preserve inference source; root/specific-module selection avoids indiscriminate Go routing. |
| R4 | **Satisfied:** declaration, readable installation, extension-needed and unavailable/unreadable states remain distinct; discovery installs nothing. |
| R5 | **Satisfied:** dry-run remains read-only; apply explicitly refuses without promising future implementation. |
| R6 | **Satisfied:** human/JSON availability agrees; MCP delegates supported inputs and explicitly refuses unsupported ones. |
| R7 | **Satisfied:** existing freshness, authorization and opt-in boundaries remain intact; mirrors share the intended instructions. |
| R8 | **Satisfied:** current full-suite and dedicated integration evidence exercise positive and negative behavior. Historical mutation results remain historical supplementary evidence. |

**Tool dispositions and evidence limits**

- **Required active tools:** supplied completed strict validation, artifact and surface outputs inspected. All sealed validation entries match canonical results. Surface findings are empty; artifact warnings do not affect subject paths.
- **Executed now:** canonical context resolution; fresh bundle resolution; digest/current-byte comparisons; `skills-check --strict` (**22 checked, zero errors/warnings**); recurrence-check (**405 records, zero flagged/flapping**, one resolved cluster); `index --verify` (**current**); whitespace inspection.
- **Recommended tools:** design and suggest outputs inspected. Integration assessment’s 63 unobserved-consumer gaps are discovery limitations, supplemented by actual consumer source inspection. Tech-debt assessment remains **INCOMPLETE** because two legacy spec sources are invalid; its covered scanned marker does not establish global completeness. Supplied component assessment was considered without claiming platform-wide completeness.
- Supplied final structural check ends in **SUCCESS with 29 warnings**; history check ends in **SUCCESS**. Requested collector outputs were completed and readable before this decision.
- **Deferred:** native `review-check`, `closeout-check`, attestation, verification following attribution, and lifecycle closeout. No passes fabricated.
- Earlier probes described in R1 are supplied historical execution evidence; I did not claim to rerun them now. Go tests, mutation runs and install/update journeys were not executed by this review session.

**Findings and rules**

No critical, high or medium source findings remain. The prior selector omission and historical EOF issue are fixed.

**Low, nonblocking editorial finding:** [R1 report](/tmp/pose-pending-b28/.pose/reports/2026-10-22-b28-pending-review-r1.md:32) contains trailing whitespace at lines 32, 33, 36 and 37, so broad `git diff --check origin/main...HEAD` remains red. Remove those spaces during report cleanup. This excluded report does not alter the approved semantic payload.

Historical low follow-ups retain their existing `@core` ownership and review dates; none was silently closed, adopted or reassigned.

Applied `pose-review`, the review workflow, backend-go, security, documentation-style, delivery-evidence, knowledge-governance and recurrence rules. Change type: mixed bugfix/documentation/regression coverage. Frontend and infrastructure runtime rules are inapplicable.

Reviewer identity: **OpenAI Codex, task-designated gpt-6.1-sol, separate same-vendor execution**, following the reported Claude weekly-limit fallback. Execution separation is declared; exact model routing, cryptographic identity, human confirmation, cross-vendor independence and cognitive independence are not independently established.

No repository writes, Git changes, attestations, closeout, release, policy adoption, Harne8/shared-checkout changes or external messages were performed.


# Fresh B30+B28 composition adjudication — 2026-10-22

Actual separate session 01a16461-775e-7943-a5ce-061174f37bd8; log /home/go/.local/state/secondary-agent/20261022T131111-codex-2440455. This approves the new composed subject, not merely retained historical closure.

**APPROVED** for the exact new sealed subject [rvb-e232a12e912dcde8](/tmp/pose-pending-b28/.pose/review-bundles/rvb-e232a12e912dcde8.json).

This is a fresh semantic approval of the B28/B30 composition, independent of the retained historical approval.

| Identity | Verified value |
|---|---|
| Bundle digest | `sha256:e232a12e912dcde8f3fe0999f140f80eae070e9231f7ac867e3ab62e5969c602` |
| Source HEAD | `ad0f23c5636571a98e6e88ac663d7b72de25f280` |
| Scoped provenance | `sha256:1a86ae37e8f13e97907f7acb03dc544561628d3260a4fd74bc84742b31f8fa4d` |
| Plan digest | `sha256:17360b38b9526fdaf638b6aa4a5a02851630585b28885268659591be3f8a7db8` |

I reproduced the bundle, implementation, patch and tree digests. All 27 subject files match their sealed byte digests. Fresh read-only bundle resolution returned the identical payload and digest without blockers.

The attributed scope contains 21 commits and 69 observed paths, distinct from its 98-commit provenance range. I compared it with current `origin/main` (`47d1cf04`) and inspected `c31cf389..ad0f23c` as B30 integration—not as exclusively B28 work. B28 runtime code remains unchanged from `c31cf38`; imported B30 runtime code matches `11bfbbf`. Shared manuals combine both changes. The matrix preserves the complete parent check union without changing shared check definitions.

| ID | passed/notapplicable/finding | Reason |
|---|---|---|
| assumption-integrity | passed | Exact bytes and current evidence agree. Incomplete debt coverage, heuristic integration gaps and declared identity remain explicit limitations. |
| backend-concurrency | passed | B28 adds synchronous inspection without new shared mutable state or goroutines. Collector evidence includes full tests and targeted race checks. |
| backend-contracts | passed | Canonical valid inputs remain supported; ambiguous spellings, duplicate keys and invalid types receive explicit refusals. B30 preserves shared result parsing and trailer interpretation. |
| backend-input-errors | passed | Null/type validation, reserved fields, canonical-path conflicts and rule identifiers prevent silent loss of obligations. Checker and suggest share operational-field spelling enforcement. |
| backend-integration-impact | passed | MCP delegates through `Store.Suggest` to CLI JSON. B30’s parser and attribution changes compose without replacing B28’s resolver. Both dedicated checks remain registered and pass. |
| backend-observability | passed | Source/status/detail expose invalid maps, unmatched domains and unreadable resources. B30 distinguishes candidates from associated evidence and escapes control characters in human output. |
| compatibility | passed | Status fields are additive; stricter configuration refusals address ambiguity. Existing MCP restrictions remain explicit. Migrate still refuses apply. |
| correctness | passed | Root fallback, module specificity, aliases, availability and refusal paths remain coherent. Current strict evidence covers the operational-field regression. |
| design-causality | passed | Shared normalization and availability inspection address divergent discovery; conditional instructions reuse existing freshness behavior. Every material structural fact is mapped below. |
| documentation | passed | EN/pt-BR workflow/rule mirrors agree; Portuguese manuals match. Combined manuals retain both freshness/apply guidance and B30 evidence-binding limitations. |
| operability | passed | Missing extensions receive concrete installation guidance; unreadable rules receive diagnostics. Discovery performs no automatic installation or adoption. |
| requirements | passed | R1–R8 retain relevant source and current evidence: freshness, distributed installation, aliases, availability, migrate refusal, consumer behavior and negative cases. |
| scope | passed | Full attributed scope is reconciled. B30 integration is separately identified; derived histories and historical reports do not substitute for current implementation approval. |
| security | passed | Identifier checks, duplicate/exact-key refusals and resolved `go.mod` confinement preserve obligation integrity. No new dependency, authorization bypass or command interpolation identified; limited subject secret-pattern scan found no candidates. |
| solution-proportionality | passed | Text-only changes would leave runtime misreporting; separate consumer resolvers would duplicate logic. Existing freshness, extension mappings and CLI delegation suffice without new installation machinery. |

Required structural mappings—each uses one basis from B28:

| ID | single R or D basis | Causal reason |
|---|---|---|
| `SD-813aeaa2` | R8 | The validation matrix registers executable behavior checks, retains B28’s spelling regression and adds the separately approved B30 check without dropping either parent’s coverage. |
| `SD-bc5a864b` | R1 | The feature workflow invokes `discover --if-stale` at existing execution points, removing unconditional rescans while retaining invalidation when inputs change. |
| `SD-f906fe94` | R1 | The delivery-evidence rule retains closeout state refresh while allowing unchanged component assessments to be reused. |

The cheapest formal shortcut is correcting command text while leaving installed copies or runtime behavior divergent. Recursive source/installed inventories, mirror comparisons, install/upgrade checks and current functional regressions constrain that shortcut. The adversarial corpus includes `stale-assessment-reused`; unreadable-inventory tests additionally reject silently incomplete coverage.

These conclusions would be disproved by a changed subject digest, dropped selector, installed-copy divergence, consumer departure from CLI delegation, ambiguous configuration being accepted, or current results losing their HEAD/scoped binding. Filesystem availability is an observation at inspection time, not a permanent guarantee.

Current evidence and tool dispositions:

- **Collector execution:** `/tmp/pose-b28-composed-validation.log` finishes **144/144, SUCCESS**. Canonical results name `ad0f23c`, match the matrix digest and B28 scoped provenance, and agree with all **140 collected sealed entries**, plus one structural entry.
- **Completed collectors:** artifact and surface JSON files parse completely. Surface findings are empty; artifact warnings do not intersect the sealed subject. Index, discovery, debt and integration logs were inspected.
- **Limits:** debt assessment remains **INCOMPLETE** for two invalid legacy sources. Integration discovery reports 63 heuristic unobserved-consumer gaps; it is not comprehensive certification.
- **Executed by this reviewer:** read-only context/bundle resolution, Git comparisons, digest/evidence reconciliation, mirror inspection, limited secret-pattern scan, `index --verify` (**current**), `skills-check --strict` (**22, zero errors/warnings**) and recurrence-check (**408 records, zero flagged/flapping keys**). `git diff --check origin/main...HEAD` passed.
- **Runtime probe limitation:** the installed `pose 7.4.1-dev` suggest command returned older output without B28 status fields. I did not treat that binary as proof of the reviewed runtime. Collector Go tests compile repository source; their `runPose` helper invokes `Main` directly.
- **Deferred completion:** parent attribution of this actual decision, then native `review-check`/closeout verification. Neither spec needs another lifecycle transition.

Rules applied during review: `pose-review`, review workflow, backend-go, security, documentation-style, delivery-evidence, knowledge-governance and recurrence guidance. Change type is mixed bugfix/documentation/regression coverage. Frontend and infrastructure runtime rules are inapplicable.

No new blocking finding identified. Existing low follow-ups remain open with their recorded ownership and review dates. Historical approvals, reports and mutation runs remain historical.

Reviewer: OpenAI Codex, task-designated `gpt-6.1-sol`, this separate same-vendor session under the reported Claude-limit fallback. No independently verified session identifier is available here. Identity and separation are declared; no human, cryptographic, cross-vendor or proven cognitive independence is claimed.

No repository writes, Git mutations, policy changes, attestations, lifecycle changes or external messages were performed.
