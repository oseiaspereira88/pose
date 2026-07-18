---
slug: agent-interop-ecosystem
status: active
created_at: 2026-07-18
depends_on:
---

# Roadmap: Agent interoperability and ecosystem

**Portfolio order:** 5 of 7  
**Outcome:** expose a protocol-complete, project-safe MCP surface and a versioned extension ecosystem without making remote agents implicit code executors.

Follow the MCP and Agent Skills specifications. Add protocol features only where they improve governance use cases, and keep execution behind explicit policy and sandbox boundaries.

## Milestone: project-protocol
- after: 
- target_start: 2026-09-21
- target_due: 2026-10-23
- specs: pose-mcp-project-scope-contract, pose-mcp-protocol-completeness

**Exit gate:** independent clients pass catalog, schema, project selection and lifecycle conformance.

## Milestone: controlled-execution
- after: project-protocol
- target_start: 2026-10-26
- target_due: 2026-11-20
- specs: pose-safe-validate-orchestration

**Exit gate:** validation requests require explicit approval, policy and isolated execution.

## Milestone: extension-ecosystem
- after: controlled-execution
- target_start: 2026-11-23
- target_due: 2026-12-18
- specs: pose-agent-skills-conformance, pose-extension-catalog-lifecycle

**Exit gate:** clients discover signed, compatible extensions without bypassing policy or repository ownership.

## Risk controls

- Preserve human control for consequential tool calls.
- Fail closed on ambiguous project selection or policy evaluation.
- Separate read-only MCP governance from execution orchestration.

