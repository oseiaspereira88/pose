package pose

// Delegation policy schema 2 (spec pose-delegation-surface R6, R7; ADR
// 2026-10-26-delegation-is-a-first-class-execution-surface).
//
// .pose/policy/delegation.json replaces reviewers.json. It declares agents,
// routing and per-purpose budgets. The delivered dispatch, ledger, continuity,
// provenance and containment paths keep reading a ReviewerAdapter: each agent
// is projected for a purpose into exactly the adapter those paths verify, so a
// schema 1 policy migrated to schema 2 projects byte-identical adapters and the
// runs recorded under it keep verifying.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// DelegationPolicySchemaVersion versions .pose/policy/delegation.json.
const DelegationPolicySchemaVersion = 2

const (
	delegationPolicyPath = ".pose/policy/delegation.json"
	reviewersPolicyPath  = ".pose/policy/reviewers.json"
)

// Delegation purposes. A purpose is what a run is for; review, adjudicate and
// smoke share the review path and its guarantees, task does not.
const (
	DelegationPurposeReview     = "review"
	DelegationPurposeAdjudicate = "adjudicate"
	DelegationPurposeSmoke      = "smoke"
	DelegationPurposeTask       = "task"
)

// Delegation access levels, from the narrowest.
const (
	DelegationAccessRead      = "read"
	DelegationAccessSandboxed = "sandboxed"
	DelegationAccessWrite     = "write"
	DelegationAccessFull      = "full"
)

// Event formats an agent's stream can be parsed with (R9).
const (
	DelegationEventsCodexJSONL       = "codex-jsonl"
	DelegationEventsClaudeStreamJSON = "claude-stream-json"
	DelegationEventsPlain            = "plain"
)

var delegationPurposes = []string{DelegationPurposeReview, DelegationPurposeAdjudicate, DelegationPurposeSmoke, DelegationPurposeTask}
var delegationAccessLevels = []string{DelegationAccessRead, DelegationAccessSandboxed, DelegationAccessWrite, DelegationAccessFull}

// DelegationPurposeForKind maps a brief kind to the purpose that runs it.
func DelegationPurposeForKind(kind string) string {
	switch kind {
	case "adjudication":
		return DelegationPurposeAdjudicate
	case "smoke":
		return DelegationPurposeSmoke
	default:
		return DelegationPurposeReview
	}
}

// DelegationKindForPurpose maps a review-path purpose to its brief kind.
func DelegationKindForPurpose(purpose string) string {
	switch purpose {
	case DelegationPurposeAdjudicate:
		return "adjudication"
	case DelegationPurposeSmoke:
		return "smoke"
	default:
		return "review"
	}
}

// ReviewerEffort maps an effort level to the argv fragment that stands where
// the template has the whole element "{effort}".
type ReviewerEffort struct {
	Default string              `json:"default"`
	Levels  map[string][]string `json:"levels"`
}

// DelegationAgent is one configured CLI profile.
type DelegationAgent struct {
	Vendor           string               `json:"vendor"`
	Model            string               `json:"model"`
	CLI              string               `json:"cli,omitempty"`
	Events           string               `json:"events,omitempty"`
	Commands         map[string][]string  `json:"commands"`
	Effort           *ReviewerEffort      `json:"effort,omitempty"`
	Timeout          string               `json:"timeout,omitempty"`
	MaxRunsPerBundle int                  `json:"max_runs_per_bundle,omitempty"`
	Limit            *ReviewerLimit       `json:"limit,omitempty"`
	Session          *ReviewerSession     `json:"session,omitempty"`
	Resume           *ReviewerResume      `json:"resume,omitempty"`
	Executor         string               `json:"executor,omitempty"`
	Containment      *ReviewerContainment `json:"containment,omitempty"`
	// Probe is an optional availability check (spec pose-delegation-routing
	// R6). It is not part of the adapter the review path verifies, so it never
	// changes the adapter digest a run's provenance binds; adding one is still a
	// policy edit, which changes the policy baseline as any edit does (D5).
	Probe *DelegationProbe `json:"probe,omitempty"`
	// OnTerminal runs once after a run of this agent ends (spec
	// pose-delegated-review-completion-signal R7); it wins over the purpose's.
	OnTerminal *DelegationHook `json:"on_terminal,omitempty"`
}

