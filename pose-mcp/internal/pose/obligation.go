package pose

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Obligation is one thing still owed, projected from the subsystem that owns
// it (spec pose-obligation-contract, ADR
// obligations-are-projected-action-requests-are-persisted).
//
// It is a read model. The aggregator never stores a "resolved" state of its
// own: an obligation disappears or turns satisfied because its source
// changed. Four questions stay on separate fields because they have
// different answers — is the condition met (Satisfaction), does the engine
// have current data to say (Knowledge), does it depend on something outside
// the running execution (Waiting), and which phase of which scope it
// restricts (Effects).
type Obligation struct {
	SchemaVersion int `json:"schema_version"`
	// ID depends on the logical identity of the source, never on message
	// text, list order or query time.
	ID           string                `json:"id"`
	Project      string                `json:"project_id"`
	Source       ObligationSource      `json:"source"`
	Category     string                `json:"category"`
	ReasonCode   string                `json:"reason_code"`
	Condition    string                `json:"condition"`
	Rule         string                `json:"rule,omitempty"`
	Recipient    ObligationActor       `json:"recipient"`
	Targets      []NodeRef             `json:"targets"`
	Effects      []ObligationEffect    `json:"effects"`
	Satisfaction string                `json:"satisfaction"`
	Knowledge    string                `json:"knowledge"`
	Waiting      string                `json:"waiting"`
	Observation  ObligationObservation `json:"observation"`
	// Message is a rendering for people. Nothing may parse it.
	Message string `json:"message"`
	// CorrelatedWith lists other producers that reported the same logical
	// obligation; their restrictions are merged, never dropped.
	CorrelatedWith []string `json:"correlated_with,omitempty"`
}

// ObligationSource names the producer and the authoritative record. Detail is
// a stable pointer back into that producer's own output (a criterion id, a
// waiting ref) so a consumer can reach the source-specific data the common
// shape does not carry.
type ObligationSource struct {
	Producer string  `json:"producer"`
	Ref      NodeRef `json:"ref"`
	Detail   string  `json:"detail,omitempty"`
}

// ObligationActor is who can satisfy the obligation: a known principal, a
// role, or nobody yet. An unassigned obligation stays visible as such.
type ObligationActor struct {
	Principal  string `json:"principal,omitempty"`
	Role       string `json:"role,omitempty"`
	Unassigned bool   `json:"unassigned,omitempty"`
}

// ObligationEffect restricts one phase over a scope. An empty Scope means the
// whole source artifact; the effect never widens beyond what the producer
// stated.
type ObligationEffect struct {
	Phase string    `json:"phase"`
	Scope []NodeRef `json:"scope,omitempty"`
	Mode  string    `json:"mode"`
}

// ObligationObservation records what the answer was computed from.
type ObligationObservation struct {
	SourceRevision string   `json:"source_revision,omitempty"`
	Freshness      string   `json:"freshness"`
	Coverage       string   `json:"coverage"`
	Limitations    []string `json:"limitations,omitempty"`
}

const ObligationSchemaVersion = 1

// Categories.
const (
	ObligationDependency     = "dependency"
	ObligationJudgment       = "judgment"
	ObligationEvidence       = "evidence"
	ObligationReconciliation = "reconciliation"
	ObligationRemediation    = "remediation"
	ObligationRelease        = "release"
	ObligationActorAction    = "actor-action"
	ObligationResidualDebt   = "residual-debt"
)

// Phases.
const (
	PhaseStart     = "start"
	PhaseExecution = "execution"
	PhaseReview    = "review"
	PhaseCloseout  = "closeout"
	PhaseRelease   = "release"
)

// Effect modes. Advisory is shown, never enforced.
const (
	EffectBlock    = "block"
	EffectAdvisory = "advisory"
)

// Satisfaction, as the producing domain defines it.
const (
	SatisfactionPending     = "pending"
	SatisfactionSatisfied   = "satisfied"
	SatisfactionWaived      = "waived"
	SatisfactionCancelled   = "cancelled"
	SatisfactionInvalidated = "invalidated"
)

// Knowledge: whether the engine had current data to answer.
const (
	KnowledgeKnown   = "known"
	KnowledgeUnknown = "unknown"
)

