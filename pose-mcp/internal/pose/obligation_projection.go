package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ObligationQuery filters the projection. Every field is optional; an empty
// Scope reads every non-terminal spec.
type ObligationQuery struct {
	Scope        string `json:"scope,omitempty"`
	Actor        string `json:"actor,omitempty"`
	Category     string `json:"category,omitempty"`
	Phase        string `json:"phase,omitempty"`
	Satisfaction string `json:"satisfaction,omitempty"`
}

// ObligationReport is one read of everything still owed (spec
// pose-obligation-projection). It is bound to a snapshot and says, per
// producer, whether that producer was read: an empty list with incomplete
// coverage is not "nothing is owed".
type ObligationReport struct {
	SchemaVersion int                  `json:"schema_version"`
	Query         ObligationQuery      `json:"query"`
	Snapshot      ObligationSnapshot   `json:"snapshot"`
	Coverage      []ProducerCoverage   `json:"coverage"`
	Complete      bool                 `json:"complete"`
	Obligations   []Obligation         `json:"obligations"`
	Counts        map[string]int       `json:"counts"`
	Limitations   []string             `json:"limitations,omitempty"`
	Durations     map[string]float64   `json:"producer_seconds,omitempty"`
	scopeSpecs    []Spec               `json:"-"`
	producerState map[string]*Coverage `json:"-"`
}

// ObligationSnapshot says which state an answer describes. It reuses the
// digests the engine already computes; it is not a second policy.
type ObligationSnapshot struct {
	Project        string   `json:"project_id"`
	SourceRevision string   `json:"source_revision"`
	WorktreeDirty  bool     `json:"worktree_dirty"`
	PolicyDigest   string   `json:"policy_digest"`
	ContractDigest string   `json:"contract_digest"`
	IndexDigest    string   `json:"index_digest,omitempty"`
	GeneratedAt    string   `json:"generated_at"`
	Coherent       bool     `json:"coherent"`
	Limitations    []string `json:"limitations,omitempty"`
	Digest         string   `json:"digest"`
}

// ProducerCoverage is one producer's state for this read.
type ProducerCoverage struct {
	Producer string `json:"producer"`
	State    string `json:"state"`
	Detail   string `json:"detail,omitempty"`
}

// Coverage is the mutable form used while producers run.
type Coverage = ProducerCoverage

// Producer coverage states.
const (
	CoverageStateCurrent     = "current"
	CoverageStateStale       = "stale"
	CoverageStateUnavailable = "unavailable"
	CoverageStateUnsupported = "unsupported"
)

// Producers the projection integrates, and the ones it does not yet. The
// second list is part of every answer so absence is never silent.
var (
	obligationProducers        = []string{"readiness", "closeout", "review", "start", "followups", "action-requests", "assessments"}
	obligationProducersPending = map[string]string{
		"release":            "release queues are not projected yet",
		"docs-review":        "docs review pendencies are not projected yet",
		"capability-trigger": "capability stale triggers are not projected yet",
		"findings":           "open findings outside review attestations are not projected yet",
	}
)

// actionRequestObligationSource lets the ActionRequest domain register itself
// without the projection importing its storage details.
var actionRequestObligationSource func(s Store, project string, specs []Spec) ([]Obligation, error)

const ObligationReportSchemaVersion = 1

