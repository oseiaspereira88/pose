# Integration Assessment: pose-dist

> **Gerado por**: POSE Integration Engine (`pose assess integrate`)
> **Data de Avaliação**: 2026-08-06T22:30:08Z
> **Baseline Commit**: 2b9e3426159c

## 1. Resumo Executivo

- **Total de Contratos Observados**: 50
- **Contratos com Provedor e Consumidor**: 1
- **Gaps de Integração**: 49

## 2. Matriz de Contratos

| Nome do Contrato | Protocolo | Provedor | Consumidor | Status |
|---|---|---|---|---|
| MCP tool conductor_run_close | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool conductor_run_event | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool conductor_run_open | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_capability_history | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_capability_stale | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_capability_state | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_check | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_closeout_state | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_component_discover | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_delivery_integrity | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_docs_state | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_extension_list | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_assessment | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_changelog | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_followups | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_integration_matrix | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_knowledge | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_report | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_roadmap | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_rules | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_skill | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_spec | `mcp` | `pose-mcp` | `mcp-enforce` | `active` |
| MCP tool pose_get_tech_debt_report | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_get_workflow | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_insights | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_integration_check | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_lint_spec | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_list_assessments | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_list_knowledge | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_list_reports | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_list_roadmaps | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_list_specs | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_mcp_context | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_project_state | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_release_status | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_requirement_trace | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_skills_check | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_spec_amendments | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_spec_readiness | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_suggest | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_surface_assurance | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_tech_debt_check | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_approve | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_cancel | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_request | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_status | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_submit | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /admin/refresh | `rest` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /healthz | `rest` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /mcp | `rest` | `pose-mcp` | `unobserved` | `gap` |

## 3. Gaps Observados

### [GAP-001] No consumer observed for MCP tool conductor_run_close
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-002] No consumer observed for MCP tool conductor_run_event
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-003] No consumer observed for MCP tool conductor_run_open
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-004] No consumer observed for MCP tool pose_capability_history
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-005] No consumer observed for MCP tool pose_capability_stale
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-006] No consumer observed for MCP tool pose_capability_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-007] No consumer observed for MCP tool pose_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-008] No consumer observed for MCP tool pose_closeout_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-009] No consumer observed for MCP tool pose_component_discover
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-010] No consumer observed for MCP tool pose_delivery_integrity
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-011] No consumer observed for MCP tool pose_docs_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-012] No consumer observed for MCP tool pose_extension_list
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-013] No consumer observed for MCP tool pose_get_assessment
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-014] No consumer observed for MCP tool pose_get_changelog
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-015] No consumer observed for MCP tool pose_get_followups
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-016] No consumer observed for MCP tool pose_get_integration_matrix
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-017] No consumer observed for MCP tool pose_get_knowledge
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-018] No consumer observed for MCP tool pose_get_report
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-019] No consumer observed for MCP tool pose_get_roadmap
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-020] No consumer observed for MCP tool pose_get_rules
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-021] No consumer observed for MCP tool pose_get_skill
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-022] No consumer observed for MCP tool pose_get_tech_debt_report
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-023] No consumer observed for MCP tool pose_get_workflow
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-024] No consumer observed for MCP tool pose_insights
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-025] No consumer observed for MCP tool pose_integration_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-026] No consumer observed for MCP tool pose_lint_spec
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-027] No consumer observed for MCP tool pose_list_assessments
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-028] No consumer observed for MCP tool pose_list_knowledge
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-029] No consumer observed for MCP tool pose_list_reports
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-030] No consumer observed for MCP tool pose_list_roadmaps
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-031] No consumer observed for MCP tool pose_list_specs
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-032] No consumer observed for MCP tool pose_mcp_context
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-033] No consumer observed for MCP tool pose_project_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-034] No consumer observed for MCP tool pose_release_status
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-035] No consumer observed for MCP tool pose_requirement_trace
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-036] No consumer observed for MCP tool pose_skills_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-037] No consumer observed for MCP tool pose_spec_amendments
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-038] No consumer observed for MCP tool pose_spec_readiness
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-039] No consumer observed for MCP tool pose_suggest
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-040] No consumer observed for MCP tool pose_surface_assurance
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-041] No consumer observed for MCP tool pose_tech_debt_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-042] No consumer observed for MCP tool pose_validate_approve
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-043] No consumer observed for MCP tool pose_validate_cancel
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-044] No consumer observed for MCP tool pose_validate_request
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-045] No consumer observed for MCP tool pose_validate_status
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-046] No consumer observed for MCP tool pose_validate_submit
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-047] No consumer observed for HTTP /admin/refresh
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-048] No consumer observed for HTTP /healthz
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-049] No consumer observed for HTTP /mcp
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.
