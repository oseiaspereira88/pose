package cli

// DORA event ingestion (spec pose-dora-adoption-metrics): explicit
// deployment and incident events, never inferred from commits (Non-goal).
// Storage is append-only JSONL, one file per calendar month, mirroring
// the .pose/reports/history/ convention. Security: the event schema has
// no identity field beyond "application" and "source" — no author, no
// email, no principal — individual ranking is structurally impossible
// from this data (Constraint: DORA metrics are team/application outcomes,
// never individual scores).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type deploymentEvent struct {
	RecordedAt      string   `json:"recorded_at"`
	Application     string   `json:"application"`
	Environment     string   `json:"environment"`
	DeployedAt      string   `json:"deployed_at"`
	Status          string   `json:"status"` // success | failure
	LeadTimeSeconds *float64 `json:"lead_time_seconds,omitempty"`
	Source          string   `json:"source"` // quality metadata: who/what reported this (manual|ci|webhook)
	ChangeRef       string   `json:"change_ref,omitempty"`
}

type incidentEvent struct {
	RecordedAt         string `json:"recorded_at"`
	Application        string `json:"application"`
	StartedAt          string `json:"started_at"`
	ResolvedAt         string `json:"resolved_at,omitempty"` // empty = still open
	Severity           string `json:"severity"`              // minor | major | critical
	CausedByDeployment bool   `json:"caused_by_deployment,omitempty"`
	Source             string `json:"source"`
}

var validDeploymentStatus = map[string]bool{"success": true, "failure": true}
var validSeverity = map[string]bool{"minor": true, "major": true, "critical": true}
var validEventSource = map[string]bool{"manual": true, "ci": true, "webhook": true}

func eventsDir(root, kind string) string {
	return filepath.Join(root, ".pose", "events", kind)
}

func monthlyEventPath(root, kind string, t time.Time) string {
	return filepath.Join(eventsDir(root, kind), t.UTC().Format("2006-01")+".jsonl")
}

// appendEvent validates line is valid JSON (already marshaled by caller)
// and appends it atomically-enough for an append-only log: O_APPEND with
// O_SYNC-free single write, matching the existing history JSONL pattern.
func appendEvent(path string, line []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func parseEventTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

func cmdRecordDeployment(root string, args []string, stdout, stderr io.Writer) int {
	var ev deploymentEvent
	var leadTime string
	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			return usageError(stderr, "Usage: pose record-deployment --application A --environment E --status success|failure --source manual|ci|webhook [--deployed-at RFC3339] [--lead-time-seconds N] [--change-ref R]")
		}
		v := args[i+1]
		switch args[i] {
		case "--application":
			ev.Application = v
		case "--environment":
			ev.Environment = v
		case "--status":
			ev.Status = v
		case "--source":
			ev.Source = v
		case "--deployed-at":
			ev.DeployedAt = v
		case "--lead-time-seconds":
			leadTime = v
		case "--change-ref":
			ev.ChangeRef = v
		default:
			return usageError(stderr, "pose record-deployment: unknown flag "+args[i])
		}
		i++
	}
	if ev.Application == "" || ev.Environment == "" {
		fmt.Fprintln(stderr, "pose record-deployment: --application and --environment are required")
		return 2
	}
	if !validDeploymentStatus[ev.Status] {
		fmt.Fprintln(stderr, "pose record-deployment: --status must be success|failure")
		return 2
	}
	if !validEventSource[ev.Source] {
		fmt.Fprintln(stderr, "pose record-deployment: --source must be manual|ci|webhook")
		return 2
	}
	now := time.Now().UTC()
	if ev.DeployedAt == "" {
		ev.DeployedAt = now.Format(time.RFC3339)
	}
	deployedAt, err := parseEventTime(ev.DeployedAt)
	if err != nil {
		fmt.Fprintf(stderr, "pose record-deployment: --deployed-at must be RFC3339: %v\n", err)
		return 2
	}
	if leadTime != "" {
		n, err := strconv.ParseFloat(leadTime, 64)
		if err != nil || n < 0 {
			fmt.Fprintln(stderr, "pose record-deployment: --lead-time-seconds must be a non-negative number")
			return 2
		}
		ev.LeadTimeSeconds = &n
	}
	ev.RecordedAt = now.Format(time.RFC3339)
	line, _ := json.Marshal(ev)
	if err := appendEvent(monthlyEventPath(root, "deployments", deployedAt), line); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "deployment recorded: application=%s environment=%s status=%s\n", ev.Application, ev.Environment, ev.Status)
	return 0
}