// ProjectObligations aggregates obligations read-only. It changes no file.
func (s Store) ProjectObligations(q ObligationQuery) (ObligationReport, error) {
	report := ObligationReport{SchemaVersion: ObligationReportSchemaVersion, Query: q, Counts: map[string]int{}, Durations: map[string]float64{}, producerState: map[string]*Coverage{}}
	for _, producer := range obligationProducers {
		report.producerState[producer] = &Coverage{Producer: producer, State: CoverageStateCurrent}
	}
	report.Snapshot = s.CurrentObligationSnapshot()
	project := report.Snapshot.Project

	specs, err := s.obligationScopeSpecs(q.Scope)
	if err != nil {
		return report, err
	}
	report.scopeSpecs = specs

	// Producers run side by side; each writes only its own result slot and
	// its own coverage entry, and results are joined in a fixed order.
	type produced struct {
		items    []Obligation
		err      error
		duration float64
	}
	producers := []struct {
		name string
		fn   func() ([]Obligation, error)
	}{
		{"readiness", func() ([]Obligation, error) { return s.readinessObligations(project, specs) }},
		{"closeout", func() ([]Obligation, error) { return s.closeoutObligations(project, specs, report.producerState) }},
		{"start", func() ([]Obligation, error) {
			return s.startObligationItems(project, specs, report.producerState["start"])
		}},
		{"followups", func() ([]Obligation, error) { return s.followupObligations(project, specs, q.Scope) }},
		{"assessments", func() ([]Obligation, error) { return s.assessmentObligations(project, specs), nil }},
		{"action-requests", func() ([]Obligation, error) {
			if actionRequestObligationSource == nil {
				cov := report.producerState["action-requests"]
				cov.State, cov.Detail = CoverageStateUnsupported, "action requests are not available in this engine"
				return nil, nil
			}
			return actionRequestObligationSource(s, project, specs)
		}},
	}
	results := make([]produced, len(producers))
	var wg sync.WaitGroup
	for i := range producers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			started := time.Now()
			items, err := producers[i].fn()
			results[i] = produced{items: items, err: err, duration: time.Since(started).Seconds()}
		}(i)
	}
	wg.Wait()
	var all []Obligation
	for i, producer := range producers {
		report.Durations[producer.name] = results[i].duration
		if results[i].err != nil {
			cov := report.producerState[producer.name]
			cov.State, cov.Detail = CoverageStateUnavailable, results[i].err.Error()
			continue
		}
		all = append(all, results[i].items...)
	}

	all = correlateObligations(all)
	for _, o := range all {
		if err := ValidateObligation(o); err != nil {
			report.Limitations = append(report.Limitations, "dropped an invalid projected obligation: "+err.Error())
			continue
		}
		if !q.matches(o) {
			continue
		}
		report.Obligations = append(report.Obligations, o)
		report.Counts[o.Category]++
	}
	sort.Slice(report.Obligations, func(i, j int) bool { return report.Obligations[i].ID < report.Obligations[j].ID })
	if report.Obligations == nil {
		report.Obligations = []Obligation{}
	}

	report.Complete = report.Snapshot.Coherent
	for _, producer := range obligationProducers {
		cov := *report.producerState[producer]
		report.Coverage = append(report.Coverage, cov)
		if cov.State != CoverageStateCurrent {
			report.Complete = false
		}
	}
	pending := make([]string, 0, len(obligationProducersPending))
	for producer := range obligationProducersPending {
		pending = append(pending, producer)
	}
	sort.Strings(pending)
	for _, producer := range pending {
		report.Coverage = append(report.Coverage, ProducerCoverage{Producer: producer, State: CoverageStateUnsupported, Detail: obligationProducersPending[producer]})
	}
	report.Limitations = append(report.Limitations, report.Snapshot.Limitations...)
	return report, nil
}

func (q ObligationQuery) matches(o Obligation) bool {
	if q.Category != "" && o.Category != q.Category {
		return false
	}
	if q.Satisfaction != "" && o.Satisfaction != q.Satisfaction {
		return false
	}
	if q.Phase != "" {
		found := false
		for _, effect := range o.Effects {
			found = found || effect.Phase == q.Phase
		}
		if !found {
			return false
		}
	}
	if q.Actor != "" && o.Recipient.Principal != q.Actor && o.Recipient.Role != q.Actor && !(q.Actor == "unassigned" && o.Recipient.Unassigned) {
		return false
	}
	return true
}

// obligationScopeSpecs resolves the specs a scope covers. A milestone or
// roadmap expands to its local members.
func (s Store) obligationScopeSpecs(scope string) ([]Spec, error) {
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	if scope == "" {
		out := []Spec{}
		for _, sp := range specs {
			if !terminalStatuses[sp.Status] {
				out = append(out, sp)
			}
		}
		return out, nil
	}
	ref, err := ParseScopeRef(scope)
	if err != nil {
		return nil, err
	}
	members := map[string]bool{}
	switch ref.Kind {
	case "spec":
		members[ref.Slug] = true
	case "milestone", "roadmap":
		slug := ref.Slug
		if ref.Kind == "milestone" {
			slug = ref.Roadmap
		}
		rm, err := s.GetRoadmap(slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			if ref.Kind == "milestone" && milestone.ID != ref.Milestone {
				continue
			}
			for _, member := range milestone.Specs {
				if local, external := s.milestoneMember(member); !external {
					members[local] = true
				}
			}
		}
	}
	out := []Spec{}
	for _, sp := range specs {
		if members[sp.Slug] {
			out = append(out, sp)
		}
	}
	if ref.Kind == "spec" && len(out) == 0 {
		return nil, fmt.Errorf("pose: spec %q not found", ref.Slug)
	}
	return out, nil
}

func specNode(project, slug string) NodeRef {
	return QualifyNodeRef(project, ArtifactRef{Kind: "spec", Slug: slug}, "", "")
}

