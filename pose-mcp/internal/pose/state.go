package pose

// Project state (spec pose-project-state-artifact): the native POSE
// artifact answering "what is the current state of this project?" in one
// read instead of a repo-wide scan. .pose/state/project-state.md is a
// frontmatter + fixed named "## " sections file; each section is either
// `curated` (free prose, preserved verbatim across regenerations) or
// `derived` (computed from other POSE subsystems — counts and typed
// pointers only, never copied content). Writing (init/refresh/diff) is a
// Harness/CLI concern (ADR-003); this file only reads, parses and
// validates the persisted artifact, mirroring the capability-assessment
// split between internal/cli (writer) and internal/pose (reader).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ProjectStateSchema is the current project-state.md schema version.
const ProjectStateSchema = 1

// DefaultStaleMaxAgeDays and DefaultStaleMaxCommits are the staleness
// thresholds applied when .pose/policy/state.json is absent.
const (
	DefaultStaleMaxAgeDays = 7
	DefaultStaleMaxCommits = 20
)

// StatePolicy is the staleness/refresh policy read from
// .pose/policy/state.json. StrictRefresh (spec
// pose-project-state-refresh-contract R5) is refresh-mode configuration,
// not a staleness threshold — it never appears in the frontmatter's
// staleness_policy snapshot (FormatStalenessPolicy only serializes the two
// staleness fields).
type StatePolicy struct {
	MaxAgeDays    int  `json:"max_age_days"`
	MaxCommits    int  `json:"max_commits"`
	StrictRefresh bool `json:"strict_refresh"`
}

// ProjectStateSection is one "## <Name>" section of the artifact.
type ProjectStateSection struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"` // curated | derived
	Hash     string `json:"hash,omitempty"`
	Status   string `json:"status,omitempty"` // e.g. "unavailable"
	Body     string `json:"body"`
	Tampered bool   `json:"tampered,omitempty"`
}

// ProjectStateStaleness reports how far the persisted artifact has drifted
// from "now", by age and (when Git is available) by commit count.
type ProjectStateStaleness struct {
	Stale        bool   `json:"stale"`
	AgeDays      int    `json:"age_days"`
	CommitsSince int    `json:"commits_since"` // -1 when unknown (no git / no baseline)
	MaxAgeDays   int    `json:"max_age_days"`
	MaxCommits   int    `json:"max_commits"`
	Reason       string `json:"reason,omitempty"`
}

// ProjectState is the parsed, validated view of project-state.md.
type ProjectState struct {
	SchemaVersion int    `json:"schema_version"`
	GeneratedAt   string `json:"generated_at"`
	// BaselineCommit is the exact Git HEAD projected by the refresh that
	// produced this artifact.
	BaselineCommit string `json:"baseline_commit"`
	// StalenessPolicyAtGeneration is the policy that was in effect when
	// this artifact was generated — a historical record, printed for
	// transparency. Current staleness (Staleness below) always judges
	// against the live policy (LoadStatePolicy), never this frozen copy,
	// so tightening the policy makes old artifacts immediately stricter.
	StalenessPolicyAtGeneration StatePolicy           `json:"staleness_policy_at_generation"`
	Sections                    []ProjectStateSection `json:"sections"`
	Staleness                   ProjectStateStaleness `json:"staleness"`
	Tampered                    bool                  `json:"tampered"`
	// RefreshPending is the hook event kind a failed automatic refresh
	// could not process (spec pose-project-state-refresh-contract R5) —
	// "" when no refresh is pending. Cleared by the next successful
	// refresh, regardless of what triggered it.
	RefreshPending string `json:"refresh_pending,omitempty"`
	Path           string `json:"path"`
}

func (s Store) stateDir() string { return filepath.Join(s.Root, ".pose", "state") }

// StatePath is the artifact path for a root.
func (s Store) StatePath() string { return filepath.Join(s.stateDir(), "project-state.md") }

// StateHistoryPath is the append-only refresh history for a root.
func (s Store) StateHistoryPath() string { return filepath.Join(s.stateDir(), "history.jsonl") }

