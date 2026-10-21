---
spec: pose-action-target-and-node-scoping
category: fixed
---

# Action request destination and material scoping

- Select restrictions by explicit effect destination, phase and mode; retain the
  documented whole-source default for empty scope and current roadmap membership.
- Keep relevant open/invalidated requests visible after their origin is terminal.
- Resolve subject and target nodes by the complete project/artifact/kind/ID tuple.
- Accept structured effect scopes via CLI `--effect-json` and MCP effect objects;
  preserve existing `phase:mode` strings and immutable journal schema/digests.
- Require subjects on new approvals/acceptances under adopted agency readiness.
  Existing unbound journals are not rewritten. Policy-off defaults are unchanged.

Compatibility: this corrects previously omitted/excess restrictions for adopted
instances and tightens material-reference validation. Assess version/adoption at
release; this implementation does not adopt policy or publish a release.
