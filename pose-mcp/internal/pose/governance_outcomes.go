package pose

// Governance outcomes are a read-only projection over artifacts POSE already
// owns. The projection deliberately keeps coverage and unknown values visible:
// it is an observation surface, not a quality score or a closeout authority.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	GovernanceOutcomesSchemaVersion = 2
	governanceDefaultMaturityDays   = 30
	governanceDefaultMinSample      = 3
	governanceMaxHistoryLineBytes   = 1 << 20
	governanceMaxEntries            = 100000
	governanceMaxFreshnessChecks    = 256
	governanceMaxRemediationLinks   = 100
)

// GovernanceOutcomesQuery controls a deterministic projection. A zero Now is
// replaced with the current UTC time at the boundary; callers that need a
// reproducible report should provide it explicitly.
type GovernanceOutcomesQuery struct {
	ProjectID    string
	SinceDays    int
	MaturityDays int
	MinSample    int
	Band         string
	ReportType   string
	Now          time.Time
}

type GovernanceCoverage struct {
	Available                 bool   `json:"available"`
	Reason                    string `json:"reason,omitempty"`
	HistoryRecordsScanned     int    `json:"history_records_scanned"`
	HistoryRecordsMatched     int    `json:"history_records_matched"`
	HistoryInvalidRecords     int    `json:"history_invalid_records"`
	HistoryRecordsDuplicate   int    `json:"history_records_duplicate"`
	HistoryIdentityConflicts  int    `json:"history_identity_conflicts"`
	HistoryIdentityUnknown    int    `json:"history_identity_unknown"`
	ReviewBundlesScanned      int    `json:"review_bundles_scanned"`
	ReviewBundlesInvalid      int    `json:"review_bundles_invalid"`
	ReviewBandsUnknown        int    `json:"review_bands_unknown"`
	AttestationsScanned       int    `json:"attestations_scanned"`
	AttestationsInvalid       int    `json:"attestations_invalid"`
	AttestationsWithoutBundle int    `json:"attestations_without_bundle"`
	RemediationLinksScanned   int    `json:"remediation_links_scanned"`
	RemediationLinksInvalid   int    `json:"remediation_links_invalid"`
	RemediationLinksTruncated bool   `json:"remediation_links_truncated"`
	FreshnessChecks           int    `json:"freshness_checks"`
	FreshnessUnknown          int    `json:"freshness_unknown"`
	FreshnessTruncated        bool   `json:"freshness_truncated"`
	PreparationPhaseCoverage  string `json:"preparation_phase_coverage"`
	RemediationCoverage       string `json:"remediation_coverage"`
}

type GovernanceAttemptDimensions struct {
	UnitsObserved          int     `json:"units_observed"`
	AttemptsObserved       int     `json:"attempts_observed"`
	Pass                   int     `json:"pass"`
	Fail                   int     `json:"fail"`
	Partial                int     `json:"partial"`
	Skipped                int     `json:"skipped"`
	Unknown                int     `json:"unknown"`
	PreparationReports     int     `json:"preparation_reports"`
	JudgmentAttempts       int     `json:"judgment_attempts"`
	SupersededRelations    int     `json:"superseded_relations"`
	FirstApprovedUnits     int     `json:"first_approved_units"`
	ActiveDurationObserved int     `json:"active_duration_observed"`
	ActiveDurationSeconds  float64 `json:"active_duration_seconds"`
	ActiveDurationUnknown  int     `json:"active_duration_unknown"`
	WaitDurationObserved   int     `json:"wait_duration_observed"`
	WaitDurationSeconds    float64 `json:"wait_duration_seconds"`
	WaitDurationUnknown    int     `json:"wait_duration_unknown"`
	ObservedCostCount      int     `json:"observed_cost_count"`
	ObservedCostUSD        float64 `json:"observed_cost_usd"`
	UnknownCostCount       int     `json:"unknown_cost_count"`
}

type GovernanceReviewDimensions struct {
	BundlesObserved          int `json:"bundles_observed"`
	AttestationsObserved     int `json:"attestations_observed"`
	Approved                 int `json:"approved"`
	ApprovedWithReservations int `json:"approved_with_reservations"`
	ChangesRequested         int `json:"changes_requested"`
	Rejected                 int `json:"rejected"`
	UnknownDecisions         int `json:"unknown_decisions"`
	InterventionsObserved    int `json:"interventions_observed"`
	FindingsObserved         int `json:"findings_observed"`
	NotApplicable            int `json:"not_applicable"`
	AcceptedRisk             int `json:"accepted_risk"`
	ResolvedFindings         int `json:"resolved_findings"`
	WontFixFindings          int `json:"wont_fix_findings"`
	OpenFindings             int `json:"open_findings"`
	StaleObserved            int `json:"stale_observed"`
	StaleUnknown             int `json:"stale_unknown"`
}

