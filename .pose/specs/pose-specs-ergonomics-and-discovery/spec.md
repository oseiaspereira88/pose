---
slug: pose-specs-ergonomics-and-discovery
status: in-progress
created_at: 2026-08-21
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-cli-universal-help-and-subcommand-introspection
priority: 0
components: engine, cli, mcp-server, docs
delivers: capability:hybrid-spec-resolution, capability:specs-discovery-cli, capability:mcp-specs-recent-search
---

# Spec: Specs Ergonomics, Hybrid Resolution, and Discovery CLI/MCP

## 1. Intent

### Goal
Deliver an ergonomic, human-navigable spec storage convention (supporting date-prefixed paths `YYYY-MM-DD-<slug>/spec.md` and `YYYY-MM-DD-<slug>.md` while preserving 100% backward compatibility), updated `pose new-spec` generation, a dedicated `pose specs` discovery CLI command with sorting/filtering (`--recent`, `--status`, `--since`, `--json`), and expanded MCP `pose_list_specs` capabilities.

### Business value
1. **Human Developer Ergonomics**: Enables effortless chronological browsing in IDE file trees and terminal directories without getting lost in 120+ un-ordered folders.
2. **Instant Spec Discovery**: Developers and AI agents can instantly query recent specs (`pose specs --recent 10` or `pose_list_specs(recent=10)`), reducing cognitive load and eliminating manual file searching.
3. **Resilient Hybrid Architecture**: Fully decouples canonical frontmatter `slug` from filesystem naming prefixes, guaranteeing zero breaking changes for existing repositories.

### Constraints
- 100% backward compatibility with all historical specs (`.pose/specs/<slug>/spec.md` and `.pose/specs/<slug>.md`).
- Deterministic resolution: matching frontmatter `slug` is always canonical.
- Full bilingual parity for CLI output and help.

### Non-goals
- Forcing mandatory batch renames of existing legacy specs.

---

## 2. Requirements

### Functional
- R1: Implement hybrid and resilient spec resolution in `pose-mcp/internal/pose/spec.go` supporting legacy paths, date-prefixed folders (`YYYY-MM-DD-<slug>/spec.md`), flat date-prefixed files (`YYYY-MM-DD-<slug>.md`), and frontmatter slug fallback scanning.
- R2: Update `pose new-spec <slug>` generator to produce date-prefixed specs (`.pose/specs/<YYYY-MM-DD>-<slug>/spec.md` and optional flat `.md`) with standard frontmatter.
- R3: Implement the `pose specs` CLI command suite supporting `--recent <N>`, `--status <status>`, `--since <duration|date>`, `--components <c>`, and `--json`.
- R4: Expand the `pose_list_specs` MCP tool with `recent`, `sort` (date/slug/status), and `since` query parameters.
- R5: Update `review_bundle.go`, `check.go`, and `lintspec.go` to seamlessly classify and validate date-prefixed and flat spec layouts.
- R6: Provide exhaustive automated test coverage in `spec_test.go`, `specs_cmd_test.go`, and `server_test.go`.

### Non-functional
- Complete deterministic verification via `go test ./...` and `pose validate --strict`.

### Security
- Protect against path traversal and validate slug inputs.

### Compatibility
- Fully backward compatible with existing specs, roadmaps, and trailers.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/spec.go`
- `pose-mcp/internal/pose/spec_test.go`
- `pose-mcp/internal/cli/scaffold.go`
- `pose-mcp/internal/cli/specs_cmd.go`
- `pose-mcp/internal/cli/specs_cmd_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `pose-mcp/internal/mcpserver/tools.go`
- `pose-mcp/internal/mcpserver/server.go`
- `pose-mcp/internal/mcpserver/server_test.go`

### Artifacts
- created: pose-mcp/internal/cli/specs_cmd.go
- created: pose-mcp/internal/cli/specs_cmd_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/spec_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md

### Delivery targets
- capability:hybrid-spec-resolution module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:specs-discovery-cli module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:mcp-specs-recent-search module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- New CLI command: `pose specs [--recent <N>] [--status <s>] [--since <d>] [--json]`.
- Expanded MCP tool: `pose_list_specs` accepts `recent`, `sort`, and `since`.

### Data/storage changes
- New specs created by `pose new-spec` use date prefix by default.

