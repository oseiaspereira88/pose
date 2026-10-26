package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// A release version is a Semantic Versioning 2.0 version with a leading v and no build
// metadata: numeric prerelease identifiers carry no leading zero and none is empty.
var releaseVersionRE = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$`)

type ReleasePolicy struct {
	SchemaVersion int    `json:"schema_version"`
	AdoptedAt     string `json:"adopted_at"`
	TagPattern    string `json:"tag_pattern"`
	AllowEmpty    bool   `json:"allow_empty"`
	LegacyCutoff  string `json:"legacy_cutoff"`
	Provider      string `json:"provider"`
	Repository    string `json:"repository"`
	RequireVerify bool   `json:"require_verification"`
	// VersionSource names the project file that holds the project's own version.
	// It is a pointer with omitempty so a policy that leaves it unset serializes,
	// and therefore digests, exactly as it did before the field existed.
	VersionSource *ReleaseVersionSource `json:"version_source,omitempty"`
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
	// Snapshot is what the tree of the release's tag says, read from the tag and not from
	// the working tree. It is absent while the release has no tag.
	Snapshot *ReleaseSnapshot `json:"snapshot,omitempty"`
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

// ErrReleasePolicyMissing marks a release policy proven not to be there, as opposed to one that
// could not be read.
var ErrReleasePolicyMissing = errors.New("release policy missing")

func LoadReleasePolicy(root string) (ReleasePolicy, error) {
	return LoadReleasePolicyIn(NativeFiles(), root)
}

func LoadReleasePolicyIn(files Files, root string) (ReleasePolicy, error) {
	var policy ReleasePolicy
	primary := filepath.Join(root, ".pose", "policy", "release.json")
	absent, absentErr := ProvenAbsentIn(files, primary)
	if absentErr != nil {
		return policy, fmt.Errorf("release policy could not be read: %w", absentErr)
	}
	path := primary
	if absent {
		// The legacy place is read only when the current one is proven not to be there.
		path = filepath.Join(root, ".pose", "release-policy.json")
		legacyAbsent, legacyErr := ProvenAbsentIn(files, path)
		if legacyErr != nil {
			return policy, fmt.Errorf("release policy could not be read: %w", legacyErr)
		}
		if legacyAbsent {
			return policy, fmt.Errorf("%w: create .pose/policy/release.json", ErrReleasePolicyMissing)
		}
	}
	raw, err := files.ReadFile(path)
	if err != nil {
		return policy, fmt.Errorf("release policy could not be read: %w", err)
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return policy, fmt.Errorf("invalid release policy: %w", err)
	}
	if policy.SchemaVersion != 1 {
		return policy, fmt.Errorf("invalid release policy: schema_version must be 1")
	}
	if err := ValidateReleasePolicy(policy); err != nil {
		return policy, err
	}
	return policy, nil
}

// ValidateReleasePolicy checks a release policy without touching the file system, so the policy
// a cut was made under can be held to the contract it was written for.
func ValidateReleasePolicy(policy ReleasePolicy) error {
	if policy.SchemaVersion != 1 {
		return fmt.Errorf("invalid release policy: schema_version must be 1")
	}
	if policy.AdoptedAt != "" && (policy.Provider == "" || policy.Repository == "") {
		return fmt.Errorf("invalid release policy: provider and repository are required once adopted")
	}
	if policy.VersionSource != nil {
		if err := policy.VersionSource.Validate(); err != nil {
			return fmt.Errorf("invalid release policy: version_source: %w", err)
		}
	}
	if policy.LegacyCutoff != "" {
		// A cutoff that is not a version orders after every version: it would hide every tag.
		if err := ValidateReleaseVersion(policy.LegacyCutoff); err != nil {
			return fmt.Errorf("invalid release policy: legacy_cutoff: %w", err)
		}
	}
	return nil
}

func LoadReleaseFragments(dir string, policy ChangelogPolicy) ([]ReleaseFragment, error) {
	return LoadReleaseFragmentsIn(NativeFiles(), dir, policy)
}

func LoadReleaseFragmentsIn(files Files, dir string, policy ChangelogPolicy) ([]ReleaseFragment, error) {
	return loadReleaseFragments(files, dir, policy, false)
}

// LoadArchivedFragmentIdentities reads the fragments of a release that was cut, for what they are
// (a spec each, once) and not for the categories the project allows today: the categories of a cut
// are the ones its own policy held, and a later change of the policy does not change the cut.
func LoadArchivedFragmentIdentities(dir string) ([]ReleaseFragment, error) {
	return LoadArchivedFragmentIdentitiesIn(NativeFiles(), dir)
}

func LoadArchivedFragmentIdentitiesIn(files Files, dir string) ([]ReleaseFragment, error) {
	return loadReleaseFragments(files, dir, ChangelogPolicy{}, true)
}

func loadReleaseFragments(files Files, dir string, policy ChangelogPolicy, anyCategory bool) ([]ReleaseFragment, error) {
	entries, err := files.ReadDir(dir)
	if err != nil {
		if absent, absentErr := ProvenAbsentIn(files, dir); absentErr == nil && absent {
			return []ReleaseFragment{}, nil
		} else if absentErr != nil {
			return nil, absentErr
		}
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
		raw, err := files.ReadFile(path)
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
		if !anyCategory && !policy.ValidCategories()[category] {
			if category == "" {
				problems = append(problems, "missing `category:` ("+policy.CategoryList()+")")
			} else {
				problems = append(problems, fmt.Sprintf("invalid `category: %s` (want %s)", category, policy.CategoryList()))
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

func RenderReleaseNotes(version string, fragments []ReleaseFragment, policy ChangelogPolicy) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# POSE %s\n\n", version)
	// A release that introduces a governance contract says so before it says
	// anything else, because adopting one costs something an operator cannot see
	// from the fragment list.
	//
	// What it costs depends on how the adoption is written, and the first draft
	// of this got it wrong in the direction that matters: it told every reader of
	// every future contract to upgrade in lockstep. A contract carried by a
	// top-level policy key does refuse older strict decoders. A contract recorded
	// as an id inside `contract_adoptions` does not — the map is a key those
	// engines already model, so the policy still loads and they simply do not
	// apply the contract. The registry knows which kind each one is.
	if contracts := ContractsIntroducedIn(version); len(contracts) > 0 {
		subject, pronoun := "a governance contract", "it"
		if len(contracts) > 1 {
			subject, pronoun = fmt.Sprintf("%d governance contracts", len(contracts)), "them"
		}
		b.WriteString("## Compatibility\n\n")
		fmt.Fprintf(&b, "This release introduces %s. An engine older than %s does not know %s, so it never applies %s: reviews it performs are judged by the rules it has.\n\n", subject, version, pronoun, pronoun)
		for _, contract := range contracts {
			fmt.Fprintf(&b, "- `%s` — %s\n", contract.ID, contract.Summary)
			if contract.AdoptionAddsATopLevelKey() {
				fmt.Fprintf(&b, "  Adopting it writes `%s` at the top level of the review policy. Engines before %s refuse a policy carrying a key they do not model, so they stop reading this repository altogether until they are updated — every tool reading it has to move together.\n", contract.LegacyField, StrictPolicyDecoderDroppedIn)
			} else {
				fmt.Fprintf(&b, "  Adopting it records the id inside `contract_adoptions`, which older engines already model, so they keep reading the repository — they only stop enforcing this contract.\n")
			}
		}
		b.WriteString("\n")
	}
	for _, category := range policy.RenderOrder() {
		items := []ReleaseFragment{}
		for _, fragment := range fragments {
			if fragment.Category == category {
				items = append(items, fragment)
			}
		}
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", CategoryLabel(category))
		for _, fragment := range items {
			fmt.Fprintf(&b, "- %s (%s)\n", strings.ReplaceAll(fragment.Body, "\n", " "), fragment.Spec)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func LoadReleaseManifest(root, version string) (*ReleaseManifest, error) {
	return LoadReleaseManifestIn(NativeFiles(), root, version)
}

func LoadReleaseManifestIn(files Files, root, version string) (*ReleaseManifest, error) {
	if err := ValidateReleaseVersion(version); err != nil {
		return nil, err
	}
	raw, err := files.ReadFile(filepath.Join(root, ".pose", "releases", version, "manifest.json"))
	if err != nil {
		return nil, err
	}
	manifest, issues := ValidateReleaseManifestBytes(raw, version)
	if len(issues) > 0 {
		parts := make([]string, 0, len(issues))
		for _, issue := range issues {
			parts = append(parts, issue.String())
		}
		return nil, fmt.Errorf("pose: release manifest of %s is not valid: %s", version, strings.Join(parts, "; "))
	}
	return manifest, nil
}

func LoadReleaseEvents(root, version string) ([]ReleaseEvent, error) {
	return LoadReleaseEventsIn(NativeFiles(), root, version)
}

func LoadReleaseEventsIn(files Files, root, version string) ([]ReleaseEvent, error) {
	path := filepath.Join(root, ".pose", "releases", version, "events.jsonl")
	raw, err := files.ReadFile(path)
	if err != nil {
		if absent, absentErr := ProvenAbsentIn(files, path); absentErr == nil && absent {
			return []ReleaseEvent{}, nil
		} else if absentErr != nil {
			return nil, absentErr
		}
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

// Sound reports a release whose record can be relied on as evidence: no gap in its events or in
// what the tag holds, and, once it has a tag, a snapshot that is intact. A state an event
// claimed on top of a snapshot that is not intact is a claim, not evidence.
func (p ReleaseProjection) Sound() bool {
	if len(p.Gaps) > 0 {
		return false
	}
	if p.Snapshot != nil && !p.Snapshot.Intact() {
		return false
	}
	switch p.State {
	case "tagged", "published", "verified", "yanked":
		return p.Snapshot != nil
	}
	return true
}

func (s Store) GetReleaseStatus(version string) (*ReleaseStatus, error) {
	if version != "" {
		if err := ValidateReleaseVersion(version); err != nil {
			return nil, err
		}
	}
	pending, err := LoadReleaseFragmentsIn(s.fsys(), filepath.Join(s.Root, ".pose", "changelogs", "unreleased"), LoadChangelogPolicyIn(s.fsys(), s.Root))
	if err != nil {
		return nil, err
	}
	status := &ReleaseStatus{Pending: pending, Releases: []ReleaseProjection{}}
	base := filepath.Join(s.Root, ".pose", "releases")
	entries, err := s.fsys().ReadDir(base)
	if err != nil {
		// A directory that cannot be listed (a file where it belongs, no access, a link to a
		// place that was lost, a lost ancestor) is not a project with no releases: only a
		// proven absence is.
		if absent, absentErr := ProvenAbsentIn(s.fsys(), base); absentErr != nil || !absent {
			if absentErr == nil {
				absentErr = err
			}
			return nil, fmt.Errorf("pose: the release directory could not be read: %w", absentErr)
		}
	}
	inWorktree := map[string]bool{}
	for _, entry := range entries {
		if ValidateReleaseVersion(entry.Name()) == nil && !entry.IsDir() && (version == "" || entry.Name() == version) {
			// A release written as a file or a link is not a release this record can read.
			return nil, fmt.Errorf("pose: the release %s is %s, not a directory", entry.Name(), entry.Type())
		}
		if !entry.IsDir() || ValidateReleaseVersion(entry.Name()) != nil || version != "" && entry.Name() != version {
			continue
		}
		inWorktree[entry.Name()] = true
		manifest, err := LoadReleaseManifestIn(s.fsys(), s.Root, entry.Name())
		if err != nil {
			return nil, err
		}
		events, err := LoadReleaseEventsIn(s.fsys(), s.Root, entry.Name())
		if err != nil {
			return nil, err
		}
		projection := ProjectRelease(manifest, events)
		_, _, tagErr := ResolveReleaseTag(s.Root, entry.Name())
		if tagErr != nil && errors.Is(tagErr, ErrReleaseTagMissing) {
			// No tag yet: the cut is a candidate, and the files it froze must still be there.
			gaps, _ := ReleaseCandidateGapsIn(s.fsys(), s.Root, entry.Name(), manifest)
			projection.Gaps = append(projection.Gaps, gaps...)
			if projection.State != "prepared" && projection.State != "failed" {
				// The events say a tag exists; the repository says it does not.
				projection.Gaps = append(projection.Gaps, "the events record "+projection.State+", and the tag "+entry.Name()+" does not exist")
			}
		}
		if tagErr == nil {
			snap, snapErr := ReadReleaseSnapshot(s.Root, entry.Name())
			projection.Snapshot = &snap
			switch {
			case snapErr != nil:
				// A tag whose tree holds no snapshot is not a tagged release, whatever the working tree holds.
				projection.Gaps = append(projection.Gaps, "tag "+entry.Name()+" exists but its tree holds no release snapshot: "+snapErr.Error())
			default:
				for _, issue := range snap.Issues {
					projection.Gaps = append(projection.Gaps, "tag snapshot manifest "+issue.String())
				}
				for _, gap := range snap.Gaps {
					projection.Gaps = append(projection.Gaps, "tag snapshot: "+gap)
				}
				projection.Gaps = append(projection.Gaps, TaggedReleaseWorktreeGapsIn(s.fsys(), s.Root, entry.Name(), manifest, snap)...)
				if projection.State == "prepared" && snap.Intact() {
					projection.State = "tagged"
				}
			}
		} else if !errors.Is(tagErr, ErrReleaseTagMissing) {
			projection.Gaps = append(projection.Gaps, "the tag of "+entry.Name()+" could not be resolved: "+tagErr.Error())
		}
		status.Releases = append(status.Releases, projection)
	}
	// A release whose directory is gone from the working tree is still a release if its tag holds
	// the snapshot: removing the files does not remove the release from the record. A tag that
	// holds none is reported, not dropped, and a listing that fails is not an empty one.
	tags, tagsErr := ListReleaseTags(s.Root)
	if tagsErr != nil && !gitNotARepository(s.Root) {
		return nil, fmt.Errorf("pose: the release tags could not be listed: %w", tagsErr)
	}
	cutoff := ""
	adopted := false
	if policy, perr := LoadReleasePolicyIn(s.fsys(), s.Root); perr == nil {
		cutoff = policy.LegacyCutoff
		adopted = policy.AdoptedAt != ""
	} else if errors.Is(perr, ErrReleasePolicyMissing) {
		// Proven gone: a cut in the working tree still says the project adopted governed releases.
		for _, entry := range entries {
			if ValidateReleaseVersion(entry.Name()) == nil {
				adopted = true
			}
		}
	} else {
		// A policy that is there and cannot be read is not "never adopted".
		adopted = true
	}
	for _, tag := range strings.Fields(tags) {
		if ValidateReleaseVersion(tag) != nil || inWorktree[tag] || version != "" && tag != version {
			continue
		}
		if cutoff != "" && CompareReleasePrecedence(cutoff, tag) >= 0 {
			continue // before the governed releases: not a release this record claims
		}
		snap, snapErr := ReadReleaseSnapshot(s.Root, tag)
		if errors.Is(snapErr, ErrReleaseSnapshotMissing) && !adopted {
			// A tag with no manifest is a release this record claims only for a project that adopted
			// governed releases: the policy of today, or the one in the tag's own tree, says so. A
			// policy that cannot be read is not "never adopted".
			if held, heldErr := TagHoldsAdoptedReleasePolicy(s.Root, tag); heldErr == nil && !held {
				continue
			}
		}
		if snapErr != nil || snap.Manifest == nil {
			cause := "no manifest"
			if snapErr != nil {
				cause = snapErr.Error()
			}
			status.Releases = append(status.Releases, ReleaseProjection{Version: tag, State: "prepared", Events: []ReleaseEvent{},
				Gaps: []string{"tag " + tag + " exists, its release directory is missing from the working tree and its tree holds no readable snapshot: " + cause}})
			continue
		}
		projection := ProjectRelease(snap.Manifest, []ReleaseEvent{})
		projection.Snapshot = &snap
		projection.Gaps = append(projection.Gaps, "the release directory is missing from the working tree; the release is read from its tag")
		for _, issue := range snap.Issues {
			projection.Gaps = append(projection.Gaps, "tag snapshot manifest "+issue.String())
		}
		for _, gap := range snap.Gaps {
			projection.Gaps = append(projection.Gaps, "tag snapshot: "+gap)
		}
		if snap.Intact() {
			projection.State = "tagged"
		}
		status.Releases = append(status.Releases, projection)
	}
	sort.Slice(status.Releases, func(i, j int) bool {
		return compareReleaseVersions(status.Releases[i].Version, status.Releases[j].Version) > 0
	})
	return status, nil
}

// compareReleaseVersions orders two release versions by Semantic Versioning precedence.
func compareReleaseVersions(a, b string) int { return CompareReleasePrecedence(a, b) }

// CompareReleasePrecedence orders two release versions by Semantic Versioning 2.0
// precedence: the core numbers first, then a version with a prerelease before the same core
// without one, then the prerelease identifiers one by one (numbers as numbers and before
// words, words as text, a shorter list before a longer one that starts with it). A text that is
// not a release version sorts by its text after every version that is, so the order is total.
func CompareReleasePrecedence(a, b string) int {
	ma, mb := releaseVersionRE.FindStringSubmatch(a), releaseVersionRE.FindStringSubmatch(b)
	switch {
	case ma == nil && mb == nil:
		return strings.Compare(a, b)
	case ma == nil:
		return 1
	case mb == nil:
		return -1
	}
	for i := 1; i <= 3; i++ {
		if c := compareNumericString(ma[i], mb[i]); c != 0 {
			return c
		}
	}
	pa, pb := ma[4], mb[4]
	switch {
	case pa == "" && pb == "":
		return 0
	case pa == "":
		return 1
	case pb == "":
		return -1
	}
	ia, ib := strings.Split(pa, "."), strings.Split(pb, ".")
	for i := 0; i < len(ia) && i < len(ib); i++ {
		x, y := ia[i], ib[i]
		xn, yn := isNumericIdentifier(x), isNumericIdentifier(y)
		switch {
		case xn && yn:
			if c := compareNumericString(x, y); c != 0 {
				return c
			}
		case xn:
			return -1
		case yn:
			return 1
		default:
			if c := strings.Compare(x, y); c != 0 {
				return c
			}
		}
	}
	switch {
	case len(ia) < len(ib):
		return -1
	case len(ia) > len(ib):
		return 1
	}
	return 0
}

func isNumericIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// compareNumericString compares two decimal strings without leading zeros as numbers, however long.
func compareNumericString(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func NewReleaseManifest(version, previous, preparedAt string, fragments []ReleaseFragment, policy ReleasePolicy, changelog ChangelogPolicy, versionEvidence map[string]string) ReleaseManifest {
	specs := []string{}
	categories := map[string]int{}
	breaking := false
	if fragments == nil {
		// An empty release freezes an empty list, never null.
		fragments = []ReleaseFragment{}
	}
	for _, f := range fragments {
		specs = append(specs, f.Spec)
		categories[f.Category]++
		breaking = breaking || f.Breaking
	}
	notes := RenderReleaseNotes(version, fragments, changelog)
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

// GitNotARepository reports that root is not inside a Git repository at all, which is the only
// reason a failing git command says nothing about the releases there are. A `.git` that is there,
// whose gitdir was lost, is a repository that could not be read: Git words both the same way.
func GitNotARepository(root string) bool {
	_, err := gitOutputUntranslated(root, "rev-parse", "--git-dir")
	var failure *gitCommandError
	if !errors.As(err, &failure) || !strings.Contains(failure.Stderr, "not a git repository") {
		return false
	}
	abs, absErr := filepath.Abs(root)
	if absErr != nil {
		return false
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, statErr := os.Lstat(filepath.Join(dir, ".git")); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			return false
		}
		if filepath.Dir(dir) == dir {
			return true
		}
	}
}

func gitNotARepository(root string) bool { return GitNotARepository(root) }

// TaggedReleaseWorktreeGaps binds the working tree's copy of a tagged release to the cut the tag
// holds: the manifest, the notes and every archived fragment are byte for byte what the tag's
// snapshot froze, and nothing is archived that the manifest does not freeze. The tag's commit
// itself is not compared with HEAD.
func TaggedReleaseWorktreeGaps(root, target string, manifest *ReleaseManifest, snap ReleaseSnapshot) []string {
	return TaggedReleaseWorktreeGapsIn(NativeFiles(), root, target, manifest, snap)
}

func TaggedReleaseWorktreeGapsIn(files Files, root, target string, manifest *ReleaseManifest, snap ReleaseSnapshot) []string {
	if snap.Manifest == nil {
		return []string{} // reported by the snapshot
	}
	gaps := []string{}
	if rawManifest, rerr := files.ReadFile(filepath.Join(root, ".pose", "releases", target, "manifest.json")); rerr != nil {
		gaps = append(gaps, "the manifest of the working tree could not be read: "+rerr.Error())
	} else if ReleaseDigest(string(rawManifest)) != snap.ManifestDigest {
		gaps = append(gaps, "the working tree's manifest is not the manifest of the tag")
	}
	return append(gaps, ReleaseFilesGapsIn(files, root, target, manifest)...)
}

// ReleaseFilesGaps checks that the files a manifest freezes are in the working tree as it froze
// them: the notes, every archived fragment, and nothing archived that it does not freeze. It is
// the check of a prepared cut that has no tag, and the second half of the check of a tagged one.
func ReleaseFilesGaps(root, target string, manifest *ReleaseManifest) []string {
	return ReleaseFilesGapsIn(NativeFiles(), root, target, manifest)
}

func ReleaseFilesGapsIn(files Files, root, target string, manifest *ReleaseManifest) []string {
	gaps := []string{}
	notes, nerr := files.ReadFile(filepath.Join(root, ".pose", "changelogs", target+".md"))
	if nerr != nil {
		gaps = append(gaps, "canonical notes missing")
	} else if ReleaseDigest(string(notes)) != manifest.NotesDigest {
		gaps = append(gaps, "canonical notes digest mismatch")
	}
	frozen := map[string]bool{}
	for _, f := range manifest.Fragments {
		frozen[f.Path] = true
		raw, rerr := files.ReadFile(filepath.Join(root, ".pose", "changelogs", target, f.Path))
		switch {
		case rerr != nil:
			gaps = append(gaps, "fragment "+f.Spec+" could not be read from the working tree: "+rerr.Error())
		case ReleaseDigest(string(raw)) != f.Digest:
			gaps = append(gaps, "fragment digest mismatch: "+f.Spec)
		}
	}
	archiveDir := filepath.Join(root, ".pose", "changelogs", target)
	entries, derr := files.ReadDir(archiveDir)
	if derr != nil {
		if absent, absentErr := ProvenAbsentIn(files, archiveDir); absentErr == nil && absent {
			derr = nil // a release with no fragments leaves no directory
		} else if absentErr != nil {
			derr = absentErr
		}
	}
	switch {
	case derr != nil:
		gaps = append(gaps, "the archive of the working tree could not be listed: "+derr.Error())
	default:
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") && entry.Name() != "README.md" && !frozen[entry.Name()] {
				gaps = append(gaps, "the working tree archives fragment "+entry.Name()+" that the manifest does not freeze")
			}
		}
	}
	return gaps
}
