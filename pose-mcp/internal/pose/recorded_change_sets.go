package pose

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// ReportRecord is one line of a report history.
type ReportRecord struct {
	GeneratedAt       string     `json:"generated_at"`
	Sequence          int        `json:"sequence"`
	Task              string     `json:"task"`
	TaskSlug          string     `json:"task_slug"`
	ReportType        string     `json:"report_type"`
	Spec              string     `json:"spec"`
	Workflow          string     `json:"workflow"`
	Rules             string     `json:"rules"`
	ValidationProfile string     `json:"validation_profile"`
	Context           string     `json:"context"`
	Risk              string     `json:"risk"`
	Outcome           string     `json:"outcome"`
	OutcomeSource     string     `json:"outcome_source"`
	StableHash        string     `json:"stable_hash"`
	ReportPath        string     `json:"report_path"`
	ChangeSet         *ChangeSet `json:"change_set,omitempty"`
	// Optional telemetry (spec pose-recurrence-effectiveness); absent when
	// not provided — downstream metrics stay partial, never fabricated.
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	CostUSD         *float64 `json:"cost_usd,omitempty"`
}

// RecordedChangeSetLineLimit bounds one record of a report history; a record
// a native report writes for a wide change set is far larger than a scanner's
// default token (review round 5 of spec pose-delivery-later-removal).
var RecordedChangeSetLineLimit = 64 * 1024 * 1024

// LoadRecordedChangeSetsIn reads the recorded change sets and says why a part of the recorded
// history could not be read — a file that did not open, a record over the
// limit, a line that is not a record — so that a later removal refuses
// instead of judging the sets that were read (LR-R5-01).
func LoadRecordedChangeSetsIn(files Files, root string) ([]ChangeSet, string) {
	gap := ""
	note := func(format string, args ...any) {
		if gap == "" {
			gap = "the recorded change sets could not be read: " + fmt.Sprintf(format, args...)
		}
	}
	// The directory is listed literally, and only its absence is an empty
	// history: a glob would read the root as a pattern and drop the errors
	// it meets (LR-R6-01).
	dir := filepath.Join(root, ".pose", "reports", "history")
	entries, err := files.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		note("%v", err)
	}
	var paths []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	sets := map[string]ChangeSet{}
	for _, path := range paths {
		file, err := files.Open(path)
		if err != nil {
			note("%v", err)
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), RecordedChangeSetLineLimit)
		line := 0
		for scanner.Scan() {
			line++
			if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
				continue
			}
			var record ReportRecord
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				note("%s:%d: %v", filepath.Base(path), line, err)
				continue
			}
			if record.ChangeSet != nil {
				sets[record.ChangeSet.ID] = *record.ChangeSet
			}
		}
		if err := scanner.Err(); err != nil {
			note("%s:%d: %v", filepath.Base(path), line+1, err)
		}
		_ = file.Close()
	}
	values := make([]ChangeSet, 0, len(sets))
	for _, set := range sets {
		values = append(values, set)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values, gap
}

// recordedChangeSetsDigest digests what the delivery graph reads from the
// report history, its own reader's result, so no second decoding can disagree
// with it: a report without a change set changes nothing, a loss changes it,
// and a history with neither reads as absent (spec
// pose-removal-witness-freshness R3, D2).
func recordedChangeSetsDigest(files Files, root string) string {
	sets, gap := LoadRecordedChangeSetsIn(files, root)
	if len(sets) == 0 && gap == "" {
		return "absent"
	}
	raw, err := json.Marshal(struct {
		Sets []ChangeSet `json:"sets"`
		Gap  string      `json:"gap"`
	}{sets, strings.ReplaceAll(gap, root, "")})
	if err != nil {
		return "unreadable: " + err.Error()
	}
	return sha256Prefixed(raw)
}