// StatePolicyPath is the staleness policy file for a root.
func (s Store) StatePolicyPath() string {
	return filepath.Join(s.Root, ".pose", "policy", "state.json")
}

// HasProjectState reports whether the opt-in artifact exists — projects
// without it stay valid in every gate (additive, spec Restrições).
func (s Store) HasProjectState() bool {
	_, err := os.Stat(s.StatePath())
	return err == nil
}

// LoadStatePolicy reads the staleness policy, falling back to defaults
// when the file is absent (never an error — the policy is optional).
func (s Store) LoadStatePolicy() StatePolicy {
	policy := StatePolicy{MaxAgeDays: DefaultStaleMaxAgeDays, MaxCommits: DefaultStaleMaxCommits}
	raw, err := os.ReadFile(s.StatePolicyPath())
	if err != nil {
		return policy
	}
	var parsed StatePolicy
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return policy
	}
	if parsed.MaxAgeDays > 0 {
		policy.MaxAgeDays = parsed.MaxAgeDays
	}
	if parsed.MaxCommits > 0 {
		policy.MaxCommits = parsed.MaxCommits
	}
	policy.StrictRefresh = parsed.StrictRefresh
	return policy
}

// FormatStalenessPolicy renders a policy as the flat frontmatter value
// `pose state refresh` stamps into staleness_policy — kept in package pose
// (not internal/cli) so the writer and the reader share one format.
func FormatStalenessPolicy(p StatePolicy) string {
	return fmt.Sprintf("max_age_days=%d,max_commits=%d", p.MaxAgeDays, p.MaxCommits)
}

// parseStalenessPolicyField parses the flat staleness_policy frontmatter
// value; malformed or absent values degrade to the zero StatePolicy
// (informational field — never blocks parsing the rest of the artifact).
func parseStalenessPolicyField(value string) StatePolicy {
	var policy StatePolicy
	for _, field := range strings.Split(value, ",") {
		key, v, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok {
			continue
		}
		n := atoiDefault(strings.TrimSpace(v), 0)
		switch strings.TrimSpace(key) {
		case "max_age_days":
			policy.MaxAgeDays = n
		case "max_commits":
			policy.MaxCommits = n
		}
	}
	return policy
}

var stateSectionHeading = regexp.MustCompile(`^##\s+(.+?)\s*$`)
var stateSectionMarker = regexp.MustCompile(`^<!--\s*state:(curated|derived)(?:\s+hash:([0-9a-f]{12}))?(?:\s+status:(\S+))?\s*-->\s*$`)

// ParseProjectState parses the artifact body (frontmatter already split
// out) into its named sections, recomputing each derived section's hash to
// detect manual edits since the last refresh (R4).
func ParseProjectState(fm map[string]string, body string) (*ProjectState, error) {
	state := &ProjectState{SchemaVersion: atoiDefault(fm["schema_version"], 0),
		GeneratedAt: fm["generated_at"], BaselineCommit: fm["baseline_commit"],
		StalenessPolicyAtGeneration: parseStalenessPolicyField(fm["staleness_policy"]),
		RefreshPending:              fm["refresh_pending"]}

	lines := strings.Split(body, "\n")
	var current *ProjectStateSection
	var bodyLines []string
	var pendingMarker bool
	flush := func() {
		if current == nil {
			return
		}
		text := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		current.Body = text
		if current.Kind == "derived" {
			recomputed := ContentHash12(text)
			if current.Hash != "" && recomputed != current.Hash {
				current.Tampered = true
				state.Tampered = true
			}
		}
		state.Sections = append(state.Sections, *current)
	}
	for _, line := range lines {
		if m := stateSectionHeading.FindStringSubmatch(line); m != nil {
			flush()
			current = &ProjectStateSection{Name: strings.TrimSpace(m[1])}
			bodyLines = nil
			pendingMarker = true
			continue
		}
		if current == nil {
			continue
		}
		if pendingMarker {
			pendingMarker = false
			if m := stateSectionMarker.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				current.Kind, current.Hash, current.Status = m[1], m[2], m[3]
				continue
			}
			// No marker on the line right after the heading: fall through,
			// this line is already section body content.
		}
		bodyLines = append(bodyLines, line)
	}
	flush()

	if state.SchemaVersion != ProjectStateSchema {
		return state, fmt.Errorf("pose: project-state schema_version %d unsupported (want %d)", state.SchemaVersion, ProjectStateSchema)
	}
	if state.GeneratedAt == "" || state.BaselineCommit == "" {
		return state, fmt.Errorf("pose: project-state frontmatter missing generated_at or baseline_commit")
	}
	return state, nil
}

