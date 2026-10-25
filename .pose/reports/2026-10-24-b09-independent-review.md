# B09 — separate code review

Date: 2026-10-24. Spec: spec:pose-action-waiver-and-supersession-authority.
Reviewer: agent execution `/root/b09_review`, OpenAI GPT-6.1-sol, reasoning medium,
separate execution of the same vendor. Identity/separation are declared; no
cross-vendor or cognitive-independence claim, human adoption or cryptographic
identity proof is asserted. Claude review was attempted read-only but the
organization disabled subscription access; no Claude review occurred.

## Subject and rounds

Baseline: 27c5be7. Round 1 inspected the working implementation; round 2 inspected
its corrected code, now committed in 04e2bec. These were code reviews, not sealed
bundle certifications. Final mechanical evidence is collected separately.

Round 1 requested changes with one high finding F1: the signed successor ID
included RequestedAt but neither CLI nor MCP preserved the preview timestamp.
External signing across a second changed the ID, so legitimate supersession
failed. No unauthorized dispensation was demonstrated.

Round 2 approved the examined code, F1 resolved. `--requested-at` / `requested_at`
preserve the prepared timestamp and the domain validates RFC3339. Positive CLI
and MCP tests exercise an earlier fixed timestamp, a real schema-2 signature and
predecessor/successor lineage. Changing the timestamp/material/requester still
changes the bound identity. This timestamp is metadata and never substitutes for
the retirement claim's acceptance clock. No additional actionable finding was
identified.

## Judgments returned by the reviewer

| Criterion | Conclusion |
| --- | --- |
| assumption-integrity | F1 resolved: prepared identity can be explicitly preserved. |
| backend-concurrency | Predecessor revision, policy, key and authority are revalidated under lock; cross-journal convergence remains B10. |
| backend-contracts | Both transports carry required operation and successor bindings without weakening proof. |
| backend-input-errors | Invalid timestamps and divergent bindings are refused in the domain; no permissive fallback observed. |
| backend-integration-impact | Positive signed supersession controls exercise real CLI and MCP transports. |
| backend-observability | Operation, actor, proof, lineage and explicit partial-opening errors remain visible. |
| compatibility | Stricter semantics and release assessment are documented; no automatic adoption. |
| correctness | F1 corrected and targeted regressions pass. |
| design-causality | Distinct namespace/schema prevent answer-to-retirement replay; timestamp preservation addresses its actual cause. |
| documentation | Manuals and catalog document prepared identity, historical trust and B10 boundaries. |
| operability | External signing preserves successor identity across preview/apply. |
| requirements | No additional divergence from R1–R8 observed in examined paths. |
| scope | Changes remain in lifecycle, required transports, docs and regression coverage. |
| security | No further cheap bypass identified; operations retain distinct signed bindings and MCP remains proof-only. |
| solution-proportionality | Reuses existing identity/time verifiers and timestamp contract; no new issuer, session receipt or multi-journal transaction. |

## Executed evidence and limits

The reviewer ran `go test ./internal/pose ./internal/cli ./internal/mcpserver
-run ActionLifecycle -count=1` successfully in both rounds; `git diff --check`
passed in round 2. The reviewer did not certify final full-suite, race, vet/build,
scanners, native validation or sealed review gates in those rounds.

The primary's assessments report 63 unobserved integration consumers and a debt
scan incomplete because two existing specs are invalid. Those are coverage
limitations, not newly accepted B09 defects or full integration evidence.
Historical role authorization, hardware provenance, arbitrary filesystem writers
and cross-journal recovery remain the documented boundaries.

## Rules applied during review

Bugfix/security contract change. Read `.pose/workflows/review.md`,
`.pose/rules/security.md`, `.pose/rules/backend-go.md`,
`.pose/rules/documentation-style.md` and
`knowledge:action-answer-proof-and-historical-time`. Delivery evidence is required
before claiming completion. No frontend or infrastructure behavior changed.
Security takes precedence over declared labels and transport input. Native
verify/review-check and closeout remain separate required gates.

## Final evidence follow-up

The same separate reviewer directly checked the final native log and JSON bound
to HEAD `db37c8b`: 149/149 SUCCESS, full pose-mcp build/test/vet, full race and
final lifecycle race, vulnerability scan without findings, two-commit secret scan
without leaks, and strict check SUCCESS with 30 preserved historical warnings.
The reviewer confirmed this report faithfully records both rounds and maintained
technical approval without additional findings or newly accepted residual risk.

The 16th observed criterion, cross-component-integration, was approved for the
changed scope: signed CLI/MCP controls exercise the common domain and catalog;
docs-site changes are documentary, with no runtime integration change. Its
appearance in the matrix is not a site build proof because no checks are
registered for that component. Index/artifact/surface and the later sealed review
and closeout gates remain separate from this evidence follow-up.
