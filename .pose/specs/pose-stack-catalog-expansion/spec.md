---
slug: pose-stack-catalog-expansion
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-structured-validation-results
priority: 18
---

# Spec: Polyglot stack catalog

## 1. Intent

### Goal
add maintained profiles for Python, .NET and modern build ecosystems.
### Business value
Expands addressable repositories without low-level setup.
### Constraints
- Delegate to native tools and never download dependencies implicitly.
### Non-goals
- Certify every framework or replace overrides.

## 2. Requirements

### Functional
- R1: Python and .NET profiles shall detect standard markers and propose checks.
- R2: Profiles shall declare prerequisites, confidence and override behavior.
- R3: Fixtures shall cover absent tools, multiple managers and conflicting markers.

### Non-functional
- Keep detection offline, bounded and deterministic.

### Security
- Never execute project files during detection.

### Compatibility
- Existing Node.js, Go, Rust and Java selection remains unchanged.

## 3. Technical Plan

### Affected areas
- Detection, wizard, matrix, docs and fixtures.

### API/contract changes
- Version profile IDs and default check semantics.

### Data/storage changes
- Add maintainer, status and compatibility metadata.

### Technical risks
- Defaults can be expensive in large repos; expose overrides.

### Primary references
- [Python Packaging User Guide](https://packaging.python.org/)
- [.NET CLI build](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Python Packaging User Guide](https://packaging.python.org/): baseline matrix covered only node/go/rust/java; Python's manager plurality (poetry/pipenv/pip/setuptools/pep517) required a priority-resolution mechanism the existing single-field `when` predicates could not express.

### Implementation
- [x] Define profile lifecycle, support tiers and conflicts: `stackCatalog` (`internal/cli/stack_catalog.go`) — id, stack, manager, marker, prerequisite, priority; conflicts resolve by lowest priority number, reported with `confidence: medium` (ADR `2026-07-19-polyglot-stack-catalog-with-priority-resolution`). ([reference](https://packaging.python.org/))
- [x] Implement Python and .NET profiles without implicit installs: five Python manager profiles and four .NET marker profiles added to the catalog and to `.pose/indexes/validation-matrix.json`; detection is `os.ReadDir` + name/suffix match and `exec.LookPath` for prerequisites — no project file is ever executed, no dependency ever downloaded. New `validationWhen.FileNotExistsAny` predicate expresses Python manager exclusion in the matrix (`pip-test` yields to `poetry.lock`/`Pipfile`). ([reference](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build))
- [x] Add fixture compatibility tests and generated profile docs: `stack_catalog_test.go` covers conflict resolution, high-confidence single match, .NET suffix markers, absent-tool reporting, matrix-level manager exclusion and the `pose stacks` command (text + JSON + path confinement); `pose stacks` itself is the queryable "generated" profile doc (R2), documented in `POSE.md`. ([reference](https://packaging.python.org/))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Stack|Wizard|Matrix'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://packaging.python.org/))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-polyglot-stack-catalog-with-priority-resolution.md` (Accepted): marker-based profile catalog with declared priority and an additive `when.fileNotExistsAny` predicate, over executing project tooling to detect the manager (rejected: violates the no-execution constraint) and over one check per manager with no conflict handling (rejected: silently runs redundant or wrong checks).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Stack|Wizard|Matrix'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-stack-catalog-expansion --ready-check`.

### Requirement trace
- R1 [satisfied] Python/.NET profiles detect standard markers and pose stacks proposes the resolved manager/checks; check:test (TestDetectStackProfilesDotnetSuffixMarkers, TestStacksCommandJSONAndPathConfinement)
- R2 [satisfied] profiles declare prerequisites, confidence and override behavior, queryable via pose stacks; check:test (TestDetectStackProfilesNoConflictIsHighConfidence, TestDetectStackProfilesAbsentToolReported, TestStacksCommandReportsConflictAndOverrideHint) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] fixtures cover absent tools, multiple managers and conflicting markers; check:test (TestDetectStackProfilesResolvesConflictByPriority, TestValidationMatrixPythonManagerExclusion)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Stack|Detect|ValidationMatrixPython' -count=1` — SUCCESS (nine tests).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose stacks --path pose-mcp` on the live repository — reports `go (winner): marker=go.mod confidence=high prerequisite=go(found)`, confirming the catalog runs correctly against this repo's real Go module.
- `pose check --strict` — SUCCESS; `pose lint-spec pose-stack-catalog-expansion --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Maintained profile catalog covering Node.js, Go, Rust, Java, Python (five
managers) and .NET; `pose stacks` read-only detection command (text/JSON,
path-confined); `validationWhen.FileNotExistsAny` matrix predicate for
manager-priority exclusion; `python`/`dotnet` stacks added to
`validation-matrix.json` with structured, non-shell checks; existing
Node/Go/Rust/Java selection unchanged; operating-manual documentation and
ADR.

### Residual risks

- Defaults can be expensive or ambiguous in large repos (e.g. PEP
  517-only, no lockfile) — mitigated by the lowest-confidence tier
  defaulting to `optional` severity and the documented override path.
- .NET directories with multiple `.csproj` and no `.sln` may fail
  `dotnet test` without an explicit project argument — a native `dotnet`
  CLI limitation, documented rather than worked around.

### Follow-ups

- [open] Add fixture repositories (poetry, pipenv, dotnet solution) under tests/ to exercise pose stacks end-to-end once the monorepo recipes milestone lands. (owner:@pose-maintainers crit:low review:2026-11-20)
