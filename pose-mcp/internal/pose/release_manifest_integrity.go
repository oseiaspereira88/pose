package pose

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Spec pose-release-manifest-tag-integrity: a release manifest is valid only if its shape and
// the projections it repeats agree with the fragments it freezes, and a tag is a release only
// if the tree it points at holds that manifest, its notes and its fragments.

// ReleaseManifestSchemaVersion is the manifest schema this engine reads.
const ReleaseManifestSchemaVersion = 1

// ReleaseManifestIssue is one reason a manifest is not valid, localized to its field.
type ReleaseManifestIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (i ReleaseManifestIssue) String() string { return i.Field + ": " + i.Message }

var releaseDigestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidateReleaseManifestBytes decodes raw as the manifest of version and checks what can be
// checked from the manifest and nothing else: the schema and its shapes, unknown fields,
// the version, the paths, and the projections (specs, categories, breaking) and the input digest it
// repeats, recomposed from the fragments it freezes. A parseable JSON object is not a manifest.
func ValidateReleaseManifestBytes(raw []byte, version string) (*ReleaseManifest, []ReleaseManifestIssue) {
	var issues []ReleaseManifestIssue
	add := func(field, msg string) { issues = append(issues, ReleaseManifestIssue{Field: field, Message: msg}) }
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest ReleaseManifest
	if err := decoder.Decode(&manifest); err != nil {
		add("manifest", "does not decode as a release manifest: "+err.Error())
		return nil, issues
	}
	if decoder.More() {
		add("manifest", "has content after the manifest object")
	}
	if manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		add("schema_version", fmt.Sprintf("%d is not a schema this engine reads (supported: %d)", manifest.SchemaVersion, ReleaseManifestSchemaVersion))
		return &manifest, issues
	}
	if err := ValidateReleaseVersion(manifest.Version); err != nil {
		add("version", err.Error())
	} else if version != "" && manifest.Version != version {
		add("version", fmt.Sprintf("%q is not the release of this directory (%q)", manifest.Version, version))
	}
	if manifest.PreviousVersion != "" {
		if err := ValidateReleaseVersion(manifest.PreviousVersion); err != nil {
			add("previous_version", err.Error())
		} else if manifest.PreviousVersion == manifest.Version {
			add("previous_version", "equals the release itself")
		}
	}
	if _, err := time.Parse(time.RFC3339, manifest.PreparedAt); err != nil {
		add("prepared_at", "is not RFC3339")
	}
	for _, field := range []struct{ name, value string }{{"notes_digest", manifest.NotesDigest}, {"release_input_digest", manifest.ReleaseInputDigest}, {"policy_digest", manifest.PolicyDigest}} {
		if !releaseDigestRE.MatchString(field.value) {
			add(field.name, "is not a sha256 digest")
		}
	}
	if manifest.VersionEvidence == nil {
		add("version_evidence", "is absent")
	}
	for key := range manifest.VersionEvidence {
		if key == "" {
			add("version_evidence", "has an empty key")
		}
	}
	if source := manifest.VersionEvidence["source"]; source != "" {
		if filepath.IsAbs(source) || strings.Contains(source, "\\") || path.Clean(source) != source || strings.HasPrefix(source, "../") {
			add("version_evidence.source", "is not a clean project-relative path")
		}
	}
	specs, categories, breaking := []string{}, map[string]int{}, false
	seen := map[string]bool{}
	for i, f := range manifest.Fragments {
		field := fmt.Sprintf("fragments[%d]", i)
		switch {
		case f.Spec == "":
			add(field+".spec", "is empty")
		case seen[f.Spec]:
			add(field+".spec", fmt.Sprintf("%q is frozen more than once", f.Spec))
		}
		seen[f.Spec] = true
		if f.Path == "" || strings.ContainsAny(f.Path, `/\`) || f.Path == "." || f.Path == ".." {
			add(field+".path", "is not a file name inside the release directory")
		}
		if !releaseDigestRE.MatchString(f.Digest) {
			add(field+".digest", "is not a sha256 digest")
		}
		if f.Category == "" {
			add(field+".category", "is empty")
		}
		specs = append(specs, f.Spec)
		categories[f.Category]++
		breaking = breaking || f.Breaking
	}
	if !equalStrings(manifest.Specs, specs) {
		add("specs", "differs from the specs of the frozen fragments")
	}
	if !equalCounts(manifest.Categories, categories) {
		add("categories", "differs from the categories of the frozen fragments")
	}
	if manifest.Breaking != breaking {
		add("breaking", "differs from the breaking flag of the frozen fragments")
	}
	// The input digest is composed of the manifest's own fields: it can be recomposed without
	// the policy or the notes, and a manifest whose digest does not follow from its fields was
	// edited after it was prepared.
	input := struct {
		Version   string            `json:"version"`
		Previous  string            `json:"previous"`
		Fragments []ReleaseFragment `json:"fragments"`
		Policy    string            `json:"policy"`
		Evidence  map[string]string `json:"evidence"`
	}{manifest.Version, manifest.PreviousVersion, manifest.Fragments, manifest.PolicyDigest, manifest.VersionEvidence}
	if ReleaseDigest(input) != manifest.ReleaseInputDigest {
		add("release_input_digest", "does not follow from the version, the fragments, the policy digest and the version evidence")
	}
	return &manifest, issues
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Why a tag could not be read as a release.
var (
	ErrReleaseTagMissing      = errors.New("pose: the release tag does not exist")
	ErrReleaseSnapshotMissing = errors.New("pose: the tag's tree does not hold the release snapshot")
)

// ReleaseSnapshot is what the tree a tag points at says about its release.
type ReleaseSnapshot struct {
	Tag       string                 `json:"tag"`
	Commit    string                 `json:"commit"`
	Annotated bool                   `json:"annotated"`
	Manifest  *ReleaseManifest       `json:"manifest,omitempty"`
	Issues    []ReleaseManifestIssue `json:"issues,omitempty"`
	Gaps      []string               `json:"gaps,omitempty"`
	// Limits say what could not be re-verified from the snapshot, which is not a failure.
	Limits []string `json:"limits,omitempty"`
}

// Intact reports a snapshot with a valid manifest and nothing contradicting it.
func (s ReleaseSnapshot) Intact() bool {
	return s.Manifest != nil && len(s.Issues) == 0 && len(s.Gaps) == 0
}

// ResolveReleaseTag peels a tag to the commit it names: a lightweight tag is that commit, an
// annotated one is the object it points at. A name that is not a tag is not resolved.
func ResolveReleaseTag(root, tag string) (commit string, annotated bool, err error) {
	if ValidateReleaseVersion(tag) != nil {
		return "", false, fmt.Errorf("pose: %q is not a release tag", tag)
	}
	kind, err := gitOutput(root, "cat-file", "-t", "refs/tags/"+tag)
	if err != nil {
		return "", false, fmt.Errorf("%w: %s", ErrReleaseTagMissing, tag)
	}
	out, err := gitOutput(root, "rev-parse", "--verify", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		return "", false, fmt.Errorf("%w: %s does not point at a commit", ErrReleaseTagMissing, tag)
	}
	return strings.TrimSpace(out), strings.TrimSpace(kind) == "tag", nil
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pose: git %s: %s", args[0], strings.TrimSpace(stderr.String()+" "+err.Error()))
	}
	return string(out), nil
}

// gitBlobAt reads one file of a commit's tree; a path that is not there is os.ErrNotExist.
func gitBlobAt(root, commit, rel string) ([]byte, error) {
	cmd := exec.Command("git", "-C", root, "cat-file", "blob", commit+":"+rel)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "does not exist") || strings.Contains(stderr.String(), "exists on disk, but not in") || strings.Contains(stderr.String(), "Not a valid object name") {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("pose: git cat-file: %s", strings.TrimSpace(stderr.String()+" "+err.Error()))
	}
	return out, nil
}

func gitTreeNames(root, commit, dir string) ([]string, error) {
	out, err := gitOutput(root, "ls-tree", "--name-only", "-z", commit+":"+dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// ReadReleaseSnapshot reads the release from the tree its tag points at, never from the
// working tree: manifest, notes and frozen fragments are blobs of the tagged commit. A tag whose
// tree holds no manifest is refused whatever the working tree holds. It does not compare the
// tagged commit with any other: a later commit, provider evidence or the next cycle cannot make a correct
// cut stale.
func ReadReleaseSnapshot(root, tag string) (ReleaseSnapshot, error) {
	commit, annotated, err := ResolveReleaseTag(root, tag)
	if err != nil {
		return ReleaseSnapshot{Tag: tag}, err
	}
	snap := ReleaseSnapshot{Tag: tag, Commit: commit, Annotated: annotated}
	manifestPath := ".pose/releases/" + tag + "/manifest.json"
	raw, err := gitBlobAt(root, commit, manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return snap, fmt.Errorf("%w: %s has no %s at %s", ErrReleaseSnapshotMissing, tag, manifestPath, commit[:12])
	}
	if err != nil {
		return snap, err
	}
	manifest, issues := ValidateReleaseManifestBytes(raw, tag)
	snap.Manifest, snap.Issues = manifest, issues
	if manifest == nil || manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		return snap, nil
	}
	// Notes.
	notes, err := gitBlobAt(root, commit, ".pose/changelogs/"+tag+".md")
	switch {
	case errors.Is(err, os.ErrNotExist):
		snap.Gaps = append(snap.Gaps, "canonical notes are missing from the tagged tree")
	case err != nil:
		snap.Gaps = append(snap.Gaps, "canonical notes could not be read from the tagged tree: "+err.Error())
	case ReleaseDigest(string(notes)) != manifest.NotesDigest:
		snap.Gaps = append(snap.Gaps, "canonical notes digest differs from the manifest in the tagged tree")
	}
	// Frozen fragments.
	frozen := map[string]bool{}
	for _, f := range manifest.Fragments {
		frozen[f.Path] = true
		blob, ferr := gitBlobAt(root, commit, ".pose/changelogs/"+tag+"/"+f.Path)
		switch {
		case errors.Is(ferr, os.ErrNotExist):
			snap.Gaps = append(snap.Gaps, "fragment "+f.Spec+" is missing from the tagged tree")
		case ferr != nil:
			snap.Gaps = append(snap.Gaps, "fragment "+f.Spec+" could not be read from the tagged tree: "+ferr.Error())
		case ReleaseDigest(string(blob)) != f.Digest:
			snap.Gaps = append(snap.Gaps, "fragment "+f.Spec+" differs from the digest the manifest froze")
		}
	}
	if names, nerr := gitTreeNames(root, commit, ".pose/changelogs/"+tag); nerr == nil {
		sort.Strings(names)
		for _, name := range names {
			if !frozen[name] {
				snap.Gaps = append(snap.Gaps, "the tagged tree holds fragment "+name+" that the manifest does not freeze")
			}
		}
	}
	// Version evidence: re-read from the tagged tree when the manifest carries enough to do so.
	if gap, limit := snapshotVersionEvidence(root, commit, manifest); gap != "" {
		snap.Gaps = append(snap.Gaps, gap)
	} else if limit != "" {
		snap.Limits = append(snap.Limits, limit)
	}
	return snap, nil
}

// snapshotVersionEvidence re-verifies the version a manifest recorded against the file of the
// tagged tree that holds it. Only a text source can be recomputed from the evidence the manifest
// carries; any other is reported as a limit, not guessed.
func snapshotVersionEvidence(root, commit string, manifest *ReleaseManifest) (gap, limit string) {
	source, kind, value := manifest.VersionEvidence["source"], manifest.VersionEvidence["kind"], manifest.VersionEvidence["value"]
	if source == "" {
		return "", "the manifest records no version source to re-verify"
	}
	if kind != ReleaseVersionSourceText {
		return "", "the version evidence is not re-verifiable from the manifest alone (kind " + fmt.Sprintf("%q", kind) + ")"
	}
	raw, err := gitBlobAt(root, commit, source)
	if err != nil {
		return "version source " + source + " could not be read from the tagged tree", ""
	}
	tmp, err := os.MkdirTemp("", "pose-release-snapshot-*")
	if err != nil {
		return "", "the version source could not be re-read: " + err.Error()
	}
	defer os.RemoveAll(tmp)
	target := filepath.Join(tmp, filepath.FromSlash(source))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", "the version source could not be re-read: " + err.Error()
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return "", "the version source could not be re-read: " + err.Error()
	}
	got, err := ReadReleaseVersionSource(tmp, ReleaseVersionSource{Path: source, Kind: kind})
	if err != nil {
		return "version source " + source + " in the tagged tree is not readable: " + err.Error(), ""
	}
	if got != value {
		return fmt.Sprintf("the tagged tree's version source says %s, the manifest recorded %s", got, value), ""
	}
	return "", ""
}