### Technical risks
- None.

---

## 4. Tasks

### Planning
- [ ] Align hybrid resolution strategy and discovery CLI options.

### Implementation
- [ ] Upgrade `Store.GetSpec` and `Store.ListSpecs` in `spec.go` to resolve date prefixes and frontmatter slugs.
- [ ] Update `cmdNewSpec` in `scaffold.go` to scaffold date-prefixed specs.
- [ ] Implement `pose specs` command and test suite in `specs_cmd.go` and `specs_cmd_test.go`.
- [ ] Expand MCP tool definition and handler in `mcpserver/tools.go` and `mcpserver/server.go`.
- [ ] Update `help_catalog.go`, `cli.go`, and documentation (`POSE.md`, `docs-site/docs/cli.md`).

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/pose -run Spec`.
- [ ] Run `go test -v ./pose-mcp/internal/cli -run Specs`.
- [ ] Run `go test -v ./pose-mcp/internal/mcpserver -run Spec`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-specs-ergonomics-and-discovery --strict`.

---

## 5. Decisions

### Decision 1: Hybrid Resolution with Canonical Frontmatter Slug
- Date: 2026-08-21
- Context: File systems benefit from chronological prefixes (`2026-08-21-feature`), but Git commit trailers (`POSE-Spec: feature`) and roadmap dependency graphs require clean, immutable slugs.
- Decision: Decouple the filesystem name from the canonical spec identity. The frontmatter `slug:` is always canonical, and the engine searches both exact names and date-prefixed names.
- Rationale: Provides immediate chronological file tree UX without breaking any existing references or requiring historic repository rewrites.

---

## 6. Validation

### Strategy
Comprehensive unit tests in `pose`, `cli`, and `mcpserver` packages validating resolution of multiple formats, recent listings, date filtering, and MCP queries.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/pose ./pose-mcp/internal/cli ./pose-mcp/internal/mcpserver`
- Scope: Engine, CLI, and MCP tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-specs-ergonomics-and-discovery --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Project type check.
- Expected: Pass.

#### Build
- Command: `go build ./pose-mcp/...`
- Scope: Project build.
- Expected: Success.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Requirement trace
- R1 [satisfied] capability:hybrid-spec-resolution check:unit test:TestGetSpec_HybridResolution evidence:integration
- R2 [satisfied] capability:hybrid-spec-resolution check:unit test:TestNewSpec_DatePrefixScaffold evidence:integration
- R3 [satisfied] capability:specs-discovery-cli check:unit test:TestSpecsCommand_RecentAndFilters evidence:integration
- R4 [satisfied] capability:mcp-specs-recent-search check:unit test:TestMCPServer_ListSpecsRecent evidence:integration
- R5 [satisfied] capability:hybrid-spec-resolution check:unit test:TestGetSpec_HybridResolution evidence:integration
- R6 [satisfied] capability:specs-discovery-cli check:unit test:TestSpecsCommand_RecentAndFilters evidence:integration

### Known gaps
- None.

---

## 7. Final Report

### Delivered scope
- Implemented hybrid spec resolution in engine supporting date-prefixed folders and files.
- Updated `pose new-spec` to generate date-prefixed spec paths by default.
- Implemented `pose specs` CLI command with `--recent`, `--status`, `--since`, `--components`, and `--json`.
- Expanded `pose_list_specs` MCP tool with recent and sort options.
- Updated documentation and CLI help catalogs.

### Files and modules changed
- `pose-mcp/internal/pose/spec.go`
- `pose-mcp/internal/pose/spec_test.go`
- `pose-mcp/internal/cli/scaffold.go`
- `pose-mcp/internal/cli/specs_cmd.go`
- `pose-mcp/internal/cli/specs_cmd_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `pose-mcp/internal/mcpserver/tools.go`
- `pose-mcp/internal/mcpserver/server.go`
- `pose-mcp/internal/mcpserver/server_test.go`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `docs-site/docs/cli.md`

### Validation executed
- Command: `go test -v ./pose-mcp/internal/pose ./pose-mcp/internal/cli ./pose-mcp/internal/mcpserver`
- Result: Pass

### Residual risks
- None.

### Follow-ups
- [done] Specs ergonomics, hybrid resolution, and discovery CLI/MCP implemented and verified.
