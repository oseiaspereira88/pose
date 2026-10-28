package pose

// Destinations of delegated verdicts (spec
// pose-delegated-verdict-destinations).
//
// An adjudication run's verdict is recorded as an adjudication record of the
// adjudicated spec, under .pose/adjudications, which lint-spec reads (R1). A
// smoke run's observations are recorded as validation results of class
// smoke, one per delivery target it exercised, under .pose/results/smoke,
// which the surface check reads (R2). Both are written by the conclusion of
// the run, under the same checks, lock and refusals as a review conclusion
// (R3), and both are verified on read: the run's proof must verify and the
// record must be what the run's draft concludes.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// AdjudicationsDir holds one adjudication record per run (R1).
const AdjudicationsDir = ".pose/adjudications"

// SmokeResultsDir holds one smoke result document per run (R2).
const SmokeResultsDir = ".pose/results/smoke"

// SmokeEvidenceClass is the evidence class of a smoke observation.
const SmokeEvidenceClass = "smoke"

// AdjudicationQuestion is one question an adjudication answers: a decision
// or a requirement the scope's spec records.
type AdjudicationQuestion struct {
	ID   string
	Text string
}

var (
	adjudicationDecisionHeading = regexp.MustCompile(`^###\s+Decision\s+(D[0-9]{1,4})\s*$`)
	adjudicationRequirementLine = regexp.MustCompile(`^-\s+(R[0-9]{1,4}):\s*(.*)$`)
)

// AdjudicationQuestions are the decisions and the requirements of the
// scope's spec, decisions first, each once.
func (s Store) AdjudicationQuestions(scope ReviewBundleScope) ([]AdjudicationQuestion, error) {
	if scope.Slug == "" {
		return nil, nil
	}
	spec, err := s.GetSpec(scope.Slug)
	if err != nil {
		return nil, err
	}
	var decisions, requirements []AdjudicationQuestion
	seen := map[string]bool{}
	section := ""
	for _, raw := range strings.Split(strings.ReplaceAll(spec.Body, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			section = strings.ToLower(line)
			continue
		}
		if m := adjudicationDecisionHeading.FindStringSubmatch(line); m != nil && strings.Contains(section, "decisions") && !seen[m[1]] {
			decisions = append(decisions, AdjudicationQuestion{ID: m[1], Text: "decision " + m[1] + " holds as recorded"})
			seen[m[1]] = true
		}
		if m := adjudicationRequirementLine.FindStringSubmatch(line); m != nil && strings.Contains(section, "requirements") && !seen[m[1]] {
			requirements = append(requirements, AdjudicationQuestion{ID: m[1], Text: strings.TrimSpace(m[2])})
			seen[m[1]] = true
		}
	}
	return append(decisions, requirements...), nil
}

// AdjudicationRecord is the verdict of an adjudication run on its scope's
// spec (R1).
type AdjudicationRecord struct {
	SchemaVersion int                   `json:"schema_version"`
	RunID         string                `json:"run_id"`
	ProofID       string                `json:"proof_id"`
	Issuer        string                `json:"issuer"`
	BundleID      string                `json:"bundle_id"`
	BundleDigest  string                `json:"bundle_digest"`
	Scope         string                `json:"scope"`
	Spec          string                `json:"spec"`
	Decision      string                `json:"decision"`
	Questions     []ReviewDraftQuestion `json:"questions"`
	RecordedAt    string                `json:"recorded_at"`
}

// SmokeObservation is the part of a smoke result document that says which
// run observed it and what it ran (R2).
type SmokeObservation struct {
	RunID    string               `json:"run_id"`
	ProofID  string               `json:"proof_id"`
	Issuer   string               `json:"issuer"`
	BundleID string               `json:"bundle_id"`
	Scope    string               `json:"scope"`
	Surfaces []ReviewDraftSurface `json:"surfaces"`
}

// smokeResultDocument is a validation result document — the format
// `pose validate` writes and the surface check reads — with the observation
// that produced it.
type smokeResultDocument struct {
	SchemaVersion   int                `json:"schema_version"`
	GeneratedAt     string             `json:"generated_at"`
	GitHead         string             `json:"git_head,omitempty"`
	ScopeProvenance map[string]string  `json:"scope_provenance,omitempty"`
	Checks          []smokeResultCheck `json:"checks"`
	Smoke           SmokeObservation   `json:"smoke"`
}

type smokeResultCheck struct {
	ID            string `json:"id"`
	Module        string `json:"module"`
	Name          string `json:"name"`
	Severity      string `json:"severity"`
	EvidenceClass string `json:"evidence_class"`
	Outcome       string `json:"outcome"`
	Target        string `json:"target"`
}