var statePointerToken = regexp.MustCompile("\\b(spec|report|adr|knowledge|doc|commit|check|url|component):[^\\s,;)\"'`]+")

// ExtractPointers returns every typed reference token found in the
// artifact's derived sections. Curated prose is never scanned — it is
// free-form human text, not a data contract.
func (state *ProjectState) ExtractPointers() []string {
	seen := map[string]bool{}
	var out []string
	for _, sec := range state.Sections {
		if sec.Kind != "derived" {
			continue
		}
		for _, m := range statePointerToken.FindAllString(sec.Body, -1) {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ValidatePointers resolves every typed reference found in derived
// sections against local artifacts (R6): a broken pointer is reported,
// never silently ignored.
func (s Store) ValidatePointers(state *ProjectState) []string {
	var issues []string
	for _, ref := range state.ExtractPointers() {
		if ok, reason := s.ResolvePointer(ref); !ok {
			issues = append(issues, reason)
		}
	}
	return issues
}

// ProjectState loads, parses and validates the persisted artifact. When
// section is non-empty, only that section (case-insensitive exact name
// match) is returned — the MCP tool's context-saving path (R5).
func (s Store) ProjectState(ctx context.Context, section string) (*ProjectState, error) {
	path := s.StatePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: project state not found at %s (run `pose state init`)", path)
	}
	fm, body := SplitFrontmatter(string(raw))
	state, err := ParseProjectState(fm, body)
	if err != nil {
		return nil, err
	}
	state.Path = path
	state.Staleness = s.computeStaleness(ctx, state, s.LoadStatePolicy())

	if section != "" {
		for _, sec := range state.Sections {
			if strings.EqualFold(sec.Name, section) {
				filtered := *state
				filtered.Sections = []ProjectStateSection{sec}
				return &filtered, nil
			}
		}
		return nil, fmt.Errorf("pose: project-state section %q not found", section)
	}
	return state, nil
}

func (s Store) computeStaleness(ctx context.Context, state *ProjectState, policy StatePolicy) ProjectStateStaleness {
	st := ProjectStateStaleness{MaxAgeDays: policy.MaxAgeDays, MaxCommits: policy.MaxCommits, CommitsSince: -1}
	generated, err := time.Parse(time.RFC3339, state.GeneratedAt)
	if err != nil {
		st.Stale, st.Reason = true, "generated_at_unparseable"
		return st
	}
	st.AgeDays = int(time.Since(generated).Hours() / 24)
	if st.AgeDays > policy.MaxAgeDays {
		st.Stale, st.Reason = true, "age_exceeded"
	}
	if count, err := s.commitsSince(ctx, state.BaselineCommit); err == nil {
		st.CommitsSince = count
		if count > policy.MaxCommits {
			st.Stale = true
			if st.Reason == "" {
				st.Reason = "commits_exceeded"
			}
		}
	}
	return st
}

var stateCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// commitsSince runs `git rev-list --count <baseline>..HEAD` without a
// shell, confined to the project root. Any failure (no git, no repo,
// unknown baseline) is reported as an error — callers treat that as
// "commit-based staleness unknown", never as zero.
func (s Store) commitsSince(ctx context.Context, baseline string) (int, error) {
	if !stateCommitPattern.MatchString(baseline) {
		return 0, fmt.Errorf("pose: baseline_commit is not a valid hash")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", s.Root, "rev-list", "--count", baseline+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count); err != nil {
		return 0, err
	}
	return count, nil
}

func atoiDefault(value string, fallback int) int {
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return fallback
	}
	return n
}

// sortedKeys returns the map's keys sorted, a small helper the section
// providers reuse to keep derived output deterministic (spec NFR).
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
