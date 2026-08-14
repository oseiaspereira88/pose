package pose

// Review bundles provide the fixed review subject described by
// ADR 2026-08-13-sealed-review-bundles-and-attestations. A bundle hashes only
// governed semantic inputs. Its attestation is a separate immutable artifact,
// so recording approval and applying closeout cannot invalidate that approval.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ReviewBundleSchemaVersion = 1
const maxReviewBundleBytes = 4 << 20

type ReviewBundleInput struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Digest string `json:"digest,omitempty"`
	Reason string `json:"reason"`
}

type ReviewBundleScope struct {
	Ref        string              `json:"ref"`
	Kind       string              `json:"kind"`
	Slug       string              `json:"slug,omitempty"`
	Roadmap    string              `json:"roadmap,omitempty"`
	Milestone  string              `json:"milestone,omitempty"`
	DependsOn  []string            `json:"depends_on,omitempty"`
	Supersedes string              `json:"supersedes,omitempty"`
	Components []string            `json:"components,omitempty"`
	Deliveries []string            `json:"deliveries,omitempty"`
	Sections   []ReviewBundleInput `json:"sections"`
}

type ReviewBundleSubjectEntry struct {
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
	Class   string `json:"class"`
	Digest  string `json:"digest,omitempty"`
	Reason  string `json:"reason"`
}

type ReviewBundleSubject struct {
	ChangeSets  []string                   `json:"change_sets"`
	Base        string                     `json:"base,omitempty"`
	Head        string                     `json:"head,omitempty"`
	PatchDigest string                     `json:"patch_digest"`
	TreeDigest  string                     `json:"tree_digest"`
	Entries     []ReviewBundleSubjectEntry `json:"entries"`
}

type ReviewBundlePlan struct {
	PlanDigest   string                `json:"plan_digest"`
	Independence string                `json:"independence"`
	Components   []ReviewPlanComponent `json:"components"`
	Criteria     []ReviewPlanCriterion `json:"criteria"`
	Tools        []ReviewPlanTool      `json:"tools"`
}

type ReviewBundleEvidence struct {
	ID               string `json:"id"`
	Module           string `json:"module,omitempty"`
	Check            string `json:"check"`
	EvidenceClass    string `json:"evidence_class"`
	Outcome          string `json:"outcome"`
	GitHead          string `json:"git_head,omitempty"`
	ProvenanceDigest string `json:"provenance_digest,omitempty"`
	Report           string `json:"report,omitempty"`
}

type ReviewBundleChild struct {
	Scope        string `json:"scope"`
	BundleID     string `json:"bundle_id"`
	BundleDigest string `json:"bundle_digest"`
}

type ReviewBundlePayload struct {
	Scope          ReviewBundleScope      `json:"scope"`
	Subject        ReviewBundleSubject    `json:"subject"`
	Plan           ReviewBundlePlan       `json:"plan"`
	Evidence       []ReviewBundleEvidence `json:"evidence"`
	Children       []ReviewBundleChild    `json:"children,omitempty"`
	ConsumedInputs []ReviewBundleInput    `json:"consumed_inputs"`
}

type ReviewBundle struct {
	SchemaVersion  int                 `json:"schema_version"`
	BundleID       string              `json:"bundle_id"`
	BundleDigest   string              `json:"bundle_digest"`
	State          string              `json:"state"`
	SealedAt       string              `json:"sealed_at,omitempty"`
	Payload        ReviewBundlePayload `json:"payload"`
	ExcludedInputs []ReviewBundleInput `json:"excluded_inputs,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
	Blockers       []string            `json:"blockers,omitempty"`
	Path           string              `json:"path,omitempty"`
}

type ReviewAttestationReuse struct {
	Criterion       string `json:"criterion"`
	FromAttestation string `json:"from_attestation"`
	InputDigest     string `json:"input_digest"`
}

type ReviewAttestationSignature struct {
	Issuer    string `json:"issuer"`
	Subject   string `json:"subject"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type ReviewAttestation struct {
	SchemaVersion int                         `json:"schema_version"`
	AttestationID string                      `json:"attestation_id"`
	BundleID      string                      `json:"bundle_id"`
	BundleDigest  string                      `json:"bundle_digest"`
	Reviewer      string                      `json:"reviewer"`
	Decision      string                      `json:"decision"`
	Criteria      []ReviewCriterion           `json:"criteria"`
	Tools         []ReviewToolDisposition     `json:"tools,omitempty"`
	EvidenceRefs  []string                    `json:"evidence_refs,omitempty"`
	Findings      []ReviewFinding             `json:"findings"`
	ReusedFrom    []ReviewAttestationReuse    `json:"reused_from,omitempty"`
	Supersedes    string                      `json:"supersedes,omitempty"`
	Envelope      *ReviewAttestationSignature `json:"envelope,omitempty"`
	AttestedAt    string                      `json:"attested_at"`
	Path          string                      `json:"path,omitempty"`
}

type ReviewAttestationEnvelope struct {
	SchemaVersion int               `json:"schema_version"`
	Issuer        string            `json:"issuer"`
	Subject       string            `json:"subject"`
	Algorithm     string            `json:"algorithm"`
	PublicKey     string            `json:"public_key"`
	Signature     string            `json:"signature"`
	Attestation   ReviewAttestation `json:"attestation"`
}

type ReviewBundleDelta struct {
	FromBundle             string   `json:"from_bundle,omitempty"`
	ToBundle               string   `json:"to_bundle"`
	ChangedComponents      []string `json:"changed_components,omitempty"`
	ChangedSections        []string `json:"changed_sections,omitempty"`
	ChangedPaths           []string `json:"changed_paths,omitempty"`
	ChangedCriteria        []string `json:"changed_criteria,omitempty"`
	ChangedEvidence        []string `json:"changed_evidence,omitempty"`
	ChangedEvidenceClasses []string `json:"changed_evidence_classes,omitempty"`
	ChangedFindings        []string `json:"changed_findings,omitempty"`
	ReusableCriteria       []string `json:"reusable_criteria,omitempty"`
}

type ReviewBundleVerification struct {
	Scope       string             `json:"scope"`
	State       string             `json:"state"`
	Bundle      *ReviewBundle      `json:"bundle,omitempty"`
	Attestation *ReviewAttestation `json:"attestation,omitempty"`
	Delta       *ReviewBundleDelta `json:"delta,omitempty"`
	Fresh       bool               `json:"fresh"`
	Approved    bool               `json:"approved"`
	NextAction  string             `json:"next_action"`
	Warnings    []string           `json:"warnings,omitempty"`
	Blockers    []string           `json:"blockers,omitempty"`
}

// PrepareReviewBundle resolves a deterministic bundle without writing it.
func (s Store) PrepareReviewBundle(ref string) (ReviewBundle, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	plan, err := s.ReviewPlan(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle := ReviewBundle{SchemaVersion: ReviewBundleSchemaVersion, State: "prepared", Warnings: uniqueSorted(append([]string{}, plan.Warnings...)), Blockers: uniqueSorted(append([]string{}, plan.Blockers...))}
	bundlePlanDigest, err := reviewBundlePlanDigest(plan)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.Payload.Plan = ReviewBundlePlan{PlanDigest: bundlePlanDigest, Independence: plan.Independence, Components: append([]ReviewPlanComponent{}, plan.Components...), Criteria: append([]ReviewPlanCriterion{}, plan.Criteria...), Tools: append([]ReviewPlanTool{}, plan.Tools...)}

	scopeProjection, excluded, err := s.reviewBundleScopeProjection(scope)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.Payload.Scope = scopeProjection
	bundle.ExcludedInputs = excluded

	graph, graphErr := s.GetDeliveryIntegrity("")
	if graphErr != nil {
		bundle.Blockers = append(bundle.Blockers, "delivery integrity is unavailable: run pose index")
	} else {
		var subjectExcluded []ReviewBundleInput
		bundle.Payload.Subject, subjectExcluded, bundle.Blockers, err = s.reviewBundleSubject(scope, bundle.Payload.Plan.Components, graph, bundle.Blockers)
		if err != nil {
			return ReviewBundle{}, err
		}
		bundle.ExcludedInputs = append(bundle.ExcludedInputs, subjectExcluded...)
		bundle.Payload.Evidence = s.reviewBundleEvidence(scope, graph)
		if len(bundle.Payload.Evidence) == 0 {
			bundle.Blockers = append(bundle.Blockers, "no passed structured validation evidence is attributed to the review scope")
		}
	}

	if scope.Kind != "spec" {
		children, childBlockers, childErr := s.reviewBundleChildren(scope)
		if childErr != nil {
			return ReviewBundle{}, childErr
		}
		bundle.Payload.Children = children
		bundle.Blockers = append(bundle.Blockers, childBlockers...)
	}

	bundle.Payload.ConsumedInputs = s.reviewBundleConsumedInputs(plan)
	bundle.Blockers = uniqueSorted(bundle.Blockers)
	bundle.Warnings = uniqueSorted(bundle.Warnings)
	bundle.ExcludedInputs = sortedBundleInputs(bundle.ExcludedInputs)
	normalizeReviewBundlePayload(&bundle.Payload)
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.BundleDigest = digest
	bundle.BundleID = "rvb-" + strings.TrimPrefix(digest, "sha256:")[:16]
	return bundle, nil
}

func (s Store) reviewBundleScopeProjection(scope ScopeRef) (ReviewBundleScope, []ReviewBundleInput, error) {
	projection := ReviewBundleScope{Ref: scope.String(), Kind: scope.Kind, Slug: scope.Slug, Roadmap: scope.Roadmap, Milestone: scope.Milestone, Sections: []ReviewBundleInput{}}
	excluded := []ReviewBundleInput{}
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return projection, nil, err
		}
		projection.DependsOn = append([]string{}, sp.DependsOn...)
		projection.Supersedes = sp.Supersedes
		projection.Components = append([]string{}, sp.Components...)
		projection.Deliveries = append([]string{}, sp.Delivers...)
		sort.Strings(projection.DependsOn)
		sort.Strings(projection.Components)
		sort.Strings(projection.Deliveries)
		sections := markdownLevelTwoSections(sp.Body)
		for _, name := range []string{"intent", "requirements", "technical plan", "decisions"} {
			body := sections[name]
			if body == "" {
				continue
			}
			projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "semantic-section", Path: name, Digest: digestText(body), Reason: "governed semantic review input"})
		}
		for _, name := range []string{"tasks", "validation", "final report"} {
			if sections[name] != "" {
				excluded = append(excluded, ReviewBundleInput{Kind: "derived-section", Path: name, Digest: digestText(sections[name]), Reason: "operational closeout content is verified by its own gate"})
			}
		}
		excluded = append(excluded, ReviewBundleInput{Kind: "lifecycle", Path: filepath.ToSlash(sp.Path), Reason: "status and completed_at do not define the reviewed semantic subject"})
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return projection, nil, err
		}
		content := normalizeBundleText(roadmapMilestoneSection(rm.Body, scope.Milestone))
		if content == "" {
			return projection, nil, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
		}
		projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "milestone", Path: scope.Milestone, Digest: digestText(content), Reason: "governed milestone exit and cut criteria"})
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return projection, nil, err
		}
		projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "roadmap", Path: filepath.ToSlash(rm.Path), Digest: digestText(normalizeBundleText(rm.Body)), Reason: "governed roadmap outcomes and ordered membership"})
	}
	return projection, excluded, nil
}

