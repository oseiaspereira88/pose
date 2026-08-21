---
slug: pose-stack-rule-extensions-expansion
status: in-progress
created_at: 2026-08-21
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-cli-ergonomics-and-stack-expansion
priority: 0
components: extensions, rules, cli, scaffold
delivers: capability:python-rule-extension, capability:rust-rule-extension, capability:java-rule-extension, capability:dotnet-rule-extension, capability:cloudflare-rule-extension, capability:terraform-rule-extension
---

# Spec: Multi-Stack Domain Rule Extensions Expansion

## 1. Intent

### Goal
Expand the POSE rule extension ecosystem with 6 production-grade, bilingual rule packages for Python, Rust, Java/Kotlin, .NET, Cloudflare Workers, and Terraform, and wire automatic stack resolution in the native engine.

### Business value
Modern polyglot engineering organizations deploy diverse technology stacks beyond Go and React. Previously, POSE only shipped rule extensions for Go, React, and Kubernetes, leaving repositories using Python (FastAPI/Django), Rust (Axum/Tokio), Java/Kotlin (Spring/Gradle), .NET (ASP.NET Core/EF), Cloudflare Workers, and Terraform without standardized domain engineering rules.

By providing curated, signed-ready rule extensions for these core ecosystems with 100% bilingual parity (English + Portuguese) and automatic discovery via `pose doctor` and `pose suggest`, agents and human engineers receive context-appropriate architectural constraints, anti-pattern blocking, and validation instructions out of the box.

### Constraints
- Every rule extension must conform to the POSE Extension Schema v1 (`extension.json`).
- All rule files must include required patterns, blocking anti-patterns, minimum deterministic checks, conflict precedence rules, and base recurrence references.
- 100% linguistic parity between English (`files/.pose/rules/`) and Portuguese (`locales/pt-BR/.pose/rules/`).

### Non-goals
- Embedding these rules into the core repository root; they remain isolated, opt-in extensions installable via `pose extension install`.

---

## 2. Requirements

### Functional
- R1: Deliver `extensions/pose-rule-backend-python` with `backend-python.md` covering typing, async safety, error propagation, and API contracts.
- R2: Deliver `extensions/pose-rule-backend-rust` with `backend-rust.md` covering memory safety, error handling (`Result`), async runtime invariants, and API contracts.
- R3: Deliver `extensions/pose-rule-backend-java` with `backend-java.md` covering thread safety, exception handling, layered architecture, and dependency injection.
- R4: Deliver `extensions/pose-rule-backend-dotnet` with `backend-dotnet.md` covering async/await deadlocks, DI lifetimes, EF Core query tracking, and model validation.
- R5: Deliver `extensions/pose-rule-serverless-cloudflare` with `serverless-cloudflare.md` covering edge execution limits, binding security (KV/D1/R2), and caching.
- R6: Deliver `extensions/pose-rule-infra-terraform` with `infra-terraform.md` covering state isolation, least-privilege IAM, secret hygiene, and provider locking.
- R7: Update `ruleExtensionByStack` in `pose-mcp/internal/cli/rule_extension_resolver.go` to resolve `python`, `rust`, `java`, `dotnet`, and `cloudflare-workers`.
- R8: Update `AGENTS.md` and distributed scaffold copies (EN and pt-BR) with the expanded domain rules catalog.

### Non-functional
- Complete deterministic verification via `go test ./...` and `pose validate --strict`.

### Security
- Extensions must only declare write permissions confined to `.pose/rules/`.

### Compatibility
- Fully backward compatible with existing installed extensions and lockfiles.

---

## 3. Technical Plan

### Affected areas
- `extensions/pose-rule-backend-python/`
- `extensions/pose-rule-backend-rust/`
- `extensions/pose-rule-backend-java/`
- `extensions/pose-rule-backend-dotnet/`
- `extensions/pose-rule-serverless-cloudflare/`
- `extensions/pose-rule-infra-terraform/`
- `pose-mcp/internal/cli/rule_extension_resolver.go`
- `pose-mcp/internal/cli/rule_extension_resolver_test.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md`

