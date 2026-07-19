# ADR: Agent Skills compatibility metadata and conformance gate

## Status
Accepted (2026-07-19) — spec `pose-agent-skills-conformance`

## Context

The nine shipped skills carried only the [Agent Skills](https://agentskills.io/specification)
baseline fields (`name`, `description`, `when_to_use`) with no POSE-specific
compatibility declaration and no automated check of any kind — a skill's
required-reading links, layout and content were unverified prose. The link
check immediately found a real, shipped defect: `pose-feature/SKILL.md`
pointed at `.pose-spec-closeout/SKILL.md` (a typo missing `../`), a dead
link no prior review had caught. The spec's own stated technical risk —
"schema-valid skills can still be semantically unsafe" — is real: nothing
prevented a skill from instructing an agent toward `curl | sh` or leaking a
pasted credential into shipped, version-controlled prose.

Alternatives considered for the security scan: a full secrets-scanning
engine (rejected — CI already runs gitleaks over the full history per
`pose-ossf-security-baseline`; duplicating that here would be exactly the
non-overlapping-scanner violation that baseline's own ADR rejected). A
handful of deterministic, offline regexes as defense-in-depth for the
specific case of hand-authored prose is the right-sized addition.

## Decision

- **Additive compatibility frontmatter (R2):** every skill declares
  `pose_schema_range` (`"min-max"`, validated as a valid, non-inverted
  integer range against `.pose/schema-version`), `clients`
  (comma-separated: `agents-skills`, `mcp`, `claude-code` today) and
  `capabilities` (comma-separated descriptive tags — `read`, `spec-write`,
  `validate`, etc. — informational for discovery/filtering, not an
  enforced sandboxed permission system; POSE does not claim to sandbox
  skill execution). These are POSE's own layer on top of the Agent Skills
  baseline, not a redefinition of it.
- **`pose skills-check`** (mirrors `knowledge-check`'s `--strict`/
  `--tolerant` shape) validates, per skill: required fields present;
  `name` matches its directory; `pose_schema_range` well-formed; every
  relative markdown link resolves to a real file confined inside the
  repository (path escape rejected); content scanned against a small,
  documented set of unsafe-instruction and secret-shaped patterns;
  `claude-code` in `clients` cross-checked against a real entry in
  `scaffold.ClaudeSkillLinks` — a client cannot be declared without a real
  link surface.
- **CI-enforced (R1):** `.github/workflows/ci.yml`'s `governance` job runs
  `pose skills-check --strict` alongside the structural gate, using the
  same explicitly identified development build.
- **MCP parity:** `pose_skills_check` exposes the same gate read-only over
  MCP, implemented as a thin adapter (`Store.SkillsCheck`) re-invoking the
  binary — ADR-003's existing pattern, identical to `pose_check`/
  `pose_lint_spec`; no logic duplication between CLI and MCP.
- **Compatibility fixture (R3):** `TestSkillsCheckDiscoveryAndBoundedWorkflowFixture`
  runs the real gate against this repository's own `.agents/skills/` tree
  (dogfooding, not a synthetic fixture) and asserts all nine are
  discoverable and pass — a genuine discovery-plus-bounded-workflow
  compatibility proof.

## Consequences

- Positive: skill content is now a tested, CI-blocking product surface;
  the pre-existing broken link was caught and fixed as a direct result.
- Positive: no duplicated secret-scanning coverage; the offline patterns
  here are explicitly scoped as defense-in-depth, not a substitute.
- Trade-off: the compatibility metadata (`clients`, `capabilities`) is
  descriptive, not enforced at runtime — an honest boundary given POSE has
  no execution sandbox for skills; claiming enforcement would overstate
  the guarantee.
- Residual: the pt-BR locale mirror (`locales/pt-BR/.agents/skills/`)
  received the same frontmatter fields but is not yet covered by
  `skills-check` itself (the gate only scans the installed
  `.agents/skills/` tree) — tracked as an open follow-up.