func markdownLevelTwoSections(body string) map[string]string {
	sections := map[string]string{}
	current := ""
	var lines []string
	flush := func() {
		if current != "" {
			sections[current] = normalizeBundleText(strings.Join(lines, "\n"))
		}
		lines = nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flush()
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if dot := strings.Index(title, "."); dot >= 0 && dot < 3 {
				title = strings.TrimSpace(title[dot+1:])
			}
			current = strings.ToLower(title)
			continue
		}
		if current != "" {
			lines = append(lines, strings.TrimRight(raw, " \t"))
		}
	}
	flush()
	return sections
}

func (s Store) reviewBundleSubject(scope ScopeRef, components []ReviewPlanComponent, graph DeliveryIntegrityGraph, blockers []string) (ReviewBundleSubject, []ReviewBundleInput, []string, error) {
	subject := ReviewBundleSubject{ChangeSets: []string{}, Entries: []ReviewBundleSubjectEntry{}}
	excluded := []ReviewBundleInput{}
	allowedSpecs, err := s.reviewBundleScopeSpecs(scope)
	if err != nil {
		return subject, excluded, blockers, err
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if allowedSpecs[set.Spec] {
			sets = append(sets, set)
		}
	}
	sets = reduceReviewBundleChangeSets(sets)
	if len(sets) == 0 {
		blockers = append(blockers, "no immutable attributed change set exists for "+scope.String())
		return subject, excluded, blockers, nil
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].ID < sets[j].ID })
	seen := map[string]bool{}
	supersededPaths := map[string]bool{}
	for _, set := range sets {
		for _, observed := range set.Paths {
			if observed.Action == "renamed" && observed.OldPath != "" {
				supersededPaths[observed.OldPath] = true
			}
		}
	}
	for _, set := range sets {
		subject.ChangeSets = append(subject.ChangeSets, set.ID)
		if subject.Base == "" {
			subject.Base = set.ResolvedBase
		}
		subject.Head = set.ResolvedHead
		for _, observed := range set.Paths {
			if observed.Action != "renamed" && supersededPaths[observed.Path] {
				excluded = append(excluded, ReviewBundleInput{Kind: "superseded-path", Path: observed.Path, Reason: "a later attributed rename supplies the current review subject identity"})
				continue
			}
			key := observedKey(observed)
			if seen[key] {
				continue
			}
			seen[key] = true
			entry := ReviewBundleSubjectEntry{Action: observed.Action, Path: observed.Path, OldPath: observed.OldPath, NewPath: observed.NewPath}
			path := observed.Path
			if observed.Action == "renamed" {
				path = observed.NewPath
			}
			class, include := reviewBundlePathClass(path, scope, components)
			if class == "" {
				blockers = append(blockers, "unclassified review subject path "+path)
				entry.Class = "unclassified"
				entry.Reason = "attributed path has no governed review classification"
			} else {
				entry.Class = class
				entry.Reason = "attributed " + class + " path in the immutable change set"
			}
			if !include {
				excluded = append(excluded, ReviewBundleInput{Kind: class, Path: path, Reason: "attributed path is outside the semantic review subject"})
				continue
			}
			if dirty, detail := reviewBundleWorkingTreeChange(s.Root, path); dirty {
				blockers = append(blockers, "review subject path "+path+" has working-tree-only content"+detail)
			}
			if include && observed.Action != "removed" {
				digest, err := s.reviewBundleFileDigest(path)
				if err != nil {
					blockers = append(blockers, err.Error())
				} else {
					entry.Digest = digest
				}
			}
			subject.Entries = append(subject.Entries, entry)
		}
	}
	sort.Slice(subject.Entries, func(i, j int) bool {
		a, b := subject.Entries[i], subject.Entries[j]
		return a.Action+"\x00"+a.Path+"\x00"+a.OldPath+"\x00"+a.NewPath < b.Action+"\x00"+b.Path+"\x00"+b.OldPath+"\x00"+b.NewPath
	})
	patchRaw, _ := json.Marshal(subject.Entries)
	subject.PatchDigest = digestBytes(patchRaw)
	treeEntries := make([]struct {
		Path, Digest string
	}, 0, len(subject.Entries))
	for _, entry := range subject.Entries {
		path := entry.Path
		if entry.NewPath != "" {
			path = entry.NewPath
		}
		treeEntries = append(treeEntries, struct{ Path, Digest string }{path, entry.Digest})
	}
	treeRaw, _ := json.Marshal(treeEntries)
	subject.TreeDigest = digestBytes(treeRaw)
	return subject, sortedBundleInputs(excluded), blockers, nil
}