// Waiting: what the condition depends on.
const (
	WaitingNone       = "none"
	WaitingArtifact   = "artifact"
	WaitingActor      = "actor"
	WaitingExternal   = "external"
	WaitingExecution  = "execution"
	WaitingUnknownDep = "unknown"
)

// Freshness and coverage of an observation.
const (
	FreshnessCurrent = "current"
	FreshnessStale   = "stale"
	FreshnessUnknown = "unknown"

	CoverageComplete     = "complete"
	CoveragePartial      = "partial"
	CoverageLegacyOpaque = "legacy-opaque"
)

var (
	obligationCategories   = enumSet(ObligationDependency, ObligationJudgment, ObligationEvidence, ObligationReconciliation, ObligationRemediation, ObligationRelease, ObligationActorAction, ObligationResidualDebt)
	obligationPhases       = enumSet(PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease)
	obligationModes        = enumSet(EffectBlock, EffectAdvisory)
	obligationSatisfaction = enumSet(SatisfactionPending, SatisfactionSatisfied, SatisfactionWaived, SatisfactionCancelled, SatisfactionInvalidated)
	obligationKnowledge    = enumSet(KnowledgeKnown, KnowledgeUnknown)
	obligationWaiting      = enumSet(WaitingNone, WaitingArtifact, WaitingActor, WaitingExternal, WaitingExecution, WaitingUnknownDep)
	obligationFreshness    = enumSet(FreshnessCurrent, FreshnessStale, FreshnessUnknown)
	obligationCoverage     = enumSet(CoverageComplete, CoveragePartial, CoverageLegacyOpaque)
)

func enumSet(values ...string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[v] = true
	}
	return out
}

// ObligationEnums exposes the closed vocabularies, so the published schema
// can be checked against the engine instead of restating them.
func ObligationEnums() map[string][]string {
	list := func(m map[string]bool) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return map[string][]string{
		"category": list(obligationCategories), "phase": list(obligationPhases), "mode": list(obligationModes),
		"satisfaction": list(obligationSatisfaction), "knowledge": list(obligationKnowledge), "waiting": list(obligationWaiting),
		"freshness": list(obligationFreshness), "coverage": list(obligationCoverage),
	}
}

// NodeRef qualifies a node by its artifact and project: `R4` alone is local
// to one spec and never a global identity. The string form is
// `xref:<project>/<kind>:<slug>[/<milestone>]#<node-kind>:<id>`; the node part
// is optional when the reference is the artifact itself.
type NodeRef struct {
	Artifact string `json:"artifact"`
	Kind     string `json:"kind,omitempty"`
	ID       string `json:"id,omitempty"`
}

// Node kinds a reference may name inside an artifact.
var nodeKinds = enumSet("requirement", "assumption", "decision", "criterion", "surface", "release", "followup", "finding", "action", "doc", "capability")

func (n NodeRef) String() string {
	if n.Kind == "" {
		return n.Artifact
	}
	return n.Artifact + "#" + n.Kind + ":" + n.ID
}

// QualifyNodeRef binds a local reference to its project. Only an artifact
// already qualified, or one qualified here, may become an obligation target.
func QualifyNodeRef(project string, artifact ArtifactRef, kind, id string) NodeRef {
	if artifact.Project == "" {
		artifact.Project = project
	}
	return NodeRef{Artifact: artifact.String(), Kind: kind, ID: id}
}

// ParseNodeRef parses the string form. The artifact part must be qualified
// with `xref:`: an unqualified `spec:x#requirement:R4` would collide across
// projects, which is what this type exists to prevent.
func ParseNodeRef(raw string) (NodeRef, error) {
	bad := errors.New("invalid-node-reference")
	artifactPart, nodePart, hasNode := strings.Cut(raw, "#")
	ref, err := ParseArtifactRef(artifactPart)
	if err != nil || ref.Project == "" {
		return NodeRef{}, bad
	}
	out := NodeRef{Artifact: ref.String()}
	if !hasNode {
		return out, nil
	}
	kind, id, ok := strings.Cut(nodePart, ":")
	if !ok || !nodeKinds[kind] || ValidateSlug(strings.ToLower(id)) != nil || len(id) > 128 {
		return NodeRef{}, bad
	}
	out.Kind, out.ID = kind, id
	return out, nil
}

