package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// Project-level obligation sources (spec pose-attention-projects-every-source).
//
// Four producers were listed as "not projected yet", so Attention reported
// incomplete coverage for every project that used them: docs review
// pendencies, capability stale triggers, release queues and findings outside
// review attestations. None of them belongs to a spec, so their obligations
// originate on the project itself: `xref:<project>/project:<project>` with a
// node naming the doc, capability mechanism, release or finding. A
// spec-scoped query never sees them; a project read always does.

// projectNode returns the project-level node for kind and id.
func projectNode(project, kind, id string) NodeRef {
	return QualifyNodeRef(project, ArtifactRef{Kind: "project", Slug: project}, kind, id)
}

// obligationNodeID turns free text (a path, a version) into a node id: a
// readable slug plus a short digest, so two values never share an id.
func obligationNodeID(value string) string {
	slug := strings.Trim(slugSepRE.ReplaceAllString(strings.ToLower(value), "-"), "-")
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	sum := sha256.Sum256([]byte(value))
	if slug == "" {
		return hex.EncodeToString(sum[:6])
	}
	return slug + "-" + hex.EncodeToString(sum[:4])
}

// docsReviewObligations projects every doc with an open review mark.
func (s Store) docsReviewObligations(project string) ([]Obligation, error) {
	events, err := LoadDocsReviewEvents(s.DocsReviewPath())
	if err != nil {
		return nil, err
	}
	pending := PendingDocsReviews(events)
	if len(pending) == 0 {
		return nil, nil
	}
	owners := map[string]string{}
	if manifest, err := s.LoadDocsManifest(); err == nil && manifest != nil {
		for _, entry := range manifest.Entries {
			owners[entry.Path] = entry.Owner
		}
	}
	docs := make([]string, 0, len(pending))
	for doc := range pending {
		docs = append(docs, doc)
	}
	sort.Strings(docs)
	out := []Obligation{}
	for _, doc := range docs {
		triggers := []string{}
		for _, trigger := range pending[doc] {
			triggers = append(triggers, trigger.Trigger)
		}
		o := newObligation(project, "docs-review", projectNode(project, "doc", obligationNodeID(doc)), "docs-review-pending", doc)
		o.Category = ObligationReconciliation
		o.ReasonCode = "docs-review-pending"
		o.Condition = "`pose docs-review resolve " + doc + "` records that the doc was updated, or that no change was needed and why"
		o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectAdvisory}}
		o.Waiting = WaitingExecution
		if owner := strings.TrimSpace(owners[doc]); owner != "" {
			o.Recipient = ObligationActor{Principal: owner}
			o.Waiting = WaitingActor
		}
		o.Message = doc + " may need review after " + strings.Join(triggers, ", ")
		out = append(out, o)
	}
	return out, nil
}

// capabilityTriggerObligations projects every mechanism with a pending
// reassessment demand.
func (s Store) capabilityTriggerObligations(project string) ([]Obligation, error) {
	if !s.HasCapabilityAssessment() {
		return nil, nil
	}
	assessment, err := s.LoadCapabilityAssessment()
	if err != nil {
		return nil, err
	}
	out := []Obligation{}
	for _, mechanism := range assessment.Mechanisms {
		if mechanism.Retired || len(mechanism.StaleTriggers) == 0 {
			continue
		}
		triggers := []string{}
		for _, trigger := range mechanism.StaleTriggers {
			triggers = append(triggers, trigger.Trigger)
		}
		o := newObligation(project, "capability-trigger", projectNode(project, "capability", obligationNodeID(mechanism.ID)), "capability-reassessment", mechanism.ID)
		o.Category = ObligationEvidence
		o.ReasonCode = "capability-stale"
		o.Condition = "the mechanism is reassessed and `pose assess snapshot` clears its stale triggers"
		o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectAdvisory}}
		o.Waiting = WaitingExecution
		o.Message = "mechanism " + mechanism.ID + " may need reassessment after " + strings.Join(triggers, ", ")
		out = append(out, o)
	}
	return out, nil
}

