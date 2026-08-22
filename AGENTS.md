# AGENTS.md — {{PROJECT_NAME}}

This repository uses **POSE** (Project Operating Standard for Engineering) to
govern agent work. This file is the short contract. For the operating manual
(structure, CLI, per-task-type flows, CI policy), see [`POSE.md`](POSE.md).

## Project context

<!-- pose:instance-owned -->

<!-- Describe here, in 3-6 lines, what this repository is: components,
     high-level architecture, and where the project's canonical references
     live (vision, backlog, decisions). Point to subproject AGENTS.md files
     when they exist. `pose install` excerpts this from an existing
     README.md/CLAUDE.md on first install when one is present; `pose
     update` never touches this section afterward — edit it directly. -->

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
- **Commit trailer (`POSE-Spec:`)**: required on every commit that implements,
  modifies, or tests artifacts declared by a spec (`POSE-Spec: <slug>`).
  Without this trailer, `pose artifact-check` and `pose close` cannot attribute
  Git change sets to the spec's `### Artifacts` section.

## Active paths in the flow

- POSE operating manual: [`POSE.md`](POSE.md)
- Workflows per task type: [`.pose/workflows/`](.pose/workflows/)
- Domain rules (cumulative): [`.pose/rules/`](.pose/rules/)
- Specs per feature/scope: [`.pose/specs/`](.pose/specs/)
- Governed roadmaps: [`.pose/roadmaps/`](.pose/roadmaps/)
- Implementation ADRs: [`.pose/adr/`](.pose/adr/)
- Skills for recurring tasks: [`.agents/skills/`](.agents/skills/)
- Native automation entrypoint: the `pose` Go binary (`pose help`).

## Assessment Tool Execution Timing

AI agents must use assessment tools at specific points in the flow:

1. **Task / Spec Start (`pose-feature`)**:
   - Run `pose assess discover [--component <dir>]` / `pose_component_discover` to obtain LOC metrics, debts, and module structure before modifying code.
2. **Inter-Module Contract Change / PR Review (`pose-review`)**:
   - Run `pose assess integrate` / `pose_integration_check` when touching Protobuf, Kafka, REST APIs, or MCP tools.
   - Run `pose assess tech-debt` / `pose_tech_debt_check` during code review to ensure markers (`TODO`, `FIXME`, `stub`, `panic`) are covered by follow-ups or specs.
3. **Spec Closure (`pose-spec-closeout`)**:
   - Run `pose assess discover --update-state` upon delivery completion to recalculate dynamic platform completeness and update `.pose/assessments/` and `.pose/state/`.

## Domain rules

Apply the rules relevant to the scope, cumulatively:

- Go backend: shipped as the extension `pose-rule-backend-go` — install with `pose extension install pose-rule-backend-go`.
- Python backend: shipped as the extension `pose-rule-backend-python` — install with `pose extension install pose-rule-backend-python`.
- Rust backend: shipped as the extension `pose-rule-backend-rust` — install with `pose extension install pose-rule-backend-rust`.
- Java/Kotlin backend: shipped as the extension `pose-rule-backend-java` — install with `pose extension install pose-rule-backend-java`.
- .NET backend: shipped as the extension `pose-rule-backend-dotnet` — install with `pose extension install pose-rule-backend-dotnet`.
- React frontend: shipped as the extension `pose-rule-frontend-react` — install with `pose extension install pose-rule-frontend-react`.
- Vue frontend: shipped as the extension `pose-rule-frontend-vue` — install with `pose extension install pose-rule-frontend-vue`.
- Svelte frontend: shipped as the extension `pose-rule-frontend-svelte` — install with `pose extension install pose-rule-frontend-svelte`.
- Cloudflare Workers: shipped as the extension `pose-rule-serverless-cloudflare` — install with `pose extension install pose-rule-serverless-cloudflare`.
- Docker: shipped as the extension `pose-rule-infra-docker` — install with `pose extension install pose-rule-infra-docker`.
- Terraform: shipped as the extension `pose-rule-infra-terraform` — install with `pose extension install pose-rule-infra-terraform`.
- Kubernetes: shipped as the reference extension `pose-rule-kubernetes` — install with `pose extension install pose-rule-kubernetes`.
- GitHub Actions CI/CD: shipped as the extension `pose-rule-cicd-github-actions` — install with `pose extension install pose-rule-cicd-github-actions`.
- Security: [`.pose/rules/security.md`](.pose/rules/security.md)
- Documentation / Process: [`.pose/rules/documentation-style.md`](.pose/rules/documentation-style.md)
- Delivery evidence (claiming delivery requires a gate): [`.pose/rules/delivery-evidence.md`](.pose/rules/delivery-evidence.md)
- Knowledge governance: [`.pose/rules/knowledge-governance.md`](.pose/rules/knowledge-governance.md)

**Precedence between domains:** on conflict, apply the most restrictive rule
(usually `security`) without breaking frontend/backend contracts.

## Available skills

Use the skill matching the task type (do not load all of them). Catalog in
[`.agents/skills/README.md`](.agents/skills/README.md); machine-readable
discovery: `pose suggest <type> [--path <dir>]`.

- `pose-feature` · `pose-bugfix` · `pose-review` · `pose-adr` · `pose-test-plan`
- `pose-doc-update` · `pose-knowledge` · `pose-spec-closeout` · `pose-recurrence-escalation`

## Verification

Prefer deterministic checks whenever they exist: `test`, `lint`, `typecheck`,
`build`, security/contract validations. Canonical matrix in
[`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json),
executed by `pose validate`.

## Do not

- Large refactors unrelated to the task at hand.
- Change public contracts without updating the applicable spec/ADR/docs.
- Skip tests when an applicable test command exists in the module.
- Expose secrets in code, docs, examples or logs.

## Instance notes

<!-- pose:instance-owned -->

<!-- Repository-specific agent guidance. `pose update` refreshes every other
     section of this file from the shipped contract but never touches this one,
     so put local conventions, exceptions and pointers here rather than editing
     the engine-owned sections above. -->

## Open-Source POSE Contributor Mode

<!-- pose:contributor-mode -->

**Contributor Mode is ACTIVE.** When executing tasks, if you encounter POSE engine defects, tool frictions, missing stack rules, diagnostic false-positives, or clear improvement opportunities:

1. **Stage structured feedback locally**: create a report artifact under `.pose/contributions/<timestamp>-<slug>.md` documenting the observed limitation, synthetic reproduction, and proposed solution.
2. **Strict Privacy Invariant**: NEVER include proprietary business logic, internal hostnames/domains, customer data, API keys, credentials, or private source code in staged contributions. All examples must use generic, synthetic reproductions.
3. **Developer Adjudication**: Staging is automatic and local. Submitting or creating upstream GitHub issues (`oseiaspereira88/pose`) is always an explicit developer decision.
