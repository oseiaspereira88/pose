# ADR: Semantic governance assist — lexical-only provider, no real LLM call in this release

## Status
Accepted (2026-07-19) — spec `pose-semantic-governance-assist`

## Context

POSE already had two independent, deterministic-only advisory surfaces:
`pose knowledge-suggest` (query → related knowledge, lexical ranking) and
`followupSimilarity`/`followupTokens` (near-duplicate follow-up detection
inside `pose followups`). Neither cited artifacts with an explicit
provider metadata field, neither covered follow-ups or recurrence
patterns as suggestion candidates, and neither had a feedback mechanism.
The spec's own Constraint is unambiguous — "suggestions are advisory and
never mutate lifecycle automatically" — and its Non-goal forbids making
an LLM verdict a blocking check. The Security requirement additionally
demands "approved providers, prompt-injection defenses and data policy,"
language that presumes a real external provider exists, which this
sandbox has no way to configure or safely verify end to end.

Alternatives considered:

1. **Wire a real LLM provider (API key from env, HTTP call) as the
   primary path, lexical as fallback only.** Matches the spec's
   "Non-functional: provide lexical fallback" literally, but there is no
   way to safely test the primary path in this environment (no approved
   endpoint, no key, no verifiable prompt-injection defense against a
   live model) — shipping untested provider code that handles
   security-sensitive redaction is a worse outcome than not shipping it.
2. **Duplicate the lexical similarity logic into a new, parallel
   implementation** to keep this spec self-contained. Rejected: the
   existing `followupSimilarity`/`followupTokens`/`normalizeFollowup`
   (from `pose-followup-ownership-sla` follow-up-clustering work) are
   already tested, already the project's one deterministic-similarity
   algorithm — a second implementation would just be two things to keep
   in sync for no benefit.
3. **A `SuggestionProvider`-shaped surface with exactly one approved,
   fully-tested implementation (`lexical`) reusing the existing
   similarity primitives, an explicit provider allowlist that rejects
   anything else, and a prompt-injection sanitization pass applied to
   every candidate regardless of provider** — so a future real provider
   inherits the same defense by construction, not by remembering to add
   it later.

## Decision

Option 3.

- **One approved provider**: `approvedSuggestionProviders = {"lexical":
  true}` (`internal/cli/semantic_suggest.go`). `--provider <anything
  else>` is rejected with exit 2 before any retrieval happens — the
  Security requirement's "require approved providers" is a literal
  allowlist check, not a comment.
- **Reuse, not reimplementation**: candidate scoring calls the exact
  `followupSimilarity`/`followupTokens`/`normalizeFollowup` functions
  `pose knowledge-suggest` and `pose followups`' near-duplicate detection
  already use and already have test coverage for.
- **Three candidate sources, one instrumentation point each**:
  `loadKnowledgeArtifacts` (existing, already sensitivity-filters),
  `collectFollowups` (existing, filtered here to exclude the target
  spec's own follow-ups and any non-open disposition), and a new
  `collectRecurringPatterns` — deliberately a standalone function
  mirroring `cmdRecurrenceCheck`'s bucketing rather than refactoring that
  command, so this spec can never regress `pose-recurrence-effectiveness`'s
  already-shipped gate.
- **Prompt-injection defense is structural, not provider-specific**:
  `sanitizeForPrompt` (secret-shaped + unsafe-instruction pattern
  removal, reusing `redactSecretShapedContent` and `unsafeSkillPatterns`)
  runs on every candidate's text and on the query itself before any
  scoring or citation happens — the lexical provider never sends
  anything anywhere, but the sanitization runs anyway, so a future
  non-lexical provider added to the allowlist automatically inherits it
  rather than needing its own redaction pass remembered separately.
- **Every suggestion is `{artifact_ref, kind, score, rationale,
  provider}`** (R1) — `rationale` is the literal shared-token list, the
  same explainability shape `knowledge-suggest` already used, now
  formalized as a typed field alongside the provider metadata that was
  previously absent.
- **Feedback (R3) is a new, separate append-only record**
  (`.pose/reports/history/semantic-feedback-<YYYY-MM>.jsonl`, same
  monthly-JSONL convention as the DORA events) carrying only
  `{recorded_at, for_spec, artifact_ref, kind, decision, score,
  provider}` — never the candidate's text or rationale. Because
  restricted knowledge is filtered before it can ever become a
  suggestion, "never training on restricted content" is satisfied
  structurally: there is no code path where restricted content reaches
  the feedback record in the first place.

## Consequences

- Positive: shipping only the provider that can be fully tested and
  verified in this environment is a defensible, documented boundary
  rather than a silent gap — the `SuggestionProvider` shape (allowlist +
  mandatory sanitization pass) is exactly what a real provider would plug
  into later, so adding one is additive, not a redesign.
- Positive: zero new similarity algorithm to maintain — this spec is
  entirely new orchestration and citation/feedback plumbing over already-
  tested primitives.
- Negative: no real semantic (embedding/LLM) similarity exists yet — a
  suggestion is exactly as good as literal token/sequence overlap, which
  can miss conceptually related but differently-worded content. Tracked
  as a follow-up: add a real provider once an approved, testable endpoint
  exists.
- Neutral: `collectRecurringPatterns` duplicates `cmdRecurrenceCheck`'s
  bucketing logic (~15 lines) rather than sharing it — a deliberate
  trade of a small duplication for zero regression risk to a separately-
  owned, already-shipped gate.
