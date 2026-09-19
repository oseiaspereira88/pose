package cli

import (
	"encoding/json"
	"fmt"
	"io"
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
	return 0
}