func reduceReviewBundleChangeSets(sets []ChangeSet) []ChangeSet {
	result := make([]ChangeSet, 0, len(sets))
	for i, candidate := range sets {
		candidateCommits := map[string]bool{}
		for _, commit := range candidate.Commits {
			candidateCommits[commit] = true
		}
		subsumed := false
		if len(candidateCommits) > 0 {
			for j, other := range sets {
				if i == j || len(other.Commits) <= len(candidate.Commits) {
					continue
				}
				containsAll := true
				otherCommits := map[string]bool{}
				for _, commit := range other.Commits {
					otherCommits[commit] = true
				}
				for commit := range candidateCommits {
					if !otherCommits[commit] {
						containsAll = false
						break
					}
				}
				if containsAll {
					subsumed = true
					break
				}
			}
		}
		if !subsumed {
			result = append(result, candidate)
		}
	}
	return result
}

func (s Store) reviewBundleScopeSpecs(scope ScopeRef) (map[string]bool, error) {
	result := map[string]bool{}
	switch scope.Kind {
	case "spec":
		result[scope.Slug] = true
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				for _, slug := range milestone.Specs {
					result[slug] = true
				}
				return result, nil
			}
		}
		return nil, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			for _, slug := range milestone.Specs {
				result[slug] = true
			}
		}
	}
	return result, nil
}

func reviewBundleWorkingTreeChange(root, path string) (bool, string) {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--untracked-files=all", "--", path)
	out, err := cmd.Output()
	if err != nil {
		// Unit fixtures and exported source trees may not have Git metadata. The
		// immutable change-set gate remains authoritative there.
		return false, ""
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false, ""
	}
	if newline := strings.IndexByte(line, '\n'); newline >= 0 {
		line = line[:newline]
	}
	status := strings.TrimSpace(strings.TrimSuffix(line, path))
	if status == "" {
		return true, ""
	}
	return true, " (git status " + status + ")"
}

func reviewBundlePathClass(path string, scope ScopeRef, components []ReviewPlanComponent) (string, bool) {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || path == "" {
		return "", false
	}
	if scope.Kind == "spec" && (path == ".pose/specs/"+scope.Slug+"/spec.md" || path == ".pose/specs/"+scope.Slug+".md") {
		return "semantic-scope", false
	}
	if strings.HasPrefix(path, ".pose/specs/") {
		return "semantic-scope", false
	}
	for _, prefix := range []string{".pose/state/", ".pose/assessments/", ".pose/reports/", ".pose/results/", ".pose/reviews/", ".pose/review-bundles/", ".pose/review-attestations/"} {
		if strings.HasPrefix(path, prefix) {
			return "derived-evidence", false
		}
	}
	for _, exact := range []string{".pose/indexes/delivery-integrity.json", ".pose/indexes/releases.json", ".pose/indexes/spec-graph.json"} {
		if path == exact {
			return "derived-index", false
		}
	}
	for _, exact := range []string{".pose/indexes/validation-matrix.json", ".pose/indexes/module-metadata.json", ".pose/indexes/task-map.json"} {
		if path == exact {
			return "governance", true
		}
	}
	if strings.HasPrefix(path, ".pose/indexes/") {
		return "derived-index", false
	}
	for _, prefix := range []string{".pose/policy/", ".pose/releases/", ".pose/review-profiles/", ".pose/rules/", ".pose/workflows/", ".agents/skills/"} {
		if strings.HasPrefix(path, prefix) {
			return "governance", true
		}
	}
	for _, component := range components {
		root := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(component.Path)), "/")
		if root != "." && root != "" && (path == root || strings.HasPrefix(path, root+"/")) {
			return "implementation", true
		}
	}
	for _, prefix := range []string{"docs-site/docs/", "locales/"} {
		if strings.HasPrefix(path, prefix) {
			return "documentation", true
		}
	}
	for _, prefix := range []string{"pose-mcp/", "mcp-enforce/", "docs-site/", "locales/", "scripts/", "tests/", ".github/"} {
		if strings.HasPrefix(path, prefix) {
			return "implementation", true
		}
	}
	if path == "POSE.md" || path == "AGENTS.md" {
		return "documentation", true
	}
	if strings.HasPrefix(path, ".pose/adr/") || strings.HasPrefix(path, ".pose/knowledge/") || strings.HasPrefix(path, ".pose/changelogs/") {
		return "governance", true
	}
	return "", false
}

func (s Store) reviewBundleFileDigest(rel string) (string, error) {
	if err := ValidateArtifactPath(s.Root, rel, false); err != nil {
		return "", fmt.Errorf("review subject path %s is invalid: %w", rel, err)
	}
	clean, _ := validateArtifactPathSyntax(rel)
	raw, err := os.ReadFile(filepath.Join(s.Root, clean))
	if err != nil {
		return "", fmt.Errorf("review subject path %s cannot be read: %w", rel, err)
	}
	return digestBytes(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))), nil
}

func (s Store) reviewBundleEvidence(scope ScopeRef, graph DeliveryIntegrityGraph) []ReviewBundleEvidence {
	modules := map[string]bool{}
	for _, target := range graph.Deliveries {
		if scope.Kind == "spec" && target.Spec == scope.Slug {
			modules[target.Module] = true
		}
	}
	status := ""
	if scope.Kind == "spec" {
		if spec, err := s.GetSpec(scope.Slug); err == nil {
			status = spec.Status
		}
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if scope.Kind == "spec" && set.Spec == scope.Slug {
			sets = append(sets, set)
		}
	}
	result := []ReviewBundleEvidence{}
	for _, evidence := range graph.ValidationResults {
		if evidence.Outcome != "pass" || evidence.Severity != "required" || (len(modules) > 0 && !modules[evidence.Module]) {
			continue
		}
		if scope.Kind == "spec" && !deliveryEvidenceCurrent(evidence, scope.Slug, status, graph, sets) {
			continue
		}
		result = append(result, ReviewBundleEvidence{ID: evidence.ID, Module: evidence.Module, Check: evidence.Check, EvidenceClass: evidence.EvidenceClass, Outcome: evidence.Outcome, GitHead: evidence.GitHead, ProvenanceDigest: evidence.ProvenanceDigest, Report: evidence.Report})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s Store) reviewBundleChildRefs(scope ScopeRef) ([]string, error) {
	refs := []string{}
	if scope.Kind == "milestone" {
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				for _, slug := range milestone.Specs {
					refs = append(refs, "spec:"+slug)
				}
			}
		}
	} else if scope.Kind == "roadmap" {
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			refs = append(refs, "milestone:"+rm.Slug+"/"+milestone.ID)
		}
	}
	return refs, nil
}

func (s Store) reviewBundleChildren(scope ScopeRef) ([]ReviewBundleChild, []string, error) {
	refs, err := s.reviewBundleChildRefs(scope)
	if err != nil {
		return nil, nil, err
	}
	children, blockers := []ReviewBundleChild{}, []string{}
	for _, ref := range refs {
		sealed, err := s.CurrentReviewBundle(ref)
		if err != nil {
			return nil, nil, err
		}
		if sealed == nil {
			blockers = append(blockers, "child "+ref+" has no sealed review bundle")
			continue
		}
		children = append(children, ReviewBundleChild{Scope: ref, BundleID: sealed.BundleID, BundleDigest: sealed.BundleDigest})
	}
	return children, blockers, nil
}

