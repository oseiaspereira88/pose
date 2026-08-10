package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var releaseVersionRE = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?$`)

type ReleasePolicy struct {
	SchemaVersion int    `json:"schema_version"`
	AdoptedAt     string `json:"adopted_at"`
	TagPattern    string `json:"tag_pattern"`
	AllowEmpty    bool   `json:"allow_empty"`
	LegacyCutoff  string `json:"legacy_cutoff"`
	Provider      string `json:"provider"`
	Repository    string `json:"repository"`
	RequireVerify bool   `json:"require_verification"`
}

type ReleaseFragment struct {
	Spec     string `json:"spec"`
	Category string `json:"category"`
	Breaking bool   `json:"breaking"`
	Refs     string `json:"refs,omitempty"`
	Body     string `json:"body"`
	Path     string `json:"path"`
	Digest   string `json:"digest"`
}

type ReleaseManifest struct {
	SchemaVersion      int               `json:"schema_version"`
	Version            string            `json:"version"`
	PreviousVersion    string            `json:"previous_version,omitempty"`
	PreparedAt         string            `json:"prepared_at"`
	Specs              []string          `json:"specs"`
	Categories         map[string]int    `json:"categories"`
	Fragments          []ReleaseFragment `json:"fragments"`
	Breaking           bool              `json:"breaking"`
	NotesDigest        string            `json:"notes_digest"`
	ReleaseInputDigest string            `json:"release_input_digest"`
	PolicyDigest       string            `json:"policy_digest"`
	VersionEvidence    map[string]string `json:"version_evidence"`
}

type ReleaseEvidence struct {
	SchemaVersion int               `json:"schema_version"`
	Provider      string            `json:"provider"`
	Repository    string            `json:"repository"`
	Version       string            `json:"version"`
	Tag           string            `json:"tag"`
	Commit        string            `json:"commit"`
	PublishedAt   string            `json:"published_at"`
	URL           string            `json:"url"`
	WorkflowRun   string            `json:"workflow_run,omitempty"`
	Assets        map[string]string `json:"assets,omitempty"`
	Publication   string            `json:"publication_digest,omitempty"`
}

type ReleaseEvent struct {
	SchemaVersion  int             `json:"schema_version"`
	Version        string          `json:"version"`
	State          string          `json:"state"`
	RecordedAt     string          `json:"recorded_at"`
	Evidence       ReleaseEvidence `json:"evidence"`
	EvidenceDigest string          `json:"evidence_digest"`
}

type ReleaseProjection struct {
	Version  string           `json:"version"`
	State    string           `json:"state"`
	Manifest *ReleaseManifest `json:"manifest,omitempty"`
	Events   []ReleaseEvent   `json:"events"`
	Gaps     []string         `json:"gaps"`
}

type ReleaseStatus struct {
	Pending  []ReleaseFragment   `json:"pending"`
	Releases []ReleaseProjection `json:"releases"`
}

func ValidateReleaseVersion(version string) error {
	if !releaseVersionRE.MatchString(version) || filepath.Base(version) != version {
		return fmt.Errorf("invalid release version %q (expected vX.Y.Z)", version)
	}
	return nil
}

func ReleaseDigest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func LoadReleasePolicy(root string) (ReleasePolicy, error) {
	var policy ReleasePolicy
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "release-policy.json"))
	if err != nil {
		return policy, fmt.Errorf("release policy missing: create .pose/release-policy.json")
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return policy, fmt.Errorf("invalid release policy: %w", err)
	}
	if policy.SchemaVersion != 1 || policy.AdoptedAt == "" || policy.Provider == "" || policy.Repository == "" {
		return policy, fmt.Errorf("invalid release policy: schema_version, adopted_at, provider and repository are required")
	}
	return policy, nil
}

func LoadReleaseFragments(dir string) ([]ReleaseFragment, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReleaseFragment{}, nil
	}
	if err != nil {
		return nil, err
	}
	fragments := []ReleaseFragment{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" || entry.Name() == ".gitkeep" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("release fragment %s must not be a symlink", entry.Name())
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fm, body := SplitFrontmatter(string(raw))
		body = strings.TrimSpace(stripHTMLComments(body))
		spec, category := strings.TrimSpace(fm["spec"]), strings.TrimSpace(fm["category"])
		// Name the defect and the path: "malformed" alone left an operator
		// guessing which of three fields was wrong, in a file the message did
		// not locate.
		var problems []string
		if spec == "" {
			problems = append(problems, "missing `spec:` in frontmatter")
		}
		if !map[string]bool{"added": true, "changed": true, "fixed": true, "removed": true, "security": true, "deprecated": true}[category] {
			if category == "" {
				problems = append(problems, "missing `category:` (added|changed|fixed|removed|security|deprecated)")
			} else {
				problems = append(problems, fmt.Sprintf("invalid `category: %s` (want added|changed|fixed|removed|security|deprecated)", category))
			}
		}
		if body == "" {
			problems = append(problems, "empty body below the frontmatter")
		}
		if len(problems) > 0 {
			return nil, fmt.Errorf("malformed release fragment %s: %s", path, strings.Join(problems, "; "))
		}
		if seen[spec] {
			return nil, fmt.Errorf("duplicate release fragment for spec %s", spec)
		}
		seen[spec] = true
		fragments = append(fragments, ReleaseFragment{Spec: spec, Category: category, Breaking: fm["breaking"] == "true", Refs: fm["refs"], Body: body, Path: entry.Name(), Digest: ReleaseDigest(string(raw))})
	}
	sort.Slice(fragments, func(i, j int) bool { return fragments[i].Spec < fragments[j].Spec })
	return fragments, nil
}

func RenderReleaseNotes(version string, fragments []ReleaseFragment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# POSE %s\n\n", version)
	order := []string{"security", "removed", "deprecated", "added", "changed", "fixed"}
	labels := map[string]string{"security": "Security", "removed": "Removed", "deprecated": "Deprecated", "added": "Added", "changed": "Changed", "fixed": "Fixed"}
	for _, category := range order {
		items := []ReleaseFragment{}
		for _, fragment := range fragments {
			if fragment.Category == category {
				items = append(items, fragment)
			}
		}
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", labels[category])
		for _, fragment := range items {
			fmt.Fprintf(&b, "- %s (%s)\n", strings.ReplaceAll(fragment.Body, "\n", " "), fragment.Spec)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func LoadReleaseManifest(root, version string) (*ReleaseManifest, error) {
	if err := ValidateReleaseVersion(version); err != nil {
		return nil, err
	}
	var manifest ReleaseManifest
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "releases", version, "manifest.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func LoadReleaseEvents(root, version string) ([]ReleaseEvent, error) {
	path := filepath.Join(root, ".pose", "releases", version, "events.jsonl")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []ReleaseEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	events := []ReleaseEvent{}
	for i, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event ReleaseEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("event line %d: %w", i+1, err)
		}
		events = append(events, event)
	}
	return events, nil
}

func ProjectRelease(manifest *ReleaseManifest, events []ReleaseEvent) ReleaseProjection {
	p := ReleaseProjection{Version: manifest.Version, State: "prepared", Manifest: manifest, Events: events, Gaps: []string{}}
	publicationDigest := ""
	for _, event := range events {
		switch event.State {
		case "tagged":
			if p.State == "prepared" || p.State == "failed" {
				p.State = "tagged"
			}
		case "published":
			if p.State == "tagged" || p.State == "failed" {
				p.State = "published"
				publicationDigest = event.EvidenceDigest
			} else {
				p.Gaps = append(p.Gaps, "published event lacks tagged predecessor")
			}
		case "verified":
			if p.State == "published" && event.Evidence.Publication == publicationDigest {
				p.State = "verified"
			} else {
				p.Gaps = append(p.Gaps, "verified evidence is not bound to current publication")
			}
		case "failed":
			p.State = "failed"
		case "yanked":
			if p.State == "published" || p.State == "verified" {
				p.State = "yanked"
			} else {
				p.Gaps = append(p.Gaps, "yanked event lacks published predecessor")
			}
		default:
			p.Gaps = append(p.Gaps, "unknown event state "+event.State)
		}
	}
	return p
}

func (s Store) GetReleaseStatus(version string) (*ReleaseStatus, error) {
	if version != "" {
		if err := ValidateReleaseVersion(version); err != nil {
			return nil, err
		}
	}
	pending, err := LoadReleaseFragments(filepath.Join(s.Root, ".pose", "changelogs", "unreleased"))
	if err != nil {
		return nil, err
	}
	status := &ReleaseStatus{Pending: pending, Releases: []ReleaseProjection{}}
	base := filepath.Join(s.Root, ".pose", "releases")
	entries, _ := os.ReadDir(base)
	for _, entry := range entries {
		if !entry.IsDir() || ValidateReleaseVersion(entry.Name()) != nil || version != "" && entry.Name() != version {
			continue
		}
		manifest, err := LoadReleaseManifest(s.Root, entry.Name())
		if err != nil {
			return nil, err
		}
		events, err := LoadReleaseEvents(s.Root, entry.Name())
		if err != nil {
			return nil, err
		}
		projection := ProjectRelease(manifest, events)
		if projection.State == "prepared" {
			cmd := exec.Command("git", "rev-list", "-n", "1", entry.Name()+"^{}")
			cmd.Dir = s.Root
			if commit, err := cmd.Output(); err == nil && regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(strings.TrimSpace(string(commit))) {
				projection.State = "tagged"
			}
		}
		status.Releases = append(status.Releases, projection)
	}
	sort.Slice(status.Releases, func(i, j int) bool {
		return compareReleaseVersions(status.Releases[i].Version, status.Releases[j].Version) > 0
	})
	return status, nil
}

func compareReleaseVersions(a, b string) int {
	parse := func(v string) [3]int {
		m := releaseVersionRE.FindStringSubmatch(v)
		var n [3]int
		if len(m) > 3 {
			for i := 0; i < 3; i++ {
				n[i], _ = strconv.Atoi(m[i+1])
			}
		}
		return n
	}
	x, y := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if x[i] < y[i] {
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
	}
	return strings.Compare(a, b)
}

func NewReleaseManifest(version, previous, preparedAt string, fragments []ReleaseFragment, policy ReleasePolicy, versionEvidence map[string]string) ReleaseManifest {
	specs := []string{}
	categories := map[string]int{}
	breaking := false
	for _, f := range fragments {
		specs = append(specs, f.Spec)
		categories[f.Category]++
		breaking = breaking || f.Breaking
	}
	notes := RenderReleaseNotes(version, fragments)
	input := struct {
		Version   string            `json:"version"`
		Previous  string            `json:"previous"`
		Fragments []ReleaseFragment `json:"fragments"`
		Policy    string            `json:"policy"`
		Evidence  map[string]string `json:"evidence"`
	}{version, previous, fragments, ReleaseDigest(policy), versionEvidence}
	return ReleaseManifest{SchemaVersion: 1, Version: version, PreviousVersion: previous, PreparedAt: preparedAt, Specs: specs, Categories: categories, Fragments: fragments, Breaking: breaking, NotesDigest: ReleaseDigest(notes), ReleaseInputDigest: ReleaseDigest(input), PolicyDigest: ReleaseDigest(policy), VersionEvidence: versionEvidence}
}

func CanonicalJSON(value any) []byte {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return append(raw, '\n')
}

func NewReleaseEvent(version, state string, evidence ReleaseEvidence, now time.Time) ReleaseEvent {
	return ReleaseEvent{SchemaVersion: 1, Version: version, State: state, RecordedAt: now.UTC().Format(time.RFC3339), Evidence: evidence, EvidenceDigest: ReleaseDigest(evidence)}
}
