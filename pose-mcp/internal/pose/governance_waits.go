package pose

import (
	"sort"
	"time"
)

// GovernanceWaitReport measures how long requests to actors stayed open and
// why (spec pose-governance-wait-rework-observability). Three different
// quantities are kept apart because they mean different things: the age of
// a request, the interval attributed to waiting on an actor or an external
// system, and the interval during which a request is known to have
// restricted a whole spec's execution. None of them is inactivity: work may
// continue on what a request does not restrict.
type GovernanceWaitReport struct {
	SchemaVersion int                `json:"schema_version"`
	Requests      int                `json:"requests"`
	Resolved      int                `json:"resolved"`
	Open          int                `json:"open"`
	AgeSeconds    WaitDistribution   `json:"age_seconds"`
	ByCause       map[string]float64 `json:"attributed_wait_seconds_by_cause"`
	// KnownBlockingSeconds is the union, per spec, of intervals in which a
	// request restricted execution of the whole spec; simultaneous requests
	// are counted once.
	KnownBlockingSeconds float64  `json:"known_blocking_seconds"`
	Unknown              int      `json:"unknown"`
	Limitations          []string `json:"limitations"`
}

// WaitDistribution summarizes durations without ranking anyone.
type WaitDistribution struct {
	Count  int     `json:"count"`
	Median float64 `json:"median"`
	Max    float64 `json:"max"`
}

// GovernanceWaits reads action request journals. Open requests are measured
// up to now and labelled open.
func (s Store) GovernanceWaits(now time.Time) (GovernanceWaitReport, error) {
	report := GovernanceWaitReport{SchemaVersion: 1, ByCause: map[string]float64{},
		Limitations: []string{
			"age, attributed waiting and known blocking are different measures; none is inactivity",
			"only action requests are timed; waits recorded nowhere stay unknown",
		}}
	views, err := s.ListActionRequests()
	if err != nil {
		return report, err
	}
	var ages []float64
	blocking := map[string][][2]time.Time{}
	for _, view := range views {
		report.Requests++
		opened, err := time.Parse(time.RFC3339, view.Request.RequestedAt)
		if err != nil {
			report.Unknown++
			continue
		}
		end := now
		closed := false
		for _, event := range view.Events[1:] {
			if event.Type == ActionEventAnswered || event.Type == ActionEventCancelled || event.Type == ActionEventWaived || event.Type == ActionEventSuperseded {
				if at, err := time.Parse(time.RFC3339, event.At); err == nil {
					end, closed = at, true
				} else {
					report.Unknown++
				}
				break
			}
		}
		if closed {
			report.Resolved++
		} else {
			report.Open++
		}
		if end.Before(opened) {
			report.Unknown++
			continue
		}
		duration := end.Sub(opened).Seconds()
		ages = append(ages, duration)
		cause := "actor"
		if view.Request.Kind == ActionExternalOperation {
			cause = "external"
		}
		report.ByCause[cause] += duration
		for _, effect := range view.Request.Effects {
			if effect.Phase == PhaseExecution && effect.Mode == EffectBlock && len(effect.Scope) == 0 {
				blocking[view.Request.Origin] = append(blocking[view.Request.Origin], [2]time.Time{opened, end})
			}
		}
	}
	report.AgeSeconds = distribution(ages)
	for _, intervals := range blocking {
		report.KnownBlockingSeconds += unionSeconds(intervals)
	}
	return report, nil
}

func distribution(values []float64) WaitDistribution {
	if len(values) == 0 {
		return WaitDistribution{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	return WaitDistribution{Count: len(sorted), Median: median, Max: sorted[len(sorted)-1]}
}

// unionSeconds merges overlapping intervals so simultaneous waits count once.
func unionSeconds(intervals [][2]time.Time) float64 {
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0].Before(intervals[j][0]) })
	total := 0.0
	start, end := intervals[0][0], intervals[0][1]
	for _, iv := range intervals[1:] {
		if iv[0].After(end) {
			total += end.Sub(start).Seconds()
			start, end = iv[0], iv[1]
			continue
		}
		if iv[1].After(end) {
			end = iv[1]
		}
	}
	return total + end.Sub(start).Seconds()
}

