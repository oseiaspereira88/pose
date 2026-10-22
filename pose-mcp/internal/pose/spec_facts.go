package pose

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	ID string `json:"id,omitempty"`
	// Source is the results file the result was read from, and Severity says whether the check
	// was required: two results that differ only in them are two results.
	Source         string          `json:"source,omitempty"`
	Severity       string          `json:"severity,omitempty"`
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
	// The graph the facts are made from was read from one generation: the one this store is
	// pinned to, if it is pinned. The judgment is about that generation, and a newer one is a
	// finding of its own, not a reason to label the facts of an older one with the newer's identity.
	if pinned := s.indexPin; pinned != nil && pinned.Status != IndexGenerationCurrent {
		return SpecFactsFreshness{Status: SpecFactsUnknown, Reason: "the graph was read without a published generation: " + pinned.Reason}
	}
	if pinned := s.indexPin; pinned != nil && pinned.Status == IndexGenerationCurrent {
		if view.Status == IndexGenerationCurrent && view.Descriptor.GenerationID != pinned.Descriptor.GenerationID {
			return SpecFactsFreshness{Status: SpecFactsStale, Generation: pinned.Descriptor.GenerationID,
				Reason: "generation " + view.Descriptor.GenerationID + " was published after the one these facts were read from"}
		}
		view = *pinned
	}
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

// resultsFileDisagrees reports why the validation results file the index was built from no
// longer matches the results the index holds, or "" when it matches or there is none. The file is
// read the way the index read it (ParseDeliveryValidationRun) and compared result by result, in
// every field the index keeps: a head, a scope provenance, a module or a class that moved is a
// results file the facts no longer describe. A file that cannot be read, a contract this engine
// does not read and a policy that cannot be loaded are not agreement.
func (s Store) resultsFileDisagrees(graph DeliveryIntegrityGraph) string {
	policy, err := LoadDeliveryPolicy(s.Root)
	if err != nil {
		return "the delivery policy could not be loaded, so the results file it names is unknown: " + err.Error()
	}
	held := []DeliveryValidationResult{}
	for _, r := range graph.ValidationResults {
		// A result with no report was not read from a results file (a fixture, an older index).
		if r.Report != "" && (policy.ResultsPath == "" || r.Report == policy.ResultsPath) {
			held = append(held, r)
		}
	}
	if policy.ResultsPath == "" {
		if len(held) > 0 {
			return "the delivery policy names no results file, and the index holds results read from one"
		}
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(policy.ResultsPath)))
	if errors.Is(err, os.ErrNotExist) {
		if len(held) > 0 {
			return policy.ResultsPath + " is gone, and the index holds results read from it"
		}
		return ""
	}
	if err != nil {
		return policy.ResultsPath + " could not be read: " + err.Error()
	}
	now, perr := ParseDeliveryValidationRun(raw, policy.ResultsPath)
	if perr != nil {
		return policy.ResultsPath + " is not a validation result this engine reads"
	}
	if len(now) != len(held) {
		return fmt.Sprintf("%s holds %d result(s), the index holds %d read from it", policy.ResultsPath, len(now), len(held))
	}
	keys := func(list []DeliveryValidationResult) []string {
		out := make([]string, 0, len(list))
		for _, r := range list {
			out = append(out, factResultKey(r))
		}
		sort.Strings(out)
		return out
	}
	a, b := keys(now), keys(held)
	for i := range a {
		if a[i] != b[i] {
			return policy.ResultsPath + " was written again after the index read it"
		}
	}
	return ""
}

// factResultKey is the identity of a result in every field the index keeps, the scope
// provenance in a fixed order: two results with the same key are the same result.
func factResultKey(r DeliveryValidationResult) string {
	scopes := make([][2]string, 0, len(r.ScopeProvenance))
	for slug, digest := range r.ScopeProvenance {
		scopes = append(scopes, [2]string{slug, digest})
	}
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i][0] != scopes[j][0] {
			return scopes[i][0] < scopes[j][0]
		}
		return scopes[i][1] < scopes[j][1]
	})
	// JSON is injective over the fields: no separator a field may hold can make two results one.
	raw, _ := json.Marshal([]any{r.ID, r.Module, r.Check, r.EvidenceClass, r.Severity, r.Outcome, r.GitHead, r.GeneratedAt, r.ProvenanceDigest, r.Report, scopes})
	return string(raw)
}