// verdictCoverageReasons are why a draft's typed verdict does not answer
// exactly what the run was asked: every question of an adjudication, every
// delivery target of a smoke run, nothing else.
func (s Store) verdictCoverageReasons(run ReviewRun, bundle ReviewBundle, draft *ReviewDraft) []string {
	if draft == nil {
		return nil
	}
	var asked, answered []string
	switch run.Kind {
	case ReviewBriefKindAdjudication:
		questions, err := s.AdjudicationQuestions(bundle.Payload.Scope)
		if err != nil {
			return []string{"the adjudicated questions cannot be read: " + err.Error()}
		}
		for _, q := range questions {
			asked = append(asked, q.ID)
		}
		for _, q := range draft.Questions {
			answered = append(answered, q.ID)
		}
	case ReviewBriefKindSmoke:
		targets, err := s.smokeTargets(bundle.Payload.Scope)
		if err != nil {
			return []string{"the delivery targets cannot be read: " + err.Error()}
		}
		for _, t := range targets {
			asked = append(asked, t.Ref)
		}
		for _, sf := range draft.Surfaces {
			answered = append(answered, sf.Ref)
		}
	default:
		return nil
	}
	if len(asked) == 0 {
		return []string{"the scope records nothing for a " + run.Kind + " run to answer"}
	}
	var reasons []string
	have := map[string]bool{}
	for _, id := range answered {
		have[id] = true
	}
	want := map[string]bool{}
	for _, id := range asked {
		want[id] = true
		if !have[id] {
			reasons = append(reasons, id+" is not answered")
		}
	}
	for _, id := range answered {
		if !want[id] {
			reasons = append(reasons, id+" was not asked")
		}
	}
	if len(reasons) > 0 {
		return []string{"the draft's verdict does not answer what the run was asked: " + strings.Join(reasons, "; ")}
	}
	return nil
}

func (s Store) smokeTargets(scope ReviewBundleScope) ([]DeliveryTarget, error) {
	script, err := s.smokeScriptFor(scope)
	return script.Surfaces, err
}

// adjudicationRecordFor is the record an adjudication run's draft concludes.
func adjudicationRecordFor(run ReviewRun, bundle ReviewBundle, draft ReviewDraft, proofID, issuer string, at time.Time) AdjudicationRecord {
	questions := append([]ReviewDraftQuestion(nil), draft.Questions...)
	sort.Slice(questions, func(i, j int) bool { return questions[i].ID < questions[j].ID })
	return AdjudicationRecord{SchemaVersion: 1, RunID: run.RunID, ProofID: proofID, Issuer: issuer, BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest,
		Scope: bundle.Payload.Scope.Ref, Spec: bundle.Payload.Scope.Slug, Decision: draft.Decision, Questions: questions, RecordedAt: at.UTC().Format(time.RFC3339Nano)}
}

// smokeDocumentFor is the result document a smoke run's draft concludes:
// one required check of class smoke per surface, passing when the surface
// was observed.
func smokeDocumentFor(run ReviewRun, bundle ReviewBundle, draft ReviewDraft, targets []DeliveryTarget, proofID, issuer, scoped string, at time.Time) smokeResultDocument {
	byRef := map[string]DeliveryTarget{}
	for _, t := range targets {
		byRef[t.Ref] = t
	}
	surfaces := append([]ReviewDraftSurface(nil), draft.Surfaces...)
	sort.Slice(surfaces, func(i, j int) bool { return surfaces[i].Ref < surfaces[j].Ref })
	doc := smokeResultDocument{SchemaVersion: DeliveryValidationResultSchema, GeneratedAt: at.UTC().Format(time.RFC3339Nano), GitHead: run.SealedCommit,
		Smoke: SmokeObservation{RunID: run.RunID, ProofID: proofID, Issuer: issuer, BundleID: bundle.BundleID, Scope: bundle.Payload.Scope.Ref, Surfaces: surfaces}}
	if scoped != "" {
		doc.ScopeProvenance = map[string]string{bundle.Payload.Scope.Slug: scoped}
	}
	for _, sf := range surfaces {
		outcome := "fail"
		if sf.Outcome == SmokeObserved {
			outcome = "pass"
		}
		target := byRef[sf.Ref]
		doc.Checks = append(doc.Checks, smokeResultCheck{ID: "smoke-" + strings.TrimPrefix(run.RunID, "rrn-") + "-" + target.ID, Module: target.Module, Name: "delegated smoke of " + sf.Ref,
			Severity: "required", EvidenceClass: SmokeEvidenceClass, Outcome: outcome, Target: sf.Ref})
	}
	return doc
}