// DelegationHook is a command run after a run ends, with the run identifier
// and state in its environment and nothing else.
type DelegationHook struct {
	Command []string `json:"command"`
	Timeout string   `json:"timeout,omitempty"`
}

// DelegationProbe is a command whose success means the agent can run now.
type DelegationProbe struct {
	Command []string `json:"command"`
	Timeout string   `json:"timeout,omitempty"`
}

// Routing defaults (spec pose-delegation-routing R5, R7).
const (
	DelegationPreferOpposite    = "opposite-vendor-of-implementation"
	DelegationPreferAny         = "any"
	DelegationStepOtherModel    = "same-vendor-other-model"
	DelegationStepWait          = "wait"
	DelegationStepSameModel     = "same-model"
	delegationDefaultMaxWait    = 5 * time.Hour
	delegationProbeDefaultBound = 10 * time.Second
	delegationProbeMaxBound     = 60 * time.Second
	delegationHookDefaultBound  = 30 * time.Second
	delegationHookMaxBound      = 5 * time.Minute
)

// RoutingPrefer is the preference in force.
func (r *DelegationRouting) RoutingPrefer() string {
	if r == nil || r.Prefer == "" {
		return DelegationPreferOpposite
	}
	return r.Prefer
}

// RoutingSteps are the fallback steps in force.
func (r *DelegationRouting) RoutingSteps() []string {
	if r == nil || r.OnUnavailable == nil {
		return []string{DelegationStepOtherModel, DelegationStepWait, DelegationStepSameModel}
	}
	return append([]string{}, r.OnUnavailable...)
}

// RoutingMaxWait bounds the wait step.
func (r *DelegationRouting) RoutingMaxWait() time.Duration {
	if r == nil || r.MaxWait == "" {
		return delegationDefaultMaxWait
	}
	d, _ := time.ParseDuration(r.MaxWait)
	return d
}

// DelegationRouting orders the candidates of --agent auto (spec
// pose-delegation-routing R5–R7).
type DelegationRouting struct {
	Prefer        string   `json:"prefer,omitempty"`
	OnUnavailable []string `json:"on_unavailable,omitempty"`
	Rounds        string   `json:"rounds,omitempty"`
	// LeadModels names the model of each vendor's leading agent, so a sealed
	// identity can establish a model difference (spec pose-delegation-routing
	// R2, D2); MaxWait bounds the wait step (R7).
	LeadModels map[string]string `json:"lead_models,omitempty"`
	MaxWait    string            `json:"max_wait,omitempty"`
}

// DelegationPurposePolicy is the budget of one purpose.
type DelegationPurposePolicy struct {
	MaxRunsPerBundle int             `json:"max_runs_per_bundle,omitempty"`
	Timeout          string          `json:"timeout,omitempty"`
	OnTerminal       *DelegationHook `json:"on_terminal,omitempty"`
}

// DelegationPolicy is .pose/policy/delegation.json.
type DelegationPolicy struct {
	SchemaVersion int                                `json:"schema_version"`
	Agents        map[string]DelegationAgent         `json:"agents"`
	Routing       *DelegationRouting                 `json:"routing,omitempty"`
	Purposes      map[string]DelegationPurposePolicy `json:"purposes,omitempty"`
	EnvUnset      []string                           `json:"env_unset,omitempty"`
	Provenance    *ReviewRunProvenancePolicy         `json:"provenance,omitempty"`
	// MigratedFrom names the schema 1 file this policy was converted from
	// and its archived bytes, so runs recorded before the conversion keep
	// their policy baseline (spec pose-delegation-surface D7).
	MigratedFrom *DelegationMigration `json:"migrated_from,omitempty"`
	// agentOrder is the order the file declares its agents in, which routing
	// follows (spec pose-delegation-routing R5).
	agentOrder []string
}

