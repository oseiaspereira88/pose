// Package usage records privacy-bounded local POSE tool outcomes and builds
// project-scoped aggregates. It never sends data over the network and never
// persists repository paths, command arguments, output, content or identity.
package usage

import (
	"bufio"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SchemaVersion = 1

const maxFindingCount = 1_000_000

var (
	saltMu                sync.Mutex
	validSurface          = map[string]bool{"cli": true, "mcp": true}
	validExecutionOutcome = map[string]bool{
		"completed": true, "failed": true, "invalid": true,
		"denied": true, "error": true,
	}
	validSemanticOutcome = map[string]bool{
		"pass": true, "fail": true, "partial": true,
		"unavailable": true, "unknown": true,
	}
)

// Finding is a transient, raw stable identity. Record hashes ID before any
// persistence; callers must never place user content in Severity.
type Finding struct {
	ID       string
	Severity string
}

// Observation is the in-memory contract used by CLI and MCP adapters. Scope
// and Finding.ID are HMAC'd with the project-local salt before persistence.
type Observation struct {
	At                 time.Time
	Tool               string
	Surface            string
	DurationMS         float64
	ExecutionOutcome   string
	SemanticOutcome    string
	Findings           []Finding
	FindingCount       int
	FindingsBySeverity map[string]int
	FindingSetComplete bool
	Scope              string
	Version            string
}

// Event is the only persisted shape. Keep this field set deliberately narrow.
type Event struct {
	SchemaVersion       int            `json:"schema_version"`
	EventID             string         `json:"event_id"`
	OccurredAt          string         `json:"occurred_at"`
	Tool                string         `json:"tool"`
	Surface             string         `json:"surface"`
	DurationMS          float64        `json:"duration_ms"`
	ExecutionOutcome    string         `json:"execution_outcome"`
	SemanticOutcome     string         `json:"semantic_outcome"`
	FindingCount        int            `json:"finding_count"`
	FindingsBySeverity  map[string]int `json:"findings_by_severity,omitempty"`
	FindingFingerprints []string       `json:"finding_fingerprints,omitempty"`
	FindingSetComplete  bool           `json:"finding_set_complete"`
	ScopeHash           string         `json:"scope_hash"`
	Version             string         `json:"version,omitempty"`
}

type Query struct {
	SinceDays int
	Tool      string
	Surface   string
	Now       time.Time
}

type Row struct {
	Tool               string         `json:"tool"`
	Surface            string         `json:"surface"`
	Calls              int            `json:"calls"`
	Completed          int            `json:"completed"`
	Failed             int            `json:"failed"`
	Invalid            int            `json:"invalid"`
	Denied             int            `json:"denied"`
	Errors             int            `json:"errors"`
	Pass               int            `json:"pass"`
	Fail               int            `json:"fail"`
	Partial            int            `json:"partial"`
	Unavailable        int            `json:"unavailable"`
	Unknown            int            `json:"unknown"`
	RunsWithFindings   int            `json:"runs_with_findings"`
	FindingsObserved   int            `json:"findings_observed"`
	FindingsBySeverity map[string]int `json:"findings_by_severity"`
	UniqueFindings     int            `json:"unique_findings"`
	NewFindings        int            `json:"new_findings"`
	ResolvedFindings   int            `json:"resolved_findings"`
	ReopenedFindings   int            `json:"reopened_findings"`
	AverageDurationMS  float64        `json:"average_duration_ms"`
	P50DurationMS      float64        `json:"p50_duration_ms"`
	P95DurationMS      float64        `json:"p95_duration_ms"`
	durations          []float64
	uniqueFingerprints map[string]bool
}

type Report struct {
	SchemaVersion  int    `json:"schema_version"`
	GeneratedAt    string `json:"generated_at"`
	SinceDays      int    `json:"since_days"`
	ToolFilter     string `json:"tool_filter,omitempty"`
	SurfaceFilter  string `json:"surface_filter,omitempty"`
	Available      bool   `json:"available"`
	Reason         string `json:"reason,omitempty"`
	RecordsScanned int    `json:"records_scanned"`
	RecordsMatched int    `json:"records_matched"`
	InvalidRecords int    `json:"invalid_records"`
	Rows           []Row  `json:"rows"`
}

// Record persists one event. Failure is returned for diagnostics, but callers
// intentionally treat it as best-effort and preserve their primary outcome.
func Record(root string, observation Observation) error {
	if value := os.Getenv("POSE_USAGE_DISABLED"); value == "1" || strings.EqualFold(value, "true") {
		return nil
	}
	if err := validateObservation(observation); err != nil {
		return err
	}
	dir, err := storageDir(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("usage: create storage: %w", err)
	}
	salt, err := loadOrCreateSalt(dir)
	if err != nil {
		return err
	}
	at := observation.At.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	fingerprints := make([]string, 0, len(observation.Findings))
	bySeverity := map[string]int{}
	for severity, count := range observation.FindingsBySeverity {
		bySeverity[normalizedSeverity(severity)] += count
	}
	seenFindingIDs := map[string]bool{}
	for _, finding := range observation.Findings {
		if strings.TrimSpace(finding.ID) == "" || seenFindingIDs[finding.ID] {
			continue
		}
		seenFindingIDs[finding.ID] = true
		fingerprints = append(fingerprints, token(salt, "finding\x00"+observation.Tool+"\x00"+finding.ID))
		severity := normalizedSeverity(finding.Severity)
		bySeverity[severity]++
	}
	sort.Strings(fingerprints)
	fingerprints = compactStrings(fingerprints)
	findingCount := observation.FindingCount
	if findingCount < len(fingerprints) {
		findingCount = len(fingerprints)
	}
	severityCount := 0
	for _, count := range bySeverity {
		severityCount += count
	}
	if findingCount < severityCount {
		findingCount = severityCount
	} else if findingCount > severityCount {
		bySeverity["unspecified"] += findingCount - severityCount
	}
	event := Event{
		SchemaVersion: SchemaVersion, EventID: randomID(), OccurredAt: at.Format(time.RFC3339Nano),
		Tool: observation.Tool, Surface: observation.Surface, DurationMS: observation.DurationMS,
		ExecutionOutcome: observation.ExecutionOutcome, SemanticOutcome: observation.SemanticOutcome,
		FindingCount: findingCount, FindingsBySeverity: bySeverity, FindingFingerprints: fingerprints,
		FindingSetComplete: observation.FindingSetComplete,
		ScopeHash:          token(salt, "scope\x00"+observation.Tool+"\x00"+observation.Scope), Version: observation.Version,
	}
	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("usage: encode event: %w", err)
	}
	path := filepath.Join(dir, at.Format("2006-01")+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("usage: open journal: %w", err)
	}
	defer f.Close()
	// One bounded write keeps independent O_APPEND writers atomic enough on
	// ordinary local filesystems; malformed lines remain visible to Aggregate.
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("usage: append event: %w", err)
	}
	return nil
}