// GovernanceReworkReport explains why review work was redone (spec
// pose-governance-wait-rework-observability): a superseded bundle is
// classified by what changed, an extra attestation on one bundle by whether
// the earlier one is invalid against it. Causes without evidence stay
// unknown; nothing is attributed to a person or to the engine by default.
type GovernanceReworkReport struct {
	SchemaVersion       int            `json:"schema_version"`
	Scopes              int            `json:"scopes"`
	Bundles             int            `json:"bundles"`
	Attestations        int            `json:"attestations"`
	BundleSupersessions map[string]int `json:"bundle_supersessions_by_cause"`
	AttestationRework   map[string]int `json:"attestation_rework_by_cause"`
	Limitations         []string       `json:"limitations"`
}

// GovernanceRework reads sealed bundles and their attestations.
func (s Store) GovernanceRework() (GovernanceReworkReport, error) {
	report := GovernanceReworkReport{SchemaVersion: 1, BundleSupersessions: map[string]int{}, AttestationRework: map[string]int{},
		Limitations: []string{
			"many review operations are not a sign of bad governance; a superseded bundle often means a gate found something",
			"an earlier attestation is checked against today's policy; a cause the records do not show stays unknown",
		}}
	bundles, err := s.ListReviewBundles("")
	if err != nil {
		return report, err
	}
	byScope := map[string][]ReviewBundle{}
	for _, b := range bundles {
		if b.State != "sealed" {
			continue
		}
		byScope[b.Payload.Scope.Ref] = append(byScope[b.Payload.Scope.Ref], b)
	}
	// One read of every attestation, grouped by bundle: listing per bundle
	// rereads the whole directory each time.
	all, err := s.ListReviewAttestations("")
	if err != nil {
		return report, err
	}
	byBundle := map[string][]ReviewAttestation{}
	for _, att := range all {
		byBundle[att.BundleID] = append(byBundle[att.BundleID], att)
	}
	for id := range byBundle {
		sort.SliceStable(byBundle[id], func(i, j int) bool { return byBundle[id][i].AttestedAt < byBundle[id][j].AttestedAt })
	}
	report.Scopes = len(byScope)
	for _, list := range byScope {
		sort.Slice(list, func(i, j int) bool { return list[i].SealedAt < list[j].SealedAt })
		report.Bundles += len(list)
		for i := 1; i < len(list); i++ {
			delta := ReviewBundleDiff(list[i-1], list[i])
			switch {
			case len(delta.ChangedPaths) > 0:
				report.BundleSupersessions["subject-changed"]++
			case len(delta.ChangedSections) > 0:
				report.BundleSupersessions["intent-or-plan-changed"]++
			case len(delta.ChangedCriteria) > 0 || len(delta.ChangedComponents) > 0:
				report.BundleSupersessions["review-plan-changed"]++
			case len(delta.ChangedEvidence) > 0 || len(delta.ChangedEvidenceClasses) > 0:
				report.BundleSupersessions["evidence-changed"]++
			default:
				report.BundleSupersessions["unknown"]++
			}
		}
		for _, bundle := range list {
			attestations := byBundle[bundle.BundleID]
			report.Attestations += len(attestations)
			for i := 0; i+1 < len(attestations); i++ {
				earlier := attestations[i]
				switch {
				case earlier.Decision == "changes-requested" || earlier.Decision == "rejected":
					report.AttestationRework["review-disagreement"]++
				case len(s.validateBundleAttestation(bundle, earlier)) > 0:
					report.AttestationRework["invalid-attestation"]++
				default:
					report.AttestationRework["unknown"]++
				}
			}
		}
	}
	return report, nil
}