func (s Store) reviewBundleConsumedInputs(plan ReviewPlan) []ReviewBundleInput {
	planDigest, _ := reviewBundlePlanDigest(plan)
	inputs := []ReviewBundleInput{
		{Kind: "review-plan", Path: plan.BaseProfile, Digest: planDigest, Reason: "effective component-aware plan contract"},
		{Kind: "tool-catalog", Path: "native-review-tools@1", Digest: planDigest, Reason: "closed resolved native tool contract"},
		{Kind: "schema", Path: "review-bundle@1", Digest: digestText("review-bundle-schema-v1\nreview-attestation-schema-v1\nreview-attestation-envelope-schema-v1"), Reason: "canonical bundle and attestation schema identity"},
	}
	for _, rel := range []string{".pose/policy/review.json", ".pose/indexes/validation-matrix.json"} {
		if raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "governed-config", Path: rel, Digest: digestBytes(raw), Reason: "consumed review policy or validation contract"})
		}
	}
	for _, profile := range plan.SelectedProfiles {
		raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(profile.Source)))
		if err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "review-profile", Path: profile.Source, Digest: digestBytes(raw), Reason: "selected by effective review plan"})
		}
	}
	rules := map[string]bool{}
	for _, criterion := range plan.Criteria {
		for _, rule := range criterion.Rules {
			rules[rule] = true
		}
	}
	for rule := range rules {
		rel := ".pose/rules/" + rule + ".md"
		if raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "rule", Path: rel, Digest: digestBytes(raw), Reason: "selected by effective review criterion"})
		}
	}
	return sortedBundleInputs(inputs)
}

func reviewBundlePlanDigest(plan ReviewPlan) (string, error) {
	contract := struct {
		SchemaVersion       int                   `json:"schema_version"`
		PolicySchemaVersion int                   `json:"policy_schema_version"`
		BaseProfile         string                `json:"base_profile"`
		Independence        string                `json:"independence"`
		Components          []ReviewPlanComponent `json:"components"`
		SelectedProfiles    []ReviewPlanProfile   `json:"selected_profiles"`
		Criteria            []ReviewPlanCriterion `json:"criteria"`
		Tools               []ReviewPlanTool      `json:"tools"`
	}{plan.SchemaVersion, plan.PolicySchemaVersion, plan.BaseProfile, plan.Independence, append([]ReviewPlanComponent{}, plan.Components...), append([]ReviewPlanProfile{}, plan.SelectedProfiles...), append([]ReviewPlanCriterion{}, plan.Criteria...), append([]ReviewPlanTool{}, plan.Tools...)}
	return digestJSON(contract)
}

func normalizeReviewBundlePayload(payload *ReviewBundlePayload) {
	sort.Slice(payload.Scope.Sections, func(i, j int) bool { return payload.Scope.Sections[i].Path < payload.Scope.Sections[j].Path })
	sort.Slice(payload.Plan.Components, func(i, j int) bool { return payload.Plan.Components[i].ID < payload.Plan.Components[j].ID })
	sort.Slice(payload.Plan.Criteria, func(i, j int) bool { return payload.Plan.Criteria[i].ID < payload.Plan.Criteria[j].ID })
	sort.Slice(payload.Plan.Tools, func(i, j int) bool {
		return reviewToolKey(payload.Plan.Tools[i].ID, payload.Plan.Tools[i].Component) < reviewToolKey(payload.Plan.Tools[j].ID, payload.Plan.Tools[j].Component)
	})
	payload.ConsumedInputs = sortedBundleInputs(payload.ConsumedInputs)
}

func reviewBundlePayloadDigest(payload ReviewBundlePayload) (string, error) {
	canonical := payload
	// Change-set IDs and Git refs remain exported for audit/debugging but are
	// advisory provenance. Stable subject identity is the classified patch and
	// tree manifest, so provider ref movement or a derived-only follow-up commit
	// cannot change the bundle identity.
	canonical.Subject.ChangeSets = nil
	canonical.Subject.Base = ""
	canonical.Subject.Head = ""
	return digestJSON(canonical)
}

func sortedBundleInputs(values []ReviewBundleInput) []ReviewBundleInput {
	result := append([]ReviewBundleInput{}, values...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Kind+"\x00"+result[i].Path+"\x00"+result[i].Digest < result[j].Kind+"\x00"+result[j].Path+"\x00"+result[j].Digest
	})
	return result
}

func normalizeBundleText(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func digestText(value string) string { return digestBytes([]byte(normalizeBundleText(value))) }
func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s Store) reviewBundlesDir() string { return filepath.Join(s.Root, ".pose", "review-bundles") }
func (s Store) reviewAttestationsDir() string {
	return filepath.Join(s.Root, ".pose", "review-attestations")
}

func ensureReviewArtifactDir(root, rel string, create bool) (string, error) {
	clean, err := validateArtifactPathSyntax(rel)
	if err != nil {
		return "", err
	}
	current := root
	for _, part := range strings.Split(filepath.Clean(clean), string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) && create {
			if mkdirErr := os.Mkdir(current, 0o755); mkdirErr != nil && !os.IsExist(mkdirErr) {
				return "", mkdirErr
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			relPath, _ := filepath.Rel(root, current)
			return "", fmt.Errorf("pose: refusing to follow review artifact symlink at %s", filepath.ToSlash(relPath))
		}
		if !info.IsDir() {
			return "", fmt.Errorf("pose: review artifact path is not a directory: %s", filepath.ToSlash(rel))
		}
	}
	return current, nil
}

// SealReviewBundle writes a prepared, unblocked bundle atomically. Replaying
// the same payload is idempotent.
func (s Store) SealReviewBundle(ref string, now time.Time) (ReviewBundle, error) {
	bundle, err := s.PrepareReviewBundle(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	if len(bundle.Blockers) > 0 {
		return bundle, fmt.Errorf("pose: review bundle is not sealable: %s", strings.Join(bundle.Blockers, "; "))
	}
	bundle.State = "sealed"
	bundle.SealedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	bundle.Path = filepath.ToSlash(filepath.Join(".pose", "review-bundles", bundle.BundleID+".json"))
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), true)
	if err != nil {
		return ReviewBundle{}, err
	}
	path := filepath.Join(dir, bundle.BundleID+".json")
	if existing, readErr := s.LoadReviewBundle(bundle.BundleID); readErr == nil {
		if existing.BundleDigest != bundle.BundleDigest {
			return ReviewBundle{}, fmt.Errorf("pose: review bundle identity collision for %s", bundle.BundleID)
		}
		return existing, nil
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return ReviewBundle{}, err
	}
	raw = append(raw, '\n')
	if err := writeImmutableJSON(path, raw); err != nil {
		return ReviewBundle{}, err
	}
	return bundle, nil
}

func writeImmutableJSON(path string, raw []byte) error {
	if len(raw) > maxReviewBundleBytes {
		return fmt.Errorf("pose: review artifact exceeds %d bytes", maxReviewBundleBytes)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pose-review-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("pose: immutable review artifact already exists: %s", filepath.Base(path))
		}
		return err
	}
	return os.Remove(tmpPath)
}

func strictJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) > maxReviewBundleBytes {
		return fmt.Errorf("review artifact exceeds %d bytes", maxReviewBundleBytes)
	}
	if err := rejectDuplicateJSONKeysAndControls(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeysAndControls(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var walk func() error
	checkText := func(value string) error {
		for _, r := range value {
			if r < 0x20 || r == 0x7f {
				return fmt.Errorf("JSON string contains a control character")
			}
		}
		return nil
	}
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{':
				seen := map[string]bool{}
				for decoder.More() {
					keyToken, err := decoder.Token()
					if err != nil {
						return err
					}
					key, ok := keyToken.(string)
					if !ok {
						return fmt.Errorf("JSON object key must be a string")
					}
					if err := checkText(key); err != nil {
						return err
					}
					if seen[key] {
						return fmt.Errorf("duplicate JSON field %q", key)
					}
					seen[key] = true
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := decoder.Token()
				if err != nil || end != json.Delim('}') {
					return fmt.Errorf("malformed JSON object")
				}
			case '[':
				for decoder.More() {
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := decoder.Token()
				if err != nil || end != json.Delim(']') {
					return fmt.Errorf("malformed JSON array")
				}
			default:
				return fmt.Errorf("unexpected JSON delimiter %q", value)
			}
		case string:
			return checkText(value)
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

func (s Store) LoadReviewBundle(id string) (ReviewBundle, error) {
	if !strings.HasPrefix(id, "rvb-") || len(id) != 20 {
		return ReviewBundle{}, fmt.Errorf("pose: invalid review bundle id %q", id)
	}
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), false)
	if err != nil {
		return ReviewBundle{}, err
	}
	path := filepath.Join(dir, id+".json")
	var bundle ReviewBundle
	if err := strictJSONFile(path, &bundle); err != nil {
		return ReviewBundle{}, fmt.Errorf("pose: reading review bundle %s: %w", id, err)
	}
	if bundle.SchemaVersion != ReviewBundleSchemaVersion || bundle.BundleID != id || bundle.State != "sealed" {
		return ReviewBundle{}, fmt.Errorf("pose: malformed review bundle %s", id)
	}
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil || digest != bundle.BundleDigest || id != "rvb-"+strings.TrimPrefix(digest, "sha256:")[:16] {
		return ReviewBundle{}, fmt.Errorf("pose: review bundle %s digest mismatch", id)
	}
	bundle.Path = filepath.ToSlash(filepath.Join(".pose", "review-bundles", id+".json"))
	return bundle, nil
}

func (s Store) ListReviewBundles(scope string) ([]ReviewBundle, error) {
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), false)
	if os.IsNotExist(err) {
		return []ReviewBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReviewBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	bundles := []ReviewBundle{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		bundle, loadErr := s.LoadReviewBundle(strings.TrimSuffix(entry.Name(), ".json"))
		if loadErr != nil {
			return nil, loadErr
		}
		if scope == "" || bundle.Payload.Scope.Ref == scope {
			bundles = append(bundles, bundle)
		}
	}
	sort.Slice(bundles, func(i, j int) bool {
		if bundles[i].SealedAt == bundles[j].SealedAt {
			return bundles[i].BundleID < bundles[j].BundleID
		}
		return bundles[i].SealedAt < bundles[j].SealedAt
	})
	return bundles, nil
}

func (s Store) CurrentReviewBundle(scope string) (*ReviewBundle, error) {
	prepared, err := s.PrepareReviewBundle(scope)
	if err != nil {
		return nil, err
	}
	bundles, err := s.ListReviewBundles(scope)
	if err != nil {
		return nil, err
	}
	for i := len(bundles) - 1; i >= 0; i-- {
		if bundles[i].BundleDigest == prepared.BundleDigest {
			bundle := bundles[i]
			return &bundle, nil
		}
	}
	return nil, nil
}

// AutoAttestReviewBundle constructs an attestation automatically from the
// bundle's validated evidence and plan dispositions.
func (s Store) AutoAttestReviewBundle(bundleID, reviewer string, apply bool, now time.Time) (ReviewAttestation, error) {
	bundle, err := s.LoadReviewBundle(bundleID)
	if err != nil {
		return ReviewAttestation{}, err
	}
	if reviewer == "" {
		reviewer = "agent:auto-attest"
	}
	if len(bundle.Payload.Evidence) == 0 {
		return ReviewAttestation{}, fmt.Errorf("pose: bundle %s has no passed structured validation evidence", bundleID)
	}
	evidenceRefs := make([]string, 0, len(bundle.Payload.Evidence))
	byClass := map[string][]string{}
	for _, ev := range bundle.Payload.Evidence {
		ref := ev.EvidenceClass + ":" + ev.ID
		evidenceRefs = append(evidenceRefs, ref)
		byClass[ev.EvidenceClass] = append(byClass[ev.EvidenceClass], ref)
	}
	sort.Strings(evidenceRefs)

	criteria := make([]ReviewCriterion, 0, len(bundle.Payload.Plan.Criteria))
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if !criterion.Required {
			continue
		}
		critEvidence := evidenceRefs[0]
		for _, class := range criterion.EvidenceClasses {
			if refs, ok := byClass[class]; ok && len(refs) > 0 {
				critEvidence = refs[0]
				break
			}
		}
		criteria = append(criteria, ReviewCriterion{
			ID:          criterion.ID,
			Disposition: "passed",
			Evidence:    critEvidence,
		})
	}

	tools := make([]ReviewToolDisposition, 0, len(bundle.Payload.Plan.Tools))
	for _, tool := range bundle.Payload.Plan.Tools {
		disposition := ReviewToolDisposition{
			ID:        tool.ID,
			Component: tool.Component,
		}
		if containsFold(tool.Preconditions, "review-complete") {
			disposition.Disposition = "deferred"
			disposition.Rationale = "post-review gate"
		} else if tool.Requiredness == "recommended" {
			disposition.Disposition = "not-used"
			disposition.Rationale = "not used during automated attestation"
		} else {
			toolEv := ""
			for _, class := range tool.EvidenceClasses {
				if refs, ok := byClass[class]; ok && len(refs) > 0 {
					toolEv = refs[0]
					break
				}
				if class == "validation" {
					toolEv = "validation:auto-attest"
					break
				}
			}
			if toolEv == "" {
				if len(tool.EvidenceClasses) > 0 {
					toolEv = tool.EvidenceClasses[0] + ":auto-attest"
				} else {
					toolEv = evidenceRefs[0]
				}
			}
			disposition.Disposition = "passed"
			disposition.Evidence = toolEv
		}
		tools = append(tools, disposition)
	}

	att := ReviewAttestation{
		BundleID:     bundle.BundleID,
		BundleDigest: bundle.BundleDigest,
		Reviewer:     reviewer,
		Decision:     "approved",
		Criteria:     criteria,
		Tools:        tools,
		EvidenceRefs: evidenceRefs,
		Findings:     []ReviewFinding{},
	}

	if !apply {
		return att, nil
	}
	return s.RecordReviewAttestation(att, now)
}

// RecordReviewAttestation appends an immutable decision for one sealed bundle.
func (s Store) RecordReviewAttestation(att ReviewAttestation, now time.Time) (ReviewAttestation, error) {
	return s.recordReviewAttestation(att, now, false)
}

