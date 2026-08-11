# Integration Assessment: pose-dist

> **Gerado por**: POSE Integration Engine (`pose assess integrate`)
> **Data de Avaliação**: 2026-08-10T23:21:58Z
> **Baseline Commit**: 83f5bfdf786c

## 1. Resumo Executivo

- **Total de Contratos Observados**: 51
- **Contratos com Provedor e Consumidor**: 1
- **Gaps de Integração**: 50

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
| MCP tool pose_usage | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_approve | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_cancel | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_request | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_status | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| MCP tool pose_validate_submit | `mcp` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /admin/refresh | `rest` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /healthz | `rest` | `pose-mcp` | `unobserved` | `gap` |
| HTTP /mcp | `rest` | `pose-mcp` | `unobserved` | `gap` |

## 3. Gaps Observados

### [GAP-3dcadd4f] No consumer observed for MCP tool conductor_run_close
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-253d911d] No consumer observed for MCP tool conductor_run_event
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-96ba2db7] No consumer observed for MCP tool conductor_run_open
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-42f38156] No consumer observed for MCP tool pose_capability_history
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-526d5db1] No consumer observed for MCP tool pose_capability_stale
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-1723b6ee] No consumer observed for MCP tool pose_capability_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-3b191a46] No consumer observed for MCP tool pose_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-d83dfa41] No consumer observed for MCP tool pose_closeout_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-2b269c89] No consumer observed for MCP tool pose_component_discover
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-4d3323ce] No consumer observed for MCP tool pose_delivery_integrity
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-a0de8c4e] No consumer observed for MCP tool pose_docs_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-28319ba1] No consumer observed for MCP tool pose_extension_list
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-6beb18b0] No consumer observed for MCP tool pose_get_assessment
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-4591b983] No consumer observed for MCP tool pose_get_changelog
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c243e080] No consumer observed for MCP tool pose_get_followups
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-5c88244f] No consumer observed for MCP tool pose_get_integration_matrix
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-6184cb5e] No consumer observed for MCP tool pose_get_knowledge
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-ca59e900] No consumer observed for MCP tool pose_get_report
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-49656cfd] No consumer observed for MCP tool pose_get_roadmap
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-2c4566b3] No consumer observed for MCP tool pose_get_rules
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-bcb0d677] No consumer observed for MCP tool pose_get_skill
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c631ddfd] No consumer observed for MCP tool pose_get_tech_debt_report
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-6e11b88a] No consumer observed for MCP tool pose_get_workflow
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-52b62b39] No consumer observed for MCP tool pose_insights
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-efbf903c] No consumer observed for MCP tool pose_integration_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-de0a1bcc] No consumer observed for MCP tool pose_lint_spec
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-b30f7eaf] No consumer observed for MCP tool pose_list_assessments
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-a92d9477] No consumer observed for MCP tool pose_list_knowledge
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c74bb593] No consumer observed for MCP tool pose_list_reports
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-3546aaa0] No consumer observed for MCP tool pose_list_roadmaps
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c158e52d] No consumer observed for MCP tool pose_list_specs
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-d2b3bba3] No consumer observed for MCP tool pose_mcp_context
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-95c9942d] No consumer observed for MCP tool pose_project_state
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-4be1e8a3] No consumer observed for MCP tool pose_release_status
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-8c1171f0] No consumer observed for MCP tool pose_requirement_trace
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-e0b19772] No consumer observed for MCP tool pose_skills_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c478d47e] No consumer observed for MCP tool pose_spec_amendments
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-3337a34d] No consumer observed for MCP tool pose_spec_readiness
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-47837534] No consumer observed for MCP tool pose_suggest
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-4e92b6ec] No consumer observed for MCP tool pose_surface_assurance
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-2a822d27] No consumer observed for MCP tool pose_tech_debt_check
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c94b5e59] No consumer observed for MCP tool pose_usage
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-c8c850e8] No consumer observed for MCP tool pose_validate_approve
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-5424a827] No consumer observed for MCP tool pose_validate_cancel
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-bb4e7fbb] No consumer observed for MCP tool pose_validate_request
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-e32ba49c] No consumer observed for MCP tool pose_validate_status
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-4b014c9c] No consumer observed for MCP tool pose_validate_submit
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-7baa2119] No consumer observed for HTTP /admin/refresh
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-61862039] No consumer observed for HTTP /healthz
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.

### [GAP-0338c2bf] No consumer observed for HTTP /mcp
- **Severidade**: medium
- **Provedor**: `pose-mcp`
- **Consumidor**: `unobserved`
- **Evidência**: A provider declaration was observed, but no repository consumer reference was found.
