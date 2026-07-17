package pose

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Report represents a single execution run history record from POSE validation.
type Report struct {
	GeneratedAt       string          `json:"generated_at"`
	Sequence          int             `json:"sequence"`
	Task              string          `json:"task"`
	TaskSlug          string          `json:"task_slug"`
	ReportType        string          `json:"report_type"`
	Spec              string          `json:"spec"`
	Workflow          string          `json:"workflow"`
	Rules             string          `json:"rules"`
	ValidationProfile string          `json:"validation_profile"`
	Context           string          `json:"context"`
	Risk              string          `json:"risk"`
	Outcome           string          `json:"outcome"`
	OutcomeSource     string          `json:"outcome_source"`
	StableHash        string          `json:"stable_hash"`
	ReportPath        string          `json:"report_path"`
	Filename          string          `json:"filename,omitempty"`      // Base filename derived from report_path
	Body              string          `json:"body,omitempty"`          // The raw markdown content, loaded on demand
	Retrospective     json.RawMessage `json:"retrospective,omitempty"` // Structured companion for retrospective reports.
}

func (s Store) reportsHistoryDir() string {
	return filepath.Join(s.Root, ".pose", "reports", "history")
}

// ListReports returns all historical reports from .pose/reports/history/*.jsonl,
// sorted descending by GeneratedAt so the newest records appear first.
func (s Store) ListReports() ([]Report, error) {
	dir := s.reportsHistoryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Report{}, nil
		}
		return nil, fmt.Errorf("pose: reading reports history dir: %w", err)
	}

	var reports []Report
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		file, err := os.Open(path)
		if err != nil {
			continue // skip unreadable files
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var r Report
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				continue // skip malformed JSON-RPC or JSON lines
			}
			if r.ReportPath != "" {
				r.Filename = filepath.Base(r.ReportPath)
			}
			reports = append(reports, r)
		}
		file.Close()
	}

	// Sort descending by generated_at
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].GeneratedAt > reports[j].GeneratedAt
	})

	return reports, nil
}

// GetReport retrieves the full markdown content of a report from .pose/reports/.
// Filename parameter must be safe from path traversal and restricted to *.md.
func (s Store) GetReport(filename string) (*Report, error) {
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return nil, fmt.Errorf("pose: invalid report filename %q", filename)
	}
	if !strings.HasSuffix(filename, ".md") {
		return nil, fmt.Errorf("pose: invalid report file type, only .md is allowed")
	}

	path := filepath.Clean(filepath.Join(s.Root, ".pose", "reports", filename))

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("pose: report %q not found", filename)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("pose: path %q is a directory", filename)
	}

	bodyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: reading report file: %w", err)
	}

	return &Report{
		Filename: filename,
		Body:     string(bodyBytes),
	}, nil
}