func (s Store) recordReviewAttestation(att ReviewAttestation, now time.Time, signed bool) (ReviewAttestation, error) {
	bundle, err := s.LoadReviewBundle(att.BundleID)
	if err != nil {
		return ReviewAttestation{}, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewAttestation{}, err
	}
	if policy.RequireSignedAttestations && !signed {
		return ReviewAttestation{}, fmt.Errorf("pose: review policy requires a trusted signed attestation envelope")
	}
	if att.BundleDigest == "" {
		att.BundleDigest = bundle.BundleDigest
	}
	if att.BundleDigest != bundle.BundleDigest {
		return ReviewAttestation{}, fmt.Errorf("pose: attestation bundle digest mismatch")
	}
	if att.Reviewer == "" || strings.ContainsAny(att.Reviewer, "\r\n") || (!strings.HasPrefix(att.Reviewer, "agent:") && !strings.HasPrefix(att.Reviewer, "human:")) {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid reviewer execution identity")
	}
	if att.Decision != "approved" && att.Decision != "approved-with-reservations" && att.Decision != "changes-requested" && att.Decision != "rejected" {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid review decision %q", att.Decision)
	}
	att.SchemaVersion = ReviewBundleSchemaVersion
	if att.AttestedAt == "" {
		att.AttestedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, att.AttestedAt); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: attested_at must be RFC3339")
	}
	att.BundleID = bundle.BundleID
	att.BundleDigest = bundle.BundleDigest
	expectedAttestationID := reviewAttestationID(att)
	if att.AttestationID == "" {
		att.AttestationID = expectedAttestationID
	} else if att.AttestationID != expectedAttestationID {
		return ReviewAttestation{}, fmt.Errorf("pose: attestation id does not match its content digest")
	}
	if !strings.HasPrefix(att.AttestationID, "rva-") || len(att.AttestationID) != 20 {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid attestation id")
	}
	att.Path = filepath.ToSlash(filepath.Join(".pose", "review-attestations", att.AttestationID+".json"))
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), true)
	if err != nil {
		return ReviewAttestation{}, err
	}
	path := filepath.Join(dir, att.AttestationID+".json")
	if existing, loadErr := s.LoadReviewAttestation(att.AttestationID); loadErr == nil {
		existingRaw, _ := json.Marshal(existing)
		candidateRaw, _ := json.Marshal(att)
		if bytes.Equal(existingRaw, candidateRaw) {
			return existing, nil
		}
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation identity collision for %s", att.AttestationID)
	}
	raw, err := json.MarshalIndent(att, "", "  ")
	if err != nil {
		return ReviewAttestation{}, err
	}
	if err := writeImmutableJSON(path, append(raw, '\n')); err != nil {
		return ReviewAttestation{}, err
	}
	return att, nil
}

// VerifyReviewAttestationEnvelope verifies an optional provider-neutral
// Ed25519 envelope. Trust is pinned as <issuer>#sha256:<public-key-digest> in
// review policy; a self-declared issuer or public key is never trusted alone.
func (s Store) VerifyReviewAttestationEnvelope(envelope ReviewAttestationEnvelope) (ReviewAttestation, error) {
	if envelope.SchemaVersion != ReviewBundleSchemaVersion || envelope.Algorithm != "ed25519" {
		return ReviewAttestation{}, fmt.Errorf("pose: unsupported review attestation envelope")
	}
	if envelope.Subject != envelope.Attestation.BundleID || envelope.Issuer == "" || strings.ContainsAny(envelope.Issuer, "\r\n") {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation envelope subject or issuer mismatch")
	}
	publicKey, err := base64.StdEncoding.DecodeString(envelope.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid Ed25519 public key")
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid Ed25519 signature")
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewAttestation{}, err
	}
	pin := envelope.Issuer + "#" + digestBytes(publicKey)
	trusted := false
	for _, candidate := range policy.TrustedAttestationIssuers {
		if candidate == pin {
			trusted = true
			break
		}
	}
	if !trusted {
		return ReviewAttestation{}, fmt.Errorf("pose: untrusted review attestation issuer or public key")
	}
	attestation := envelope.Attestation
	attestation.Path = ""
	attestation.Envelope = nil
	raw, err := json.Marshal(attestation)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), raw, signature) {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation signature verification failed")
	}
	bundle, err := s.LoadReviewBundle(attestation.BundleID)
	if err != nil {
		return ReviewAttestation{}, err
	}
	if attestation.BundleDigest != bundle.BundleDigest {
		return ReviewAttestation{}, fmt.Errorf("pose: signed attestation does not bind the exact bundle")
	}
	attestation.Envelope = &ReviewAttestationSignature{Issuer: envelope.Issuer, Subject: envelope.Subject, Algorithm: envelope.Algorithm, PublicKey: envelope.PublicKey, Signature: envelope.Signature}
	return attestation, nil
}

func (s Store) ImportReviewAttestationEnvelope(rel string, apply bool) (ReviewAttestation, error) {
	if err := ValidateArtifactPath(s.Root, rel, false); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid attestation envelope path: %w", err)
	}
	clean, _ := validateArtifactPathSyntax(rel)
	var envelope ReviewAttestationEnvelope
	if err := strictJSONFile(filepath.Join(s.Root, clean), &envelope); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: reading review attestation envelope: %w", err)
	}
	attestation, err := s.VerifyReviewAttestationEnvelope(envelope)
	if err != nil || !apply {
		return attestation, err
	}
	return s.recordReviewAttestation(attestation, time.Now(), true)
}

func (s Store) LoadReviewAttestation(id string) (ReviewAttestation, error) {
	if !strings.HasPrefix(id, "rva-") || len(id) != 20 {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid review attestation id %q", id)
	}
	var att ReviewAttestation
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), false)
	if err != nil {
		return ReviewAttestation{}, err
	}
	path := filepath.Join(dir, id+".json")
	if err := strictJSONFile(path, &att); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: reading review attestation %s: %w", id, err)
	}
	if att.SchemaVersion != ReviewBundleSchemaVersion || att.AttestationID != id {
		return ReviewAttestation{}, fmt.Errorf("pose: malformed review attestation %s", id)
	}
	if reviewAttestationID(att) != id {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation %s content digest mismatch", id)
	}
	att.Path = filepath.ToSlash(filepath.Join(".pose", "review-attestations", id+".json"))
	return att, nil
}

