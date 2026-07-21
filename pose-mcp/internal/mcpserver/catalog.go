package mcpserver

// Catalog governance (spec pose-mcp-catalog-conformance): the advertised tool
// catalog is a release-gated public contract. Every tool carries an explicit
// risk class, and optional tools declare their activation condition. The
// catalog_test.go golden fixture freezes names and schemas; changing either
// requires reviewing the golden diff, and removals or incompatible schema
// changes additionally require an ADR and a release note.

// RiskClass classifies the worst-case effect of invoking a tool.
type RiskClass string

const (
	// RiskRead tools only read repository-owned governance state.
	RiskRead RiskClass = "read"
	// RiskGate tools execute deterministic local gates (no writes, no network).
	RiskGate RiskClass = "gate"
	// RiskExternal tools emit events to an external system.
	RiskExternal RiskClass = "external-side-effect"
)

// toolGovernance is the per-tool governance record advertised to reviewers
// and frozen by the golden catalog fixture.
type toolGovernance struct {
	Risk RiskClass `json:"risk"`
	// Optional tools are always advertised in tools/list but only callable
	// when Activation holds; disabled calls return isError with guidance.
	Optional   bool   `json:"optional,omitempty"`
	Activation string `json:"activation,omitempty"`
}

const conductorActivation = "Conductor reporter configured (CONDUCTOR_URL, CONDUCTOR_RUN_TOKEN, CONDUCTOR_PROJECT_ID)"
const harnessActivation = "Harness executor configured via WithHarnessExecutor"

// catalogGovernance must cover exactly the tools returned by
// toolDefinitions(); catalog_test.go enforces the bijection.
var catalogGovernance = map[string]toolGovernance{
	"pose_get_spec":           {Risk: RiskRead},
	"pose_requirement_trace":  {Risk: RiskRead},
	"pose_capability_state":   {Risk: RiskRead},
	"pose_capability_history": {Risk: RiskRead},
	"pose_spec_amendments":    {Risk: RiskRead},
	"pose_list_specs":         {Risk: RiskRead},
	"pose_spec_readiness":     {Risk: RiskRead},
	"pose_get_changelog":      {Risk: RiskRead},
	"pose_list_roadmaps":      {Risk: RiskRead},
	"pose_get_roadmap":        {Risk: RiskRead},
	"pose_suggest":            {Risk: RiskRead},
	"pose_get_workflow":       {Risk: RiskRead},
	"pose_get_rules":          {Risk: RiskRead},
	"pose_insights":           {Risk: RiskRead},
	"pose_get_followups":      {Risk: RiskRead},
	"pose_check":              {Risk: RiskGate},
	"pose_skills_check":       {Risk: RiskGate},
	"pose_extension_list":     {Risk: RiskRead},
	"pose_lint_spec":          {Risk: RiskGate},
	"pose_list_knowledge":     {Risk: RiskRead},
	"pose_get_knowledge":      {Risk: RiskRead},
	"pose_list_reports":       {Risk: RiskRead},
	"pose_get_report":         {Risk: RiskRead},
	"pose_get_skill":          {Risk: RiskRead},
	"pose_validate_request":   {Risk: RiskGate},
	"pose_validate_approve":   {Risk: RiskGate},
	"pose_validate_submit":    {Risk: RiskExternal, Optional: true, Activation: harnessActivation},
	"pose_validate_status":    {Risk: RiskRead},
	"pose_validate_cancel":    {Risk: RiskGate},
	"conductor_run_open":      {Risk: RiskExternal, Optional: true, Activation: conductorActivation},
	"conductor_run_event":     {Risk: RiskExternal, Optional: true, Activation: conductorActivation},
	"conductor_run_close":     {Risk: RiskExternal, Optional: true, Activation: conductorActivation},
}
