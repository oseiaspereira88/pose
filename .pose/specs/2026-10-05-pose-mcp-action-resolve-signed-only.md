---
slug: pose-mcp-action-resolve-signed-only
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:mcp-signed-action-resolution
---

# Spec: An agent relays an answer over MCP only with the principal's proof

## 1. Intent

### Goal

Add `pose_action_resolve`, the MCP way to record an answer to an action request, accepting only an answer the principal proved: a signature by a key registered to the principal (spec pose-signed-action-answers) or a trusted issuer's claim. Its preview returns the exact statement to sign and the command that signs it.

### Business value

An agent conducts the conversation in which a person decides, but an agent writing `human:maintainer` into a tool call proves nothing — the reason no MCP tool resolved requests until now. With a signature the agent can relay the person's answer without being able to forge it, closing the loop inside the session with or without Harne8. Part of roadmap pose-v7-onboarding-and-consolidation (milestone identity).

### Constraints

The same domain function as the CLI; no declared answer over MCP under any assurance; cancellations, waivers and invalidations stay on the CLI.

### Non-goals

Signing inside the MCP server: the server never holds a key.

## 2. Requirements

### Functional

- R1: `pose_action_resolve` shall be a governance-write tool that previews unless `apply` is true; the preview shall return the request, the canonical statement for the given actor, answer and idempotency key, and the `ssh-keygen -Y sign` command that signs it.
- R2: With `apply`, it shall record the answer through the CLI's resolution path only when it carries a `signature` or a `claim` with its `envelope`, and refuse a call without either, whatever the policy's assurance.
- R3: A signature or claim that does not verify shall be refused with the reason, and nothing recorded.
- R4: The tool catalog, `pose_action_requests` and the manuals shall name the tool and its proof requirement.

### Non-functional

- None.

### Security

- The MCP server holds no key and accepts no declared answer, so an agent cannot record an answer the principal did not sign.

### Compatibility

- Additive tool.

## 3. Technical Plan

### Affected areas

MCP server tool dispatch, definitions and catalog.

### Artifacts

- created: .pose/specs/2026-10-05-pose-mcp-action-resolve-signed-only.md
- created: .pose/starts/pose-mcp-action-resolve-signed-only.json
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/mcpserver/server_test.go
- created: pose-mcp/internal/mcpserver/action_resolve_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/mcp.md
- renamed: .pose/changelogs/unreleased/pose-mcp-action-resolve-signed-only.md -> .pose/changelogs/v7.0.0/pose-mcp-action-resolve-signed-only.md

### Delivery targets

- capability:mcp-signed-action-resolution module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- None beyond the verifier's, covered by spec pose-signed-action-answers.

## 6. Validation

### Strategy

The tool driven over HTTP JSON-RPC: preview, unsigned refusal under declared assurance, a bad signature refused, a good one recorded.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/mcpserver -run 'ActionResolve|Catalog'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestToolsCall_ActionResolve_RefusesAnAnswerWithoutProof check:mcp-action-resolve-integration
- R2 [satisfied] test:TestToolsCall_ActionResolve_RefusesAnAnswerWithoutProof test:TestToolsCall_ActionResolve_RecordsASignedAnswer check:mcp-action-resolve-integration
- R3 [satisfied] test:TestToolsCall_ActionResolve_RefusesAnAnswerWithoutProof check:mcp-action-resolve-integration
- R4 [satisfied] test:TestCatalogMatchesGolden test:TestCatalogDocsConformance test:TestToolsList check:mcp-action-resolve-integration

### Known gaps

- The positive path needs `ssh-keygen` and skips without it; the refusals do not.

## 7. Final Report

### Delivered scope

`pose_action_resolve` (governance-write): preview with the statement and signing command; apply only with a registered key's signature or an issuer claim, recorded with channel `mcp`; descriptions of `pose_action_open` and `pose_action_requests` updated.

### Residual risks

None.

### Follow-ups