type GovernanceRemediationDimensions struct {
	Available          bool   `json:"available"`
	Reason             string `json:"reason,omitempty"`
	MaturePopulation   int    `json:"mature_population"`
	ObservedRemediated int    `json:"observed_remediated"`
	Censored           int    `json:"censored"`
	InsufficientSample bool   `json:"insufficient_sample"`
	// LinksDeclared is the whole graph; the fields below are the mature
	// population only. A delivery nobody linked to is UnlinkedUnknown and not a
	// clean one: nothing in the repository asserts it had no defect, so it is
	// reported beside the rate instead of being folded into it.
	LinksDeclared   int `json:"links_declared"`
	LinkedObserved  int `json:"linked_observed"`
	UnlinkedUnknown int `json:"unlinked_unknown"`
	// InvalidLinks is coverage loss, never a zero.
	InvalidLinks int `json:"invalid_links"`
	// ByCategory reports every category observed on the mature population, and
	// CountedCategories says which of them enter ObservedRemediated, so a reader
	// never has to infer which links were treated as defects.
	ByCategory        map[string]int              `json:"by_category,omitempty"`
	CountedCategories []string                    `json:"counted_categories,omitempty"`
	LinksTotal        int                         `json:"links_total"`
	LinksTruncated    bool                        `json:"links_truncated"`
	Links             []GovernanceRemediationLink `json:"links,omitempty"`
}

// GovernanceRemediationLink is the allowlisted path from an explicit finding
// reference to its declared remediation spec. It intentionally omits finding
// text, reviewer identity, owner, rationale and evidence content.
type GovernanceRemediationLink struct {
	AttestationID   string `json:"attestation_id"`
	FindingID       string `json:"finding_id"`
	SourceSpec      string `json:"source_spec"`
	RemediationSpec string `json:"remediation_spec"`
	Category        string `json:"category"`
	SourceBand      string `json:"source_band"`
}

type GovernanceProvenanceSource struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type GovernanceProvenance struct {
	Producer              string                       `json:"producer"`
	ReportAttemptIdentity string                       `json:"report_attempt_identity"`
	Sources               []GovernanceProvenanceSource `json:"sources"`
}

type GovernanceFilters struct {
	Band                string   `json:"band,omitempty"`
	ReportType          string   `json:"report_type,omitempty"`
	BandAppliesTo       []string `json:"band_applies_to"`
	ReportTypeAppliesTo []string `json:"report_type_applies_to"`
}

// GovernanceOutcomesReport is intentionally a set of independent dimensions.
// There is no aggregate score and no field whose value means “good”.
type GovernanceOutcomesReport struct {
	SchemaVersion int                             `json:"schema_version"`
	ProjectID     string                          `json:"project_id,omitempty"`
	GeneratedAt   string                          `json:"generated_at"`
	WindowEnd     string                          `json:"window_end"`
	WindowStart   string                          `json:"window_start,omitempty"`
	SinceDays     int                             `json:"since_days"`
	MaturityDays  int                             `json:"maturity_days"`
	MinSample     int                             `json:"min_sample"`
	Filters       GovernanceFilters               `json:"filters"`
	Provenance    GovernanceProvenance            `json:"provenance"`
	Coverage      GovernanceCoverage              `json:"coverage"`
	Attempts      GovernanceAttemptDimensions     `json:"attempts"`
	Reviews       GovernanceReviewDimensions      `json:"reviews"`
	Remediation   GovernanceRemediationDimensions `json:"remediation"`
}

