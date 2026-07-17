# AGENTS.md — {{PROJECT_NAME}}

This repository uses **POSE** (Project Operating Standard for Engineering) to
govern agent work. This file is the short contract. For the operating manual
(structure, CLI, per-task-type flows, CI policy), see [`POSE.md`](POSE.md).

## Project context

<!-- Describe here, in 3-6 lines, what this repository is: components,
     high-level architecture, and where the project's canonical references
     live (vision, backlog, decisions). Point to subproject AGENTS.md files
     when they exist. -->

{{PROJECT_NAME}}: describe the repository's purpose and its main components.

## Instruction precedence

On conflict: (1) direct instruction of the current task; (2) the most specific
`AGENTS.md` (deepest in the affected directory); (3) the broadest `AGENTS.md`
(root). Read only the `AGENTS.md` files needed for the paths involved.

## Mandatory artifacts (spec / ADR / checks)

- **Spec**: required for non-trivial feature/scope changes.
- **ADR**: required for architectural decisions or structural contract changes.
- **Checks**: required whenever an applicable command exists in the changed
  module (`test`, `lint`, `typecheck`, `build`, security/contract checks).

## Active paths in the flow

- POSE operating manual: [`POSE.md`](POSE.md)
- Workflows per task type: [`.pose/workflows/`](.pose/workflows/)
- Domain rules (cumulative): [`.pose/rules/`](.pose/rules/)
- Specs per feature/scope: [`.pose/specs/`](.pose/specs/)
- Governed roadmaps: [`.pose/roadmaps/`](.pose/roadmaps/)
- Implementation ADRs: [`.pose/adr/`](.pose/adr/)
- Skills for recurring tasks: [`.agents/skills/`](.agents/skills/)
- Automation support (CLI `./pose`): [`.pose/scripts/`](.pose/scripts/)

## Domain rules

Apply the rules relevant to the scope, cumulatively:

- Go backend: [`.pose/rules/backend-go.md`](.pose/rules/backend-go.md)
- React frontend: [`.pose/rules/frontend-react.md`](.pose/rules/frontend-react.md)
- Kubernetes: [`.pose/rules/kubernetes.md`](.pose/rules/kubernetes.md)
- Security: [`.pose/rules/security.md`](.pose/rules/security.md)
- Documentation / Process: [`.pose/rules/documentation-style.md`](.pose/rules/documentation-style.md)
- Delivery evidence (claiming delivery requires a gate): [`.pose/rules/delivery-evidence.md`](.pose/rules/delivery-evidence.md)
- Knowledge governance: [`.pose/rules/knowledge-governance.md`](.pose/rules/knowledge-governance.md)

**Precedence between domains:** on conflict, apply the most restrictive rule
(usually `security`) without breaking frontend/backend contracts.

## Available skills

Use the skill matching the task type (do not load all of them). Catalog in
[`.agents/skills/README.md`](.agents/skills/README.md); machine-readable
discovery: `./pose suggest <type> [--path <dir>]`.

- `pose-feature` · `pose-bugfix` · `pose-review` · `pose-adr` · `pose-test-plan`
- `pose-doc-update` · `pose-knowledge` · `pose-spec-closeout` · `pose-recurrence-escalation`

## Verification

Prefer deterministic checks whenever they exist: `test`, `lint`, `typecheck`,
`build`, security/contract validations. Canonical matrix in
[`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json),
executed by `./pose validate`.

## Do not

- Large refactors unrelated to the task at hand.
- Change public contracts without updating the applicable spec/ADR/docs.
- Skip tests when an applicable test command exists in the module.
- Expose secrets in code, docs, examples or logs.