func cmdRecordIncident(root string, args []string, stdout, stderr io.Writer) int {
	var ev incidentEvent
	causedByDeployment := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--caused-by-deployment" {
			causedByDeployment = true
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, "Usage: pose record-incident --application A --started-at RFC3339 --severity minor|major|critical --source manual|ci|webhook [--resolved-at RFC3339] [--caused-by-deployment]")
		}
		v := args[i+1]
		switch args[i] {
		case "--application":
			ev.Application = v
		case "--started-at":
			ev.StartedAt = v
		case "--resolved-at":
			ev.ResolvedAt = v
		case "--severity":
			ev.Severity = v
		case "--source":
			ev.Source = v
		default:
			return usageError(stderr, "pose record-incident: unknown flag "+args[i])
		}
		i++
	}
	ev.CausedByDeployment = causedByDeployment
	if ev.Application == "" || ev.StartedAt == "" {
		fmt.Fprintln(stderr, "pose record-incident: --application and --started-at are required")
		return 2
	}
	if !validSeverity[ev.Severity] {
		fmt.Fprintln(stderr, "pose record-incident: --severity must be minor|major|critical")
		return 2
	}
	if !validEventSource[ev.Source] {
		fmt.Fprintln(stderr, "pose record-incident: --source must be manual|ci|webhook")
		return 2
	}
	startedAt, err := parseEventTime(ev.StartedAt)
	if err != nil {
		fmt.Fprintf(stderr, "pose record-incident: --started-at must be RFC3339: %v\n", err)
		return 2
	}
	if ev.ResolvedAt != "" {
		resolvedAt, err := parseEventTime(ev.ResolvedAt)
		if err != nil {
			fmt.Fprintf(stderr, "pose record-incident: --resolved-at must be RFC3339: %v\n", err)
			return 2
		}
		if resolvedAt.Before(startedAt) {
			fmt.Fprintln(stderr, "pose record-incident: --resolved-at cannot precede --started-at")
			return 2
		}
	}
	ev.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	line, _ := json.Marshal(ev)
	if err := appendEvent(monthlyEventPath(root, "incidents", startedAt), line); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "incident recorded: application=%s severity=%s\n", ev.Application, ev.Severity)
	return 0
}

func readDeploymentEvents(root string, stderr io.Writer) ([]deploymentEvent, int) {
	dir := eventsDir(root, "deployments")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}
	var out []deploymentEvent
	invalid := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			if strings.TrimSpace(s.Text()) == "" {
				continue
			}
			var ev deploymentEvent
			if json.Unmarshal([]byte(s.Text()), &ev) != nil {
				invalid++
				continue
			}
			out = append(out, ev)
		}
		_ = f.Close()
	}
	return out, invalid
}

func readIncidentEvents(root string, stderr io.Writer) ([]incidentEvent, int) {
	dir := eventsDir(root, "incidents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}
	var out []incidentEvent
	invalid := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			if strings.TrimSpace(s.Text()) == "" {
				continue
			}
			var ev incidentEvent
			if json.Unmarshal([]byte(s.Text()), &ev) != nil {
				invalid++
				continue
			}
			out = append(out, ev)
		}
		_ = f.Close()
	}
	return out, invalid
}

// cmdEventsHousekeeping supports retention/deletion (Security requirement):
// list monthly event files older than the retention window and optionally
// purge them. Aggregation (the Security requirement's third clause) is
// satisfied structurally — dora-metrics/adoption-metrics never emit a
// per-event or per-identity row, only window aggregates.
func cmdEventsHousekeeping(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose events-housekeeping <list-expired|purge> [--older-than-days N] [--apply]")
	}
	mode := args[0]
	args = args[1:]
	days := 400
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--older-than-days":
			if i+1 >= len(args) {
				return 2
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return 2
			}
			days = n
		case "--apply":
			apply = true
		default:
			return usageError(stderr, "pose events-housekeeping: invalid argument")
		}
	}
	if mode != "list-expired" && mode != "purge" {
		return usageError(stderr, "pose events-housekeeping: invalid command")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	removed := 0
	for _, kind := range []string{"deployments", "incidents"} {
		dir := eventsDir(root, kind)
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			month, err := time.Parse("2006-01", strings.TrimSuffix(e.Name(), ".jsonl"))
			if err != nil || !month.Before(cutoff) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if mode == "list-expired" {
				fmt.Fprintf(stdout, "%s\n", path)
				continue
			}
			if apply {
				if err := os.Remove(path); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				removed++
			} else {
				fmt.Fprintf(stdout, "[DRY-RUN] would remove: %s\n", path)
			}
		}
	}
	if mode == "purge" {
		if apply {
			fmt.Fprintf(stdout, "Result: SUCCESS — %d event file(s) removed.\n", removed)
		} else {
			fmt.Fprintln(stdout, "Result: DRY-RUN — re-run with --apply to remove.")
		}
	}
	return 0
}
