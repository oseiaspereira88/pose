package pose

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SpecFacts is the factual part of a spec's Final Report, derived from the
// delivery index instead of transcribed (spec pose-progressive-spec-surface):
// the paths the attributed commits changed and the checks that produced
// evidence. It never contains intent, rationale or accepted risk — those
// stay written by a person or an agent.
//
// Checks are the results the index ties to this spec (spec
// pose-spec-facts-evidence-binding): their scope provenance names it. Candidates are
// results the index holds that nothing ties to it; they are listed so the reader can see
// them, and are not evidence that the spec was validated.
type SpecFacts struct {
	Spec        string             `json:"spec"`
	Commits     int                `json:"commits"`
	Paths       []ObservedPath     `json:"paths"`
	Checks      []SpecFactCheck    `json:"checks"`
	Candidates  []SpecFactCheck    `json:"candidates"`
	Freshness   SpecFactsFreshness `json:"freshness"`
	Source      string             `json:"source"`
	Limitations []string           `json:"limitations,omitempty"`
}

// Classification of a result for a spec.
const (
	SpecFactAssociated = "associated"
	SpecFactCandidate  = "candidate"
)

// SpecFactCheck is one validation result the spec's evidence came from, or a candidate.
type SpecFactCheck struct {
	ID             string          `json:"id,omitempty"`
	Module         string          `json:"module"`
	Check          string          `json:"check"`
	EvidenceClass  string          `json:"evidence_class"`
	Outcome        string          `json:"outcome"`
	Classification string          `json:"classification"`
	Basis          string          `json:"basis"`
	Binding        SpecFactBinding `json:"binding"`
	Limits         []string        `json:"limits,omitempty"`
}

// SpecFactBinding says what a result is bound to: the revision it ran against, the
// provenance it carries, and how that revision compares with the head of the spec's change
// sets. A result that ran against another head is reported as such, not dropped: it may be
// reusable, and whether it is is a judgment the facts do not make.
type SpecFactBinding struct {
	GitHead          string `json:"git_head,omitempty"`
	ProvenanceDigest string `json:"provenance_digest,omitempty"`
	GeneratedAt      string `json:"generated_at,omitempty"`
	Head             string `json:"head"`
	// ScopedDigest is the scope provenance the result carries for this spec, and
	// ExpectedScopedDigest the one the spec's claims and change sets give today: the
	// association rests on them being the same.
	ScopedDigest         string `json:"scoped_digest,omitempty"`
	ExpectedScopedDigest string `json:"expected_scoped_digest,omitempty"`
}

// Freshness of the index the facts were read from.
const (
	SpecFactsFresh       = "fresh"
	SpecFactsStale       = "stale"
	SpecFactsUnknown     = "unknown"
	SpecFactsUnavailable = "unavailable"
)

// SpecFactsFreshness says whether the index can be trusted to describe the inputs now: a
// stale, unpublished or unreadable index is not a current, complete one.
type SpecFactsFreshness struct {
	Status      string   `json:"status"`
	Reason      string   `json:"reason,omitempty"`
	Generation  string   `json:"generation,omitempty"`
	StaleInputs []string `json:"stale_inputs,omitempty"`
	// Unbound lists the inputs the generation does not bind, which fresh cannot be said about.
	Unbound []string `json:"unbound,omitempty"`
	// Diverged lists the index files that differ from the generation they belong to.
	Diverged []string `json:"diverged,omitempty"`
}

// specFactsFreshness judges the generation, and only what the generation binds. Fresh is
// said when every input a generation is made of is bound and unchanged, and the index
// files are the ones the generation published. The axes a generation cannot bind (the
// validation results and the Git history) are checked by the caller.
func (s Store) specFactsFreshness() SpecFactsFreshness {
	view := s.ReadIndexGeneration()
	switch view.Status {
	case IndexGenerationUnavailable:
		return SpecFactsFreshness{Status: SpecFactsUnavailable, Reason: view.Reason}
	case IndexGenerationCurrent:
		generation := view.Descriptor.GenerationID
		stale := view.StaleInputs(func(name string) (string, bool) { return IndexInputBinding(s.Root, name) })
		unbound := []string{}
		for _, name := range IndexInputNames {
			if _, ok := view.Descriptor.Inputs[name]; !ok {
				unbound = append(unbound, name)
			}
		}
		diverged := append([]string{}, view.Diverged...)
		switch {
		case len(stale) > 0:
			return SpecFactsFreshness{Status: SpecFactsStale, Generation: generation, StaleInputs: stale, Diverged: diverged,
				Reason: "an input of the index changed since it was published: " + strings.Join(stale, ", ")}
		case len(diverged) > 0:
			return SpecFactsFreshness{Status: SpecFactsStale, Generation: generation, Diverged: diverged,
				Reason: "index files differ from the generation that was published: " + strings.Join(diverged, ", ")}
		case len(unbound) > 0:
			return SpecFactsFreshness{Status: SpecFactsUnknown, Generation: generation, Unbound: unbound,
				Reason: "the generation does not bind: " + strings.Join(unbound, ", ")}
		}
		return SpecFactsFreshness{Status: SpecFactsFresh, Generation: generation}
	}
	return SpecFactsFreshness{Status: SpecFactsUnknown, Reason: view.Reason}
}