type governanceHistoryRecord struct {
	GeneratedAt     string   `json:"generated_at"`
	Sequence        int      `json:"sequence"`
	Task            string   `json:"task"`
	TaskSlug        string   `json:"task_slug"`
	Spec            string   `json:"spec"`
	ReportType      string   `json:"report_type"`
	StableHash      string   `json:"stable_hash"`
	Outcome         string   `json:"outcome"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	CostUSD         *float64 `json:"cost_usd,omitempty"`
}

type governanceHistoryRead struct {
	records []governanceHistoryRecord
	invalid int
	scanned int
}

// GovernanceOutcomes reads only fixed POSE directories and never executes
// repository content. Invalid lines are counted as coverage loss.
func (s Store) GovernanceOutcomes(query GovernanceOutcomesQuery) (*GovernanceOutcomesReport, error) {
	if query.SinceDays < 0 {
		return nil, errors.New("pose governance stats: since_days must be non-negative")
	}
	if query.MaturityDays < 0 {
		return nil, errors.New("pose governance stats: maturity_days must be non-negative")
	}
	if query.MinSample < 0 {
		return nil, errors.New("pose governance stats: min_sample must be non-negative")
	}
	if !validGovernanceBand(query.Band) {
		return nil, errors.New("pose governance stats: band must be baseline, elevated, critical or unknown")
	}
	if !validGovernanceReportType(query.ReportType) {
		return nil, errors.New("pose governance stats: report_type must be standard, doc-audit or unknown")
	}
	if query.ProjectID != "" && !validGovernanceProjectID(query.ProjectID) {
		return nil, errors.New("pose governance stats: invalid project_id")
	}
	if query.MaturityDays == 0 {
		query.MaturityDays = governanceDefaultMaturityDays
	}
	if query.MinSample == 0 {
		query.MinSample = governanceDefaultMinSample
	}
	now := query.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	report := &GovernanceOutcomesReport{
		SchemaVersion: GovernanceOutcomesSchemaVersion,
		ProjectID:     query.ProjectID,
		GeneratedAt:   now.Format(time.RFC3339), WindowEnd: now.Format(time.RFC3339), SinceDays: query.SinceDays,
		MaturityDays: query.MaturityDays, MinSample: query.MinSample,
		Filters: GovernanceFilters{
			Band: query.Band, ReportType: query.ReportType,
			BandAppliesTo:       []string{"reviews", "remediation.links"},
			ReportTypeAppliesTo: []string{"attempts"},
		},
		Provenance: GovernanceProvenance{
			Producer: "pose_governance_stats", ReportAttemptIdentity: "report_type/task_slug/sequence",
			Sources: []GovernanceProvenanceSource{
				{ID: "report-history", Version: "jsonl-sequence-v1"},
				{ID: "review-bundles", Version: "review-bundle-v1"},
				{ID: "review-attestations", Version: "review-attestation-v1"},
				{ID: "spec-remediates", Version: "remediation-links-v1"},
			},
		},
		Coverage: GovernanceCoverage{PreparationPhaseCoverage: "partial", RemediationCoverage: "unavailable"},
	}
	if query.SinceDays > 0 {
		report.WindowStart = now.AddDate(0, 0, -query.SinceDays).Format(time.RFC3339)
	}
	report.Remediation = s.remediationProjection(now, query.SinceDays, query.MaturityDays, query.MinSample, query.Band)
	if report.Remediation.Available {
		// Partial, never complete: an absent link is unknown, so coverage of the
		// mature population can only ever be the part that carries one.
		report.Coverage.RemediationCoverage = "partial"
	}

	history, err := readGovernanceHistory(s.Root)
	if err != nil {
		return nil, err
	}
	report.Coverage.HistoryRecordsScanned = history.scanned
	report.Coverage.HistoryInvalidRecords = history.invalid
	history.records, report.Coverage.HistoryRecordsDuplicate, report.Coverage.HistoryIdentityConflicts, report.Coverage.HistoryIdentityUnknown = deduplicateGovernanceHistory(history.records)
	cutoff := time.Time{}
	if query.SinceDays > 0 {
		cutoff = now.AddDate(0, 0, -query.SinceDays)
	}
	unitKeys := map[string]bool{}
	for _, record := range history.records {
		if !governanceReportTypeMatches(record.ReportType, query.ReportType) {
			continue
		}
		at, ok := parseGovernanceTime(record.GeneratedAt)
		if !ok || at.After(now) || (!cutoff.IsZero() && at.Before(cutoff)) {
			continue
		}
		report.Coverage.HistoryRecordsMatched++
		report.Attempts.AttemptsObserved++
		report.Attempts.PreparationReports++
		key := strings.TrimSpace(record.Spec)
		if key == "" {
			key = strings.TrimSpace(record.TaskSlug)
		}
		if key == "" {
			key = "_unknown_unit_"
		}
		unitKeys[key] = true
		switch record.Outcome {
		case "pass":
			report.Attempts.Pass++
		case "fail":
			report.Attempts.Fail++
		case "partial":
			report.Attempts.Partial++
		case "skipped":
			report.Attempts.Skipped++
		default:
			report.Attempts.Unknown++
		}
		if record.DurationSeconds != nil {
			report.Attempts.ActiveDurationObserved++
			report.Attempts.ActiveDurationSeconds += *record.DurationSeconds
		} else {
			report.Attempts.ActiveDurationUnknown++
		}
		// POSE report telemetry has no wait/queue field. Keep that unknown
		// rather than treating active duration as end-to-end waiting time.
		report.Attempts.WaitDurationUnknown++
		if record.CostUSD != nil {
			report.Attempts.ObservedCostCount++
			report.Attempts.ObservedCostUSD += *record.CostUSD
		} else {
			report.Attempts.UnknownCostCount++
		}
	}
	report.Attempts.UnitsObserved = len(unitKeys)

	bundles, bundleInvalid, err := readGovernanceBundles(s)
	if err != nil {
		return nil, err
	}
	report.Coverage.ReviewBundlesScanned = len(bundles) + bundleInvalid
	report.Coverage.ReviewBundlesInvalid = bundleInvalid
	for _, bundle := range bundles {
		at, ok := parseGovernanceTime(bundle.SealedAt)
		if !ok || at.After(now) || (!cutoff.IsZero() && at.Before(cutoff)) {
			continue
		}
		if !governanceBandMatches(governanceBundleBand(bundle), query.Band) {
			continue
		}
		report.Reviews.BundlesObserved++
	}

	atts, attInvalid, err := readGovernanceAttestations(s)
	if err != nil {
		return nil, err
	}
	report.Coverage.AttestationsScanned = len(atts) + attInvalid
	report.Coverage.AttestationsInvalid = attInvalid
	bundleByID := make(map[string]ReviewBundle, len(bundles))
	for _, bundle := range bundles {
		bundleByID[bundle.BundleID] = bundle
	}
	bandByBundle := make(map[string]string, len(bundles))
	for _, bundle := range bundles {
		band := governanceBundleBand(bundle)
		bandByBundle[bundle.BundleID] = band
		if band == "unknown" {
			report.Coverage.ReviewBandsUnknown++
		}
	}
	firstApproval := map[string]bool{}
	superseded := map[string]bool{}
	freshness := map[string]string{}
	freshnessChecks := 0
	// Freshness is verified through a local store on the same root: the
	// projection is local by definition, and a federated store resolves every
	// qualified dependency again per bundle, which made the MCP answer both
	// far slower than the CLI's and able to differ from it.
	verifier := Store{Root: s.Root}
	for _, att := range atts {
		at, ok := parseGovernanceTime(att.AttestedAt)
		if !ok || at.After(now) || (!cutoff.IsZero() && at.Before(cutoff)) {
			continue
		}
		bundle, found := bundleByID[att.BundleID]
		band := "unknown"
		if found {
			band = bandByBundle[bundle.BundleID]
		}
		if !governanceBandMatches(band, query.Band) {
			continue
		}
		report.Attempts.JudgmentAttempts++
		report.Reviews.AttestationsObserved++
		if !found {
			report.Coverage.AttestationsWithoutBundle++
		} else {
			key := bundle.Payload.Scope.Ref
			if key == "" {
				key = "_unknown_scope_"
			}
			if att.Decision == "approved" || att.Decision == "approved-with-reservations" {
				if !firstApproval[key] {
					firstApproval[key] = true
				}
			}
			if att.Supersedes != "" {
				superseded[bundle.BundleID+"\x00"+att.Supersedes] = true
			}
			if _, done := freshness[bundle.BundleID]; !done {
				if freshnessChecks < governanceMaxFreshnessChecks {
					freshnessChecks++
					verification, verifyErr := verifier.VerifyReviewBundle(bundle.Payload.Scope.Ref)
					if verifyErr != nil {
						freshness[bundle.BundleID] = "unknown"
					} else if verification.Fresh {
						freshness[bundle.BundleID] = "fresh"
					} else {
						freshness[bundle.BundleID] = "stale"
					}
				} else {
					freshness[bundle.BundleID] = "unknown"
					report.Coverage.FreshnessTruncated = true
				}
			}
		}
		interventionInAttempt := false
		switch att.Decision {
		case "approved":
			report.Reviews.Approved++
		case "approved-with-reservations":
			report.Reviews.ApprovedWithReservations++
		case "changes-requested":
			report.Reviews.ChangesRequested++
			interventionInAttempt = true
		case "rejected":
			report.Reviews.Rejected++
			interventionInAttempt = true
		default:
			report.Reviews.UnknownDecisions++
		}
		findingInAttempt := false
		for _, criterion := range att.Criteria {
			if criterion.Disposition == "not-applicable" {
				report.Reviews.NotApplicable++
			}
			if criterion.Disposition == "finding" {
				findingInAttempt = true
			}
		}
		for _, finding := range att.Findings {
			report.Reviews.FindingsObserved++
			findingInAttempt = true
			switch finding.Disposition {
			case "accepted-risk":
				report.Reviews.AcceptedRisk++
			case "resolved":
				report.Reviews.ResolvedFindings++
			case "wont-fix":
				report.Reviews.WontFixFindings++
			case "open", "changes-requested":
				report.Reviews.OpenFindings++
			}
		}
		if findingInAttempt {
			interventionInAttempt = true
		}
		if interventionInAttempt {
			report.Reviews.InterventionsObserved++
		}
	}
	report.Attempts.SupersededRelations = len(superseded)
	report.Attempts.FirstApprovedUnits = len(firstApproval)
	report.Coverage.FreshnessChecks = freshnessChecks
	report.Coverage.RemediationLinksScanned = report.Remediation.LinksDeclared
	report.Coverage.RemediationLinksInvalid = report.Remediation.InvalidLinks
	report.Coverage.RemediationLinksTruncated = report.Remediation.LinksTruncated
	for _, state := range freshness {
		switch state {
		case "stale":
			report.Reviews.StaleObserved++
		case "unknown":
			report.Reviews.StaleUnknown++
			report.Coverage.FreshnessUnknown++
		}
	}
	report.Coverage.Available = report.Coverage.HistoryRecordsMatched > 0 || report.Reviews.AttestationsObserved > 0 || report.Reviews.BundlesObserved > 0
	if !report.Coverage.Available {
		report.Coverage.Reason = "no observable report, bundle or attestation records matched the query"
	}
	return report, nil
}

func validGovernanceBand(value string) bool {
	switch value {
	case "", ReviewBandBaseline, ReviewBandElevated, ReviewBandCritical, "unknown":
		return true
	default:
		return false
	}
}

func validGovernanceReportType(value string) bool {
	switch value {
	case "", "standard", "doc-audit", "unknown":
		return true
	default:
		return false
	}
}

func validGovernanceProjectID(value string) bool {
	if len(value) == 0 || len(value) > 128 || !isASCIIAlphaNumeric(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !isASCIIAlphaNumeric(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func governanceReportTypeMatches(recordType, filter string) bool {
	if filter == "" {
		return true
	}
	typeValue := strings.TrimSpace(recordType)
	if typeValue != "standard" && typeValue != "doc-audit" {
		typeValue = "unknown"
	}
	return filter == typeValue
}

func governanceBundleBand(bundle ReviewBundle) string {
	band := strings.TrimSpace(bundle.Payload.Plan.Band)
	if band != ReviewBandBaseline && band != ReviewBandElevated && band != ReviewBandCritical {
		return "unknown"
	}
	return band
}

func governanceBandMatches(recordBand, filter string) bool {
	return filter == "" || recordBand == filter
}

// deduplicateGovernanceHistory deduplicates only records with an explicit
// producer identity. Legacy rows lacking it remain in the denominator and
// are reported separately; `stable_hash` is not an attempt identity.
func deduplicateGovernanceHistory(records []governanceHistoryRecord) ([]governanceHistoryRecord, int, int, int) {
	result := make([]governanceHistoryRecord, 0, len(records))
	indices := map[string]int{}
	conflicted := map[string]bool{}
	duplicates, conflicts, unknown := 0, 0, 0
	for _, record := range records {
		typeName := strings.TrimSpace(record.ReportType)
		taskSlug := strings.TrimSpace(record.TaskSlug)
		if typeName == "" || taskSlug == "" || record.Sequence <= 0 {
			unknown++
			result = append(result, record)
			continue
		}
		key := typeName + "\x00" + taskSlug + "\x00" + fmt.Sprint(record.Sequence)
		index, exists := indices[key]
		if !exists {
			indices[key] = len(result)
			result = append(result, record)
			continue
		}
		if reflect.DeepEqual(result[index], record) {
			duplicates++
			continue
		}
		if !conflicted[key] {
			conflicts++
			conflicted[key] = true
		}
		result[index] = mergeConflictingGovernanceHistory(result[index], record)
	}
	return result, duplicates, conflicts, unknown
}

func mergeConflictingGovernanceHistory(current, incoming governanceHistoryRecord) governanceHistoryRecord {
	if incomingOutcomeSeverity(incoming.Outcome) > incomingOutcomeSeverity(current.Outcome) {
		current.Outcome = incoming.Outcome
	} else if incomingOutcomeSeverity(incoming.Outcome) == incomingOutcomeSeverity(current.Outcome) && incoming.Outcome != current.Outcome {
		current.Outcome = "unknown"
	}
	currentAt, currentOK := parseGovernanceTime(current.GeneratedAt)
	incomingAt, incomingOK := parseGovernanceTime(incoming.GeneratedAt)
	if incomingOK && (!currentOK || incomingAt.Before(currentAt)) {
		current.GeneratedAt = incoming.GeneratedAt
	}
	if !reflect.DeepEqual(current.DurationSeconds, incoming.DurationSeconds) {
		current.DurationSeconds = nil
	}
	if !reflect.DeepEqual(current.CostUSD, incoming.CostUSD) {
		current.CostUSD = nil
	}
	if current.StableHash != incoming.StableHash {
		current.StableHash = ""
	}
	return current
}

func incomingOutcomeSeverity(outcome string) int {
	switch outcome {
	case "fail":
		return 5
	case "partial":
		return 4
	case "unknown":
		return 3
	case "skipped":
		return 2
	case "pass":
		return 1
	default:
		return 3
	}
}

func parseGovernanceTime(value string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func readGovernanceHistory(root string) (governanceHistoryRead, error) {
	result := governanceHistoryRead{}
	dir := filepath.Join(root, ".pose", "reports", "history")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("pose governance stats: read history: %w", err)
	}
	for _, entry := range entries {
		if result.scanned >= governanceMaxEntries || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		file, openErr := os.Open(filepath.Join(dir, entry.Name()))
		if openErr != nil {
			result.invalid++
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), governanceMaxHistoryLineBytes)
		for scanner.Scan() && result.scanned < governanceMaxEntries {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			result.scanned++
			var record governanceHistoryRecord
			if json.Unmarshal([]byte(line), &record) != nil {
				result.invalid++
				continue
			}
			if record.TaskSlug == "" {
				record.TaskSlug = record.Task
			}
			result.records = append(result.records, record)
		}
		if scanner.Err() != nil {
			result.invalid++
		}
		_ = file.Close()
	}
	return result, nil
}

func readGovernanceBundles(s Store) ([]ReviewBundle, int, error) {
	dir := filepath.Join(s.Root, ".pose", "review-bundles")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []ReviewBundle{}, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("pose governance stats: read bundles: %w", err)
	}
	valid := []ReviewBundle{}
	invalid := 0
	seen := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if seen >= governanceMaxEntries {
			break
		}
		seen++
		id := strings.TrimSuffix(entry.Name(), ".json")
		bundle, loadErr := s.LoadReviewBundle(id)
		if loadErr != nil {
			invalid++
			continue
		}
		valid = append(valid, bundle)
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].BundleID < valid[j].BundleID })
	return valid, invalid, nil
}

func readGovernanceAttestations(s Store) ([]ReviewAttestation, int, error) {
	dir := filepath.Join(s.Root, ".pose", "review-attestations")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []ReviewAttestation{}, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("pose governance stats: read attestations: %w", err)
	}
	valid := []ReviewAttestation{}
	invalid := 0
	seen := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if seen >= governanceMaxEntries {
			break
		}
		seen++
		id := strings.TrimSuffix(entry.Name(), ".json")
		att, loadErr := s.LoadReviewAttestation(id)
		if loadErr != nil {
			invalid++
			continue
		}
		valid = append(valid, att)
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].AttestedAt == valid[j].AttestedAt {
			return valid[i].AttestationID < valid[j].AttestationID
		}
		return valid[i].AttestedAt < valid[j].AttestedAt
	})
	return valid, invalid, nil
}
