package pose

// Capability assessment (spec pose-capability-mechanism): the method's
// capability scores become a POSE-native artifact at
// .pose/capabilities/assessment.md — flat frontmatter plus one
// "## Mechanism: <id>" section per mechanism with flat bullets. The bullets
// are the authority; prose below them is commentary and is never parsed.
// Snapshots append to .pose/capabilities/history.jsonl (append-only, same
// contract family as amendments.jsonl). This package parses and validates;
// writes (init/snapshot) belong to the CLI layer.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CapabilityAssessmentSchema is the current assessment.md schema version.
const CapabilityAssessmentSchema = 1

// CapabilitySnapshotSchema is the current history.jsonl schema version.
const CapabilitySnapshotSchema = 1

// CapabilityMechanism is one "## Mechanism: <id>" section's structured data.
type CapabilityMechanism struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Score    int      `json:"score"`
	Target   int      `json:"target"`
	Retired  bool     `json:"retired,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
	Gaps     []string `json:"gaps,omitempty"`
}

// CapabilityAssessment is the parsed artifact.
type CapabilityAssessment struct {
	SchemaVersion  int                   `json:"schema_version"`
	AssessedAt     string                `json:"assessed_at"`
	BaselineCommit string                `json:"baseline_commit"`
	Method         string                `json:"method,omitempty"`
	Mechanisms     []CapabilityMechanism `json:"mechanisms"`
	Path           string                `json:"path"`
}

// CapabilityScore is one mechanism's score vector inside a snapshot.
type CapabilityScore struct {
	Score   int  `json:"score"`
	Target  int  `json:"target"`
	Retired bool `json:"retired,omitempty"`
}

// CapabilitySnapshot is one append-only history event.
type CapabilitySnapshot struct {
	Schema         int                        `json:"schema"`
	At             string                     `json:"at"` // RFC3339 UTC
	BaselineCommit string                     `json:"baseline_commit"`
	ContentHash    string                     `json:"content_hash"`
	Scores         map[string]CapabilityScore `json:"scores"`
	SupersedesTS   string                     `json:"supersedes_ts,omitempty"`
}

// CapabilitiesDir returns the artifact directory for a root.
func (s Store) CapabilitiesDir() string {
	return filepath.Join(s.Root, ".pose", "capabilities")
}

// CapabilityAssessmentPath returns the assessment file path for a root.
func (s Store) CapabilityAssessmentPath() string {
	return filepath.Join(s.CapabilitiesDir(), "assessment.md")
}

// CapabilityHistoryPath returns the history file path for a root.
func (s Store) CapabilityHistoryPath() string {
	return filepath.Join(s.CapabilitiesDir(), "history.jsonl")
}

// HasCapabilityAssessment reports whether the opt-in artifact exists.
func (s Store) HasCapabilityAssessment() bool {
	_, err := os.Stat(s.CapabilityAssessmentPath())
	return err == nil
}

var mechanismHeading = regexp.MustCompile(`^## Mechanism:\s*([A-Za-z0-9._-]+)\s*$`)
var commitRefPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// LoadCapabilityAssessment reads and parses the artifact for this root.
func (s Store) LoadCapabilityAssessment() (*CapabilityAssessment, error) {
	path := s.CapabilityAssessmentPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: capability assessment not found at %s (run `pose assess init`)", path)
	}
	assessment, err := ParseCapabilityAssessment(string(raw))
	if err != nil {
		return nil, err
	}
	assessment.Path = path
	return assessment, nil
}

