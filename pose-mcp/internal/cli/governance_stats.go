package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

// cmdGovernanceStats is intentionally separate from the historical `stats`
// grouping command. The two reports answer different questions and must not
// accidentally acquire a shared pass-rate or quality score.
func cmdGovernanceStats(root string, args []string, stdout, stderr io.Writer) int {
	query := posepkg.GovernanceOutcomesQuery{}
	jsonOut, waits, rework := false, false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--waits":
			waits = true
		case "--rework":
			rework = true
		case "--since-days", "--maturity-days", "--min-sample", "--band", "--report-type":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--band baseline|elevated|critical|unknown] [--report-type standard|doc-audit|unknown] [--waits] [--rework] [--json]")
			}
			value := args[i+1]
			switch args[i] {
			case "--band":
				query.Band = value
			case "--report-type":
				query.ReportType = value
			default:
				n, err := strconv.Atoi(value)
				if err != nil || n < 0 {
					return usageError(stderr, "pose stats governance: numeric options must be integers >= 0")
				}
				switch args[i] {
				case "--since-days":
					query.SinceDays = n
				case "--maturity-days":
					query.MaturityDays = n
				case "--min-sample":
					query.MinSample = n
				}
			}
			i++
		default:
			return usageError(stderr, "Usage: pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--band baseline|elevated|critical|unknown] [--report-type standard|doc-audit|unknown] [--waits] [--rework] [--json]")
		}
	}
	report, err := (posepkg.Store{Root: root}).GovernanceOutcomes(query)
	if err != nil {
		render(stdout, stderr).Failure(err.Error())
		return 1
	}
	// Waits and rework are separate dimensions (spec
	// pose-governance-wait-rework-observability); they are reported beside
	// the outcomes, never folded into a score.
	var waitReport *posepkg.GovernanceWaitReport
	var reworkReport *posepkg.GovernanceReworkReport
	if waits {
		w, err := (posepkg.Store{Root: root}).GovernanceWaits(time.Now().UTC())
		if err != nil {
			render(stdout, stderr).Failure(err.Error())
			return 1
		}
		waitReport = &w
	}
	if rework {
		r, err := (posepkg.Store{Root: root}).GovernanceRework()
		if err != nil {
			render(stdout, stderr).Failure(err.Error())
			return 1
		}
		reworkReport = &r
	}
	if jsonOut && (waits || rework) {
		return writeJSON(stdout, struct {
			*posepkg.GovernanceOutcomesReport
			Waits  *posepkg.GovernanceWaitReport   `json:"waits,omitempty"`
			Rework *posepkg.GovernanceReworkReport `json:"rework,omitempty"`
		}{report, waitReport, reworkReport})
	}
	if jsonOut {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			render(stdout, stderr).Failure(err.Error())
			return 1
		}
		return 0
	}
	out := render(stdout, stderr)
	out.Section("# POSE governance outcomes")
	out.Field("available", strconv.FormatBool(report.Coverage.Available))
	if report.Coverage.Reason != "" {
		out.Field("coverage.reason", report.Coverage.Reason)
	}
	out.Field("project_id", report.ProjectID)
	out.Field("schema_version", strconv.Itoa(report.SchemaVersion))
	out.Field("filters.band", governanceFilterLabel(report.Filters.Band))
	out.Field("filters.report_type", governanceFilterLabel(report.Filters.ReportType))
	out.Field("filters.band_applies_to", strings.Join(report.Filters.BandAppliesTo, ","))
	out.Field("filters.report_type_applies_to", strings.Join(report.Filters.ReportTypeAppliesTo, ","))
	out.Field("coverage.history", fmt.Sprintf("matched=%d scanned=%d invalid=%d", report.Coverage.HistoryRecordsMatched, report.Coverage.HistoryRecordsScanned, report.Coverage.HistoryInvalidRecords))
	out.Field("coverage.reviews", fmt.Sprintf("bundles=%d invalid=%d attestations=%d invalid=%d without_bundle=%d", report.Coverage.ReviewBundlesScanned-report.Coverage.ReviewBundlesInvalid, report.Coverage.ReviewBundlesInvalid, report.Coverage.AttestationsScanned-report.Coverage.AttestationsInvalid, report.Coverage.AttestationsInvalid, report.Coverage.AttestationsWithoutBundle))
	out.Field("attempts", fmt.Sprintf("units=%d observed=%d pass=%d fail=%d partial=%d skipped=%d unknown=%d", report.Attempts.UnitsObserved, report.Attempts.AttemptsObserved, report.Attempts.Pass, report.Attempts.Fail, report.Attempts.Partial, report.Attempts.Skipped, report.Attempts.Unknown))
	out.Field("judgment", fmt.Sprintf("attempts=%d first_approved_units=%d approved=%d approved_with_reservations=%d changes_requested=%d rejected=%d unknown=%d", report.Attempts.JudgmentAttempts, report.Attempts.FirstApprovedUnits, report.Reviews.Approved, report.Reviews.ApprovedWithReservations, report.Reviews.ChangesRequested, report.Reviews.Rejected, report.Reviews.UnknownDecisions))
	out.Field("intervention", fmt.Sprintf("observed=%d findings=%d not_applicable=%d accepted_risk=%d resolved=%d open=%d wont_fix=%d", report.Reviews.InterventionsObserved, report.Reviews.FindingsObserved, report.Reviews.NotApplicable, report.Reviews.AcceptedRisk, report.Reviews.ResolvedFindings, report.Reviews.OpenFindings, report.Reviews.WontFixFindings))
	out.Field("freshness", fmt.Sprintf("stale=%d unknown=%d checks=%d truncated=%v", report.Reviews.StaleObserved, report.Reviews.StaleUnknown, report.Coverage.FreshnessChecks, report.Coverage.FreshnessTruncated))
	out.Field("telemetry", fmt.Sprintf("active_duration_observed=%d active_duration_seconds=%.2f active_duration_unknown=%d wait_duration_observed=%d wait_duration_unknown=%d cost_observed=%d cost_usd=%.2f cost_unknown=%d", report.Attempts.ActiveDurationObserved, report.Attempts.ActiveDurationSeconds, report.Attempts.ActiveDurationUnknown, report.Attempts.WaitDurationObserved, report.Attempts.WaitDurationUnknown, report.Attempts.ObservedCostCount, report.Attempts.ObservedCostUSD, report.Attempts.UnknownCostCount))
	remediation := fmt.Sprintf("available=%v coverage=%s", report.Remediation.Available, report.Coverage.RemediationCoverage)
	if report.Remediation.Reason != "" {
		remediation += " reason=" + strings.ReplaceAll(report.Remediation.Reason, "\n", " ")
	}
	out.Field("remediation", remediation)
	if report.Remediation.Available {
		// The denominator and what was excluded from it are printed beside the
		// rate, never under a flag: a remediation count without its censored and
		// unlinked populations reads as a quality number, and it is not one.
		out.Field("remediation.population", fmt.Sprintf("mature=%d censored=%d linked=%d unlinked_unknown=%d links_declared=%d invalid_links=%d insufficient_sample=%v",
			report.Remediation.MaturePopulation, report.Remediation.Censored, report.Remediation.LinkedObserved,
			report.Remediation.UnlinkedUnknown, report.Remediation.LinksDeclared, report.Remediation.InvalidLinks,
			report.Remediation.InsufficientSample))
		out.Field("remediation.observed", fmt.Sprintf("remediated=%d counted_categories=%s",
			report.Remediation.ObservedRemediated, strings.Join(report.Remediation.CountedCategories, ",")))
		for _, category := range sortedStringKeys(report.Remediation.ByCategory) {
			out.Field("remediation.category."+category, strconv.Itoa(report.Remediation.ByCategory[category]))
		}
	}
	for _, link := range report.Remediation.Links {
		out.Field("remediation.link", fmt.Sprintf("finding:%s/%s -> spec:%s category=%s band=%s", link.AttestationID, link.FindingID, link.RemediationSpec, link.Category, link.SourceBand))
	}
	if waitReport != nil {
		out.Field("waits.requests", fmt.Sprintf("total=%d resolved=%d open=%d unknown=%d", waitReport.Requests, waitReport.Resolved, waitReport.Open, waitReport.Unknown))
		out.Field("waits.age_seconds", fmt.Sprintf("count=%d median=%.0f max=%.0f", waitReport.AgeSeconds.Count, waitReport.AgeSeconds.Median, waitReport.AgeSeconds.Max))
		for _, cause := range sortedFloatKeys(waitReport.ByCause) {
			out.Field("waits.attributed."+cause, fmt.Sprintf("%.0f seconds", waitReport.ByCause[cause]))
		}
		out.Field("waits.known_blocking_seconds", fmt.Sprintf("%.0f (union per spec of whole-spec execution restrictions)", waitReport.KnownBlockingSeconds))
		for _, l := range waitReport.Limitations {
			out.Field("waits.limitation", l)
		}
	}
	if reworkReport != nil {
		out.Field("rework.population", fmt.Sprintf("scopes=%d bundles=%d attestations=%d", reworkReport.Scopes, reworkReport.Bundles, reworkReport.Attestations))
		for _, cause := range sortedStringKeys(reworkReport.BundleSupersessions) {
			out.Field("rework.bundle."+cause, strconv.Itoa(reworkReport.BundleSupersessions[cause]))
		}
		for _, cause := range sortedStringKeys(reworkReport.AttestationRework) {
			out.Field("rework.attestation."+cause, strconv.Itoa(reworkReport.AttestationRework[cause]))
		}
		for _, l := range reworkReport.Limitations {
			out.Field("rework.limitation", l)
		}
	}
	return 0
}

func sortedFloatKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func governanceFilterLabel(value string) string {
	if value == "" {
		return "all"
	}
	return value
}

// sortedStringKeys keeps the category lines deterministic, because this output
// is quoted into specs and diffed.
func sortedStringKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