// DelegationMigration records a pose update conversion.
type DelegationMigration struct {
	Path    string `json:"path"`
	Archive string `json:"archive"`
	Digest  string `json:"digest"`
}

// delegationArchivePath keeps the converted reviewers.json outside
// .pose/policy, whose digest is every run's policy baseline.
const delegationArchivePath = ".pose/policy-migrations/reviewers.json"

// ErrDelegationPolicyConflict means both policy files exist.
var ErrDelegationPolicyConflict = errors.New("pose: both .pose/policy/delegation.json and .pose/policy/reviewers.json exist; keep delegation.json and remove reviewers.json (pose update converts one into the other)")

// LoadDelegationPolicy reads the schema 2 policy. With only reviewers.json it
// returns that policy converted, so every reader sees schema 2; with both it
// refuses. found is false when neither exists.
func (s Store) LoadDelegationPolicy() (DelegationPolicy, string, bool, error) {
	raw2, err2 := ReadReviewAdapterFile(filepath.Join(s.Root, filepath.FromSlash(delegationPolicyPath)), 1<<20)
	raw1, err1 := ReadReviewAdapterFile(filepath.Join(s.Root, filepath.FromSlash(reviewersPolicyPath)), 1<<20)
	if err2 != nil && !errors.Is(err2, os.ErrNotExist) {
		return DelegationPolicy{}, "", false, err2
	}
	if err1 != nil && !errors.Is(err1, os.ErrNotExist) {
		return DelegationPolicy{}, "", false, err1
	}
	switch {
	case err2 == nil && err1 == nil:
		return DelegationPolicy{}, "", false, ErrDelegationPolicyConflict
	case err2 == nil:
		policy, err := ParseDelegationPolicy(raw2)
		return policy, delegationPolicyPath, err == nil, err
	case err1 == nil:
		legacy, err := parseReviewersPolicy(raw1)
		if err != nil {
			return DelegationPolicy{}, "", false, err
		}
		policy, err := MigrateReviewersPolicy(legacy)
		return policy, reviewersPolicyPath, err == nil, err
	}
	return DelegationPolicy{}, "", false, nil
}

// ParseDelegationPolicy decodes and validates schema 2 strictly.
func ParseDelegationPolicy(raw []byte) (DelegationPolicy, error) {
	var policy DelegationPolicy
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&policy); err != nil {
		return DelegationPolicy{}, fmt.Errorf("pose: invalid %s: %w", delegationPolicyPath, err)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return DelegationPolicy{}, fmt.Errorf("pose: invalid %s: data after the policy object", delegationPolicyPath)
	}
	if err := exactJSONKeys(json.NewDecoder(bytes.NewReader(raw)), reflect.TypeOf(policy)); err != nil {
		return DelegationPolicy{}, fmt.Errorf("pose: invalid %s: %w", delegationPolicyPath, err)
	}
	if err := validateExplicitDelegationBudgets(raw); err != nil {
		return DelegationPolicy{}, err
	}
	if policy.SchemaVersion != DelegationPolicySchemaVersion {
		return DelegationPolicy{}, fmt.Errorf("pose: %s schema_version %d is not supported", delegationPolicyPath, policy.SchemaVersion)
	}
	if err := validateDelegationPolicy(policy); err != nil {
		return DelegationPolicy{}, err
	}
	policy.agentOrder = declaredObjectKeys(raw, "agents")
	return policy, nil
}

// declaredObjectKeys returns the keys of the top-level object field, in the
// order the document writes them.
func declaredObjectKeys(raw []byte, field string) []string {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(doc[field]))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	keys := []string{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		key, _ := tok.(string)
		keys = append(keys, key)
		var skip json.RawMessage
		if dec.Decode(&skip) != nil {
			return nil
		}
	}
	return keys
}

