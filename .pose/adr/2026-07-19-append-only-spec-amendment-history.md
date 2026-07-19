# ADR: Append-only spec amendment history

## Status
Accepted (2026-07-19) — spec `pose-spec-amendment-history`

## Context

A spec is a living contract, but nothing prevented rewriting a requirement
*after* its evidence was recorded — the trace could point at checks that
verified a different sentence than the one now published. Git history shows
the edit but not the rationale, reviewer or materiality. Comparable systems
([OpenSpec](https://github.com/Fission-AI/OpenSpec) change deltas,
[Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
lifecycle) treat intent changes as first-class reviewable events.

Constraint: editorial corrections must stay lightweight; risk: over-sensitive
detection burdens harmless work.

Alternatives considered:

1. **Git-log archaeology** — shows diffs, not materiality, rationale or
   review; unusable as a deterministic gate.
2. **Inline amendments section in spec.md** — the log would live inside the
   document it audits and be editable in the same stroke that rewrites a
   requirement.
3. **Sibling append-only event log** (`amendments.jsonl`) with
   deterministic hash-based detection.

## Decision

Option 3:

- **Storage:** `.pose/specs/<slug>/amendments.jsonl`, schema-versioned
  (`schema: 1`), append-only, merge-friendly (one JSON event per line).
- **Event fields (R2):** RFC3339 UTC timestamp, material-change taxonomy
  (`baseline | added | withdrawn | semantic | editorial`), affected R-IDs,
  rationale (required except baseline), pseudonymous `author`/`reviewer`
  aliases (no personal data), and the post-change hash per affected ID.
- **Detection (R1/R3):** each requirement's normalized text (whitespace
  collapsed) is fingerprinted (`sha256`, 12 hex). `pose lint-spec` on a
  `done` spec with a history compares current hashes against the latest
  acknowledged state and rejects: changed text without an event, added IDs
  without an event, and removals without a `withdrawn` event. Withdrawn IDs
  keep an empty-hash entry — published IDs are never renumbered and remain
  addressable.
- **Ergonomics:** `pose amend <slug> --baseline` snapshots the current
  state; `--change editorial` acknowledges rewording in one command —
  editorial work costs one line, which is the lightweight path the
  constraint demands. `--list` renders history plus pending
  acknowledgments.
- **Adoption is opt-in per spec:** no `amendments.jsonl`, no gate — the
  contract activates when the first baseline is recorded (additive
  migration, same staging pattern as the requirement trace).
- **Projection:** MCP read tool `pose_spec_amendments` returns events and
  unacknowledged findings (additive catalog change, golden reviewed).

## Consequences

- Positive: a done spec cannot silently diverge from the requirements its
  evidence verified; auditors read materiality and rationale, not diffs.
- Positive: deterministic and offline — no daemon, no web UI, no inference.
- Trade-off: hash-based detection flags every wording change, including
  harmless ones; the one-line `editorial` acknowledgment is the accepted
  cost of never guessing materiality.
- Residual: an author can falsify rationale; review owns truthfulness, the
  gate owns existence, ordering and consistency.
