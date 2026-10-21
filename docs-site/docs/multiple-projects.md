# Govern work across projects

**Doc type:** Explanation & how-to · **Applies to:** POSE v7.4.1 · **Reviewed:** 2026-10-20

Use separate project identities when repositories own independent engineering
contracts. Use module metadata and the validation matrix for components inside
one project. Registering several roots does not make them parent and child.

## Resolve authority before writing

Qualified references use `xref:<project-id>/<artifact-ref>`. They let dependency,
readiness, roadmap and MCP readers resolve the canonical owner rather than
inventing a local spec with the same slug.

```bash
pose context --project-id <id> --task <xref> --json
pose new-spec-qualified <slug> --task <xref> --expect-context <digest>
```

Use the authority and current context revision returned by `context`. Qualified
creation refuses an old or conflicting context. An older engine that does not
recognize `new-spec-qualified` fails before writing; do not emulate it with
local `new-spec` against the same slug.

Configure explicit `POSE_PROJECT_ROOTS` bindings for cross-project writes.
Discovery grants visibility, not write authority. With MCP, pass `project_id`
on each call and read `pose_mcp_context` after changing workspace or configuration.
Review [MCP project selection](mcp.md) before exposing multiple roots.

## Consume another project's outcome

A federated roadmap can accept a reviewed source outcome while retaining its
own integration and approval gates. Acceptance binds to source revisions,
review evidence and trust policy. Changed or unavailable sources and revoked
trust block acceptance rather than falling back to a local artifact. Qualified
consumers and schema upgrades require explicit adoption through compatible
readers. Inspect `pose adopt --list` before changing these contracts.

A milestone answers for its own edges and members. A later open milestone is
not evidence that an earlier milestone failed; the roadmap still owes its
registered cut criteria at the final boundary.

## Transfer ownership explicitly

Use `pose spec-transfer preview` to inspect a proposed transfer. Apply or resume
only with the preview digest, explicit authorization for each affected project
and current roots/context. A transfer preserves source history; it does not
create two owners of the same work.

`--mode reconcile-terminal` handles an executor already closed with terminal
evidence. It verifies the destination without writing it and records a
requirement map, including explicit open obligations where work remains.
Do not use a redirect as proof that those obligations were delivered.

For commands and flags, use the [CLI reference](cli.md); for in-repository
component selection and dependency fanout, use [monorepo recipes](monorepo-recipes.md).
