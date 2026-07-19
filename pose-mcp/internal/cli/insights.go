package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

type historyRecord struct {
	GeneratedAt string `json:"generated_at"`
	Outcome     string `json:"outcome"`
	Workflow    string `json:"workflow"`
	TaskSlug    string `json:"task_slug"`
	Context     string `json:"context"`
	ReportType  string `json:"report_type"`
}

func readHistory(root string, stderr io.Writer) ([]historyRecord, int) {
	dir := filepath.Join(root, ".pose", "reports", "history")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}
	var records []historyRecord
	invalid := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(stderr, "[WARN] %v\n", err)
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			var r historyRecord
			if strings.TrimSpace(s.Text()) == "" {
				continue
			}
			if json.Unmarshal([]byte(s.Text()), &r) != nil {
				invalid++
				continue
			}
			records = append(records, r)
		}
		_ = f.Close()
	}
	return records, invalid
}

func parseHistoryTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func cmdRecurrenceCheck(root string, args []string, stdout, stderr io.Writer) int {
	mode, days, threshold, includePass := "strict", 14, 3, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--include-pass":
			includePass = true
		case "--window-days", "--threshold":
			if i+1 >= len(args) {
				return usageError(stderr, "pose recurrence-check: value required")
			}
			n, e := strconv.Atoi(args[i+1])
			if e != nil || n < 1 {
				return usageError(stderr, "pose recurrence-check: expected integer > 0")
			}
			if args[i] == "--window-days" {
				days = n
			} else {
				threshold = n
			}
			i++
		default:
			return usageError(stderr, "Usage: pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold N] [--include-pass]")
		}
	}
	records, _ := readHistory(root, stderr)
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	buckets := map[string][]historyRecord{}
	for _, r := range records {
		t, ok := parseHistoryTime(r.GeneratedAt)
		if !ok || t.Before(cutoff) || (!includePass && r.Outcome == "pass") {
			continue
		}
		task := r.TaskSlug
		if task == "" {
			task = "<unknown>"
		}
		typ := r.ReportType
		if typ == "" {
			typ = "standard"
		}
		key := task + "\x00" + typ
		buckets[key] = append(buckets[key], r)
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	flagged := 0
	for _, k := range keys {
		rs := buckets[k]
		if len(rs) < threshold {
			continue
		}
		flagged++
		parts := strings.Split(k, "\x00")
		counts := map[string]int{}
		latest := ""
		workflow := ""
		for _, r := range rs {
			o := r.Outcome
			if o == "" {
				o = "unknown"
			}
			counts[o]++
			if r.GeneratedAt > latest {
				latest = r.GeneratedAt
			}
			if r.Workflow != "" {
				workflow = r.Workflow
			}
		}
		names := make([]string, 0, len(counts))
		for n := range counts {
			names = append(names, n)
		}
		sort.Strings(names)
		summary := []string{}
		for _, n := range names {
			summary = append(summary, fmt.Sprintf("%s=%d", n, counts[n]))
		}
		fmt.Fprintf(stderr, "[RECURRENT] %s (%s): %d runs in %dd; outcomes=%s; latest=%s", parts[0], parts[1], len(rs), days, strings.Join(summary, ", "), latest)
		if workflow != "" {
			fmt.Fprintf(stderr, "; workflow=%s", workflow)
		}
		fmt.Fprintln(stderr)
	}
	fmt.Fprintf(stdout, "recurrence.window_days=%d\nrecurrence.threshold=%d\nrecurrence.records_scanned=%d\nrecurrence.flagged_keys=%d\n", days, threshold, len(records), flagged)
	if flagged > 0 && mode == "strict" {
		return 1
	}
	return 0
}

type statRow = posepkg.InsightRow

