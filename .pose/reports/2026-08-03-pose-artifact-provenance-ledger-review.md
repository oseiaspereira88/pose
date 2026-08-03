# Review: pose-artifact-provenance-ledger

## Review summary

- Decision: approved after remediation.
- Change type: Go feature, public CLI/MCP contract and operational governance.
- Scope: `spec:pose-artifact-provenance-ledger`.
- Review execution: `agent:independent-provenance-review-1`.

## Rules applied during review

- `.pose/workflows/review.md`: separate review, findings remediation and immutable decision.
- `.pose/rules/backend-go.md`: deterministic data models, bounded subprocesses and test coverage.
- `.pose/rules/security.md`: revision validation, path confinement, symlink escape and data minimization.
- `.pose/rules/documentation-style.md`: CLI, workflow, template, locale and manual consistency.
- `.pose/rules/delivery-evidence.md`: independent Git witness and strict closeout evidence.
- `.pose/rules/knowledge-governance.md`: consulted `knowledge:contract-baseline-handoff`; no transient handoff remains.
- Rules not applicable: frontend React and Kubernetes; neither domain changed.

## Checks and evidence

- Focused artifact, Git, report-history, index and MCP tests: passed.
- `go test ./... -count=1`: passed after regenerating assessment mirrors.
- `pose check --strict`: passed.
- ready and strict spec lint: passed.
- `pose validate --strict --module pose-mcp --report`: passed, including `go vet`.
- Embedded scaffold parity: passed.
- `pose artifact-check` over `5403f56..HEAD`: zero error or critical findings.
- `govulncheck ./...`: no vulnerability reachable from POSE code.
- `pose assess integrate` and `pose assess tech-debt`: reviewed; their project-wide baseline findings are unrelated to this diff and no new debt marker was added.

## Contracts and compatibility

- Added commands and report fields without removing or changing existing syntax.
- Legacy narrative specs remain readable and receive an advisory migration finding.
- Strict artifact closeout activates only at the policy adoption boundary.
- Claims and Git observations remain separate witnesses in one schema-versioned graph.
- CLI and `pose_delivery_integrity` use the same persisted schema and project-scoped path rules.

## Findings

- Resolved, medium: artifact parsing consulted `.` and could depend on process working directory. Split syntactic validation from root-aware filesystem validation.
- Resolved, medium: backfill output had conflicts but no explicit confidence, and accepted omission of `--from-git`. Added deterministic per-spec confidence and a mandatory selector.
- Resolved, medium: the planned real-Git fixture did not cover rename-with-edit and removal together. Added a regression using Git rename detection plus deletion.
- Resolved, low: malformed report history lacked an explicit negative regression. Added proof that invalid JSONL cannot fabricate a change set.
- Resolved, low: refreshed assessment data drifted from the embedded scaffold. Regenerated the mirror and reran the full suite.
- Open findings: none.

## Security and operability

- Git is invoked with structured arguments, bounded output and validated revisions; specs never provide executable shell text.
- Absolute paths, traversal, globs, directories and escaping symlinks are rejected at their relevant gates.
- The ledger persists paths and digests, never file contents or unrestricted command output.
- Backfill is dry-run by default and cannot edit without both apply and confirmation flags.

## Decision

Approved. No blocking finding remains. The implementation is eligible for an
immutable spec review attempt and guarded lifecycle close.