// releaseQueueObligations projects the releases still in flight: those newer
// than the newest verified release that are not verified or yanked. An older
// release left prepared, tagged or failed was superseded by a later verified
// one and owes nothing, and one left prepared or failed was abandoned when a
// later release was tagged.
func (s Store) releaseQueueObligations(project string) ([]Obligation, error) {
	status, err := s.GetReleaseStatus("")
	if err != nil {
		return nil, err
	}
	newestVerified, newestProgressed := "", ""
	for _, release := range status.Releases {
		if release.State == "verified" && (newestVerified == "" || compareReleaseVersions(release.Version, newestVerified) > 0) {
			newestVerified = release.Version
		}
		if release.State != "prepared" && release.State != "failed" && (newestProgressed == "" || compareReleaseVersions(release.Version, newestProgressed) > 0) {
			newestProgressed = release.Version
		}
	}
	out := []Obligation{}
	for _, release := range status.Releases {
		if release.State == "verified" || release.State == "yanked" {
			continue
		}
		if newestVerified != "" && compareReleaseVersions(release.Version, newestVerified) <= 0 {
			continue
		}
		// A release left prepared or failed while a later one moved on was
		// abandoned for it.
		if (release.State == "prepared" || release.State == "failed") && newestProgressed != "" && compareReleaseVersions(release.Version, newestProgressed) < 0 {
			continue
		}
		o := newObligation(project, "release", projectNode(project, "release", obligationNodeID(release.Version)), "release-lifecycle", release.Version)
		o.Category = ObligationRelease
		o.ReasonCode = "release-" + release.State
		o.Effects = []ObligationEffect{{Phase: PhaseRelease, Mode: EffectAdvisory}}
		switch release.State {
		case "prepared":
			o.Condition = "the release commit is tagged `" + release.Version + "`"
			o.Waiting = WaitingExecution
		case "tagged":
			o.Condition = "the release workflow publishes `" + release.Version + "` and its evidence is recorded"
			o.Waiting = WaitingExternal
		case "published":
			o.Condition = "`pose release record --version " + release.Version + " --event verified --evidence <file>` binds verification to the publication"
			o.Waiting = WaitingExecution
		default:
			o.Condition = "the failure of `" + release.Version + "` is recovered and the release published"
			o.Waiting = WaitingExecution
		}
		o.Message = "release " + release.Version + " is " + release.State
		out = append(out, o)
	}
	return out, nil
}

// findingObligations projects findings recorded outside review attestations
// — in the latest review record of each scope — whose accepted risk or
// follow-up has passed its review date. Open findings of a current review are
// the review producer's; free-form investigation notes carry no state to
// project.
func (s Store) findingObligations(project string, now time.Time) ([]Obligation, error) {
	attempts, err := s.ListReviewAttempts("")
	if err != nil {
		return nil, err
	}
	latest := map[string]ReviewAttempt{}
	for _, attempt := range attempts {
		if old, ok := latest[attempt.Scope]; !ok || attempt.ReviewedAt > old.ReviewedAt {
			latest[attempt.Scope] = attempt
		}
	}
	today := now.UTC().Format("2006-01-02")
	scopes := make([]string, 0, len(latest))
	for scope := range latest {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	out := []Obligation{}
	for _, scope := range scopes {
		attempt := latest[scope]
		for _, finding := range attempt.Findings {
			if finding.Disposition != "accepted-risk" && finding.Disposition != "follow-up" {
				continue
			}
			if finding.ReviewBy == "" || finding.ReviewBy >= today {
				continue
			}
			o := newObligation(project, "findings", projectNode(project, "finding", obligationNodeID(scope+"/"+finding.ID)), "finding-review-due", attempt.ReviewID+"/"+finding.ID)
			o.Category = ObligationJudgment
			o.ReasonCode = "finding-" + finding.Disposition + "-overdue"
			o.Condition = "the " + finding.Disposition + " finding " + finding.ID + " of " + scope + " is reconsidered and a superseding review records the outcome"
			o.Effects = []ObligationEffect{{Phase: PhaseRelease, Mode: EffectAdvisory}}
			o.Waiting = WaitingExecution
			if owner := strings.TrimSpace(finding.Owner); owner != "" {
				o.Recipient = ObligationActor{Principal: owner}
				o.Waiting = WaitingActor
			}
			o.Message = finding.ID + " (" + finding.Severity + ") in " + scope + " was due for review on " + finding.ReviewBy
			out = append(out, o)
		}
	}
	return out, nil
}
