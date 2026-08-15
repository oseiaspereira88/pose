---
type: decision-log
slug: github-issue-closing-keyword-format
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-15
last_reviewed_at: 2026-08-15
expires_at: 2026-11-13
source_refs:
  spec: "pose-instance-engine-version-tracking, pose-install-locale-autodetect"
  workflow: "bugfix, release"
  commands: []
  external_sources:
    - {url: "https://github.com/oseiaspereira88/pose/issues/18", accessed_at: "2026-08-15"}
    - {url: "https://github.com/oseiaspereira88/pose/issues/20", accessed_at: "2026-08-15"}
---

# decision-log: github-issue-closing-keyword-format

## Context

Two commits landed on `pose-dist` `main` today with `Closes
github.com/oseiaspereira88/pose#20.` and `Fixes
github.com/oseiaspereira88/pose#18.` in their trailers, expecting GitHub to
auto-close the referenced issues on push. Neither closed. GitHub's
closing-keyword parser only recognizes `#N` or `owner/repo#N` — a full URL
prefix (`github.com/owner/repo#N`) is not a recognized form, closing keyword
or not. Both issues stayed open until closed manually with `gh issue close`.

Investigated whether this was a POSE defect — some rule, skill or workflow
template generating or recommending the malformed form — before writing this
down. It is not: `grep`ing `.pose/rules/*.md`, `.agents/skills/*/SKILL.md`
and `.pose/workflows/*.md` for `Closes`/`Fixes`/`github.com/.*#` returns
nothing, and `git log --all` over this repository's entire commit history
shows exactly one prior convention for a `Closes` trailer —
`Closes roadmap N milestone M: spec ...`, POSE's own internal
roadmap/milestone closeout, unrelated to GitHub issues — plus these two new
malformed ones. The malformed form was introduced by the agent authoring
those two commits, not by any POSE tooling or documented convention.

A related, independently-discovered pattern from the same investigation:
issues #7, #8 and #9 on this repository had each already been resolved by a
`status: done` spec days to weeks earlier (`pose-delivery-surface-assurance`,
2026-08-03, for #7; `pose-artifact-provenance-ledger`, 2026-08-03, for #8;
`pose-mcp-active-context`, 2026-08-06, for #9 — whose own spec explicitly
listed "Close GitHub issues" as a **Non-goal**, deliberately deferring it to
a later step that never happened) but none were ever closed. This is not an
isolated oversight — it is the same shape of gap twice in one day (once for
#7/#8/#9 historically, once for #18/#20 today), which is why it is worth a
durable note rather than a one-off fix.

## Current state

- #18 and #20 closed manually today, each citing the delivering commit(s)
  and spec slug in the closing comment.
- #7, #8 and #9 confirmed resolved by name (specs above) but were **not**
  closed as part of this session — flagged to the user, awaiting their
  explicit go-ahead to close (POSE process: closing an issue is a visible
  action on shared state, not something to do unprompted).

## Next checks

- None deterministic; this is a process convention, not a code defect.
  No `pose-*` test or lint covers commit trailer syntax.

## Risks

- Recurrence: nothing in the repository enforces or lints commit-trailer
  syntax for issue references, so the same malformed form (or another
  GitHub doesn't parse) can slip in again from any author, human or agent.
- The resolved-but-never-closed pattern is broader than commit syntax: a
  spec can fully satisfy an issue's scope and even say so in its own body
  (as `pose-artifact-provenance-ledger` and `pose-delivery-surface-assurance`
  did explicitly) and the issue still never gets closed, because nothing
  in `pose-spec-closeout` checks for or prompts a linked-issue close. Spec
  frontmatter has no `issue:`/`closes:` field today, so there's no
  structured signal a closeout tool could act on even if one existed.

## Next owner

Same owner. If a `pose-doctor`-style check or a spec-frontmatter
`closes_issue:` field to make this checkable is ever pursued, it belongs in
a proper spec of its own — not implied by this note.

## Correct format (for reuse)

- `Closes #20` or `Closes oseiaspereira88/pose#20` — either is recognized.
- `Closes github.com/oseiaspereira88/pose#20` — **not** recognized; GitHub's
  parser does not accept a URL-prefixed issue reference as a closing
  keyword, even though the same string is perfectly valid prose elsewhere
  in a commit body (which is likely why it slipped in here — the agent was
  already citing issues that way in body prose and reused the same string
  in the closing-keyword line without checking GitHub's actual grammar).

## References

- `.pose/specs/pose-delivery-surface-assurance/spec.md` (issue #7)
- `.pose/specs/pose-artifact-provenance-ledger/spec.md` (issue #8)
- `.pose/specs/pose-mcp-active-context/spec.md` (issue #9, undeclared link)
- `.pose/specs/pose-install-locale-autodetect/spec.md`,
  `pose-instance-engine-version-tracking/spec.md` (issues #18, #20)
- https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword
