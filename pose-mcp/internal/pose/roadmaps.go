package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Roadmap is the parsed view of a governed roadmap artifact
// (pose-roadmap-artifact): flat frontmatter + `## Milestone: <id>` body
// sections with flat bullets. Keep the parser in sync with
// the native CLI indexer.
type Roadmap struct {
	Slug      string   `json:"slug"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Consumes  []string `json:"consumes,omitempty"`
	// UnreviewedMilestones is the reason a roadmap gives for its reached milestones
	// standing without a review (ADR a-milestone-of-one-member-is-that-member).
	UnreviewedMilestones string `json:"-"`
	// ADRs is the raw `adrs:` declaration, `[adr:<file>.md, …]`, read by the
	// closeout review evaluator (spec pose-closeout-review-policy-by-cost-benefit).
	// Raw so that a malformed declaration is unknown, not absent.
	ADRs       string      `json:"-"`
	Milestones []Milestone `json:"milestones"`
	Path       string      `json:"path"`
	Body       string      `json:"body,omitempty"`
}

type Milestone struct {
	ID          string   `json:"id"`
	After       []string `json:"after,omitempty"`
	Consumes    []string `json:"consumes,omitempty"`
	TargetStart string   `json:"target_start,omitempty"`
	TargetDue   string   `json:"target_due,omitempty"`
	Specs       []string `json:"specs"`
	// ExitGate is the prose of the milestone's exit gate.
	ExitGate string `json:"-"`
}

var milestoneHeadingRE = regexp.MustCompile(`^##\s+Milestone:\s*(.+?)\s*$`)

// ReachedMilestones lists the scopes of the milestones of roadmaps not yet done
// whose members are all terminal and which hold no approval: the milestones
// `pose close` can be asked for (spec pose-milestone-closes-when-reached). A spec
// keeps the ones it is a member of. It runs the review of each one, so it is for a
// single close; a whole-repository read uses milestonesWithRecordedApproval.
func (s Store) ReachedMilestones(spec string) ([]string, error) {
	candidates, err := s.milestonesWithTerminalMembers(spec, "")
	if err != nil {
		return nil, err
	}
	reached := []string{}
	if adopted, err := s.closeoutReviewPolicyAdopted(); err != nil {
		return reached, err
	} else if adopted {
		// Under an adopted closeout review policy, only the explicit close
		// makes a milestone terminal: reached means no receipt yet.
		for _, ref := range candidates {
			receipt, err := s.ReadCloseoutReceipt(ref)
			if err != nil {
				return reached, err
			}
			if receipt == nil {
				reached = append(reached, ref)
			}
		}
		return reached, nil
	}
	for _, ref := range candidates {
		// Its only member has just closed, uncommitted, so its verification cannot
		// answer yet: only a refusal recorded on the milestone says it will not stand.
		if parsed, err := ParseScopeRef(ref); err == nil {
			if member, inherits := s.soleMemberOf(parsed); inherits && member == spec && !s.milestoneReviewRefused(ref) {
				continue
			}
		}
		approved, err := s.scopeApproved(ref)
		if err != nil {
			return reached, err
		}
		if !approved {
			reached = append(reached, ref)
		}
	}
	return reached, nil
}

// milestonesWithTerminalMembers lists the milestones of roadmaps not yet done whose
// members are all terminal. A spec keeps the ones it is a member of; a scope, a
// milestone or a roadmap, keeps the ones inside it.
func (s Store) milestonesWithTerminalMembers(spec, scope string) ([]string, error) {
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return nil, err
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	statuses := map[string]string{}
	for _, sp := range specs {
		statuses[sp.Slug] = sp.Status
	}
	found := []string{}
	for _, rm := range roadmaps {
		if rm.Status == "done" {
			continue
		}
		for _, milestone := range rm.Milestones {
			ref := "milestone:" + rm.Slug + "/" + milestone.ID
			if scope != "" && scope != ref && scope != "roadmap:"+rm.Slug {
				continue
			}
			if s.milestoneMembersTerminal(milestone, statuses, spec) {
				found = append(found, ref)
			}
		}
	}
	return found, nil
}

// milestoneMembersTerminal also reports that a milestone has spec as a member,
// when one is named.
func (s Store) milestoneMembersTerminal(milestone Milestone, statuses map[string]string, spec string) bool {
	member := spec == ""
	for _, raw := range milestone.Specs {
		slug, external := s.milestoneMember(raw)
		if external && !s.externalMemberDone(raw) {
			return false
		}
		if !external {
			slug = strings.TrimPrefix(slug, "spec:")
			if !terminalStatuses[statuses[slug]] {
				return false
			}
			member = member || slug == spec
		}
	}
	return member && len(milestone.Specs) > 0
}

// soleMemberOf names the spec a milestone is made of, when it is made of one local spec.
func (s Store) soleMemberOf(scope ScopeRef) (string, bool) {
	if scope.Kind != "milestone" {
		return "", false
	}
	rm, err := s.GetRoadmap(scope.Roadmap)
	if err != nil {
		return "", false
	}
	for _, milestone := range rm.Milestones {
		if milestone.ID != scope.Milestone || len(milestone.Specs) != 1 {
			continue
		}
		if slug, external := s.milestoneMember(milestone.Specs[0]); !external {
			return strings.TrimPrefix(slug, "spec:"), true
		}
	}
	return "", false
}

// declaredUnreviewed returns the reason a roadmap gives for a milestone standing with no
// review: it applies to a milestone whose members are all terminal.
func (s Store) declaredUnreviewed(scope ScopeRef) string {
	if scope.Kind != "milestone" {
		return ""
	}
	rm, err := s.GetRoadmap(scope.Roadmap)
	if err != nil || rm.UnreviewedMilestones == "" {
		return ""
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return ""
	}
	statuses := map[string]string{}
	for _, sp := range specs {
		statuses[sp.Slug] = sp.Status
	}
	for _, milestone := range rm.Milestones {
		if milestone.ID == scope.Milestone && s.milestoneMembersTerminal(milestone, statuses, "") {
			return rm.UnreviewedMilestones
		}
	}
	return ""
}

func (s Store) scopeApproved(ref string) (bool, error) {
	eval, err := s.ReviewCheck(ref)
	if err != nil {
		return false, err
	}
	return eval.Approved || !eval.Required, nil
}

// milestonesWithRecordedApproval says which of refs stand approved on file: the
// latest bundle of the scope, whose latest attestation approves it. It goes through
// the canonical readers and verifies nothing, since `pose state --attention` asks this
// about every candidate of the repository, where a review each costs minutes: an
// approval gone stale is for the milestone's own close to refuse. A record that cannot
// be read is an error, never an absence.
func (s Store) milestonesWithRecordedApproval(refs []string) (recorded, refused map[string]bool, err error) {
	recorded, refused = map[string]bool{}, map[string]bool{}
	if len(refs) == 0 {
		return recorded, refused, nil
	}
	attestations, err := s.ReadReviewAttestations("")
	if err != nil {
		return nil, nil, err
	}
	latest, err := s.latestBundlesOf(refs)
	if err != nil {
		return nil, nil, err
	}
	attempts, err := s.ReadReviewAttempts("")
	if err != nil {
		return nil, nil, err
	}
	decided := map[string]ReviewAttestation{}
	for _, att := range attestations.Attestations {
		if current, seen := decided[att.BundleID]; !seen || attestationLess(current, att) {
			decided[att.BundleID] = att
		}
	}
	for ref, bundles := range latest {
		approved := true
		for _, bundle := range bundles {
			if !attestations.Coverage.CompleteFor(bundle) {
				return nil, nil, fmt.Errorf("pose: the attestations of %s could not all be read", ref)
			}
			decision := decided[bundle].Decision
			approved = approved && reviewDecisionApproves(decision)
			refused[ref] = refused[ref] || decision == "rejected" || decision == "changes-requested"
		}
		recorded[ref] = approved
	}
	// The same veto as milestoneReviewRefused, from one read of the attempts.
	latestAttempt := map[string]string{}
	for _, attempt := range attempts.Attempts {
		latestAttempt[attempt.Scope] = attempt.Decision
	}
	for _, ref := range refs {
		decision := latestAttempt[ref]
		refused[ref] = refused[ref] || decision == "rejected" || decision == "changes-requested" || !attempts.Coverage.CompleteFor(ref)
	}
	return recorded, refused, nil
}

// latestBundlesOf names the bundles sealed last for each of refs, more than one when
// they share the instant: seal times have a resolution of a second, so all of them
// have to approve, as attestationLess reads a tie. It takes scope and seal time from
// each record without materializing it, which would take most of a minute here. A
// record that does not settle them may be any scope's, so it is an error.
func (s Store) latestBundlesOf(refs []string) (map[string][]string, error) {
	wanted := map[string]bool{}
	for _, ref := range refs {
		wanted[ref] = true
	}
	dir := s.reviewBundlesDir()
	entries, err := s.fsys().ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string][]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	latest, sealed := map[string][]string{}, map[string]time.Time{}
	for _, entry := range entries {
		id, isBundle := strings.CutSuffix(entry.Name(), ".json")
		if !isBundle {
			continue
		}
		raw, err := s.fsys().ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		ref, found, ambiguous := sniffJSONString(raw, "payload", "scope", "ref")
		if !found || ambiguous {
			return nil, fmt.Errorf("pose: bundle %s names no single scope", id)
		}
		if !wanted[ref] {
			continue
		}
		text, found, ambiguous := sniffJSONString(raw, "sealed_at")
		at, valid := reviewInstant(text)
		if !found || ambiguous || !valid {
			return nil, fmt.Errorf("pose: bundle %s has no valid sealed_at", id)
		}
		switch current, seen := sealed[ref]; {
		case !seen || at.After(current):
			latest[ref], sealed[ref] = []string{id}, at
		case at.Equal(current):
			latest[ref] = append(latest[ref], id)
		}
	}
	return latest, nil
}

