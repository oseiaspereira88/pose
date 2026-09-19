# ADR: Material design basis remains an advisory projection

## Status
Proposed

## Context

Assumptions and design decisions are currently prose inside a spec. The next
ABM slice needs stable, reviewable relationships without inventing a second
document, a quality score or an opaque model judgment. The parser must also
remain safe when a spec contains examples, URLs or deliberately hostile text.

## Decision

Extend the existing `Decisions` section with explicit `Assumption A<N>` and
`Decision D<N>` headings. Keep Markdown authoritative and expose a deterministic
read-only projection through `lint-spec --design-check` and the equivalent MCP
parameter. The engine validates only objective structure, references, statuses,
and evidence states; it emits warnings for incomplete semantic fields and leaves
proportionality and technical truth to the review judgment.

## Consequences

Specs that do not declare A/D nodes remain unchanged. Material bases become
falsifiable and can distinguish verified, unverified, invalidated, withdrawn and
unknown external evidence. The projection adds no persistent state and cannot
prove that a referenced contract or rationale is true. A later delivery profile
may make selected diagnostics blocking, but this slice does not activate that
policy.

## Alternatives rejected

- **New mandatory ABM document:** duplicates the spec and raises ceremony for
  trivial changes.
- **LLM or text-quality score:** would be opaque, non-deterministic and easy to
  game by agents optimizing a proxy.
- **Treat every URL/knowledge note as verified evidence:** would turn an
  inaccessible reference into false certainty in offline mode.