func newObligation(project, producer string, source NodeRef, rule, discriminator string) Obligation {
	return Obligation{
		SchemaVersion: ObligationSchemaVersion,
		ID:            ObligationID(project, producer, source, rule, discriminator),
		Project:       project,
		Source:        ObligationSource{Producer: producer, Ref: source, Detail: discriminator},
		Rule:          rule,
		Recipient:     ObligationActor{Unassigned: true},
		Targets:       []NodeRef{source},
		Satisfaction:  SatisfactionPending,
		Knowledge:     KnowledgeKnown,
		Observation:   ObligationObservation{Freshness: FreshnessCurrent, Coverage: CoverageComplete},
	}
}

// readinessObligations projects unmet prerequisites. They restrict start and
// execution of the whole spec: depends_on is a spec-level precondition.
func (s Store) readinessObligations(project string, specs []Spec) ([]Obligation, error) {
	resolver, _, err := EnvironmentArtifactResolver(s.Root, "")
	if err != nil {
		return nil, err
	}
	return forEachSpec(specs, func(sp Spec) ([]Obligation, error) {
		if terminalStatuses[sp.Status] {
			return nil, nil
		}
		var out []Obligation
		readiness, err := s.SpecReadinessWithResolver(sp.Slug, project, resolver)
		if err != nil {
			return nil, err
		}
		source := specNode(project, sp.Slug)
		for _, waiting := range readiness.WaitingOn {
			o := newObligation(project, "readiness", source, "depends_on", waiting.Ref)
			o.Category = ObligationDependency
			o.ReasonCode = firstNonempty(waiting.Code, "dependency-unresolved")
			o.Condition = DiagnosticCatalog[o.ReasonCode].Condition
			o.Effects = []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}, {Phase: PhaseExecution, Mode: EffectBlock}}
			o.Waiting = WaitingArtifact
			if o.ReasonCode == "dor-acceptance-criteria-missing" {
				o.Rule = "definition-of-ready"
				o.Waiting = WaitingExecution
				o.Effects = []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}}
			}
			if o.ReasonCode == "dependency-unresolved" && waiting.Detail != "" {
				// The engine could not observe the reference at all.
				o.Knowledge = KnowledgeUnknown
				o.Waiting = WaitingUnknownDep
			}
			o.Message = sp.Slug + " waits on " + waiting.Ref + ": " + waiting.Reason
			out = append(out, o)
		}
		if readiness.Cause == ReadinessCauseUnknown {
			o := newObligation(project, "readiness", source, "lifecycle", "blocked")
			o.Category = ObligationReconciliation
			o.ReasonCode = "blocked-cause-unknown"
			o.Condition = DiagnosticCatalog[o.ReasonCode].Condition
			o.Effects = []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}, {Phase: PhaseExecution, Mode: EffectBlock}}
			o.Waiting = WaitingUnknownDep
			o.Message = sp.Slug + " is blocked and no recorded cause explains it"
			out = append(out, o)
		}
		return out, nil
	})
}

// forEachSpec runs fn per spec on a bounded pool and returns the results in
// spec order, so the projection stays deterministic while producers that
// spawn Git or parse the project in turn run side by side.
func forEachSpec(specs []Spec, fn func(Spec) ([]Obligation, error)) ([]Obligation, error) {
	results := make([][]Obligation, len(specs))
	errs := make([]error, len(specs))
	workers := runtime.GOMAXPROCS(0)
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i], errs[i] = fn(specs[i])
			}
		}()
	}
	for i := range specs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	var out []Obligation
	for i := range specs {
		if errs[i] != nil {
			return nil, errs[i]
		}
		out = append(out, results[i]...)
	}
	return out, nil
}

