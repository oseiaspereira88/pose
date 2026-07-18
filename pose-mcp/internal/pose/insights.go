package pose

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// InsightRow is one outcome aggregation group returned by pose stats and MCP.
type InsightRow struct {
	Key      string   `json:"key"`
	Pass     int      `json:"pass"`
	Fail     int      `json:"fail"`
	Partial  int      `json:"partial"`
	Skipped  int      `json:"skipped"`
	Unknown  int      `json:"unknown"`
	Total    int      `json:"total"`
	PassRate *float64 `json:"pass_rate"`
}

// InsightsResult is the stable machine-readable contract shared by the CLI
// and the pose_insights MCP tool.
type InsightsResult struct {
	GroupBy                string       `json:"group_by"`
	SinceDays              int          `json:"since_days"`
	RecordsScanned         int          `json:"records_scanned"`
	RecordsSkippedByWindow int          `json:"records_skipped_by_window"`
	RecordsSkippedInvalid  int          `json:"records_skipped_invalid"`
	Rows                   []InsightRow `json:"rows"`
}

type insightRecord struct {
	GeneratedAt string `json:"generated_at"`
	Outcome     string `json:"outcome"`
	Workflow    string `json:"workflow"`
	TaskSlug    string `json:"task_slug"`
	Context     string `json:"context"`
}

// Insights aggregates local report history without network or shell execution.
func (s Store) Insights(groupBy string, sinceDays int) (*InsightsResult, error) {
	if groupBy == "" {
		groupBy = "workflow"
	}
	if groupBy != "workflow" && groupBy != "task" && groupBy != "context" {
		return nil, fmt.Errorf("pose_insights: invalid group_by %q", groupBy)
	}
	if sinceDays < 0 {
		return nil, fmt.Errorf("pose_insights: since_days must be non-negative")
	}
	records, invalid, err := readInsightHistory(s.Root)
	if err != nil {
		return nil, err
	}
	rows, skipped := aggregateInsightRows(records, groupBy, sinceDays)
	return &InsightsResult{
		GroupBy:                groupBy,
		SinceDays:              sinceDays,
		RecordsScanned:         len(records),
		RecordsSkippedByWindow: skipped,
		RecordsSkippedInvalid:  invalid,
		Rows:                   rows,
	}, nil
}

func readInsightHistory(root string) ([]insightRecord, int, error) {
	dir := filepath.Join(root, ".pose", "reports", "history")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []insightRecord{}, 0, nil
		}
		return nil, 0, fmt.Errorf("pose insights: reading history: %w", err)
	}
	records := []insightRecord{}
	invalid := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := os.Open(path)
		if err != nil {
			return nil, 0, fmt.Errorf("pose insights: opening %s: %w", entry.Name(), err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var record insightRecord
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				invalid++
				continue
			}
			records = append(records, record)
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return nil, 0, fmt.Errorf("pose insights: scanning %s: %w", entry.Name(), scanErr)
		}
		if closeErr != nil {
			return nil, 0, fmt.Errorf("pose insights: closing %s: %w", entry.Name(), closeErr)
		}
	}
	return records, invalid, nil
}

func aggregateInsightRows(records []insightRecord, groupBy string, sinceDays int) ([]InsightRow, int) {
	cutoff := time.Time{}
	if sinceDays > 0 {
		cutoff = time.Now().UTC().AddDate(0, 0, -sinceDays)
	}
	skipped := 0
	buckets := map[string]*InsightRow{}
	for _, record := range records {
		if !cutoff.IsZero() {
			generatedAt, ok := parseInsightTime(record.GeneratedAt)
			if !ok || generatedAt.Before(cutoff) {
				skipped++
				continue
			}
		}
		key := record.Workflow
		switch groupBy {
		case "task":
			key = record.TaskSlug
		case "context":
			key = record.Context
		}
		if key == "" {
			key = "_unset_"
		}
		row := buckets[key]
		if row == nil {
			row = &InsightRow{Key: key}
			buckets[key] = row
		}
		switch record.Outcome {
		case "pass":
			row.Pass++
		case "fail":
			row.Fail++
		case "partial":
			row.Partial++
		case "skipped":
			row.Skipped++
		default:
			row.Unknown++
		}
		row.Total++
	}
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]InsightRow, 0, len(keys))
	for _, key := range keys {
		row := buckets[key]
		graded := row.Total - row.Unknown - row.Skipped
		if graded > 0 {
			rate := float64(row.Pass) / float64(graded)
			row.PassRate = &rate
		}
		rows = append(rows, *row)
	}
	return rows, skipped
}

func parseInsightTime(value string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