// RoadmapsReachedBy lists the roadmaps a spec leaves with every milestone approved, which
// the close of a member names when its milestone inherits its approval.
func (s Store) RoadmapsReachedBy(spec string) ([]string, error) {
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return nil, err
	}
	found := []string{}
	if adopted, err := s.closeoutReviewPolicyAdopted(); err != nil || adopted {
		// Adopted: a member's close leaves its milestone reached, never closed.
		return found, err
	}
	for _, rm := range roadmaps {
		for _, milestone := range rm.Milestones {
			if member, ok := s.soleMemberOf(ScopeRef{Kind: "milestone", Roadmap: rm.Slug, Milestone: milestone.ID}); ok && member == spec {
				if !s.milestoneReviewRefused("milestone:"+rm.Slug+"/"+milestone.ID) && s.roadmapReachedApartFrom(rm.Slug, milestone.ID) {
					found = append(found, rm.Slug)
				}
				break
			}
		}
	}
	return found, nil
}

// RoadmapReached reports that every milestone of a roadmap not yet done is approved, so
// `pose close roadmap:` is the next step.
func (s Store) RoadmapReached(slug string) bool {
	return s.roadmapReachedApartFrom(slug, "")
}

// roadmapReachedApartFrom leaves out one milestone: the close of its only member has
// just run and is not committed, so its verification cannot answer yet.
func (s Store) roadmapReachedApartFrom(slug, closing string) bool {
	rm, err := s.GetRoadmap(slug)
	if err != nil || rm.Status == "done" || len(rm.Milestones) == 0 {
		return false
	}
	adopted, err := s.closeoutReviewPolicyAdopted()
	if err != nil {
		return false
	}
	for _, milestone := range rm.Milestones {
		if milestone.ID == closing {
			continue
		}
		if adopted {
			if receipt, err := s.ReadCloseoutReceipt("milestone:" + slug + "/" + milestone.ID); err != nil || receipt == nil {
				return false
			}
			continue
		}
		if approved, err := s.scopeApproved("milestone:" + slug + "/" + milestone.ID); err != nil || !approved {
			return false
		}
	}
	return true
}