// degradeFreshness lowers a fresh judgment because of an axis the generation does not
// bind. A stale axis also lowers an unknown one; nothing raises a judgment.
func degradeFreshness(f *SpecFactsFreshness, status, input, reason string) {
	if status == SpecFactsStale {
		f.StaleInputs = append(f.StaleInputs, input)
	}
	switch {
	case f.Status == SpecFactsFresh:
		f.Status, f.Reason = status, reason
	case f.Status == SpecFactsUnknown && status == SpecFactsStale:
		f.Status, f.Reason = status, reason
	}
}

// resultsFileDisagrees reports why the validation results file the index was built from
// no longer matches the results the index holds, or "" when it matches or there is none.
func (s Store) resultsFileDisagrees(graph DeliveryIntegrityGraph) string {
	policy, err := LoadDeliveryPolicy(s.Root)
	if err != nil || policy.ResultsPath == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(policy.ResultsPath)))
	if errors.Is(err, os.ErrNotExist) {
		if len(graph.ValidationResults) > 0 {
			return policy.ResultsPath + " is gone, and the index holds results read from it"
		}
		return ""
	}
	if err != nil {
		return policy.ResultsPath + " could not be read: " + err.Error()
	}
	var run struct {
		GeneratedAt      string `json:"generated_at"`
		GitHead          string `json:"git_head"`
		ProvenanceDigest string `json:"provenance_digest"`
		Checks           []struct {
			ID            string `json:"id"`
			Outcome       string `json:"outcome"`
			EvidenceClass string `json:"evidence_class"`
		} `json:"checks"`
	}
	if json.Unmarshal(raw, &run) != nil {
		return policy.ResultsPath + " is not a readable validation result"
	}
	now := map[string]string{}
	for _, c := range run.Checks {
		if c.EvidenceClass != "" {
			now[c.ID] = c.Outcome
		}
	}
	held := map[string]DeliveryValidationResult{}
	for _, r := range graph.ValidationResults {
		held[r.ID] = r
		if r.Report != policy.ResultsPath && r.Report != "" {
			continue
		}
		if outcome, ok := now[r.ID]; !ok || outcome != r.Outcome || run.GeneratedAt != r.GeneratedAt || run.ProvenanceDigest != r.ProvenanceDigest {
			return policy.ResultsPath + " was written again after the index read it"
		}
	}
	for id := range now {
		if _, ok := held[id]; !ok && len(graph.ValidationResults) > 0 {
			return policy.ResultsPath + " holds results the index does not"
		}
	}
	return ""
}

// unindexedTrailerCommits counts the commits that carry the spec's trailer and are not in
// any change set of the index, or -1 when the history could not be read.
func (s Store) unindexedTrailerCommits(slug string, indexed map[string]bool) int {
	cmd := exec.Command("git", "-C", s.Root, "log", "--format=%H", "-E", "--grep=^POSE-Spec: "+regexp.QuoteMeta(slug)+"$")
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	missing := 0
	for _, h := range strings.Fields(string(out)) {
		if !indexed[h] {
			missing++
		}
	}
	return missing
}

