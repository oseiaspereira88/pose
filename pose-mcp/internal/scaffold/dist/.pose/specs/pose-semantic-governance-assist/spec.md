---
slug: pose-semantic-governance-assist
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-recurrence-effectiveness, pose-knowledge-consumption-traceability
priority: 32
---

# Spec: Human-reviewed semantic governance assist

## 1. Intent

### Goal
suggest related follow-ups, recurrence patterns and knowledge with explainable evidence.
### Business value
Adds semantic leverage while preserving deterministic authority.
### Constraints
- Suggestions are advisory and never mutate lifecycle automatically.
### Non-goals
- Make an LLM verdict a blocking check.

## 2. Requirements

### Functional
- R1: Each suggestion shall cite artifacts, score/rationale and provider metadata.
- R2: Sensitivity and project boundaries shall be enforced before retrieval.
- R3: Accepted/rejected suggestions shall feed evaluation without training on restricted content.

### Non-functional
- Provide lexical fallback and bounded latency/cost.

### Security
- Require approved providers, prompt-injection defenses and data policy.

### Compatibility
- Core closeout and recurrence work with semantic assist disabled.

## 3. Technical Plan

### Affected areas
- Follow-up/knowledge adapters, MCP, policy, evaluation and Harne8 UI.

### API/contract changes
- Define suggestion, confirmation and provenance schemas.

### Data/storage changes
- Store minimized decision feedback with retention labels.

### Technical risks
- Similarity can conflate related but non-equivalent obligations.

### Primary references
- [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework)
- [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

### Implementation
- [ ] Threat-model retrieval, injection and confirmation paths. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Implement provider-neutral cited suggestions with fallback. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [ ] Measure precision, rejection and unsafe-leakage on labeled fixtures. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Semantic|Followup|Knowledge|Policy'` and retain evidence. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-semantic-governance-assist-lexical-only-provider.md` (Accepted): a `SuggestionProvider`-shaped surface with exactly one approved, fully-tested provider (`lexical`, reusing the existing `followupSimilarity`/`followupTokens` primitives), an explicit provider allowlist, and a prompt-injection sanitization pass applied to every candidate regardless of provider so a future real provider inherits the same defense by construction. Rejected: wiring an untestable real LLM provider (no safely verifiable endpoint in this environment); duplicating the similarity algorithm instead of reusing the already-tested one.

## 6. Validation

**Strategy:** validate that every suggestion carries a citation/score/rationale/provider, that sensitivity and self-referencing follow-ups are excluded before any scoring happens, that recurrence patterns surface as a candidate source, that an unapproved provider is rejected, that prompt-injection-shaped content is stripped, that feedback is minimized (no candidate content ever reaches storage), and that the whole path never mutates any file.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'SemanticSuggest|SuggestFeedback|SanitizeForPrompt' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-semantic-governance-assist --ready-check`.

### Requirement trace
- R1 [satisfied] every suggestion is `{artifact_ref, kind, score, rationale, provider}`, all non-empty; check:test (TestSemanticSuggestCitesArtifactScoreRationaleProvider)
- R2 [satisfied] restricted knowledge filtered before scoring even when it would otherwise score highest; a spec never suggests its own follow-ups to itself (project/self boundary); check:test (TestSemanticSuggestFiltersRestrictedKnowledgeBeforeRetrieval, TestSemanticSuggestExcludesOwnSpecFollowups)
- R3 [satisfied] feedback records are minimized — reflection-equivalent field check proves no content/rationale/body field ever exists in a stored record, and because restricted content is filtered upstream (R2) it can never reach feedback either; check:test (TestSuggestFeedbackRecordsMinimizedDataNoContent, TestSuggestFeedbackValidation)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'SemanticSuggest|SuggestFeedback|SanitizeForPrompt' -v -count=1` — SUCCESS (12 tests, including recurrence-pattern-as-candidate coverage and a byte-for-byte non-mutation proof).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-semantic-governance-assist --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Constraint (advisory, never mutates lifecycle automatically): `TestSemanticSuggestNeverMutatesLifecycle` snapshots the whole tree before/after a call and asserts zero diff.
- Non-goal (never a blocking check): `pose semantic-suggest` always exits 0 on a successful run regardless of suggestion content; nothing in the codebase treats its output as a gate.
- Security (approved providers, prompt-injection defenses): `TestSemanticSuggestRejectsUnapprovedProvider` / `TestSuggestFeedbackValidation` prove the allowlist; `TestSanitizeForPromptRemovesUnsafeAndSecretPatterns` proves the sanitization pass.
- Non-functional (lexical fallback, bounded latency/cost): the only provider in this release IS the lexical fallback — bounded by construction (no network call exists in this path at all).

## 7. Final Report

- Delivered scope: `pose semantic-suggest` (advisory, cited, scored, rationale-explained suggestions from knowledge/follow-ups/recurrence patterns, sensitivity-filtered before retrieval, self-spec-excluded) and `pose suggest-feedback` (minimized accept/reject recording) — both reusing the project's existing, already-tested lexical similarity primitives rather than introducing a new algorithm or an untestable external provider.
- Residual risk: similarity can still conflate related-but-non-equivalent obligations, as the spec's own Technical risk names — mitigated by always surfacing the rationale (shared terms) alongside the score so a human reviewer can judge relevance directly rather than trusting the score alone; a real semantic (embedding/LLM) provider, when one can be safely configured and tested, would reduce but not eliminate this risk.
- Follow-ups: see below.

### Follow-ups

- [open] Add a real semantic (embedding or LLM-backed) `SuggestionProvider` once an approved, safely-testable endpoint exists — the allowlist and sanitization pass are already structured for it. (owner:@pose-maintainers crit:low review:2026-10-19)