// startWarnings reports the milestones named in the after: of the milestones a spec
// belongs to that hold no approval. Other forms of after: are not milestones of this
// roadmap, and readiness answers for them. A warning never refuses, so what cannot be
// read is a warning too.
func (s Store) startWarnings(spec string) []string {
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return []string{"the milestones before this spec could not be read: " + err.Error()}
	}
	found := map[string]string{}
	for _, rm := range roadmaps {
		for _, milestone := range rm.Milestones {
			member := false
			for _, raw := range milestone.Specs {
				slug, external := s.milestoneMember(raw)
				member = member || !external && strings.TrimPrefix(slug, "spec:") == spec
			}
			if !member {
				continue
			}
			for _, after := range milestone.After {
				if !strings.Contains(after, ":") {
					after = "milestone:" + rm.Slug + "/" + after
				}
				ref, err := ParseArtifactRef(after)
				if err != nil || ref.Kind != "milestone" || ref.Project != "" {
					continue
				}
				if approved, err := s.scopeApproved(after); err != nil {
					found[after] = after + " comes before this spec and its approval could not be read: " + err.Error()
				} else if !approved {
					found[after] = after + " comes before this spec and is not closed: pose close " + after
				}
			}
		}
	}
	out := make([]string, 0, len(found))
	for _, warning := range found {
		out = append(out, warning)
	}
	sort.Strings(out)
	return out
}