// ObligationID derives the stable identity from the logical source: project,
// producer, source node, the rule that requires it and a producer-chosen
// discriminator (a criterion id, a dependency ref). Wording, order and time
// never enter it.
func ObligationID(project, producer string, source NodeRef, rule, discriminator string) string {
	digest, _ := digestJSON([]string{"obligation/v1", project, producer, source.String(), rule, discriminator})
	return "obl-" + strings.TrimPrefix(digest, "sha256:")[:16]
}

// ValidateObligation checks the record's own consistency. It cannot check
// that the producer told the truth; it checks that nothing is left implicit.
func ValidateObligation(o Obligation) error {
	fail := func(format string, a ...any) error {
		return fmt.Errorf("pose: obligation %s: "+format, append([]any{o.ID}, a...)...)
	}
	if o.SchemaVersion != ObligationSchemaVersion {
		return fail("unsupported schema version %d", o.SchemaVersion)
	}
	if !strings.HasPrefix(o.ID, "obl-") || len(o.ID) != 20 {
		return fail("id must be obl-<16 hex>")
	}
	if ValidateSlug(o.Project) != nil {
		return fail("project_id is required")
	}
	if o.Source.Producer == "" {
		return fail("source producer is required")
	}
	if _, err := ParseNodeRef(o.Source.Ref.String()); err != nil {
		return fail("source ref must be a qualified node reference")
	}
	if !obligationCategories[o.Category] {
		return fail("unknown category %q", o.Category)
	}
	if o.ReasonCode == "" || o.Condition == "" {
		return fail("reason_code and condition are required")
	}
	if o.Recipient.Unassigned && (o.Recipient.Principal != "" || o.Recipient.Role != "") {
		return fail("an unassigned recipient names nobody")
	}
	for _, target := range o.Targets {
		if _, err := ParseNodeRef(target.String()); err != nil {
			return fail("target %q is not a qualified node reference", target.String())
		}
	}
	for _, effect := range o.Effects {
		if !obligationPhases[effect.Phase] || !obligationModes[effect.Mode] {
			return fail("effect %s/%s is not a known phase and mode", effect.Phase, effect.Mode)
		}
		for _, scope := range effect.Scope {
			if _, err := ParseNodeRef(scope.String()); err != nil {
				return fail("effect scope %q is not a qualified node reference", scope.String())
			}
		}
	}
	if !obligationSatisfaction[o.Satisfaction] || !obligationKnowledge[o.Knowledge] || !obligationWaiting[o.Waiting] {
		return fail("satisfaction, knowledge and waiting must each be a known value")
	}
	// Unknown is never a resolution. An obligation the engine could not
	// observe stays pending; calling it satisfied, waived or cancelled would
	// turn missing data into a clear answer.
	if o.Knowledge == KnowledgeUnknown && o.Satisfaction != SatisfactionPending {
		return fail("unknown knowledge cannot carry satisfaction %q", o.Satisfaction)
	}
	if !obligationFreshness[o.Observation.Freshness] || !obligationCoverage[o.Observation.Coverage] {
		return fail("observation freshness and coverage must each be a known value")
	}
	if o.Observation.Coverage == CoverageLegacyOpaque && (o.Recipient.Principal != "" || o.Recipient.Role != "" || len(o.Targets) > 0) {
		// A legacy blocker known only as text keeps its origin and its
		// limitation; it does not gain an invented actor or target.
		return fail("a legacy-opaque obligation cannot name an actor or targets")
	}
	return nil
}

// Restricts reports whether the obligation, while unsatisfied, restricts the
// phase. Advisory effects never restrict.
//
// Only a pending or invalidated condition restricts. Satisfied and waived are
// resolutions the producing domain recorded; cancelled means the producer no
// longer requires it (cancelling an ActionRequest does not cancel the
// obligation it was asked to satisfy, which stays pending at its own source).
func (o Obligation) Restricts(phase string) bool {
	if o.Satisfaction != SatisfactionPending && o.Satisfaction != SatisfactionInvalidated {
		return false
	}
	for _, effect := range o.Effects {
		if effect.Phase == phase && effect.Mode == EffectBlock {
			return true
		}
	}
	return false
}
