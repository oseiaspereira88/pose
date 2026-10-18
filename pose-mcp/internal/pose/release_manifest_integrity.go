package pose

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// releaseManifestFields are the exact names of the manifest schema.
var releaseManifestFields = map[string]bool{
	"schema_version": true, "version": true, "previous_version": true, "prepared_at": true, "specs": true,
	"categories": true, "fragments": true, "breaking": true, "notes_digest": true, "release_input_digest": true,
	"policy_digest": true, "version_evidence": true,
}

var releaseDigestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidateReleaseManifestBytes decodes raw as the manifest of version and checks what can be
// checked from the manifest and nothing else: the schema and its shapes, unknown fields,
// the version, the paths, and the projections (specs, categories, breaking) and the input digest it
// repeats, recomposed from the fragments it freezes. A parseable JSON object is not a manifest.
func ValidateReleaseManifestBytes(raw []byte, version string) (*ReleaseManifest, []ReleaseManifestIssue) {
	var issues []ReleaseManifestIssue
	add := func(field, msg string) { issues = append(issues, ReleaseManifestIssue{Field: field, Message: msg}) }
	// One JSON document, no repeated key, the exact names of the schema, and every field the
	// schema requires present: a struct decoder alone accepts a second object after the first,
	// the last of two repeated keys, `VERSION` for `version` and an absent `breaking`.
	if err := rejectDuplicateJSONKeysAndControls(raw); err != nil {
		add("manifest", "is not an unambiguous JSON document: "+err.Error())
		return nil, issues
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		add("manifest", "does not decode as a JSON object")
		return nil, issues
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		add("manifest", "has content after the manifest object")
		return nil, issues
	}
	for name, value := range fields {
		if !releaseManifestFields[name] {
			add(name, "is not a field of the release manifest schema (field names are exact)")
		} else if string(bytes.TrimSpace(value)) == "null" {
			add(name, "is null")
		}
	}
	for name := range releaseManifestFields {
		if _, ok := fields[name]; !ok && name != "previous_version" {
			add(name, "is required and absent")
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}
	var manifest ReleaseManifest
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&manifest); err != nil {
		add("manifest", "does not decode as a release manifest: "+err.Error())
		return nil, issues
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
		} else if ValidateReleaseVersion(manifest.Version) == nil && CompareReleasePrecedence(manifest.PreviousVersion, manifest.Version) >= 0 {
			add("previous_version", "is not older than the release by version precedence")
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
	for key := range manifest.VersionEvidence {
		if key != "source" && key != "kind" && key != "value" {
			add("version_evidence."+key, "is not part of the version evidence contract (source, kind, value)")
		}
	}
	evidence := manifest.VersionEvidence
	switch {
	case evidence["value"] == "":
		add("version_evidence.value", "is absent: the manifest records no version the cut was compared with")
	case manifest.Version != "" && evidence["value"] != manifest.Version:
		add("version_evidence.value", fmt.Sprintf("%q is not the version of the release (%q)", evidence["value"], manifest.Version))
	}
	if source := evidence["source"]; source == "" {
		add("version_evidence.source", "is absent: the manifest names no file the version came from")
	} else if filepath.IsAbs(source) || strings.Contains(source, "\\") || path.Clean(source) != source || strings.HasPrefix(source, "../") {
		add("version_evidence.source", "is not a clean project-relative path")
	}
	if kind := evidence["kind"]; kind != "" && kind != ReleaseVersionSourceText && kind != ReleaseVersionSourceJSON {
		add("version_evidence.kind", fmt.Sprintf("%q is not a version source kind (%s, %s, or none for the engine version file)", kind, ReleaseVersionSourceText, ReleaseVersionSourceJSON))
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
	Tag string `json:"tag"`
	// ManifestDigest is the digest of the manifest as the tagged tree holds it, which the
	// working tree's manifest is bound to.
	ManifestDigest string                 `json:"manifest_digest,omitempty"`
	Commit         string                 `json:"commit"`
	Annotated      bool                   `json:"annotated"`
	Manifest       *ReleaseManifest       `json:"manifest,omitempty"`
	Issues         []ReleaseManifestIssue `json:"issues,omitempty"`
	Gaps           []string               `json:"gaps,omitempty"`
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

// gitTreeNames lists the entries of one directory of a commit's tree. A directory that is not
// in the tree is exists=false, which is not an error: a release with no fragments leaves no
// directory, and Git cannot version an empty one. Anything else that goes wrong is an error, and
// so is an entry that is not a directory.
func gitTreeNames(root, commit, dir string) (names []string, exists bool, err error) {
	kind, err := gitOutput(root, "cat-file", "-t", commit+":"+dir)
	if err != nil {
		if strings.Contains(err.Error(), "Not a valid object name") || strings.Contains(err.Error(), "does not exist") {
			return nil, false, nil
		}
		return nil, false, err
	}
	if strings.TrimSpace(kind) != "tree" {
		return nil, true, fmt.Errorf("pose: %s of the tagged tree is a %s, not a directory", dir, strings.TrimSpace(kind))
	}
	out, err := gitOutput(root, "ls-tree", "--name-only", "-z", commit+":"+dir)
	if err != nil {
		return nil, true, err
	}
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names, true, nil
}

// releasePolicyAt reads the release policy the tagged tree held: the policy the cut was made
// under, not the one the project has today.
func releasePolicyAt(root, commit string) (ReleasePolicy, string, error) {
	for _, rel := range []string{".pose/policy/release.json", ".pose/release-policy.json"} {
		raw, err := gitBlobAt(root, commit, rel)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return ReleasePolicy{}, rel, err
		}
		var policy ReleasePolicy
		if err := json.Unmarshal(raw, &policy); err != nil {
			return ReleasePolicy{}, rel, fmt.Errorf("invalid release policy: %w", err)
		}
		if policy.SchemaVersion != 1 {
			return ReleasePolicy{}, rel, fmt.Errorf("invalid release policy: schema_version must be 1")
		}
		return policy, rel, nil
	}
	return ReleasePolicy{}, "", os.ErrNotExist
}

// ReadReleaseSnapshot reads the release from the tree its tag points at, never from the
// working tree: manifest, notes and frozen fragments are blobs of the tagged commit. A tag whose
// tree holds no manifest is refused whatever the working tree holds. The projections the
// manifest repeats are recomposed from the fragment blobs and the policy blob of that tree,
// not trusted because they agree with each other. It does not compare the tagged commit with
// any other: a later commit, provider evidence or the next cycle cannot make a correct cut
// stale.
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
	snap.ManifestDigest = ReleaseDigest(string(raw))
	if manifest == nil || manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		return snap, nil
	}
	gap := func(format string, args ...any) { snap.Gaps = append(snap.Gaps, fmt.Sprintf(format, args...)) }
	// Notes.
	notes, err := gitBlobAt(root, commit, ".pose/changelogs/"+tag+".md")
	switch {
	case errors.Is(err, os.ErrNotExist):
		gap("canonical notes are missing from the tagged tree")
	case err != nil:
		gap("canonical notes could not be read from the tagged tree: %v", err)
	case ReleaseDigest(string(notes)) != manifest.NotesDigest:
		gap("canonical notes digest differs from the manifest in the tagged tree")
	}
	// Frozen fragments: each is a blob of the tagged tree, with the identity, category,
	// breaking flag, references and body the manifest repeats, at a path nothing else uses.
	frozen := map[string]string{}
	specs := map[string]bool{}
	for _, f := range manifest.Fragments {
		if other, dup := frozen[f.Path]; dup {
			gap("fragments %s and %s share the file %s", other, f.Spec, f.Path)
		}
		frozen[f.Path] = f.Spec
		if specs[f.Spec] {
			gap("fragment spec %s is frozen more than once", f.Spec)
		}
		specs[f.Spec] = true
		blob, ferr := gitBlobAt(root, commit, ".pose/changelogs/"+tag+"/"+f.Path)
		switch {
		case errors.Is(ferr, os.ErrNotExist):
			gap("fragment %s is missing from the tagged tree", f.Spec)
			continue
		case ferr != nil:
			gap("fragment %s could not be read from the tagged tree: %v", f.Spec, ferr)
			continue
		case ReleaseDigest(string(blob)) != f.Digest:
			gap("fragment %s differs from the digest the manifest froze", f.Spec)
			continue
		}
		fm, body := SplitFrontmatter(string(blob))
		body = strings.TrimSpace(stripHTMLComments(body))
		switch {
		case strings.TrimSpace(fm["spec"]) != f.Spec:
			gap("fragment %s names spec %q in the tagged tree", f.Spec, strings.TrimSpace(fm["spec"]))
		case strings.TrimSpace(fm["category"]) == "" || strings.TrimSpace(fm["category"]) != f.Category:
			gap("fragment %s has category %q in the tagged tree, the manifest says %q", f.Spec, strings.TrimSpace(fm["category"]), f.Category)
		case (fm["breaking"] == "true") != f.Breaking:
			gap("fragment %s differs from the manifest in its breaking flag", f.Spec)
		case fm["refs"] != f.Refs:
			gap("fragment %s differs from the manifest in its references", f.Spec)
		case body == "":
			gap("fragment %s has no body in the tagged tree", f.Spec)
		case body != f.Body:
			gap("fragment %s differs from the manifest in its body", f.Spec)
		}
	}
	names, exists, lerr := gitTreeNames(root, commit, ".pose/changelogs/"+tag)
	switch {
	case lerr != nil:
		gap("the fragments of the tagged tree could not be listed: %v", lerr)
	case !exists && len(manifest.Fragments) > 0:
		gap("the tagged tree holds no fragment directory, the manifest freezes %d", len(manifest.Fragments))
	default:
		sort.Strings(names)
		for _, name := range names {
			if _, ok := frozen[name]; !ok {
				gap("the tagged tree holds fragment %s that the manifest does not freeze", name)
			}
		}
	}
	// The policy the cut was made under.
	policy, policyPath, perr := releasePolicyAt(root, commit)
	switch {
	case errors.Is(perr, os.ErrNotExist):
		gap("the tagged tree holds no release policy, so the manifest's policy digest cannot be tied to the policy of the cut")
	case perr != nil:
		gap("the release policy of the tagged tree (%s) could not be read: %v", policyPath, perr)
	case ReleaseDigest(policy) != manifest.PolicyDigest:
		gap("the manifest's policy digest is not the digest of the release policy of the tagged tree (%s)", policyPath)
	default:
		// Version evidence follows the policy of the cut when it names a source.
		if policy.VersionSource != nil {
			if filepath.ToSlash(filepath.Clean(policy.VersionSource.Path)) != manifest.VersionEvidence["source"] || policy.VersionSource.Kind != manifest.VersionEvidence["kind"] {
				gap("the version evidence names a source or kind other than the version_source of the release policy of the tagged tree")
			}
		}
	}
	// Version evidence: the file of the tagged tree that holds the version.
	if g, limit := snapshotVersionEvidence(root, commit, manifest); g != "" {
		snap.Gaps = append(snap.Gaps, g)
	} else if limit != "" {
		snap.Limits = append(snap.Limits, limit)
	}
	return snap, nil
}

