# B01 disclosure qualification — 2026-10-15

## Scope and reconciliation

Base main: `1f3244614d56f377d6d17b7e19ba5dd24e0285ac` (#181). #178 published
B04's native closeout; #179 closed action-answer. #181 published its final
verification and exact Security false-positive diagnosis. B01 remains partial
until the existing `pose-review-disclosure-requires-proof` reaches native
closeout. B02 is next on the preferred DAG chain, after merge; independent
roots do not acquire artificial dependencies. Administrative authoring closeouts
and conditional B43 are not functional implementation.

This delivery reuses the #158 implementation in `verifiedAuthorityBlockers`.
No new authority, policy, migration, release or persistent service is introduced.
R1–R4/D1 and the original `pose start` record remain unchanged. Native start
status classifies every node as recorded-before-managed-execution; no second
start is applied. The start preview reports no dependency wait and refuses a
new start because the spec is already in-progress, as expected.

## Qualification

Built the current source with Go 1.26.9. `assess discover --if-stale --component
pose-mcp`, `assess integrate`, `assess tech-debt` and `assess design --spec
pose-review-disclosure-requires-proof` executed before review. Related knowledge:
`knowledge:review-assurance-requires-cryptographic-proof` and
`knowledge:action-answer-proof-and-historical-time`. Their original dated
limitations remain historical; B04 now owns the implemented temporal/reuse
contract, with live pins/grants and no historical authorization receipt.

Executed domain regressions passed: invalid/malformed/missing signatures,
unsigned decision/claim changes, wrong subject/algorithm, untrusted key/revoked
pin, supplied-bundle mismatch, valid identity with invalid approval semantics,
all shared gate consumers, and invalid human confirmation. Renamed the last
test to `TestReviewAssuranceInvalidProofCannotVerifyHumanConfirmation` so the
existing `ReviewAssurance` integration selector includes it.

Added `TestReviewAssuranceProofCLIAndMCPConsumers`: signed positive control,
damaged signature, missing proof and valid signature with missing criteria.
It calls the CLI verify/review-check/closeout-check in JSON and human form,
and the real HTTP JSON-RPC `pose_review_bundle` and `pose_closeout_state` tools.
All wire assurance fields match the domain result; valid identity remains
verified when semantic criteria prevent approval. Empty omitted slices are
normalized by JSON round-trip for comparison. Directly seeded synthetic records
exercise the read path independently of approving-write admission (B02).
The integration test passed with `go test ./internal/cli -run
TestReviewAssuranceProofCLIAndMCPConsumers -count=1`.

Targeted race and current global native strict validation results will be
recorded below when complete. Historical failed validation from #158 remains
preserved and is not current evidence. CI and Security must finish on the final
published head.

## Rules applied during review

- Change type: qualification of a Go security bugfix, integration tests and governance documentation.
- Workflows: `.pose/workflows/feature.md`, `.pose/workflows/review.md`.
- Security: authenticate signed bytes and current issuer trust before disclosing identity; include negative tests without exposing credentials.
- Delivery evidence: require native strict execution, artifact provenance and composed-capability evidence.
- Documentation style and knowledge governance: preserve original baseline/history and distinguish implementation from formal closeout.
- Backend-go extension is not installed; `pose suggest feature --path pose-mcp` selects security/documentation-style. No unrelated extension adoption.
- Frontend, Kubernetes and release publication rules are outside this scope.

## Review and closeout

Pending current validation and required different-actor judgments. No mechanical
preview constitutes approval. Identity/separation are declarations unless the
native verifier proves otherwise. No maintainer conclusion is inferred from
authorization to implement. B01 delivery_complete remains false until both
specs are terminal; no next unit starts before this PR merges.
