---
slug: artifact-claims-stop-at-the-section-end
status: done
created_at: 2026-09-20
completed_at: 2026-09-20
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:artifact-claim-boundary
---

# Spec: an artifact claim list ends where its section ends

## 1. Intent

### Goal

`ParseArtifactClaims` leaves the `### Artifacts` section only on the next `### `
heading. A `## ` heading does not end it, so the first `- ` list after it is parsed
as artifact claims — and the scaffold `pose new-spec` writes puts `## 4. Tasks`,
whose checklist is exactly such a list, right after the Technical Plan.

### Business value

A spec that declares artifacts and has no further `###` subsection makes every
task line a malformed claim. The failure is not local: `roadmap-check` and
`artifact-check` refuse the whole run with
`malformed artifact claim "- [ ] <task text>"`, so one spec's layout takes out a
repository-wide gate. Reproduced on 2026-09-20 with a spec written from the
scaffold, which is the layout the tool itself suggests.

### Constraints

Preserve every currently valid claim and the exact claim syntax. Specs whose
Artifacts section is followed by `### Delivery targets` must parse identically; a
fix that changed their claims would rewrite attributed provenance.

### Non-goals

Changing the scaffold to avoid the layout. The parser's section boundary is the
defect; moving the scaffold around it would leave the trap in place for anyone who
writes a spec by hand.

## 2. Requirements

- R1: The `### Artifacts` section ends at the next heading of any level, not only
  at another `### `.
- R2: Claims for every existing spec in this repository are byte-identical before
  and after the fix, proven by comparing parsed claims across all specs.
- R3: A spec whose Artifacts section is followed directly by `## 4. Tasks` parses
  its claims and no task line, and the negative case is pinned by a test.

## 3. Technical Plan

`ParseArtifactClaims` in `delivery_integrity.go` breaks on
`strings.HasPrefix(line, "### ")` while inside the section. Break on any heading
prefix instead, and cover both layouts plus a whole-repository claim-equality test
so R2 is measured rather than asserted.

### Artifacts

- created: .pose/specs/2026-09-20-artifact-claims-stop-at-the-section-end.md
- created: .pose/changelogs/unreleased/artifact-claims-stop-at-the-section-end.md
- created: pose-mcp/internal/pose/artifact_claims_boundary_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:artifact-claim-boundary module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

The claim parser is the composed capability `artifact-check` and `roadmap-check`
read through, which is where the defect surfaced.

### Rollout and reversal

A boundary condition in one parser. Reverting is reverting that condition; no
stored claim changes, because R2 requires the parsed set to be unchanged.

## 4. Tasks

- [x] Pin the current claims of every spec in the repository as a baseline.
- [x] Reproduce the task-line claim from a scaffold-shaped spec.
- [x] End the section at any heading and prove the baseline is unchanged.

## 5. Decisions

Recorded as its own spec rather than folded into
`check-strict-verdict-names-its-mode`: both were found in the same closeout, but one
is a verdict string in `check` and this is a section boundary in the claim parser
that can invalidate provenance. A single spec would put a reporting fix and a
provenance-affecting fix behind one review.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Scaffold layout | `go test ./internal/pose -run ArtifactClaimsSectionEnd -count=1` | Claims parsed, task lines ignored, no malformed claim |
| No regression across specs | `go test ./internal/pose -run ArtifactClaimsRepositoryBaseline -count=1` | Parsed claims identical for every spec in the repository |
| Gate recovers | `pose artifact-check --spec <scaffold-shaped-slug> --strict` | Exits 0 instead of refusing the run |
| Registered producer | `pose validate --strict --module pose-mcp` | The dedicated check runs the boundary corpus |

### Execution log

2026-09-20: reproduced while recording another finding during the
`pose-abm-progressive-review` closeout. A new spec written from `pose new-spec` broke
`roadmap-check` for the whole repository with
`malformed artifact claim "- [ ] Reproduce the mislabelled verdict in a test before
changing the message."`. Worked around in that spec by adding a `### ` subsection
after its artifact list; the parser is unchanged and still exposed.

2026-09-20, implemented. The section now ends at a heading of any level, including a
deeper one and a bare `##`. The previous rule is kept inside the test rather than
replaced by a golden file, so the fix is measured against the behaviour it changes
instead of against a number that would have to be regenerated by the thing it checks.

R2 was measured over the real corpus: 188 specs in this repository declare artifacts,
and for every one of them each bullet the previous rule read that is a valid claim
action is still a claim. A bullet it read that is not — a task checkbox, a prose
bullet from the next section — is the defect, and losing it is the fix.

Two mistakes of my own, both caught by the test rather than by reasoning:
`ListSpecs` returns frontmatter without bodies, so the first baseline compared empty
bodies and reported a clean result over nothing; the final assertion that the
comparison covered at least one spec is what exposed it. And the defect injection had
to restore the exact previous condition to reproduce the original error message,
which it does.

`delivery_integrity.go` was already outside `gofmt` before this change, verified by
stashing; it is left as it was rather than reformatted in a bugfix.

### Closeout

2026-09-20 UTC. Bundle `rvb-d0670abf492ecc16`, twelve criteria, ten evidence items,
under the five sealed contracts. Attestation `rva-dba88638a104f4c8`,
`agent:claude-opus-5`, approved; `review-check` fresh and approved;
`closeout-check` terminal. `surface-check --strict` exits 0 with one
inferred-coverage warning kept, for the same module-granularity reason recorded on the
two specs closed before it.

### Requirement trace

- R1 [satisfied] capability:artifact-claim-boundary evidence:integration
  check:artifact-claim-boundary-integration test:TestArtifactClaimsSectionEnd — the
  section ends at a heading of any level, asserted for `##`, `###`, `####` and `#`
- R2 [satisfied] capability:artifact-claim-boundary evidence:integration
  check:artifact-claim-boundary-integration
  test:TestArtifactClaimsRepositoryBaseline — 188 specs compared against the previous
  rule; no valid claim lost, no claim invented
- R3 [satisfied] capability:artifact-claim-boundary evidence:integration
  check:artifact-claim-boundary-integration test:TestArtifactClaimsSectionEnd — the
  scaffold layout parses its claims and no task line, and the previous rule is
  asserted to still reproduce the malformed claim it produced

## 7. Final Report

### Scope delivered

A claim list now knows where it ends. One spec's layout can no longer take
`artifact-check` and `roadmap-check` down for a whole repository, and the corpus
proves no existing claim moved.

### Residual risks

A `####` subsection inside `### Artifacts` would now end the section rather than
continue it. Nothing in either repository writes that, and the alternative — tracking
heading depth — would reintroduce the ambiguity this rule removes. Recorded rather
than left for someone to discover.

### Follow-ups

None.
