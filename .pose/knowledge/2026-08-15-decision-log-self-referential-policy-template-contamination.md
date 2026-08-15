---
type: decision-log
slug: self-referential-policy-template-contamination
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-15
last_reviewed_at: 2026-08-15
expires_at: 2026-09-14
source_refs:
  spec: ""
  workflow: "bugfix"
  commands: ["go -C pose-mcp generate ./internal/scaffold", "go -C pose-mcp build ./...", "go -C pose-mcp vet ./...", "go -C pose-mcp test ./...", "gofmt -l pose-mcp", "cd mcp-enforce && go build ./... && go test ./..."]
  external_sources: [{url: "https://github.com/oseiaspereira88/pose/issues/17", accessed_at: "2026-08-15"}]
---

# decision-log: self-referential-policy-template-contamination

## Context

Issue #17 (reported against pose-mcp/pose CLI 1.2.0): `.pose/policy/delivery.json`
and `.pose/policy/artifacts.json` on a target project seeded by `pose install`/
`pose update --force` carried `roots`/`governed_roots` that were literal paths
into **pose-mcp's own source tree** (`pose-mcp/internal/cli`,
`pose-mcp/internal/mcpserver`, `pose-mcp/internal/pose`, entrypoint
`pose-mcp/cmd/pose/main.go`), not the target project's. Every delivery-integrity
and artifact-contract finding for that project's own specs went silent — a
finished spec's `review bundle --seal` failed with "no immutable attributed
change set exists" because the graph never resolved a `DeliveryTarget` for
that project's own module paths.

Root cause: `pose-mcp/internal/scaffold/gen/main.go` syncs `.pose/policy/`
wholesale into the embedded scaffold via `distpolicy.IncludedPoseSubtrees`,
which listed `"policy"` with no per-file exception. `delivery.json`/
`artifacts.json` are pose-dist's own dogfooding config (adopted 2026-08-03/04,
close to the 1.2.0 cut), not a generic template — the wholesale sync copied
this repository's live self-referential values into every downstream install.

This is the same disease as [[project-agnostic-assessment-evidence]] (2026-08-03,
`pose-project-agnostic-assessment-engines`): pose-mcp's own identity leaking
into what an installed instance evaluates about a *different* project. That
earlier fix scoped the assessment **engines** (discovery/integration/tech-debt
scanners) to observed evidence under the authorized project root, with no
hardcoded pose-dist/Harne8 names. It did not touch the scaffold **template**
pipeline (`internal/scaffold/gen`, `internal/scaffold/distpolicy`), which is a
separate mechanism carrying separate self-referential data (JSON policy
content, not scanner logic) — the earlier fix's invariant ("no hardcoded
identity from the producing repository") was never extended there.

## Current state

Fixed at the source, not just the symptom:

- `distpolicy.SelfReferentialPolicyFiles` (`internal/scaffold/distpolicy/
  distpolicy.go`) excludes `delivery.json`/`artifacts.json` from the wholesale
  `.pose/policy` allowlist match in `IsIncluded`.
- `distpolicy.NeutralPolicyTemplates()` is the single source of truth for what
  ships instead: schema-valid, `enabled: false`, empty `roots`/
  `governed_roots` — an explicit, obvious placeholder (issue #17's suggestion
  (a)), not a silently-broken default. Both `gen/main.go` (writes it into the
  embedded scaffold) and `scaffold_test.go` (asserts the embedded copy still
  matches it) call the same function, so generator and drift guard cannot
  diverge from each other the way the pre-existing `IncludedTopLevel`/
  `IncludedPoseSubtrees` split was explicitly designed to prevent for
  everything else.
- `pose-mcp/internal/scaffold/dist/.pose/policy/{delivery,artifacts}.json`
  regenerated via `go generate ./internal/scaffold`; drift test green.
- `pose doctor` gained two detectable (not auto-fixable — the correct roots
  are project-specific) findings, `policy.delivery-roots`/
  `policy.artifact-roots`: warn when every configured root fails to resolve
  on disk, naming issue #17 in the hint. This is the mitigation for instances
  **already** contaminated by 1.2.0 before this fix — `pose install`/`update`
  never overwrite an existing policy file, so the template fix alone cannot
  self-heal a project that already seeded the bad content; the operator still
  has to hand-edit the JSON, but doctor now tells them why and where.
- Achado 2 of the same issue (pose_validate_approve unusable over stdio) was
  already correctly documented in
  `.pose/adr/2026-07-19-mcp-conductor-harness-trust-boundary-for-safe-validation-orchestration.md`
  and `docs-site/docs/mcp.md`; the actual gap was a missing cross-reference
  from the runtime error message and `mcp-enforce/README.md` to the already-
  existing local alternative (`pose review attest`/`auto-attest`). Both now
  name it explicitly.

Regression coverage: `distpolicy_test.go` (new package, exclusion +
schema-validity of the placeholder against the real `pose.LoadDeliveryPolicy`/
`LoadArtifactPolicy` loaders), `scaffold_test.go` (drift guard special-cases
the two placeholder files instead of false-flagging them as "extra"),
`doctor_remediation_test.go` (contaminated-roots detection and a
no-false-positive case with genuinely-existing roots),
`validate_orchestration_test.go` (stdio-specific approval diagnostic names
both the cause and the alternative).

## Next checks

- `pose validate --tolerant --module pose-mcp/internal/scaffold` and
  `--module pose-mcp/internal/mcpserver` before closeout.
- `pose check --strict` at release-closeout time, once this change is
  committed and reported per the repo's provenance sequence (see
  `[[pose-gate-closeout-procedure]]`-equivalent local convention: one commit,
  `artifact-check`, `pose report`, `pose validate --json`, in that order).
- Version bump to 1.2.1 and changelog entry referencing issue #17.

## Risks

- The `pose doctor` heuristic (all configured roots missing → warn) is
  conservative by design: a project with a genuinely-legitimate root that
  happens not to exist yet (e.g. a module not yet created) would only trigger
  if *every* root in the file is currently missing. A partially-contaminated
  file (some real, some inherited) will not be flagged — acceptable, since a
  false negative there still leaves the existing delivery-integrity checks as
  the backstop, and a false positive would train operators to ignore the hint.
- Instances that ran `pose install`/`pose update --force` on 1.2.0 and already
  have the contaminated files on disk are not auto-repaired by upgrading to
  1.2.1 — only diagnosed. This is a deliberate scope boundary (`pose install`/
  `update` never overwrite existing policy — a stronger invariant than this
  bug), not an oversight; document it in the 1.2.1 changelog so operators
  don't expect a silent fix.

## Next owner

`@pose-maintainers` — no open follow-up beyond the next checks above.

## References

- Issue: https://github.com/oseiaspereira88/pose/issues/17
- Precedent: [[project-agnostic-assessment-evidence]]
- ADR (achado 2 stdio boundary):
  `.pose/adr/2026-07-19-mcp-conductor-harness-trust-boundary-for-safe-validation-orchestration.md`
- Changed files: `pose-mcp/internal/scaffold/distpolicy/{distpolicy.go,distpolicy_test.go}`,
  `pose-mcp/internal/scaffold/gen/main.go`,
  `pose-mcp/internal/scaffold/scaffold_test.go`,
  `pose-mcp/internal/scaffold/dist/.pose/policy/{delivery,artifacts}.json`,
  `pose-mcp/internal/cli/doctor.go`, `pose-mcp/internal/cli/doctor_remediation_test.go`,
  `pose-mcp/internal/mcpserver/server.go`,
  `pose-mcp/internal/mcpserver/validate_orchestration_test.go`,
  `mcp-enforce/README.md`.
