package pose

// Routing of delegated runs (spec pose-delegation-routing R5–R9). --agent auto
// orders the agents that can run a purpose by routing.prefer, then by the steps
// of routing.on_unavailable, and takes the first available one. Availability is
// read from the path and from the recorded runs; only a configured probe runs
// anything, so routing never spends a run to discover a blocked vendor. Every
// run records the decision and the independence it achieved.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Availability states of an agent (R6).
const (
	DelegationAgentAvailable   = "available"
	DelegationAgentLimited     = "limited"
	DelegationAgentMissing     = "missing"
	DelegationAgentProbeFailed = "probe-failed"
	DelegationAgentIneligible  = "ineligible"
	// DelegationAgentUnknown means the run history could not be read whole:
	// a run that cannot be read may be the limit that blocks the agent.
	DelegationAgentUnknown = "unknown"
)

// DelegationStepPreferred and DelegationStepExplicit name how a run's agent
// was chosen besides the fallback steps.
const (
	DelegationStepPreferred = "preferred"
	DelegationStepExplicit  = "explicit"
)

// DelegationAvailability is what routing knows of an agent now.
type DelegationAvailability struct {
	Agent  string `json:"agent"`
	CLI    string `json:"cli"`
	OnPath bool   `json:"on_path"`
	// LimitResetAt is the reset of the last recorded usage limit; State is
	// limited only while the agent's last run is that limit and it is ahead.
	LimitResetAt string `json:"limit_reset_at,omitempty"`
	State        string `json:"state"`
	LimitedUntil string `json:"limited_until,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

// DelegationRoutingCandidate is one agent routing considered.
type DelegationRoutingCandidate struct {
	Agent   string `json:"agent"`
	Vendor  string `json:"vendor"`
	Model   string `json:"model"`
	Group   string `json:"group"`
	State   string `json:"state"`
	Skipped string `json:"skipped,omitempty"`
}

// DelegationRoutingRecord is the routing decision a run records (R8).
type DelegationRoutingRecord struct {
	Requested string `json:"requested"`
	Prefer    string `json:"prefer,omitempty"`
	// Implementation is the identity the run was routed against, and From
	// where it was read: the sealed bundle, the policy (a bundle sealed before
	// bundles recorded it), --lead, or none.
	Implementation     ReviewBundleImplementation   `json:"implementation"`
	ImplementationFrom string                       `json:"implementation_from"`
	Candidates         []DelegationRoutingCandidate `json:"candidates,omitempty"`
	Step               string                       `json:"step"`
	WaitUntil          string                       `json:"wait_until,omitempty"`
}

// Where a routing implementation was read.
const (
	ImplementationFromBundle = "bundle"
	ImplementationFromPolicy = "policy"
	ImplementationFromLead   = "lead"
	ImplementationFromNone   = "none"
	// ImplementationFromSession is the launching session, the implementation
	// a task is routed against (spec pose-delegation-task R10).
	ImplementationFromSession = "session"
)

// ErrNoDelegationAgentAvailable means routing found no agent it may start.
type ErrNoDelegationAgentAvailable struct{ Record DelegationRoutingRecord }

func (e *ErrNoDelegationAgentAvailable) Error() string {
	reasons := make([]string, 0, len(e.Record.Candidates))
	for _, c := range e.Record.Candidates {
		reasons = append(reasons, c.Agent+": "+c.Skipped)
	}
	return "pose: no agent is available for this run (" + strings.Join(reasons, "; ") + ")"
}

// DeclaredAgentNames lists the agents in the order the policy file declares
// them; a policy converted from schema 1 has them in name order.
func (p DelegationPolicy) DeclaredAgentNames() []string {
	if len(p.agentOrder) == len(p.Agents) {
		ok := true
		for _, name := range p.agentOrder {
			if _, found := p.Agents[name]; !found {
				ok = false
			}
		}
		if ok {
			return append([]string{}, p.agentOrder...)
		}
	}
	return p.AgentNames()
}

// RoutingImplementation is the identity a run of bundle is routed against
// (R1, R4): the bundle's, the policy's for a bundle sealed before bundles
// recorded it, the --lead given, or none.
func (s Store) RoutingImplementation(bundle ReviewBundle, policy DelegationPolicy, lead string) (ReviewBundleImplementation, string, error) {
	if impl := bundle.Payload.Implementation; impl != nil {
		return *impl, ImplementationFromBundle, nil
	}
	if p := policy.Provenance; p != nil && p.Implementation != (ModelIdentity{}) {
		id, err := normalizeModelIdentity(p.Implementation)
		if err == nil {
			out := ReviewBundleImplementation{Principal: id.Principal, Vendor: id.Vendor, Model: id.Model, Source: ImplementationDeclared}
			if id.Execution != "" {
				out.ExecutionDigest = digestBytes([]byte(id.Execution))
			}
			return out, ImplementationFromPolicy, nil
		}
	}
	if lead != "" {
		var leadModels map[string]string
		if policy.Routing != nil {
			leadModels = policy.Routing.LeadModels
		}
		id, _, err := SessionIdentity(func(string) string { return "" }, lead, leadModels)
		if err != nil {
			return ReviewBundleImplementation{}, "", err
		}
		return id, ImplementationFromLead, nil
	}
	return ReviewBundleImplementation{Vendor: ImplementationUnknown, Model: ImplementationUnknown, Source: ImplementationDetected}, ImplementationFromNone, nil
}

// DelegationAvailabilityOf reads every agent's availability (R6). probe runs
// the configured probes of agents that are otherwise available.
func (s Store) DelegationAvailabilityOf(policy DelegationPolicy, now time.Time, probe bool) map[string]DelegationAvailability {
	lastRun := map[string]ReviewRun{}
	lastLimit := map[string]string{}
	runs, lost, err := s.ReadReviewRuns("")
	historyLoss := ""
	switch {
	case err != nil:
		historyLoss = err.Error()
	case len(lost) > 0:
		historyLoss = strings.Join(lost, "; ")
	default:
		// By the instants the runs started, not by their spelling: fractions of
		// a second are written with as many digits as they need.
		sort.SliceStable(runs, func(i, j int) bool {
			a, errA := time.Parse(time.RFC3339Nano, runs[i].StartedAt)
			b, errB := time.Parse(time.RFC3339Nano, runs[j].StartedAt)
			return errA == nil && errB == nil && a.Before(b)
		})
		for _, run := range runs {
			lastRun[run.Adapter] = run
			if run.Interruption != nil && run.Interruption.Cause == "usage-limit" {
				lastLimit[run.Adapter] = run.Interruption.ResetAt
			}
		}
	}
	out := map[string]DelegationAvailability{}
	for _, name := range policy.DeclaredAgentNames() {
		agent := policy.Agents[name]
		a := DelegationAvailability{Agent: name, CLI: delegationAgentCLI(agent), LimitResetAt: lastLimit[name], State: DelegationAgentAvailable}
		_, lookErr := exec.LookPath(a.CLI)
		a.OnPath = lookErr == nil
		last, ran := lastRun[name]
		switch {
		case !a.OnPath:
			a.State, a.Detail = DelegationAgentMissing, "its CLI "+a.CLI+" is not on the path"
		case historyLoss != "":
			a.State, a.Detail = DelegationAgentUnknown, "the run history could not be read whole: "+historyLoss
		case ran && last.Interruption != nil && last.Interruption.Cause == "usage-limit":
			if at, err := time.Parse(time.RFC3339Nano, last.Interruption.ResetAt); err == nil && at.After(now) {
				a.State, a.LimitedUntil = DelegationAgentLimited, at.UTC().Format(time.RFC3339Nano)
				a.Detail = "limited until " + a.LimitedUntil
			} else if err != nil {
				// A limit with no known reset blocks the agent without a wait.
				a.State, a.Detail = DelegationAgentLimited, "its last run hit a usage limit with no known reset"
			}
		}
		if probe && a.State == DelegationAgentAvailable && agent.Probe != nil {
			if err := runDelegationProbe(*agent.Probe, policy.EnvUnset); err != nil {
				a.State, a.Detail = DelegationAgentProbeFailed, "its probe failed: "+err.Error()
			}
		}
		out[name] = a
	}
	return out
}

func delegationAgentCLI(agent DelegationAgent) string {
	if agent.CLI != "" {
		return agent.CLI
	}
	for _, access := range delegationAccessLevels {
		if argv := agent.Commands[access]; len(argv) > 0 {
			return argv[0]
		}
	}
	return ""
}

// runDelegationProbe runs a probe within its bound, without the parent
// agent's session variables, discarding its output.
func runDelegationProbe(probe DelegationProbe, envUnset []string) error {
	bound := delegationProbeDefaultBound
	if probe.Timeout != "" {
		if d, err := time.ParseDuration(probe.Timeout); err == nil && d > 0 {
			bound = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	cmd := exec.CommandContext(ctx, probe.Command[0], probe.Command[1:]...)
	cmd.Env, _ = withoutParentSession(os.Environ(), envUnset)
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("no answer within %s", bound)
	}
	return err
}

// RouteDelegation picks the agent of a run among eligible, in the policy's
// declared order (R5, R7). The record lists every candidate it considered and
// why it skipped it.
func RouteDelegation(policy DelegationPolicy, eligible []string, impl ReviewBundleImplementation, from string, avail map[string]DelegationAvailability, now time.Time) (string, DelegationRoutingRecord, error) {
	record := DelegationRoutingRecord{Requested: "auto", Prefer: policy.Routing.RoutingPrefer(), Implementation: impl, ImplementationFrom: from}
	isEligible := map[string]bool{}
	for _, name := range eligible {
		isEligible[name] = true
	}
	impl.Vendor, impl.Model = normalizeIdentityPart(impl.Vendor), normalizeIdentityPart(impl.Model)
	knownVendor := knownIdentityPart(impl.Vendor)
	groups := map[string][]string{}
	for _, name := range policy.DeclaredAgentNames() {
		if !isEligible[name] {
			continue
		}
		agent := policy.Agents[name]
		vendor, model := normalizeIdentityPart(agent.Vendor), normalizeIdentityPart(agent.Model)
		// A difference is proved only between identities both known: an agent
		// whose vendor is unknown is not the opposite vendor, and one whose
		// model is unknown is not another model (D2); it waits for the last
		// step.
		group := DelegationStepPreferred
		if record.Prefer == DelegationPreferOpposite && knownVendor {
			switch {
			case !knownIdentityPart(vendor):
				group = DelegationStepSameModel
			case vendor == impl.Vendor:
				group = DelegationStepSameModel
				if knownIdentityPart(impl.Model) && knownIdentityPart(model) && model != impl.Model {
					group = DelegationStepOtherModel
				}
			}
		}
		groups[group] = append(groups[group], name)
	}
	considered := []string{}
	consider := func(group string) string {
		for _, name := range groups[group] {
			agent, a := policy.Agents[name], avail[name]
			c := DelegationRoutingCandidate{Agent: name, Vendor: agent.Vendor, Model: agent.Model, Group: group, State: a.State}
			if a.State != DelegationAgentIneligible {
				considered = append(considered, name)
			}
			if a.State == DelegationAgentAvailable {
				record.Candidates = append(record.Candidates, c)
				return name
			}
			c.Skipped = a.Detail
			if c.Skipped == "" {
				c.Skipped = a.State
			}
			record.Candidates = append(record.Candidates, c)
		}
		return ""
	}
	if name := consider(DelegationStepPreferred); name != "" {
		record.Step = DelegationStepPreferred
		return name, record, nil
	}
	for _, step := range policy.Routing.RoutingSteps() {
		if step == DelegationStepWait {
			if name, until, ok := delegationWaitCandidate(considered, avail, now, policy.Routing.RoutingMaxWait()); ok {
				record.Step, record.WaitUntil = DelegationStepWait, until
				return name, record, nil
			}
			continue
		}
		if name := consider(step); name != "" {
			record.Step = step
			return name, record, nil
		}
	}
	return "", record, &ErrNoDelegationAgentAvailable{Record: record}
}

// delegationWaitCandidate applies the wait step only when every earlier
// candidate is limited with a known reset within maxWait, and picks the one
// that resets first.
func delegationWaitCandidate(considered []string, avail map[string]DelegationAvailability, now time.Time, maxWait time.Duration) (string, string, bool) {
	if len(considered) == 0 {
		return "", "", false
	}
	best, bestAt := "", time.Time{}
	for _, name := range considered {
		a := avail[name]
		at, err := time.Parse(time.RFC3339Nano, a.LimitedUntil)
		if a.State != DelegationAgentLimited || err != nil || at.Sub(now) > maxWait {
			return "", "", false
		}
		if best == "" || at.Before(bestAt) {
			best, bestAt = name, at
		}
	}
	return best, bestAt.UTC().Format(time.RFC3339Nano), true
}

// ExplicitDelegationRoute records an agent named with --agent (R9).
func ExplicitDelegationRoute(name string, impl ReviewBundleImplementation, from string) DelegationRoutingRecord {
	return DelegationRoutingRecord{Requested: name, Implementation: impl, ImplementationFrom: from, Step: DelegationStepExplicit}
}
