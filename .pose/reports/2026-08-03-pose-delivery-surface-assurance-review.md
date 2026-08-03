# Review: pose-delivery-surface-assurance

## Review summary

- Decision: approved after remediation.
- Scope: `spec:pose-delivery-surface-assurance`.
- Review execution: `agent:independent-surface-review-1`.
- Change type: Go feature, public CLI/MCP contract and governance distribution.

## Rules and evidence

- Applied backend Go, security, delivery-evidence, delivery-surface,
  documentation and review rules; frontend and Kubernetes rules were not
  applicable because no product frontend or cluster resource changed.
- Focused delivery/roadmap/trace/result tests, the full Go suite, `go vet`,
  catalog golden, scaffold parity, strict POSE checks, four-class module
  validation, artifact/surface assurance and `govulncheck` passed.
- Integration and technical-debt assessments were reviewed. Their project-wide
  baseline findings are unrelated to this diff; no new debt marker was added.

## Findings

- Resolved, high: delivery edges initially trusted declared artifacts without
  requiring a matching Git observation. Restricted composition provenance to
  paths witnessed by attributed immutable change sets.
- Resolved, medium: generated evidence/index commits made otherwise current
  validation stale. Split the source-provenance digest from the full graph
  input digest and bound validation to claims plus change sets.
- Resolved, medium: nested JSON result paths failed when the parent directory
  did not exist. Added confined directory creation and regression coverage.
- Resolved, medium: `deferred-integration` passed trace validation but still
  asserted delivery and could satisfy a roadmap criterion. It now creates only
  a `defers` edge and remains ineligible for cut acceptance.
- Resolved, medium: final artifact declarations were compared independently
  with every earlier immutable implementation snapshot. Reconciliation now
  preserves all snapshots while evaluating cumulative observed coverage.
- Open findings: none.

## Security and compatibility

- Roadmaps reference registered checks or confined reports and cannot execute
  embedded commands. Paths and MCP reads remain project-scoped.
- Legacy specs and roadmaps stay readable under staged warnings; adopted specs
  fail closed on missing provenance, composition, freshness or reachability.
- Existing validation checks run unchanged when `evidenceClass` is omitted.

## Decision

Approved. The three declared delivery targets have current provenance-bound
evidence, and no blocking review finding remains.