func (s Store) ListReviewAttestations(bundleID string) ([]ReviewAttestation, error) {
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), false)
	if os.IsNotExist(err) {
		return []ReviewAttestation{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReviewAttestation{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []ReviewAttestation{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		att, loadErr := s.LoadReviewAttestation(strings.TrimSuffix(entry.Name(), ".json"))
		if loadErr != nil {
			return nil, loadErr
		}
		if bundleID == "" || att.BundleID == bundleID {
			result = append(result, att)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].AttestedAt == result[j].AttestedAt {
			return result[i].AttestationID < result[j].AttestationID
		}
		return result[i].AttestedAt < result[j].AttestedAt
	})
	return result, nil
}

func (s Store) VerifyReviewBundle(scope string) (ReviewBundleVerification, error) {
	prepared, err := s.PrepareReviewBundle(scope)
	if err != nil {
		return ReviewBundleVerification{}, err
	}
	verification := ReviewBundleVerification{Scope: scope, State: "needs-validation", Fresh: false, Approved: false, Warnings: append([]string{}, prepared.Warnings...), Blockers: append([]string{}, prepared.Blockers...)}
	bundles, err := s.ListReviewBundles(scope)
	if err != nil {
		return verification, err
	}
	if len(prepared.Blockers) == 0 {
		verification.State = "ready-to-seal"
	}
	var current *ReviewBundle
	for i := len(bundles) - 1; i >= 0; i-- {
		if bundles[i].BundleDigest == prepared.BundleDigest {
			bundle := bundles[i]
			current = &bundle
			break
		}
	}
	if current == nil {
		if len(bundles) > 0 {
			previous := bundles[len(bundles)-1]
			delta := ReviewBundleDiff(previous, prepared)
			if attempts, listErr := s.ListReviewAttestations(previous.BundleID); listErr == nil && len(attempts) > 0 {
				for _, finding := range attempts[len(attempts)-1].Findings {
					delta.ChangedFindings = append(delta.ChangedFindings, finding.ID)
				}
				delta.ChangedFindings = uniqueSorted(delta.ChangedFindings)
			}
			verification.Delta = &delta
			verification.State = "superseded"
			verification.Blockers = append(verification.Blockers, "current semantic inputs are not represented by a sealed bundle")
		}
		verification.NextAction = "seal the current review bundle for " + scope
		verification.Blockers = uniqueSorted(verification.Blockers)
		return verification, nil
	}
	verification.Bundle = current
	verification.Fresh = true
	verification.State = "ready-for-review"
	attestations, err := s.ListReviewAttestations(current.BundleID)
	if err != nil {
		return verification, err
	}
	if len(attestations) == 0 {
		verification.Blockers = append(verification.Blockers, "sealed review bundle has no attestation")
		verification.NextAction = "review and attest bundle " + current.BundleID
		verification.Blockers = uniqueSorted(verification.Blockers)
		return verification, nil
	}
	att := attestations[len(attestations)-1]
	verification.Attestation = &att
	verification.Blockers = append(verification.Blockers, s.validateBundleAttestation(*current, att)...)
	if len(verification.Blockers) == 0 {
		verification.Approved = true
		scopeRef, scopeErr := ParseScopeRef(scope)
		done, doneErr := false, scopeErr
		if scopeErr == nil {
			done, doneErr = s.scopeLifecycleDone(scopeRef)
		}
		if doneErr == nil && done {
			verification.State = "closed"
			verification.NextAction = "scope is closed with a fresh bundle attestation"
		} else {
			verification.State = "ready-to-close"
			verification.NextAction = "apply the guarded lifecycle transition for " + scope
		}
	} else if att.Decision == "changes-requested" || att.Decision == "rejected" {
		verification.State = "changes-requested"
		verification.NextAction = "remediate findings and seal a superseding bundle"
	} else {
		verification.NextAction = "replace the invalid attestation for " + current.BundleID
	}
	verification.Blockers = uniqueSorted(verification.Blockers)
	return verification, nil
}

func (s Store) validateBundleAttestation(bundle ReviewBundle, att ReviewAttestation) []string {
	blockers := []string{}
	policy, _, policyErr := s.loadReviewPolicy()
	if policyErr != nil {
		blockers = append(blockers, policyErr.Error())
	}
	if att.BundleDigest != bundle.BundleDigest || att.BundleID != bundle.BundleID {
		blockers = append(blockers, "attestation does not reference the exact sealed bundle")
	}
	if policy.RequireSignedAttestations || att.Envelope != nil {
		if err := s.verifyStoredReviewAttestationSignature(att); err != nil {
			blockers = append(blockers, err.Error())
		}
	}
	if _, err := time.Parse(time.RFC3339, att.AttestedAt); err != nil {
		blockers = append(blockers, "attested_at must be RFC3339")
	}
	if !strings.HasPrefix(att.Reviewer, "agent:") && !strings.HasPrefix(att.Reviewer, "human:") {
		blockers = append(blockers, "reviewer execution identity is malformed")
	}
	switch bundle.Payload.Plan.Independence {
	case "different-actor":
		if !strings.HasPrefix(att.Reviewer, "agent:independent-") && !strings.HasPrefix(att.Reviewer, "human:") {
			blockers = append(blockers, "review policy requires an independent reviewer identity")
		}
	case "mandatory-human":
		if !strings.HasPrefix(att.Reviewer, "human:") {
			blockers = append(blockers, "review policy requires human approval")
		}
	}
	required := map[string]ReviewPlanCriterion{}
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if criterion.Required {
			required[criterion.ID] = criterion
		}
	}
	seen := map[string]bool{}
	for _, criterion := range att.Criteria {
		if seen[criterion.ID] {
			blockers = append(blockers, "duplicate criterion "+criterion.ID)
		}
		seen[criterion.ID] = true
		if _, ok := required[criterion.ID]; !ok {
			blockers = append(blockers, "unknown criterion "+criterion.ID)
		}
		if criterion.Disposition != "passed" && criterion.Disposition != "not-applicable" && criterion.Disposition != "finding" {
			blockers = append(blockers, "criterion "+criterion.ID+" has invalid disposition")
		}
		if criterion.Disposition == "not-applicable" && criterion.Rationale == "" {
			blockers = append(blockers, "criterion "+criterion.ID+" lacks not-applicable rationale")
		}
	}
	for id := range required {
		if !seen[id] {
			blockers = append(blockers, "missing criterion "+id)
		}
	}
	reused := map[string]bool{}
	for _, reuse := range att.ReusedFrom {
		if reused[reuse.Criterion] {
			blockers = append(blockers, "duplicate reused criterion "+reuse.Criterion)
			continue
		}
		reused[reuse.Criterion] = true
		if !policy.AllowCriterionReuse {
			blockers = append(blockers, "criterion reuse is disabled by review policy")
			continue
		}
		current, ok := required[reuse.Criterion]
		if !ok {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" is not required by the current bundle")
			continue
		}
		currentDigest := reviewCriterionInputDigest(bundle, current)
		if currentDigest != reuse.InputDigest {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" input digest changed")
			continue
		}
		prior, err := s.LoadReviewAttestation(reuse.FromAttestation)
		if err != nil {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" references an unavailable attestation")
			continue
		}
		priorBundle, err := s.LoadReviewBundle(prior.BundleID)
		if err != nil {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" references an unavailable bundle")
			continue
		}
		priorContract := ""
		for _, criterion := range priorBundle.Payload.Plan.Criteria {
			if criterion.ID == reuse.Criterion {
				priorContract = reviewCriterionInputDigest(priorBundle, criterion)
				break
			}
		}
		priorPassed := false
		for _, criterion := range prior.Criteria {
			if criterion.ID == reuse.Criterion && criterion.Disposition == "passed" {
				priorPassed = true
				break
			}
		}
		if priorContract != reuse.InputDigest || !priorPassed {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" is not unchanged and passed in the referenced attestation")
		}
	}
	toolWarnings, toolBlockers := evaluateReviewToolCoverage(s.Root, bundle.Payload.Plan.Tools, att.Tools)
	_ = toolWarnings
	blockers = append(blockers, toolBlockers...)
	for _, finding := range att.Findings {
		if finding.Disposition == "open" || finding.Disposition == "changes-requested" {
			blockers = append(blockers, "finding "+finding.ID+" is "+finding.Disposition)
		}
	}
	if att.Decision != "approved" {
		blockers = append(blockers, "review decision does not permit closeout: "+att.Decision)
	}
	return blockers
}

func (s Store) verifyStoredReviewAttestationSignature(att ReviewAttestation) error {
	if att.Envelope == nil {
		return fmt.Errorf("signed attestation proof is required by review policy")
	}
	proof := *att.Envelope
	unsigned := att
	unsigned.Path = ""
	unsigned.Envelope = nil
	_, err := s.VerifyReviewAttestationEnvelope(ReviewAttestationEnvelope{SchemaVersion: ReviewBundleSchemaVersion, Issuer: proof.Issuer, Subject: proof.Subject, Algorithm: proof.Algorithm, PublicKey: proof.PublicKey, Signature: proof.Signature, Attestation: unsigned})
	return err
}

