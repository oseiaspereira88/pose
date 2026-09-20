package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

// cmdGovernanceStats is intentionally separate from the historical `stats`
// grouping command. The two reports answer different questions and must not
// accidentally acquire a shared pass-rate or quality score.
func cmdGovernanceStats(root string, args []string, stdout, stderr io.Writer) int {
	query := posepkg.GovernanceOutcomesQuery{}
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--since-days", "--maturity-days", "--min-sample":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--json]")
			}
			n, err := strconv.Atoi(args[i+1])
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
			i++
		default:
			return usageError(stderr, "Usage: pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--json]")
		}
	}
	report, err := (posepkg.Store{Root: root}).GovernanceOutcomes(query)
	if err != nil {
		render(stdout, stderr).Failure(err.Error())
		return 1
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
	return 0
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