// recordDelegatedVerdict writes the verdict of an adjudication or a smoke
// run where its origin reads it, once, and returns where and the proof it
// names. A record an interrupted conclusion wrote is kept when it verifies as
// this run's verdict.
func (s Store) recordDelegatedVerdict(run ReviewRun, bundle ReviewBundle, draft ReviewDraft, proofID, issuer string, now time.Time) (string, string, error) {
	var rel string
	var raw []byte
	switch run.Kind {
	case ReviewBriefKindAdjudication:
		rel = AdjudicationsDir + "/" + run.RunID + ".json"
		raw, _ = json.MarshalIndent(adjudicationRecordFor(run, bundle, draft, proofID, issuer, now), "", "  ")
	case ReviewBriefKindSmoke:
		targets, err := s.smokeTargets(bundle.Payload.Scope)
		if err != nil {
			return "", "", err
		}
		scoped := ""
		if graph, err := s.GetDeliveryIntegrity(""); err == nil {
			scoped = ScopedDeliveryProvenanceDigest(graph, bundle.Payload.Scope.Slug)
		}
		rel = SmokeResultsDir + "/" + run.RunID + ".json"
		raw, _ = json.MarshalIndent(smokeDocumentFor(run, bundle, draft, targets, proofID, issuer, scoped, now), "", "  ")
	default:
		return "", "", fmt.Errorf("pose: a %s run has no verdict destination", run.Kind)
	}
	if _, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Dir(rel)), true); err != nil {
		return "", "", err
	}
	err := PublishFile(filepath.Join(s.Root, filepath.FromSlash(rel)), append(raw, '\n'), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	var conflict *PublishConflictError
	if errors.As(err, &conflict) {
		existing, why := s.verifiedVerdictProof(run.Kind, run.RunID)
		if existing == "" {
			return "", "", fmt.Errorf("pose: %s holds a record that is not this run's verdict: %s", rel, why)
		}
		return rel, existing, nil
	}
	return rel, proofID, err
}

// verifiedVerdictProof is the proof the verdict record of run names, when
// that record verifies as the run's verdict; otherwise "" and why.
func (s Store) verifiedVerdictProof(kind, runID string) (string, string) {
	switch kind {
	case ReviewBriefKindAdjudication:
		raw, err := readBoundedRegular(filepath.Join(s.Root, filepath.FromSlash(AdjudicationsDir), runID+".json"), 1<<20)
		var rec AdjudicationRecord
		if err != nil || json.Unmarshal(raw, &rec) != nil {
			return "", "the record cannot be read"
		}
		adjudications, err := s.SpecAdjudications(rec.Spec)
		if err != nil {
			return "", err.Error()
		}
		for _, a := range adjudications {
			if a.Record.RunID == runID {
				if a.Verified {
					return rec.ProofID, ""
				}
				return "", a.Why
			}
		}
		return "", "the record names no spec"
	case ReviewBriefKindSmoke:
		results, err := s.SmokeResults()
		if err != nil {
			return "", err.Error()
		}
		for _, r := range results {
			if r.Path == SmokeResultsDir+"/"+runID+".json" {
				if !r.Verified {
					return "", r.Why
				}
				raw, _ := readBoundedRegular(filepath.Join(s.Root, filepath.FromSlash(r.Path)), 1<<20)
				var doc smokeResultDocument
				_ = json.Unmarshal(raw, &doc)
				return doc.Smoke.ProofID, ""
			}
		}
		return "", "no record"
	}
	return "", "no destination"
}

// verifiedVerdictRun is the run behind a verdict record, when its proof
// verifies and the run is of kind, with its parsed draft.
func (s Store) verifiedVerdictRun(runID, proofID, kind string) (ReviewRun, ReviewBundle, ReviewDraft, error) {
	assurance, err := s.VerifyReviewRunProof(runID, proofID)
	if err != nil {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, err
	}
	if !assurance.CompleteReview {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, fmt.Errorf("the run's proof does not establish a complete run")
	}
	run, err := s.LoadProvenanceReviewRun(runID)
	if err != nil {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, err
	}
	if run.Kind != kind {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, fmt.Errorf("the run is a %s run, not %s", run.Kind, kind)
	}
	draft, err := ParseReviewDraft(run.Output)
	if err != nil || draft == nil {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, fmt.Errorf("the run's draft cannot be read")
	}
	bundle, err := s.LoadReviewBundle(run.BundleID)
	if err != nil {
		return ReviewRun{}, ReviewBundle{}, ReviewDraft{}, err
	}
	return run, bundle, *draft, nil
}

