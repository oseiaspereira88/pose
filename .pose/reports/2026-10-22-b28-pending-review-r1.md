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