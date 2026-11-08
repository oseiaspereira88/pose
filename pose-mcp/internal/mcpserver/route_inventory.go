package mcpserver

import "github.com/harne8/pose-mcp/internal/pose"

// mcpRouteInventory declares, for every MCP tool, how it relates to the CLI (spec
// pose-common-domain-and-cli-mcp-handoff R1). TestEveryMCPRouteHasADeclaredClass keeps
// it equal to the advertised catalog: a new tool cannot ship without a class.
//
//   - RouteCLIWrapper: pose_suggest and pose_get_followups run the CLI package's
//     read-only command in process (no binary is started); pose_check,
//     pose_skills_check and pose_lint_spec keep the process boundary because a gate
//     is long and can be killed on a deadline without leaving a half verdict; the
//     pose_delegate_* tools drive `pose delegate`, a legitimate external harness.
//   - RouteReadOnlyPlan: the plan or preview of an operation, which writes nothing;
//     pose_closeout_state with include_plan also describes, as a typed handoff, the
//     CLI operation the plan leaves to another actor.
//   - RouteAdmittedWrite: the write points of the domain, which revalidate
//     authority, assurance, policy and subjects themselves.
//   - RouteExternalHandoff: the MCP-Conductor-Harness boundary: request, approve,
//     submit and the Conductor reporter; none turns tools/call into the execution
//     of a project's checks.
var mcpRouteInventory = pose.RouteInventory{
	"conductor_run_close":               pose.RouteExternalHandoff,
	"conductor_run_event":               pose.RouteExternalHandoff,
	"conductor_run_open":                pose.RouteExternalHandoff,
	"pose_action_fulfil":                pose.RouteAdmittedWrite,
	"pose_action_open":                  pose.RouteAdmittedWrite,
	"pose_action_requests":              pose.RouteDomainDirect,
	"pose_action_resolve":               pose.RouteAdmittedWrite,
	"pose_capability_history":           pose.RouteDomainDirect,
	"pose_capability_stale":             pose.RouteDomainDirect,
	"pose_capability_state":             pose.RouteDomainDirect,
	"pose_check":                        pose.RouteCLIWrapper,
	"pose_closeout_state":               pose.RouteReadOnlyPlan,
	"pose_component_discover":           pose.RouteDomainDirect,
	"pose_delegate_handoff":             pose.RouteCLIWrapper,
	"pose_delegate_resume":              pose.RouteCLIWrapper,
	"pose_delegate_start":               pose.RouteCLIWrapper,
	"pose_delegate_status":              pose.RouteCLIWrapper,
	"pose_delegate_wait":                pose.RouteCLIWrapper,
	"pose_delivery_integrity":           pose.RouteDomainDirect,
	"pose_design_delta":                 pose.RouteDomainDirect,
	"pose_docs_state":                   pose.RouteDomainDirect,
	"pose_extension_list":               pose.RouteDomainDirect,
	"pose_federated_roadmap_acceptance": pose.RouteDomainDirect,
	"pose_get_assessment":               pose.RouteDomainDirect,
	"pose_get_changelog":                pose.RouteDomainDirect,
	"pose_get_followups":                pose.RouteCLIWrapper,
	"pose_get_integration_matrix":       pose.RouteDomainDirect,
	"pose_get_knowledge":                pose.RouteDomainDirect,
	"pose_get_report":                   pose.RouteDomainDirect,
	"pose_get_roadmap":                  pose.RouteDomainDirect,
	"pose_get_rules":                    pose.RouteDomainDirect,
	"pose_get_skill":                    pose.RouteDomainDirect,
	"pose_get_spec":                     pose.RouteDomainDirect,
	"pose_get_tech_debt_report":         pose.RouteDomainDirect,
	"pose_get_workflow":                 pose.RouteDomainDirect,
	"pose_governance_stats":             pose.RouteDomainDirect,
	"pose_insights":                     pose.RouteDomainDirect,
	"pose_integration_check":            pose.RouteDomainDirect,
	"pose_lint_spec":                    pose.RouteCLIWrapper,
	"pose_list_assessments":             pose.RouteDomainDirect,
	"pose_list_knowledge":               pose.RouteDomainDirect,
	"pose_list_reports":                 pose.RouteDomainDirect,
	"pose_list_roadmaps":                pose.RouteDomainDirect,
	"pose_list_specs":                   pose.RouteDomainDirect,
	"pose_mcp_context":                  pose.RouteDomainDirect,
	"pose_obligations":                  pose.RouteDomainDirect,
	"pose_project_state":                pose.RouteDomainDirect,
	"pose_release_status":               pose.RouteDomainDirect,
	"pose_requirement_trace":            pose.RouteDomainDirect,
	"pose_review_attest":                pose.RouteAdmittedWrite,
	"pose_review_bundle":                pose.RouteReadOnlyPlan,
	"pose_review_plan":                  pose.RouteReadOnlyPlan,
	"pose_review_prepare":               pose.RouteReadOnlyPlan,
	"pose_skills_check":                 pose.RouteCLIWrapper,
	"pose_spec_amendments":              pose.RouteDomainDirect,
	"pose_spec_readiness":               pose.RouteDomainDirect,
	"pose_spec_transfer_status":         pose.RouteReadOnlyPlan,
	"pose_start_status":                 pose.RouteReadOnlyPlan,
	"pose_suggest":                      pose.RouteCLIWrapper,
	"pose_surface_assurance":            pose.RouteDomainDirect,
	"pose_tech_debt_check":              pose.RouteDomainDirect,
	"pose_usage":                        pose.RouteDomainDirect,
	"pose_validate_approve":             pose.RouteExternalHandoff,
	"pose_validate_cancel":              pose.RouteExternalHandoff,
	"pose_validate_request":             pose.RouteExternalHandoff,
	"pose_validate_status":              pose.RouteExternalHandoff,
	"pose_validate_submit":              pose.RouteExternalHandoff,
}