// closeoutObligations projects what stands between an in-progress spec and a
// terminal closeout, and the judgment its current bundle still owes. Draft
// specs owe nothing at closeout yet.
func (s Store) closeoutObligations(project string, specs []Spec, coverage map[string]*Coverage) ([]Obligation, error) {
	review := coverage["review"]
	var reviewMu sync.Mutex
	return forEachSpec(specs, func(sp Spec) ([]Obligation, error) {
		if sp.Status != "in-progress" {
			return nil, nil
		}
		var out []Obligation
		ref := "spec:" + sp.Slug
		state, err := s.GetCloseoutState(ref)
		if err != nil {
			return nil, err
		}
		source := specNode(project, sp.Slug)
		for _, d := range state.Diagnostics {
			var o Obligation
			switch d.Code {
			case "lifecycle-not-done":
				// The closeout transition itself, not an obligation on it.
				continue
			case "review-not-approved":
				o = newObligation(project, "closeout", source, "review-policy", d.Code)
				o.Category = ObligationJudgment
				o.Recipient = ObligationActor{Role: "reviewer"}
				o.Waiting = WaitingActor
			case "child-scope-open", "external-member-unaccepted":
				o = newObligation(project, "closeout", source, "closeout", d.Code+":"+strings.Join(d.Refs, ","))
				o.Category = ObligationDependency
				o.Waiting = WaitingArtifact
			case "review-blocker", "federated-acceptance-blocker":
				// An opaque text blocker has no identity but its text.
				o = newObligation(project, "closeout", source, "closeout", d.Code+":"+d.Message)
				o.Category = ObligationReconciliation
				o.Targets = nil
				o.Observation.Coverage = CoverageLegacyOpaque
				o.Observation.Limitations = []string{"reported as text by the producer; actor, target and unblock condition are not known"}
				o.Waiting = WaitingUnknownDep
			default:
				continue
			}
			o.ReasonCode = d.Code
			o.Condition = firstNonempty(d.Condition, DiagnosticCatalog[d.Code].Condition)
			o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
			o.Message = sp.Slug + ": " + d.Message
			out = append(out, o)
		}
		// Judgment the current sealed bundle still owes: one obligation per
		// criterion, so a reviewer sees which conclusion is missing.
		if state.Review.BundleID != "" && state.Review.AttestationID == "" {
			prepared, prepErr := s.PrepareReviewAttestation(state.Review.BundleID, "agent:obligation-projection", time.Now())
			if prepErr != nil {
				reviewMu.Lock()
				review.State, review.Detail = CoverageStateUnavailable, "pending judgment of "+state.Review.BundleID+" could not be read: "+prepErr.Error()
				reviewMu.Unlock()
				return out, nil
			}
			for _, p := range prepared.Pending {
				node := QualifyNodeRef(project, ArtifactRef{Kind: "spec", Slug: sp.Slug}, "criterion", p.Criterion)
				o := newObligation(project, "review", node, "review-plan", state.Review.BundleID)
				o.Category = ObligationJudgment
				o.ReasonCode = firstNonempty(p.Code, "judgment-unanswered")
				o.Condition = firstNonempty(p.Condition, DiagnosticCatalog["judgment-unanswered"].Condition)
				o.Recipient = ObligationActor{Role: "reviewer"}
				o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
				o.Waiting = WaitingActor
				o.Message = sp.Slug + ": criterion " + p.Criterion + " needs a reviewer's conclusion (" + p.Reason + ")"
				out = append(out, o)
			}
		}
		return out, nil
	})
}

// startObligationItems projects start reconciliation needs, which exist only
// when atomic start is adopted.
func (s Store) startObligationItems(project string, specs []Spec, coverage *Coverage) ([]Obligation, error) {
	adopted, err := s.AtomicStartAdopted()
	if err != nil {
		return nil, err
	}
	if !adopted {
		coverage.State, coverage.Detail = CoverageStateUnsupported, "atomic start is not adopted, so no start record is reconciled"
		return nil, nil
	}
	var out []Obligation
	for _, sp := range specs {
		if sp.Status == "draft" {
			continue
		}
		status, err := s.GetStartStatus(sp.Slug)
		if err != nil {
			return nil, err
		}
		for _, need := range status.Reconciliation {
			o := newObligation(project, "start", specNode(project, sp.Slug), "atomic-start", need)
			o.Category = ObligationReconciliation
			o.ReasonCode = "start-reconciliation"
			o.Condition = "the start record and the spec agree again, or the start is cancelled explicitly"
			o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
			o.Waiting = WaitingExecution
			o.Message = sp.Slug + ": " + need
			out = append(out, o)
		}
	}
	return out, nil
}

// followupObligations projects open follow-ups as residual debt: advisory,
// never a delivery gate by default. Done specs keep their debt visible.
func (s Store) followupObligations(project string, specs []Spec, scope string) ([]Obligation, error) {
	inScope := map[string]bool{}
	for _, sp := range specs {
		inScope[sp.Slug] = true
	}
	var out []Obligation
	for _, item := range ParseSpecFollowups(s.Root) {
		if item.RawDisposition != "open" {
			continue
		}
		if scope != "" && !inScope[item.Spec] {
			continue
		}
		node := QualifyNodeRef(project, ArtifactRef{Kind: "spec", Slug: item.Spec}, "followup", fmt.Sprintf("f%d", item.Ordinal))
		o := newObligation(project, "followups", node, "followup-ownership", normalizeObligationText(item.Text))
		o.Category = ObligationResidualDebt
		o.ReasonCode = "followup-open"
		o.Condition = "the follow-up receives a disposition in its spec"
		o.Effects = []ObligationEffect{{Phase: PhaseExecution, Mode: EffectAdvisory}}
		o.Waiting = WaitingNone
		o.Observation.Limitations = []string{"follow-ups carry no stable identifier; this identity is the normalized text, so rewording the bullet yields a new id"}
		if item.Owner != "" && item.Owner != "unowned" {
			o.Recipient = ObligationActor{Principal: item.Owner}
		}
		o.Message = item.Spec + ": " + item.Text
		out = append(out, o)
	}
	return out, nil
}

