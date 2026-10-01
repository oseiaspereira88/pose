# ADR: Reviewable usage verdict journal

## Status

Accepted (2026-10-01) — spec `pose-usage-findings-adjudication`.

## Context

The accepted local-usage ADR stores only HMAC finding identities in untracked,
per-machine events. A human verdict needs to follow the same stable finding
across clones without turning automatic usage events into tracked project data.
Some finding IDs contain absolute paths or free-form text, so copying every raw
ID into a tracked journal would violate the security boundary.

## Decision

Store explicit human verdicts as append-only JSONL in
`.pose/usage/verdicts.jsonl`. Each line carries tool, bounded stable finding ID,
verdict, reason, reviewer alias and timestamp. The CLI accepts bounded relative
structured identifiers (including slash-separated check IDs) and rejects
absolute paths, traversal, whitespace, control characters and free-form input.
A later journal line for the same tool and ID is
the effective verdict; earlier lines remain reviewable. Reports hash each
accepted ID with the machine-local salt at read time and compare it with local
event fingerprints. Automatic event storage and schema do not change.
The join uses all local event history; a query window limits automatic usage
counts but does not reinterpret the team's standing verdicts.
Structured validation uses the stable check ID alone as its finding identity;
the previous in-memory `check ID + NUL + outcome` shape could not be named in a
CLI argument and made one check's failure and execution error unrelated
findings. The event schema and HMAC storage remain unchanged, but verdicts for
validation checks match only events recorded after this normalization.

The report shows adjudicated totals and the false-positive rate separately
from automatic usage metrics. Verdicts with no matching local event remain
visible as unmatched, including on a new clone. No report aggregates by
reviewer and no network transmission is added.

## Consequences

- A verdict can be shared and reviewed in Git while local events remain
  privacy-bounded and worktrees stay clean during ordinary tool invocations.
- A tracked review decision names its subject, so contributors must review
  identity and reason before committing. The CLI's bounded syntax reduces the
  risk of accidentally recording paths or free-form content.
- Relative check IDs can disclose module names. Reviewers must inspect IDs and
  reasons before committing the journal. Absolute, traversing and free-form
  IDs cannot be adjudicated until the producer supplies a safe stable ID.
- A local-only HMAC journal was rejected because verdicts would not follow a
  clone. Storing every raw identity was rejected because some include paths or
  user content. Reusing decision logs per finding was rejected as too heavy for
  aggregate reporting.