func cmdStats(root string, args []string, stdout, stderr io.Writer) int {
	by, since, jsonOut, htmlOut, out := "workflow", 0, false, false, ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "workflows":
			by = "workflow"
		case "tasks":
			by = "task"
		case "contexts":
			by = "context"
		case "outcomes":
		default:
			return usageError(stderr, "Usage: pose stats [workflows|tasks|contexts] [--since-days N] [--json|--html [--out file]]")
		}
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--by":
			if i+1 >= len(args) {
				return 2
			}
			i++
			by = args[i]
		case "--since-days":
			if i+1 >= len(args) {
				return 2
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 0 {
				return 2
			}
			since = n
		case "--json":
			jsonOut = true
		case "--html":
			htmlOut = true
		case "--out":
			if i+1 >= len(args) {
				return 2
			}
			i++
			out = args[i]
		default:
			return usageError(stderr, "pose stats: invalid argument: "+args[i])
		}
	}
	if by != "workflow" && by != "task" && by != "context" {
		return usageError(stderr, "pose stats: invalid grouping")
	}
	result, err := (posepkg.Store{Root: root}).Insights(by, since)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rows := result.Rows
	if htmlOut {
		if out == "" {
			out = filepath.Join(root, ".pose", "reports", "pose-stats.html")
		}
		if !confinedOutput(root, out) {
			return usageError(stderr, "pose stats: --out must remain inside project")
		}
		workflows, err := (posepkg.Store{Root: root}).Insights("workflow", since)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		tasks, err := (posepkg.Store{Root: root}).Insights("task", since)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		content := renderStatsHTML(workflows.Rows, tasks.Rows, result.RecordsScanned, result.RecordsSkippedInvalid, collectSpecInsights(root))
		if err := writeAtomic(out, []byte(content), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "stats.html=%s\n", out)
		return 0
	}
	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(result)
		return 0
	}
	fmt.Fprintf(stdout, "# Stats by %s\n\n", by)
	if len(rows) == 0 {
		fmt.Fprintf(stdout, "_No records grouped by %s._\n", by)
	} else {
		fmt.Fprintln(stdout, "KEY | PASS | FAIL | PART | SKIP | UNK | TOT | RATE")
		for _, r := range rows {
			rate := "n/a"
			if r.PassRate != nil {
				rate = fmt.Sprintf("%.0f%%", *r.PassRate*100)
			}
			fmt.Fprintf(stdout, "%s | %d | %d | %d | %d | %d | %d | %s\n", r.Key, r.Pass, r.Fail, r.Partial, r.Skipped, r.Unknown, r.Total, rate)
		}
	}
	fmt.Fprintf(stdout, "\nstats.records_scanned=%d\nstats.records_skipped_by_window=%d\nstats.records_skipped_invalid=%d\nstats.groups=%d\n", result.RecordsScanned, result.RecordsSkippedByWindow, result.RecordsSkippedInvalid, len(rows))
	return 0
}

func confinedOutput(root, path string) bool {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, e := filepath.Rel(root, path)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type specInsights struct {
	Completed       int
	AverageLeadDays *float64
	OpenFollowups   int
	OldestOpenDays  *int
}

func collectSpecInsights(root string) specInsights {
	result := specInsights{}
	var leadTotal float64
	base := filepath.Join(root, ".pose", "specs")
	_ = filepath.WalkDir(base, func(path string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() || e.Name() != "spec.md" {
			return nil
		}
		raw, er := os.ReadFile(path)
		if er != nil {
			return nil
		}
		fm, er := readFlatFrontmatter(path)
		if er != nil {
			return nil
		}
		created, cerr := time.Parse("2006-01-02", fm["created_at"])
		completed, derr := time.Parse("2006-01-02", fm["completed_at"])
		if cerr == nil && derr == nil && !completed.Before(created) {
			result.Completed++
			leadTotal += completed.Sub(created).Hours() / 24
		}
		opens := strings.Count(string(raw), "- [open]")
		result.OpenFollowups += opens
		if opens > 0 && cerr == nil {
			days := int(time.Since(created).Hours() / 24)
			if result.OldestOpenDays == nil || days > *result.OldestOpenDays {
				result.OldestOpenDays = &days
			}
		}
		return nil
	})
	if result.Completed > 0 {
		avg := leadTotal / float64(result.Completed)
		result.AverageLeadDays = &avg
	}
	return result
}

func renderStatsHTML(workflows, tasks []statRow, scanned, invalid int, specs specInsights) string {
	var workflowRows, taskRows, recurrenceRows strings.Builder
	for _, r := range workflows {
		fmt.Fprintf(&workflowRows, "<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>", html.EscapeString(r.Key), r.Pass, r.Fail, r.Partial, r.Total)
	}
	for _, r := range tasks {
		fmt.Fprintf(&taskRows, "<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>", html.EscapeString(r.Key), r.Pass, r.Fail, r.Partial, r.Total)
		if r.Total >= 2 {
			fmt.Fprintf(&recurrenceRows, "<tr><td>%s</td><td>%d</td><td>%d</td></tr>", html.EscapeString(r.Key), r.Total, r.Fail+r.Partial)
		}
	}
	lead := "unavailable"
	if specs.AverageLeadDays != nil {
		lead = fmt.Sprintf("%.1f days", *specs.AverageLeadDays)
	}
	oldest := "unavailable"
	if specs.OldestOpenDays != nil {
		oldest = fmt.Sprintf("%d days", *specs.OldestOpenDays)
	}
	return fmt.Sprintf("<!doctype html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'\"><title>POSE local insights</title><style>body{font-family:system-ui;margin:2rem}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.4rem}.cards{display:flex;gap:1rem;flex-wrap:wrap}.card{border:1px solid #ccc;padding:1rem}</style></head><body><h1>POSE local insights</h1><div class=\"cards\"><div class=\"card\">History records: %d</div><div class=\"card\">Invalid skipped: %d</div><div class=\"card\">Open follow-ups: %d</div><div class=\"card\">Oldest open follow-up: %s</div><div class=\"card\">Completed specs: %d</div><div class=\"card\">Average lead time: %s</div></div><h2>Outcomes by workflow</h2><table>%s</table><h2>Outcomes by task</h2><table>%s</table><h2>Recurrence candidates</h2><table>%s</table></body></html>", scanned, invalid, specs.OpenFollowups, oldest, specs.Completed, lead, workflowRows.String(), taskRows.String(), recurrenceRows.String())
}
func usageError(w io.Writer, msg string) int { fmt.Fprintln(w, msg); return 2 }