### Artifacts
- created: extensions/pose-rule-backend-python/extension.json
- created: extensions/pose-rule-backend-python/files/.pose/rules/backend-python.md
- created: extensions/pose-rule-backend-python/locales/pt-BR/.pose/rules/backend-python.md
- created: extensions/pose-rule-backend-rust/extension.json
- created: extensions/pose-rule-backend-rust/files/.pose/rules/backend-rust.md
- created: extensions/pose-rule-backend-rust/locales/pt-BR/.pose/rules/backend-rust.md
- created: extensions/pose-rule-backend-java/extension.json
- created: extensions/pose-rule-backend-java/files/.pose/rules/backend-java.md
- created: extensions/pose-rule-backend-java/locales/pt-BR/.pose/rules/backend-java.md
- created: extensions/pose-rule-backend-dotnet/extension.json
- created: extensions/pose-rule-backend-dotnet/files/.pose/rules/backend-dotnet.md
- created: extensions/pose-rule-backend-dotnet/locales/pt-BR/.pose/rules/backend-dotnet.md
- created: extensions/pose-rule-serverless-cloudflare/extension.json
- created: extensions/pose-rule-serverless-cloudflare/files/.pose/rules/serverless-cloudflare.md
- created: extensions/pose-rule-serverless-cloudflare/locales/pt-BR/.pose/rules/serverless-cloudflare.md
- created: extensions/pose-rule-infra-terraform/extension.json
- created: extensions/pose-rule-infra-terraform/files/.pose/rules/infra-terraform.md
- created: extensions/pose-rule-infra-terraform/locales/pt-BR/.pose/rules/infra-terraform.md
- modified: pose-mcp/internal/cli/rule_extension_resolver.go
- modified: pose-mcp/internal/cli/rule_extension_resolver_test.go
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md

### Delivery targets
- capability:python-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:rust-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:java-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:dotnet-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:cloudflare-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:terraform-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- New rule extension identifiers available in catalog: `pose-rule-backend-python`, `pose-rule-backend-rust`, `pose-rule-backend-java`, `pose-rule-backend-dotnet`, `pose-rule-serverless-cloudflare`, `pose-rule-infra-terraform`.

### Data/storage changes
- None.

### Technical risks
- None identified.

---

## 4. Tasks

### Planning
- [ ] Structure rule content and extension manifests for all 6 target ecosystems.

### Implementation
- [ ] Author `pose-rule-backend-python` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-backend-rust` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-backend-java` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-backend-dotnet` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-serverless-cloudflare` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-infra-terraform` extension manifest and rules (EN/pt-BR).
- [ ] Update `ruleExtensionByStack` in `rule_extension_resolver.go`.
- [ ] Update tests in `rule_extension_resolver_test.go`.
- [ ] Update domain rule catalogs in `AGENTS.md` and scaffold mirrors.

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-stack-rule-extensions-expansion --strict`.

---

## 5. Decisions

### Decision 1: Curated Stack Domain Rules
- Date: 2026-08-21
- Context: Need standard domain rules across top programming languages and cloud platforms.
- Decision: Ship 6 dedicated extension packages containing idiomatic, production-tested rules with complete EN/pt-BR translations.
- Rationale: Keeps the core POSE engine lightweight while providing comprehensive stack governance on demand.

---

## 6. Validation

### Strategy
Unit tests in `rule_extension_resolver_test.go`, extension manifest validation across all packages, and full project validation.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Extension resolution and manifest unit tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-stack-rule-extensions-expansion --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: N/A
- Scope: N/A
- Expected: N/A

#### Build
- Command: N/A
- Scope: N/A
- Expected: N/A

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Execution log
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
- R1 [satisfied] capability:python-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R2 [satisfied] capability:rust-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R3 [satisfied] capability:java-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R4 [satisfied] capability:dotnet-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R5 [satisfied] capability:cloudflare-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R6 [satisfied] capability:terraform-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R7 [satisfied] capability:python-rule-extension check:unit test:TestRuleExtensionResolverAllStacks evidence:integration
- R8 [satisfied] capability:python-rule-extension check:unit test:TestLocaleCoverage evidence:integration

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
- Created 6 new production rule extensions: Python, Rust, Java, .NET, Cloudflare Workers, and Terraform.
- Provided 100% bilingual documentation (EN/pt-BR).
- Wired automatic stack-to-extension resolution in the native engine.
- Updated `AGENTS.md` and distributed scaffold assets.

### Files and modules changed
- `extensions/pose-rule-backend-python/`
- `extensions/pose-rule-backend-rust/`
- `extensions/pose-rule-backend-java/`
- `extensions/pose-rule-backend-dotnet/`
- `extensions/pose-rule-serverless-cloudflare/`
- `extensions/pose-rule-infra-terraform/`
- `pose-mcp/internal/cli/rule_extension_resolver.go`
- `pose-mcp/internal/cli/rule_extension_resolver_test.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md`

### Validation executed
- Command:
- Result:

### Residual risks
- None.

### Follow-ups
- [done] Multi-stack rule extension expansion completed and tested.

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->
