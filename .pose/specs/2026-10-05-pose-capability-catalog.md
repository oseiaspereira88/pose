---
slug: pose-capability-catalog
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:capability-catalog
---

# Spec: One catalog of adoptable capabilities, one command to decide each

## 1. Intent

### Goal

Describe every capability a project can adopt in one catalog — what it changes day to day, since which version, whether a new instance gets it, what it requires — and let `pose adopt` turn each on or off, or record that the project declined or deferred it, so nothing adoptable needs a hand-edited policy key and nothing decided is asked again.

### Business value

The 6.2.0 → current upgrade walked for the 7.0.0 review showed `pose adopt` toggling four capabilities while four others (qualified references, spec authority transfer, signed attestations, verified identity), every review overlay and the Definition of Ready still needed hand-edited JSON, some with a `schema_version` change. Nothing recorded that a project had looked at a capability and said no, so any later prompt would ask again. The setup map and the update review of this roadmap read from this catalog. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

Adopting never re-dates an adopted capability. A prerequisite that is not met refuses the adoption with what is missing. A decline is a recorded decision, not an absence. The policy reader decides every write.

### Non-goals

Configuring project-specific data (delivery roots, release provider); the catalog names those as configuration, not toggles.

## 2. Requirements

### Functional

- R1: The catalog shall list each adoptable capability with its id, effect, introducing version, whether a new instance adopts it, the capabilities it requires and any prerequisite outside the catalog, and its state in this instance: on, off, declined, deferred or needs-setup.
- R2: `pose adopt <capability>` shall adopt any catalog capability, including the review overlays, the Definition of Ready, qualified references and spec authority transfer (raising `schema_version` as their contract requires); `--off` shall reverse it and refuse while another adopted capability requires it.
- R3: An adoption whose required capability is off, or whose prerequisite is unmet, shall be refused with the missing item and the command that provides it.
- R4: `pose adopt <capability> --decline` or `--defer` with a reason shall record the decision in `.pose/policy/adoption-decisions.json`; adopting later shall clear it; declining an adopted capability shall be refused.
- R5: `pose adopt --list` shall print the catalog with each state, and `--json` the same data.
- R6: The decisions file shall never be shipped in the distribution, and the manual shall describe the catalog and the decision record.

### Non-functional

- No network or Git access.

### Security

- Overlay adoption goes through the same dated rule as causality closeout, so a late overlay never re-judges work in flight.

### Compatibility

- Existing policies read unchanged; `pose adopt` keeps its four-capability behaviour.

## 3. Technical Plan

### Affected areas

Capability registry, `pose adopt`, effective governance, distribution policy, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-capability-catalog.md
- created: .pose/starts/pose-capability-catalog.json
- created: pose-mcp/internal/pose/capability_catalog.go
- created: pose-mcp/internal/pose/capability_catalog_test.go
- modified: pose-mcp/internal/pose/effective_governance.go
- modified: pose-mcp/internal/cli/adopt.go
- created: pose-mcp/internal/cli/adopt_catalog_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/policy_keys.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-capability-catalog.md

### Delivery targets

- capability:capability-catalog module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Lowering `schema_version` when a schema-bound capability is turned off could drop a key another feature reads; the reader validates the result before it is written.

## 6. Validation

### Strategy

Drive every catalog entry on and off through `pose adopt` on an installed fixture and read the result back through the policy reader; dependency and prerequisite refusals; decline, defer and re-adopt.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'CapabilityCatalog|AdoptCatalog'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestCapabilityCatalogDescribesEveryEntry test:TestCapabilityCatalogRecordsDeclineAndDefer check:capability-catalog-integration
- R2 [satisfied] test:TestCapabilityCatalogAdoptsAndRetiresEveryToggleThroughTheReader test:TestAdoptCatalogTogglesEveryCapabilityKind check:capability-catalog-integration
- R3 [satisfied] test:TestCapabilityCatalogRefusesMissingRequirementsAndPrerequisites test:TestAdoptCatalogTogglesEveryCapabilityKind check:capability-catalog-integration
- R4 [satisfied] test:TestCapabilityCatalogRecordsDeclineAndDefer test:TestAdoptCatalogRecordsDeclineAndDeferAndAdoptClearsThem check:capability-catalog-integration
- R5 [satisfied] test:TestAdoptCatalogRecordsDeclineAndDeferAndAdoptClearsThem check:capability-catalog-integration
- R6 [satisfied] test:TestCapabilityCatalogIsDocumented test:TestSelfReferentialPolicyFilesExcluded check:capability-catalog-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk.

### Follow-ups

None.
