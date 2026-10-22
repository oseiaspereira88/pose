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