func (s Store) roadmapsDir() string { return filepath.Join(s.Root, ".pose", "roadmaps") }

// GetRoadmap returns one roadmap by slug, body included.
func (s Store) GetRoadmap(slug string) (*Roadmap, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, fmt.Errorf("pose: invalid roadmap slug %q", slug)
	}
	path := filepath.Join(s.roadmapsDir(), slug+".md")
	rm, err := parseRoadmapFile(s.fsys(), path, slug, true)
	if err != nil {
		return nil, fmt.Errorf("pose: roadmap %q not found", slug)
	}
	return rm, nil
}

// ListRoadmaps returns every roadmap (no body), sorted by slug.
func (s Store) ListRoadmaps() ([]Roadmap, error) {
	entries, err := s.fsys().ReadDir(s.roadmapsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []Roadmap{}, nil
		}
		return nil, fmt.Errorf("pose: reading roadmaps dir: %w", err)
	}
	roadmaps := []Roadmap{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		rm, err := parseRoadmapFile(s.fsys(), filepath.Join(s.roadmapsDir(), e.Name()), slug, false)
		if err != nil {
			continue // um artefato quebrado não derruba a listagem
		}
		roadmaps = append(roadmaps, *rm)
	}
	sort.Slice(roadmaps, func(i, j int) bool { return roadmaps[i].Slug < roadmaps[j].Slug })
	return roadmaps, nil
}

func parseRoadmapFile(files Files, path, slug string, includeBody bool) (*Roadmap, error) {
	raw, err := files.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm, body := SplitFrontmatter(string(raw))
	rm := &Roadmap{Slug: slug, Path: path, Milestones: []Milestone{}}
	for key, value := range fm {
		switch key {
		case "slug":
			if value != "" {
				rm.Slug = value
			}
		case "status":
			rm.Status = value
		case "created_at":
			rm.CreatedAt = value
		case "depends_on":
			rm.DependsOn = parseInlineList(value)
		case "consumes":
			rm.Consumes = parseInlineList(value)
		case "unreviewed_milestones":
			rm.UnreviewedMilestones = cleanValue(value)
		case "adrs":
			rm.ADRs = strings.TrimSpace(value)
		}
	}
	var current *Milestone
	inGate := false
	for _, line := range strings.Split(body, "\n") {
		stripped := strings.TrimSpace(line)
		if inGate {
			if stripped == "" || strings.HasPrefix(stripped, "#") || strings.HasPrefix(stripped, "- ") {
				inGate = false
			} else {
				current.ExitGate += " " + stripped
				continue
			}
		}
		if text, ok := strings.CutPrefix(stripped, "**Exit gate:**"); ok && current != nil {
			current.ExitGate, inGate = strings.TrimSpace(text), true
			continue
		}
		if m := milestoneHeadingRE.FindStringSubmatch(stripped); m != nil {
			rm.Milestones = append(rm.Milestones, Milestone{ID: m[1]})
			current = &rm.Milestones[len(rm.Milestones)-1]
			continue
		}
		if strings.HasPrefix(stripped, "## ") {
			current = nil
			continue
		}
		if current == nil || !strings.HasPrefix(stripped, "- ") {
			continue
		}
		key, value, ok := strings.Cut(stripped[2:], ":")
		if !ok {
			continue
		}
		value = cleanValue(value)
		switch strings.TrimSpace(key) {
		case "after":
			current.After = parseInlineList(value)
		case "consumes":
			current.Consumes = parseInlineList(value)
		case "target_start":
			current.TargetStart = value
		case "target_due":
			current.TargetDue = value
		case "specs":
			current.Specs = parseInlineList(value)
		}
	}
	if includeBody {
		rm.Body = body
	}
	return rm, nil
}

func (s Store) closeoutReviewPolicyAdopted() (bool, error) {
	policy, _, err := s.loadReviewPolicy()
	return policy.CloseoutReviewAdopted(), err
}
