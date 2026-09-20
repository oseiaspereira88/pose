---
slug: artifact-claims-stop-at-the-section-end
status: draft
created_at: 2026-09-20
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
delivers:
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

Remaining artifacts are declared when the fix is implemented.

### Rollout and reversal

A boundary condition in one parser. Reverting is reverting that condition; no
stored claim changes, because R2 requires the parsed set to be unchanged.

## 4. Tasks

- [ ] Pin the current claims of every spec in the repository as a baseline.
- [ ] Reproduce the task-line claim from a scaffold-shaped spec.
- [ ] End the section at any heading and prove the baseline is unchanged.

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

### Execution log

2026-09-20: reproduced while recording another finding during the
`pose-abm-progressive-review` closeout. A new spec written from `pose new-spec` broke
`roadmap-check` for the whole repository with
`malformed artifact claim "- [ ] Reproduce the mislabelled verdict in a test before
changing the message."`. Worked around in that spec by adding a `### ` subsection
after its artifact list; the parser is unchanged and still exposed.

## 7. Final Report

Not implemented yet.