// componentDeclares reports that module is written, after normalization, as a declared component.
func componentDeclares(components map[string]bool, module string) bool {
	normalized, kind := factModuleKind(module)
	if kind != "path" && kind != "root" {
		return false
	}
	for c := range components {
		if n, k := factModuleKind(c); (k == "path" || k == "root") && n == normalized {
			return true
		}
	}
	return false
}

// DeriveSpecFacts reads the delivery index. A spec with no attributed change
// set says so instead of reporting an empty delivery.
func (s Store) DeriveSpecFacts(slug string) (SpecFacts, error) {
	if s.indexPin == nil {
		s = s.PinIndexes()
	}
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
	resolvedSets, unresolvedSets := 0, 0
	for _, cs := range graph.ChangeSets {
		if cs.Spec != slug {
			continue
		}
		// A change set is resolved when Git gave it the head, the commits and the paths together. The
		// predicate holds for all of the spec's change sets: one that is resolved says nothing about
		// the delta another one stands for.
		if changeSetIsResolved(cs) {
			resolvedSets++
		} else {
			unresolvedSets++
		}
		if cs.ResolvedHead != "" {
			heads[cs.ResolvedHead] = true
		}
		for _, c := range cs.Commits {
			commits[c] = true
		}
		for _, p := range cs.Paths {
			// A rename has no Path: its identity is both of its ends.
			key := p.Action + "\x00" + p.Path + "\x00" + p.OldPath + "\x00" + p.NewPath
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
	// The history is an axis a generation does not bind. It is read the way the attribution of
	// change sets reads it (every ref, the trailer lines of the body), and a history that cannot
	// be read leaves the facts unknown rather than fresh.
	if facts.Freshness.Status == SpecFactsFresh || facts.Freshness.Status == SpecFactsUnknown {
		trailers, terr := SpecTrailerCommits(s.Root, slug)
		missing, gone := 0, 0
		current := map[string]bool{}
		for _, c := range trailers {
			current[c] = true
			if !commits[c] {
				missing++
			}
		}
		// And the other way: a commit the index attributed by the trailer that no longer carries it
		// (the branch was deleted, the history rewritten) is attribution the index no longer has.
		for _, cs := range graph.ChangeSets {
			if cs.Spec != slug || !strings.HasPrefix(cs.Selector, "trailers:") {
				continue
			}
			for _, c := range cs.Commits {
				if !current[c] {
					gone++
				}
			}
		}
		explicit, explicitErr := hasExplicitAttribution(s.Root, slug)
		for _, cs := range graph.ChangeSets {
			if cs.Spec == slug && !strings.HasPrefix(cs.Selector, "trailers:") {
				explicit = true
			}
		}
		switch {
		case terr != nil:
			facts.Limitations = append(facts.Limitations, "the Git history could not be read, so whether commits belong to this spec that the index does not hold is unknown")
			degradeFreshness(&facts.Freshness, SpecFactsUnknown, "", "the Git history could not be read")
		case missing > 0:
			degradeFreshness(&facts.Freshness, SpecFactsStale, "spec-commits", fmt.Sprintf("%d commit(s) carrying this spec's trailer are not in the index", missing))
		case gone > 0:
			degradeFreshness(&facts.Freshness, SpecFactsStale, "spec-commits", fmt.Sprintf("%d commit(s) the index attributed to this spec by trailer no longer carry it", gone))
		case explicitErr != nil || explicit:
			facts.Limitations = append(facts.Limitations, "explicit attribution records (.pose/reports/history) name this spec or could not be read, and the generation does not bind them")
			degradeFreshness(&facts.Freshness, SpecFactsUnknown, "", "explicit attribution records are not bound by the generation")
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
	pathKey := func(p ObservedPath) string {
		return p.Path + "\x00" + p.OldPath + "\x00" + p.NewPath + "\x00" + p.Action
	}
	sort.Slice(facts.Paths, func(i, j int) bool { return pathKey(facts.Paths[i]) < pathKey(facts.Paths[j]) })
	modules := map[string]bool{}
	for _, c := range spec.Components {
		if strings.TrimSpace(c) != "" {
			modules[c] = true
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
		identity := factResultKey(r)
		if seenResult[identity] {
			continue
		}
		seenResult[identity] = true
		fact := SpecFactCheck{ID: r.ID, Source: r.Report, Severity: r.Severity, Module: r.Module, Check: r.Check, EvidenceClass: r.EvidenceClass, Outcome: r.Outcome,
			Binding: SpecFactBinding{GitHead: r.GitHead, ProvenanceDigest: r.ProvenanceDigest, GeneratedAt: r.GeneratedAt, Head: bindingHead(r.GitHead, heads),
				ScopedDigest: scopedValue, ExpectedScopedDigest: expected}}
		covered, inferred := moduleCoversSpec(r.Module, modules, facts.Paths)
		switch {
		case scoped && scopedValue != "" && scopedValue == expected && covered && resolvedSets > 0 && unresolvedSets == 0:
			fact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"
			if inferred {
				fact.Limits = append(fact.Limits, "the result's module contains the spec's components or paths without being one of them: coverage is inferred from the module's ancestry, not shown")
			}
		case scoped:
			// The index ties the result to the spec by name, but the tie does not hold: it
			// is a digest of other claims, of nothing, or of a module that is not the spec's.
			fact.Classification, fact.Basis = SpecFactCandidate, "scope-provenance-unverified"
			if resolvedSets == 0 {
				fact.Limits = append(fact.Limits, "the spec has no resolved change set with paths: the scope provenance of the result is the digest of nothing and binds it to no delta")
			} else if unresolvedSets > 0 {
				fact.Limits = append(fact.Limits, "the spec has a change set that is not resolved: the scope provenance binds the result to a delta that is not fully observed")
			} else if scopedValue == "" || scopedValue != expected {
				fact.Limits = append(fact.Limits, "the scope provenance the result carries for this spec ("+emptyAs(scopedValue, "empty")+") is not the one its claims and change sets give today ("+emptyAs(expected, "none")+")")
			}
			if !covered {
				fact.Limits = append(fact.Limits, "the result's module ("+emptyAs(r.Module, "none")+") does not cover the spec's declared components or attributed paths")
			}
			if strings.TrimSpace(r.Module) == "" {
				fact.Limits = append(fact.Limits, "the result names no module: an omitted module is not the root of the repository")
			}
		default:
			// Nothing ties this result to the spec: a module that happens to be declared, or
			// a result the index holds, does not show that the check covered its delta.
			fact.Classification, fact.Basis = SpecFactCandidate, "none"
			if componentDeclares(modules, r.Module) {
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

// factModuleKind says what a module or component reference is: unknown (absent), invalid (an
// absolute path or one that leaves the project), the root of the repository (written as such) or
// a path. An omitted module is not the root.
func factModuleKind(p string) (normalized, kind string) {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return "", "unknown"
	}
	slashed := filepath.ToSlash(trimmed)
	if strings.HasPrefix(slashed, "/") || filepath.IsAbs(trimmed) {
		return "", "invalid"
	}
	cleaned := path.Clean(slashed)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", "invalid"
	}
	if cleaned == "." || cleaned == "root" {
		return "", "root"
	}
	return strings.TrimPrefix(cleaned, "./"), "path"
}

// normalizeFactPath makes a module or a component comparable as a path: slash-separated,
// cleaned, "" for the root. It does not make an unknown or invalid reference the root; callers
// that need the difference ask factModuleKind.
func normalizeFactPath(p string) string {
	normalized, _ := factModuleKind(p)
	return normalized
}

func emptyAs(value, word string) string {
	if value == "" {
		return word
	}
	return value
}

// moduleCoversSpec says whether a result's module covers what the spec declares or changed:
// the module is a declared component or an attributed path's module, or it is an ancestor of
// one (inferred), or it is the repository root written as such. A module that is absent or
// invalid, a sibling, a descendant or a module that merely shares a prefix does not cover it. Both
// ends of a rename are attributed: the path the change left and the one it reached. A spec that
// declares nothing and changed nothing has nothing a module could be shown to cover.
func moduleCoversSpec(module string, components map[string]bool, paths []ObservedPath) (covered, inferred bool) {
	normalized, kind := factModuleKind(module)
	if kind == "unknown" || kind == "invalid" {
		return false, false
	}
	// Two kinds of target, read two ways. A component is a reference to a module: it may be written
	// as the root and is normalized. A path is a file name Git reported, byte for byte: a file called
	// "root" is a file, and a name with a space at its start is not the directory without it.
	rootDeclared := false
	targets := []string{}
	for c := range components {
		if n, k := factModuleKind(c); k == "path" {
			targets = append(targets, n)
		} else if k == "root" {
			rootDeclared = true
		}
	}
	for _, p := range paths {
		if !observedPathIsWellFormed(p) {
			continue
		}
		for _, candidate := range []string{p.Path, p.OldPath, p.NewPath} {
			if candidate != "" && !strings.HasPrefix(candidate, "/") && candidate != ".." && !strings.HasPrefix(candidate, "../") {
				targets = append(targets, candidate)
			}
		}
	}
	if len(targets) == 0 && !rootDeclared {
		return false, false
	}
	if rootDeclared || kind == "root" {
		return true, false
	}
	for _, target := range targets {
		switch {
		case normalized == target:
			return true, false
		case strings.HasPrefix(target, normalized+"/"):
			covered, inferred = true, true
		}
	}
	return covered, inferred
}

// hasExplicitAttribution reports that the recorded change sets (.pose/reports/history) hold a
// record of the spec, or that they could not be read. The generation binds neither them nor
// their absence, so the facts of a spec they name cannot be called fresh by it.
func hasExplicitAttribution(root, slug string) (bool, error) {
	dir := filepath.Join(root, ".pose", "reports", "history")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		if _, lerr := os.Lstat(dir); lerr == nil {
			return true, err // a link to a place that was lost
		}
		return false, nil
	}
	if err != nil {
		return true, err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			return true, rerr
		}
		scanner := bufio.NewScanner(bytes.NewReader(raw))
		for scanner.Scan() {
			if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
				continue
			}
			var record struct {
				Spec      string     `json:"spec"`
				ChangeSet *ChangeSet `json:"change_set"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				return true, err
			}
			if record.Spec == slug || (record.ChangeSet != nil && record.ChangeSet.Spec == slug) {
				return true, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return true, err
		}
	}
	return false, nil
}

// changeSetIsResolved says that a change set carries what resolving it produces, together and in
// shape: a base and a head, a diff digest, commits that are named, and paths that are one of the
// four actions with the fields that action has. Anything less is not a delta observed, and binds
// nothing.
func changeSetIsResolved(cs ChangeSet) bool {
	if cs.ResolvedBase == "" || cs.ResolvedHead == "" || cs.DiffDigest == "" || len(cs.Commits) == 0 || len(cs.Paths) == 0 {
		return false
	}
	for _, c := range cs.Commits {
		if c == "" {
			return false
		}
	}
	for _, p := range cs.Paths {
		if !observedPathIsWellFormed(p) {
			return false
		}
	}
	return true
}

// observedPathIsWellFormed: created, modified and removed name a Path; renamed names both of its ends.
func observedPathIsWellFormed(p ObservedPath) bool {
	switch p.Action {
	case "created", "modified", "removed":
		return observedGitPath(p.Path) && p.OldPath == "" && p.NewPath == ""
	case "renamed":
		return p.Path == "" && observedGitPath(p.OldPath) && observedGitPath(p.NewPath)
	}
	return false
}

// A Git path is a literal canonical relative name. Do not normalize aliases or spaces;
// removed paths and symlink entries need no current filesystem existence check.
func observedGitPath(value string) bool {
	if value == "" || strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