// DeriveSpecFacts reads the delivery index. A spec with no attributed change
// set says so instead of reporting an empty delivery.
func (s Store) DeriveSpecFacts(slug string) (SpecFacts, error) {
	spec, err := s.GetSpec(slug)
	if err != nil {
		return SpecFacts{}, err
	}
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		return SpecFacts{}, err
	}
	facts := SpecFacts{Spec: slug, Source: ".pose/indexes/delivery-integrity.json", Paths: []ObservedPath{}, Checks: []SpecFactCheck{}, Candidates: []SpecFactCheck{}}
	facts.Freshness = s.specFactsFreshness()
	seen := map[string]bool{}
	commits := map[string]bool{}
	heads := map[string]bool{}
	for _, cs := range graph.ChangeSets {
		if cs.Spec != slug {
			continue
		}
		if cs.ResolvedHead != "" {
			heads[cs.ResolvedHead] = true
		}
		for _, c := range cs.Commits {
			commits[c] = true
		}
		for _, p := range cs.Paths {
			key := p.Action + "\x00" + p.Path
			if !seen[key] {
				seen[key] = true
				facts.Paths = append(facts.Paths, p)
			}
		}
	}
	facts.Commits = len(commits)
	if reason := s.resultsFileDisagrees(graph); reason != "" {
		degradeFreshness(&facts.Freshness, SpecFactsStale, "validation-results", reason)
	}
	if facts.Freshness.Status == SpecFactsFresh || facts.Freshness.Status == SpecFactsUnknown {
		switch missing := s.unindexedTrailerCommits(slug, commits); {
		case missing > 0:
			degradeFreshness(&facts.Freshness, SpecFactsStale, "spec-commits", fmt.Sprintf("%d commit(s) carrying this spec's trailer are not in the index", missing))
		case missing < 0 && gitHeadAtRoot(s.Root) != "":
			facts.Limitations = append(facts.Limitations, "the Git history could not be read, so whether later commits belong to this spec is unknown")
			degradeFreshness(&facts.Freshness, SpecFactsUnknown, "", "the Git history could not be read")
		}
	}
	if len(facts.Paths) == 0 {
		facts.Limitations = append(facts.Limitations, "no attributed change set is indexed for this spec; run `pose index` after committing with its POSE-Spec trailer")
	}
	switch facts.Freshness.Status {
	case SpecFactsStale:
		facts.Limitations = append(facts.Limitations, "the index is stale ("+facts.Freshness.Reason+"): run `pose index` before relying on these facts")
	case SpecFactsUnknown:
		facts.Limitations = append(facts.Limitations, "the index was not published as a set, so its freshness is unknown: run `pose index`")
	case SpecFactsUnavailable:
		facts.Limitations = append(facts.Limitations, "the index is unavailable ("+facts.Freshness.Reason+"): run `pose index`")
	}
	sort.Slice(facts.Paths, func(i, j int) bool { return facts.Paths[i].Path < facts.Paths[j].Path })
	modules := map[string]bool{}
	for _, c := range spec.Components {
		if n := normalizeFactPath(c); n != "" || strings.TrimSpace(c) != "" {
			modules[n] = true
		}
	}
	expected := ScopedDeliveryProvenanceDigest(graph, slug)
	otherSpecs := 0
	seenResult := map[string]bool{}
	for _, r := range graph.ValidationResults {
		scopedValue, scoped := r.ScopeProvenance[slug]
		if len(r.ScopeProvenance) > 0 && !scoped {
			// The index says this result was produced for other specs.
			otherSpecs++
			continue
		}
		// Distinct results are kept apart: the identity of a result is its own, not the
		// module and the name of the check, which several stacks share.
		identity := strings.Join([]string{r.ID, r.Module, r.Check, r.Outcome, r.GitHead, r.Report}, "\x00")
		if seenResult[identity] {
			continue
		}
		seenResult[identity] = true
		fact := SpecFactCheck{ID: r.ID, Module: r.Module, Check: r.Check, EvidenceClass: r.EvidenceClass, Outcome: r.Outcome,
			Binding: SpecFactBinding{GitHead: r.GitHead, ProvenanceDigest: r.ProvenanceDigest, GeneratedAt: r.GeneratedAt, Head: bindingHead(r.GitHead, heads),
				ScopedDigest: scopedValue, ExpectedScopedDigest: expected}}
		covered, inferred := moduleCoversSpec(r.Module, modules, facts.Paths)
		switch {
		case scoped && scopedValue != "" && scopedValue == expected && covered:
			fact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"
			if inferred {
				fact.Limits = append(fact.Limits, "the result's module contains the spec's components or paths without being one of them: coverage is inferred from the module's ancestry, not shown")
			}
		case scoped:
			// The index ties the result to the spec by name, but the tie does not hold: it
			// is a digest of other claims, of nothing, or of a module that is not the spec's.
			fact.Classification, fact.Basis = SpecFactCandidate, "scope-provenance-unverified"
			if scopedValue == "" || scopedValue != expected {
				fact.Limits = append(fact.Limits, "the scope provenance the result carries for this spec ("+emptyAs(scopedValue, "empty")+") is not the one its claims and change sets give today ("+emptyAs(expected, "none")+")")
			}
			if !covered {
				fact.Limits = append(fact.Limits, "the result's module ("+emptyAs(r.Module, "none")+") does not cover the spec's declared components or attributed paths")
			}
		default:
			// Nothing ties this result to the spec: a module that happens to be declared, or
			// a result the index holds, does not show that the check covered its delta.
			fact.Classification, fact.Basis = SpecFactCandidate, "none"
			if modules[normalizeFactPath(r.Module)] {
				fact.Basis = "components"
				fact.Limits = append(fact.Limits, "the module is declared by the spec, but the result carries no scope provenance: it is not shown to cover this spec's delta")
			} else {
				fact.Limits = append(fact.Limits, "the result carries no scope provenance and its module is not declared by the spec: it is not evidence of this spec")
			}
		}
		if fact.Binding.Head == "other-head" {
			fact.Limits = append(fact.Limits, "the result ran against a head other than the spec's change sets; whether it can be reused is a judgment these facts do not make")
		}
		if fact.Classification == SpecFactAssociated {
			facts.Checks = append(facts.Checks, fact)
		} else {
			facts.Candidates = append(facts.Candidates, fact)
		}
	}
	order := func(list []SpecFactCheck) {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Module != list[j].Module {
				return list[i].Module < list[j].Module
			}
			if list[i].Check != list[j].Check {
				return list[i].Check < list[j].Check
			}
			if list[i].ID != list[j].ID {
				return list[i].ID < list[j].ID
			}
			return list[i].Outcome < list[j].Outcome
		})
	}
	order(facts.Checks)
	order(facts.Candidates)
	if len(facts.Checks) == 0 {
		facts.Limitations = append(facts.Limitations, "no indexed validation result is tied to this spec by scope provenance")
	}
	if len(facts.Candidates) > 0 {
		facts.Limitations = append(facts.Limitations, fmt.Sprintf("%d indexed result(s) are candidates only: nothing ties them to this spec", len(facts.Candidates)))
	}
	if otherSpecs > 0 {
		facts.Limitations = append(facts.Limitations, fmt.Sprintf("%d indexed result(s) carry scope provenance for other specs and are left out", otherSpecs))
	}
	return facts, nil
}