func Aggregate(root string, query Query) (Report, error) {
	if query.SinceDays < 0 {
		return Report{}, errors.New("usage: since_days must be >= 0")
	}
	if query.Surface != "" && !validSurface[query.Surface] {
		return Report{}, errors.New("usage: surface must be cli|mcp")
	}
	now := query.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	report := Report{SchemaVersion: SchemaVersion, GeneratedAt: now.Format(time.RFC3339), SinceDays: query.SinceDays, ToolFilter: query.Tool, SurfaceFilter: query.Surface, Rows: []Row{}}
	dir, err := storageDir(root)
	if err != nil {
		return report, err
	}
	events, invalid, err := readEvents(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.Reason = "no usage events recorded"
			return report, nil
		}
		return report, err
	}
	report.RecordsScanned, report.InvalidRecords = len(events), invalid
	sort.SliceStable(events, func(i, j int) bool { return events[i].OccurredAt < events[j].OccurredAt })
	cutoff := time.Time{}
	if query.SinceDays > 0 {
		cutoff = now.AddDate(0, 0, -query.SinceDays)
	}
	rows := map[string]*Row{}
	seen := map[string]bool{}
	active := map[string]map[string]bool{}
	for _, event := range events {
		occurred, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
		if err != nil || occurred.After(now) {
			continue
		}
		filterMatch := (query.Tool == "" || event.Tool == query.Tool) && (query.Surface == "" || event.Surface == query.Surface)
		inWindow := cutoff.IsZero() || !occurred.Before(cutoff)
		include := filterMatch && inWindow
		key := event.Tool + "\x00" + event.Surface
		row := rows[key]
		if include && row == nil {
			row = &Row{Tool: event.Tool, Surface: event.Surface, FindingsBySeverity: map[string]int{}, uniqueFingerprints: map[string]bool{}}
			rows[key] = row
		}
		stateKey := key + "\x00" + event.ScopeHash
		previous := active[stateKey]
		current := make(map[string]bool, len(event.FindingFingerprints))
		for _, fingerprint := range event.FindingFingerprints {
			current[fingerprint] = true
			globalKey := key + "\x00" + fingerprint
			if include {
				row.uniqueFingerprints[fingerprint] = true
				if !seen[globalKey] {
					row.NewFindings++
				} else if event.FindingSetComplete && previous != nil && !previous[fingerprint] {
					row.ReopenedFindings++
				}
			}
			seen[globalKey] = true
		}
		if event.FindingSetComplete {
			if include && previous != nil {
				for fingerprint := range previous {
					if !current[fingerprint] {
						row.ResolvedFindings++
					}
				}
			}
			active[stateKey] = current
		}
		if !include {
			continue
		}
		report.RecordsMatched++
		row.Calls++
		incrementExecution(row, event.ExecutionOutcome)
		incrementSemantic(row, event.SemanticOutcome)
		if event.FindingCount > 0 {
			row.RunsWithFindings++
			row.FindingsObserved += event.FindingCount
		}
		for severity, count := range event.FindingsBySeverity {
			row.FindingsBySeverity[severity] += count
		}
		row.durations = append(row.durations, event.DurationMS)
	}
	for _, row := range rows {
		row.UniqueFindings = len(row.uniqueFingerprints)
		deleteTransient(row)
		report.Rows = append(report.Rows, *row)
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].Calls != report.Rows[j].Calls {
			return report.Rows[i].Calls > report.Rows[j].Calls
		}
		if report.Rows[i].Tool != report.Rows[j].Tool {
			return report.Rows[i].Tool < report.Rows[j].Tool
		}
		return report.Rows[i].Surface < report.Rows[j].Surface
	})
	report.Available = report.RecordsMatched > 0
	if !report.Available {
		report.Reason = "no usage events matched the query"
	}
	return report, nil
}