// ParseCapabilityAssessment parses the artifact content. Structural errors
// (bad schema, malformed bullets, duplicate ids) fail here; evidence
// resolution is a separate, root-aware step (ValidateCapabilityEvidence).
func ParseCapabilityAssessment(content string) (*CapabilityAssessment, error) {
	fm, body := splitFrontmatter(content)
	assessment := &CapabilityAssessment{}

	schemaRaw, ok := fm["schema_version"]
	if !ok {
		return nil, fmt.Errorf("pose: assessment frontmatter missing schema_version")
	}
	schema, err := strconv.Atoi(schemaRaw)
	if err != nil || schema < 1 {
		return nil, fmt.Errorf("pose: assessment schema_version %q is not a positive integer", schemaRaw)
	}
	if schema > CapabilityAssessmentSchema {
		return nil, fmt.Errorf("pose: assessment schema_version %d is newer than supported %d", schema, CapabilityAssessmentSchema)
	}
	assessment.SchemaVersion = schema
	assessment.AssessedAt = fm["assessed_at"]
	assessment.BaselineCommit = fm["baseline_commit"]
	assessment.Method = fm["method"]
	if assessment.AssessedAt == "" {
		return nil, fmt.Errorf("pose: assessment frontmatter missing assessed_at")
	}
	if _, err := time.Parse("2006-01-02", assessment.AssessedAt); err != nil {
		return nil, fmt.Errorf("pose: assessment assessed_at %q is not YYYY-MM-DD", assessment.AssessedAt)
	}
	if assessment.BaselineCommit == "" {
		return nil, fmt.Errorf("pose: assessment frontmatter missing baseline_commit")
	}
	if !commitRefPattern.MatchString(assessment.BaselineCommit) {
		return nil, fmt.Errorf("pose: assessment baseline_commit %q is not a 7-40 char lowercase hex hash", assessment.BaselineCommit)
	}

	seen := map[string]bool{}
	var current *CapabilityMechanism
	flush := func() error {
		if current == nil {
			return nil
		}
		if err := validateMechanism(*current); err != nil {
			return err
		}
		assessment.Mechanisms = append(assessment.Mechanisms, *current)
		current = nil
		return nil
	}
	for _, line := range strings.Split(body, "\n") {
		if m := mechanismHeading.FindStringSubmatch(line); m != nil {
			if err := flush(); err != nil {
				return nil, err
			}
			id := m[1]
			if !slugPattern.MatchString(id) {
				return nil, fmt.Errorf("pose: mechanism id %q is not a valid slug", id)
			}
			if seen[id] {
				return nil, fmt.Errorf("pose: duplicate mechanism id %q", id)
			}
			seen[id] = true
			current = &CapabilityMechanism{ID: id, Score: -1, Target: -1}
			continue
		}
		if current == nil || !strings.HasPrefix(line, "- ") {
			continue
		}
		key, value, found := strings.Cut(strings.TrimPrefix(line, "- "), ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "title":
			current.Title = value
		case "score":
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("pose: mechanism %q score %q is not an integer", current.ID, value)
			}
			current.Score = n
		case "target":
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("pose: mechanism %q target %q is not an integer", current.ID, value)
			}
			current.Target = n
		case "retired":
			current.Retired = value == "true"
		case "evidence":
			current.Evidence = splitInlineList(value)
		case "gaps":
			current.Gaps = splitSemicolonList(value)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(assessment.Mechanisms) == 0 {
		return nil, fmt.Errorf("pose: assessment declares no `## Mechanism:` sections")
	}
	return assessment, nil
}

func validateMechanism(m CapabilityMechanism) error {
	if m.Title == "" {
		return fmt.Errorf("pose: mechanism %q is missing `- title:`", m.ID)
	}
	if m.Score < 0 || m.Score > 5 {
		return fmt.Errorf("pose: mechanism %q score must be an integer 0-5 (got %d; missing `- score:`?)", m.ID, m.Score)
	}
	if m.Target < 0 || m.Target > 5 {
		return fmt.Errorf("pose: mechanism %q target must be an integer 0-5 (got %d; missing `- target:`?)", m.ID, m.Target)
	}
	return nil
}

func splitInlineList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func splitSemicolonList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ";") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// CapabilityEvidenceTypes lists the supported typed reference prefixes.
var CapabilityEvidenceTypes = []string{"spec", "report", "adr", "knowledge", "doc", "commit", "check", "url"}

// ValidateCapabilityEvidence resolves every typed evidence reference against
// this root. It returns one nominal issue string per unresolvable or
// malformed reference; local file types must exist, commit/check/url are
// syntactic only (offline contract).
func (s Store) ValidateCapabilityEvidence(assessment *CapabilityAssessment) []string {
	var issues []string
	knowledgeSlugs := map[string]bool{}
	if entries, err := s.ListKnowledge(); err == nil {
		for _, entry := range entries {
			knowledgeSlugs[entry.Slug] = true
		}
	}
	for _, mechanism := range assessment.Mechanisms {
		for _, ref := range mechanism.Evidence {
			kind, value, found := strings.Cut(ref, ":")
			if !found || value == "" {
				issues = append(issues, fmt.Sprintf("mechanism %q: evidence %q is not a typed reference (<type>:<value>)", mechanism.ID, ref))
				continue
			}
			switch kind {
			case "spec":
				if err := ValidateSlug(value); err != nil {
					issues = append(issues, fmt.Sprintf("mechanism %q: spec reference %q has an invalid slug", mechanism.ID, ref))
					continue
				}
				if _, err := s.GetSpec(value); err != nil {
					issues = append(issues, fmt.Sprintf("mechanism %q: spec %q not found in .pose/specs", mechanism.ID, value))
				}
			case "report":
				if !localArtifactExists(s.Root, ".pose/reports", value) {
					issues = append(issues, fmt.Sprintf("mechanism %q: report %q not found in .pose/reports", mechanism.ID, value))
				}
			case "adr":
				if !localArtifactExists(s.Root, ".pose/adr", value) {
					issues = append(issues, fmt.Sprintf("mechanism %q: adr %q not found in .pose/adr", mechanism.ID, value))
				}
			case "knowledge":
				if !knowledgeSlugs[value] {
					issues = append(issues, fmt.Sprintf("mechanism %q: knowledge %q not found in .pose/knowledge", mechanism.ID, value))
				}
			case "doc":
				if !localArtifactExists(s.Root, ".", value) {
					issues = append(issues, fmt.Sprintf("mechanism %q: doc %q not found under the project root", mechanism.ID, value))
				}
			case "commit":
				if !commitRefPattern.MatchString(value) {
					issues = append(issues, fmt.Sprintf("mechanism %q: commit %q is not a 7-40 char lowercase hex hash", mechanism.ID, value))
				}
			case "check":
				// Syntactic: any non-empty command string is acceptable.
			case "url":
				if !strings.HasPrefix(value, "https://") {
					issues = append(issues, fmt.Sprintf("mechanism %q: url reference %q must start with https://", mechanism.ID, ref))
				}
			default:
				issues = append(issues, fmt.Sprintf("mechanism %q: evidence type %q is not one of %s", mechanism.ID, kind, strings.Join(CapabilityEvidenceTypes, "/")))
			}
		}
	}
	return issues
}

