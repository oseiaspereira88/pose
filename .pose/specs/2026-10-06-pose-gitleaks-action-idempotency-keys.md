---
slug: pose-gitleaks-action-idempotency-keys
status: done
created_at: 2026-10-06
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 0
components: ci
task_type: bugfix
surface: minimal
delivers: 
changelog: none
---

# Spec: Secret detection stops reading action-request idempotency keys as API keys

## 1. Intent

### Goal

Make the `secrets` gate pass on action-request journals without weakening detection: gitleaks' `generic-api-key` rule matched every answer's `"idempotency_key":"decision4-2026-10-05"` on the word "key", failing the full-history scan on `main` (three journals) and on PR #131 (a fourth).

### Business value

A gate that is red for a non-secret trains people to ignore it, and blocks every pull request until someone looks.

### Constraints

The exception is conjunctive and match-scoped, like the repository's existing ones: only inside `.pose/actions/act-<16 hex>.jsonl`, only for the slug-shaped idempotency-key match. A secret anywhere else in a journal line still fails.

### Non-goals

Changing the journal format or any detection rule.

## 2. Requirements

### Functional

- R1: The full-history gitleaks scan with `.gitleaks.toml` shall report no finding for slug-shaped idempotency keys in action-request journals.
- R2: A secret in another field of a journal event (a token in `answer`, an `api_key` field) shall still be reported.

### Non-functional

- None.

### Security

- The exception is scoped to the match, not the line, so it cannot excuse a secret pasted into the same event.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

`.gitleaks.toml`.

### Artifacts

- created: .pose/specs/2026-10-06-pose-gitleaks-action-idempotency-keys.md
- created: .pose/starts/pose-gitleaks-action-idempotency-keys.json
- modified: .gitleaks.toml

### Technical risks

- None.

## 6. Validation

### Strategy

The CI command on the repository, and a scratch repository with a journal carrying a GitHub token and an `api_key` field beside a slug idempotency key.

### Deterministic checks

#### Security / Contract
- Command: `go -C pose-mcp run github.com/zricethezav/gitleaks/v8@v8.21.2 git --no-banner --redact --config ../.gitleaks.toml ..`
- Expected: no leaks found

### Requirement trace

- R1 [satisfied] <go -C pose-mcp run github.com/zricethezav/gitleaks/v8@v8.21.2 git --config ../.gitleaks.toml ..: no leaks found, 2026-10-06>
- R2 [satisfied] <scratch journal with a ghp_ token and an api_key field: github-pat and generic-api-key still reported, idempotency key not, 2026-10-06>

### Known gaps

None.

## 7. Final Report

### Delivered scope

A match-scoped `generic-api-key` exception for slug-shaped idempotency keys in action-request journals.

### Residual risks

None.

### Follow-ups
