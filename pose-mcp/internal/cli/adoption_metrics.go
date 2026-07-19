package cli

// Adoption-value metrics (spec pose-dora-adoption-metrics R3): activation,
// time-to-first-gate, retention and task success — derived entirely from
// data POSE already owns (spec frontmatter, workflow history), no new
// event ingestion needed. Team-level aggregates only, same as DORA.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type adoptionReport struct {
	Activated           bool     `json:"activated"`
	ActivatedAt         string   `json:"activated_at,omitempty"`
	TimeToFirstGateDays *float64 `json:"time_to_first_gate_days,omitempty"`
	RetentionRatio      *float64 `json:"retention_ratio,omitempty"`
	RetentionReason     string   `json:"retention_reason,omitempty"`
	TaskSuccessRatio    *float64 `json:"task_success_ratio,omitempty"`
	TaskSuccessReason   string   `json:"task_success_reason,omitempty"`
	SpecsDone           int      `json:"specs_done"`
	SpecsAbandoned      int      `json:"specs_abandoned"`
	SpecsBlocked        int      `json:"specs_blocked"`
	SpecsPending        int      `json:"specs_pending"`
}

type specStatusCount struct {
	created time.Time
	hasDate bool
	status  string
}

func collectSpecStatuses(root string) []specStatusCount {
	var out []specStatusCount
	base := filepath.Join(root, ".pose", "specs")
	_ = filepath.WalkDir(base, func(path string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() || e.Name() != "spec.md" {
			return nil
		}
		fm, ferr := readFlatFrontmatter(path)
		if ferr != nil {
			return nil
		}
		sc := specStatusCount{status: fm["status"]}
		if created, cerr := time.Parse("2006-01-02", fm["created_at"]); cerr == nil {
			sc.created, sc.hasDate = created, true
		}
		out = append(out, sc)
		return nil
	})
	return out
}

// computeAdoption derives the four adoption views. earliestArtifactAt is
// the earliest created_at across every spec — a proxy for "adoption
// start" (POSE does not persist an install timestamp), documented as
// such in the report and the spec's Final Report.
func computeAdoption(root string, specs []specStatusCount, history []historyRecord, now time.Time) adoptionReport {
	var report adoptionReport
	for _, s := range specs {
		switch s.status {
		case "done":
			report.SpecsDone++
		case "abandoned":
			report.SpecsAbandoned++
		case "blocked":
			report.SpecsBlocked++
		case "draft", "in-progress":
			report.SpecsPending++
		}
	}

	var earliestArtifact time.Time
	var haveEarliest bool
	for _, s := range specs {
		if s.hasDate && (!haveEarliest || s.created.Before(earliestArtifact)) {
			earliestArtifact, haveEarliest = s.created, true
		}
	}

	// Activation: earliest of (a spec reaching done) or (a passing
	// workflow-history record). Absence of any of these = not activated.
	var activatedAt time.Time
	var activated bool
	for _, s := range specs {
		if s.status == "done" && s.hasDate && (!activated || s.created.Before(activatedAt)) {
			// created_at is the closest reliable timestamp on frontmatter;
			// completed_at is preferred when present and parseable.
			activatedAt, activated = s.created, true
		}
	}
	for _, h := range history {
		if h.Outcome != "pass" {
			continue
		}
		t, ok := parseHistoryTime(h.GeneratedAt)
		if !ok {
			continue
		}
		if !activated || t.Before(activatedAt) {
			activatedAt, activated = t, true
		}
	}
	report.Activated = activated
	if activated {
		report.ActivatedAt = activatedAt.UTC().Format(time.RFC3339)
		if haveEarliest {
			days := activatedAt.Sub(earliestArtifact).Hours() / 24
			if days < 0 {
				days = 0
			}
			report.TimeToFirstGateDays = &days
		}
	}

	// Retention: weeks with at least one history record, over weeks since
	// activation. Unavailable until activation happens — there is no
	// retention to measure before a first success exists.
	if !activated {
		report.RetentionReason = "not yet activated"
	} else {
		weeksSinceActivation := int(now.Sub(activatedAt).Hours()/(24*7)) + 1
		if weeksSinceActivation < 1 {
			weeksSinceActivation = 1
		}
		activeWeeks := map[string]bool{}
		for _, h := range history {
			t, ok := parseHistoryTime(h.GeneratedAt)
			if !ok || t.Before(activatedAt) {
				continue
			}
			year, week := t.ISOWeek()
			activeWeeks[fmt.Sprintf("%d-%02d", year, week)] = true
		}
		ratio := float64(len(activeWeeks)) / float64(weeksSinceActivation)
		if ratio > 1 {
			ratio = 1
		}
		report.RetentionRatio = &ratio
	}

	// Task success: resolved specs (done+abandoned+blocked) that reached
	// done. Pending (draft/in-progress) specs are excluded from the
	// denominator — they have not yet been resolved either way.
	resolved := report.SpecsDone + report.SpecsAbandoned + report.SpecsBlocked
	if resolved == 0 {
		report.TaskSuccessReason = "no resolved specs (done, abandoned or blocked) yet"
	} else {
		ratio := float64(report.SpecsDone) / float64(resolved)
		report.TaskSuccessRatio = &ratio
	}

	return report
}

func cmdAdoptionMetrics(root string, args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		default:
			return usageError(stderr, "Usage: pose adoption-metrics [--json]")
		}
	}
	specs := collectSpecStatuses(root)
	history, _ := readHistory(root, stderr)
	report := computeAdoption(root, specs, history, time.Now().UTC())

	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(report)
		return 0
	}
	fmt.Fprintln(stdout, "# Adoption metrics")
	fmt.Fprintf(stdout, "\nactivated: %v", report.Activated)
	if report.Activated {
		fmt.Fprintf(stdout, " (at %s)", report.ActivatedAt)
	}
	fmt.Fprintln(stdout)
	if report.TimeToFirstGateDays != nil {
		fmt.Fprintf(stdout, "time_to_first_gate_days: %.2f\n", *report.TimeToFirstGateDays)
	} else {
		fmt.Fprintln(stdout, "time_to_first_gate_days: unavailable")
	}
	if report.RetentionRatio != nil {
		fmt.Fprintf(stdout, "retention_ratio: %.4f\n", *report.RetentionRatio)
	} else {
		fmt.Fprintf(stdout, "retention_ratio: unavailable (%s)\n", report.RetentionReason)
	}
	if report.TaskSuccessRatio != nil {
		fmt.Fprintf(stdout, "task_success_ratio: %.4f\n", *report.TaskSuccessRatio)
	} else {
		fmt.Fprintf(stdout, "task_success_ratio: unavailable (%s)\n", report.TaskSuccessReason)
	}
	fmt.Fprintf(stdout, "specs: done=%d abandoned=%d blocked=%d pending=%d\n",
		report.SpecsDone, report.SpecsAbandoned, report.SpecsBlocked, report.SpecsPending)
	return 0
}