func validateObservation(observation Observation) error {
	if !safeName(observation.Tool) {
		return errors.New("usage: invalid tool name")
	}
	if !validSurface[observation.Surface] {
		return errors.New("usage: surface must be cli|mcp")
	}
	if !validExecutionOutcome[observation.ExecutionOutcome] {
		return errors.New("usage: invalid execution outcome")
	}
	if !validSemanticOutcome[observation.SemanticOutcome] {
		return errors.New("usage: invalid semantic outcome")
	}
	if observation.DurationMS < 0 || observation.FindingCount < 0 || observation.FindingCount > maxFindingCount || len(observation.Findings) > maxFindingCount {
		return errors.New("usage: counts and duration must be within bounds")
	}
	severityTotal := 0
	for severity, count := range observation.FindingsBySeverity {
		if normalizedSeverity(severity) != strings.ToLower(strings.TrimSpace(severity)) || count < 0 || count > maxFindingCount {
			return errors.New("usage: invalid severity count")
		}
		severityTotal += count
	}
	if severityTotal > maxFindingCount {
		return errors.New("usage: severity count exceeds limit")
	}
	return nil
}

func validEvent(event Event) bool {
	if event.SchemaVersion != SchemaVersion || !safeName(event.EventID) || !safeName(event.Tool) || !validSurface[event.Surface] || !validExecutionOutcome[event.ExecutionOutcome] || !validSemanticOutcome[event.SemanticOutcome] || !validToken(event.ScopeHash) || event.FindingCount < 0 || event.FindingCount > maxFindingCount || event.DurationMS < 0 {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, event.OccurredAt); err != nil {
		return false
	}
	for severity, count := range event.FindingsBySeverity {
		if normalizedSeverity(severity) != severity || count < 0 || count > maxFindingCount {
			return false
		}
	}
	for _, fingerprint := range event.FindingFingerprints {
		if !validToken(fingerprint) {
			return false
		}
	}
	return true
}

func validToken(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func safeName(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.:-", r) {
			continue
		}
		return false
	}
	return true
}

