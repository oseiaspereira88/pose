package cli

// DORA metric calculation (spec pose-dora-adoption-metrics R2): the five
// current DORA metrics (dora.dev), each computed only when its own
// denominator has real data — "unavailable" (never a fabricated zero) is
// the explicit third state alongside a numeric value. Team/application
// scoped only; the event schema has no per-identity field to rank by.

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

type doraMetric struct {
	Name       string   `json:"name"`
	Value      *float64 `json:"value,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Available  bool     `json:"available"`
	Reason     string   `json:"reason,omitempty"` // populated when !Available
	SampleSize int      `json:"sample_size"`
}

type doraReport struct {
	Application string       `json:"application,omitempty"`
	WindowDays  int          `json:"window_days"`
	Metrics     []doraMetric `json:"metrics"`
}

func unavailable(name, reason string) doraMetric {
	return doraMetric{Name: name, Available: false, Reason: reason}
}

func available(name string, value float64, unit string, n int) doraMetric {
	return doraMetric{Name: name, Value: &value, Unit: unit, Available: true, SampleSize: n}
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func computeDORA(deployments []deploymentEvent, incidents []incidentEvent, application string, windowDays int) doraReport {
	now := time.Now().UTC()
	windowStart := now.AddDate(0, 0, -windowDays)

	var deploysInWindow []deploymentEvent
	for _, d := range deployments {
		if application != "" && d.Application != application {
			continue
		}
		t, err := parseEventTime(d.DeployedAt)
		if err != nil || t.Before(windowStart) || t.After(now) {
			continue
		}
		deploysInWindow = append(deploysInWindow, d)
	}
	var incidentsInWindow []incidentEvent
	for _, in := range incidents {
		if application != "" && in.Application != application {
			continue
		}
		t, err := parseEventTime(in.StartedAt)
		if err != nil || t.Before(windowStart) || t.After(now) {
			continue
		}
		incidentsInWindow = append(incidentsInWindow, in)
	}

	report := doraReport{Application: application, WindowDays: windowDays}

	// 1. Deployment Frequency (successful deployments / day).
	if len(deploysInWindow) == 0 {
		report.Metrics = append(report.Metrics, unavailable("deployment_frequency", "no deployment events recorded in window"))
	} else {
		success := 0
		for _, d := range deploysInWindow {
			if d.Status == "success" {
				success++
			}
		}
		rate := float64(success) / float64(windowDays)
		report.Metrics = append(report.Metrics, available("deployment_frequency", rate, "deployments/day", len(deploysInWindow)))
	}

	// 2. Lead Time for Changes (median, successful deployments only).
	var leadTimes []float64
	for _, d := range deploysInWindow {
		if d.Status == "success" && d.LeadTimeSeconds != nil {
			leadTimes = append(leadTimes, *d.LeadTimeSeconds)
		}
	}
	if len(leadTimes) == 0 {
		report.Metrics = append(report.Metrics, unavailable("lead_time_for_changes", "no successful deployment in window reported lead_time_seconds"))
	} else {
		report.Metrics = append(report.Metrics, available("lead_time_for_changes", median(leadTimes), "seconds", len(leadTimes)))
	}

	// 3. Change Failure Rate.
	if len(deploysInWindow) == 0 {
		report.Metrics = append(report.Metrics, unavailable("change_failure_rate", "no deployment events recorded in window"))
	} else {
		failed := 0
		for _, d := range deploysInWindow {
			if d.Status == "failure" {
				failed++
			}
		}
		rate := float64(failed) / float64(len(deploysInWindow))
		report.Metrics = append(report.Metrics, available("change_failure_rate", rate, "ratio", len(deploysInWindow)))
	}

	// 4. Failed Deployment Recovery Time (MTTR, resolved incidents only).
	var recoveryTimes []float64
	for _, in := range incidentsInWindow {
		if in.ResolvedAt == "" {
			continue
		}
		started, e1 := parseEventTime(in.StartedAt)
		resolved, e2 := parseEventTime(in.ResolvedAt)
		if e1 != nil || e2 != nil {
			continue
		}
		recoveryTimes = append(recoveryTimes, resolved.Sub(started).Seconds())
	}
	if len(recoveryTimes) == 0 {
		report.Metrics = append(report.Metrics, unavailable("failed_deployment_recovery_time", "no resolved incidents in window"))
	} else {
		report.Metrics = append(report.Metrics, available("failed_deployment_recovery_time", median(recoveryTimes), "seconds", len(recoveryTimes)))
	}

	// 5. Reliability (documented proxy: % of window days with no ongoing
	// major/critical incident). Unavailable only when there is genuinely
	// no data at all for the scope — zero incidents with deployment
	// activity present is a legitimate, not fabricated, 100%.
	if len(deploysInWindow) == 0 && len(incidentsInWindow) == 0 {
		report.Metrics = append(report.Metrics, unavailable("reliability", "no deployment or incident events recorded in window"))
	} else if windowDays <= 0 {
		report.Metrics = append(report.Metrics, unavailable("reliability", "window_days must be positive"))
	} else {
		reliableDays := 0
		for day := 0; day < windowDays; day++ {
			dayStart := windowStart.AddDate(0, 0, day)
			dayEnd := dayStart.AddDate(0, 0, 1)
			ongoingMajorIncident := false
			for _, in := range incidentsInWindow {
				if in.Severity != "major" && in.Severity != "critical" {
					continue
				}
				started, e1 := parseEventTime(in.StartedAt)
				if e1 != nil || !started.Before(dayEnd) {
					continue
				}
				resolvedAt := now
				if in.ResolvedAt != "" {
					if r, e2 := parseEventTime(in.ResolvedAt); e2 == nil {
						resolvedAt = r
					}
				}
				if resolvedAt.After(dayStart) {
					ongoingMajorIncident = true
					break
				}
			}
			if !ongoingMajorIncident {
				reliableDays++
			}
		}
		pct := float64(reliableDays) / float64(windowDays) * 100
		report.Metrics = append(report.Metrics, available("reliability", pct, "percent_days_without_major_incident", windowDays))
	}

	return report
}

func cmdDORAMetrics(root string, args []string, stdout, stderr io.Writer) int {
	application := ""
	windowDays := 30
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--application":
			if i+1 >= len(args) {
				return usageError(stderr, "pose dora-metrics: --application requires a value")
			}
			i++
			application = args[i]
		case "--window-days":
			if i+1 >= len(args) {
				return usageError(stderr, "pose dora-metrics: --window-days requires a value")
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return usageError(stderr, "pose dora-metrics: --window-days must be a positive integer")
			}
			windowDays = n
		case "--json":
			jsonOut = true
		default:
			return usageError(stderr, "Usage: pose dora-metrics [--application A] [--window-days N] [--json]")
		}
	}
	deployments, invalidD := readDeploymentEvents(root, stderr)
	incidents, invalidI := readIncidentEvents(root, stderr)
	report := computeDORA(deployments, incidents, application, windowDays)

	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(report)
		return 0
	}
	fmt.Fprintf(stdout, "# DORA metrics (window: %d days)\n\n", windowDays)
	if application != "" {
		fmt.Fprintf(stdout, "application: %s\n\n", application)
	}
	for _, m := range report.Metrics {
		if !m.Available {
			fmt.Fprintf(stdout, "%-32s unavailable (%s)\n", m.Name, m.Reason)
			continue
		}
		fmt.Fprintf(stdout, "%-32s %.4f %s (n=%d)\n", m.Name, *m.Value, m.Unit, m.SampleSize)
	}
	fmt.Fprintf(stdout, "\ndora.events_deployments_invalid=%d\ndora.events_incidents_invalid=%d\n", invalidD, invalidI)
	return 0
}
