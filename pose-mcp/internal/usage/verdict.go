package usage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const verdictSchemaVersion = 1

type Verdict struct {
	SchemaVersion int    `json:"schema_version"`
	RecordedAt    string `json:"recorded_at"`
	Tool          string `json:"tool"`
	FindingID     string `json:"finding_id"`
	Disposition   string `json:"verdict"`
	Reason        string `json:"reason"`
	By            string `json:"by"`
}

type VerdictInput struct {
	Tool        string
	FindingID   string
	Disposition string
	Reason      string
	By          string
	At          time.Time
}

type AdjudicationRow struct {
	Tool              string  `json:"tool"`
	Valid             int     `json:"valid"`
	WontFix           int     `json:"wont_fix"`
	FalsePositive     int     `json:"false_positive"`
	Unmatched         int     `json:"unmatched"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

func RecordVerdict(root string, input VerdictInput) error {
	at := input.At.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	verdict := Verdict{
		SchemaVersion: verdictSchemaVersion, RecordedAt: at.Format(time.RFC3339Nano),
		Tool: input.Tool, FindingID: input.FindingID, Disposition: input.Disposition,
		Reason: strings.TrimSpace(input.Reason), By: input.By,
	}
	if err := validateVerdict(verdict); err != nil {
		return err
	}
	path := verdictPath(root)
	if info, err := os.Lstat(filepath.Join(root, ".pose")); err != nil || !info.IsDir() {
		return errors.New("usage: a POSE instance is required to record a verdict")
	}
	if err := rejectVerdictSymlinks(root); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("usage: create verdict directory: %w", err)
	}
	if _, err := readVerdicts(root); err != nil {
		return err
	}
	line, err := json.Marshal(verdict)
	if err != nil {
		return fmt.Errorf("usage: encode verdict: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("usage: open verdict journal: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("usage: append verdict: %w", err)
	}
	return nil
}

func verdictPath(root string) string {
	return filepath.Join(root, ".pose", "usage", "verdicts.jsonl")
}

func readVerdicts(root string) ([]Verdict, error) {
	if err := rejectVerdictSymlinks(root); err != nil {
		return nil, err
	}
	f, err := os.Open(verdictPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("usage: open verdict journal: %w", err)
	}
	defer f.Close()
	var verdicts []Verdict
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 8192)
	for line := 1; scanner.Scan(); line++ {
		var verdict Verdict
		if err := json.Unmarshal(scanner.Bytes(), &verdict); err != nil {
			return nil, fmt.Errorf("usage: invalid verdict journal line %d: %w", line, err)
		}
		if err := validateVerdict(verdict); err != nil {
			return nil, fmt.Errorf("usage: invalid verdict journal line %d: %w", line, err)
		}
		verdicts = append(verdicts, verdict)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("usage: read verdict journal: %w", err)
	}
	return verdicts, nil
}

func rejectVerdictSymlinks(root string) error {
	for _, path := range []string{filepath.Join(root, ".pose"), filepath.Join(root, ".pose", "usage"), verdictPath(root)} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("usage: inspect verdict path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("usage: verdict path must not contain symlinks")
		}
	}
	return nil
}

func validateVerdict(verdict Verdict) error {
	if verdict.SchemaVersion != verdictSchemaVersion {
		return errors.New("unsupported verdict schema")
	}
	if _, err := time.Parse(time.RFC3339Nano, verdict.RecordedAt); err != nil {
		return errors.New("invalid verdict timestamp")
	}
	if !safeName(verdict.Tool) || !safeFindingID(verdict.FindingID) || !safeName(verdict.By) {
		return errors.New("tool, finding and by must be bounded stable identifiers; finding cannot be absolute, traversing or free-form")
	}
	switch verdict.Disposition {
	case "valid", "wont-fix", "false-positive":
	default:
		return errors.New("verdict must be valid|wont-fix|false-positive")
	}
	if verdict.Reason == "" || len(verdict.Reason) > 500 || strings.TrimSpace(verdict.Reason) != verdict.Reason {
		return errors.New("reason must be 1-500 characters")
	}
	for _, r := range verdict.Reason {
		if r < ' ' || r == 0x7f {
			return errors.New("reason must be one line without control characters")
		}
	}
	return nil
}

func safeFindingID(value string) bool {
	if value == "" || len(value) > 256 || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return false
		}
		for _, r := range segment {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.:-", r) {
				continue
			}
			return false
		}
	}
	return true
}