// localArtifactExists confines value under root/base and checks existence.
// Traversal attempts resolve to false, never to an out-of-root read.
func localArtifactExists(root, base, value string) bool {
	if strings.Contains(value, "..") || strings.HasPrefix(value, "/") {
		return false
	}
	full := filepath.Join(root, filepath.FromSlash(base), filepath.FromSlash(value))
	cleanRoot := filepath.Clean(root) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(full)+string(filepath.Separator), cleanRoot) {
		return false
	}
	_, err := os.Stat(full)
	return err == nil
}

// CapabilityContentHash fingerprints the artifact content for snapshots.
func CapabilityContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:12]
}

// ScoresOf projects the assessment's score vector for a snapshot.
func (a *CapabilityAssessment) ScoresOf() map[string]CapabilityScore {
	scores := map[string]CapabilityScore{}
	for _, m := range a.Mechanisms {
		scores[m.ID] = CapabilityScore{Score: m.Score, Target: m.Target, Retired: m.Retired}
	}
	return scores
}

// LoadCapabilityHistory reads the append-only snapshot log. A missing file is
// an empty history, not an error. Entries with a schema newer than supported
// fail loudly instead of being skipped.
func LoadCapabilityHistory(path string) ([]CapabilitySnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var events []CapabilitySnapshot
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var event CapabilitySnapshot
		if err := json.Unmarshal([]byte(text), &event); err != nil {
			return nil, fmt.Errorf("pose: %s line %d: %v", path, line, err)
		}
		if event.Schema > CapabilitySnapshotSchema {
			return nil, fmt.Errorf("pose: %s line %d: snapshot schema %d is newer than supported %d", path, line, event.Schema, CapabilitySnapshotSchema)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// EffectiveSnapshots filters superseded entries: a snapshot whose timestamp
// is named by a later entry's supersedes_ts is replaced by that entry.
func EffectiveSnapshots(events []CapabilitySnapshot) []CapabilitySnapshot {
	superseded := map[string]bool{}
	for _, event := range events {
		if event.SupersedesTS != "" {
			superseded[event.SupersedesTS] = true
		}
	}
	var out []CapabilitySnapshot
	for _, event := range events {
		if !superseded[event.At] {
			out = append(out, event)
		}
	}
	return out
}

// CapabilityDiffEntry is one mechanism's movement between two snapshots.
type CapabilityDiffEntry struct {
	ID   string `json:"id"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// CapabilityDiff is the mechanical comparison between two snapshots.
type CapabilityDiff struct {
	FromAt  string                `json:"from_at"`
	ToAt    string                `json:"to_at"`
	Raised  []CapabilityDiffEntry `json:"raised,omitempty"`
	Lowered []CapabilityDiffEntry `json:"lowered,omitempty"`
	Stable  []string              `json:"stable,omitempty"`
	Added   []string              `json:"added,omitempty"`
	Removed []string              `json:"removed,omitempty"`
	Retired []string              `json:"retired,omitempty"`
}

// DiffCapabilitySnapshots compares two snapshots deterministically.
func DiffCapabilitySnapshots(from, to CapabilitySnapshot) CapabilityDiff {
	diff := CapabilityDiff{FromAt: from.At, ToAt: to.At}
	var ids []string
	seen := map[string]bool{}
	for id := range from.Scores {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range to.Scores {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		before, inFrom := from.Scores[id]
		after, inTo := to.Scores[id]
		switch {
		case inFrom && !inTo:
			diff.Removed = append(diff.Removed, id)
		case !inFrom && inTo:
			diff.Added = append(diff.Added, id)
		case !before.Retired && after.Retired:
			diff.Retired = append(diff.Retired, id)
		case after.Score > before.Score:
			diff.Raised = append(diff.Raised, CapabilityDiffEntry{ID: id, From: before.Score, To: after.Score})
		case after.Score < before.Score:
			diff.Lowered = append(diff.Lowered, CapabilityDiffEntry{ID: id, From: before.Score, To: after.Score})
		default:
			diff.Stable = append(diff.Stable, id)
		}
	}
	return diff
}

// RenumberedMechanisms returns ids present (non-retired) in the latest
// snapshot but absent from the current assessment — the stable-id contract:
// published mechanisms are never removed, only retired.
func RenumberedMechanisms(latest CapabilitySnapshot, current *CapabilityAssessment) []string {
	present := map[string]bool{}
	for _, m := range current.Mechanisms {
		present[m.ID] = true
	}
	var missing []string
	for id, score := range latest.Scores {
		if !score.Retired && !present[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}