// VerifiedAdjudication is an adjudication record and whether it is the
// verified verdict of its run, and of the spec's current bundle.
type VerifiedAdjudication struct {
	Record   AdjudicationRecord
	Path     string
	Verified bool
	Why      string
	Current  bool
}

// SpecAdjudications are the adjudication records of slug, each verified:
// its run's proof verifies and the record is what the run's draft
// concludes (R1).
func (s Store) SpecAdjudications(slug string) ([]VerifiedAdjudication, error) {
	dir := filepath.Join(s.Root, filepath.FromSlash(AdjudicationsDir))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []VerifiedAdjudication
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		rel := AdjudicationsDir + "/" + e.Name()
		raw, err := readBoundedRegular(filepath.Join(dir, e.Name()), 1<<20)
		var rec AdjudicationRecord
		if err != nil || json.Unmarshal(raw, &rec) != nil {
			out = append(out, VerifiedAdjudication{Path: rel, Why: "the record cannot be read"})
			continue
		}
		if rec.Spec != slug {
			continue
		}
		v := VerifiedAdjudication{Record: rec, Path: rel}
		run, bundle, draft, err := s.verifiedVerdictRun(rec.RunID, rec.ProofID, ReviewBriefKindAdjudication)
		if err != nil {
			v.Why = err.Error()
		} else if want := adjudicationRecordFor(run, bundle, draft, rec.ProofID, rec.Issuer, time.Time{}); !sameAdjudication(want, rec) || e.Name() != rec.RunID+".json" {
			v.Why = "the record is not what the run's draft concludes"
		} else {
			v.Verified = true
			if current, err := s.CurrentReviewBundle(rec.Scope); err == nil && current != nil && current.BundleID == rec.BundleID {
				v.Current = true
			}
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func sameAdjudication(a, b AdjudicationRecord) bool {
	a.RecordedAt, b.RecordedAt = "", ""
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// SmokeResult is one smoke result document, verified or why not.
type SmokeResult struct {
	Path     string
	Results  []DeliveryValidationResult
	Verified bool
	Why      string
}

// SmokeResults are the smoke result documents, each verified: its run's
// proof verifies and its checks are what the run's draft concludes (R2).
// Only verified documents yield results.
func (s Store) SmokeResults() ([]SmokeResult, error) {
	return LoadSmokeResultsIn(s, s.Root)
}

// LoadSmokeResultsIn reads the smoke result documents of root.
func LoadSmokeResultsIn(s Store, root string) ([]SmokeResult, error) {
	dir := filepath.Join(root, filepath.FromSlash(SmokeResultsDir))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []SmokeResult
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		rel := SmokeResultsDir + "/" + e.Name()
		r := SmokeResult{Path: rel}
		raw, err := readBoundedRegular(filepath.Join(dir, e.Name()), 1<<20)
		var doc smokeResultDocument
		if err != nil || json.Unmarshal(raw, &doc) != nil {
			r.Why = "the document cannot be read"
			out = append(out, r)
			continue
		}
		run, bundle, draft, err := s.verifiedVerdictRun(doc.Smoke.RunID, doc.Smoke.ProofID, ReviewBriefKindSmoke)
		if err != nil {
			r.Why = err.Error()
			out = append(out, r)
			continue
		}
		targets, err := s.smokeTargets(bundle.Payload.Scope)
		if err != nil {
			r.Why = err.Error()
			out = append(out, r)
			continue
		}
		want := smokeDocumentFor(run, bundle, draft, targets, doc.Smoke.ProofID, doc.Smoke.Issuer, "", time.Time{})
		if !sameSmokeChecks(want, doc) || e.Name() != doc.Smoke.RunID+".json" {
			r.Why = "the document is not what the run's draft concludes"
			out = append(out, r)
			continue
		}
		results, err := ParseDeliveryValidationRun(raw, rel)
		if err != nil {
			r.Why = err.Error()
			out = append(out, r)
			continue
		}
		r.Results, r.Verified = results, true
		out = append(out, r)
	}
	return out, nil
}

func sameSmokeChecks(want, got smokeResultDocument) bool {
	x, _ := json.Marshal(struct {
		Checks []smokeResultCheck
		Smoke  SmokeObservation
		Head   string
	}{want.Checks, want.Smoke, want.GitHead})
	y, _ := json.Marshal(struct {
		Checks []smokeResultCheck
		Smoke  SmokeObservation
		Head   string
	}{got.Checks, got.Smoke, got.GitHead})
	return string(x) == string(y)
}