// bindingHead compares the head a result ran against with the heads of the spec's change sets.
func bindingHead(head string, heads map[string]bool) string {
	switch {
	case head == "":
		return "unknown-head"
	case len(heads) == 0:
		return "unknown-head"
	case heads[head]:
		return "same-head"
	}
	return "other-head"
}

// SpecSurface names how much of the template a spec carries. Full is the
// whole template; minimal keeps what governance needs from a small change:
// intent, requirements, the declared artifacts, validation with the
// requirement trace, and the final report with its follow-ups.
const (
	SpecSurfaceFull     = "full"
	SpecSurfaceStandard = "standard"
	SpecSurfaceMinimal  = "minimal"
)

// RequiredSpecSections returns the sections a surface requires.
func RequiredSpecSections(surface string) []string {
	switch surface {
	case SpecSurfaceMinimal:
		return []string{"Intent", "Requirements", "Technical Plan", "Validation", "Final Report"}
	default:
		return []string{"Intent", "Requirements", "Technical Plan", "Tasks", "Validation", "Final Report"}
	}
}

// normalizeFactPath makes a module or component comparable as a path: slash-separated,
// cleaned, "" for the repository root.
func normalizeFactPath(p string) string {
	p = path.Clean(filepath.ToSlash(strings.TrimSpace(p)))
	if p == "." || p == "/" || p == "root" {
		return ""
	}
	return strings.TrimPrefix(p, "./")
}

func emptyAs(value, word string) string {
	if value == "" {
		return word
	}
	return value
}

// moduleCoversSpec says whether a result's module covers what the spec declares or changed:
// the module is a declared component or an attributed path's module, or it is an ancestor of
// one (inferred), or the repository root. A sibling, a descendant or a module that merely
// shares a prefix does not cover it. A spec that declares nothing and changed nothing has
// nothing a module could be shown to cover.
func moduleCoversSpec(module string, components map[string]bool, paths []ObservedPath) (covered, inferred bool) {
	module = normalizeFactPath(module)
	targets := []string{}
	for c := range components {
		targets = append(targets, c)
	}
	for _, p := range paths {
		targets = append(targets, normalizeFactPath(firstObservedDeliveryPath(p)))
	}
	for _, target := range targets {
		switch {
		case module == target:
			return true, false
		case module == "" || target == "":
			return true, false
		case strings.HasPrefix(target, module+"/"):
			covered, inferred = true, true
		}
	}
	return covered, inferred
}