// snapshotVersionEvidence re-verifies the version a manifest recorded against the file of the
// tagged tree that holds it. The file must be there and be readable whatever its kind; only a
// text source can be recomputed from the evidence the manifest carries (a JSON source also needs
// the key it was read from, which the manifest does not retain), and that limit is reported
// as such, not guessed.
func snapshotVersionEvidence(root, commit string, manifest *ReleaseManifest) (gap, limit string) {
	source, kind, value := manifest.VersionEvidence["source"], manifest.VersionEvidence["kind"], manifest.VersionEvidence["value"]
	if source == "" {
		return "the manifest records no version source", ""
	}
	raw, err := gitBlobAt(root, commit, source)
	if errors.Is(err, os.ErrNotExist) {
		return "version source " + source + " is missing from the tagged tree", ""
	}
	if err != nil {
		return "version source " + source + " could not be read from the tagged tree: " + err.Error(), ""
	}
	switch kind {
	case ReleaseVersionSourceText:
	case ReleaseVersionSourceJSON:
		return "", "the version evidence is a JSON source whose key the manifest does not retain: its file exists in the tagged tree and its value was not re-read"
	case "":
		// The engine's own version file is a Go source that holds the number in code: it was
		// found in the tagged tree, and its value is not re-read from the manifest alone.
		return "", "the version evidence is the engine's own version file: it exists in the tagged tree and its value was not re-read"
	default:
		return "", "the version evidence has a kind (" + fmt.Sprintf("%q", kind) + ") this engine does not re-verify"
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
