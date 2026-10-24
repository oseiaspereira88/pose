# ADR: Shared supported scope taxonomy for review policy

## Status

Accepted — prospective decision under user authorization on 2026-10-23. No human attestation, policy adoption or release is implied.

## Context

B05 reproduces an arbitrary identity_assurance key (spce) accepted by the loader and counted as verified protection, whereas real bundle selection consumes spec, milestone and roadmap. Independence keys have the same missing validation. Existing policy parsing deliberately accepts unknown top-level fields for forward compatibility; that does not authorize unknown members of a security selector map. The component-aware effective review plan ADR governs selection, not arbitrary new scope kinds.

## Decision

Use one closed scope-kind taxonomy for canonical scope parsing, current policy maps identity_assurance and reviewer_independence, and the identity assurance getter. Validate exact keys without trimming or case folding in every supported policy schema, including inactive maps. Known kinds retain their current defaults; an unsupported direct getter input returns no assurance rather than declared/verified. Diagnose field, quoted key and supported kinds deterministically. A current invalid policy fails as a whole through domain, CLI and MCP; no partial projection advertises effective protection. Preserve operator bytes and permissive unknown top-level fields. Future scope kinds require an explicit engine contract/version extension; older engines refuse them.

Keep sealed payloads and their stamped gates as historical records. Reading them never rewrites their policy or applies a current configuration retroactively. Current preparation and protection projection require current valid policy; failing those operations does not invalidate or mutate the sealed record.

Classify the stricter refusal of previously tolerated invalid configuration as breaking/major, including schema-1 maps. Do not bump the policy schema solely to permit invalid keys on older schemas; that would leave the false-protection path. Do not release, migrate an operator file or adopt verified identity automatically.

## Consequences

Operators must correct invalid keys explicitly before current governed operations can proceed. Valid known-kind configuration keeps its defaults and security gates. No assurance is inferred from raw configured presence. Historical inspection remains separate from current-policy evaluation.

Rejected: silently normalize typos — guesses alter security selection. Ignore unsupported keys with an effective partial policy — hides broken intended protection. Reject unknown top-level fields — violates established forward compatibility. Rewrite sealed history — changes the reviewed contract. Expand into unrelated profile/configuration validation — not needed to close B05's assurance/independence defect.