// exactJSONKeys refuses a key that names a field by another spelling, or that
// an object repeats: encoding/json matches field names without regard to case
// and keeps the last duplicate, so such a key would set a value the explicit
// checks below never read.
func exactJSONKeys(dec *json.Decoder, t reflect.Type) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '[':
		elem := t
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			elem = t.Elem()
		}
		for dec.More() {
			if err := exactJSONKeys(dec, elem); err != nil {
				return err
			}
		}
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return err
			}
			key, _ := keyTok.(string)
			if seen[key] {
				return fmt.Errorf("key %q appears twice in one object", key)
			}
			seen[key] = true
			next := t
			switch t.Kind() {
			case reflect.Map:
				next = t.Elem()
			case reflect.Struct:
				field, ok := exactJSONField(t, key)
				if !ok {
					return fmt.Errorf("key %q is not spelled as a field of this object (field names are exact)", key)
				}
				next = field
			}
			if err := exactJSONKeys(dec, next); err != nil {
				return err
			}
		}
	}
	_, err = dec.Token()
	return err
}

func exactJSONField(t reflect.Type, key string) (reflect.Type, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name == key && f.IsExported() {
			return f.Type, true
		}
	}
	return nil, false
}

// validateExplicitDelegationBudgets refuses a budget or timeout written as
// zero, null or empty, as schema 1 refuses an explicit zero budget: omit a
// field to take the default.
func validateExplicitDelegationBudgets(raw []byte) error {
	var doc struct {
		Agents   map[string]map[string]json.RawMessage `json:"agents"`
		Purposes map[string]map[string]json.RawMessage `json:"purposes"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	check := func(where string, fields map[string]json.RawMessage) error {
		if value, ok := fields["max_runs_per_bundle"]; ok {
			var n int
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &n) != nil || n < 1 || n > 100 {
				return fmt.Errorf("pose: %s: explicit max_runs_per_bundle must be 1..100; omit it for the default", where)
			}
		}
		if value, ok := fields["timeout"]; ok {
			var d string
			if json.Unmarshal(value, &d) != nil || d == "" {
				return fmt.Errorf("pose: %s: explicit timeout must be a positive duration; omit it for the default", where)
			}
		}
		return nil
	}
	for name, fields := range doc.Agents {
		if err := check("delegation agent "+name, fields); err != nil {
			return err
		}
	}
	for name, fields := range doc.Purposes {
		if err := check("delegation purpose "+name, fields); err != nil {
			return err
		}
	}
	return nil
}

func validateDelegationPolicy(policy DelegationPolicy) error {
	for name, purpose := range policy.Purposes {
		if !containsString(delegationPurposes, name) {
			return fmt.Errorf("pose: delegation purpose %q is not one of %s", name, strings.Join(delegationPurposes, ", "))
		}
		if err := validateDelegationHook(purpose.OnTerminal); err != nil {
			return fmt.Errorf("pose: delegation purpose %s on_terminal: %w", name, err)
		}
		if purpose.MaxRunsPerBundle < 0 || purpose.MaxRunsPerBundle > 100 {
			return fmt.Errorf("pose: delegation purpose %s: max_runs_per_bundle must be 1..100; omit for default 3", name)
		}
		if purpose.Timeout != "" {
			if d, err := time.ParseDuration(purpose.Timeout); err != nil || d <= 0 {
				return fmt.Errorf("pose: delegation purpose %s: timeout must be a positive duration", name)
			}
		}
	}
	if r := policy.Routing; r != nil {
		if r.Prefer != "" && r.Prefer != "opposite-vendor-of-implementation" && r.Prefer != "any" {
			return fmt.Errorf("pose: routing.prefer %q is not known", r.Prefer)
		}
		for _, step := range r.OnUnavailable {
			switch step {
			case "same-vendor-other-model", "wait", "same-model":
			default:
				return fmt.Errorf("pose: routing.on_unavailable step %q is not known", step)
			}
		}
		if r.Rounds != "" && r.Rounds != "continue-thread" && r.Rounds != "cold-session" {
			return fmt.Errorf("pose: routing.rounds %q is not known", r.Rounds)
		}
		seenStep := map[string]bool{}
		for _, step := range r.OnUnavailable {
			if seenStep[step] {
				return fmt.Errorf("pose: routing.on_unavailable names %s twice", step)
			}
			seenStep[step] = true
		}
		for vendor, model := range r.LeadModels {
			if _, _, err := ParseLead(vendor + "/" + model); err != nil || strings.Contains(vendor, "/") || model == "" || vendor != strings.ToLower(vendor) || model != strings.ToLower(model) {
				return fmt.Errorf("pose: routing.lead_models %q: %q is not a vendor and a model", vendor, model)
			}
		}
		if r.MaxWait != "" {
			if d, err := time.ParseDuration(r.MaxWait); err != nil || d < 0 || d > 24*time.Hour {
				return fmt.Errorf("pose: routing.max_wait must be a duration from 0 to 24h")
			}
		}
	}
	for _, name := range policy.EnvUnset {
		if !validEnvName(name) {
			return fmt.Errorf("pose: env_unset entry %q is not an environment variable name", name)
		}
	}
	for name, agent := range policy.Agents {
		if ValidateSlug(name) != nil {
			return fmt.Errorf("pose: delegation agent name %q is not a slug", name)
		}
		if agent.Vendor == "" || agent.Model == "" {
			return fmt.Errorf("pose: delegation agent %s needs vendor and model", name)
		}
		switch agent.Events {
		case "", DelegationEventsCodexJSONL, DelegationEventsClaudeStreamJSON, DelegationEventsPlain:
		default:
			return fmt.Errorf("pose: delegation agent %s: events %q is not one of %s, %s, %s", name, agent.Events, DelegationEventsCodexJSONL, DelegationEventsClaudeStreamJSON, DelegationEventsPlain)
		}
		if len(agent.Commands) == 0 {
			return fmt.Errorf("pose: delegation agent %s declares no command", name)
		}
		for access, argv := range agent.Commands {
			if !containsString(delegationAccessLevels, access) {
				return fmt.Errorf("pose: delegation agent %s: access %q is not one of %s", name, access, strings.Join(delegationAccessLevels, ", "))
			}
			if err := validateDelegationArgv(argv, agent.Effort); err != nil {
				return fmt.Errorf("pose: delegation agent %s command %s: %w", name, access, err)
			}
		}
		if agent.Resume != nil && len(agent.Resume.Command) > 0 {
			if err := validateDelegationArgv(agent.Resume.Command, agent.Effort); err != nil {
				return fmt.Errorf("pose: delegation agent %s resume command: %w", name, err)
			}
		}
		if agent.Timeout != "" {
			if d, err := time.ParseDuration(agent.Timeout); err != nil || d <= 0 {
				return fmt.Errorf("pose: delegation agent %s: timeout must be a positive duration", name)
			}
		}
		if agent.Effort != nil {
			if len(agent.Effort.Levels) == 0 {
				return fmt.Errorf("pose: delegation agent %s: effort needs levels", name)
			}
			if _, ok := agent.Effort.Levels[agent.Effort.Default]; !ok {
				return fmt.Errorf("pose: delegation agent %s: effort default %q is not a declared level", name, agent.Effort.Default)
			}
			for level, fragment := range agent.Effort.Levels {
				for _, arg := range fragment {
					if strings.Contains(arg, "{") && delegationPlaceholder(arg) != "" {
						return fmt.Errorf("pose: delegation agent %s: effort level %s carries a placeholder", name, level)
					}
				}
			}
		}
		if err := validateDelegationHook(agent.OnTerminal); err != nil {
			return fmt.Errorf("pose: delegation agent %s on_terminal: %w", name, err)
		}
		if probe := agent.Probe; probe != nil {
			if err := validateDelegationArgv(probe.Command, nil); err != nil {
				return fmt.Errorf("pose: delegation agent %s probe: %w", name, err)
			}
			for _, arg := range probe.Command {
				if delegationPlaceholder(arg) != "" {
					return fmt.Errorf("pose: delegation agent %s probe: a probe takes no placeholder", name)
				}
			}
			if probe.Timeout != "" {
				if d, err := time.ParseDuration(probe.Timeout); err != nil || d <= 0 || d > delegationProbeMaxBound {
					return fmt.Errorf("pose: delegation agent %s probe: timeout must be a positive duration up to %s", name, delegationProbeMaxBound)
				}
			}
		}
		if agent.MaxRunsPerBundle < 0 || agent.MaxRunsPerBundle > 100 {
			return fmt.Errorf("pose: delegation agent %s: max_runs_per_bundle must be 1..100; omit for the purpose budget", name)
		}
		for _, purpose := range []string{DelegationPurposeReview, DelegationPurposeAdjudicate, DelegationPurposeSmoke} {
			if _, ok := agent.Commands[DelegationAccessRead]; !ok {
				continue
			}
			adapter, _ := policy.agentAdapter(name, purpose)
			if _, err := ResolveReviewerContainment(adapter); err != nil {
				return fmt.Errorf("pose: delegation agent %s: %w", name, err)
			}
			if err := validateContinuityAdapter(adapter); err != nil {
				return fmt.Errorf("pose: delegation agent %s: %w", name, err)
			}
		}
	}
	if m := policy.MigratedFrom; m != nil && (m.Path != reviewersPolicyPath || m.Archive != delegationArchivePath || !isSHA256(m.Digest)) {
		return fmt.Errorf("pose: migrated_from must name %s, its archive %s and the archive's digest", reviewersPolicyPath, delegationArchivePath)
	}
	return validateRunProvenancePolicy(policy.reviewersView(DelegationPurposeReview))
}

// validateDelegationHook checks an on_terminal command: an argv with no
// placeholder and a timeout up to five minutes.
func validateDelegationHook(hook *DelegationHook) error {
	if hook == nil {
		return nil
	}
	if err := validateDelegationArgv(hook.Command, nil); err != nil {
		return err
	}
	for _, arg := range hook.Command {
		if delegationPlaceholder(arg) != "" {
			return fmt.Errorf("a hook takes no placeholder; the run and its state are in its environment")
		}
	}
	if hook.Timeout != "" {
		if d, err := time.ParseDuration(hook.Timeout); err != nil || d <= 0 || d > delegationHookMaxBound {
			return fmt.Errorf("timeout must be a positive duration up to %s", delegationHookMaxBound)
		}
	}
	return nil
}

// delegationPlaceholder names the placeholder an argv element is, or "".
func delegationPlaceholder(arg string) string {
	switch arg {
	case "{output}", "{session}", "{effort}":
		return arg
	}
	return ""
}

func validateDelegationArgv(argv []string, effort *ReviewerEffort) error {
	if len(argv) == 0 || argv[0] == "" {
		return fmt.Errorf("the argv is empty")
	}
	for i, arg := range argv {
		if delegationPlaceholder(arg) != "" {
			if i == 0 {
				return fmt.Errorf("a placeholder cannot be the executable")
			}
			if arg == "{effort}" && effort == nil {
				return fmt.Errorf("{effort} needs the agent's effort levels")
			}
			continue
		}
		for _, p := range []string{"{output}", "{session}", "{effort}"} {
			if strings.Contains(arg, p) {
				return fmt.Errorf("%s must be a whole argv element, never interpolated", p)
			}
		}
	}
	return nil
}

// agentAdapter projects agent name for purpose onto the adapter the delivered
// review path reads. Budgets: the agent's own max_runs_per_bundle wins, then
// the purpose's; the purpose's timeout wins, then the agent's.
func (p DelegationPolicy) agentAdapter(name, purpose string) (ReviewerAdapter, bool) {
	agent, ok := p.Agents[name]
	if !ok {
		return ReviewerAdapter{}, false
	}
	budget := p.Purposes[purpose]
	adapter := ReviewerAdapter{
		Containment: agent.Containment, Executor: agent.Executor, Command: agent.Commands[DelegationAccessRead],
		Vendor: agent.Vendor, Model: agent.Model, Timeout: agent.Timeout, MaxRunsPerBundle: agent.MaxRunsPerBundle,
		Limit: agent.Limit, Session: agent.Session, Resume: agent.Resume, Events: agent.Events, Effort: agent.Effort,
	}
	if budget.Timeout != "" {
		adapter.Timeout = budget.Timeout
	}
	if adapter.MaxRunsPerBundle == 0 {
		adapter.MaxRunsPerBundle = budget.MaxRunsPerBundle
	}
	return adapter, true
}

// reviewersView is the schema 1 shape of the policy for one purpose: every
// agent that can read, projected for that purpose.
func (p DelegationPolicy) reviewersView(purpose string) ReviewersPolicy {
	view := ReviewersPolicy{SchemaVersion: ReviewersPolicySchemaVersion, Provenance: p.Provenance, Adapters: map[string]ReviewerAdapter{}}
	for name, agent := range p.Agents {
		if _, ok := agent.Commands[DelegationAccessRead]; !ok {
			continue
		}
		view.Adapters[name], _ = p.agentAdapter(name, purpose)
	}
	return view
}

// AgentNames lists the configured agents in order.
func (p DelegationPolicy) AgentNames() []string {
	names := make([]string, 0, len(p.Agents))
	for name := range p.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MigrateReviewersPolicy converts schema 1 into schema 2 (R7). Each adapter
// becomes an agent whose read command is the adapter's command; a budget all
// adapters share moves to the review-path purposes, a budget they do not
// share stays on each agent. The projection of every migrated agent is the
// adapter it came from, byte for byte.
func MigrateReviewersPolicy(legacy ReviewersPolicy) (DelegationPolicy, error) {
	policy := DelegationPolicy{SchemaVersion: DelegationPolicySchemaVersion, Agents: map[string]DelegationAgent{}, Provenance: legacy.Provenance}
	budgets := map[int]bool{}
	for _, adapter := range legacy.Adapters {
		budgets[adapter.MaxRunsPerBundle] = true
	}
	shared := 0
	if len(budgets) == 1 {
		for b := range budgets {
			shared = b
		}
	}
	for name, adapter := range legacy.Adapters {
		agent := DelegationAgent{Vendor: adapter.Vendor, Model: adapter.Model, Events: adapter.Events, Commands: map[string][]string{DelegationAccessRead: adapter.Command},
			Effort: adapter.Effort, Timeout: adapter.Timeout, Limit: adapter.Limit, Session: adapter.Session, Resume: adapter.Resume, Executor: adapter.Executor, Containment: adapter.Containment}
		if len(budgets) > 1 {
			agent.MaxRunsPerBundle = adapter.MaxRunsPerBundle
		}
		policy.Agents[name] = agent
	}
	if shared != 0 {
		policy.Purposes = map[string]DelegationPurposePolicy{}
		for _, purpose := range []string{DelegationPurposeReview, DelegationPurposeAdjudicate, DelegationPurposeSmoke} {
			policy.Purposes[purpose] = DelegationPurposePolicy{MaxRunsPerBundle: shared}
		}
	}
	for name, adapter := range legacy.Adapters {
		got, _ := policy.agentAdapter(name, DelegationPurposeReview)
		a, _ := json.Marshal(adapter)
		b, _ := json.Marshal(got)
		if !bytes.Equal(a, b) {
			return DelegationPolicy{}, fmt.Errorf("pose: reviewer adapter %s does not convert losslessly", name)
		}
	}
	return policy, nil
}

// CanonicalDelegationPolicy is the bytes pose update writes: indented JSON
// with sorted keys and a final newline, the same for the same policy.
func CanonicalDelegationPolicy(policy DelegationPolicy) ([]byte, error) {
	raw, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// MigrateReviewersPolicyFile converts root's reviewers.json into
// delegation.json bytes; ok is false when there is nothing to convert.
func MigrateReviewersPolicyFile(files Files, root string) (out []byte, ok bool, err error) {
	if _, statErr := files.Stat(filepath.Join(root, filepath.FromSlash(delegationPolicyPath))); statErr == nil {
		return nil, false, nil
	}
	raw, readErr := files.ReadFile(filepath.Join(root, filepath.FromSlash(reviewersPolicyPath)))
	if errors.Is(readErr, os.ErrNotExist) {
		return nil, false, nil
	}
	if readErr != nil {
		return nil, false, readErr
	}
	legacy, err := parseReviewersPolicy(raw)
	if err != nil {
		return nil, false, err
	}
	policy, err := MigrateReviewersPolicy(legacy)
	if err != nil {
		return nil, false, err
	}
	policy.MigratedFrom = &DelegationMigration{Path: reviewersPolicyPath, Archive: delegationArchivePath, Digest: digestBytes(raw)}
	out, err = CanonicalDelegationPolicy(policy)
	if err != nil {
		return nil, false, err
	}
	if _, err := ParseDelegationPolicy(out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func validEnvName(name string) bool {
	if name == "" || len(name) > 256 {
		return false
	}
	for i, r := range name {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// ExampleDelegationPolicy is what POSE suggests and never enables by itself.
func ExampleDelegationPolicy() DelegationPolicy {
	return DelegationPolicy{SchemaVersion: DelegationPolicySchemaVersion, Agents: map[string]DelegationAgent{
		"codex": {Vendor: "openai", Model: "gpt-6.1-sol", CLI: "codex", Events: DelegationEventsCodexJSONL, Timeout: "60m",
			Commands: map[string][]string{DelegationAccessRead: {"codex", "exec", "-m", "gpt-6.1-sol", "{effort}", "-s", "read-only", "--skip-git-repo-check", "--json", "-o", "{output}", "-"}},
			Effort:   &ReviewerEffort{Default: "medium", Levels: map[string][]string{"low": {"-c", `model_reasoning_effort="low"`}, "medium": {"-c", `model_reasoning_effort="medium"`}, "high": {"-c", `model_reasoning_effort="high"`}}}},
		"claude": {Vendor: "anthropic", Model: "claude-opus-5-5", CLI: "claude", Events: DelegationEventsClaudeStreamJSON, Timeout: "60m",
			Commands: map[string][]string{DelegationAccessRead: {"claude", "-p", "--model", "claude-opus-5-5", "--permission-mode", "plan", "--output-format", "stream-json", "--verbose"}}},
	}, Purposes: map[string]DelegationPurposePolicy{DelegationPurposeReview: {MaxRunsPerBundle: 3}}}
}

// DelegationArchivePath is where pose update keeps the converted schema 1 file.
func DelegationArchivePath() string { return delegationArchivePath }

// migratedPolicyBaseline is the digest .pose/policy had before pose update
// converted reviewers.json: the directory as it is, with delegation.json in
// the place of the archived reviewers.json. It is known only while
// delegation.json is exactly the conversion of that archive, so an edited
// policy never inherits the baseline of the one it replaced.
func (s Store) migratedPolicyBaseline() (string, bool) {
	files := s.fsys()
	dir := filepath.Join(s.Root, ".pose", "policy")
	current, err := files.ReadFile(filepath.Join(s.Root, filepath.FromSlash(delegationPolicyPath)))
	if err != nil {
		return "", false
	}
	policy, err := ParseDelegationPolicy(current)
	if err != nil || policy.MigratedFrom == nil {
		return "", false
	}
	archived, err := files.ReadFile(filepath.Join(s.Root, filepath.FromSlash(delegationArchivePath)))
	if err != nil || digestBytes(archived) != policy.MigratedFrom.Digest {
		return "", false
	}
	legacy, err := parseReviewersPolicy(archived)
	if err != nil {
		return "", false
	}
	again, err := MigrateReviewersPolicy(legacy)
	if err != nil {
		return "", false
	}
	again.MigratedFrom = policy.MigratedFrom
	expected, err := CanonicalDelegationPolicy(again)
	if err != nil || !bytes.Equal(expected, current) {
		return "", false
	}
	return directoryDigestReplacing(files, dir, "delegation.json", "reviewers.json", archived), true
}