func storageDir(root string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("POSE_USAGE_DIR")); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("usage: POSE_USAGE_DIR must be absolute")
		}
		cleanOverride := filepath.Clean(override)
		if absRoot, rootErr := filepath.Abs(root); rootErr == nil {
			if rel, relErr := filepath.Rel(absRoot, cleanOverride); relErr == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
				return "", errors.New("usage: POSE_USAGE_DIR must remain outside the project worktree")
			}
		}
		return cleanOverride, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("usage: resolve project root: %w", err)
	}
	cmd := exec.Command("git", "-C", absRoot, "rev-parse", "--git-common-dir")
	if out, err := cmd.Output(); err == nil {
		common := strings.TrimSpace(string(out))
		if common != "" {
			if !filepath.IsAbs(common) {
				common = filepath.Join(absRoot, common)
			}
			return filepath.Join(filepath.Clean(common), "pose", "usage"), nil
		}
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("usage: resolve user cache: %w", err)
	}
	digest := sha256.Sum256([]byte(filepath.Clean(absRoot)))
	return filepath.Join(cache, "pose", "usage", hex.EncodeToString(digest[:12])), nil
}

func loadOrCreateSalt(dir string) ([]byte, error) {
	saltMu.Lock()
	defer saltMu.Unlock()
	return loadOrCreateSaltLocked(dir)
}

func loadOrCreateSaltLocked(dir string) ([]byte, error) {
	path := filepath.Join(dir, "salt")
	if salt, exists, err := readSalt(path); exists {
		if err == nil {
			return salt, nil
		}
		for attempt := 0; attempt < 10; attempt++ {
			time.Sleep(time.Millisecond)
			if retrySalt, retryExists, retryErr := readSalt(path); retryExists && retryErr == nil {
				return retrySalt, nil
			}
		}
		return nil, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("usage: generate salt: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		// Another process won first-use initialization. Its O_EXCL file can be
		// visible just before its contents, so briefly retry the bounded read
		// instead of dropping this otherwise-valid event.
		for attempt := 0; attempt < 10; attempt++ {
			if salt, exists, readErr := readSalt(path); exists && readErr == nil {
				return salt, nil
			}
			time.Sleep(time.Millisecond)
		}
		return nil, errors.New("usage: local salt initialization did not complete")
	}
	if err != nil {
		return nil, fmt.Errorf("usage: create salt: %w", err)
	}
	if _, err := f.WriteString(hex.EncodeToString(raw) + "\n"); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("usage: write salt: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("usage: close salt: %w", err)
	}
	return raw, nil
}

func readSalt(path string) ([]byte, bool, error) {
	encoded, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("usage: read salt: %w", err)
	}
	salt, err := hex.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil || len(salt) < 16 {
		return nil, true, errors.New("usage: invalid local salt")
	}
	return salt, true, nil
}

func readEvents(dir string) ([]Event, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	var events []Event
	invalid := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			invalid++
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var event Event
			if json.Unmarshal([]byte(line), &event) != nil || !validEvent(event) {
				invalid++
				continue
			}
			events = append(events, event)
		}
		if scanner.Err() != nil {
			invalid++
		}
		_ = f.Close()
	}
	return events, invalid, nil
}

func token(salt []byte, value string) string {
	mac := hmac.New(sha256.New, salt)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

func randomID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "use_" + hex.EncodeToString(raw[:])
	}
	return "use_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func normalizedSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "high", "medium", "low", "error", "warning", "info", "required", "optional":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unspecified"
	}
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func incrementExecution(row *Row, outcome string) {
	switch outcome {
	case "completed":
		row.Completed++
	case "failed":
		row.Failed++
	case "invalid":
		row.Invalid++
	case "denied":
		row.Denied++
	case "error":
		row.Errors++
	}
}

func incrementSemantic(row *Row, outcome string) {
	switch outcome {
	case "pass":
		row.Pass++
	case "fail":
		row.Fail++
	case "partial":
		row.Partial++
	case "unavailable":
		row.Unavailable++
	case "unknown":
		row.Unknown++
	}
}

func deleteTransient(row *Row) {
	if len(row.durations) == 0 {
		return
	}
	sort.Float64s(row.durations)
	var total float64
	for _, duration := range row.durations {
		total += duration
	}
	row.AverageDurationMS = total / float64(len(row.durations))
	row.P50DurationMS = percentile(row.durations, 0.50)
	row.P95DurationMS = percentile(row.durations, 0.95)
	row.durations = nil
	row.uniqueFingerprints = nil
}

func percentile(sorted []float64, quantile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1)*quantile + 0.5)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