// normalizeObligationText makes a follow-up's identity survive rewrapping.
func normalizeObligationText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// correlateObligations merges obligations with the same logical id, keeping
// every distinct effect and naming the other producers.
func correlateObligations(items []Obligation) []Obligation {
	index := map[string]int{}
	var out []Obligation
	for _, o := range items {
		if at, seen := index[o.ID]; seen {
			merged := &out[at]
			if merged.Source.Producer != o.Source.Producer {
				merged.CorrelatedWith = uniqueSorted(append(merged.CorrelatedWith, o.Source.Producer))
			}
			for _, effect := range o.Effects {
				present := false
				for _, existing := range merged.Effects {
					present = present || (existing.Phase == effect.Phase && existing.Mode == effect.Mode)
				}
				if !present {
					merged.Effects = append(merged.Effects, effect)
				}
			}
			continue
		}
		index[o.ID] = len(out)
		out = append(out, o)
	}
	return out
}

// CurrentObligationSnapshot binds an answer to the repository state it read.
func (s Store) CurrentObligationSnapshot() ObligationSnapshot {
	snap := ObligationSnapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Coherent: true}
	_, project, err := EnvironmentArtifactResolver(s.Root, "")
	if err != nil || project == "" {
		project = DefaultProjectID(s.Root)
		snap.Limitations = append(snap.Limitations, "project identity fell back to the directory name")
	}
	snap.Project = project
	if head, err := exec.Command("git", "-C", s.Root, "rev-parse", "--verify", "HEAD").Output(); err == nil {
		snap.SourceRevision = strings.TrimSpace(string(head))
	} else {
		snap.SourceRevision = "unknown"
		snap.Coherent = false
		snap.Limitations = append(snap.Limitations, "no Git revision: the answer describes the working tree only")
	}
	if status, err := exec.Command("git", "-C", s.Root, "status", "--porcelain", "--untracked-files=no", "--", ".pose").Output(); err == nil && len(strings.TrimSpace(string(status))) > 0 {
		snap.WorktreeDirty = true
		snap.Limitations = append(snap.Limitations, "governed files under .pose differ from "+shortRevision(snap.SourceRevision)+"; the answer describes the working tree, not the commit")
	}
	snap.PolicyDigest = directoryDigest(filepath.Join(s.Root, ".pose", "policy"))
	snap.ContractDigest = projectContractDigest(s.Root)
	if raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "delivery-integrity.json")); err == nil {
		sum := sha256.Sum256(raw)
		snap.IndexDigest = "sha256:" + hex.EncodeToString(sum[:])
	}
	snap.Digest = snapshotDigest(snap)
	return snap
}

func snapshotDigest(snap ObligationSnapshot) string {
	digest, _ := digestJSON([]string{snap.Project, snap.SourceRevision, fmt.Sprint(snap.WorktreeDirty), snap.PolicyDigest, snap.ContractDigest, snap.IndexDigest})
	return digest
}

// ObligationSnapshotChanges reports which bound inputs changed since snap was
// taken. A write gate revalidates with this instead of trusting an earlier
// read; an empty result means the answer still describes the repository
// (the working tree may still differ inside a dirty snapshot, which snap
// already said).
func (s Store) ObligationSnapshotChanges(snap ObligationSnapshot) []string {
	now := s.CurrentObligationSnapshot()
	var changed []string
	if now.Project != snap.Project {
		changed = append(changed, "project")
	}
	if now.SourceRevision != snap.SourceRevision {
		changed = append(changed, "source-revision")
	}
	if now.PolicyDigest != snap.PolicyDigest {
		changed = append(changed, "policy")
	}
	if now.ContractDigest != snap.ContractDigest {
		changed = append(changed, "authority-contract")
	}
	if now.IndexDigest != snap.IndexDigest {
		changed = append(changed, "delivery-index")
	}
	return changed
}

func shortRevision(revision string) string {
	if len(revision) > 12 {
		return revision[:12]
	}
	return revision
}

// directoryDigest hashes the regular files of one directory, name and
// content, in name order.
func directoryDigest(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "unavailable"
	}
	h := sha256.New()
	names := []string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(raw)
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