func reviewCriterionInputDigest(bundle ReviewBundle, criterion ReviewPlanCriterion) string {
	evidence := []ReviewBundleEvidence{}
	classes := map[string]bool{}
	for _, class := range criterion.EvidenceClasses {
		classes[class] = true
	}
	for _, item := range bundle.Payload.Evidence {
		if classes[item.EvidenceClass] {
			evidence = append(evidence, item)
		}
	}
	inputs := []ReviewBundleInput{}
	for _, input := range bundle.Payload.ConsumedInputs {
		if input.Kind == "schema" || input.Kind == "governed-config" {
			inputs = append(inputs, input)
			continue
		}
		if (input.Kind == "review-plan" || input.Kind == "tool-catalog") && reviewCriterionSubjectSensitive(criterion) {
			inputs = append(inputs, input)
			continue
		}
		if input.Kind == "rule" {
			for _, rule := range criterion.Rules {
				if input.Path == ".pose/rules/"+rule+".md" {
					inputs = append(inputs, input)
				}
			}
		}
		if input.Kind == "review-profile" {
			for _, profile := range criterion.Profiles {
				if input.Path == ".pose/review-profiles/"+strings.Split(profile, "@")[0]+".json" {
					inputs = append(inputs, input)
				}
			}
		}
	}
	subjectSensitive := reviewCriterionSubjectSensitive(criterion)
	relevantTools := []ReviewPlanTool{}
	for _, tool := range bundle.Payload.Plan.Tools {
		for _, criterionID := range tool.Criteria {
			if criterionID == criterion.ID {
				relevantTools = append(relevantTools, tool)
				break
			}
		}
	}
	contract := struct {
		Criterion    ReviewPlanCriterion        `json:"criterion"`
		Independence string                     `json:"independence"`
		Scope        []ReviewBundleInput        `json:"scope,omitempty"`
		Subject      []ReviewBundleSubjectEntry `json:"subject,omitempty"`
		PatchDigest  string                     `json:"patch_digest,omitempty"`
		TreeDigest   string                     `json:"tree_digest,omitempty"`
		Evidence     []ReviewBundleEvidence     `json:"evidence,omitempty"`
		Inputs       []ReviewBundleInput        `json:"inputs"`
		Tools        []ReviewPlanTool           `json:"tools,omitempty"`
	}{Criterion: criterion, Independence: bundle.Payload.Plan.Independence, Evidence: evidence, Inputs: sortedBundleInputs(inputs), Tools: relevantTools}
	if subjectSensitive {
		contract.Scope = append([]ReviewBundleInput{}, bundle.Payload.Scope.Sections...)
		contract.Subject = append([]ReviewBundleSubjectEntry{}, bundle.Payload.Subject.Entries...)
		contract.PatchDigest = bundle.Payload.Subject.PatchDigest
		contract.TreeDigest = bundle.Payload.Subject.TreeDigest
	} else {
		contract.Scope = append([]ReviewBundleInput{}, bundle.Payload.Scope.Sections...)
		for _, entry := range bundle.Payload.Subject.Entries {
			if entry.Class == "documentation" || entry.Class == "governance" {
				contract.Subject = append(contract.Subject, entry)
			}
		}
	}
	digest, _ := digestJSON(contract)
	return digest
}

func reviewAttestationID(att ReviewAttestation) string {
	identity := att
	identity.AttestationID = ""
	identity.Path = ""
	identity.Envelope = nil
	digest, _ := digestJSON(identity)
	return "rva-" + strings.TrimPrefix(digest, "sha256:")[:16]
}

func reviewCriterionSubjectSensitive(criterion ReviewPlanCriterion) bool {
	if len(criterion.EvidenceClasses) > 0 || len(criterion.Rules) == 0 {
		return true
	}
	for _, rule := range criterion.Rules {
		if rule != "documentation-style" && rule != "knowledge-governance" {
			return true
		}
	}
	return false
}

func ReviewBundleDiff(from, to ReviewBundle) ReviewBundleDelta {
	delta := ReviewBundleDelta{FromBundle: from.BundleID, ToBundle: to.BundleID}
	fromComponents := map[string]string{}
	toComponents := map[string]string{}
	for _, component := range from.Payload.Plan.Components {
		raw, _ := json.Marshal(component)
		fromComponents[component.ID] = digestBytes(raw)
	}
	for _, component := range to.Payload.Plan.Components {
		raw, _ := json.Marshal(component)
		toComponents[component.ID] = digestBytes(raw)
	}
	delta.ChangedComponents = changedReviewBundleKeys(fromComponents, toComponents)
	fromSections := map[string]string{}
	toSections := map[string]string{}
	for _, input := range from.Payload.Scope.Sections {
		fromSections[input.Path] = input.Digest
	}
	for _, input := range to.Payload.Scope.Sections {
		toSections[input.Path] = input.Digest
	}
	delta.ChangedSections = changedReviewBundleKeys(fromSections, toSections)
	fromPaths := map[string]string{}
	toPaths := map[string]string{}
	pathNames := map[string]string{}
	for _, entry := range from.Payload.Subject.Entries {
		key := entry.Action + "\x00" + entry.Path + "\x00" + entry.NewPath
		fromPaths[key] = entry.Digest
		pathNames[key] = reviewBundleEntryPath(entry)
	}
	for _, entry := range to.Payload.Subject.Entries {
		key := entry.Action + "\x00" + entry.Path + "\x00" + entry.NewPath
		toPaths[key] = entry.Digest
		pathNames[key] = reviewBundleEntryPath(entry)
	}
	for _, key := range changedReviewBundleKeys(fromPaths, toPaths) {
		delta.ChangedPaths = append(delta.ChangedPaths, pathNames[key])
	}
	fromCriteria := map[string]string{}
	toCriteria := map[string]string{}
	for _, criterion := range from.Payload.Plan.Criteria {
		fromCriteria[criterion.ID] = reviewCriterionInputDigest(from, criterion)
	}
	for _, criterion := range to.Payload.Plan.Criteria {
		toCriteria[criterion.ID] = reviewCriterionInputDigest(to, criterion)
		if fromCriteria[criterion.ID] == toCriteria[criterion.ID] {
			delta.ReusableCriteria = append(delta.ReusableCriteria, criterion.ID)
		}
	}
	delta.ChangedCriteria = changedReviewBundleKeys(fromCriteria, toCriteria)
	fromEvidence := map[string]string{}
	toEvidence := map[string]string{}
	evidenceClasses := map[string][]string{}
	for _, evidence := range from.Payload.Evidence {
		raw, _ := json.Marshal(evidence)
		fromEvidence[evidence.ID] = digestBytes(raw)
		evidenceClasses[evidence.ID] = append(evidenceClasses[evidence.ID], evidence.EvidenceClass)
	}
	for _, evidence := range to.Payload.Evidence {
		raw, _ := json.Marshal(evidence)
		toEvidence[evidence.ID] = digestBytes(raw)
		evidenceClasses[evidence.ID] = append(evidenceClasses[evidence.ID], evidence.EvidenceClass)
	}
	delta.ChangedEvidence = changedReviewBundleKeys(fromEvidence, toEvidence)
	for _, id := range delta.ChangedEvidence {
		delta.ChangedEvidenceClasses = append(delta.ChangedEvidenceClasses, evidenceClasses[id]...)
	}
	delta.ChangedComponents = uniqueSorted(delta.ChangedComponents)
	delta.ChangedSections = uniqueSorted(delta.ChangedSections)
	delta.ChangedPaths = uniqueSorted(delta.ChangedPaths)
	delta.ChangedCriteria = uniqueSorted(delta.ChangedCriteria)
	delta.ChangedEvidence = uniqueSorted(delta.ChangedEvidence)
	delta.ChangedEvidenceClasses = uniqueSorted(delta.ChangedEvidenceClasses)
	delta.ReusableCriteria = uniqueSorted(delta.ReusableCriteria)
	return delta
}

func changedReviewBundleKeys(from, to map[string]string) []string {
	keys := map[string]bool{}
	for key, value := range from {
		if to[key] != value {
			keys[key] = true
		}
	}
	for key, value := range to {
		if from[key] != value {
			keys[key] = true
		}
	}
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	return uniqueSorted(result)
}

func reviewBundleEntryPath(entry ReviewBundleSubjectEntry) string {
	if entry.NewPath != "" {
		return entry.NewPath
	}
	return entry.Path
}
